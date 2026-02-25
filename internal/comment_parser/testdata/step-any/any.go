package step_any

import (
	"github.com/denizgursoy/cacik/pkg/cacik"
)

// SeeAnything uses built-in {any} type to match any text
// @cacik `^I see {any}$`
func SeeAnything(ctx *cacik.Context, thing string) {
	ctx.Logger().Info("I see", "thing", thing)
}

// DescriptionIs uses {any} for free-form text
// @cacik `^the description is {any}$`
func DescriptionIs(ctx *cacik.Context, desc string) {
	ctx.Logger().Info("description is", "desc", desc)
}
