package runner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	tagexpressions "github.com/cucumber/tag-expressions/go/v6"

	messages "github.com/cucumber/messages/go/v21"
	"github.com/denizgursoy/cacik/pkg/cacik"
	"github.com/denizgursoy/cacik/pkg/executor"
	"github.com/denizgursoy/cacik/pkg/gherkin_parser"
)

type (
	CucumberRunner struct {
		config             *cacik.Config
		hooks              []*cacik.Hooks
		hookExecutor       *cacik.HookExecutor
		featureDirectories []string
		executor           *executor.StepExecutor
		logger             cacik.Logger
		t                  *testing.T
	}
)

// NewCucumberRunner creates a new runner with an internal step executor.
// The *testing.T is required — each scenario runs as a Go subtest via t.Run().
func NewCucumberRunner(t *testing.T) *CucumberRunner {
	return &CucumberRunner{
		executor: executor.NewStepExecutor(),
		t:        t,
	}
}

// NewCucumberRunnerWithExecutor creates a runner with a custom executor (for testing).
// The *testing.T is required — each scenario runs as a Go subtest via t.Run().
func NewCucumberRunnerWithExecutor(t *testing.T, exec *executor.StepExecutor) *CucumberRunner {
	return &CucumberRunner{
		executor: exec,
		t:        t,
	}
}

// WithConfig sets the configuration for the runner.
// CLI flags (--fail-fast, --no-color) override config values.
func (c *CucumberRunner) WithConfig(config *cacik.Config) *CucumberRunner {
	c.config = config
	return c
}

// WithHooks sets the lifecycle hooks for the runner.
// All hooks are executed in order sorted by their Order field.
func (c *CucumberRunner) WithHooks(hooks ...*cacik.Hooks) *CucumberRunner {
	c.hooks = append(c.hooks, hooks...)
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
// Tag expression is passed via --tags flag, e.g.: go test -v -- --tags "@smoke and @fast"
// All scenarios run as parallel subtests via t.Parallel(). Control concurrency with
// go test -parallel N (defaults to GOMAXPROCS).
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

	// Apply config with CLI overrides
	failFast, useColors, disableLog, disableReporter, disableWatch := c.resolveSettings()

	// Apply logger from config
	if c.config != nil && c.config.Logger != nil {
		c.logger = c.config.Logger
	}

	// If DisableLog is set, replace logger with a no-op logger
	if disableLog {
		c.logger = &cacik.NoopLogger{}
	}

	// Create hook executor
	c.hookExecutor = cacik.NewHookExecutor(c.hooks...)

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

	reportFile := c.resolveReportFile()

	return c.runWithTestingT(docs, failFast, useColors, disableReporter, disableWatch, reportFile, c.hookExecutor)
}

// runWithTestingT executes scenarios using t.Run() subtests.
// Each scenario becomes a Go subtest and calls t.Parallel() so Go's test runner
// controls concurrency (via go test -parallel N).
func (c *CucumberRunner) runWithTestingT(docs []*documentWithFile, failFast bool, useColors bool, disableReporter bool, disableWatch bool, reportFile string, hookExecutor *cacik.HookExecutor) error {
	runStartedAt := time.Now()

	// Collect all scenarios from filtered documents
	scenarios := c.collectScenarios(docs)
	if len(scenarios) == 0 {
		return nil
	}

	// Validate all step texts have matching definitions before running
	if err := c.resolveAllSteps(scenarios); err != nil {
		return err
	}

	// ── Watch server ────────────────────────────────────────────────
	var broker *cacik.EventBroker
	var watchServer *cacik.WatchServer
	if !disableWatch {
		broker = cacik.NewEventBroker()
		var err error
		watchServer, err = cacik.StartWatchServer(broker)
		if err != nil {
			// Non-fatal: log and continue without watch
			fmt.Fprintf(os.Stderr, "Warning: could not start watch server: %v\n", err)
		} else {
			fmt.Fprintf(os.Stdout, "Watch: %s\n", watchServer.Addr())
			cacik.OpenBrowser(watchServer.Addr())
		}

		// Publish run_started with all scenario metadata
		scenarioMeta := make([]cacik.ScenarioMetadataEvent, len(scenarios))
		for i, se := range scenarios {
			tags := extractTagNames(se.Scenario.Tags)
			scenarioMeta[i] = cacik.ScenarioMetadataEvent{
				Index:   i,
				Name:    se.Scenario.Name,
				Feature: se.FeatureName,
				Rule:    se.RuleName,
				Tags:    tags,
			}
		}
		broker.Publish(cacik.EventRunStarted, cacik.RunStartedEvent{
			Total:     len(scenarios),
			Scenarios: scenarioMeta,
		})
	}

	// Default report file to "report" when watch is active
	if !disableWatch && reportFile == "" {
		reportFile = "report.html"
	}
	// ────────────────────────────────────────────────────────────────

	// Provide scenario tag sets so BeforeAll/AfterAll can filter tagged hooks
	allTags := make([][]string, len(scenarios))
	for i, se := range scenarios {
		allTags[i] = extractTagNames(se.Scenario.Tags)
	}
	hookExecutor.SetAllScenarioTags(allTags)

	// Execute BeforeAll hooks (after scenarios are known, so tag filtering works)
	hookExecutor.ExecuteBeforeAll()
	// Ensure AfterAll hooks run even on error
	defer hookExecutor.ExecuteAfterAll()

	// Create main reporter for summary aggregation
	var mainReporter *cacik.ConsoleReporter
	if disableReporter {
		mainReporter = cacik.NewNoopConsoleReporter()
	} else {
		mainReporter = cacik.NewConsoleReporter(useColors)
	}

	// Wrap all parallel subtests in a barrier group so that the code after
	// this t.Run returns only once every subtest has finished.  Without
	// this, st.Parallel() causes t.Run to return immediately and the
	// summary / HTML-report code would read uninitialised data.
	c.t.Run("all", func(gt *testing.T) {
		for idx := range scenarios {
			scenario := &scenarios[idx] // pointer into original slice so writes propagate
			scenarioIdx := idx
			testName := scenario.FeatureName + "/" + scenario.Scenario.Name
			gt.Run(testName, func(st *testing.T) {
				st.Parallel()

				// Record scenario timing
				scenario.StartedAt = time.Now()
				defer func() { scenario.Duration = time.Since(scenario.StartedAt) }()

				// Use buffered reporter per subtest to avoid interleaved output.
				var reporter *cacik.ConsoleReporter
				if disableReporter {
					reporter = cacik.NewNoopConsoleReporter()
				} else {
					reporter = cacik.NewBufferedReporter(useColors)
					defer reporter.Flush()
				}
				// Always merge summary into mainReporter (even when output is disabled,
				// summary stats are still tracked and needed for RunResult/HTML report).
				defer mainReporter.MergeSummary(reporter)

				// Always print feature header (each subtest has its own buffered reporter)
				reporter.FeatureStart(scenario.FeatureName)

				// Create fresh context for this scenario with the subtest's *testing.T
				opts := make([]cacik.Option, 0)
				var capLogger *cacik.CapturingLogger
				if c.logger != nil {
					capLogger = cacik.NewCapturingLogger(c.logger)
				} else {
					capLogger = cacik.NewCapturingLogger(nil)
				}
				opts = append(opts, cacik.WithLogger(capLogger))
				opts = append(opts, cacik.WithTestingT(st))
				opts = append(opts, cacik.WithReporter(reporter))
				ctx := cacik.New(opts...)

				// Store the scenario UUID for events and result building.
				scenario.ContextID = ctx.ID()

				// Publish scenario_started event
				if broker != nil {
					broker.Publish(cacik.EventScenarioStarted, cacik.ScenarioStartedEvent{
						Index: scenarioIdx,
						ID:    ctx.ID(),
					})
				}
				// Publish scenario_completed on exit (deferred first = runs last)
				defer func() {
					// Store captured logs for buildRunResult
					scenario.CapturedLogs = capLogger.Entries()
					if broker != nil {
						broker.Publish(cacik.EventScenarioCompleted, cacik.ScenarioCompletedEvent{
							Index:      scenarioIdx,
							Passed:     scenario.Passed,
							Error:      scenario.Error,
							DurationMs: scenario.Duration.Milliseconds(),
							ID:         ctx.ID(),
							Logs:       logEntriesToJSON(capLogger.Entries()),
						})
					}
				}()

				// Helper to publish step events to the watch broker.
				publishStep := func(rs *executor.ResolvedStep, category string, stepIdx int) {
					if broker == nil {
						return
					}
					evt := cacik.StepCompletedEvent{
						ScenarioIndex: scenarioIdx,
						StepCategory:  category,
						StepIndex:     stepIdx,
						Keyword:       rs.Keyword,
						Text:          rs.Text,
						Status:        rs.Status,
						Error:         rs.Error,
						DurationMs:    rs.Duration.Milliseconds(),
						MatchLocs:     rs.MatchLocs,
						DataTable:     dataTableFromMessages(rs.DataTable),
						DataSnapshot:  formatDataSnapshot(ctx.Data().Snapshot()),
					}
					broker.Publish(cacik.EventStepCompleted, evt)
				}

				// Clone executor and set context
				isolatedExec := c.executor.Clone()
				isolatedExec.SetCacikContext(ctx)
				isolatedExec.SetHookExecutor(hookExecutor)

				cacikScenario := cacik.ScenarioFromMessage(scenario.Scenario)
				var scenarioErr error

				// Set scenario tags for step-level hook filtering, then execute BeforeScenario hooks
				if hookExecutor != nil {
					hookExecutor.SetScenarioTags(cacikScenario.Tags)
					hookExecutor.ExecuteBeforeScenario(cacikScenario)
				}
				// Ensure AfterScenario hooks always run, then clear scenario tags
				defer func() {
					if hookExecutor != nil {
						hookExecutor.ExecuteAfterScenario(cacikScenario, scenarioErr)
						hookExecutor.ClearScenarioTags()
					}
				}()

				scenarioPassed := true

				// Execute feature background
				if len(scenario.ResolvedFeatureBgSteps) > 0 {
					reporter.BackgroundStart()
					for i, rs := range scenario.ResolvedFeatureBgSteps {
						if err := isolatedExec.ExecuteResolvedStep(rs); err != nil {
							scenarioPassed = false
							scenarioErr = err
							scenario.Passed = false
							scenario.Error = err.Error()
							publishStep(rs, "feature_bg", i)
							// Skip remaining background steps
							for j := i + 1; j < len(scenario.ResolvedFeatureBgSteps); j++ {
								remaining := scenario.ResolvedFeatureBgSteps[j]
								remaining.Status = "skipped"
								reporter.StepSkipped(remaining.Keyword, remaining.Text)
								reportStepDataTable(reporter, remaining.DataTable)
								reporter.AddStepResult(false, true)
								publishStep(remaining, "feature_bg", j)
							}
							// Print rule header and skip rule bg steps
							if scenario.RuleName != "" {
								reporter.RuleStart(scenario.RuleName)
							}
							for si, s := range scenario.ResolvedRuleBgSteps {
								s.Status = "skipped"
								reporter.StepSkipped(s.Keyword, s.Text)
								reportStepDataTable(reporter, s.DataTable)
								reporter.AddStepResult(false, true)
								publishStep(s, "rule_bg", si)
							}
							// Skip scenario steps
							reporter.ScenarioStart(scenario.Scenario.Name)
							for si, s := range scenario.ResolvedScenarioSteps {
								s.Status = "skipped"
								reporter.StepSkipped(s.Keyword, s.Text)
								reportStepDataTable(reporter, s.DataTable)
								reporter.AddStepResult(false, true)
								publishStep(s, "scenario", si)
							}
							reporter.AddScenarioResult(false)
							st.Fatalf("feature background step %q failed: %v", rs.Text, err)
							return
						}
						publishStep(rs, "feature_bg", i)
					}
				}

				// Print rule header when the scenario belongs to a rule
				// (after feature background, before rule background).
				if scenario.RuleName != "" {
					reporter.RuleStart(scenario.RuleName)
				}

				// Execute rule background
				if len(scenario.ResolvedRuleBgSteps) > 0 {
					reporter.BackgroundStart()
					for i, rs := range scenario.ResolvedRuleBgSteps {
						if err := isolatedExec.ExecuteResolvedStep(rs); err != nil {
							scenarioPassed = false
							scenarioErr = err
							scenario.Passed = false
							scenario.Error = err.Error()
							publishStep(rs, "rule_bg", i)
							// Skip remaining background steps
							for j := i + 1; j < len(scenario.ResolvedRuleBgSteps); j++ {
								remaining := scenario.ResolvedRuleBgSteps[j]
								remaining.Status = "skipped"
								reporter.StepSkipped(remaining.Keyword, remaining.Text)
								reportStepDataTable(reporter, remaining.DataTable)
								reporter.AddStepResult(false, true)
								publishStep(remaining, "rule_bg", j)
							}
							// Skip scenario steps
							reporter.ScenarioStart(scenario.Scenario.Name)
							for si, s := range scenario.ResolvedScenarioSteps {
								s.Status = "skipped"
								reporter.StepSkipped(s.Keyword, s.Text)
								reportStepDataTable(reporter, s.DataTable)
								reporter.AddStepResult(false, true)
								publishStep(s, "scenario", si)
							}
							reporter.AddScenarioResult(false)
							st.Fatalf("rule background step %q failed: %v", rs.Text, err)
							return
						}
						publishStep(rs, "rule_bg", i)
					}
				}

				// Execute scenario steps
				reporter.ScenarioStart(scenario.Scenario.Name)
				for i, rs := range scenario.ResolvedScenarioSteps {
					if err := isolatedExec.ExecuteResolvedStep(rs); err != nil {
						scenarioPassed = false
						scenarioErr = err
						scenario.Passed = false
						scenario.Error = err.Error()
						publishStep(rs, "scenario", i)
						// Skip remaining steps
						for j := i + 1; j < len(scenario.ResolvedScenarioSteps); j++ {
							remaining := scenario.ResolvedScenarioSteps[j]
							remaining.Status = "skipped"
							reporter.StepSkipped(remaining.Keyword, remaining.Text)
							reportStepDataTable(reporter, remaining.DataTable)
							reporter.AddStepResult(false, true)
							publishStep(remaining, "scenario", j)
						}
						reporter.AddScenarioResult(false)
						st.Fatalf("step %q failed: %v", rs.Text, err)
						return
					}
					publishStep(rs, "scenario", i)
				}

				scenario.Passed = scenarioPassed
				reporter.AddScenarioResult(scenarioPassed)
			})
		}
	})

	// All parallel subtests have now completed (the "all" group acts as a barrier).

	// Print summary after all subtests complete
	mainReporter.PrintSummary()

	// Build RunResult from scenario execution data
	runResult := c.buildRunResult(scenarios, mainReporter, runStartedAt)

	// Publish run_completed and shutdown watch server
	if broker != nil {
		s := runResult.Summary
		broker.Publish(cacik.EventRunCompleted, cacik.RunCompletedEvent{
			DurationMs: runResult.Duration.Milliseconds(),
			Summary: cacik.RunCompletedSummary{
				ScenariosTotal:  s.ScenariosTotal,
				ScenariosPassed: s.ScenariosPassed,
				ScenariosFailed: s.ScenariosFailed,
				StepsTotal:      s.StepsTotal,
				StepsPassed:     s.StepsPassed,
				StepsFailed:     s.StepsFailed,
				StepsSkipped:    s.StepsSkipped,
			},
		})
	}

	// Generate HTML report (both watch and non-watch paths use the same static report)
	if reportFile != "" {
		if err := cacik.GenerateHTMLReport(reportFile, runResult); err != nil {
			return fmt.Errorf("failed to generate HTML report: %w", err)
		}
	}

	// Shutdown watch server after report is written
	if watchServer != nil {
		// Grace period so browser receives final events before server dies
		time.Sleep(2 * time.Second)
		watchServer.Shutdown(3 * time.Second)
		broker.Close()
	}

	// Call AfterRun callback if configured
	if c.config != nil && c.config.AfterRun != nil {
		c.config.AfterRun(runResult)
	}

	return nil
}

// resolveSettings resolves runtime settings from config and CLI flags.
// CLI flags always override config values.
func (c *CucumberRunner) resolveSettings() (failFast bool, useColors bool, disableLog bool, disableReporter bool, disableWatch bool) {
	// Start with config values (if any)
	if c.config != nil {
		failFast = c.config.FailFast
		useColors = !c.config.NoColor
		disableLog = c.config.DisableLog
		disableReporter = c.config.DisableReporter
		disableWatch = c.config.DisableWatch
	} else {
		useColors = true // default to colors
	}

	// CLI overrides
	if parseFailFastFromArgs() {
		failFast = true
	}
	if parseNoColorFromArgs() {
		useColors = false
	}
	if parseDisableLogFromArgs() {
		disableLog = true
	}
	if parseDisableReporterFromArgs() {
		disableReporter = true
	}
	if parseDisableWatchFromArgs() {
		disableWatch = true
	}

	return failFast, useColors, disableLog, disableReporter, disableWatch
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
			// Expand outline first so Examples-level tags are on each expanded scenario
			for _, expanded := range expandScenarioOutline(child.Scenario) {
				scenarioTags := mergeTags(featureTags, extractTagNames(expanded.Tags))
				if evaluator.Evaluate(scenarioTags) {
					filteredChildren = append(filteredChildren, &messages.FeatureChild{Scenario: expanded})
				}
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
			// Expand outline first so Examples-level tags are on each expanded scenario
			for _, expanded := range expandScenarioOutline(child.Scenario) {
				scenarioTags := mergeTags(ruleTags, extractTagNames(expanded.Tags))
				if evaluator.Evaluate(scenarioTags) {
					filteredChildren = append(filteredChildren, &messages.RuleChild{Scenario: expanded})
				}
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

// prependTags prepends parent *messages.Tag entries before child tags.
// Used by collectScenarios to inherit Feature/Rule tags onto scenarios.
func prependTags(parent, child []*messages.Tag) []*messages.Tag {
	if len(parent) == 0 {
		return child
	}
	result := make([]*messages.Tag, 0, len(parent)+len(child))
	result = append(result, parent...)
	result = append(result, child...)
	return result
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
// Scenario Execution Types
// =============================================================================

// ScenarioExecution holds a scenario with its associated backgrounds for execution
type ScenarioExecution struct {
	Scenario          *messages.Scenario
	FeatureBackground *messages.Background
	RuleBackground    *messages.Background
	FeatureFile       string
	FeatureName       string
	RuleName          string
	// Pre-resolved step matches (populated by resolveAllSteps before execution)
	ResolvedFeatureBgSteps []*executor.ResolvedStep
	ResolvedRuleBgSteps    []*executor.ResolvedStep
	ResolvedScenarioSteps  []*executor.ResolvedStep
	// Execution outcome (populated during runWithTestingT)
	StartedAt    time.Time
	Duration     time.Duration
	Passed       bool
	Error        string
	ContextID    string           // scenario UUID from ctx.ID()
	CapturedLogs []cacik.LogEntry // captured log entries from CapturingLogger
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

// parseDisableLogFromArgs checks if --disable-log flag is present in command-line arguments.
func parseDisableLogFromArgs() bool {
	args := os.Args[1:]
	for _, arg := range args {
		if arg == "--disable-log" {
			return true
		}
	}
	return false
}

// parseDisableReporterFromArgs checks if --disable-reporter flag is present in command-line arguments.
func parseDisableReporterFromArgs() bool {
	args := os.Args[1:]
	for _, arg := range args {
		if arg == "--disable-reporter" {
			return true
		}
	}
	return false
}

// parseDisableWatchFromArgs checks if --disable-watch flag is present in command-line arguments.
func parseDisableWatchFromArgs() bool {
	args := os.Args[1:]
	for _, arg := range args {
		if arg == "--disable-watch" {
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

// parseReportFileFromArgs extracts the report file path from command-line arguments.
// Supports: --report-file path or --report-file=path
func parseReportFileFromArgs() string {
	args := os.Args[1:]
	for i, arg := range args {
		if arg == "--report-file" && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(arg, "--report-file=") {
			return strings.TrimPrefix(arg, "--report-file=")
		}
	}
	return ""
}

// resolveReportFile resolves the report file path from config and CLI flags.
// CLI flag --report-file overrides config value.
// The value is treated as a file name without extension; ".html" is appended automatically.
func (c *CucumberRunner) resolveReportFile() string {
	var reportFile string
	if c.config != nil && c.config.ReportFile != "" {
		reportFile = c.config.ReportFile
	}
	if cliReport := parseReportFileFromArgs(); cliReport != "" {
		reportFile = cliReport
	}
	if reportFile != "" {
		reportFile = reportFile + ".html"
	}
	return reportFile
}

// buildRunResult converts internal ScenarioExecution data into a public RunResult.
func (c *CucumberRunner) buildRunResult(scenarios []ScenarioExecution, reporter *cacik.ConsoleReporter, runStartedAt time.Time) cacik.RunResult {
	scenarioResults := make([]cacik.ScenarioResult, 0, len(scenarios))

	for i := range scenarios {
		se := &scenarios[i]

		// Collect steps by origin: feature bg, rule bg, scenario
		var featureBgSteps []cacik.StepResult
		for _, rs := range se.ResolvedFeatureBgSteps {
			featureBgSteps = append(featureBgSteps, resolvedStepToResult(rs))
		}
		var ruleBgSteps []cacik.StepResult
		for _, rs := range se.ResolvedRuleBgSteps {
			ruleBgSteps = append(ruleBgSteps, resolvedStepToResult(rs))
		}
		var steps []cacik.StepResult
		for _, rs := range se.ResolvedScenarioSteps {
			steps = append(steps, resolvedStepToResult(rs))
		}

		// Extract tags
		var tags []string
		for _, tag := range se.Scenario.Tags {
			tags = append(tags, tag.Name)
		}

		scenarioResults = append(scenarioResults, cacik.ScenarioResult{
			ID:             se.ContextID,
			FeatureName:    se.FeatureName,
			RuleName:       se.RuleName,
			Name:           se.Scenario.Name,
			Tags:           tags,
			Passed:         se.Passed,
			Error:          se.Error,
			Duration:       se.Duration,
			StartedAt:      se.StartedAt,
			FeatureBgSteps: featureBgSteps,
			RuleBgSteps:    ruleBgSteps,
			Steps:          steps,
			Logs:           se.CapturedLogs,
		})
	}

	return cacik.RunResult{
		Scenarios: scenarioResults,
		Summary:   reporter.GetSummary(),
		Duration:  time.Since(runStartedAt),
		StartedAt: runStartedAt,
	}
}

// resolvedStepToResult maps an internal ResolvedStep to a public StepResult.
func resolvedStepToResult(rs *executor.ResolvedStep) cacik.StepResult {
	var status cacik.StepStatus
	switch rs.Status {
	case "passed":
		status = cacik.StepPassed
	case "failed":
		status = cacik.StepFailed
	case "skipped":
		status = cacik.StepSkipped
	default:
		status = cacik.StepSkipped
	}

	return cacik.StepResult{
		Keyword:   rs.Keyword,
		Text:      rs.Text,
		Status:    status,
		Error:     rs.Error,
		Duration:  rs.Duration,
		StartedAt: rs.StartedAt,
		MatchLocs: rs.MatchLocs,
		DataTable: dataTableFromMessages(rs.DataTable),
	}
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
				for _, expanded := range expandScenarioOutline(child.Scenario) {
					// Prepend feature tags so hooks and reports see inherited tags
					expanded.Tags = prependTags(doc.Feature.Tags, expanded.Tags)
					scenarios = append(scenarios, ScenarioExecution{
						Scenario:          expanded,
						FeatureBackground: featureBackground,
						FeatureFile:       docWithFile.file,
						FeatureName:       featureName,
					})
				}
			} else if child.Rule != nil {
				var ruleBackground *messages.Background
				for _, rc := range child.Rule.Children {
					if rc.Background != nil {
						ruleBackground = rc.Background
					} else if rc.Scenario != nil {
						for _, expanded := range expandScenarioOutline(rc.Scenario) {
							// Prepend feature + rule tags so hooks and reports see inherited tags
							expanded.Tags = prependTags(doc.Feature.Tags, prependTags(child.Rule.Tags, expanded.Tags))
							scenarios = append(scenarios, ScenarioExecution{
								Scenario:          expanded,
								FeatureBackground: featureBackground,
								RuleBackground:    ruleBackground,
								FeatureFile:       docWithFile.file,
								FeatureName:       featureName,
								RuleName:          child.Rule.Name,
							})
						}
					}
				}
			}
		}
	}
	return scenarios
}

// resolveAllSteps resolves every step in every scenario against the registered
// step definitions. Each step's matching definition and captured arguments are
// stored in the ResolvedStep fields of ScenarioExecution. Fails fast on the
// first unmatched step.
func (c *CucumberRunner) resolveAllSteps(scenarios []ScenarioExecution) error {
	for i := range scenarios {
		se := &scenarios[i]

		if se.FeatureBackground != nil {
			for _, step := range se.FeatureBackground.Steps {
				rs, err := c.executor.ResolveStep(step.Keyword, step.Text, step.DataTable)
				if err != nil {
					return fmt.Errorf("no matching step definition found for: %q in Feature: %s / Scenario: %s (%s)",
						step.Text, se.FeatureName, se.Scenario.Name, se.FeatureFile)
				}
				se.ResolvedFeatureBgSteps = append(se.ResolvedFeatureBgSteps, rs)
			}
		}

		if se.RuleBackground != nil {
			for _, step := range se.RuleBackground.Steps {
				rs, err := c.executor.ResolveStep(step.Keyword, step.Text, step.DataTable)
				if err != nil {
					return fmt.Errorf("no matching step definition found for: %q in Feature: %s / Scenario: %s (%s)",
						step.Text, se.FeatureName, se.Scenario.Name, se.FeatureFile)
				}
				se.ResolvedRuleBgSteps = append(se.ResolvedRuleBgSteps, rs)
			}
		}

		for _, step := range se.Scenario.Steps {
			rs, err := c.executor.ResolveStep(step.Keyword, step.Text, step.DataTable)
			if err != nil {
				return fmt.Errorf("no matching step definition found for: %q in Feature: %s / Scenario: %s (%s)",
					step.Text, se.FeatureName, se.Scenario.Name, se.FeatureFile)
			}
			se.ResolvedScenarioSteps = append(se.ResolvedScenarioSteps, rs)
		}
	}
	return nil
}

// expandScenarioOutline expands a Scenario Outline into concrete scenarios by
// substituting <placeholder> values from each Examples row into step text and
// DataTable cells.  Regular scenarios (no Examples) are returned as-is in a
// single-element slice.
func expandScenarioOutline(scenario *messages.Scenario) []*messages.Scenario {
	if len(scenario.Examples) == 0 {
		return []*messages.Scenario{scenario}
	}

	var expanded []*messages.Scenario

	for _, examples := range scenario.Examples {
		if examples.TableHeader == nil {
			continue
		}
		// Column names from the header row
		headers := make([]string, len(examples.TableHeader.Cells))
		for i, cell := range examples.TableHeader.Cells {
			headers[i] = cell.Value
		}

		for rowIdx, row := range examples.TableBody {
			// Build placeholder → value map
			replacements := make(map[string]string, len(headers))
			for i, cell := range row.Cells {
				if i < len(headers) {
					replacements["<"+headers[i]+">"] = cell.Value
				}
			}

			// Deep-copy steps with substitutions
			newSteps := make([]*messages.Step, len(scenario.Steps))
			for si, step := range scenario.Steps {
				newStep := &messages.Step{
					Location:    step.Location,
					Keyword:     step.Keyword,
					Text:        substituteText(step.Text, replacements),
					Id:          step.Id,
					KeywordType: step.KeywordType,
					DocString:   step.DocString,
				}
				// Substitute inside DataTable cells too
				if step.DataTable != nil {
					newStep.DataTable = substituteDataTable(step.DataTable, replacements)
				}
				newSteps[si] = newStep
			}

			// Merge scenario tags + examples tags
			mergedTags := make([]*messages.Tag, 0, len(scenario.Tags)+len(examples.Tags))
			mergedTags = append(mergedTags, scenario.Tags...)
			mergedTags = append(mergedTags, examples.Tags...)

			// Build a descriptive name: "Outline Name -- <examplesName> #<row>"
			name := scenario.Name
			if examples.Name != "" {
				name += " -- " + examples.Name
			}
			name += fmt.Sprintf(" (#%d)", rowIdx+1)

			expanded = append(expanded, &messages.Scenario{
				Location:    scenario.Location,
				Tags:        mergedTags,
				Keyword:     "Scenario",
				Name:        name,
				Description: scenario.Description,
				Steps:       newSteps,
				Id:          scenario.Id,
			})
		}
	}
	return expanded
}

// substituteText replaces all <placeholder> occurrences in text.
func substituteText(text string, replacements map[string]string) string {
	for placeholder, value := range replacements {
		text = strings.ReplaceAll(text, placeholder, value)
	}
	return text
}

// substituteDataTable deep-copies a DataTable with placeholder substitution.
func substituteDataTable(dt *messages.DataTable, replacements map[string]string) *messages.DataTable {
	newRows := make([]*messages.TableRow, len(dt.Rows))
	for ri, row := range dt.Rows {
		newCells := make([]*messages.TableCell, len(row.Cells))
		for ci, cell := range row.Cells {
			newCells[ci] = &messages.TableCell{
				Location: cell.Location,
				Value:    substituteText(cell.Value, replacements),
			}
		}
		newRows[ri] = &messages.TableRow{
			Location: row.Location,
			Cells:    newCells,
			Id:       row.Id,
		}
	}
	return &messages.DataTable{
		Location: dt.Location,
		Rows:     newRows,
	}
}

// reportStepDataTable prints a step's DataTable via the reporter, if present.
func reportStepDataTable(reporter cacik.Reporter, dt *messages.DataTable) {
	if dt == nil {
		return
	}
	rows := make([][]string, 0, len(dt.Rows))
	for _, row := range dt.Rows {
		cells := make([]string, 0, len(row.Cells))
		for _, cell := range row.Cells {
			cells = append(cells, cell.Value)
		}
		rows = append(rows, cells)
	}
	reporter.StepDataTable(rows)
}

// dataTableFromMessages converts a *messages.DataTable to [][]string.
// Returns nil if dt is nil.
func dataTableFromMessages(dt *messages.DataTable) [][]string {
	if dt == nil {
		return nil
	}
	rows := make([][]string, 0, len(dt.Rows))
	for _, row := range dt.Rows {
		cells := make([]string, 0, len(row.Cells))
		for _, cell := range row.Cells {
			cells = append(cells, cell.Value)
		}
		rows = append(rows, cells)
	}
	return rows
}

// formatDataSnapshot converts a ctx.Data().Snapshot() map to a map[string]string
// with smart formatting: simple types use fmt.Sprintf("%v"), complex types use JSON.
func formatDataSnapshot(snap map[string]any) map[string]string {
	if len(snap) == 0 {
		return nil
	}
	out := make(map[string]string, len(snap))
	for k, v := range snap {
		out[k] = formatValue(v)
	}
	return out
}

// formatValue formats a single value for display. Simple scalar types use
// fmt.Sprintf("%v"); complex types (maps, slices, structs) are JSON-marshaled.
func formatValue(v any) string {
	if v == nil {
		return "<nil>"
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Map, reflect.Slice, reflect.Array, reflect.Struct, reflect.Ptr:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// logEntriesToJSON converts []cacik.LogEntry to []cacik.LogEntryJSON for SSE events.
func logEntriesToJSON(entries []cacik.LogEntry) []cacik.LogEntryJSON {
	if len(entries) == 0 {
		return nil
	}
	out := make([]cacik.LogEntryJSON, len(entries))
	for i, e := range entries {
		out[i] = cacik.LogEntryJSON{
			Level:   e.Level,
			Message: e.Message,
			Args:    e.Args,
		}
	}
	return out
}
