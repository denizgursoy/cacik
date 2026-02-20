package step_color

import (
	"context"
	"fmt"
)

// Color represents a color choice
type Color string

const (
	Red   Color = "red"
	Blue  Color = "blue"
	Green Color = "green"
)

// SelectColor selects a color
// @cacik `^I select {color}$`
func SelectColor(ctx context.Context, c Color) (context.Context, error) {
	fmt.Printf("Selected color: %s\n", c)
	return ctx, nil
}

// ColorIs checks if the color matches
// @cacik `^the color is {color}$`
func ColorIs(ctx context.Context, c Color) (context.Context, error) {
	fmt.Printf("The color is: %s\n", c)
	return ctx, nil
}
