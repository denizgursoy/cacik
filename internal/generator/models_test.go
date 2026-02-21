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
		Run()

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

func TestCustomType_RegexPattern(t *testing.T) {
	t.Run("generates case-insensitive pattern", func(t *testing.T) {
		ct := &CustomType{
			Name:       "Color",
			Underlying: "string",
			Values:     map[string]string{"Red": "red", "Blue": "blue"},
		}

		pattern := ct.RegexPattern()

		// Should use (?i:...) for case-insensitive matching
		require.True(t, strings.HasPrefix(pattern, "(?i:"))
		require.True(t, strings.HasSuffix(pattern, ")"))
		require.Contains(t, pattern, "red")
		require.Contains(t, pattern, "blue")
	})

	t.Run("includes both constant names and values", func(t *testing.T) {
		ct := &CustomType{
			Name:       "Priority",
			Underlying: "int",
			Values:     map[string]string{"Low": "1", "High": "3"},
		}

		pattern := ct.RegexPattern()

		// Should include both the constant names (lowercase) and values
		require.Contains(t, pattern, "low")
		require.Contains(t, pattern, "high")
		require.Contains(t, pattern, "1")
		require.Contains(t, pattern, "3")
	})

	t.Run("escapes regex special characters", func(t *testing.T) {
		ct := &CustomType{
			Name:       "Pattern",
			Underlying: "string",
			Values:     map[string]string{"Star": "*", "Plus": "+"},
		}

		pattern := ct.RegexPattern()

		// Special characters should be escaped
		require.Contains(t, pattern, "\\*")
		require.Contains(t, pattern, "\\+")
	})
}
