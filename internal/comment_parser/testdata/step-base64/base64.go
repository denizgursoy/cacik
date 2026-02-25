package step_base64

import (
	"github.com/denizgursoy/cacik/pkg/cacik"
)

// MatchBase64 uses built-in {base64} type
// @cacik `^the encoded data is {base64}$`
func MatchBase64(ctx *cacik.Context, data []byte) {
	ctx.Logger().Info("base64", "data", data)
}
