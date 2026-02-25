package step_timezone

import (
	"time"

	"github.com/denizgursoy/cacik/pkg/cacik"
)

// ConvertToTimezone uses standalone {timezone} type - parses to *time.Location
// Supports: IANA names (Europe/London), UTC, Z, offsets (+05:30, -08:00)
// @cacik `^convert to {timezone}$`
func ConvertToTimezone(ctx *cacik.Context, loc *time.Location) {
	ctx.Logger().Info("convert to timezone", "timezone", loc.String())
}

// ShowTimeIn shows current time in a specific timezone
// @cacik `^show current time in {timezone}$`
func ShowTimeIn(ctx *cacik.Context, loc *time.Location) {
	now := time.Now().In(loc)
	ctx.Logger().Info("current time in timezone", "timezone", loc.String(), "time", now.Format("15:04:05"))
}
