package runner

import "github.com/denizgursoy/cacik/pkg/models"

type (
	CucumberRunner struct {
		config             *models.Config
		featureDirectories []string
		tags               []string
	}
)

func NewCucumberRunner() *CucumberRunner {
	return &CucumberRunner{}
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
	return c
}

func (c *CucumberRunner) RunWithTags(tags ...string) error {
	c.tags = tags
	return nil
}
