package executor

import (
	"context"
	"errors"
	"testing"

	messages "github.com/cucumber/messages/go/v21"
	"github.com/stretchr/testify/require"
)

func TestStepExecutor_RegisterStep(t *testing.T) {
	t.Run("registers valid step", func(t *testing.T) {
		exec := NewStepExecutor()
		err := exec.RegisterStep("^I have (\\d+) apples$", func(ctx context.Context, count int) (context.Context, error) {
			return ctx, nil
		})
		require.NoError(t, err)
	})

	t.Run("returns error for invalid regex", func(t *testing.T) {
		exec := NewStepExecutor()
		err := exec.RegisterStep("[invalid", func() {})
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid step pattern")
	})

	t.Run("returns error for duplicate pattern", func(t *testing.T) {
		exec := NewStepExecutor()
		err := exec.RegisterStep("^test$", func() {})
		require.NoError(t, err)

		err = exec.RegisterStep("^test$", func() {})
		require.Error(t, err)
		require.Contains(t, err.Error(), "duplicate step pattern")
	})

	t.Run("returns error for non-function handler", func(t *testing.T) {
		exec := NewStepExecutor()
		err := exec.RegisterStep("^test$", "not a function")
		require.Error(t, err)
		require.Contains(t, err.Error(), "must be a function")
	})
}

func TestStepExecutor_ExecuteStep(t *testing.T) {
	t.Run("executes step with int argument", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedCount int

		err := exec.RegisterStep("^I have (\\d+) apples$", func(ctx context.Context, count int) (context.Context, error) {
			capturedCount = count
			return ctx, nil
		})
		require.NoError(t, err)

		// Create a simple scenario with the step
		doc := createDocument("I have 5 apples")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 5, capturedCount)
	})

	t.Run("executes step with string argument", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedName string

		err := exec.RegisterStep("^my name is (.+)$", func(ctx context.Context, name string) (context.Context, error) {
			capturedName = name
			return ctx, nil
		})
		require.NoError(t, err)

		doc := createDocument("my name is John")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, "John", capturedName)
	})

	t.Run("executes step with float argument", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedPrice float64

		err := exec.RegisterStep("^the price is ([\\d.]+)$", func(ctx context.Context, price float64) (context.Context, error) {
			capturedPrice = price
			return ctx, nil
		})
		require.NoError(t, err)

		doc := createDocument("the price is 19.99")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 19.99, capturedPrice)
	})

	t.Run("executes step with multiple arguments", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedCount int
		var capturedItem string

		err := exec.RegisterStep("^I have (\\d+) (\\w+)$", func(ctx context.Context, count int, item string) (context.Context, error) {
			capturedCount = count
			capturedItem = item
			return ctx, nil
		})
		require.NoError(t, err)

		doc := createDocument("I have 3 oranges")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 3, capturedCount)
		require.Equal(t, "oranges", capturedItem)
	})

	t.Run("executes step without context parameter", func(t *testing.T) {
		exec := NewStepExecutor()
		executed := false

		err := exec.RegisterStep("^simple step$", func() {
			executed = true
		})
		require.NoError(t, err)

		doc := createDocument("simple step")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.True(t, executed)
	})

	t.Run("propagates context between steps", func(t *testing.T) {
		exec := NewStepExecutor()
		type ctxKey string
		key := ctxKey("value")

		err := exec.RegisterStep("^I set value to (\\d+)$", func(ctx context.Context, val int) (context.Context, error) {
			return context.WithValue(ctx, key, val), nil
		})
		require.NoError(t, err)

		var capturedVal int
		err = exec.RegisterStep("^I read the value$", func(ctx context.Context) (context.Context, error) {
			capturedVal = ctx.Value(key).(int)
			return ctx, nil
		})
		require.NoError(t, err)

		doc := createDocumentWithSteps("I set value to 42", "I read the value")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 42, capturedVal)
	})

	t.Run("returns error when step function returns error", func(t *testing.T) {
		exec := NewStepExecutor()
		expectedErr := errors.New("step failed")

		err := exec.RegisterStep("^failing step$", func(ctx context.Context) (context.Context, error) {
			return ctx, expectedErr
		})
		require.NoError(t, err)

		doc := createDocument("failing step")
		err = exec.Execute(doc)
		require.Error(t, err)
		require.Contains(t, err.Error(), "step failed")
	})

	t.Run("returns error for unmatched step", func(t *testing.T) {
		exec := NewStepExecutor()
		err := exec.RegisterStep("^known step$", func() {})
		require.NoError(t, err)

		doc := createDocument("unknown step")
		err = exec.Execute(doc)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no matching step definition")
	})

	t.Run("returns error for type conversion failure", func(t *testing.T) {
		exec := NewStepExecutor()

		err := exec.RegisterStep("^I have (\\w+) apples$", func(ctx context.Context, count int) (context.Context, error) {
			return ctx, nil
		})
		require.NoError(t, err)

		doc := createDocument("I have many apples")
		err = exec.Execute(doc)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to convert")
	})
}

func TestStepExecutor_Execute_Background(t *testing.T) {
	t.Run("executes background before scenario", func(t *testing.T) {
		exec := NewStepExecutor()
		order := []string{}

		err := exec.RegisterStep("^background step$", func() {
			order = append(order, "background")
		})
		require.NoError(t, err)

		err = exec.RegisterStep("^scenario step$", func() {
			order = append(order, "scenario")
		})
		require.NoError(t, err)

		doc := createDocumentWithBackground("background step", "scenario step")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, []string{"background", "scenario"}, order)
	})
}

func TestStepExecutor_BoolArgument(t *testing.T) {
	testCases := []struct {
		name     string
		stepText string
		expected bool
	}{
		// Standard boolean values
		{"true", "it is true", true},
		{"false", "it is false", false},
		{"TRUE (uppercase)", "it is TRUE", true},
		{"FALSE (uppercase)", "it is FALSE", false},
		{"True (mixed case)", "it is True", true},
		{"False (mixed case)", "it is False", false},

		// Yes/No
		{"yes", "it is yes", true},
		{"no", "it is no", false},
		{"YES (uppercase)", "it is YES", true},
		{"NO (uppercase)", "it is NO", false},

		// On/Off
		{"on", "it is on", true},
		{"off", "it is off", false},
		{"ON (uppercase)", "it is ON", true},
		{"OFF (uppercase)", "it is OFF", false},

		// Enabled/Disabled
		{"enabled", "it is enabled", true},
		{"disabled", "it is disabled", false},
		{"ENABLED (uppercase)", "it is ENABLED", true},
		{"DISABLED (uppercase)", "it is DISABLED", false},

		// Numeric
		{"1", "it is 1", true},
		{"0", "it is 0", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			exec := NewStepExecutor()
			var capturedValue bool

			err := exec.RegisterStep("^it is (.+)$", func(ctx context.Context, value bool) (context.Context, error) {
				capturedValue = value
				return ctx, nil
			})
			require.NoError(t, err)

			doc := createDocument(tc.stepText)
			err = exec.Execute(doc)
			require.NoError(t, err)
			require.Equal(t, tc.expected, capturedValue)
		})
	}

	t.Run("returns error for invalid bool value", func(t *testing.T) {
		exec := NewStepExecutor()

		err := exec.RegisterStep("^it is (.+)$", func(ctx context.Context, value bool) (context.Context, error) {
			return ctx, nil
		})
		require.NoError(t, err)

		doc := createDocument("it is maybe")
		err = exec.Execute(doc)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot parse")
	})
}

func TestStepExecutor_FeatureToggle(t *testing.T) {
	t.Run("feature is enabled", func(t *testing.T) {
		exec := NewStepExecutor()
		var featureEnabled bool

		err := exec.RegisterStep("^the feature is (enabled|disabled)$", func(ctx context.Context, enabled bool) (context.Context, error) {
			featureEnabled = enabled
			return ctx, nil
		})
		require.NoError(t, err)

		doc := createDocument("the feature is enabled")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.True(t, featureEnabled)
	})

	t.Run("feature is disabled", func(t *testing.T) {
		exec := NewStepExecutor()
		var featureEnabled bool

		err := exec.RegisterStep("^the feature is (enabled|disabled)$", func(ctx context.Context, enabled bool) (context.Context, error) {
			featureEnabled = enabled
			return ctx, nil
		})
		require.NoError(t, err)

		doc := createDocument("the feature is disabled")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.False(t, featureEnabled)
	})
}

// Custom type for testing
type Color string

func TestStepExecutor_CustomStringType(t *testing.T) {
	t.Run("converts string to custom type", func(t *testing.T) {
		exec := NewStepExecutor()

		// Register the custom type with allowed values
		exec.RegisterCustomType("Color", "string", map[string]string{
			"red":   "red",
			"blue":  "blue",
			"green": "green",
		})

		var capturedColor Color

		err := exec.RegisterStep("^I select (red|blue|green)$", func(ctx context.Context, c Color) (context.Context, error) {
			capturedColor = c
			return ctx, nil
		})
		require.NoError(t, err)

		doc := createDocument("I select red")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, Color("red"), capturedColor)
	})

	t.Run("case-insensitive matching with explicit pattern", func(t *testing.T) {
		exec := NewStepExecutor()

		// Register with lowercase keys for case-insensitive matching
		exec.RegisterCustomType("Color", "string", map[string]string{
			"red":  "red",
			"blue": "blue",
		})

		var capturedColor Color

		err := exec.RegisterStep("^I select (red|blue|RED|BLUE)$", func(ctx context.Context, c Color) (context.Context, error) {
			capturedColor = c
			return ctx, nil
		})
		require.NoError(t, err)

		doc := createDocument("I select RED")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, Color("red"), capturedColor) // Should be lowercase
	})

	t.Run("case-insensitive matching with (?i:) pattern", func(t *testing.T) {
		exec := NewStepExecutor()

		// Register with lowercase keys for case-insensitive matching
		exec.RegisterCustomType("Color", "string", map[string]string{
			"red":  "red",
			"blue": "blue",
		})

		var capturedColor Color

		// Use (?i:...) for case-insensitive matching - this is what the generator produces
		err := exec.RegisterStep("^I select ((?i:red|blue))$", func(ctx context.Context, c Color) (context.Context, error) {
			capturedColor = c
			return ctx, nil
		})
		require.NoError(t, err)

		// Test various case combinations
		testCases := []struct {
			input    string
			expected Color
		}{
			{"RED", Color("red")},
			{"Red", Color("red")},
			{"red", Color("red")},
			{"rEd", Color("red")},
			{"BLUE", Color("blue")},
			{"Blue", Color("blue")},
			{"blue", Color("blue")},
		}
		for _, tc := range testCases {
			doc := createDocument("I select " + tc.input)
			err = exec.Execute(doc)
			require.NoError(t, err, "Failed for: %s", tc.input)
			require.Equal(t, tc.expected, capturedColor, "Wrong color for: %s", tc.input)
		}
	})

	t.Run("rejects invalid value", func(t *testing.T) {
		exec := NewStepExecutor()

		exec.RegisterCustomType("Color", "string", map[string]string{
			"red":  "red",
			"blue": "blue",
		})

		err := exec.RegisterStep("^I select (\\w+)$", func(ctx context.Context, c Color) (context.Context, error) {
			return ctx, nil
		})
		require.NoError(t, err)

		doc := createDocument("I select purple")
		err = exec.Execute(doc)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid Color")
		require.Contains(t, err.Error(), "purple")
	})
}

// Custom type for testing int-based enums
type Priority int

func TestStepExecutor_CustomIntType(t *testing.T) {
	t.Run("converts string to custom int type by name", func(t *testing.T) {
		exec := NewStepExecutor()

		// Register with both names and values
		exec.RegisterCustomType("Priority", "int", map[string]string{
			"low":    "1",
			"medium": "2",
			"high":   "3",
			"1":      "1",
			"2":      "2",
			"3":      "3",
		})

		var capturedPriority Priority

		err := exec.RegisterStep("^priority is (low|medium|high|1|2|3)$", func(ctx context.Context, p Priority) (context.Context, error) {
			capturedPriority = p
			return ctx, nil
		})
		require.NoError(t, err)

		// Test with name
		doc := createDocument("priority is high")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, Priority(3), capturedPriority)
	})

	t.Run("converts string to custom int type by value", func(t *testing.T) {
		exec := NewStepExecutor()

		exec.RegisterCustomType("Priority", "int", map[string]string{
			"low":    "1",
			"medium": "2",
			"high":   "3",
			"1":      "1",
			"2":      "2",
			"3":      "3",
		})

		var capturedPriority Priority

		err := exec.RegisterStep("^priority is (low|medium|high|1|2|3)$", func(ctx context.Context, p Priority) (context.Context, error) {
			capturedPriority = p
			return ctx, nil
		})
		require.NoError(t, err)

		// Test with numeric value
		doc := createDocument("priority is 2")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, Priority(2), capturedPriority)
	})
}

func TestStepExecutor_CustomTypeWithoutRegistration(t *testing.T) {
	t.Run("custom type without registration still works", func(t *testing.T) {
		exec := NewStepExecutor()

		// Don't register the custom type - it should still convert
		var capturedColor Color

		err := exec.RegisterStep("^I select (\\w+)$", func(ctx context.Context, c Color) (context.Context, error) {
			capturedColor = c
			return ctx, nil
		})
		require.NoError(t, err)

		doc := createDocument("I select anything")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, Color("anything"), capturedColor)
	})
}

// Helper functions to create test documents

func createDocument(stepText string) *messages.GherkinDocument {
	return createDocumentWithSteps(stepText)
}

func createDocumentWithSteps(stepTexts ...string) *messages.GherkinDocument {
	steps := make([]*messages.Step, len(stepTexts))
	for i, text := range stepTexts {
		steps[i] = &messages.Step{
			Text: text,
		}
	}

	return &messages.GherkinDocument{
		Feature: &messages.Feature{
			Children: []*messages.FeatureChild{
				{
					Scenario: &messages.Scenario{
						Steps: steps,
					},
				},
			},
		},
	}
}

func createDocumentWithBackground(backgroundStep, scenarioStep string) *messages.GherkinDocument {
	return &messages.GherkinDocument{
		Feature: &messages.Feature{
			Children: []*messages.FeatureChild{
				{
					Background: &messages.Background{
						Steps: []*messages.Step{
							{Text: backgroundStep},
						},
					},
				},
				{
					Scenario: &messages.Scenario{
						Steps: []*messages.Step{
							{Text: scenarioStep},
						},
					},
				},
			},
		},
	}
}
