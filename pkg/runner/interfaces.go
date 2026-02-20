//go:generate mockgen -source=interfaces.go -destination=interfaces_mock.go -package=runner
package runner

import messages "github.com/cucumber/messages/go/v21"

type (
	// Executor executes gherkin documents
	Executor interface {
		Execute(*messages.GherkinDocument) error
	}

	// StepRegistry allows registering step definitions
	StepRegistry interface {
		RegisterStep(pattern string, fn any) error
	}
)
