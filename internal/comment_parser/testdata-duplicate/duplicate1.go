package step_duplicate

import (
	"github.com/denizgursoy/cacik/pkg/cacik"
)

// FirstDuplicateStep is the first definition of a duplicate step
// @cacik `^I have (\d+) items$`
func FirstDuplicateStep(ctx *cacik.Context, count int) {
	ctx.Logger().Info("first duplicate step", "count", count)
}
