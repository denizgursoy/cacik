package runner

import (
	"bytes"
	"fmt"
	"os"
	"slices"

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

func (c *CucumberRunner) RunWithTags(userTags ...string) error {
	if len(c.featureDirectories) == 0 {
		c.featureDirectories = append(c.featureDirectories, ".")
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

		// Skip documents that don't match tags (if tags specified)
		if len(userTags) > 0 && document.Feature != nil {
			if !includeTags(document.Feature.Tags, userTags) {
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

func getBackground(feature *messages.Feature) *messages.Background {
	for _, child := range feature.Children {
		if child.Background != nil {
			return child.Background
		}
	}

	return nil
}

func getRuleBackground(rule *messages.Rule) *messages.Background {
	for _, child := range rule.Children {
		if child.Background != nil {
			return child.Background
		}
	}

	return nil
}

func includeTags(docTags []*messages.Tag, userTags []string) bool {
	for _, tag := range docTags {
		s := tag.Name[1:]
		if slices.Contains(userTags, s) {
			return true
		}
	}
	return false
}
