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

		// Custom type step definitions - Color (string-based)
		// {color} should be transformed to (blue|green|red) - sorted alphabetically
		require.Contains(t, stepMap["SelectColor"], "blue")
		require.Contains(t, stepMap["SelectColor"], "green")
		require.Contains(t, stepMap["SelectColor"], "red")

		// Custom type step definitions - Priority (int-based)
		// {priority} should include both names and values: high, low, medium, 1, 2, 3
		require.Contains(t, stepMap["SetPriority"], "low")
		require.Contains(t, stepMap["SetPriority"], "medium")
		require.Contains(t, stepMap["SetPriority"], "high")
		require.Contains(t, stepMap["SetPriority"], "1")
		require.Contains(t, stepMap["SetPriority"], "2")
		require.Contains(t, stepMap["SetPriority"], "3")

		// Built-in type step definitions
		require.Equal(t, `^I have (-?\d+) apples$`, stepMap["HaveApples"])
		require.Equal(t, `^the price is (-?\d*\.?\d+)$`, stepMap["PriceIs"])
		require.Equal(t, `^my name is (\w+)$`, stepMap["NameIs"])
		require.Equal(t, `^I say "([^"]*)"$`, stepMap["Say"])
		require.Equal(t, `^I see (.*)$`, stepMap["SeeAnything"])

		// Mixed type step definitions - verify they contain expected patterns
		// WantColoredVehicle: {color} (car|bike) {int} {float}
		require.Contains(t, stepMap["WantColoredVehicle"], "(car|bike)")
		require.Contains(t, stepMap["WantColoredVehicle"], "(?i:")          // case-insensitive color
		require.Contains(t, stepMap["WantColoredVehicle"], `(-?\d+)`)       // int
		require.Contains(t, stepMap["WantColoredVehicle"], `(-?\d*\.?\d+)`) // float

		// NamedItemWithPriority: {color} {string} {priority}
		require.Contains(t, stepMap["NamedItemWithPriority"], "(?i:")
		require.Contains(t, stepMap["NamedItemWithPriority"], `"([^"]*)"`) // string

		// SizedItemCount: {int} {size} {color}
		require.Contains(t, stepMap["SizedItemCount"], `(-?\d+)`)
		require.Contains(t, stepMap["SizedItemCount"], "(?i:") // size and color are case-insensitive
	})
}

func TestCustomTypeParsing(t *testing.T) {
	t.Run("parses Color custom type", func(t *testing.T) {
		dir, err := os.Getwd()
		require.Nil(t, err)

		parser := NewGoSourceFileParser()
		output, err := parser.
			ParseFunctionCommentsOfGoFilesInDirectoryRecursively(context.Background(), filepath.Join(dir, "testdata"))

		require.Nil(t, err)

		// Check Color custom type
		colorType, ok := output.CustomTypes["color"]
		require.True(t, ok, "Color type should be found")
		require.Equal(t, "Color", colorType.Name)
		require.Equal(t, "string", colorType.Underlying)
		require.Equal(t, "red", colorType.Values["Red"])
		require.Equal(t, "blue", colorType.Values["Blue"])
		require.Equal(t, "green", colorType.Values["Green"])
	})

	t.Run("parses Priority custom type", func(t *testing.T) {
		dir, err := os.Getwd()
		require.Nil(t, err)

		parser := NewGoSourceFileParser()
		output, err := parser.
			ParseFunctionCommentsOfGoFilesInDirectoryRecursively(context.Background(), filepath.Join(dir, "testdata"))

		require.Nil(t, err)

		// Check Priority custom type
		priorityType, ok := output.CustomTypes["priority"]
		require.True(t, ok, "Priority type should be found")
		require.Equal(t, "Priority", priorityType.Name)
		require.Equal(t, "int", priorityType.Underlying)
		require.Equal(t, "1", priorityType.Values["Low"])
		require.Equal(t, "2", priorityType.Values["Medium"])
		require.Equal(t, "3", priorityType.Values["High"])
	})
}

func TestTransformStepPattern(t *testing.T) {
	t.Run("transforms {color} to regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{
			"color": {
				Name:       "Color",
				Underlying: "string",
				Values:     map[string]string{"Red": "red", "Blue": "blue"},
			},
		}

		result, err := transformStepPattern("^I select {color}$", customTypes)
		require.Nil(t, err)
		require.Contains(t, result, "blue")
		require.Contains(t, result, "red")
		require.Contains(t, result, "(")
		require.Contains(t, result, ")")
	})

	t.Run("returns error for unknown type", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		_, err := transformStepPattern("^I select {unknown}$", customTypes)
		require.NotNil(t, err)
		require.Contains(t, err.Error(), "unknown parameter type")
	})

	t.Run("returns error for type with no constants", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{
			"empty": {
				Name:       "Empty",
				Underlying: "string",
				Values:     map[string]string{},
			},
		}

		_, err := transformStepPattern("^I select {empty}$", customTypes)
		require.NotNil(t, err)
		require.Contains(t, err.Error(), "no defined constants")
	})

	t.Run("handles multiple custom types in pattern", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{
			"color": {
				Name:       "Color",
				Underlying: "string",
				Values:     map[string]string{"Red": "red"},
			},
			"size": {
				Name:       "Size",
				Underlying: "string",
				Values:     map[string]string{"Large": "large"},
			},
		}

		result, err := transformStepPattern("^I want {color} and {size}$", customTypes)
		require.Nil(t, err)
		require.Contains(t, result, "red")
		require.Contains(t, result, "large")
	})

	// Built-in parameter type tests
	t.Run("transforms {int} to regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^I have {int} apples$", customTypes)
		require.Nil(t, err)
		require.Equal(t, `^I have (-?\d+) apples$`, result)
	})

	t.Run("transforms {float} to regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^the price is {float}$", customTypes)
		require.Nil(t, err)
		require.Equal(t, `^the price is (-?\d*\.?\d+)$`, result)
	})

	t.Run("transforms {word} to regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^my name is {word}$", customTypes)
		require.Nil(t, err)
		require.Equal(t, `^my name is (\w+)$`, result)
	})

	t.Run("transforms {string} to regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^I say {string}$", customTypes)
		require.Nil(t, err)
		require.Equal(t, `^I say "([^"]*)"$`, result)
	})

	t.Run("transforms {} (empty) to regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^I have {} items$", customTypes)
		require.Nil(t, err)
		require.Equal(t, `^I have (.*) items$`, result)
	})

	t.Run("transforms {any} to regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^I see {any}$", customTypes)
		require.Nil(t, err)
		require.Equal(t, `^I see (.*)$`, result)
	})

	t.Run("transforms {time} to regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^the meeting is at {time}$", customTypes)
		require.Nil(t, err)
		require.Contains(t, result, `\d{1,2}:\d{2}`)
	})

	t.Run("transforms {date} to regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^the event is on {date}$", customTypes)
		require.Nil(t, err)
		require.Contains(t, result, `\d{4}[-/]\d{2}[-/]\d{2}`)
	})

	t.Run("transforms {datetime} to regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^the appointment is at {datetime}$", customTypes)
		require.Nil(t, err)
		require.Contains(t, result, `\d{4}[-/]\d{2}[-/]\d{2}`)
		require.Contains(t, result, `\d{1,2}:\d{2}`)
	})

	t.Run("transforms {timezone} to regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^convert to {timezone}$", customTypes)
		require.Nil(t, err)
		// Should contain patterns for Z, UTC, offset, and IANA names
		require.Contains(t, result, "Z")
		require.Contains(t, result, "UTC")
		require.Contains(t, result, `[+-]\d{2}`)
		require.Contains(t, result, `[A-Za-z_]+/[A-Za-z_]+`)
	})

	t.Run("time pattern includes optional timezone", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^meeting at {time}$", customTypes)
		require.Nil(t, err)
		// Should contain timezone patterns as optional
		require.Contains(t, result, "Z|UTC")
		require.Contains(t, result, `[A-Za-z_]+/[A-Za-z_]+`)
	})

	t.Run("datetime pattern includes optional timezone", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^appointment at {datetime}$", customTypes)
		require.Nil(t, err)
		// Should contain timezone patterns as optional
		require.Contains(t, result, "Z|UTC")
		require.Contains(t, result, `[A-Za-z_]+/[A-Za-z_]+`)
	})

	t.Run("transforms {email} to regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^user {email} logged in$", customTypes)
		require.Nil(t, err)
		require.Equal(t, `^user ([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}) logged in$`, result)
	})

	t.Run("transforms {duration} to regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^wait for {duration}$", customTypes)
		require.Nil(t, err)
		require.Equal(t, `^wait for (-?(?:\d+\.?\d*(?:ns|us|µs|ms|s|m|h))+)$`, result)
	})

	t.Run("transforms {url} to regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^navigate to {url}$", customTypes)
		require.Nil(t, err)
		require.Equal(t, `^navigate to (https?://[^\s]+)$`, result)
	})

	t.Run("handles mixed built-in and custom types", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{
			"color": {
				Name:       "Color",
				Underlying: "string",
				Values:     map[string]string{"Red": "red"},
			},
		}

		result, err := transformStepPattern("^I have {int} {color} items$", customTypes)
		require.Nil(t, err)
		require.Contains(t, result, `(-?\d+)`)
		require.Contains(t, result, "red")
	})

	t.Run("handles complex pattern with custom type, built-in types, and regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{
			"color": {
				Name:       "Color",
				Underlying: "string",
				Values:     map[string]string{"Red": "red", "Blue": "blue", "Green": "green"},
			},
			"priority": {
				Name:       "Priority",
				Underlying: "int",
				Values:     map[string]string{"Low": "1", "Medium": "2", "High": "3"},
			},
		}

		// Pattern: custom type + word + int + float + string + another custom type
		result, err := transformStepPattern(
			"^I want a {color} (car|bike) with {int} doors costing {float} dollars named {string} at {priority} priority$",
			customTypes,
		)
		require.Nil(t, err)

		// Verify custom type {color} is transformed with case-insensitive matching
		require.Contains(t, result, "(?i:")
		require.Contains(t, result, "red")
		require.Contains(t, result, "blue")
		require.Contains(t, result, "green")

		// Verify normal regex (car|bike) is preserved
		require.Contains(t, result, "(car|bike)")

		// Verify built-in {int} is transformed
		require.Contains(t, result, `(-?\d+)`)

		// Verify built-in {float} is transformed
		require.Contains(t, result, `(-?\d*\.?\d+)`)

		// Verify built-in {string} is transformed
		require.Contains(t, result, `"([^"]*)"`)

		// Verify custom type {priority} is transformed
		require.Contains(t, result, "low")
		require.Contains(t, result, "medium")
		require.Contains(t, result, "high")
		require.Contains(t, result, "1")
		require.Contains(t, result, "2")
		require.Contains(t, result, "3")
	})
}

func TestDuplicateStepDetection(t *testing.T) {
	t.Run("returns error for duplicate step patterns", func(t *testing.T) {
		dir, err := os.Getwd()
		require.Nil(t, err)

		parser := NewGoSourceFileParser()
		_, err = parser.ParseFunctionCommentsOfGoFilesInDirectoryRecursively(
			context.Background(),
			filepath.Join(dir, "duplicate"),
		)

		require.NotNil(t, err)
		require.Contains(t, err.Error(), "duplicate step pattern")
		require.Contains(t, err.Error(), "I have")
		require.Contains(t, err.Error(), "items")
		require.Contains(t, err.Error(), "FirstDuplicateStep")
		require.Contains(t, err.Error(), "SecondDuplicateStep")
	})
}
