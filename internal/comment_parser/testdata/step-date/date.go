package step_date

import (
	"time"

	"github.com/denizgursoy/cacik/pkg/cacik"
)

// EventOn uses built-in {date} type - parses to time.Time at midnight in Local timezone
// Supports: EU (DD/MM/YYYY), ISO (YYYY-MM-DD), written (15 Jan 2024)
// @cacik `^the event is on {date}$`
func EventOn(ctx *cacik.Context, d time.Time) {
	ctx.Logger().Info("event on", "date", d.Format("2006-01-02"))
}

// DateRange checks date range with two {date} parameters
// @cacik `^the sale runs from {date} to {date}$`
func DateRange(ctx *cacik.Context, startDate, endDate time.Time) {
	ctx.Logger().Info("sale runs", "from", startDate.Format("2006-01-02"), "to", endDate.Format("2006-01-02"))
}
