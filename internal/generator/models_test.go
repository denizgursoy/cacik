package generator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	data = Output{
		ConfigFunctions: []*FunctionLocator{
			{
				FullPackageName: "a",
				FunctionName:    "ConfigFunction",
			},
		},
		HooksFunctions: []*FunctionLocator{
			{
				FullPackageName: "b",
				FunctionName:    "HooksFunction",
			},
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
	b "b"
	cacik "github.com/denizgursoy/cacik/pkg/cacik"
	runner "github.com/denizgursoy/cacik/pkg/runner"
	package1 "package1"
	package2 "package2"
	"testing"
)

func TestCacik(t *testing.T) {
	config := cacik.MergeConfigs(a.ConfigFunction())
	hooks := []*cacik.Hooks{b.HooksFunction()}
	err := runner.NewCucumberRunner(t).
		WithConfig(config).
		WithHooks(hooks...).
		RegisterStep("^step 1$", package1.Step1Function).
		RegisterStep("^step 2$", package2.Step2Function).
		Run()
	if err != nil {
		t.Fatal(err)
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

func TestOutput_Generate_TestMode(t *testing.T) {
	t.Run("should generate test file with TestCacik and NewCucumberRunner(t)", func(t *testing.T) {
		testData := Output{
			PackageName:        "myapp",
			CurrentPackagePath: "github.com/example/myapp",
			ConfigFunctions: []*FunctionLocator{
				{
					FullPackageName: "github.com/example/myapp/config",
					FunctionName:    "ConfigFunction",
				},
			},
			HooksFunctions: []*FunctionLocator{
				{
					FullPackageName: "github.com/example/myapp/hooks",
					FunctionName:    "HooksFunction",
				},
			},
			StepFunctions: []*StepFunctionLocator{
				{
					StepName: "^step 1$",
					FunctionLocator: &FunctionLocator{
						FullPackageName: "github.com/example/myapp/steps",
						FunctionName:    "Step1Function",
					},
				},
			},
		}

		builder := &strings.Builder{}
		err := testData.Generate(builder)

		require.Nil(t, err)
		output := builder.String()

		// Should use the detected package name, not "main"
		require.Contains(t, output, "package myapp")
		require.NotContains(t, output, "package main")

		// Should use func TestCacik(t *testing.T) instead of func main()
		require.Contains(t, output, "func TestCacik(t *testing.T)")
		require.NotContains(t, output, "func main()")

		// Should import "testing" instead of "log"
		require.Contains(t, output, `"testing"`)
		require.NotContains(t, output, `"log"`)

		// Should use t.Fatal(err) instead of log.Fatal(err)
		require.Contains(t, output, "t.Fatal(err)")
		require.NotContains(t, output, "log.Fatal(err)")

		// Should pass t to NewCucumberRunner constructor
		require.Contains(t, output, "NewCucumberRunner(t)")
		require.NotContains(t, output, "WithTestingT")
	})

	t.Run("should call same-package functions without import qualifier", func(t *testing.T) {
		testData := Output{
			PackageName:        "myapp",
			CurrentPackagePath: "github.com/example/myapp",
			ConfigFunctions: []*FunctionLocator{
				{
					FullPackageName: "github.com/example/myapp", // same package
					FunctionName:    "GetConfig",
				},
			},
			HooksFunctions: []*FunctionLocator{
				{
					FullPackageName: "github.com/example/myapp", // same package
					FunctionName:    "GetHooks",
				},
			},
			StepFunctions: []*StepFunctionLocator{
				{
					StepName: "^local step$",
					FunctionLocator: &FunctionLocator{
						FullPackageName: "github.com/example/myapp", // same package
						FunctionName:    "LocalStep",
					},
				},
				{
					StepName: "^external step$",
					FunctionLocator: &FunctionLocator{
						FullPackageName: "github.com/example/other", // different package
						FunctionName:    "ExternalStep",
					},
				},
			},
		}

		builder := &strings.Builder{}
		err := testData.Generate(builder)

		require.Nil(t, err)
		output := builder.String()

		// Same-package functions should be called directly (no import qualifier)
		require.Contains(t, output, "GetConfig()")
		require.NotContains(t, output, "myapp.GetConfig()")

		require.Contains(t, output, "GetHooks()")
		require.NotContains(t, output, "myapp.GetHooks()")

		require.Contains(t, output, "LocalStep)")
		require.NotContains(t, output, "myapp.LocalStep")

		// External functions should still have a qualifier
		require.Contains(t, output, "other.ExternalStep")

		// Should NOT import the current package
		require.NotContains(t, output, `"github.com/example/myapp"`)
		// Should import the external package
		require.Contains(t, output, `"github.com/example/other"`)
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
