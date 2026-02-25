package step_csv

import (
	"github.com/denizgursoy/cacik/pkg/cacik"
)

// MatchCSV uses built-in {csv} type
// @cacik `^the items are {csv}$`
func MatchCSV(ctx *cacik.Context, items []string) {
	ctx.Logger().Info("csv", "items", items)
}
