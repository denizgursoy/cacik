package generator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	data = Output{
		ConfigFunction: &FunctionLocator{
			FullPackageName: "a",
			FunctionName:    "ConfigFunction",
		},
		StepFunctions: []*StepFunctionLocator{
			{
				StepName: "^step 1$",
				FunctionLocator: &FunctionLocator{
					FullPackageName: "package1",
					FunctionName:    "Step1Function",
				},
			},
			{
				StepName: "^step 2$",
				FunctionLocator: &FunctionLocator{
					FullPackageName: "package2",
					FunctionName:    "Step2Function",
				},
			},
		},
	}

	expected = `package main

import (
	a "a"
	runner "github.com/denizgursoy/cacik/pkg/runner"
	"log"
	package1 "package1"
	package2 "package2"
)

func main() {
	err := runner.NewCucumberRunner().
		WithConfigFunc(a.ConfigFunction).
		RegisterStep("^step 1$", package1.Step1Function).
		RegisterStep("^step 2$", package2.Step2Function).
		RunWithTags()

	if err != nil {
		log.Fatal(err)
	}
}
`
)

func TestOutput_Generate(t *testing.T) {
	t.Run("should generate correct output files", func(t *testing.T) {
		builder := &strings.Builder{}
		err := data.Generate(builder)

		require.Nil(t, err)
		require.EqualValues(t, expected, builder.String())
	})
}
