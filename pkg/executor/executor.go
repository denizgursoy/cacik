package executor

import (
	"fmt"

	messages "github.com/cucumber/messages/go/v21"
)

type (
	StepExecutor struct {
	}
)

func (c *StepExecutor) Execute(*messages.GherkinDocument) error {
	//TODO implement me
	panic("implement me")
}

func NewStepExecutor() *StepExecutor {
	return &StepExecutor{}
}

func (c *StepExecutor) execute(document *messages.GherkinDocument) error {
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

func (c *StepExecutor) executeRule(rule *messages.Rule, featureBackground *messages.Background) {
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

func (c *StepExecutor) executeScenarioWithBackground(scenario *messages.Scenario, background *messages.Background) {

	c.executeBackground(background)

	isScenarioOutline := false
	if scenario.Examples != nil {
		isScenarioOutline = true
	}

	for _, step := range scenario.Steps {
		fmt.Println(step.Text, isScenarioOutline)
	}

}

func (c *StepExecutor) executeBackground(background *messages.Background) {
	if background != nil {
		for _, step := range background.Steps {
			fmt.Println(step.Text)
		}
	}
}
