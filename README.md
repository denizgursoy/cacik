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

- Use `// @cacik` followed by a backtick-enclosed regex pattern
- Use capture groups `()` to extract arguments from the step text
- Arguments are automatically converted to the function parameter types

### Supported Parameter Types

- `string` - text values
- `int`, `int8`, `int16`, `int32`, `int64` - integer values
- `uint`, `uint8`, `uint16`, `uint32`, `uint64` - unsigned integers
- `float32`, `float64` - floating point values
- `bool` - boolean values (see below)
- `context.Context` - automatically passed (should be first parameter)

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
