package step_word

import (
	"github.com/denizgursoy/cacik/pkg/cacik"
)

// NameIs uses built-in {word} type to match a single word
// @cacik `^my name is {word}$`
func NameIs(ctx *cacik.Context, name string) {
	ctx.Logger().Info("my name is", "name", name)
}

// StatusIs uses {word} to match a status keyword
// @cacik `^the status is {word}$`
func StatusIs(ctx *cacik.Context, status string) {
	ctx.Logger().Info("status is", "status", status)
}
