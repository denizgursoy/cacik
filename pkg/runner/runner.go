package runner

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	tagexpressions "github.com/cucumber/tag-expressions/go/v6"

	messages "github.com/cucumber/messages/go/v21"
	"github.com/denizgursoy/cacik/pkg/cacik"
	"github.com/denizgursoy/cacik/pkg/executor"
	"github.com/denizgursoy/cacik/pkg/gherkin_parser"
	"github.com/denizgursoy/cacik/pkg/models"
)

type (
	CucumberRunner struct {
		config             *models.Config
		featureDirectories []string
		executor           *executor.StepExecutor
		logger             cacik.Logger
	}
)

// NewCucumberRunner creates a new runner with an internal step executor
func NewCucumberRunner() *CucumberRunner {
	return &CucumberRunner{
		executor: executor.NewStepExecutor(),
	}
}

// NewCucumberRunnerWithExecutor creates a runner with a custom executor (for testing)
func NewCucumberRunnerWithExecutor(exec *executor.StepExecutor) *CucumberRunner {
	return &CucumberRunner{
		executor: exec,
	}
}

func (c *CucumberRunner) WithConfigFunc(configFunction func() *models.Config) *CucumberRunner {
	if configFunction != nil {
		c.config = configFunction()
	}

	return c
}

func (c *CucumberRunner) WithFeaturesDirectories(directories ...string) *CucumberRunner {
	c.featureDirectories = directories

	return c
}

// RegisterStep registers a step definition with the executor
func (c *CucumberRunner) RegisterStep(definition string, function any) *CucumberRunner {
	if err := c.executor.RegisterStep(definition, function); err != nil {
		panic(fmt.Sprintf("failed to register step %q: %v", definition, err))
	}
	return c
}

// RegisterCustomType registers a custom type with its allowed values for validation
// name: the type name (e.g., "Color")
// underlying: the underlying primitive type (e.g., "string")
// values: map of lowercase name/value -> actual value for case-insensitive matching
func (c *CucumberRunner) RegisterCustomType(name, underlying string, values map[string]string) *CucumberRunner {
	c.executor.RegisterCustomType(name, underlying, values)
	return c
}

// WithLogger sets the logger for step functions
func (c *CucumberRunner) WithLogger(logger cacik.Logger) *CucumberRunner {
	c.logger = logger
	return c
}

// Run executes all feature files, optionally filtering by tag expression from CLI args.
// Tag expression is passed via --tags flag, e.g.: go run . --tags "@smoke and @fast"
// Parallel execution is enabled via --parallel flag, e.g.: go run . --parallel 4
// Supports Cucumber tag expression syntax: and, or, not, parentheses
// Examples:
//   - @smoke
//   - @smoke and @fast
//   - @gui or @database
//   - @wip and not @slow
//   - (@smoke or @ui) and not @slow
func (c *CucumberRunner) Run() error {
	if len(c.featureDirectories) == 0 {
		c.featureDirectories = append(c.featureDirectories, ".")
	}

	// Parse parallel worker count from CLI arguments
	workers := parseParallelFromArgs()

	// Parse tag expression from CLI arguments
	tagExpr := parseTagsFromArgs()
	var evaluator tagexpressions.Evaluatable
	if tagExpr != "" {
		var err error
		evaluator, err = tagexpressions.Parse(tagExpr)
		if err != nil {
			return fmt.Errorf("invalid tag expression %q: %w", tagExpr, err)
		}
	}

	featureFiles, err := gherkin_parser.SearchFeatureFilesIn(c.featureDirectories)
	if err != nil {
		return err
	}

	// Parse all documents and filter by tags
	var docs []*documentWithFile
	for _, file := range featureFiles {
		readFile, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("could not read file %s, error=%w", file, err)
		}
		document, err := gherkin_parser.ParseGherkinFile(bytes.NewReader(readFile))
		if err != nil {
			return fmt.Errorf("gherkin parse error in file %s, error=%w", file, err)
		}

		// Filter document by tags if expression provided
		if evaluator != nil && document.Feature != nil {
			document = filterDocumentByTags(document, evaluator)
			// Skip if no scenarios match
			if document == nil || !hasScenarios(document) {
				continue
			}
		}

		docs = append(docs, &documentWithFile{document: document, file: file})
	}

	// If parallel execution is requested
	if workers > 1 {
		// Collect all scenarios from filtered documents
		scenarios := c.collectScenarios(docs)
		if len(scenarios) == 0 {
			return nil
		}

		// Parse flags
		useColors := !parseNoColorFromArgs()
		failFast := parseFailFastFromArgs()

		// Create main reporter for summary aggregation
		mainReporter := cacik.NewConsoleReporter(useColors)

		// Execute in parallel with buffered reporters
		results := c.executeParallelWithReporter(scenarios, workers, useColors, mainReporter, failFast)

		// Print summary
		mainReporter.PrintSummary()

		// Collect errors
		var failedResults []ScenarioResult
		for _, r := range results {
			if r.Error != nil {
				failedResults = append(failedResults, r)
			}
		}

		if len(failedResults) > 0 {
			return fmt.Errorf("%d scenario(s) failed:\n%s", len(failedResults), formatErrors(failedResults))
		}
		return nil
	}

	// Sequential execution (original behavior)
	// Parse flags
	useColors := !parseNoColorFromArgs()
	failFast := parseFailFastFromArgs()

	// Create reporter
	reporter := cacik.NewConsoleReporter(useColors)

	// Set up cacik context with logger and reporter
	opts := make([]cacik.Option, 0)
	if c.logger != nil {
		opts = append(opts, cacik.WithLogger(c.logger))
	}
	opts = append(opts, cacik.WithReporter(reporter))
	ctx := cacik.New(opts...)
	c.executor.SetCacikContext(ctx)

	// Execute documents with reporter lifecycle calls
	var runErr error
	for _, docWithFile := range docs {
		if err := c.executeDocumentWithReporter(docWithFile.document, reporter, failFast); err != nil {
			runErr = fmt.Errorf("execution failed for %s: %w", docWithFile.file, err)
			break
		}
	}

	// Print summary
	reporter.PrintSummary()

	return runErr
}

// executeDocumentWithReporter executes a document with reporter lifecycle calls
func (c *CucumberRunner) executeDocumentWithReporter(document *messages.GherkinDocument, reporter *cacik.ConsoleReporter, failFast bool) error {
	if document == nil || document.Feature == nil {
		return nil
	}

	feature := document.Feature
	reporter.FeatureStart(feature.Name)

	var featureBackground *messages.Background

	for _, child := range feature.Children {
		if child.Background != nil {
			featureBackground = child.Background
		} else if child.Rule != nil {
			if err := c.executeRuleWithReporter(child.Rule, featureBackground, reporter, failFast); err != nil {
				return err
			}
		} else if child.Scenario != nil {
			if err := c.executeScenarioWithReporter(child.Scenario, featureBackground, nil, reporter, failFast); err != nil {
				return err
			}
		}
	}

	return nil
}

// executeRuleWithReporter executes a rule with reporter lifecycle calls
func (c *CucumberRunner) executeRuleWithReporter(rule *messages.Rule, featureBackground *messages.Background, reporter *cacik.ConsoleReporter, failFast bool) error {
	var ruleBackground *messages.Background

	for _, child := range rule.Children {
		if child.Background != nil {
			ruleBackground = child.Background
		} else if child.Scenario != nil {
			// Execute feature background first
			if featureBackground != nil {
				reporter.BackgroundStart()
				if err := c.executeBackgroundStepsWithReporter(featureBackground, reporter); err != nil {
					// Mark scenario as failed since background failed
					reporter.AddScenarioResult(false)
					if failFast {
						return fmt.Errorf("scenario %q failed: %w", child.Scenario.Name, err)
					}
					continue
				}
			}
			// Execute rule background and scenario
			if err := c.executeScenarioWithReporter(child.Scenario, ruleBackground, nil, reporter, failFast); err != nil {
				return err
			}
		}
	}
	return nil
}

// executeScenarioWithReporter executes a scenario with reporter lifecycle calls
func (c *CucumberRunner) executeScenarioWithReporter(scenario *messages.Scenario, background *messages.Background, ruleBackground *messages.Background, reporter *cacik.ConsoleReporter, failFast bool) error {
	// Execute background if present
	if background != nil {
		reporter.BackgroundStart()
		if err := c.executeBackgroundStepsWithReporter(background, reporter); err != nil {
			// Background failed - skip scenario steps and mark remaining as skipped
			reporter.ScenarioStart(scenario.Name)
			for _, step := range scenario.Steps {
				reporter.StepSkipped(step.Keyword, step.Text)
				reporter.AddStepResult(false, true)
			}
			reporter.AddScenarioResult(false)
			if failFast {
				return fmt.Errorf("scenario %q failed: background step failed", scenario.Name)
			}
			return nil // Continue with next scenario
		}
	}

	reporter.ScenarioStart(scenario.Name)

	scenarioPassed := true
	var scenarioErr error
	for i, step := range scenario.Steps {
		if err := c.executor.ExecuteStepWithKeyword(step.Keyword, step.Text); err != nil {
			scenarioPassed = false
			scenarioErr = err
			// Skip remaining steps
			for j := i + 1; j < len(scenario.Steps); j++ {
				remainingStep := scenario.Steps[j]
				reporter.StepSkipped(remainingStep.Keyword, remainingStep.Text)
				reporter.AddStepResult(false, true)
			}
			break
		}
	}

	reporter.AddScenarioResult(scenarioPassed)

	if !scenarioPassed && failFast {
		return fmt.Errorf("scenario %q failed: %w", scenario.Name, scenarioErr)
	}
	return nil
}

// executeBackgroundStepsWithReporter executes background steps with reporter
func (c *CucumberRunner) executeBackgroundStepsWithReporter(background *messages.Background, reporter *cacik.ConsoleReporter) error {
	for i, step := range background.Steps {
		if err := c.executor.ExecuteStepWithKeyword(step.Keyword, step.Text); err != nil {
			// Skip remaining background steps
			for j := i + 1; j < len(background.Steps); j++ {
				remainingStep := background.Steps[j]
				reporter.StepSkipped(remainingStep.Keyword, remainingStep.Text)
				reporter.AddStepResult(false, true)
			}
			return err
		}
	}
	return nil
}

// parseTagsFromArgs extracts the tag expression from command-line arguments.
// Supports: --tags "@expression" or --tags="@expression"
func parseTagsFromArgs() string {
	args := os.Args[1:]
	for i, arg := range args {
		if arg == "--tags" && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(arg, "--tags=") {
			return strings.TrimPrefix(arg, "--tags=")
		}
	}
	return ""
}

// filterDocumentByTags returns a copy of the document with only matching scenarios.
// Tags are inherited: Feature tags → Rule tags → Scenario tags
func filterDocumentByTags(doc *messages.GherkinDocument, evaluator tagexpressions.Evaluatable) *messages.GherkinDocument {
	if doc.Feature == nil {
		return doc
	}

	featureTags := extractTagNames(doc.Feature.Tags)
	filteredChildren := make([]*messages.FeatureChild, 0)

	for _, child := range doc.Feature.Children {
		if child.Background != nil {
			// Always include backgrounds
			filteredChildren = append(filteredChildren, child)
		} else if child.Scenario != nil {
			// Check if scenario matches tags (inherit feature tags)
			scenarioTags := mergeTags(featureTags, extractTagNames(child.Scenario.Tags))
			if evaluator.Evaluate(scenarioTags) {
				filteredChildren = append(filteredChildren, child)
			}
		} else if child.Rule != nil {
			// Filter scenarios within the rule
			filteredRule := filterRuleByTags(child.Rule, featureTags, evaluator)
			if filteredRule != nil && hasRuleScenarios(filteredRule) {
				filteredChildren = append(filteredChildren, &messages.FeatureChild{Rule: filteredRule})
			}
		}
	}

	// Return new document with filtered children
	return &messages.GherkinDocument{
		Uri:      doc.Uri,
		Comments: doc.Comments,
		Feature: &messages.Feature{
			Location:    doc.Feature.Location,
			Tags:        doc.Feature.Tags,
			Language:    doc.Feature.Language,
			Keyword:     doc.Feature.Keyword,
			Name:        doc.Feature.Name,
			Description: doc.Feature.Description,
			Children:    filteredChildren,
		},
	}
}

// filterRuleByTags returns a copy of the rule with only matching scenarios.
func filterRuleByTags(rule *messages.Rule, featureTags []string, evaluator tagexpressions.Evaluatable) *messages.Rule {
	ruleTags := mergeTags(featureTags, extractTagNames(rule.Tags))
	filteredChildren := make([]*messages.RuleChild, 0)

	for _, child := range rule.Children {
		if child.Background != nil {
			// Always include backgrounds
			filteredChildren = append(filteredChildren, child)
		} else if child.Scenario != nil {
			// Check if scenario matches tags (inherit feature + rule tags)
			scenarioTags := mergeTags(ruleTags, extractTagNames(child.Scenario.Tags))
			if evaluator.Evaluate(scenarioTags) {
				filteredChildren = append(filteredChildren, child)
			}
		}
	}

	return &messages.Rule{
		Location:    rule.Location,
		Tags:        rule.Tags,
		Keyword:     rule.Keyword,
		Name:        rule.Name,
		Description: rule.Description,
		Children:    filteredChildren,
		Id:          rule.Id,
	}
}

// extractTagNames extracts tag names from a slice of Tag pointers.
// Returns tags WITH the @ prefix (e.g., "@smoke", "@fast")
func extractTagNames(tags []*messages.Tag) []string {
	names := make([]string, len(tags))
	for i, tag := range tags {
		names[i] = tag.Name // Includes @ prefix
	}
	return names
}

// mergeTags combines parent and child tags into a single slice.
func mergeTags(parent, child []string) []string {
	result := make([]string, 0, len(parent)+len(child))
	result = append(result, parent...)
	result = append(result, child...)
	return result
}

// hasScenarios checks if a document has any scenarios.
func hasScenarios(doc *messages.GherkinDocument) bool {
	if doc.Feature == nil {
		return false
	}
	for _, child := range doc.Feature.Children {
		if child.Scenario != nil {
			return true
		}
		if child.Rule != nil && hasRuleScenarios(child.Rule) {
			return true
		}
	}
	return false
}

// hasRuleScenarios checks if a rule has any scenarios.
func hasRuleScenarios(rule *messages.Rule) bool {
	for _, child := range rule.Children {
		if child.Scenario != nil {
			return true
		}
	}
	return false
}

// =============================================================================
// Parallel Execution Support
// =============================================================================

// ScenarioExecution holds a scenario with its associated backgrounds for execution
type ScenarioExecution struct {
	Scenario          *messages.Scenario
	FeatureBackground *messages.Background
	RuleBackground    *messages.Background
	FeatureFile       string
	FeatureName       string
}

// ScenarioResult holds the result of executing a scenario
type ScenarioResult struct {
	Execution *ScenarioExecution
	Error     error
}

// documentWithFile pairs a parsed document with its source file path
type documentWithFile struct {
	document *messages.GherkinDocument
	file     string
}

// parseNoColorFromArgs checks if --no-color flag is present in command-line arguments.
func parseNoColorFromArgs() bool {
	args := os.Args[1:]
	for _, arg := range args {
		if arg == "--no-color" {
			return true
		}
	}
	return false
}

// parseFailFastFromArgs checks if --fail-fast flag is present in command-line arguments.
func parseFailFastFromArgs() bool {
	args := os.Args[1:]
	for _, arg := range args {
		if arg == "--fail-fast" {
			return true
		}
	}
	return false
}

// parseParallelFromArgs extracts the parallel worker count from command-line arguments.
// Supports: --parallel 4 or --parallel=4
// Returns 1 (sequential) if not specified or invalid.
func parseParallelFromArgs() int {
	args := os.Args[1:]
	for i, arg := range args {
		if arg == "--parallel" && i+1 < len(args) {
			n, err := strconv.Atoi(args[i+1])
			if err == nil && n > 0 {
				return n
			}
		}
		if strings.HasPrefix(arg, "--parallel=") {
			val := strings.TrimPrefix(arg, "--parallel=")
			n, err := strconv.Atoi(val)
			if err == nil && n > 0 {
				return n
			}
		}
	}
	return 1
}

// collectScenarios gathers all scenarios from documents with their backgrounds
func (c *CucumberRunner) collectScenarios(docs []*documentWithFile) []ScenarioExecution {
	var scenarios []ScenarioExecution

	for _, docWithFile := range docs {
		doc := docWithFile.document
		if doc.Feature == nil {
			continue
		}

		var featureBackground *messages.Background
		featureName := doc.Feature.Name

		for _, child := range doc.Feature.Children {
			if child.Background != nil {
				featureBackground = child.Background
			} else if child.Scenario != nil {
				scenarios = append(scenarios, ScenarioExecution{
					Scenario:          child.Scenario,
					FeatureBackground: featureBackground,
					FeatureFile:       docWithFile.file,
					FeatureName:       featureName,
				})
			} else if child.Rule != nil {
				var ruleBackground *messages.Background
				for _, rc := range child.Rule.Children {
					if rc.Background != nil {
						ruleBackground = rc.Background
					} else if rc.Scenario != nil {
						scenarios = append(scenarios, ScenarioExecution{
							Scenario:          rc.Scenario,
							FeatureBackground: featureBackground,
							RuleBackground:    ruleBackground,
							FeatureFile:       docWithFile.file,
							FeatureName:       featureName,
						})
					}
				}
			}
		}
	}
	return scenarios
}

// executeScenarioWithIsolatedContext executes a single scenario with its own context
func (c *CucumberRunner) executeScenarioWithIsolatedContext(exec ScenarioExecution) error {
	// Create fresh context for this scenario
	opts := make([]cacik.Option, 0)
	if c.logger != nil {
		opts = append(opts, cacik.WithLogger(c.logger))
	}
	ctx := cacik.New(opts...)

	// Clone executor and set context
	isolatedExec := c.executor.Clone()
	isolatedExec.SetCacikContext(ctx)

	// Execute feature background
	if exec.FeatureBackground != nil {
		for _, step := range exec.FeatureBackground.Steps {
			if err := isolatedExec.ExecuteStep(step.Text); err != nil {
				return fmt.Errorf("[%s] feature background step %q failed: %w", exec.FeatureFile, step.Text, err)
			}
		}
	}

	// Execute rule background
	if exec.RuleBackground != nil {
		for _, step := range exec.RuleBackground.Steps {
			if err := isolatedExec.ExecuteStep(step.Text); err != nil {
				return fmt.Errorf("[%s] rule background step %q failed: %w", exec.FeatureFile, step.Text, err)
			}
		}
	}

	// Execute scenario steps
	for _, step := range exec.Scenario.Steps {
		if err := isolatedExec.ExecuteStep(step.Text); err != nil {
			return fmt.Errorf("[%s] scenario %q step %q failed: %w", exec.FeatureFile, exec.Scenario.Name, step.Text, err)
		}
	}

	return nil
}

// executeParallel runs scenarios in parallel using a worker pool
func (c *CucumberRunner) executeParallel(scenarios []ScenarioExecution, workers int) []ScenarioResult {
	jobs := make(chan ScenarioExecution, len(scenarios))
	results := make(chan ScenarioResult, len(scenarios))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for scenario := range jobs {
				err := c.executeScenarioWithIsolatedContext(scenario)
				results <- ScenarioResult{
					Execution: &scenario,
					Error:     err,
				}
			}
		}()
	}

	// Send jobs
	for _, s := range scenarios {
		jobs <- s
	}
	close(jobs)

	// Wait and close results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var allResults []ScenarioResult
	for r := range results {
		allResults = append(allResults, r)
	}
	return allResults
}

// executeParallelWithReporter runs scenarios in parallel with buffered reporters
func (c *CucumberRunner) executeParallelWithReporter(scenarios []ScenarioExecution, workers int, useColors bool, mainReporter *cacik.ConsoleReporter, failFast bool) []ScenarioResult {
	type resultWithReporter struct {
		result   ScenarioResult
		reporter *cacik.ConsoleReporter
	}

	jobs := make(chan ScenarioExecution, len(scenarios))
	results := make(chan resultWithReporter, len(scenarios))

	// For fail-fast: track if we should stop
	var failed int32 // atomic flag

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for scenario := range jobs {
				// Check if we should skip due to fail-fast
				if failFast && atomic.LoadInt32(&failed) == 1 {
					// Still need to drain the channel but skip execution
					continue
				}

				reporter, err := c.executeScenarioWithBufferedReporter(scenario, useColors)

				// Mark as failed if this scenario failed
				if err != nil && failFast {
					atomic.StoreInt32(&failed, 1)
				}

				results <- resultWithReporter{
					result: ScenarioResult{
						Execution: &scenario,
						Error:     err,
					},
					reporter: reporter,
				}
			}
		}()
	}

	// Send jobs
	for _, s := range scenarios {
		jobs <- s
	}
	close(jobs)

	// Wait and close results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results and flush output atomically
	var allResults []ScenarioResult
	for r := range results {
		// Flush buffered output atomically
		r.reporter.Flush()
		// Merge summary into main reporter
		mainReporter.MergeSummary(r.reporter)
		allResults = append(allResults, r.result)
	}
	return allResults
}

// executeScenarioWithBufferedReporter executes a scenario with a buffered reporter
func (c *CucumberRunner) executeScenarioWithBufferedReporter(exec ScenarioExecution, useColors bool) (*cacik.ConsoleReporter, error) {
	// Create buffered reporter for this scenario
	reporter := cacik.NewBufferedReporter(useColors)

	// Create fresh context for this scenario
	opts := make([]cacik.Option, 0)
	if c.logger != nil {
		opts = append(opts, cacik.WithLogger(c.logger))
	}
	opts = append(opts, cacik.WithReporter(reporter))
	ctx := cacik.New(opts...)

	// Clone executor and set context
	isolatedExec := c.executor.Clone()
	isolatedExec.SetCacikContext(ctx)

	// Print feature header
	reporter.FeatureStart(exec.FeatureName)

	scenarioPassed := true
	var execErr error

	// Execute feature background
	if exec.FeatureBackground != nil {
		reporter.BackgroundStart()
		for i, step := range exec.FeatureBackground.Steps {
			if err := isolatedExec.ExecuteStepWithKeyword(step.Keyword, step.Text); err != nil {
				scenarioPassed = false
				execErr = fmt.Errorf("[%s] feature background step %q failed: %w", exec.FeatureFile, step.Text, err)
				// Skip remaining background steps
				for j := i + 1; j < len(exec.FeatureBackground.Steps); j++ {
					remainingStep := exec.FeatureBackground.Steps[j]
					reporter.StepSkipped(remainingStep.Keyword, remainingStep.Text)
					reporter.AddStepResult(false, true)
				}
				// Skip scenario steps
				reporter.ScenarioStart(exec.Scenario.Name)
				for _, step := range exec.Scenario.Steps {
					reporter.StepSkipped(step.Keyword, step.Text)
					reporter.AddStepResult(false, true)
				}
				reporter.AddScenarioResult(false)
				return reporter, execErr
			}
		}
	}

	// Execute rule background
	if exec.RuleBackground != nil {
		reporter.BackgroundStart()
		for i, step := range exec.RuleBackground.Steps {
			if err := isolatedExec.ExecuteStepWithKeyword(step.Keyword, step.Text); err != nil {
				scenarioPassed = false
				execErr = fmt.Errorf("[%s] rule background step %q failed: %w", exec.FeatureFile, step.Text, err)
				// Skip remaining background steps
				for j := i + 1; j < len(exec.RuleBackground.Steps); j++ {
					remainingStep := exec.RuleBackground.Steps[j]
					reporter.StepSkipped(remainingStep.Keyword, remainingStep.Text)
					reporter.AddStepResult(false, true)
				}
				// Skip scenario steps
				reporter.ScenarioStart(exec.Scenario.Name)
				for _, step := range exec.Scenario.Steps {
					reporter.StepSkipped(step.Keyword, step.Text)
					reporter.AddStepResult(false, true)
				}
				reporter.AddScenarioResult(false)
				return reporter, execErr
			}
		}
	}

	// Execute scenario steps
	reporter.ScenarioStart(exec.Scenario.Name)
	for i, step := range exec.Scenario.Steps {
		if err := isolatedExec.ExecuteStepWithKeyword(step.Keyword, step.Text); err != nil {
			scenarioPassed = false
			execErr = fmt.Errorf("[%s] scenario %q step %q failed: %w", exec.FeatureFile, exec.Scenario.Name, step.Text, err)
			// Skip remaining steps
			for j := i + 1; j < len(exec.Scenario.Steps); j++ {
				remainingStep := exec.Scenario.Steps[j]
				reporter.StepSkipped(remainingStep.Keyword, remainingStep.Text)
				reporter.AddStepResult(false, true)
			}
			break
		}
	}

	reporter.AddScenarioResult(scenarioPassed)
	return reporter, execErr
}

// formatErrors formats multiple errors into a single error message
func formatErrors(results []ScenarioResult) string {
	var sb strings.Builder
	for i, r := range results {
		if r.Error != nil {
			if i > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(fmt.Sprintf("- %s: %v", r.Execution.Scenario.Name, r.Error))
		}
	}
	return sb.String()
}
