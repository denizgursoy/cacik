package runner

import "github.com/denizgursoy/cacik/pkg/models"

type (
	CucumberRunner struct {
		config *models.Config
	}
)

func NewCucumberRunner() *CucumberRunner {
	return &CucumberRunner{}
}

func (c *CucumberRunner) SetConfigFunc(configFunction func() *models.Config) *CucumberRunner {
	if configFunction != nil {
		c.config = configFunction()
	}

	return c
}

func (c *CucumberRunner) RegisterStep(definition string, function any) *CucumberRunner {
	return c
}

func (c *CucumberRunner) Run(tags ...string) error {
	return nil
}
