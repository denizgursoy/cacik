package runner

import (
	"bytes"
	"fmt"
	"os"
	"slices"

	messages "github.com/cucumber/messages/go/v21"
	"github.com/denizgursoy/cacik/pkg/gherkin_parser"
	"github.com/denizgursoy/cacik/pkg/models"
)

type (
	CucumberRunner struct {
		config             *models.Config
		featureDirectories []string
		steps              map[string]any
		executor           Executor
	}
)

func NewCucumberRunner(exec Executor) *CucumberRunner {
	return &CucumberRunner{
		steps:    make(map[string]any),
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

func (c *CucumberRunner) RegisterStep(definition string, function any) *CucumberRunner {
	if _, ok := c.steps[definition]; ok {
		panic(definition)
	}
	c.steps[definition] = function

	return c
}

type (
	Executed struct {
		Tags        []*messages.Tag
		Backgrounds []*messages.Background
		Steps       StepWithExamples
	}

	StepWithExamples struct {
		Steps    []*messages.Step
		Examples []*messages.Examples
	}
)

func (c *CucumberRunner) RunWithTags(userTags ...string) error {
	if len(c.featureDirectories) == 0 {
		c.featureDirectories = append(c.featureDirectories, ".")
	}

	featureFiles, err := gherkin_parser.SearchFeatureFilesIn(c.featureDirectories)
	if err != nil {
		return err
	}

	execs := make([]*Executed, 0)
	for _, file := range featureFiles {
		readFile, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("could not read file %s, error=%w", file, err)
		}
		document, err := gherkin_parser.ParseGherkinFile(bytes.NewReader(readFile))
		if err != nil {
			return fmt.Errorf("gherkin parse error in file %s, error=%w", file, err)
		}

		featureBackground := getBackground(document.Feature)
		featureTag := document.Feature.Tags

		for _, child := range document.Feature.Children {
			if child.Scenario != nil {
				ex := NewExecuted()
				ex.addTags(featureTag, child.Scenario.Tags)
				ex.addBackground(featureBackground)
				ex.addStepWithExample(child.Scenario.Steps, child.Scenario.Examples)
				execs = append(execs, ex)
			} else if child.Rule != nil {
				ruleTags := child.Rule.Tags
				ruleBackGround := getRuleBackground(child.Rule)
				for _, ruleChild := range child.Rule.Children {
					if ruleChild.Scenario != nil {
						ex := NewExecuted()
						ex.addTags(featureTag, ruleTags, ruleChild.Scenario.Tags)
						ex.addBackground(featureBackground, ruleBackGround)
						ex.addStepWithExample(ruleChild.Scenario.Steps, ruleChild.Scenario.Examples)
						execs = append(execs, ex)
					}

				}
			}

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

func NewExecuted() *Executed {
	return &Executed{
		Tags:        make([]*messages.Tag, 0),
		Backgrounds: make([]*messages.Background, 0),
	}
}

func (e *Executed) addTags(tags ...[]*messages.Tag) {
	for _, tag := range tags {
		e.Tags = append(e.Tags, tag...)
	}
}

func (e *Executed) addBackground(backgrounds ...*messages.Background) {
	for _, background := range backgrounds {
		e.Backgrounds = append(e.Backgrounds, background)
	}

}

func (e *Executed) addStepWithExample(steps []*messages.Step, examples []*messages.Examples) {
	e.Steps.Steps = steps
	e.Steps.Examples = examples
}
