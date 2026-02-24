package step_mixed

import (
	"github.com/denizgursoy/cacik/pkg/cacik"
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
	Low    Priority = 1
	Medium Priority = 2
	High   Priority = 3
)

// Size represents item size
type Size string

const (
	Small      Size = "small"
	MediumSize Size = "medium"
	Large      Size = "large"
)

// WantColoredVehicle combines custom type {color}, normal regex (car|bike), {int}, and {float}
// @cacik `^I want a {color} (car|bike) with {int} doors costing {float} dollars$`
func WantColoredVehicle(ctx *cacik.Context, color Color, vehicle string, doors int, price float64) {
	ctx.Logger().Info("want colored vehicle", "color", color, "vehicle", vehicle, "doors", doors, "price", price)
}

// NamedItemWithPriority combines {color}, {string}, and {priority}
// @cacik `^a {color} item named {string} at {priority} priority$`
func NamedItemWithPriority(ctx *cacik.Context, color Color, name string, priority Priority) {
	ctx.Logger().Info("named item with priority", "color", color, "name", name, "priority", priority)
}

// OwnedByWithVisibility combines {color}, {word}, and boolean
// @cacik `^{color} owned by {word} is (true|false|yes|no)$`
func OwnedByWithVisibility(ctx *cacik.Context, color Color, owner string, visible bool) {
	ctx.Logger().Info("owned by with visibility", "color", color, "owner", owner, "visible", visible)
}

// SizedItemCount combines {size}, {int}, and {color}
// @cacik `^I have {int} {size} {color} boxes$`
func SizedItemCount(ctx *cacik.Context, count int, size Size, color Color) {
	ctx.Logger().Info("sized item count", "count", count, "size", size, "color", color)
}

// ProductWithAllTypes combines {word}, {color}, {size}, {float}, {priority}, and {string}
// @cacik `^product {word} is {color} and {size} priced at {float} with {priority} priority described as {string}$`
func ProductWithAllTypes(ctx *cacik.Context, sku string, color Color, size Size, price float64, priority Priority, description string) {
	ctx.Logger().Info("product with all types", "sku", sku, "color", color, "size", size, "price", price, "priority", priority, "description", description)
}

// QuantityWithAny combines {int} and {any}
// @cacik `^I ordered {int} of {any}$`
func QuantityWithAny(ctx *cacik.Context, quantity int, item string) {
	ctx.Logger().Info("quantity with any", "quantity", quantity, "item", item)
}

// ConditionalAction combines normal regex with {color} and boolean
// @cacik `^(enable|disable) the {color} (button|switch) and set active to {bool}$`
func ConditionalAction(ctx *cacik.Context, action string, color Color, element string, active bool) {
	ctx.Logger().Info("conditional action", "action", action, "color", color, "element", element, "active", active)
}
