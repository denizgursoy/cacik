package step_builtin

import (
	"context"
	"fmt"
)

// HaveApples uses built-in {int} type
// @cacik `^I have {int} apples$`
func HaveApples(ctx context.Context, count int) (context.Context, error) {
	fmt.Printf("I have %d apples\n", count)
	return ctx, nil
}

// PriceIs uses built-in {float} type
// @cacik `^the price is {float}$`
func PriceIs(ctx context.Context, price float64) (context.Context, error) {
	fmt.Printf("The price is %.2f\n", price)
	return ctx, nil
}

// NameIs uses built-in {word} type
// @cacik `^my name is {word}$`
func NameIs(ctx context.Context, name string) (context.Context, error) {
	fmt.Printf("My name is %s\n", name)
	return ctx, nil
}

// Say uses built-in {string} type (quoted string)
// @cacik `^I say {string}$`
func Say(ctx context.Context, message string) (context.Context, error) {
	fmt.Printf("I say: %s\n", message)
	return ctx, nil
}

// SeeAnything uses built-in {any} type
// @cacik `^I see {any}$`
func SeeAnything(ctx context.Context, thing string) (context.Context, error) {
	fmt.Printf("I see: %s\n", thing)
	return ctx, nil
}
