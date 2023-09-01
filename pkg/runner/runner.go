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

func (c *CucumberRunner) SetConfig(runner *models.Config) *CucumberRunner {
	c.config = runner

	return c
}

func (c *CucumberRunner) RegisterStep(definition string, function any) *CucumberRunner {
	return c
}

func (c *CucumberRunner) Run() error {
	return nil
}
