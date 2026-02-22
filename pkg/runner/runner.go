package runner

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

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

		// Execute in parallel
		results := c.executeParallel(scenarios, workers)

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
	// Set up cacik context with logger
	opts := make([]cacik.Option, 0)
	if c.logger != nil {
		opts = append(opts, cacik.WithLogger(c.logger))
	}
	ctx := cacik.New(opts...)
	c.executor.SetCacikContext(ctx)

	for _, docWithFile := range docs {
		if err := c.executor.Execute(docWithFile.document); err != nil {
			return fmt.Errorf("execution failed for %s: %w", docWithFile.file, err)
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
