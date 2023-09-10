# cacik

## Create directories

Create your feature file and steps in a directory.

```
├── apple.feature
└── steps.go
```

apple.feature

```gherkin
Feature: My first feature

  Scenario: My first scenario
    When I get 3 apples
```

steps.go

```go
package main

import (
	"context"
	"fmt"
)

// IGetApples
// @cacik `I have \d apples`
func IGetApples(ctx context.Context, appleCount int) (context.Context, error) {
	fmt.Printf("I have %d apples", appleCount)

	return ctx, nil
}

```

## Install

```shell
go install github.com/denizgursoy/cacik/cmd/cacik@latest
```

## Execute `cacik` to crate main.go

```shell
cacik
```

Cacik will create main file

```
├── apple.feature
├── main.go
└── steps.go
```

## Execute main.go

To execute scenarios in the feature file, execute:

```shell
go run main.go
```
