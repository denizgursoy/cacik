package step_mixed

import (
	"context"
	"fmt"
)

// Color represents a color choice
type Color string

const (
	Red   Color = "red"
	Blue  Color = "blue"
	Green Color = "green"
)

// Priority represents task priority
type Priority int

const (
	PriorityLow    Priority = 1
	PriorityMedium Priority = 2
	PriorityHigh   Priority = 3
)

// Size represents item size
type Size string

const (
	SizeSmall  Size = "small"
	SizeMedium Size = "medium"
	SizeLarge  Size = "large"
)

// WantColoredVehicle combines custom type {color}, normal regex (car|bike), {int}, and {float}
// @cacik `^I want a {color} (car|bike) with {int} doors costing {float} dollars$`
func WantColoredVehicle(ctx context.Context, color Color, vehicle string, doors int, price float64) (context.Context, error) {
	fmt.Printf("Want a %s %s with %d doors costing %.2f dollars\n", color, vehicle, doors, price)
	return ctx, nil
}

// NamedItemWithPriority combines {color}, {string}, and {priority}
// @cacik `^a {color} item named {string} at {priority} priority$`
func NamedItemWithPriority(ctx context.Context, color Color, name string, priority Priority) (context.Context, error) {
	fmt.Printf("A %s item named %q at priority %d\n", color, name, priority)
	return ctx, nil
}

// OwnedByWithVisibility combines {color}, {word}, and boolean
// @cacik `^{color} owned by {word} is (true|false|yes|no)$`
func OwnedByWithVisibility(ctx context.Context, color Color, owner string, visible bool) (context.Context, error) {
	fmt.Printf("%s owned by %s, visible: %v\n", color, owner, visible)
	return ctx, nil
}

// SizedItemCount combines {size}, {int}, and {color}
// @cacik `^I have {int} {size} {color} boxes$`
func SizedItemCount(ctx context.Context, count int, size Size, color Color) (context.Context, error) {
	fmt.Printf("I have %d %s %s boxes\n", count, size, color)
	return ctx, nil
}

// ProductWithAllTypes combines {word}, {color}, {size}, {float}, {priority}, and {string}
// @cacik `^product {word} is {color} and {size} priced at {float} with {priority} priority described as {string}$`
func ProductWithAllTypes(ctx context.Context, sku string, color Color, size Size, price float64, priority Priority, description string) (context.Context, error) {
	fmt.Printf("Product %s: %s %s, $%.2f, priority %d, desc: %q\n", sku, color, size, price, priority, description)
	return ctx, nil
}

// QuantityWithAny combines {int} and {any}
// @cacik `^I ordered {int} of {any}$`
func QuantityWithAny(ctx context.Context, quantity int, item string) (context.Context, error) {
	fmt.Printf("Ordered %d of %s\n", quantity, item)
	return ctx, nil
}

// ConditionalAction combines normal regex with {color} and boolean
// @cacik `^(enable|disable) the {color} (button|switch) and set active to (true|false)$`
func ConditionalAction(ctx context.Context, action string, color Color, element string, active bool) (context.Context, error) {
	fmt.Printf("%s the %s %s, active: %v\n", action, color, element, active)
	return ctx, nil
}
