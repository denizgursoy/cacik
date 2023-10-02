package runner

import (
	"bytes"
	"fmt"
	"os"
	"slices"

	gherkin "github.com/cucumber/gherkin/go/v26"
	messages "github.com/cucumber/messages/go/v21"
	"github.com/denizgursoy/cacik/pkg/gherkin_parser"
	"github.com/denizgursoy/cacik/pkg/models"
	"github.com/gofrs/uuid"
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

func (c *CucumberRunner) RunWithTags(userTags ...string) error {
	if len(c.featureDirectories) == 0 {
		c.featureDirectories = append(c.featureDirectories, ".")
	}

	featureFiles, err := gherkin_parser.SearchFeatureFilesIn(c.featureDirectories)
	if err != nil {
		return err
	}

	allPickles := make([]*messages.Pickle, 0)
	for _, file := range featureFiles {
		readFile, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("could not read file %s, error=%w", file, err)
		}
		document, err := gherkin_parser.ParseGherkinFile(bytes.NewReader(readFile))
		if err != nil {
			return fmt.Errorf("gherkin parse error in file %s, error=%w", file, err)
		}

		pickles := gherkin.Pickles(*document, document.Uri, name)
		allPickles = append(allPickles, pickles...)
	}

	fmt.Println(allPickles)
	return nil
}
func name() string {
	v4, _ := uuid.NewV4()
	return v4.String()
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
