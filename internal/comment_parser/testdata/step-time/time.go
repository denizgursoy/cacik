package step_time

import (
	"time"

	"github.com/denizgursoy/cacik/pkg/cacik"
)

// MeetingAt uses built-in {time} type - parses to time.Time with zero date (0001-01-01)
// Supports: HH:MM, HH:MM:SS, HH:MM:SS.mmm, with optional AM/PM and timezone
// @cacik `^the meeting is at {time}$`
func MeetingAt(ctx *cacik.Context, t time.Time) {
	ctx.Logger().Info("meeting at", "time", t.Format("15:04:05"), "location", t.Location())
}

// TimeBetween checks time range with two {time} parameters
// @cacik `^the store is open between {time} and {time}$`
func TimeBetween(ctx *cacik.Context, openTime, closeTime time.Time) {
	ctx.Logger().Info("store open", "from", openTime.Format("15:04"), "to", closeTime.Format("15:04"))
}
