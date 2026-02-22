package step_datetime

import (
	"time"

	"github.com/denizgursoy/cacik/pkg/cacik"
)

// ================
// TIME STEPS
// ================

// MeetingAt uses built-in {time} type - parses to time.Time with zero date (0001-01-01)
// Supports: HH:MM, HH:MM:SS, HH:MM:SS.mmm, with optional AM/PM and timezone
// Examples: 14:30, 2:30pm, 14:30:45, 14:30+05:30, 2:30pm Europe/London
// @cacik `^the meeting is at {time}$`
func MeetingAt(ctx *cacik.Context, t time.Time) {
	ctx.Logger().Info("meeting at", "time", t.Format("15:04:05"), "location", t.Location())
}

// TimeBetween checks time range with two {time} parameters
// @cacik `^the store is open between {time} and {time}$`
func TimeBetween(ctx *cacik.Context, openTime, closeTime time.Time) {
	ctx.Logger().Info("store open", "from", openTime.Format("15:04"), "to", closeTime.Format("15:04"))
}

// ================
// DATE STEPS
// ================

// EventOn uses built-in {date} type - parses to time.Time at midnight in Local timezone
// Supports: EU (DD/MM/YYYY), ISO (YYYY-MM-DD), written (15 Jan 2024)
// Examples: 15/01/2024, 2024-01-15, 15 Jan 2024, January 15, 2024
// @cacik `^the event is on {date}$`
func EventOn(ctx *cacik.Context, d time.Time) {
	ctx.Logger().Info("event on", "date", d.Format("2006-01-02"))
}

// DateRange checks date range with two {date} parameters
// @cacik `^the sale runs from {date} to {date}$`
func DateRange(ctx *cacik.Context, startDate, endDate time.Time) {
	ctx.Logger().Info("sale runs", "from", startDate.Format("2006-01-02"), "to", endDate.Format("2006-01-02"))
}

// ================
// DATETIME STEPS
// ================

// AppointmentAt uses built-in {datetime} type - parses to time.Time with full date, time, and optional timezone
// Supports: ISO with space/T separator, with optional timezone
// Examples: 2024-01-15 14:30:00, 2024-01-15T14:30:00Z, 15/01/2024 2:30pm Europe/London
// @cacik `^the appointment is at {datetime}$`
func AppointmentAt(ctx *cacik.Context, dt time.Time) {
	ctx.Logger().Info("appointment at", "datetime", dt.Format(time.RFC3339))
}

// FlightDeparts demonstrates datetime with timezone
// @cacik `^the flight departs at {datetime}$`
func FlightDeparts(ctx *cacik.Context, dt time.Time) {
	ctx.Logger().Info("flight departs", "datetime", dt.Format(time.RFC3339), "location", dt.Location())
}

// ScheduleRange combines {date} and {time} in one step
// @cacik `^schedule from {date} at {time} to {date} at {time}$`
func ScheduleRange(ctx *cacik.Context, startDate, startTime, endDate, endTime time.Time) {
	// Combine date and time
	start := time.Date(startDate.Year(), startDate.Month(), startDate.Day(),
		startTime.Hour(), startTime.Minute(), startTime.Second(), 0, startDate.Location())
	end := time.Date(endDate.Year(), endDate.Month(), endDate.Day(),
		endTime.Hour(), endTime.Minute(), endTime.Second(), 0, endDate.Location())
	ctx.Logger().Info("schedule", "from", start.Format(time.RFC3339), "to", end.Format(time.RFC3339))
}

// ================
// TIMEZONE STEPS
// ================

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

// ConvertDatetimeToTimezone converts a datetime to a different timezone
// @cacik `^convert {datetime} to {timezone}$`
func ConvertDatetimeToTimezone(ctx *cacik.Context, dt time.Time, loc *time.Location) {
	converted := dt.In(loc)
	ctx.Logger().Info("converted datetime", "result", converted.Format(time.RFC3339))
}

// ================
// COMBINED STEPS
// ================

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
	// Apply the timezone to the time
	meeting := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, loc)
	ctx.Logger().Info("meeting", "time", meeting.Format("15:04:05"), "timezone", loc.String())
}
