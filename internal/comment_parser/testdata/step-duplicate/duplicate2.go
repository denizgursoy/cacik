package step_duplicate

import (
	"github.com/denizgursoy/cacik/pkg/cacik"
)

// SecondDuplicateStep is the second definition of the same step pattern
// @cacik `^I have (\d+) items$`
func SecondDuplicateStep(ctx *cacik.Context, count int) {
	ctx.Logger().Info("second duplicate step", "count", count)
}
