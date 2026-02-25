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

Step functions do not return anything. Use `ctx.Assert()` for assertions or `ctx.TestingT()` for direct test control:

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

### DataTables

When a Gherkin step has an attached DataTable, cacik converts it to a `cacik.Table` and auto-injects it into your step function, just like `*cacik.Context`.

```gherkin
Feature: User management

  Scenario: Create users
    Given the following users:
      | name  | age |
      | Alice | 30  |
      | Bob   | 25  |
```

```go
// @cacik `^the following users:$`
func TheFollowingUsers(ctx *cacik.Context, table cacik.Table) {
    for _, row := range table.SkipHeader() {
        name := row.Get("name")
        age := row.Get("age")
        ctx.Logger().Info("user", "name", name, "age", age)
    }
}
```

The `cacik.Table` parameter can appear anywhere in the function signature alongside `*cacik.Context` and regex capture arguments:

```gherkin
Feature: Inventory

  Scenario: Add items with details
    Given I have 3 items:
      | item   | price |
      | apple  | 1.50  |
      | banana | 0.75  |
      | cherry | 2.00  |
```

```go
// @cacik `^I have (\d+) items:$`
func IHaveItems(ctx *cacik.Context, count int, table cacik.Table) {
    ctx.Logger().Info("items", "count", count)
    for _, row := range table.SkipHeader() {
        ctx.Logger().Info("item", "name", row.Get("item"), "price", row.Get("price"))
    }
}
```

If a step function declares a `cacik.Table` parameter but the step has no DataTable attached, execution fails with an error.

#### Iterating Rows

`Table` provides two iterators using Go 1.24's range-over-func:

- **`All()`** - iterates over all rows including the header row
- **`SkipHeader()`** - iterates over data rows only (skips the first row)

Both return `iter.Seq2[int, Row]` where the int is a 0-based index.

```go
// Iterate all rows (including header)
for i, row := range table.All() {
    fmt.Println(i, row.Cell(0))
}

// Iterate data rows only (skip header)
for i, row := range table.SkipHeader() {
    name := row.Get("name")
    fmt.Println(i, name)
}
```

#### Row Access Methods

| Method | Description |
|--------|-------------|
| `row.Get(col)` | Lookup by column header name (case-insensitive) |
| `row.Cell(index)` | Lookup by column index (0-based) |
| `row.Values()` | Returns all cell values as `[]string` |
| `row.Len()` | Number of cells in the row |

#### Table Methods

| Method | Description |
|--------|-------------|
| `table.Headers()` | Returns column headers (first row values) |
| `table.Len()` | Total number of rows (including header) |
| `table.All()` | Iterator over all rows |
| `table.SkipHeader()` | Iterator over data rows only |

#### Headerless Tables

For tables without a meaningful header row, use `Cell(index)` for positional access:

```gherkin
Feature: Geometry

  Scenario: Plot coordinates
    Given the coordinates are:
      | 10 | 20 |
      | 30 | 40 |
      | 50 | 60 |
```

```go
// @cacik `^the coordinates are:$`
func Coordinates(table cacik.Table) {
    for _, row := range table.All() {
        x := row.Cell(0)
        y := row.Cell(1)
        fmt.Println(x, y)
    }
}
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

### Scenario ID

Each scenario execution gets a unique UUID (v4) via `ctx.ID()`. This is useful for correlating logs, creating unique test resources, or tagging external systems per scenario:

```go
// @cacik `^I create a test user$`
func CreateTestUser(ctx *cacik.Context) {
    username := fmt.Sprintf("test-user-%s", ctx.ID())
    ctx.Logger().Info("creating user", "id", ctx.ID(), "username", username)
    ctx.Data().Set("username", username)
}
```

When running in parallel, each scenario has its own context with a distinct ID, so there is no risk of collision.

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
	err := runner.NewCucumberRunner(t).
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

All scenarios run as parallel subtests via `t.Parallel()`. Concurrency is controlled by Go's built-in `-parallel` flag (defaults to `GOMAXPROCS`).

```shell
# Default: all scenarios run in parallel (limited by GOMAXPROCS)
go test -v ./...

# Limit to 4 concurrent scenarios
go test -v -parallel 4 ./...

# Run sequentially (one scenario at a time)
go test -v -parallel 1 ./...

# Combine with tags
go test -v -parallel 4 -- --tags "@smoke"
```

**Note:** `-parallel` is a native `go test` flag — no `--` separator needed for it. Use `--` only to separate cacik-specific flags like `--tags`.

### How It Works

- Each scenario runs as a `t.Run()` subtest that calls `t.Parallel()`
- Go's test runner controls how many parallel subtests execute concurrently
- Each scenario runs in complete isolation with its own `*cacik.Context`
- Background steps are re-executed for each scenario

### Context Isolation

Each scenario gets its own isolated context:

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

Each scenario has its own `Data()` store, so there's no risk of race conditions or data leakage between scenarios.

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

import (
	"fmt"
	"github.com/denizgursoy/cacik/pkg/cacik"
)

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
		BeforeScenario: func(s cacik.Scenario) {
			// Runs before each scenario
			fmt.Println("Starting scenario:", s.Name)
		},
		AfterScenario: func(s cacik.Scenario, err error) {
			// Runs after each scenario (always runs, even on failure)
			// err is nil on success, non-nil on failure
			if err != nil {
				fmt.Println("Scenario failed:", s.Name, err)
			}
		},
		BeforeStep: func(s cacik.Step) {
			// Runs before each step
			fmt.Println("Running step:", s.Keyword+s.Text)
		},
		AfterStep: func(s cacik.Step, err error) {
			// Runs after each step
			// err is nil on success, non-nil on failure
			if err != nil {
				fmt.Println("Step failed:", s.Text, err)
			}
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

### Hook Types

#### Scenario and Step Info

Scenario and step hooks receive metadata about the currently executing scenario or step:

```go
// cacik.Scenario — passed to BeforeScenario/AfterScenario
type Scenario struct {
    Name        string   // Scenario name (e.g. "User login")
    Tags        []string // Tags including inherited (e.g. "@smoke", "@auth")
    Description string   // Optional description text
    Keyword     string   // "Scenario" or "Scenario Outline"
    Line        int64    // Source file line number
}

// cacik.Step — passed to BeforeStep/AfterStep
type Step struct {
    Keyword string // Gherkin keyword with trailing space (e.g. "Given ", "When ")
    Text    string // Step text after keyword (e.g. "the user is logged in")
    Line    int64  // Source file line number
}
```

#### AfterScenario Always Runs

`AfterScenario` is guaranteed to run even if background steps or scenario steps fail. This makes it safe for cleanup logic (closing connections, resetting state, etc.):

```go
BeforeScenario: func(s cacik.Scenario) {
    db.Begin() // start transaction
},
AfterScenario: func(s cacik.Scenario, err error) {
    db.Rollback() // always rolls back, even on failure
},
```

### Hook Execution Order

1. **BeforeAll**: All hooks execute in `Order` ascending (0, 10, 20, ...)
2. **BeforeScenario**: All hooks execute in `Order` ascending (before each scenario)
3. **BeforeStep**: All hooks execute in `Order` ascending (before each step)
4. Step executes
5. **AfterStep**: All hooks execute in `Order` ascending (after each step, receives step error)
6. **AfterScenario**: All hooks execute in `Order` ascending (after each scenario, receives scenario error)
7. **AfterAll**: All hooks execute in `Order` ascending (after all scenarios)

### Hooks Fields

| Field | Type | Description |
|-------|------|-------------|
| `Order` | `int` | Execution order (lower = first, default: 0) |
| `BeforeAll` | `func()` | Runs once before all scenarios |
| `AfterAll` | `func()` | Runs once after all scenarios |
| `BeforeScenario` | `func(Scenario)` | Runs before each scenario |
| `AfterScenario` | `func(Scenario, error)` | Runs after each scenario (always runs; error is nil on success) |
| `BeforeStep` | `func(Step)` | Runs before each step |
| `AfterStep` | `func(Step, error)` | Runs after each step (error is nil on success) |
