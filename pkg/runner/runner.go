package runner

import (
	"bytes"
	"fmt"
	"os"

	messages "github.com/cucumber/messages/go/v21"
	"github.com/denizgursoy/cacik/pkg/gherkin_parser"
	"github.com/denizgursoy/cacik/pkg/models"
)

type (
	CucumberRunner struct {
		config             *models.Config
		featureDirectories []string
		tags               []string
		steps              map[string]any
	}
)

func NewCucumberRunner() *CucumberRunner {
	return &CucumberRunner{
		steps: make(map[string]any),
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

func (c *CucumberRunner) RegisterStep(definition string, function any) *CucumberRunner {
	if _, ok := c.steps[definition]; ok {
		panic(definition)
	}
	c.steps[definition] = function

	return c
}

func (c *CucumberRunner) RunWithTags(tags ...string) error {
	c.tags = tags

	if len(c.featureDirectories) == 0 {
		c.featureDirectories = append(c.featureDirectories, "./")
	}

	featureFiles, err := gherkin_parser.SearchFeatureFilesIn(c.featureDirectories)
	if err != nil {
		return err
	}
	for _, file := range featureFiles {
		readFile, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		document, err := gherkin_parser.ParseGherkinFile(bytes.NewReader(readFile))
		if err != nil {
			return err
		}
		err = c.execute(document)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *CucumberRunner) execute(document *messages.GherkinDocument) error {
	var featureBackground *messages.Background

	for _, child := range document.Feature.Children {
		if child.Background != nil {
			featureBackground = child.Background
		} else if child.Rule != nil {
			c.executeRule(child.Rule, featureBackground)
		} else {
			c.executeScenarioWithBackground(child.Scenario, featureBackground)
		}
	}

	return nil
}

func (c *CucumberRunner) executeRule(rule *messages.Rule, featureBackground *messages.Background) {
	var ruleBackground *messages.Background

	for _, child := range rule.Children {
		if child.Background != nil {
			ruleBackground = child.Background
		} else {
			c.executeBackground(featureBackground)
			c.executeScenarioWithBackground(child.Scenario, ruleBackground)
		}
	}
}

func (c *CucumberRunner) executeScenarioWithBackground(scenario *messages.Scenario, background *messages.Background) {

	c.executeBackground(background)

	isScenarioOutline := false
	if scenario.Examples != nil {
		isScenarioOutline = true
	}

	for _, step := range scenario.Steps {
		fmt.Println(step.Text, isScenarioOutline)
	}

}

func (c *CucumberRunner) executeBackground(background *messages.Background) {
	if background != nil {
		for _, step := range background.Steps {
			fmt.Println(step.Text)
		}
	}
}
