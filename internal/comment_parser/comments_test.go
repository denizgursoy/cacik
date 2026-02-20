package comment_parser

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/denizgursoy/cacik/internal/generator"
	"github.com/stretchr/testify/require"
)

var (
	expectedOutput = &generator.Output{
		ConfigFunction: &generator.FunctionLocator{
			FullPackageName: "github.com/denizgursoy/cacik/internal/parser/testdata",
			FunctionName:    "Method1",
		},
		StepFunctions: []*generator.StepFunctionLocator{
			{
				StepName: "^step 1$",
				FunctionLocator: &generator.FunctionLocator{
					FullPackageName: "github.com/denizgursoy/cacik/internal/parser/testdata/step-one",
					FunctionName:    "Step1",
				},
			},
			{
				StepName: "^step 2$",
				FunctionLocator: &generator.FunctionLocator{
					FullPackageName: "github.com/denizgursoy/cacik/internal/parser/testdata/step-two",
					FunctionName:    "Step2",
				},
			},
		},
	}
)

func TestGetComments(t *testing.T) {
	t.Run("parses step definitions from testdata", func(t *testing.T) {
		dir, err := os.Getwd()
		require.Nil(t, err)

		parser := NewGoSourceFileParser()
		recursively, err := parser.
			ParseFunctionCommentsOfGoFilesInDirectoryRecursively(context.Background(), filepath.Join(dir, "testdata"))

		require.Nil(t, err)

		// Check that config function is found
		require.NotNil(t, recursively.ConfigFunction)
		require.Equal(t, "Method1", recursively.ConfigFunction.FunctionName)

		// Check that step functions are found (order may vary)
		stepMap := make(map[string]string)
		for _, step := range recursively.StepFunctions {
			stepMap[step.FunctionName] = step.StepName
		}

		require.Equal(t, "^step 1$", stepMap["Step1"])
		require.Equal(t, "^step 2$", stepMap["Step2"])
		require.Equal(t, "^I have (\\d+) apples$", stepMap["IGetApples"])

		// Bool step definitions
		require.Equal(t, "^it is (true|false|yes|no|on|off|enabled|disabled)$", stepMap["ItIs"])
		require.Equal(t, "^the feature is (enabled|disabled)$", stepMap["FeatureToggle"])
	})
}
