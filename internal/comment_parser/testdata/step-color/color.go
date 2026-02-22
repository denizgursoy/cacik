package step_color

import (
	"github.com/denizgursoy/cacik/pkg/cacik"
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
func SelectColor(ctx *cacik.Context, c Color) {
	ctx.Logger().Info("color selected", "color", c)
}

// ColorIs checks if the color matches
// @cacik `^the color is {color}$`
func ColorIs(ctx *cacik.Context, c Color) {
	ctx.Logger().Info("color is", "color", c)
}
