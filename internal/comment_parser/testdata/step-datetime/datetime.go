package step_datetime

import (
	"time"

	"github.com/denizgursoy/cacik/pkg/cacik"
)

// AppointmentAt uses built-in {datetime} type - parses to time.Time with full date, time, and optional timezone
// Supports: ISO with space/T separator, with optional timezone
// @cacik `^the appointment is at {datetime}$`
func AppointmentAt(ctx *cacik.Context, dt time.Time) {
	ctx.Logger().Info("appointment at", "datetime", dt.Format(time.RFC3339))
}

// FlightDeparts demonstrates datetime with timezone
// @cacik `^the flight departs at {datetime}$`
func FlightDeparts(ctx *cacik.Context, dt time.Time) {
	ctx.Logger().Info("flight departs", "datetime", dt.Format(time.RFC3339), "location", dt.Location())
}
