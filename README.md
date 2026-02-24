# cacik

Cacik executes cucumber scenarios with Go functions. Cacik parses Go function comments starting with `@cacik` to find step definitions.

## Create files

Create your feature file and steps in a directory.

```
├── apple.feature
└── steps.go
```

apple.feature

```gherkin
Feature: My first feature

  Scenario: My first scenario
    When I have 3 apples
```

steps.go

```go
package myapp

import "github.com/denizgursoy/cacik/pkg/cacik"

// IHaveApples handles the step "I have X apples"
// @cacik `^I have (\d+) apples$`
func IHaveApples(ctx *cacik.Context, appleCount int) {
	ctx.Logger().Info("I have apples", "count", appleCount)
}
```

### Step Definition Syntax

- Use `// @cacik` followed by a backtick-enclosed pattern
- Use `{type}` placeholders for built-in types or custom types
- Arguments are automatically converted to the function parameter types

### Built-in Parameter Types

Cacik supports Cucumber-style parameter placeholders:

| Placeholder | Go Type | Description | Example Match |
|-------------|---------|-------------|---------------|
| `{int}` | `int` | Integer (positive/negative) | `42`, `-5` |
| `{float}` | `float64` | Floating point number | `3.14`, `-0.5` |
| `{word}` | `string` | Single word (no spaces) | `hello`, `test123` |
| `{string}` | `string` | Double-quoted string | `"hello world"` |
| `{any}` or `{}` | `string` | Matches anything | `anything here` |
| `{bool}` | `bool` | Boolean value | `true`, `false`, `1`, `0` |
| `{time}` | `time.Time` | Time values (zero date) | `14:30`, `2:30pm`, `14:30 Europe/London` |
| `{date}` | `time.Time` | Date values (midnight) | `15/01/2024`, `2024-01-15`, `15 Jan 2024` |
| `{datetime}` | `time.Time` | Date and time | `2024-01-15 14:30`, `2024-01-15T14:30:00Z` |
| `{timezone}` | `*time.Location` | Timezone | `UTC`, `Europe/London`, `+05:30` |
| `{email}` | `string` | Email address | `user@example.com`, `name+tag@domain.org` |
| `{duration}` | `time.Duration` | Go duration | `5s`, `1h30m`, `500ms` |
| `{url}` | `*url.URL` | HTTP/HTTPS URL | `https://example.com/path?q=1` |

Example:

```go
// @cacik `^I have {int} apples$`
func IHaveApples(ctx *cacik.Context, count int) {
    ctx.Logger().Info("I have apples", "count", count)
}

// @cacik `^the price is {float}$`
func PriceIs(ctx *cacik.Context, price float64) {
    ctx.Logger().Info("price set", "price", price)
}

// @cacik `^my name is {word}$`
func NameIs(ctx *cacik.Context, name string) {
    ctx.Logger().Info("name set", "name", name)
}

// @cacik `^I say {string}$`
func Say(ctx *cacik.Context, message string) {
    ctx.Logger().Info("saying message", "message", message)
}

// @cacik `^the meeting is at {time}$`
func MeetingAt(ctx *cacik.Context, t time.Time) {
    ctx.Logger().Info("meeting scheduled", "time", t.Format("15:04"))
}

// @cacik `^the event is on {date}$`
func EventOn(ctx *cacik.Context, d time.Time) {
    ctx.Logger().Info("event scheduled", "date", d.Format("2006-01-02"))
}

// @cacik `^the appointment is at {datetime}$`
func AppointmentAt(ctx *cacik.Context, dt time.Time) {
    ctx.Logger().Info("appointment scheduled", "datetime", dt.Format(time.RFC3339))
}

// @cacik `^convert to {timezone}$`
func ConvertTo(ctx *cacik.Context, loc *time.Location) {
    ctx.Logger().Info("converting timezone", "timezone", loc.String())
}

// @cacik `^user {email} logged in$`
func UserLoggedIn(ctx *cacik.Context, email string) {
    ctx.Logger().Info("user logged in", "email", email)
}

// @cacik `^wait for {duration}$`
func WaitFor(ctx *cacik.Context, d time.Duration) {
    ctx.Logger().Info("waiting", "duration", d)
}

// @cacik `^navigate to {url}$`
func NavigateTo(ctx *cacik.Context, u *url.URL) {
    ctx.Logger().Info("navigating", "url", u.String())
}

// @cacik `^the feature is {bool}$`
func FeatureEnabled(ctx *cacik.Context, enabled bool) {
    ctx.Logger().Info("feature state", "enabled", enabled)
}
```

Feature file:

```gherkin
Feature: Built-in types

  Scenario: Using built-in types
    Given I have 5 apples
    And the price is 19.99
    And my name is John
    And I say "Hello World"
    And the meeting is at 2:30pm
    And the event is on 15/01/2024
    And the appointment is at 2024-01-15 14:30
    And convert to Europe/London
    And user john@example.com logged in
    And wait for 5s
    And navigate to https://example.com/api
    And the feature is true
```

### Time, Date, DateTime, and Timezone Formats

All time-related types parse to Go's `time.Time` or `*time.Location` types.

#### `{time}` - Time Values → `time.Time`

Parses to `time.Time` with zero date (0001-01-01). Supports optional timezone.

| Format | Examples |
|--------|----------|
| 24-hour | `14:30`, `09:15`, `00:00`, `23:59` |
| With seconds | `14:30:45`, `09:15:00` |
| With milliseconds | `14:30:45.123`, `09:15:00.500` |
| 12-hour AM/PM | `2:30pm`, `9:15am`, `2:30 PM`, `12:00am` |
| With timezone Z | `14:30Z`, `14:30:00Z` |
| With timezone offset | `14:30+05:30`, `14:30-08:00`, `14:30+0530` |
| With IANA timezone | `14:30 Europe/London`, `2:30pm America/New_York` |

#### `{date}` - Date Values → `time.Time`

Parses to `time.Time` at midnight (00:00:00) in local timezone. **EU format (DD/MM/YYYY) is the default.**

| Format | Examples |
|--------|----------|
| EU (DD/MM/YYYY) - default | `15/01/2024`, `31/12/2024` |
| EU with dashes | `15-01-2024`, `31-12-2024` |
| EU with dots | `15.01.2024`, `31.12.2024` |
| ISO (YYYY-MM-DD) | `2024-01-15`, `2024-12-31` |
| ISO with slashes | `2024/01/15`, `2024/12/31` |
| Written (Day Month Year) | `15 Jan 2024`, `31 December 2024` |
| Written (Month Day, Year) | `Jan 15, 2024`, `January 15, 2024` |

#### `{datetime}` - DateTime Values → `time.Time`

Combines date and time. Supports optional timezone.

| Format | Examples |
|--------|----------|
| ISO with space | `2024-01-15 14:30`, `2024-01-15 14:30:45` |
| ISO with T | `2024-01-15T14:30`, `2024-01-15T14:30:45` |
| With milliseconds | `2024-01-15 14:30:45.123` |
| With AM/PM | `2024-01-15 2:30pm`, `15/01/2024 9:00am` |
| With timezone Z | `2024-01-15T14:30:00Z` |
| With timezone offset | `2024-01-15T14:30:00+05:30`, `2024-01-15 14:30-08:00` |
| With IANA timezone | `2024-01-15 14:30 Europe/London`, `15/01/2024 2:30pm America/New_York` |

#### `{timezone}` - Timezone Values → `*time.Location`

Parses to Go's `*time.Location`.

| Format | Examples |
|--------|----------|
| UTC | `UTC`, `Z` |
| Offset with colon | `+05:30`, `-08:00`, `+00:00` |
| Offset without colon | `+0530`, `-0800` |
| IANA timezone names | `Europe/London`, `America/New_York`, `Asia/Tokyo` |

### Supported Go Parameter Types

- `string` - text values
- `int`, `int8`, `int16`, `int32`, `int64` - integer values
- `uint`, `uint8`, `uint16`, `uint32`, `uint64` - unsigned integers
- `float32`, `float64` - floating point values
- `bool` - boolean values (see below)
- `time.Time` - for `{time}`, `{date}`, `{datetime}` types
- `*time.Location` - for `{timezone}` type
- `*cacik.Context` - automatically passed (should be first parameter)

### Using Regex Directly

You can also use raw regex patterns with capture groups:

```go
// Using regex capture group instead of {int}
// @cacik `^I have (\d+) apples$`
func IHaveApples(ctx *cacik.Context, count int) {
    // Step implementation
}
```

### Boolean Values

Use `{bool}` placeholder for boolean parameters. Accepts human-readable values (case-insensitive):

| Truthy | Falsy |
|--------|-------|
| `true` | `false` |
| `yes` | `no` |
| `on` | `off` |
| `enabled` | `disabled` |
| `1` | `0` |
| `t` | `f` |

Example:

```go
// FeatureToggle handles feature state
// @cacik `^the feature is {bool}$`
func FeatureToggle(ctx *cacik.Context, enabled bool) {
    ctx.Logger().Info("feature toggled", "enabled", enabled)
}
```

```gherkin
Feature: Feature toggles

  Scenario: Enable feature
    Given the feature is enabled

  Scenario: Disable feature
    Given the feature is disabled

  Scenario: Turn on
    Given the feature is on

  Scenario: Using yes/no
    Given the feature is yes
```

### Custom Parameter Types

Cacik supports custom enum-like types. Define a type based on a primitive and use constants to define allowed values:

```go
package steps

import (
    "fmt"
    "github.com/denizgursoy/cacik/pkg/cacik"
)

// Define a custom type based on string
type Color string

const (
    Red   Color = "red"
    Blue  Color = "blue"
    Green Color = "green"
)

// Use {typename} syntax in step definition
// @cacik `^I select {color}$`
func SelectColor(ctx *cacik.Context, c Color) {
    ctx.Logger().Info("color selected", "color", c)
}
```

Feature file:

```gherkin
Feature: Color selection

  Scenario: Select red
    When I select red

  Scenario: Select blue
    When I select blue
```

The `{color}` placeholder is automatically replaced with a regex pattern matching all defined constants. Invalid values are rejected at runtime.

#### Supported Custom Type Bases

Custom types can be based on any primitive type:

- `string` - e.g., `type Color string`
- `int`, `int8`, `int16`, `int32`, `int64` - e.g., `type Priority int`
- `uint`, `uint8`, `uint16`, `uint32`, `uint64`
- `float32`, `float64`
- `bool`

#### Integer-based Custom Types

For integer types, you can use either the constant name or value:

```go
type Priority int

const (
    Low    Priority = 1
    Medium Priority = 2
    High   Priority = 3
)

// @cacik `^priority is {priority}$`
func SetPriority(ctx *cacik.Context, p Priority) {
    ctx.Logger().Info("priority set", "priority", p)
}
```

```gherkin
# Both work:
Given priority is high    # matches High constant, p = 3
Given priority is 3       # direct value, p = 3
```

#### Case Sensitivity

Custom type matching is case-insensitive:

```gherkin
# All these match the Red constant:
When I select red
When I select RED
When I select Red
```

### Function Signature

Step functions do not return anything. Use `ctx.SetError()` or `ctx.Errorf()` for error handling:

```go
// Simple function with no arguments
func MyStep() {}

// Function with context
func MyStep(ctx *cacik.Context) {}

// Function with captured arguments
func MyStep(ctx *cacik.Context, arg1 int, arg2 string) {}

// Function without context but with arguments
func MyStep(count int, name string) {}

// Function with custom type
func MyStep(ctx *cacik.Context, color Color) {}
```

## Context API

The `*cacik.Context` provides logging, assertions, and state management for BDD tests.

### Logging

```go
func MyStep(ctx *cacik.Context) {
    ctx.Logger().Debug("debugging info", "key", "value")
    ctx.Logger().Info("informational message")
    ctx.Logger().Warn("warning message")
    ctx.Logger().Error("error message")
}
```

### State Management

Store and retrieve values across steps within a scenario via `ctx.Data()`:

```go
// @cacik `^I have {int} apples$`
func IHaveApples(ctx *cacik.Context, count int) {
    ctx.Data().Set("apples", count)
}

// @cacik `^I eat {int} apples$`
func IEatApples(ctx *cacik.Context, eaten int) {
    current := ctx.Data().MustGet("apples").(int)
    ctx.Data().Set("apples", current - eaten)
}

// @cacik `^I should have {int} apples$`
func IShouldHaveApples(ctx *cacik.Context, expected int) {
    actual := ctx.Data().MustGet("apples").(int)
    ctx.Assert().Equal(expected, actual, "apple count mismatch")
}
```

### Assertions

All assertions fail immediately (fail-fast behavior). Access assertions via `ctx.Assert()`:

```go
func MyStep(ctx *cacik.Context, value int) {
    // Equality
    ctx.Assert().Equal(expected, actual, "optional message")
    ctx.Assert().NotEqual(a, b)
    
    // Nil checks
    ctx.Assert().Nil(value)
    ctx.Assert().NotNil(value)
    
    // Boolean
    ctx.Assert().True(condition, "message")
    ctx.Assert().False(condition)
    
    // Errors
    ctx.Assert().NoError(err)
    ctx.Assert().Error(err)
    ctx.Assert().ErrorContains(err, "substring")
    
    // Collections
    ctx.Assert().Contains(slice, element)
    ctx.Assert().NotContains(slice, element)
    ctx.Assert().Len(collection, expectedLen)
    ctx.Assert().Empty(collection)
    ctx.Assert().NotEmpty(collection)
    
    // Comparisons
    ctx.Assert().Greater(5, 3)
    ctx.Assert().GreaterOrEqual(5, 5)
    ctx.Assert().Less(3, 5)
    ctx.Assert().LessOrEqual(5, 5)
    
    // Zero values
    ctx.Assert().Zero(value)
    ctx.Assert().NotZero(value)
    
    // Fail immediately
    ctx.Assert().Fail("reason")
}
```

### Access Standard Context

For compatibility with Go libraries that expect `context.Context`:

```go
func MyStep(ctx *cacik.Context) {
    // Get the underlying context.Context
    stdCtx := ctx.Context()
    
    // Use with libraries
    result, err := someLibrary.DoSomething(stdCtx)
    ctx.Assert().NoError(err)
    
    // Update the context (for timeouts, cancellation, etc.)
    ctx.WithContext(context.WithTimeout(stdCtx, 5*time.Second))
}
```

## Install

```shell
go install github.com/denizgursoy/cacik/cmd/cacik@latest
```

## Execute `cacik` to create cacik_test.go

```shell
cacik
```

Cacik will detect your package name and create a Go test file:

```
├── apple.feature
├── cacik_test.go
└── steps.go
```

cacik_test.go

```go
package myapp

import (
	runner "github.com/denizgursoy/cacik/pkg/runner"
	"testing"
)

func TestCacik(t *testing.T) {
	err := runner.NewCucumberRunner().
		WithTestingT(t).
		RegisterStep("^I have (\\d+) apples$", IHaveApples).
		Run()

	if err != nil {
		t.Fatal(err)
	}
}
```

Since the step functions are in the same package, they are called directly without an import qualifier. If steps are in a different package, cacik will add the appropriate import and qualifier automatically.

## Execute tests

To execute scenarios in the feature file, run:

```shell
go test -v
```

Each scenario runs as a Go subtest via `t.Run()`, so you get standard `go test` output with per-scenario pass/fail reporting. Assertion failures use `t.Fatalf()` instead of panicking.

## Parallel Execution

Run scenarios in parallel using the `--parallel` flag:

```shell
# Run with 4 workers
go test -v -- --parallel 4

# Alternative syntax
go test -v -- --parallel=8

# Combine with tags
go test -v -- --tags "@smoke" --parallel 4
```

**Note:** Use `--` to separate `go test` flags from cacik flags.

### How It Works

- When using `WithTestingT(t)`, each scenario runs as a `t.Run()` subtest
- Parallel scenarios use `t.Parallel()` inside their subtests, leveraging Go's native test parallelism
- Each scenario runs in complete isolation with its own `*cacik.Context`
- Background steps are re-executed for each scenario
- Default: 1 (sequential execution)

### Context Isolation

When running in parallel, each scenario gets its own isolated context:

```go
// @cacik `^I set value to {int}$`
func SetValue(ctx *cacik.Context, val int) {
    ctx.Data().Set("value", val)  // This is isolated per scenario
}

// @cacik `^the value should be {int}$`
func CheckValue(ctx *cacik.Context, expected int) {
    actual := ctx.Data().MustGet("value").(int)
    ctx.Assert().Equal(expected, actual)
}
```

Each parallel scenario has its own `Data()` store, so there's no risk of race conditions or data leakage between scenarios.

## Running with Tags

Cacik supports Cucumber tag expressions for filtering scenarios. Tags are passed via the `--tags` command-line flag.

### Tag Expression Syntax

Tag expressions support `and`, `or`, `not` operators and parentheses for complex filtering:

| Expression | Description |
|------------|-------------|
| `@smoke` | Scenarios tagged with `@smoke` |
| `@smoke and @fast` | Scenarios with both `@smoke` AND `@fast` |
| `@gui or @database` | Scenarios with either `@gui` OR `@database` |
| `not @slow` | Scenarios NOT tagged with `@slow` |
| `@wip and not @slow` | Scenarios with `@wip` but NOT `@slow` |
| `(@smoke or @ui) and not @slow` | Complex expression with parentheses |

### Examples

```shell
# Run all scenarios
go test -v

# Run only @smoke scenarios
go test -v -- --tags "@smoke"

# Run scenarios with both @smoke AND @fast
go test -v -- --tags "@smoke and @fast"

# Run scenarios with @gui OR @database
go test -v -- --tags "@gui or @database"

# Run scenarios that are NOT @slow
go test -v -- --tags "not @slow"

# Complex expression
go test -v -- --tags "(@smoke or @ui) and not @slow"

# Alternative syntax with equals sign
go test -v -- --tags="@smoke and @fast"
```

### Tag Inheritance

Tags are inherited following the Gherkin specification:
- **Scenario** inherits tags from its parent **Feature**
- **Scenario** inside a **Rule** inherits tags from both **Feature** and **Rule**

```gherkin
@billing
Feature: Billing

  @smoke
  Scenario: Quick payment
    # This scenario has both @billing and @smoke tags
    When I make a payment

  Rule: Subscriptions
    @subscription
    Scenario: Monthly billing
      # This scenario has @billing, @subscription tags
      When I check my subscription
```

```shell
# Matches "Quick payment" (has @billing)
go test -v -- --tags "@billing"

# Matches "Quick payment" (has @smoke)
go test -v -- --tags "@smoke"

# Matches "Monthly billing" (has @subscription)
go test -v -- --tags "@subscription"

# Matches both scenarios (both have @billing from feature)
go test -v -- --tags "@billing"

# Matches "Quick payment" only (needs both @billing AND @smoke)
go test -v -- --tags "@billing and @smoke"
```

## Configuration

Cacik automatically discovers functions returning `*cacik.Config` for runtime settings. CLI flags always override config values.

```go
package mysteps

import "github.com/denizgursoy/cacik/pkg/cacik"

// MyConfig returns runtime configuration
func MyConfig() *cacik.Config {
	return &cacik.Config{
		Parallel:        4,            // Number of parallel workers (0 = sequential)
		FailFast:        true,         // Stop on first failure
		NoColor:         false,        // Colored output (default: true)
		DisableLog:      false,        // Logger (ctx.Logger()) enabled (default: false)
		DisableReporter: false,        // Reporter output enabled (default: false)
		Logger:          customLogger, // Custom logger (default: slog)
	}
}
```

### Config Fields

| Field | Type | Description | CLI Override |
|-------|------|-------------|--------------|
| `Parallel` | `int` | Number of parallel workers (0 = sequential) | `--parallel N` |
| `FailFast` | `bool` | Stop execution on first failure | `--fail-fast` |
| `NoColor` | `bool` | Disable colored output | `--no-color` |
| `DisableLog` | `bool` | Disable the structured logger (`ctx.Logger()`) | `--disable-log` |
| `DisableReporter` | `bool` | Disable reporter output (feature/scenario/step lines) | `--disable-reporter` |
| `Logger` | `cacik.Logger` | Custom logger (default: slog to stdout) | - |

Multiple config functions are merged (last wins for conflicts).

## Hooks

Cacik automatically discovers functions returning `*cacik.Hooks` for lifecycle hooks. ALL discovered hooks are executed, sorted by their `Order` field.

```go
package database

import "github.com/denizgursoy/cacik/pkg/cacik"

// DatabaseHooks sets up database connection
func DatabaseHooks() *cacik.Hooks {
	return &cacik.Hooks{
		Order: 10, // Lower = runs first (default: 0)
		BeforeAll: func() {
			// Setup database connection (runs once before all scenarios)
		},
		AfterAll: func() {
			// Close database connection (runs once after all scenarios)
		},
		BeforeStep: func() {
			// Runs before each step
		},
		AfterStep: func() {
			// Runs after each step
		},
	}
}
```

```go
package api

import "github.com/denizgursoy/cacik/pkg/cacik"

// APIHooks sets up mock API server
func APIHooks() *cacik.Hooks {
	return &cacik.Hooks{
		Order: 20, // Runs after DatabaseHooks (Order: 10)
		BeforeAll: func() {
			// Start mock API server (needs database)
		},
		AfterAll: func() {
			// Stop mock API server
		},
	}
}
```

### Hook Execution Order

1. **BeforeAll**: All hooks execute in `Order` ascending (0, 10, 20, ...)
2. **BeforeStep**: All hooks execute in `Order` ascending (before each step)
3. Step executes
4. **AfterStep**: All hooks execute in `Order` ascending (after each step)
5. **AfterAll**: All hooks execute in `Order` ascending (after all scenarios)

### Hooks Fields

| Field | Type | Description |
|-------|------|-------------|
| `Order` | `int` | Execution order (lower = first, default: 0) |
| `BeforeAll` | `func()` | Runs once before all scenarios |
| `AfterAll` | `func()` | Runs once after all scenarios |
| `BeforeStep` | `func()` | Runs before each step |
| `AfterStep` | `func()` | Runs after each step |
