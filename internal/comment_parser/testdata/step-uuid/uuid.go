package step_uuid

import (
	"github.com/denizgursoy/cacik/pkg/cacik"
)

// MatchUUID uses built-in {uuid} type
// @cacik `^the identifier is {uuid}$`
func MatchUUID(ctx *cacik.Context, id string) {
	ctx.Logger().Info("uuid", "id", id)
}
