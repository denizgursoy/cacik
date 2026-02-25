package cacik

import messages "github.com/cucumber/messages/go/v21"

// Scenario holds metadata about the currently executing scenario.
// Passed to BeforeScenario/AfterScenario hooks.
type Scenario struct {
	// Name is the scenario name as written in the .feature file.
	// For expanded Scenario Outlines this includes the substituted values
	// (e.g. "User login -- Valid credentials (#1)").
	Name string

	// Tags contains the tag names attached to this scenario
	// (e.g. "@smoke", "@wip"). Includes tags inherited from the Feature
	// or Examples block.
	Tags []string

	// Description is the optional free-text description below the
	// Scenario: line.
	Description string

	// Keyword is "Scenario" or "Scenario Outline".
	Keyword string

	// Line is the source file line number where the scenario is defined.
	Line int64
}

// Step holds metadata about the currently executing step.
// Passed to BeforeStep/AfterStep hooks.
type Step struct {
	// Keyword is the Gherkin keyword including trailing whitespace
	// (e.g. "Given ", "When ", "Then ", "And ", "But ").
	Keyword string

	// Text is the step text after the keyword
	// (e.g. "the user is logged in").
	Text string

	// Line is the source file line number where the step is defined.
	Line int64
}

// ScenarioFromMessage converts a parsed Gherkin Scenario message into a
// cacik.Scenario value suitable for hook functions.
func ScenarioFromMessage(s *messages.Scenario) Scenario {
	tags := make([]string, len(s.Tags))
	for i, t := range s.Tags {
		tags[i] = t.Name
	}
	var line int64
	if s.Location != nil {
		line = s.Location.Line
	}
	return Scenario{
		Name:        s.Name,
		Tags:        tags,
		Description: s.Description,
		Keyword:     s.Keyword,
		Line:        line,
	}
}

// StepFromMessage converts a parsed Gherkin Step message into a
// cacik.Step value suitable for hook functions.
func StepFromMessage(s *messages.Step) Step {
	var line int64
	if s.Location != nil {
		line = s.Location.Line
	}
	return Step{
		Keyword: s.Keyword,
		Text:    s.Text,
		Line:    line,
	}
}
