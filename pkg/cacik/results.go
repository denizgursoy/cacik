package cacik

import "time"

// StepStatus represents the execution outcome of a step.
type StepStatus int

const (
	// StepPassed indicates the step executed successfully.
	StepPassed StepStatus = iota
	// StepFailed indicates the step failed (assertion, panic, or returned error).
	StepFailed
	// StepSkipped indicates the step was skipped due to an earlier failure.
	StepSkipped
)

// String returns a human-readable label for the step status.
func (s StepStatus) String() string {
	switch s {
	case StepPassed:
		return "passed"
	case StepFailed:
		return "failed"
	case StepSkipped:
		return "skipped"
	default:
		return "unknown"
	}
}

// StepResult holds the execution result of a single step.
type StepResult struct {
	// Keyword is the Gherkin keyword including trailing whitespace
	// (e.g. "Given ", "When ", "Then ").
	Keyword string

	// Text is the step text after the keyword.
	Text string

	// Status is the execution outcome (passed, failed, or skipped).
	Status StepStatus

	// Error is the error message when the step failed. Empty for passed/skipped.
	Error string

	// Duration is the wall-clock execution time. Zero for skipped steps.
	Duration time.Duration

	// StartedAt is when the step started executing. Zero for skipped steps.
	StartedAt time.Time

	// MatchLocs holds pairs of [start, end] byte offsets for each capture
	// group within Text (same format as regexp.FindStringSubmatchIndex,
	// minus the full-match pair). Used for parameter highlighting in reports.
	MatchLocs []int
}

// ScenarioResult holds the execution result of a single scenario.
type ScenarioResult struct {
	// FeatureName is the name of the parent feature.
	FeatureName string

	// RuleName is the name of the parent rule. Empty if the scenario is not
	// inside a rule.
	RuleName string

	// Name is the scenario name as written in the .feature file.
	Name string

	// Tags contains the tag names attached to this scenario (including
	// inherited tags from Feature and Rule).
	Tags []string

	// Passed is true when all steps passed.
	Passed bool

	// Error is the error message when the scenario failed. Empty if passed.
	Error string

	// Duration is the wall-clock execution time of the scenario (including
	// hooks and all steps).
	Duration time.Duration

	// StartedAt is when the scenario started executing.
	StartedAt time.Time

	// FeatureBgSteps contains the results of feature-level background steps.
	FeatureBgSteps []StepResult

	// RuleBgSteps contains the results of rule-level background steps.
	RuleBgSteps []StepResult

	// Steps contains the results of the scenario's own steps (excluding
	// background steps).
	Steps []StepResult
}

// RunResult holds the complete results of a test run.
type RunResult struct {
	// Scenarios contains the result of every executed scenario.
	Scenarios []ScenarioResult

	// Summary holds aggregate pass/fail/skip counters.
	Summary ReporterSummary

	// Duration is the total wall-clock time for the entire run.
	Duration time.Duration

	// StartedAt is when the run started.
	StartedAt time.Time
}
