package runner

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	tagexpressions "github.com/cucumber/tag-expressions/go/v6"

	messages "github.com/cucumber/messages/go/v21"
	"github.com/denizgursoy/cacik/pkg/executor"
	"github.com/denizgursoy/cacik/pkg/gherkin_parser"
	"github.com/denizgursoy/cacik/pkg/models"
)

type (
	CucumberRunner struct {
		config             *models.Config
		featureDirectories []string
		executor           *executor.StepExecutor
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

// Run executes all feature files, optionally filtering by tag expression from CLI args.
// Tag expression is passed via --tags flag, e.g.: go run . --tags "@smoke and @fast"
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

		// Execute the document
		if err := c.executor.Execute(document); err != nil {
			return fmt.Errorf("execution failed for %s: %w", file, err)
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
