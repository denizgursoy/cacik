package step_int

import (
	"github.com/denizgursoy/cacik/pkg/cacik"
)

// IGetApples
// @cacik `^I have (\d+) apples$`
func IGetApples(ctx *cacik.Context, appleCount int) {
	ctx.Logger().Info("I have apples", "count", appleCount)
}
