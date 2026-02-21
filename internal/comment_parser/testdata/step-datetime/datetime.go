package step_datetime

import (
	"context"
	"fmt"
	"time"
)

// ================
// TIME STEPS
// ================

// MeetingAt uses built-in {time} type - parses to time.Time with zero date (0001-01-01)
// Supports: HH:MM, HH:MM:SS, HH:MM:SS.mmm, with optional AM/PM and timezone
// Examples: 14:30, 2:30pm, 14:30:45, 14:30+05:30, 2:30pm Europe/London
// @cacik `^the meeting is at {time}$`
func MeetingAt(ctx context.Context, t time.Time) (context.Context, error) {
	fmt.Printf("Meeting at: %s (Location: %s)\n", t.Format("15:04:05"), t.Location())
	return ctx, nil
}

// TimeBetween checks time range with two {time} parameters
// @cacik `^the store is open between {time} and {time}$`
func TimeBetween(ctx context.Context, openTime, closeTime time.Time) (context.Context, error) {
	fmt.Printf("Store open: %s to %s\n", openTime.Format("15:04"), closeTime.Format("15:04"))
	return ctx, nil
}

// ================
// DATE STEPS
// ================

// EventOn uses built-in {date} type - parses to time.Time at midnight in Local timezone
// Supports: EU (DD/MM/YYYY), ISO (YYYY-MM-DD), written (15 Jan 2024)
// Examples: 15/01/2024, 2024-01-15, 15 Jan 2024, January 15, 2024
// @cacik `^the event is on {date}$`
func EventOn(ctx context.Context, d time.Time) (context.Context, error) {
	fmt.Printf("Event on: %s\n", d.Format("2006-01-02"))
	return ctx, nil
}

// DateRange checks date range with two {date} parameters
// @cacik `^the sale runs from {date} to {date}$`
func DateRange(ctx context.Context, startDate, endDate time.Time) (context.Context, error) {
	fmt.Printf("Sale: %s to %s\n", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	return ctx, nil
}

// ================
// DATETIME STEPS
// ================

// AppointmentAt uses built-in {datetime} type - parses to time.Time with full date, time, and optional timezone
// Supports: ISO with space/T separator, with optional timezone
// Examples: 2024-01-15 14:30:00, 2024-01-15T14:30:00Z, 15/01/2024 2:30pm Europe/London
// @cacik `^the appointment is at {datetime}$`
func AppointmentAt(ctx context.Context, dt time.Time) (context.Context, error) {
	fmt.Printf("Appointment at: %s\n", dt.Format(time.RFC3339))
	return ctx, nil
}

// FlightDeparts demonstrates datetime with timezone
// @cacik `^the flight departs at {datetime}$`
func FlightDeparts(ctx context.Context, dt time.Time) (context.Context, error) {
	fmt.Printf("Flight departs: %s (Location: %s)\n", dt.Format(time.RFC3339), dt.Location())
	return ctx, nil
}

// ScheduleRange combines {date} and {time} in one step
// @cacik `^schedule from {date} at {time} to {date} at {time}$`
func ScheduleRange(ctx context.Context, startDate, startTime, endDate, endTime time.Time) (context.Context, error) {
	// Combine date and time
	start := time.Date(startDate.Year(), startDate.Month(), startDate.Day(),
		startTime.Hour(), startTime.Minute(), startTime.Second(), 0, startDate.Location())
	end := time.Date(endDate.Year(), endDate.Month(), endDate.Day(),
		endTime.Hour(), endTime.Minute(), endTime.Second(), 0, endDate.Location())
	fmt.Printf("Schedule: %s to %s\n", start.Format(time.RFC3339), end.Format(time.RFC3339))
	return ctx, nil
}

// ================
// TIMEZONE STEPS
// ================

// ConvertToTimezone uses standalone {timezone} type - parses to *time.Location
// Supports: IANA names (Europe/London), UTC, Z, offsets (+05:30, -08:00)
// @cacik `^convert to {timezone}$`
func ConvertToTimezone(ctx context.Context, loc *time.Location) (context.Context, error) {
	fmt.Printf("Timezone: %s\n", loc.String())
	return ctx, nil
}

// ShowTimeIn shows current time in a specific timezone
// @cacik `^show current time in {timezone}$`
func ShowTimeIn(ctx context.Context, loc *time.Location) (context.Context, error) {
	now := time.Now().In(loc)
	fmt.Printf("Current time in %s: %s\n", loc.String(), now.Format("15:04:05"))
	return ctx, nil
}

// ConvertDatetimeToTimezone converts a datetime to a different timezone
// @cacik `^convert {datetime} to {timezone}$`
func ConvertDatetimeToTimezone(ctx context.Context, dt time.Time, loc *time.Location) (context.Context, error) {
	converted := dt.In(loc)
	fmt.Printf("Converted: %s\n", converted.Format(time.RFC3339))
	return ctx, nil
}

// ================
// COMBINED STEPS
// ================

// DeadlineWithCount combines {int}, {date}, and {time}
// @cacik `^I have {int} tasks due on {date} at {time}$`
func DeadlineWithCount(ctx context.Context, count int, date, t time.Time) (context.Context, error) {
	deadline := time.Date(date.Year(), date.Month(), date.Day(),
		t.Hour(), t.Minute(), t.Second(), 0, date.Location())
	fmt.Printf("%d tasks due: %s\n", count, deadline.Format(time.RFC3339))
	return ctx, nil
}

// EventAtDateTime combines {string} and {datetime}
// @cacik `^event {string} starts at {datetime}$`
func EventAtDateTime(ctx context.Context, name string, dt time.Time) (context.Context, error) {
	fmt.Printf("Event %q starts at %s\n", name, dt.Format(time.RFC3339))
	return ctx, nil
}

// MeetingInTimezone combines {time} with explicit {timezone}
// @cacik `^meeting at {time} in {timezone}$`
func MeetingInTimezone(ctx context.Context, t time.Time, loc *time.Location) (context.Context, error) {
	// Apply the timezone to the time
	meeting := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, loc)
	fmt.Printf("Meeting at %s in %s\n", meeting.Format("15:04:05"), loc.String())
	return ctx, nil
}
