package app

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOutput_Generate(t *testing.T) {

	t.Run("should generate correct output files", func(t *testing.T) {
		data := Output{
			ConfigFunction: &FunctionLocator{
				PackageName: "a",
				Name:        "ConfigFunction",
			},
			StepFunctions: []*StepFunctionLocator{
				{
					StepName: `"^step 1$"`,
					FunctionLocator: &FunctionLocator{
						PackageName: "package1",
						Name:        "Step1Function",
					},
				},
				{
					StepName: `"^step 2$"`,
					FunctionLocator: &FunctionLocator{
						PackageName: "package2",
						Name:        "Step2Function",
					},
				},
			},
		}

		expected := `package main

import (
	a "a"
	runner "github.com/denizgursoy/cacik/pkg/runner"
	"log"
)

func main() {
	err := runner.NewCucumberRunner().
		SetConfig(a.ConfigFunction).
		RegisterStep("^step 1$", Step1Function).
		RegisterStep("^step 2$", Step2Function).
		Run()

	if err != nil {
		log.Fatal(err)
	}
}
`

		builder := &strings.Builder{}
		err := data.Generate(data, builder)
		require.Nil(t, err)
		require.EqualValues(t, expected, builder.String())

	})

}
