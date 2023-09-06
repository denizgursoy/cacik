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
	t.Run("", func(t *testing.T) {
		dir, err := os.Getwd()
		require.Nil(t, err)

		parser := NewGoSourceFileParser()
		recursively, err := parser.ParseFunctionCommentsOfGoFilesInDirectoryRecursively(context.Background(), filepath.Join(dir, "testdata"))

		require.Nil(t, err)
		require.Equal(t, expectedOutput, recursively)
	})
}
