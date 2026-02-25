package step_hex

import (
	"github.com/denizgursoy/cacik/pkg/cacik"
)

// MatchHex uses built-in {hex} type
// @cacik `^the color code is {hex}$`
func MatchHex(ctx *cacik.Context, value int64) {
	ctx.Logger().Info("hex", "value", value)
}
