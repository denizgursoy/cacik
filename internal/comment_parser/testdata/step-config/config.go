package step_config

import "github.com/denizgursoy/cacik/pkg/cacik"

// MyConfig returns configuration settings
func MyConfig() *cacik.Config {
	return &cacik.Config{
		FailFast: true,
	}
}

// MyHooks returns lifecycle hooks
func MyHooks() *cacik.Hooks {
	return &cacik.Hooks{
		Order: 10,
		BeforeAll: func() {
			// setup
		},
		BeforeScenario: func(s cacik.Scenario) {
			// runs before each scenario
			_ = s.Name
		},
		AfterScenario: func(s cacik.Scenario, err error) {
			// runs after each scenario (always runs)
			_ = s.Name
			_ = err
		},
		BeforeStep: func(s cacik.Step) {
			// runs before each step
			_ = s.Text
		},
		AfterStep: func(s cacik.Step, err error) {
			// runs after each step
			_ = s.Text
			_ = err
		},
	}
}
