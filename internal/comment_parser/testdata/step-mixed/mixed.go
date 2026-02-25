package step_mixed

import (
	"time"

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

// ScheduleRange combines {date} and {time} in one step
// @cacik `^schedule from {date} at {time} to {date} at {time}$`
func ScheduleRange(ctx *cacik.Context, startDate, startTime, endDate, endTime time.Time) {
	start := time.Date(startDate.Year(), startDate.Month(), startDate.Day(),
		startTime.Hour(), startTime.Minute(), startTime.Second(), 0, startDate.Location())
	end := time.Date(endDate.Year(), endDate.Month(), endDate.Day(),
		endTime.Hour(), endTime.Minute(), endTime.Second(), 0, endDate.Location())
	ctx.Logger().Info("schedule", "from", start.Format(time.RFC3339), "to", end.Format(time.RFC3339))
}

// DeadlineWithCount combines {int}, {date}, and {time}
// @cacik `^I have {int} tasks due on {date} at {time}$`
func DeadlineWithCount(ctx *cacik.Context, count int, date, t time.Time) {
	deadline := time.Date(date.Year(), date.Month(), date.Day(),
		t.Hour(), t.Minute(), t.Second(), 0, date.Location())
	ctx.Logger().Info("tasks due", "count", count, "deadline", deadline.Format(time.RFC3339))
}

// EventAtDateTime combines {string} and {datetime}
// @cacik `^event {string} starts at {datetime}$`
func EventAtDateTime(ctx *cacik.Context, name string, dt time.Time) {
	ctx.Logger().Info("event starts", "name", name, "datetime", dt.Format(time.RFC3339))
}

// MeetingInTimezone combines {time} with explicit {timezone}
// @cacik `^meeting at {time} in {timezone}$`
func MeetingInTimezone(ctx *cacik.Context, t time.Time, loc *time.Location) {
	meeting := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, loc)
	ctx.Logger().Info("meeting", "time", meeting.Format("15:04:05"), "timezone", loc.String())
}

// ConvertDatetimeToTimezone converts a datetime to a different timezone
// @cacik `^convert {datetime} to {timezone}$`
func ConvertDatetimeToTimezone(ctx *cacik.Context, dt time.Time, loc *time.Location) {
	converted := dt.In(loc)
	ctx.Logger().Info("converted datetime", "result", converted.Format(time.RFC3339))
}
