package executor

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	messages "github.com/cucumber/messages/go/v21"
	"github.com/denizgursoy/cacik/pkg/cacik"
	"github.com/stretchr/testify/require"
)

func TestStepExecutor_RegisterStep(t *testing.T) {
	t.Run("registers valid step", func(t *testing.T) {
		exec := NewStepExecutor()
		err := exec.RegisterStep("^I have (\\d+) apples$", func(ctx *cacik.Context, count int) {

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

		err := exec.RegisterStep("^I have (\\d+) apples$", func(ctx *cacik.Context, count int) {
			capturedCount = count

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

		err := exec.RegisterStep("^my name is (.+)$", func(ctx *cacik.Context, name string) {
			capturedName = name

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

		err := exec.RegisterStep("^the price is ([\\d.]+)$", func(ctx *cacik.Context, price float64) {
			capturedPrice = price

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

		err := exec.RegisterStep("^I have (\\d+) (\\w+)$", func(ctx *cacik.Context, count int, item string) {
			capturedCount = count
			capturedItem = item

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
		err := exec.RegisterStep("^I set value to (\\d+)$", func(ctx *cacik.Context, val int) {
			ctx.Data().Set("value", val)

		})
		require.NoError(t, err)

		var capturedVal int
		err = exec.RegisterStep("^I read the value$", func(ctx *cacik.Context) {
			capturedVal = ctx.Data().MustGet("value").(int)

		})
		require.NoError(t, err)

		doc := createDocumentWithSteps("I set value to 42", "I read the value")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 42, capturedVal)
	})

	t.Run("returns error when step function assertion fails", func(t *testing.T) {
		exec := NewStepExecutor()

		err := exec.RegisterStep("^failing step$", func(ctx *cacik.Context) {
			ctx.Assert().Fail("step failed")
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

		err := exec.RegisterStep("^I have (\\w+) apples$", func(ctx *cacik.Context, count int) {

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
	// parseBool accepts (case-insensitive): true/false, yes/no, on/off, enabled/disabled, 1/0, t/f
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

		// Short form
		{"t", "it is t", true},
		{"f", "it is f", false},
		{"T (uppercase)", "it is T", true},
		{"F (uppercase)", "it is F", false},

		// Numeric
		{"1", "it is 1", true},
		{"0", "it is 0", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			exec := NewStepExecutor()
			var capturedValue bool

			err := exec.RegisterStep("^it is (.+)$", func(ctx *cacik.Context, value bool) {
				capturedValue = value

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

		err := exec.RegisterStep("^it is (.+)$", func(ctx *cacik.Context, value bool) {

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

		err := exec.RegisterStep("^the feature is (enabled|disabled)$", func(ctx *cacik.Context, enabled bool) {
			featureEnabled = enabled

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

		err := exec.RegisterStep("^the feature is (enabled|disabled)$", func(ctx *cacik.Context, enabled bool) {
			featureEnabled = enabled

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

		err := exec.RegisterStep("^I select (red|blue|green)$", func(ctx *cacik.Context, c Color) {
			capturedColor = c

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

		err := exec.RegisterStep("^I select (red|blue|RED|BLUE)$", func(ctx *cacik.Context, c Color) {
			capturedColor = c

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
		err := exec.RegisterStep("^I select ((?i:red|blue))$", func(ctx *cacik.Context, c Color) {
			capturedColor = c

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

		err := exec.RegisterStep("^I select (\\w+)$", func(ctx *cacik.Context, c Color) {

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

		err := exec.RegisterStep("^priority is (low|medium|high|1|2|3)$", func(ctx *cacik.Context, p Priority) {
			capturedPriority = p

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

		err := exec.RegisterStep("^priority is (low|medium|high|1|2|3)$", func(ctx *cacik.Context, p Priority) {
			capturedPriority = p

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

		err := exec.RegisterStep("^I select (\\w+)$", func(ctx *cacik.Context, c Color) {
			capturedColor = c

		})
		require.NoError(t, err)

		doc := createDocument("I select anything")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, Color("anything"), capturedColor)
	})
}

func TestStepExecutor_MixedTypes(t *testing.T) {
	t.Run("handles custom type + word + int + float combination", func(t *testing.T) {
		exec := NewStepExecutor()

		// Register custom type with case-insensitive values
		exec.RegisterCustomType("Color", "string", map[string]string{
			"red":   "red",
			"blue":  "blue",
			"green": "green",
		})

		var (
			capturedColor   Color
			capturedVehicle string
			capturedDoors   int
			capturedPrice   float64
		)

		// Pattern combines: custom type {color}, normal regex (car|bike), {int}, {float}
		// This simulates: ^I want a {color} (car|bike) with {int} doors costing {float} dollars$
		err := exec.RegisterStep(
			`^I want a ((?i:blue|green|red)) (car|bike) with (-?\d+) doors costing (-?\d*\.?\d+) dollars$`,
			func(ctx *cacik.Context, color Color, vehicle string, doors int, price float64) {
				capturedColor = color
				capturedVehicle = vehicle
				capturedDoors = doors
				capturedPrice = price

			},
		)
		require.NoError(t, err)

		// Test with lowercase color
		doc := createDocument("I want a red car with 4 doors costing 25000.50 dollars")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, Color("red"), capturedColor)
		require.Equal(t, "car", capturedVehicle)
		require.Equal(t, 4, capturedDoors)
		require.Equal(t, 25000.50, capturedPrice)

		// Test with uppercase color
		doc = createDocument("I want a BLUE bike with 0 doors costing 999.99 dollars")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, Color("blue"), capturedColor)
		require.Equal(t, "bike", capturedVehicle)
		require.Equal(t, 0, capturedDoors)
		require.Equal(t, 999.99, capturedPrice)

		// Test with mixed case color
		doc = createDocument("I want a GrEeN car with 2 doors costing 15000 dollars")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, Color("green"), capturedColor)
		require.Equal(t, "car", capturedVehicle)
		require.Equal(t, 2, capturedDoors)
		require.Equal(t, 15000.0, capturedPrice)
	})

	t.Run("handles custom type + string + priority combination", func(t *testing.T) {
		exec := NewStepExecutor()

		// Register Color custom type
		exec.RegisterCustomType("Color", "string", map[string]string{
			"red":  "red",
			"blue": "blue",
		})

		// Register Priority custom type
		exec.RegisterCustomType("Priority", "int", map[string]string{
			"low":    "1",
			"medium": "2",
			"high":   "3",
			"1":      "1",
			"2":      "2",
			"3":      "3",
		})

		var (
			capturedColor    Color
			capturedName     string
			capturedPriority Priority
		)

		// Pattern: {color} item named {string} at {priority} priority
		err := exec.RegisterStep(
			`^a ((?i:blue|red)) item named "([^"]*)" at ((?i:1|2|3|high|low|medium)) priority$`,
			func(ctx *cacik.Context, color Color, name string, priority Priority) {
				capturedColor = color
				capturedName = name
				capturedPriority = priority

			},
		)
		require.NoError(t, err)

		// Test with color name and priority name
		doc := createDocument(`a RED item named "Widget" at high priority`)
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, Color("red"), capturedColor)
		require.Equal(t, "Widget", capturedName)
		require.Equal(t, Priority(3), capturedPriority)

		// Test with color name and priority value
		doc = createDocument(`a blue item named "Gadget Pro" at 1 priority`)
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, Color("blue"), capturedColor)
		require.Equal(t, "Gadget Pro", capturedName)
		require.Equal(t, Priority(1), capturedPriority)

		// Test with mixed case
		doc = createDocument(`a Blue item named "Test Item" at MEDIUM priority`)
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, Color("blue"), capturedColor)
		require.Equal(t, "Test Item", capturedName)
		require.Equal(t, Priority(2), capturedPriority)
	})

	t.Run("handles word type with custom type and bool", func(t *testing.T) {
		exec := NewStepExecutor()

		exec.RegisterCustomType("Color", "string", map[string]string{
			"red":  "red",
			"blue": "blue",
		})

		var (
			capturedColor   Color
			capturedOwner   string
			capturedVisible bool
		)

		// Pattern: {color} owned by {word} is visible {bool}
		err := exec.RegisterStep(
			`^((?i:blue|red)) owned by (\w+) is (yes|no)$`,
			func(ctx *cacik.Context, color Color, owner string, visible bool) {
				capturedColor = color
				capturedOwner = owner
				capturedVisible = visible

			},
		)
		require.NoError(t, err)

		doc := createDocument("RED owned by Alice is yes")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, Color("red"), capturedColor)
		require.Equal(t, "Alice", capturedOwner)
		require.True(t, capturedVisible)

		doc = createDocument("blue owned by Bob is no")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, Color("blue"), capturedColor)
		require.Equal(t, "Bob", capturedOwner)
		require.False(t, capturedVisible)
	})
}

func TestStepExecutor_TimeType_TimeTime(t *testing.T) {
	// Time pattern from builtInTypes (with optional timezone)
	timePattern := `(\d{1,2}:\d{2}(?::\d{2})?(?:\.\d{1,3})?(?:\s*[AaPp][Mm])?(?:\s*(?:Z|UTC|[+-]\d{2}:?\d{2}|[A-Za-z_]+/[A-Za-z_]+))?)`

	t.Run("parses time to time.Time with zero date", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedTime time.Time

		err := exec.RegisterStep("^meeting at "+timePattern+"$", func(ctx *cacik.Context, t time.Time) {
			capturedTime = t

		})
		require.NoError(t, err)

		doc := createDocument("meeting at 14:30")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 14, capturedTime.Hour())
		require.Equal(t, 30, capturedTime.Minute())
		require.Equal(t, 1, capturedTime.Year()) // Zero date: year 1
	})

	t.Run("parses time with seconds", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedTime time.Time

		err := exec.RegisterStep("^meeting at "+timePattern+"$", func(ctx *cacik.Context, t time.Time) {
			capturedTime = t

		})
		require.NoError(t, err)

		doc := createDocument("meeting at 14:30:45")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 14, capturedTime.Hour())
		require.Equal(t, 30, capturedTime.Minute())
		require.Equal(t, 45, capturedTime.Second())
	})

	t.Run("parses time with AM/PM", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedTime time.Time

		err := exec.RegisterStep("^meeting at "+timePattern+"$", func(ctx *cacik.Context, t time.Time) {
			capturedTime = t

		})
		require.NoError(t, err)

		doc := createDocument("meeting at 2:30pm")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 14, capturedTime.Hour()) // 2pm = 14:00
		require.Equal(t, 30, capturedTime.Minute())
	})

	t.Run("parses time with timezone Z", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedTime time.Time

		err := exec.RegisterStep("^meeting at "+timePattern+"$", func(ctx *cacik.Context, t time.Time) {
			capturedTime = t

		})
		require.NoError(t, err)

		doc := createDocument("meeting at 14:30Z")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 14, capturedTime.Hour())
		require.Equal(t, 30, capturedTime.Minute())
		require.Equal(t, "UTC", capturedTime.Location().String())
	})

	t.Run("parses time with timezone offset", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedTime time.Time

		err := exec.RegisterStep("^meeting at "+timePattern+"$", func(ctx *cacik.Context, t time.Time) {
			capturedTime = t

		})
		require.NoError(t, err)

		doc := createDocument("meeting at 14:30+05:30")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 14, capturedTime.Hour())
		require.Equal(t, 30, capturedTime.Minute())
		_, offset := capturedTime.Zone()
		require.Equal(t, 5*3600+30*60, offset) // +05:30 in seconds
	})

	t.Run("parses time with IANA timezone", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedTime time.Time

		err := exec.RegisterStep("^meeting at "+timePattern+"$", func(ctx *cacik.Context, t time.Time) {
			capturedTime = t

		})
		require.NoError(t, err)

		doc := createDocument("meeting at 14:30 Europe/London")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 14, capturedTime.Hour())
		require.Equal(t, 30, capturedTime.Minute())
		require.Equal(t, "Europe/London", capturedTime.Location().String())
	})
}

func TestStepExecutor_DateType_TimeTime(t *testing.T) {
	// Date pattern from builtInTypes
	datePattern := `(\d{4}[-/]\d{2}[-/]\d{2}|\d{1,2}[-/\.]\d{1,2}[-/\.]\d{2,4}|(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\.?\s+\d{1,2},?\s+\d{4}|\d{1,2}\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\.?\s+\d{4})`

	t.Run("parses ISO date to time.Time at midnight", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedDate time.Time

		err := exec.RegisterStep("^event on "+datePattern+"$", func(ctx *cacik.Context, d time.Time) {
			capturedDate = d

		})
		require.NoError(t, err)

		doc := createDocument("event on 2024-01-15")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 2024, capturedDate.Year())
		require.Equal(t, time.January, capturedDate.Month())
		require.Equal(t, 15, capturedDate.Day())
		require.Equal(t, 0, capturedDate.Hour())   // Midnight
		require.Equal(t, 0, capturedDate.Minute()) // Midnight
	})

	t.Run("parses EU date format (DD/MM/YYYY)", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedDate time.Time

		err := exec.RegisterStep("^event on "+datePattern+"$", func(ctx *cacik.Context, d time.Time) {
			capturedDate = d

		})
		require.NoError(t, err)

		doc := createDocument("event on 15/01/2024")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 2024, capturedDate.Year())
		require.Equal(t, time.January, capturedDate.Month()) // EU: 15/01 = Jan 15
		require.Equal(t, 15, capturedDate.Day())
	})

	t.Run("parses EU date format with dots (DD.MM.YYYY)", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedDate time.Time

		err := exec.RegisterStep("^event on "+datePattern+"$", func(ctx *cacik.Context, d time.Time) {
			capturedDate = d

		})
		require.NoError(t, err)

		doc := createDocument("event on 15.01.2024")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 2024, capturedDate.Year())
		require.Equal(t, time.January, capturedDate.Month())
		require.Equal(t, 15, capturedDate.Day())
	})

	t.Run("parses written date format (15 Jan 2024)", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedDate time.Time

		err := exec.RegisterStep("^event on "+datePattern+"$", func(ctx *cacik.Context, d time.Time) {
			capturedDate = d

		})
		require.NoError(t, err)

		doc := createDocument("event on 15 Jan 2024")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 2024, capturedDate.Year())
		require.Equal(t, time.January, capturedDate.Month())
		require.Equal(t, 15, capturedDate.Day())
	})

	t.Run("parses written date format (Jan 15, 2024)", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedDate time.Time

		err := exec.RegisterStep("^event on "+datePattern+"$", func(ctx *cacik.Context, d time.Time) {
			capturedDate = d

		})
		require.NoError(t, err)

		doc := createDocument("event on Jan 15, 2024")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 2024, capturedDate.Year())
		require.Equal(t, time.January, capturedDate.Month())
		require.Equal(t, 15, capturedDate.Day())
	})
}

func TestStepExecutor_DateTimeType_TimeTime(t *testing.T) {
	// DateTime pattern from builtInTypes (with optional timezone)
	datetimePattern := `(\d{4}[-/]\d{2}[-/]\d{2}[T\s]\d{1,2}:\d{2}(?::\d{2})?(?:\.\d{1,3})?(?:\s*[AaPp][Mm])?(?:\s*(?:Z|UTC|[+-]\d{2}:?\d{2}|[A-Za-z_]+/[A-Za-z_]+))?|\d{1,2}[-/\.]\d{1,2}[-/\.]\d{2,4}\s+\d{1,2}:\d{2}(?::\d{2})?(?:\s*[AaPp][Mm])?(?:\s*(?:Z|UTC|[+-]\d{2}:?\d{2}|[A-Za-z_]+/[A-Za-z_]+))?)`

	t.Run("parses ISO datetime with space", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedDT time.Time

		err := exec.RegisterStep("^appointment at "+datetimePattern+"$", func(ctx *cacik.Context, dt time.Time) {
			capturedDT = dt

		})
		require.NoError(t, err)

		doc := createDocument("appointment at 2024-01-15 14:30")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 2024, capturedDT.Year())
		require.Equal(t, time.January, capturedDT.Month())
		require.Equal(t, 15, capturedDT.Day())
		require.Equal(t, 14, capturedDT.Hour())
		require.Equal(t, 30, capturedDT.Minute())
	})

	t.Run("parses ISO datetime with T separator", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedDT time.Time

		err := exec.RegisterStep("^appointment at "+datetimePattern+"$", func(ctx *cacik.Context, dt time.Time) {
			capturedDT = dt

		})
		require.NoError(t, err)

		doc := createDocument("appointment at 2024-01-15T14:30:45")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 2024, capturedDT.Year())
		require.Equal(t, 15, capturedDT.Day())
		require.Equal(t, 14, capturedDT.Hour())
		require.Equal(t, 30, capturedDT.Minute())
		require.Equal(t, 45, capturedDT.Second())
	})

	t.Run("parses datetime with Z timezone", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedDT time.Time

		err := exec.RegisterStep("^appointment at "+datetimePattern+"$", func(ctx *cacik.Context, dt time.Time) {
			capturedDT = dt

		})
		require.NoError(t, err)

		doc := createDocument("appointment at 2024-01-15T14:30:00Z")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 2024, capturedDT.Year())
		require.Equal(t, 14, capturedDT.Hour())
		require.Equal(t, "UTC", capturedDT.Location().String())
	})

	t.Run("parses datetime with offset timezone", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedDT time.Time

		err := exec.RegisterStep("^appointment at "+datetimePattern+"$", func(ctx *cacik.Context, dt time.Time) {
			capturedDT = dt

		})
		require.NoError(t, err)

		doc := createDocument("appointment at 2024-01-15T14:30:00+05:30")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 2024, capturedDT.Year())
		require.Equal(t, 14, capturedDT.Hour())
		_, offset := capturedDT.Zone()
		require.Equal(t, 5*3600+30*60, offset)
	})

	t.Run("parses datetime with IANA timezone", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedDT time.Time

		err := exec.RegisterStep("^appointment at "+datetimePattern+"$", func(ctx *cacik.Context, dt time.Time) {
			capturedDT = dt

		})
		require.NoError(t, err)

		doc := createDocument("appointment at 2024-01-15 14:30 Europe/London")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 2024, capturedDT.Year())
		require.Equal(t, 14, capturedDT.Hour())
		require.Equal(t, "Europe/London", capturedDT.Location().String())
	})

	t.Run("parses EU datetime format", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedDT time.Time

		err := exec.RegisterStep("^appointment at "+datetimePattern+"$", func(ctx *cacik.Context, dt time.Time) {
			capturedDT = dt

		})
		require.NoError(t, err)

		doc := createDocument("appointment at 15/01/2024 14:30")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 2024, capturedDT.Year())
		require.Equal(t, time.January, capturedDT.Month()) // EU format
		require.Equal(t, 15, capturedDT.Day())
		require.Equal(t, 14, capturedDT.Hour())
	})

	t.Run("parses datetime with AM/PM", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedDT time.Time

		err := exec.RegisterStep("^appointment at "+datetimePattern+"$", func(ctx *cacik.Context, dt time.Time) {
			capturedDT = dt

		})
		require.NoError(t, err)

		doc := createDocument("appointment at 2024-01-15 2:30pm")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 14, capturedDT.Hour()) // 2pm = 14:00
		require.Equal(t, 30, capturedDT.Minute())
	})
}

func TestStepExecutor_TimezoneType(t *testing.T) {
	// Timezone pattern from builtInTypes
	tzPattern := `(Z|UTC|[+-]\d{2}:?\d{2}|[A-Za-z_]+/[A-Za-z_]+)`

	t.Run("parses Z as UTC", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedLoc *time.Location

		err := exec.RegisterStep("^convert to "+tzPattern+"$", func(ctx *cacik.Context, loc *time.Location) {
			capturedLoc = loc

		})
		require.NoError(t, err)

		doc := createDocument("convert to Z")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, "UTC", capturedLoc.String())
	})

	t.Run("parses UTC", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedLoc *time.Location

		err := exec.RegisterStep("^convert to "+tzPattern+"$", func(ctx *cacik.Context, loc *time.Location) {
			capturedLoc = loc

		})
		require.NoError(t, err)

		doc := createDocument("convert to UTC")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, "UTC", capturedLoc.String())
	})

	t.Run("parses offset +05:30", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedLoc *time.Location

		err := exec.RegisterStep("^convert to "+tzPattern+"$", func(ctx *cacik.Context, loc *time.Location) {
			capturedLoc = loc

		})
		require.NoError(t, err)

		doc := createDocument("convert to +05:30")
		err = exec.Execute(doc)
		require.NoError(t, err)
		now := time.Now().In(capturedLoc)
		_, offset := now.Zone()
		require.Equal(t, 5*3600+30*60, offset)
	})

	t.Run("parses offset -08:00", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedLoc *time.Location

		err := exec.RegisterStep("^convert to "+tzPattern+"$", func(ctx *cacik.Context, loc *time.Location) {
			capturedLoc = loc

		})
		require.NoError(t, err)

		doc := createDocument("convert to -08:00")
		err = exec.Execute(doc)
		require.NoError(t, err)
		now := time.Now().In(capturedLoc)
		_, offset := now.Zone()
		require.Equal(t, -8*3600, offset)
	})

	t.Run("parses offset without colon +0530", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedLoc *time.Location

		err := exec.RegisterStep("^convert to "+tzPattern+"$", func(ctx *cacik.Context, loc *time.Location) {
			capturedLoc = loc

		})
		require.NoError(t, err)

		doc := createDocument("convert to +0530")
		err = exec.Execute(doc)
		require.NoError(t, err)
		now := time.Now().In(capturedLoc)
		_, offset := now.Zone()
		require.Equal(t, 5*3600+30*60, offset)
	})

	t.Run("parses IANA timezone Europe/London", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedLoc *time.Location

		err := exec.RegisterStep("^convert to "+tzPattern+"$", func(ctx *cacik.Context, loc *time.Location) {
			capturedLoc = loc

		})
		require.NoError(t, err)

		doc := createDocument("convert to Europe/London")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, "Europe/London", capturedLoc.String())
	})

	t.Run("parses IANA timezone America/New_York", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedLoc *time.Location

		err := exec.RegisterStep("^convert to "+tzPattern+"$", func(ctx *cacik.Context, loc *time.Location) {
			capturedLoc = loc

		})
		require.NoError(t, err)

		doc := createDocument("convert to America/New_York")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, "America/New_York", capturedLoc.String())
	})

	t.Run("parses IANA timezone Asia/Tokyo", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedLoc *time.Location

		err := exec.RegisterStep("^convert to "+tzPattern+"$", func(ctx *cacik.Context, loc *time.Location) {
			capturedLoc = loc

		})
		require.NoError(t, err)

		doc := createDocument("convert to Asia/Tokyo")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, "Asia/Tokyo", capturedLoc.String())
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

func TestStepExecutor_DurationType(t *testing.T) {
	// Duration pattern from builtInTypes
	durationPattern := `(-?(?:\d+\.?\d*(?:ns|us|µs|ms|s|m|h))+)`

	t.Run("parses simple duration in seconds", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedDuration time.Duration

		err := exec.RegisterStep("^wait for "+durationPattern+"$", func(ctx *cacik.Context, d time.Duration) {
			capturedDuration = d

		})
		require.NoError(t, err)

		doc := createDocument("wait for 5s")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 5*time.Second, capturedDuration)
	})

	t.Run("parses compound duration", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedDuration time.Duration

		err := exec.RegisterStep("^wait for "+durationPattern+"$", func(ctx *cacik.Context, d time.Duration) {
			capturedDuration = d

		})
		require.NoError(t, err)

		doc := createDocument("wait for 1h30m")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 1*time.Hour+30*time.Minute, capturedDuration)
	})

	t.Run("parses milliseconds", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedDuration time.Duration

		err := exec.RegisterStep("^wait for "+durationPattern+"$", func(ctx *cacik.Context, d time.Duration) {
			capturedDuration = d

		})
		require.NoError(t, err)

		doc := createDocument("wait for 500ms")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 500*time.Millisecond, capturedDuration)
	})

	t.Run("parses negative duration", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedDuration time.Duration

		err := exec.RegisterStep("^wait for "+durationPattern+"$", func(ctx *cacik.Context, d time.Duration) {
			capturedDuration = d

		})
		require.NoError(t, err)

		doc := createDocument("wait for -30m")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, -30*time.Minute, capturedDuration)
	})

	t.Run("parses complex duration", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedDuration time.Duration

		err := exec.RegisterStep("^wait for "+durationPattern+"$", func(ctx *cacik.Context, d time.Duration) {
			capturedDuration = d

		})
		require.NoError(t, err)

		doc := createDocument("wait for 2h45m30s")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 2*time.Hour+45*time.Minute+30*time.Second, capturedDuration)
	})

	t.Run("parses nanoseconds", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedDuration time.Duration

		err := exec.RegisterStep("^wait for "+durationPattern+"$", func(ctx *cacik.Context, d time.Duration) {
			capturedDuration = d

		})
		require.NoError(t, err)

		doc := createDocument("wait for 100ns")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 100*time.Nanosecond, capturedDuration)
	})
}

func TestStepExecutor_URLType(t *testing.T) {
	// URL pattern from builtInTypes
	urlPattern := `(https?://[^\s]+)`

	t.Run("parses simple HTTP URL", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedURL *url.URL

		err := exec.RegisterStep("^navigate to "+urlPattern+"$", func(ctx *cacik.Context, u *url.URL) {
			capturedURL = u

		})
		require.NoError(t, err)

		doc := createDocument("navigate to http://example.com")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, "http", capturedURL.Scheme)
		require.Equal(t, "example.com", capturedURL.Host)
	})

	t.Run("parses HTTPS URL with path", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedURL *url.URL

		err := exec.RegisterStep("^navigate to "+urlPattern+"$", func(ctx *cacik.Context, u *url.URL) {
			capturedURL = u

		})
		require.NoError(t, err)

		doc := createDocument("navigate to https://api.example.com/users")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, "https", capturedURL.Scheme)
		require.Equal(t, "api.example.com", capturedURL.Host)
		require.Equal(t, "/users", capturedURL.Path)
	})

	t.Run("parses URL with query string", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedURL *url.URL

		err := exec.RegisterStep("^navigate to "+urlPattern+"$", func(ctx *cacik.Context, u *url.URL) {
			capturedURL = u

		})
		require.NoError(t, err)

		doc := createDocument("navigate to https://example.com/search?q=test&page=1")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, "https", capturedURL.Scheme)
		require.Equal(t, "/search", capturedURL.Path)
		require.Equal(t, "q=test&page=1", capturedURL.RawQuery)
	})

	t.Run("parses URL with port", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedURL *url.URL

		err := exec.RegisterStep("^navigate to "+urlPattern+"$", func(ctx *cacik.Context, u *url.URL) {
			capturedURL = u

		})
		require.NoError(t, err)

		doc := createDocument("navigate to http://localhost:8080/api")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, "localhost:8080", capturedURL.Host)
		require.Equal(t, "/api", capturedURL.Path)
	})

	t.Run("parses URL with fragment", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedURL *url.URL

		err := exec.RegisterStep("^navigate to "+urlPattern+"$", func(ctx *cacik.Context, u *url.URL) {
			capturedURL = u

		})
		require.NoError(t, err)

		doc := createDocument("navigate to https://example.com/page#section")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, "/page", capturedURL.Path)
		require.Equal(t, "section", capturedURL.Fragment)
	})
}

func TestStepExecutor_EmailType(t *testing.T) {
	// Email pattern from builtInTypes
	emailPattern := `([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`

	t.Run("parses simple email", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedEmail string

		err := exec.RegisterStep("^user "+emailPattern+" logged in$", func(ctx *cacik.Context, email string) {
			capturedEmail = email

		})
		require.NoError(t, err)

		doc := createDocument("user john@example.com logged in")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, "john@example.com", capturedEmail)
	})

	t.Run("parses email with subdomain", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedEmail string

		err := exec.RegisterStep("^user "+emailPattern+" logged in$", func(ctx *cacik.Context, email string) {
			capturedEmail = email

		})
		require.NoError(t, err)

		doc := createDocument("user admin@mail.company.org logged in")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, "admin@mail.company.org", capturedEmail)
	})

	t.Run("parses email with plus tag", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedEmail string

		err := exec.RegisterStep("^user "+emailPattern+" logged in$", func(ctx *cacik.Context, email string) {
			capturedEmail = email

		})
		require.NoError(t, err)

		doc := createDocument("user john.doe+newsletter@example.com logged in")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, "john.doe+newsletter@example.com", capturedEmail)
	})

	t.Run("parses email with dots in local part", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedEmail string

		err := exec.RegisterStep("^user "+emailPattern+" logged in$", func(ctx *cacik.Context, email string) {
			capturedEmail = email

		})
		require.NoError(t, err)

		doc := createDocument("user first.middle.last@domain.co.uk logged in")
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, "first.middle.last@domain.co.uk", capturedEmail)
	})
}

// =============================================================================
// Rule and Background Tests
// =============================================================================

func TestStepExecutor_Execute_Rule(t *testing.T) {
	t.Run("executes scenario inside rule", func(t *testing.T) {
		exec := NewStepExecutor()
		executed := false

		err := exec.RegisterStep("^rule scenario step$", func(ctx *cacik.Context) {
			executed = true

		})
		require.NoError(t, err)

		doc := createDocumentWithRule([]string{"rule scenario step"})
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.True(t, executed)
	})

	t.Run("executes multiple scenarios inside rule", func(t *testing.T) {
		exec := NewStepExecutor()
		executionOrder := []string{}

		err := exec.RegisterStep("^scenario (\\d+) step$", func(ctx *cacik.Context, num int) {
			executionOrder = append(executionOrder, fmt.Sprintf("scenario-%d", num))

		})
		require.NoError(t, err)

		doc := createDocumentWithRule(
			[]string{"scenario 1 step"},
			[]string{"scenario 2 step"},
			[]string{"scenario 3 step"},
		)
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, []string{"scenario-1", "scenario-2", "scenario-3"}, executionOrder)
	})

	t.Run("executes rule background before each scenario", func(t *testing.T) {
		exec := NewStepExecutor()
		executionOrder := []string{}

		err := exec.RegisterStep("^rule background step$", func(ctx *cacik.Context) {
			executionOrder = append(executionOrder, "rule-bg")

		})
		require.NoError(t, err)

		err = exec.RegisterStep("^scenario (\\d+) step$", func(ctx *cacik.Context, num int) {
			executionOrder = append(executionOrder, fmt.Sprintf("scenario-%d", num))

		})
		require.NoError(t, err)

		doc := createDocumentWithRuleBackground(
			[]string{"rule background step"},
			[]string{"scenario 1 step"},
			[]string{"scenario 2 step"},
		)
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, []string{"rule-bg", "scenario-1", "rule-bg", "scenario-2"}, executionOrder)
	})

	t.Run("executes feature background before rule scenarios", func(t *testing.T) {
		exec := NewStepExecutor()
		executionOrder := []string{}

		err := exec.RegisterStep("^feature background step$", func(ctx *cacik.Context) {
			executionOrder = append(executionOrder, "feature-bg")

		})
		require.NoError(t, err)

		err = exec.RegisterStep("^rule scenario step$", func(ctx *cacik.Context) {
			executionOrder = append(executionOrder, "rule-scenario")

		})
		require.NoError(t, err)

		doc := createDocumentWithFeatureBackgroundAndRule(
			[]string{"feature background step"},
			[]string{"rule scenario step"},
		)
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, []string{"feature-bg", "rule-scenario"}, executionOrder)
	})

	t.Run("executes feature background then rule background", func(t *testing.T) {
		exec := NewStepExecutor()
		executionOrder := []string{}

		err := exec.RegisterStep("^feature background step$", func(ctx *cacik.Context) {
			executionOrder = append(executionOrder, "feature-bg")

		})
		require.NoError(t, err)

		err = exec.RegisterStep("^rule background step$", func(ctx *cacik.Context) {
			executionOrder = append(executionOrder, "rule-bg")

		})
		require.NoError(t, err)

		err = exec.RegisterStep("^scenario step$", func(ctx *cacik.Context) {
			executionOrder = append(executionOrder, "scenario")

		})
		require.NoError(t, err)

		doc := createDocumentWithFeatureAndRuleBackground(
			[]string{"feature background step"},
			[]string{"rule background step"},
			[]string{"scenario step"},
		)
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, []string{"feature-bg", "rule-bg", "scenario"}, executionOrder)
	})

	t.Run("executes multiple rules", func(t *testing.T) {
		exec := NewStepExecutor()
		executionOrder := []string{}

		err := exec.RegisterStep("^rule (\\d+) scenario step$", func(ctx *cacik.Context, num int) {
			executionOrder = append(executionOrder, fmt.Sprintf("rule-%d", num))

		})
		require.NoError(t, err)

		doc := createDocumentWithMultipleRules(
			[][]string{{"rule 1 scenario step"}},
			[][]string{{"rule 2 scenario step"}},
		)
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, []string{"rule-1", "rule-2"}, executionOrder)
	})

	t.Run("feature and rule backgrounds run before each scenario in rule", func(t *testing.T) {
		exec := NewStepExecutor()
		executionOrder := []string{}

		err := exec.RegisterStep("^feature bg$", func(ctx *cacik.Context) {
			executionOrder = append(executionOrder, "feature-bg")

		})
		require.NoError(t, err)

		err = exec.RegisterStep("^rule bg$", func(ctx *cacik.Context) {
			executionOrder = append(executionOrder, "rule-bg")

		})
		require.NoError(t, err)

		err = exec.RegisterStep("^scenario (\\d+)$", func(ctx *cacik.Context, num int) {
			executionOrder = append(executionOrder, fmt.Sprintf("scenario-%d", num))

		})
		require.NoError(t, err)

		doc := createDocumentWithFeatureAndRuleBackgroundMultipleScenarios(
			[]string{"feature bg"},
			[]string{"rule bg"},
			[][]string{{"scenario 1"}, {"scenario 2"}},
		)
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, []string{
			"feature-bg", "rule-bg", "scenario-1",
			"feature-bg", "rule-bg", "scenario-2",
		}, executionOrder)
	})
}

func TestStepExecutor_Execute_Background_Extended(t *testing.T) {
	t.Run("executes background before each of multiple scenarios", func(t *testing.T) {
		exec := NewStepExecutor()
		executionOrder := []string{}

		err := exec.RegisterStep("^background step$", func(ctx *cacik.Context) {
			executionOrder = append(executionOrder, "bg")

		})
		require.NoError(t, err)

		err = exec.RegisterStep("^scenario (\\d+) step$", func(ctx *cacik.Context, num int) {
			executionOrder = append(executionOrder, fmt.Sprintf("scenario-%d", num))

		})
		require.NoError(t, err)

		doc := createDocumentWithBackgroundAndMultipleScenarios(
			[]string{"background step"},
			[][]string{{"scenario 1 step"}, {"scenario 2 step"}, {"scenario 3 step"}},
		)
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, []string{
			"bg", "scenario-1",
			"bg", "scenario-2",
			"bg", "scenario-3",
		}, executionOrder)
	})

	t.Run("background step failure stops execution", func(t *testing.T) {
		exec := NewStepExecutor()
		scenarioExecuted := false

		err := exec.RegisterStep("^failing background step$", func(ctx *cacik.Context) {
			ctx.Assert().Fail("background failed")
		})
		require.NoError(t, err)

		err = exec.RegisterStep("^scenario step$", func(ctx *cacik.Context) {
			scenarioExecuted = true

		})
		require.NoError(t, err)

		doc := createDocumentWithBackground("failing background step", "scenario step")
		err = exec.Execute(doc)
		require.Error(t, err)
		require.Contains(t, err.Error(), "background failed")
		require.False(t, scenarioExecuted)
	})

	t.Run("background with parameters", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedCount int
		var capturedName string

		err := exec.RegisterStep("^I have (\\d+) items$", func(ctx *cacik.Context, count int) {
			capturedCount = count

		})
		require.NoError(t, err)

		err = exec.RegisterStep("^my name is (\\w+)$", func(ctx *cacik.Context, name string) {
			capturedName = name

		})
		require.NoError(t, err)

		doc := createDocumentWithBackgroundSteps(
			[]string{"I have 42 items"},
			[]string{"my name is John"},
		)
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 42, capturedCount)
		require.Equal(t, "John", capturedName)
	})

	t.Run("multiple background steps execute in order", func(t *testing.T) {
		exec := NewStepExecutor()
		executionOrder := []string{}

		err := exec.RegisterStep("^background step (\\d+)$", func(ctx *cacik.Context, num int) {
			executionOrder = append(executionOrder, fmt.Sprintf("bg-%d", num))

		})
		require.NoError(t, err)

		err = exec.RegisterStep("^scenario step$", func(ctx *cacik.Context) {
			executionOrder = append(executionOrder, "scenario")

		})
		require.NoError(t, err)

		doc := createDocumentWithBackgroundSteps(
			[]string{"background step 1", "background step 2", "background step 3"},
			[]string{"scenario step"},
		)
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, []string{"bg-1", "bg-2", "bg-3", "scenario"}, executionOrder)
	})
}

func TestStepExecutor_Execute_ComplexFeatureStructure(t *testing.T) {
	t.Run("feature with background, standalone scenarios, and rules", func(t *testing.T) {
		exec := NewStepExecutor()
		executionOrder := []string{}

		err := exec.RegisterStep("^feature bg$", func(ctx *cacik.Context) {
			executionOrder = append(executionOrder, "feature-bg")

		})
		require.NoError(t, err)

		err = exec.RegisterStep("^standalone scenario$", func(ctx *cacik.Context) {
			executionOrder = append(executionOrder, "standalone")

		})
		require.NoError(t, err)

		err = exec.RegisterStep("^rule scenario$", func(ctx *cacik.Context) {
			executionOrder = append(executionOrder, "rule-scenario")

		})
		require.NoError(t, err)

		doc := createComplexDocument(
			[]string{"feature bg"},          // feature background
			[]string{"standalone scenario"}, // standalone scenario
			[]string{"rule scenario"},       // rule scenario
		)
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, []string{
			"feature-bg", "standalone",
			"feature-bg", "rule-scenario",
		}, executionOrder)
	})

	t.Run("context is preserved across background and scenario steps", func(t *testing.T) {
		exec := NewStepExecutor()

		err := exec.RegisterStep("^I set value to (\\d+)$", func(ctx *cacik.Context, val int) {
			ctx.Data().Set("value", val)

		})
		require.NoError(t, err)

		var capturedValue int
		err = exec.RegisterStep("^the value should be (\\d+)$", func(ctx *cacik.Context, expected int) {
			capturedValue = ctx.Data().MustGet("value").(int)

		})
		require.NoError(t, err)

		doc := createDocumentWithBackgroundSteps(
			[]string{"I set value to 100"},
			[]string{"the value should be 100"},
		)
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, 100, capturedValue)
	})
}

// =============================================================================
// DataTable Injection Tests
// =============================================================================

func TestStepExecutor_DataTableInjection(t *testing.T) {
	t.Run("injects Table when step has DataTable", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedTable cacik.Table

		err := exec.RegisterStep("^I have the following users$", func(ctx *cacik.Context, table cacik.Table) {
			capturedTable = table
		})
		require.NoError(t, err)

		doc := createDocumentWithDataTable("I have the following users", [][]string{
			{"name", "age"},
			{"Alice", "30"},
			{"Bob", "25"},
		})
		err = exec.Execute(doc)
		require.NoError(t, err)

		require.Equal(t, 3, capturedTable.Len())
		require.Equal(t, []string{"name", "age"}, capturedTable.Headers())

		// Check data via SkipHeader + Get
		var names []string
		var ages []string
		for _, row := range capturedTable.SkipHeader() {
			names = append(names, row.Get("name"))
			ages = append(ages, row.Get("age"))
		}
		require.Equal(t, []string{"Alice", "Bob"}, names)
		require.Equal(t, []string{"30", "25"}, ages)
	})

	t.Run("injects Table with Context and regex capture args", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedCount int
		var capturedTable cacik.Table

		err := exec.RegisterStep("^I have (\\d+) items with details$", func(ctx *cacik.Context, count int, table cacik.Table) {
			capturedCount = count
			capturedTable = table
		})
		require.NoError(t, err)

		doc := createDocumentWithDataTable("I have 3 items with details", [][]string{
			{"item", "price"},
			{"apple", "1.50"},
			{"banana", "0.75"},
			{"cherry", "2.00"},
		})
		err = exec.Execute(doc)
		require.NoError(t, err)

		require.Equal(t, 3, capturedCount)
		require.Equal(t, 4, capturedTable.Len())
		require.Equal(t, []string{"item", "price"}, capturedTable.Headers())
	})

	t.Run("injects Table without Context parameter", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedTable cacik.Table

		err := exec.RegisterStep("^a table without context$", func(table cacik.Table) {
			capturedTable = table
		})
		require.NoError(t, err)

		doc := createDocumentWithDataTable("a table without context", [][]string{
			{"col1", "col2"},
			{"a", "b"},
		})
		err = exec.Execute(doc)
		require.NoError(t, err)

		require.Equal(t, 2, capturedTable.Len())
	})

	t.Run("errors when function expects Table but step has no DataTable", func(t *testing.T) {
		exec := NewStepExecutor()

		err := exec.RegisterStep("^a step expecting table$", func(ctx *cacik.Context, table cacik.Table) {
		})
		require.NoError(t, err)

		// Use createDocument which creates steps without DataTable
		doc := createDocument("a step expecting table")
		err = exec.Execute(doc)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no DataTable")
	})

	t.Run("step without Table parameter works with DataTable present", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedText string

		err := exec.RegisterStep("^a step that ignores table (.+)$", func(ctx *cacik.Context, text string) {
			capturedText = text
		})
		require.NoError(t, err)

		doc := createDocumentWithDataTable("a step that ignores table hello", [][]string{
			{"key", "value"},
			{"a", "1"},
		})
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, "hello", capturedText)
	})

	t.Run("Table All iterates all rows including header", func(t *testing.T) {
		exec := NewStepExecutor()
		var allCells []string

		err := exec.RegisterStep("^check all rows$", func(table cacik.Table) {
			for _, row := range table.All() {
				allCells = append(allCells, row.Cell(0))
			}
		})
		require.NoError(t, err)

		doc := createDocumentWithDataTable("check all rows", [][]string{
			{"header"},
			{"data1"},
			{"data2"},
		})
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, []string{"header", "data1", "data2"}, allCells)
	})

	t.Run("Table SkipHeader skips first row", func(t *testing.T) {
		exec := NewStepExecutor()
		var dataCells []string

		err := exec.RegisterStep("^check data rows$", func(table cacik.Table) {
			for _, row := range table.SkipHeader() {
				dataCells = append(dataCells, row.Cell(0))
			}
		})
		require.NoError(t, err)

		doc := createDocumentWithDataTable("check data rows", [][]string{
			{"header"},
			{"data1"},
			{"data2"},
		})
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, []string{"data1", "data2"}, dataCells)
	})

	t.Run("Row.Get is case-insensitive", func(t *testing.T) {
		exec := NewStepExecutor()
		var result string

		err := exec.RegisterStep("^check case insensitive get$", func(table cacik.Table) {
			for _, row := range table.SkipHeader() {
				result = row.Get("NaMe")
			}
		})
		require.NoError(t, err)

		doc := createDocumentWithDataTable("check case insensitive get", [][]string{
			{"name", "age"},
			{"Alice", "30"},
		})
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, "Alice", result)
	})

	t.Run("Table with Context and multiple capture groups and Table", func(t *testing.T) {
		exec := NewStepExecutor()
		var capturedA, capturedB string
		var capturedTable cacik.Table

		err := exec.RegisterStep("^user (.+) buys (.+) with receipt$", func(ctx *cacik.Context, a, b string, table cacik.Table) {
			capturedA = a
			capturedB = b
			capturedTable = table
		})
		require.NoError(t, err)

		doc := createDocumentWithDataTable("user Alice buys apples with receipt", [][]string{
			{"item", "qty"},
			{"apple", "5"},
		})
		err = exec.Execute(doc)
		require.NoError(t, err)
		require.Equal(t, "Alice", capturedA)
		require.Equal(t, "apples", capturedB)
		require.Equal(t, 2, capturedTable.Len())
	})
}

// =============================================================================
// Helper Functions for Rule and Background Documents
// =============================================================================

// createDocumentWithRule creates a document with a single rule containing multiple scenarios
func createDocumentWithRule(scenarioStepSets ...[]string) *messages.GherkinDocument {
	ruleChildren := make([]*messages.RuleChild, len(scenarioStepSets))
	for i, stepSet := range scenarioStepSets {
		steps := make([]*messages.Step, len(stepSet))
		for j, text := range stepSet {
			steps[j] = &messages.Step{Text: text}
		}
		ruleChildren[i] = &messages.RuleChild{
			Scenario: &messages.Scenario{Steps: steps},
		}
	}

	return &messages.GherkinDocument{
		Feature: &messages.Feature{
			Children: []*messages.FeatureChild{
				{
					Rule: &messages.Rule{
						Children: ruleChildren,
					},
				},
			},
		},
	}
}

// createDocumentWithRuleBackground creates a document with a rule that has a background
func createDocumentWithRuleBackground(bgSteps []string, scenarioStepSets ...[]string) *messages.GherkinDocument {
	bgStepsMsg := make([]*messages.Step, len(bgSteps))
	for i, text := range bgSteps {
		bgStepsMsg[i] = &messages.Step{Text: text}
	}

	ruleChildren := make([]*messages.RuleChild, len(scenarioStepSets)+1)
	ruleChildren[0] = &messages.RuleChild{
		Background: &messages.Background{Steps: bgStepsMsg},
	}

	for i, stepSet := range scenarioStepSets {
		steps := make([]*messages.Step, len(stepSet))
		for j, text := range stepSet {
			steps[j] = &messages.Step{Text: text}
		}
		ruleChildren[i+1] = &messages.RuleChild{
			Scenario: &messages.Scenario{Steps: steps},
		}
	}

	return &messages.GherkinDocument{
		Feature: &messages.Feature{
			Children: []*messages.FeatureChild{
				{
					Rule: &messages.Rule{
						Children: ruleChildren,
					},
				},
			},
		},
	}
}

// createDocumentWithFeatureBackgroundAndRule creates a document with feature-level background and a rule
func createDocumentWithFeatureBackgroundAndRule(featureBgSteps, ruleScenarioSteps []string) *messages.GherkinDocument {
	featureBgStepsMsg := make([]*messages.Step, len(featureBgSteps))
	for i, text := range featureBgSteps {
		featureBgStepsMsg[i] = &messages.Step{Text: text}
	}

	ruleScenarioStepsMsg := make([]*messages.Step, len(ruleScenarioSteps))
	for i, text := range ruleScenarioSteps {
		ruleScenarioStepsMsg[i] = &messages.Step{Text: text}
	}

	return &messages.GherkinDocument{
		Feature: &messages.Feature{
			Children: []*messages.FeatureChild{
				{
					Background: &messages.Background{Steps: featureBgStepsMsg},
				},
				{
					Rule: &messages.Rule{
						Children: []*messages.RuleChild{
							{
								Scenario: &messages.Scenario{Steps: ruleScenarioStepsMsg},
							},
						},
					},
				},
			},
		},
	}
}

// createDocumentWithFeatureAndRuleBackground creates a document with both feature and rule backgrounds
func createDocumentWithFeatureAndRuleBackground(featureBgSteps, ruleBgSteps, scenarioSteps []string) *messages.GherkinDocument {
	featureBgStepsMsg := make([]*messages.Step, len(featureBgSteps))
	for i, text := range featureBgSteps {
		featureBgStepsMsg[i] = &messages.Step{Text: text}
	}

	ruleBgStepsMsg := make([]*messages.Step, len(ruleBgSteps))
	for i, text := range ruleBgSteps {
		ruleBgStepsMsg[i] = &messages.Step{Text: text}
	}

	scenarioStepsMsg := make([]*messages.Step, len(scenarioSteps))
	for i, text := range scenarioSteps {
		scenarioStepsMsg[i] = &messages.Step{Text: text}
	}

	return &messages.GherkinDocument{
		Feature: &messages.Feature{
			Children: []*messages.FeatureChild{
				{
					Background: &messages.Background{Steps: featureBgStepsMsg},
				},
				{
					Rule: &messages.Rule{
						Children: []*messages.RuleChild{
							{
								Background: &messages.Background{Steps: ruleBgStepsMsg},
							},
							{
								Scenario: &messages.Scenario{Steps: scenarioStepsMsg},
							},
						},
					},
				},
			},
		},
	}
}

// createDocumentWithFeatureAndRuleBackgroundMultipleScenarios creates a document with both backgrounds and multiple scenarios
func createDocumentWithFeatureAndRuleBackgroundMultipleScenarios(featureBgSteps, ruleBgSteps []string, scenarioStepSets [][]string) *messages.GherkinDocument {
	featureBgStepsMsg := make([]*messages.Step, len(featureBgSteps))
	for i, text := range featureBgSteps {
		featureBgStepsMsg[i] = &messages.Step{Text: text}
	}

	ruleBgStepsMsg := make([]*messages.Step, len(ruleBgSteps))
	for i, text := range ruleBgSteps {
		ruleBgStepsMsg[i] = &messages.Step{Text: text}
	}

	ruleChildren := make([]*messages.RuleChild, len(scenarioStepSets)+1)
	ruleChildren[0] = &messages.RuleChild{
		Background: &messages.Background{Steps: ruleBgStepsMsg},
	}

	for i, stepSet := range scenarioStepSets {
		steps := make([]*messages.Step, len(stepSet))
		for j, text := range stepSet {
			steps[j] = &messages.Step{Text: text}
		}
		ruleChildren[i+1] = &messages.RuleChild{
			Scenario: &messages.Scenario{Steps: steps},
		}
	}

	return &messages.GherkinDocument{
		Feature: &messages.Feature{
			Children: []*messages.FeatureChild{
				{
					Background: &messages.Background{Steps: featureBgStepsMsg},
				},
				{
					Rule: &messages.Rule{
						Children: ruleChildren,
					},
				},
			},
		},
	}
}

// createDocumentWithMultipleRules creates a document with multiple rules
func createDocumentWithMultipleRules(rules ...[][]string) *messages.GherkinDocument {
	featureChildren := make([]*messages.FeatureChild, len(rules))

	for i, rule := range rules {
		ruleChildren := make([]*messages.RuleChild, len(rule))
		for j, scenarioSteps := range rule {
			steps := make([]*messages.Step, len(scenarioSteps))
			for k, text := range scenarioSteps {
				steps[k] = &messages.Step{Text: text}
			}
			ruleChildren[j] = &messages.RuleChild{
				Scenario: &messages.Scenario{Steps: steps},
			}
		}
		featureChildren[i] = &messages.FeatureChild{
			Rule: &messages.Rule{Children: ruleChildren},
		}
	}

	return &messages.GherkinDocument{
		Feature: &messages.Feature{
			Children: featureChildren,
		},
	}
}

// createDocumentWithBackgroundAndMultipleScenarios creates a document with background and multiple scenarios
func createDocumentWithBackgroundAndMultipleScenarios(bgSteps []string, scenarioStepSets [][]string) *messages.GherkinDocument {
	bgStepsMsg := make([]*messages.Step, len(bgSteps))
	for i, text := range bgSteps {
		bgStepsMsg[i] = &messages.Step{Text: text}
	}

	featureChildren := make([]*messages.FeatureChild, len(scenarioStepSets)+1)
	featureChildren[0] = &messages.FeatureChild{
		Background: &messages.Background{Steps: bgStepsMsg},
	}

	for i, stepSet := range scenarioStepSets {
		steps := make([]*messages.Step, len(stepSet))
		for j, text := range stepSet {
			steps[j] = &messages.Step{Text: text}
		}
		featureChildren[i+1] = &messages.FeatureChild{
			Scenario: &messages.Scenario{Steps: steps},
		}
	}

	return &messages.GherkinDocument{
		Feature: &messages.Feature{
			Children: featureChildren,
		},
	}
}

// createDocumentWithBackgroundSteps creates a document with multiple background steps and scenario steps
func createDocumentWithBackgroundSteps(bgSteps, scenarioSteps []string) *messages.GherkinDocument {
	bgStepsMsg := make([]*messages.Step, len(bgSteps))
	for i, text := range bgSteps {
		bgStepsMsg[i] = &messages.Step{Text: text}
	}

	scenarioStepsMsg := make([]*messages.Step, len(scenarioSteps))
	for i, text := range scenarioSteps {
		scenarioStepsMsg[i] = &messages.Step{Text: text}
	}

	return &messages.GherkinDocument{
		Feature: &messages.Feature{
			Children: []*messages.FeatureChild{
				{
					Background: &messages.Background{Steps: bgStepsMsg},
				},
				{
					Scenario: &messages.Scenario{Steps: scenarioStepsMsg},
				},
			},
		},
	}
}

// createComplexDocument creates a document with feature background, standalone scenario, and rule with scenario
func createComplexDocument(featureBgSteps, standaloneScenarioSteps, ruleScenarioSteps []string) *messages.GherkinDocument {
	featureBgStepsMsg := make([]*messages.Step, len(featureBgSteps))
	for i, text := range featureBgSteps {
		featureBgStepsMsg[i] = &messages.Step{Text: text}
	}

	standaloneStepsMsg := make([]*messages.Step, len(standaloneScenarioSteps))
	for i, text := range standaloneScenarioSteps {
		standaloneStepsMsg[i] = &messages.Step{Text: text}
	}

	ruleStepsMsg := make([]*messages.Step, len(ruleScenarioSteps))
	for i, text := range ruleScenarioSteps {
		ruleStepsMsg[i] = &messages.Step{Text: text}
	}

	return &messages.GherkinDocument{
		Feature: &messages.Feature{
			Children: []*messages.FeatureChild{
				{
					Background: &messages.Background{Steps: featureBgStepsMsg},
				},
				{
					Scenario: &messages.Scenario{Steps: standaloneStepsMsg},
				},
				{
					Rule: &messages.Rule{
						Children: []*messages.RuleChild{
							{
								Scenario: &messages.Scenario{Steps: ruleStepsMsg},
							},
						},
					},
				},
			},
		},
	}
}

// createDocumentWithDataTable creates a document with a single step that has an attached DataTable
func createDocumentWithDataTable(stepText string, tableData [][]string) *messages.GherkinDocument {
	rows := make([]*messages.TableRow, len(tableData))
	for i, rowData := range tableData {
		cells := make([]*messages.TableCell, len(rowData))
		for j, val := range rowData {
			cells[j] = &messages.TableCell{Value: val}
		}
		rows[i] = &messages.TableRow{Cells: cells}
	}

	return &messages.GherkinDocument{
		Feature: &messages.Feature{
			Children: []*messages.FeatureChild{
				{
					Scenario: &messages.Scenario{
						Steps: []*messages.Step{
							{
								Text:      stepText,
								DataTable: &messages.DataTable{Rows: rows},
							},
						},
					},
				},
			},
		},
	}
}
