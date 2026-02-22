package step_builtin

import (
	"github.com/denizgursoy/cacik/pkg/cacik"
)

// HaveApples uses built-in {int} type
// @cacik `^I have {int} apples$`
func HaveApples(ctx *cacik.Context, count int) {
	ctx.Logger().Info("I have apples", "count", count)
}

// PriceIs uses built-in {float} type
// @cacik `^the price is {float}$`
func PriceIs(ctx *cacik.Context, price float64) {
	ctx.Logger().Info("price is", "price", price)
}

// NameIs uses built-in {word} type
// @cacik `^my name is {word}$`
func NameIs(ctx *cacik.Context, name string) {
	ctx.Logger().Info("my name is", "name", name)
}

// Say uses built-in {string} type (quoted string)
// @cacik `^I say {string}$`
func Say(ctx *cacik.Context, message string) {
	ctx.Logger().Info("I say", "message", message)
}

// SeeAnything uses built-in {any} type
// @cacik `^I see {any}$`
func SeeAnything(ctx *cacik.Context, thing string) {
	ctx.Logger().Info("I see", "thing", thing)
}
