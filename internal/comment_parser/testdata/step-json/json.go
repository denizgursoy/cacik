package step_json

import (
	"github.com/denizgursoy/cacik/pkg/cacik"
)

// MatchJSON uses built-in {json} type
// @cacik `^the payload is {json}$`
func MatchJSON(ctx *cacik.Context, payload string) {
	ctx.Logger().Info("json", "payload", payload)
}
