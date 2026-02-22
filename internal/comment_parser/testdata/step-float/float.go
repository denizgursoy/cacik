package step_float

import (
	"github.com/denizgursoy/cacik/pkg/cacik"
)

// ItemCosts checks the price using {float} built-in type
// @cacik `^the item costs {float} dollars$`
func ItemCosts(ctx *cacik.Context, price float64) {
	ctx.Logger().Info("item costs", "price", price)
}

// TemperatureIs checks the temperature
// @cacik `^the temperature is {float} degrees$`
func TemperatureIs(ctx *cacik.Context, temp float64) {
	ctx.Logger().Info("temperature is", "temp", temp)
}
