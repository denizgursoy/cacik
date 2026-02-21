package step_float

import (
	"context"
	"fmt"
)

// ItemCosts checks the price using {float} built-in type
// @cacik `^the item costs {float} dollars$`
func ItemCosts(ctx context.Context, price float64) (context.Context, error) {
	fmt.Printf("The item costs %.2f dollars\n", price)
	return ctx, nil
}

// TemperatureIs checks the temperature
// @cacik `^the temperature is {float} degrees$`
func TemperatureIs(ctx context.Context, temp float64) (context.Context, error) {
	fmt.Printf("The temperature is %.1f degrees\n", temp)
	return ctx, nil
}
