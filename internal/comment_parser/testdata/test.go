package testdata

import "github.com/denizgursoy/cacik/pkg/cacik"

// MyConfig returns configuration settings
func MyConfig() *cacik.Config {
	return &cacik.Config{
		Parallel: 4,
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
	}
}
