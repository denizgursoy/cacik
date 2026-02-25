package step_phone

import (
	"github.com/denizgursoy/cacik/pkg/cacik"
)

// MatchPhone uses built-in {phone} type
// @cacik `^the contact number is {phone}$`
func MatchPhone(ctx *cacik.Context, number string) {
	ctx.Logger().Info("phone", "number", number)
}
