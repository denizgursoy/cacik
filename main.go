package main

import (
	testdata "github.com/denizgursoy/cacik/internal/comment_parser/testdata"
	stepbool "github.com/denizgursoy/cacik/internal/comment_parser/testdata/step-bool"
	stepbuiltin "github.com/denizgursoy/cacik/internal/comment_parser/testdata/step-builtin"
	stepcolor "github.com/denizgursoy/cacik/internal/comment_parser/testdata/step-color"
	stepdatetime "github.com/denizgursoy/cacik/internal/comment_parser/testdata/step-datetime"
	stepfloat "github.com/denizgursoy/cacik/internal/comment_parser/testdata/step-float"
	stepint "github.com/denizgursoy/cacik/internal/comment_parser/testdata/step-int"
	stepmixed "github.com/denizgursoy/cacik/internal/comment_parser/testdata/step-mixed"
	stepone "github.com/denizgursoy/cacik/internal/comment_parser/testdata/step-one"
	steppriority "github.com/denizgursoy/cacik/internal/comment_parser/testdata/step-priority"
	stepstring "github.com/denizgursoy/cacik/internal/comment_parser/testdata/step-string"
	steptwo "github.com/denizgursoy/cacik/internal/comment_parser/testdata/step-two"
	runner "github.com/denizgursoy/cacik/pkg/runner"
	"log"
)

func main() {
	err := runner.NewCucumberRunner().
		WithConfigFunc(testdata.Method1).
		RegisterCustomType("Priority", "int", map[string]string{
			"1":              "1",
			"2":              "2",
			"3":              "3",
			"high":           "3",
			"low":            "1",
			"medium":         "2",
			"priorityhigh":   "3",
			"prioritylow":    "1",
			"prioritymedium": "2",
		}).
		RegisterCustomType("Color", "string", map[string]string{
			"blue":  "blue",
			"green": "green",
			"red":   "red",
		}).
		RegisterCustomType("Size", "string", map[string]string{
			"large":      "large",
			"medium":     "medium",
			"sizelarge":  "large",
			"sizemedium": "medium",
			"sizesmall":  "small",
			"small":      "small",
		}).
		RegisterStep("^I select ((?i:blue|green|red))$", stepcolor.SelectColor).
		RegisterStep("^the color is ((?i:blue|green|red))$", stepcolor.ColorIs).
		RegisterStep("^I want a ((?i:blue|green|red)) (car|bike) with (-?\\d+) doors costing (-?\\d*\\.?\\d+) dollars$", stepmixed.WantColoredVehicle).
		RegisterStep("^a ((?i:blue|green|red)) item named \"([^\"]*)\" at ((?i:1|2|3|high|low|medium|priorityhigh|prioritylow|prioritymedium)) priority$", stepmixed.NamedItemWithPriority).
		RegisterStep("^((?i:blue|green|red)) owned by (\\w+) is (true|false|yes|no)$", stepmixed.OwnedByWithVisibility).
		RegisterStep("^I have (-?\\d+) ((?i:large|medium|sizelarge|sizemedium|sizesmall|small)) ((?i:blue|green|red)) boxes$", stepmixed.SizedItemCount).
		RegisterStep("^product (\\w+) is ((?i:blue|green|red)) and ((?i:large|medium|sizelarge|sizemedium|sizesmall|small)) priced at (-?\\d*\\.?\\d+) with ((?i:1|2|3|high|low|medium|priorityhigh|prioritylow|prioritymedium)) priority described as \"([^\"]*)\"$", stepmixed.ProductWithAllTypes).
		RegisterStep("^I ordered (-?\\d+) of (.*)$", stepmixed.QuantityWithAny).
		RegisterStep("^(enable|disable) the ((?i:blue|green|red)) (button|switch) and set active to (true|false)$", stepmixed.ConditionalAction).
		RegisterStep("^step 1$", stepone.Step1).
		RegisterStep("^the meeting is at (\\d{1,2}:\\d{2}(?::\\d{2})?(?:\\.\\d{1,3})?(?:\\s*[AaPp][Mm])?(?:\\s*(?:Z|UTC|[+-]\\d{2}:?\\d{2}|[A-Za-z_]+/[A-Za-z_]+))?)$", stepdatetime.MeetingAt).
		RegisterStep("^the store is open between (\\d{1,2}:\\d{2}(?::\\d{2})?(?:\\.\\d{1,3})?(?:\\s*[AaPp][Mm])?(?:\\s*(?:Z|UTC|[+-]\\d{2}:?\\d{2}|[A-Za-z_]+/[A-Za-z_]+))?) and (\\d{1,2}:\\d{2}(?::\\d{2})?(?:\\.\\d{1,3})?(?:\\s*[AaPp][Mm])?(?:\\s*(?:Z|UTC|[+-]\\d{2}:?\\d{2}|[A-Za-z_]+/[A-Za-z_]+))?)$", stepdatetime.TimeBetween).
		RegisterStep("^the event is on (\\d{4}[-/]\\d{2}[-/]\\d{2}|\\d{1,2}[-/\\.]\\d{1,2}[-/\\.]\\d{2,4}|(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\\.?\\s+\\d{1,2},?\\s+\\d{4}|\\d{1,2}\\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\\.?\\s+\\d{4})$", stepdatetime.EventOn).
		RegisterStep("^the sale runs from (\\d{4}[-/]\\d{2}[-/]\\d{2}|\\d{1,2}[-/\\.]\\d{1,2}[-/\\.]\\d{2,4}|(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\\.?\\s+\\d{1,2},?\\s+\\d{4}|\\d{1,2}\\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\\.?\\s+\\d{4}) to (\\d{4}[-/]\\d{2}[-/]\\d{2}|\\d{1,2}[-/\\.]\\d{1,2}[-/\\.]\\d{2,4}|(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\\.?\\s+\\d{1,2},?\\s+\\d{4}|\\d{1,2}\\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\\.?\\s+\\d{4})$", stepdatetime.DateRange).
		RegisterStep("^the appointment is at (\\d{4}[-/]\\d{2}[-/]\\d{2}[T\\s]\\d{1,2}:\\d{2}(?::\\d{2})?(?:\\.\\d{1,3})?(?:\\s*[AaPp][Mm])?(?:\\s*(?:Z|UTC|[+-]\\d{2}:?\\d{2}|[A-Za-z_]+/[A-Za-z_]+))?|\\d{1,2}[-/\\.]\\d{1,2}[-/\\.]\\d{2,4}\\s+\\d{1,2}:\\d{2}(?::\\d{2})?(?:\\s*[AaPp][Mm])?(?:\\s*(?:Z|UTC|[+-]\\d{2}:?\\d{2}|[A-Za-z_]+/[A-Za-z_]+))?)$", stepdatetime.AppointmentAt).
		RegisterStep("^the flight departs at (\\d{4}[-/]\\d{2}[-/]\\d{2}[T\\s]\\d{1,2}:\\d{2}(?::\\d{2})?(?:\\.\\d{1,3})?(?:\\s*[AaPp][Mm])?(?:\\s*(?:Z|UTC|[+-]\\d{2}:?\\d{2}|[A-Za-z_]+/[A-Za-z_]+))?|\\d{1,2}[-/\\.]\\d{1,2}[-/\\.]\\d{2,4}\\s+\\d{1,2}:\\d{2}(?::\\d{2})?(?:\\s*[AaPp][Mm])?(?:\\s*(?:Z|UTC|[+-]\\d{2}:?\\d{2}|[A-Za-z_]+/[A-Za-z_]+))?)$", stepdatetime.FlightDeparts).
		RegisterStep("^schedule from (\\d{4}[-/]\\d{2}[-/]\\d{2}|\\d{1,2}[-/\\.]\\d{1,2}[-/\\.]\\d{2,4}|(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\\.?\\s+\\d{1,2},?\\s+\\d{4}|\\d{1,2}\\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\\.?\\s+\\d{4}) at (\\d{1,2}:\\d{2}(?::\\d{2})?(?:\\.\\d{1,3})?(?:\\s*[AaPp][Mm])?(?:\\s*(?:Z|UTC|[+-]\\d{2}:?\\d{2}|[A-Za-z_]+/[A-Za-z_]+))?) to (\\d{4}[-/]\\d{2}[-/]\\d{2}|\\d{1,2}[-/\\.]\\d{1,2}[-/\\.]\\d{2,4}|(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\\.?\\s+\\d{1,2},?\\s+\\d{4}|\\d{1,2}\\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\\.?\\s+\\d{4}) at (\\d{1,2}:\\d{2}(?::\\d{2})?(?:\\.\\d{1,3})?(?:\\s*[AaPp][Mm])?(?:\\s*(?:Z|UTC|[+-]\\d{2}:?\\d{2}|[A-Za-z_]+/[A-Za-z_]+))?)$", stepdatetime.ScheduleRange).
		RegisterStep("^convert to (Z|UTC|[+-]\\d{2}:?\\d{2}|[A-Za-z_]+/[A-Za-z_]+)$", stepdatetime.ConvertToTimezone).
		RegisterStep("^show current time in (Z|UTC|[+-]\\d{2}:?\\d{2}|[A-Za-z_]+/[A-Za-z_]+)$", stepdatetime.ShowTimeIn).
		RegisterStep("^convert (\\d{4}[-/]\\d{2}[-/]\\d{2}[T\\s]\\d{1,2}:\\d{2}(?::\\d{2})?(?:\\.\\d{1,3})?(?:\\s*[AaPp][Mm])?(?:\\s*(?:Z|UTC|[+-]\\d{2}:?\\d{2}|[A-Za-z_]+/[A-Za-z_]+))?|\\d{1,2}[-/\\.]\\d{1,2}[-/\\.]\\d{2,4}\\s+\\d{1,2}:\\d{2}(?::\\d{2})?(?:\\s*[AaPp][Mm])?(?:\\s*(?:Z|UTC|[+-]\\d{2}:?\\d{2}|[A-Za-z_]+/[A-Za-z_]+))?) to (Z|UTC|[+-]\\d{2}:?\\d{2}|[A-Za-z_]+/[A-Za-z_]+)$", stepdatetime.ConvertDatetimeToTimezone).
		RegisterStep("^I have (-?\\d+) tasks due on (\\d{4}[-/]\\d{2}[-/]\\d{2}|\\d{1,2}[-/\\.]\\d{1,2}[-/\\.]\\d{2,4}|(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\\.?\\s+\\d{1,2},?\\s+\\d{4}|\\d{1,2}\\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\\.?\\s+\\d{4}) at (\\d{1,2}:\\d{2}(?::\\d{2})?(?:\\.\\d{1,3})?(?:\\s*[AaPp][Mm])?(?:\\s*(?:Z|UTC|[+-]\\d{2}:?\\d{2}|[A-Za-z_]+/[A-Za-z_]+))?)$", stepdatetime.DeadlineWithCount).
		RegisterStep("^event \"([^\"]*)\" starts at (\\d{4}[-/]\\d{2}[-/]\\d{2}[T\\s]\\d{1,2}:\\d{2}(?::\\d{2})?(?:\\.\\d{1,3})?(?:\\s*[AaPp][Mm])?(?:\\s*(?:Z|UTC|[+-]\\d{2}:?\\d{2}|[A-Za-z_]+/[A-Za-z_]+))?|\\d{1,2}[-/\\.]\\d{1,2}[-/\\.]\\d{2,4}\\s+\\d{1,2}:\\d{2}(?::\\d{2})?(?:\\s*[AaPp][Mm])?(?:\\s*(?:Z|UTC|[+-]\\d{2}:?\\d{2}|[A-Za-z_]+/[A-Za-z_]+))?)$", stepdatetime.EventAtDateTime).
		RegisterStep("^meeting at (\\d{1,2}:\\d{2}(?::\\d{2})?(?:\\.\\d{1,3})?(?:\\s*[AaPp][Mm])?(?:\\s*(?:Z|UTC|[+-]\\d{2}:?\\d{2}|[A-Za-z_]+/[A-Za-z_]+))?) in (Z|UTC|[+-]\\d{2}:?\\d{2}|[A-Za-z_]+/[A-Za-z_]+)$", stepdatetime.MeetingInTimezone).
		RegisterStep("^I have (\\d+) apples$", stepint.IGetApples).
		RegisterStep("^the user says \"([^\"]*)\"$", stepstring.UserSays).
		RegisterStep("^the error message is \"([^\"]*)\"$", stepstring.ErrorMessageIs).
		RegisterStep("^the title is (\\w+)$", stepstring.TitleIs).
		RegisterStep("^I have (-?\\d+) apples$", stepbuiltin.HaveApples).
		RegisterStep("^the price is (-?\\d*\\.?\\d+)$", stepbuiltin.PriceIs).
		RegisterStep("^my name is (\\w+)$", stepbuiltin.NameIs).
		RegisterStep("^I say \"([^\"]*)\"$", stepbuiltin.Say).
		RegisterStep("^I see (.*)$", stepbuiltin.SeeAnything).
		RegisterStep("^the item costs (-?\\d*\\.?\\d+) dollars$", stepfloat.ItemCosts).
		RegisterStep("^the temperature is (-?\\d*\\.?\\d+) degrees$", stepfloat.TemperatureIs).
		RegisterStep("^step 2$", steptwo.Step2).
		RegisterStep("^it is (true|false|yes|no|on|off|enabled|disabled)$", stepbool.ItIs).
		RegisterStep("^the feature is (enabled|disabled)$", stepbool.FeatureToggle).
		RegisterStep("^priority is ((?i:1|2|3|high|low|medium|priorityhigh|prioritylow|prioritymedium))$", steppriority.SetPriority).
		RegisterStep("^the priority is ((?i:1|2|3|high|low|medium|priorityhigh|prioritylow|prioritymedium))$", steppriority.PriorityIs).
		RunWithTags()

	if err != nil {
		log.Fatal(err)
	}
}
