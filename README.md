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
package main

import (
	"context"
	"fmt"
)

// IHaveApples handles the step "I have X apples"
// @cacik `^I have (\d+) apples$`
func IHaveApples(ctx context.Context, appleCount int) (context.Context, error) {
	fmt.Printf("I have %d apples\n", appleCount)

	return ctx, nil
}
```

### Step Definition Syntax

- Use `// @cacik` followed by a backtick-enclosed pattern
- Use `{type}` placeholders for built-in types or custom types
- Arguments are automatically converted to the function parameter types

### Built-in Parameter Types

Cacik supports Cucumber-style parameter placeholders:

| Placeholder | Pattern | Description | Example Match |
|-------------|---------|-------------|---------------|
| `{int}` | `-?\d+` | Integer (positive/negative) | `42`, `-5` |
| `{float}` | `-?\d*\.?\d+` | Floating point number | `3.14`, `-0.5` |
| `{word}` | `\w+` | Single word (no spaces) | `hello`, `test123` |
| `{string}` | `"[^"]*"` | Double-quoted string | `"hello world"` |
| `{any}` or `{}` | `.*` | Matches anything | `anything here` |

Example:

```go
// @cacik `^I have {int} apples$`
func IHaveApples(ctx context.Context, count int) (context.Context, error) {
    fmt.Printf("I have %d apples\n", count)
    return ctx, nil
}

// @cacik `^the price is {float}$`
func PriceIs(ctx context.Context, price float64) (context.Context, error) {
    fmt.Printf("Price: %.2f\n", price)
    return ctx, nil
}

// @cacik `^my name is {word}$`
func NameIs(ctx context.Context, name string) (context.Context, error) {
    fmt.Printf("Name: %s\n", name)
    return ctx, nil
}

// @cacik `^I say {string}$`
func Say(ctx context.Context, message string) (context.Context, error) {
    fmt.Printf("Message: %s\n", message)
    return ctx, nil
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
```

### Supported Go Parameter Types

- `string` - text values
- `int`, `int8`, `int16`, `int32`, `int64` - integer values
- `uint`, `uint8`, `uint16`, `uint32`, `uint64` - unsigned integers
- `float32`, `float64` - floating point values
- `bool` - boolean values (see below)
- `context.Context` - automatically passed (should be first parameter)

### Using Regex Directly

You can also use raw regex patterns with capture groups:

```go
// Using regex capture group instead of {int}
// @cacik `^I have (\d+) apples$`
func IHaveApples(ctx context.Context, count int) (context.Context, error) {
    return ctx, nil
}
```

### Boolean Values

Boolean parameters support human-readable values (case-insensitive):

| Truthy | Falsy |
|--------|-------|
| `true` | `false` |
| `yes` | `no` |
| `on` | `off` |
| `enabled` | `disabled` |
| `1` | `0` |

Example:

```go
// FeatureToggle handles feature state
// @cacik `^the feature is (enabled|disabled)$`
func FeatureToggle(ctx context.Context, enabled bool) (context.Context, error) {
    if enabled {
        fmt.Println("Feature is ON")
    } else {
        fmt.Println("Feature is OFF")
    }
    return ctx, nil
}
```

```gherkin
Feature: Feature toggles

  Scenario: Enable feature
    Given the feature is enabled

  Scenario: Disable feature
    Given the feature is disabled
```

### Custom Parameter Types

Cacik supports custom enum-like types. Define a type based on a primitive and use constants to define allowed values:

```go
package steps

import (
    "context"
    "fmt"
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
func SelectColor(ctx context.Context, c Color) (context.Context, error) {
    fmt.Printf("Selected: %s\n", c)
    return ctx, nil
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
func SetPriority(ctx context.Context, p Priority) (context.Context, error) {
    fmt.Printf("Priority: %d\n", p)
    return ctx, nil
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

Step functions can have the following signatures:

```go
// Simple function with no arguments
func MyStep() {}

// Function with context
func MyStep(ctx context.Context) (context.Context, error) {}

// Function with captured arguments
func MyStep(ctx context.Context, arg1 int, arg2 string) (context.Context, error) {}

// Function without context but with arguments
func MyStep(count int, name string) error {}

// Function with custom type
func MyStep(ctx context.Context, color Color) (context.Context, error) {}
```

## Install

```shell
go install github.com/denizgursoy/cacik/cmd/cacik@latest
```

## Execute `cacik` to create main.go

```shell
cacik
```

Cacik will create the main file:

```
├── apple.feature
├── main.go
└── steps.go
```

main.go

```go
package main

import (
	runner "github.com/denizgursoy/cacik/pkg/runner"
	"log"
)

func main() {
	err := runner.NewCucumberRunner().
		RegisterStep("^I have (\\d+) apples$", IHaveApples).
		RunWithTags()

	if err != nil {
		log.Fatal(err)
	}
}
```

## Execute main.go

To execute scenarios in the feature file, run:

```shell
go run .
```

It will print `I have 3 apples`

## Running with Tags

You can filter scenarios by tags:

```gherkin
@smoke
Feature: My feature

  @important
  Scenario: Important test
    When I have 5 apples
```

```go
// Run only scenarios with @smoke or @important tags
runner.NewCucumberRunner().
    RegisterStep("^I have (\\d+) apples$", IHaveApples).
    RunWithTags("smoke", "important")
```

## Configuration and Hooks

You can configure hooks by creating a config function:

```go
package mysteps

import "github.com/denizgursoy/cacik/pkg/models"

func GetConfig() *models.Config {
	return &models.Config{
		BeforeAll:  func() { /* setup */ },
		AfterAll:   func() { /* teardown */ },
		BeforeStep: func() { /* before each step */ },
		AfterStep:  func() { /* after each step */ },
	}
}
```
