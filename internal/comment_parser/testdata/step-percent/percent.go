package step_percent

import (
	"github.com/denizgursoy/cacik/pkg/cacik"
)

// MatchPercent uses built-in {percent} type
// @cacik `^the discount is {percent}$`
func MatchPercent(ctx *cacik.Context, pct float64) {
	ctx.Logger().Info("percent", "pct", pct)
}
