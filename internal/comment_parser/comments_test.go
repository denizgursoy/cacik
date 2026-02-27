package comment_parser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	messages "github.com/cucumber/messages/go/v21"
	"github.com/denizgursoy/cacik/internal/generator"
	"github.com/denizgursoy/cacik/pkg/gherkin_parser"
	"github.com/stretchr/testify/require"
)

// ─── helpers ────────────────────────────────────────────────────────────────

// testdataDir returns the absolute path to internal/comment_parser/testdata.
func testdataDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Join(dir, "testdata")
}

// parseDir parses a single testdata subdirectory and returns the Output.
func parseDir(t *testing.T, dir string) *generator.Output {
	t.Helper()
	parser := NewGoSourceFileParser()
	out, err := parser.ParseFunctionCommentsOfGoFilesInDirectoryRecursively(context.Background(), dir)
	require.NoError(t, err)
	return out
}

// buildStepMap builds funcName→regexPattern from parsed output.
func buildStepMap(out *generator.Output) map[string]string {
	m := make(map[string]string)
	for _, s := range out.StepFunctions {
		m[s.FunctionName] = s.StepName
	}
	return m
}

// parseFeatureFile finds and parses the .feature file in dir.
func parseFeatureFile(t *testing.T, dir string) *messages.GherkinDocument {
	t.Helper()
	files, err := gherkin_parser.SearchFeatureFilesIn([]string{dir})
	require.NoError(t, err)
	require.NotEmpty(t, files, "no .feature file found in %s", dir)

	f, err := os.Open(files[0])
	require.NoError(t, err)
	defer f.Close()

	doc, err := gherkin_parser.ParseGherkinFile(f)
	require.NoError(t, err)
	return doc
}

// collectStepTexts extracts every step text from a GherkinDocument,
// walking Feature-level backgrounds, scenarios, and rules (with their backgrounds).
func collectStepTexts(doc *messages.GherkinDocument) []string {
	var texts []string
	if doc.Feature == nil {
		return texts
	}
	for _, child := range doc.Feature.Children {
		if child.Background != nil {
			for _, s := range child.Background.Steps {
				texts = append(texts, s.Text)
			}
		}
		if child.Scenario != nil {
			for _, s := range child.Scenario.Steps {
				texts = append(texts, s.Text)
			}
		}
		if child.Rule != nil {
			for _, rc := range child.Rule.Children {
				if rc.Background != nil {
					for _, s := range rc.Background.Steps {
						texts = append(texts, s.Text)
					}
				}
				if rc.Scenario != nil {
					for _, s := range rc.Scenario.Steps {
						texts = append(texts, s.Text)
					}
				}
			}
		}
	}
	return texts
}

// compiledPatterns compiles every regex in stepMap and returns funcName→*regexp.Regexp.
func compiledPatterns(t *testing.T, stepMap map[string]string) map[string]*regexp.Regexp {
	t.Helper()
	m := make(map[string]*regexp.Regexp, len(stepMap))
	for fn, pat := range stepMap {
		re, err := regexp.Compile(pat)
		require.NoError(t, err, "failed to compile pattern for %s: %s", fn, pat)
		m[fn] = re
	}
	return m
}

// matchStep finds the first pattern in compiled that matches stepText.
// Returns funcName, the compiled regex, and the submatch slice.
// Fails the test if no pattern matches.
func matchStep(t *testing.T, stepText string, compiled map[string]*regexp.Regexp) (string, *regexp.Regexp, []string) {
	t.Helper()
	for fn, re := range compiled {
		if m := re.FindStringSubmatch(stepText); m != nil {
			return fn, re, m
		}
	}
	t.Fatalf("no pattern matched step text: %q", stepText)
	return "", nil, nil
}

// assertAllStepsMatch verifies that every step text in the feature file
// matches exactly one pattern and that every capture group is non-empty.
func assertAllStepsMatch(t *testing.T, dir string) map[string][][]string {
	t.Helper()
	out := parseDir(t, dir)
	stepMap := buildStepMap(out)
	compiled := compiledPatterns(t, stepMap)
	doc := parseFeatureFile(t, dir)
	texts := collectStepTexts(doc)
	require.NotEmpty(t, texts, "feature file has no steps")

	// funcName → list of captured-group slices (one per matched step)
	captures := make(map[string][][]string)
	for _, text := range texts {
		fn, _, m := matchStep(t, text, compiled)
		groups := m[1:] // strip full-match
		captures[fn] = append(captures[fn], groups)
	}
	return captures
}

// ─── per-directory tests ────────────────────────────────────────────────────

func TestStepInt(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "step-int")
	out := parseDir(t, dir)
	stepMap := buildStepMap(out)

	require.Equal(t, `^I have (\d+) apples$`, stepMap["IGetApples"])

	// Verify IsExported is populated on step functions
	require.Len(t, out.StepFunctions, 1)
	require.True(t, out.StepFunctions[0].IsExported, "step function IGetApples should be marked as exported")

	caps := assertAllStepsMatch(t, dir)
	// "I have 5 apples" → ["5"], "I have 10 apples" → ["10"], etc.
	require.Contains(t, caps["IGetApples"], []string{"5"})
	require.Contains(t, caps["IGetApples"], []string{"10"})
	require.Contains(t, caps["IGetApples"], []string{"100"})
	require.Contains(t, caps["IGetApples"], []string{"1000000"})
}

func TestStepBool(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "step-bool")
	out := parseDir(t, dir)
	stepMap := buildStepMap(out)

	require.Equal(t, "^it is (?i)(true|false|yes|no|on|off|enabled|disabled|1|0|t|f)$", stepMap["ItIs"])
	require.Equal(t, "^the feature is (?i)(true|false|yes|no|on|off|enabled|disabled|1|0|t|f)$", stepMap["FeatureToggle"])

	caps := assertAllStepsMatch(t, dir)
	require.Contains(t, caps["ItIs"], []string{"true"})
	require.Contains(t, caps["ItIs"], []string{"false"})
	require.Contains(t, caps["ItIs"], []string{"yes"})
	require.Contains(t, caps["ItIs"], []string{"no"})
	require.Contains(t, caps["ItIs"], []string{"on"})
	require.Contains(t, caps["ItIs"], []string{"off"})
	require.Contains(t, caps["ItIs"], []string{"enabled"})
	require.Contains(t, caps["ItIs"], []string{"disabled"})
	// Case-insensitive
	require.Contains(t, caps["ItIs"], []string{"TRUE"})
	require.Contains(t, caps["ItIs"], []string{"FALSE"})
	require.Contains(t, caps["ItIs"], []string{"Yes"})
	require.Contains(t, caps["ItIs"], []string{"NO"})
	require.Contains(t, caps["FeatureToggle"], []string{"enabled"})
	require.Contains(t, caps["FeatureToggle"], []string{"disabled"})
}

func TestStepWord(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "step-word")
	out := parseDir(t, dir)
	stepMap := buildStepMap(out)

	require.Equal(t, `^my name is (\w+)$`, stepMap["NameIs"])
	require.Equal(t, `^the status is (\w+)$`, stepMap["StatusIs"])

	caps := assertAllStepsMatch(t, dir)
	require.Contains(t, caps["NameIs"], []string{"John"})
	require.Contains(t, caps["NameIs"], []string{"test123"})
	require.Contains(t, caps["NameIs"], []string{"Alice"})
	require.Contains(t, caps["StatusIs"], []string{"active"})
	require.Contains(t, caps["StatusIs"], []string{"pending"})
	require.Contains(t, caps["StatusIs"], []string{"DONE"})
}

func TestStepAny(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "step-any")
	out := parseDir(t, dir)
	stepMap := buildStepMap(out)

	require.Equal(t, `^I see (.*)$`, stepMap["SeeAnything"])
	require.Equal(t, `^the description is (.*)$`, stepMap["DescriptionIs"])

	caps := assertAllStepsMatch(t, dir)
	require.Contains(t, caps["SeeAnything"], []string{"anything at all here"})
	require.Contains(t, caps["SeeAnything"], []string{"123 mixed content!"})
	require.Contains(t, caps["SeeAnything"], []string{"special chars: @#$% and more"})
	require.Contains(t, caps["DescriptionIs"], []string{"a long text with spaces and punctuation!"})
	require.Contains(t, caps["DescriptionIs"], []string{"42"})
}

func TestStepFloat(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "step-float")
	out := parseDir(t, dir)
	stepMap := buildStepMap(out)

	require.Equal(t, `^the item costs (-?\d*\.?\d+) dollars$`, stepMap["ItemCosts"])
	require.Equal(t, `^the temperature is (-?\d*\.?\d+) degrees$`, stepMap["TemperatureIs"])

	caps := assertAllStepsMatch(t, dir)
	require.Contains(t, caps["ItemCosts"], []string{"19.99"})
	require.Contains(t, caps["ItemCosts"], []string{"100.00"})
	require.Contains(t, caps["ItemCosts"], []string{"50"})
	require.Contains(t, caps["TemperatureIs"], []string{"-5.5"})
	require.Contains(t, caps["TemperatureIs"], []string{"-0.1"})
	require.Contains(t, caps["TemperatureIs"], []string{"0"})
}

func TestStepString(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "step-string")
	out := parseDir(t, dir)
	stepMap := buildStepMap(out)

	require.Equal(t, `^the user says "([^"]*)"$`, stepMap["UserSays"])
	require.Equal(t, `^the error message is "([^"]*)"$`, stepMap["ErrorMessageIs"])

	caps := assertAllStepsMatch(t, dir)
	require.Contains(t, caps["UserSays"], []string{"Hello World"})
	require.Contains(t, caps["UserSays"], []string{"Testing 123!"})
	require.Contains(t, caps["UserSays"], []string{"Special chars: @#$%"})
	require.Contains(t, caps["UserSays"], []string{""})
	require.Contains(t, caps["ErrorMessageIs"], []string{"File not found"})
	require.Contains(t, caps["ErrorMessageIs"], []string{"Connection timeout"})
}

func TestStepColor(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "step-color")
	out := parseDir(t, dir)
	stepMap := buildStepMap(out)

	// Custom type patterns contain the color values case-insensitively
	require.Contains(t, stepMap["SelectColor"], "blue")
	require.Contains(t, stepMap["SelectColor"], "green")
	require.Contains(t, stepMap["SelectColor"], "red")

	// Custom type parsing
	colorType, ok := out.CustomTypes["color"]
	require.True(t, ok, "Color type should be found")
	require.Equal(t, "Color", colorType.Name)
	require.Equal(t, "string", colorType.Underlying)
	require.Equal(t, "red", colorType.Values["Red"])
	require.Equal(t, "blue", colorType.Values["Blue"])
	require.Equal(t, "green", colorType.Values["Green"])

	caps := assertAllStepsMatch(t, dir)
	// Verify captured values
	require.NotEmpty(t, caps["SelectColor"])
	require.NotEmpty(t, caps["ColorIs"])
	// "I select red" → captures "red"
	for _, groups := range caps["SelectColor"] {
		require.Len(t, groups, 1)
		require.Contains(t, []string{"red", "blue", "green", "RED"}, groups[0])
	}
}

func TestStepPriority(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "step-priority")
	out := parseDir(t, dir)
	stepMap := buildStepMap(out)

	// Custom type patterns include both names and values
	require.Contains(t, stepMap["SetPriority"], "low")
	require.Contains(t, stepMap["SetPriority"], "medium")
	require.Contains(t, stepMap["SetPriority"], "high")
	require.Contains(t, stepMap["SetPriority"], "1")
	require.Contains(t, stepMap["SetPriority"], "2")
	require.Contains(t, stepMap["SetPriority"], "3")

	// Custom type parsing
	priorityType, ok := out.CustomTypes["priority"]
	require.True(t, ok, "Priority type should be found")
	require.Equal(t, "Priority", priorityType.Name)
	require.Equal(t, "int", priorityType.Underlying)
	require.Equal(t, "1", priorityType.Values["Low"])
	require.Equal(t, "2", priorityType.Values["Medium"])
	require.Equal(t, "3", priorityType.Values["High"])

	caps := assertAllStepsMatch(t, dir)
	require.NotEmpty(t, caps["SetPriority"])
	require.NotEmpty(t, caps["PriorityIs"])
}

func TestStepTime(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "step-time")
	out := parseDir(t, dir)
	stepMap := buildStepMap(out)

	// Both use {time}
	require.Len(t, stepMap, 2)
	require.Contains(t, stepMap["MeetingAt"], `\d{1,2}:\d{2}`)
	require.Contains(t, stepMap["TimeBetween"], `\d{1,2}:\d{2}`)

	caps := assertAllStepsMatch(t, dir)

	// 24-hour format
	require.Contains(t, caps["MeetingAt"], []string{"14:30"})
	require.Contains(t, caps["MeetingAt"], []string{"09:15"})
	require.Contains(t, caps["MeetingAt"], []string{"00:00"})
	// With seconds
	require.Contains(t, caps["MeetingAt"], []string{"14:30:45"})
	// With AM/PM
	require.Contains(t, caps["MeetingAt"], []string{"2:30pm"})
	require.Contains(t, caps["MeetingAt"], []string{"9:15am"})
	// With timezone
	require.Contains(t, caps["MeetingAt"], []string{"14:30Z"})
	require.Contains(t, caps["MeetingAt"], []string{"14:30 Europe/London"})
	require.Contains(t, caps["MeetingAt"], []string{"10:00am UTC"})
	// Time range
	require.Contains(t, caps["TimeBetween"], []string{"9:00", "21:00"})
	require.Contains(t, caps["TimeBetween"], []string{"9:00am", "9:00pm"})
}

func TestStepDate(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "step-date")
	out := parseDir(t, dir)
	stepMap := buildStepMap(out)

	// Both use {date}
	require.Len(t, stepMap, 2)
	require.Contains(t, stepMap["EventOn"], `\d{4}[-/]\d{2}[-/]\d{2}`)
	require.Contains(t, stepMap["DateRange"], `\d{4}[-/]\d{2}[-/]\d{2}`)

	caps := assertAllStepsMatch(t, dir)

	// EU format
	require.Contains(t, caps["EventOn"], []string{"15/01/2024"})
	require.Contains(t, caps["EventOn"], []string{"31/12/2024"})
	// ISO format
	require.Contains(t, caps["EventOn"], []string{"2024-01-15"})
	require.Contains(t, caps["EventOn"], []string{"2024-12-31"})
	// Written format
	require.Contains(t, caps["EventOn"], []string{"15 Jan 2024"})
	require.Contains(t, caps["EventOn"], []string{"15 January 2024"})
	require.Contains(t, caps["EventOn"], []string{"Jan 15, 2024"})
	// Date range
	require.Contains(t, caps["DateRange"], []string{"2024-01-01", "2024-12-31"})
	require.Contains(t, caps["DateRange"], []string{"01/01/2024", "31/12/2024"})
	require.Contains(t, caps["DateRange"], []string{"1 Jan 2024", "31 Dec 2024"})
}

func TestStepTimezone(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "step-timezone")
	out := parseDir(t, dir)
	stepMap := buildStepMap(out)

	// Both use {timezone}
	require.Len(t, stepMap, 2)
	require.Contains(t, stepMap["ConvertToTimezone"], "Z")
	require.Contains(t, stepMap["ConvertToTimezone"], "UTC")
	require.Contains(t, stepMap["ShowTimeIn"], "UTC")

	caps := assertAllStepsMatch(t, dir)

	// Z and UTC
	require.Contains(t, caps["ConvertToTimezone"], []string{"UTC"})
	require.Contains(t, caps["ConvertToTimezone"], []string{"Z"})
	// Offsets
	require.Contains(t, caps["ConvertToTimezone"], []string{"+05:30"})
	require.Contains(t, caps["ConvertToTimezone"], []string{"-08:00"})
	// IANA names
	require.Contains(t, caps["ConvertToTimezone"], []string{"Europe/London"})
	require.Contains(t, caps["ConvertToTimezone"], []string{"America/New_York"})
	require.Contains(t, caps["ConvertToTimezone"], []string{"Asia/Tokyo"})
	// ShowTimeIn
	require.Contains(t, caps["ShowTimeIn"], []string{"UTC"})
	require.Contains(t, caps["ShowTimeIn"], []string{"Europe/London"})
	require.Contains(t, caps["ShowTimeIn"], []string{"Asia/Tokyo"})
}

func TestStepDatetime(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "step-datetime")
	out := parseDir(t, dir)
	stepMap := buildStepMap(out)

	// Only AppointmentAt and FlightDeparts remain — both use {datetime}
	require.Len(t, stepMap, 2)
	require.Contains(t, stepMap["AppointmentAt"], `\d{4}[-/]\d{2}[-/]\d{2}`)
	require.Contains(t, stepMap["AppointmentAt"], `\d{1,2}:\d{2}`)
	require.Contains(t, stepMap["FlightDeparts"], `\d{4}[-/]\d{2}[-/]\d{2}`)
	require.Contains(t, stepMap["FlightDeparts"], `\d{1,2}:\d{2}`)

	// All steps must match a pattern
	caps := assertAllStepsMatch(t, dir)

	// Spot-check captured values — ISO with space
	require.Contains(t, caps["AppointmentAt"], []string{"2024-01-15 14:30"})
	require.Contains(t, caps["AppointmentAt"], []string{"2024-12-31 23:59:59"})
	// ISO with T separator
	require.Contains(t, caps["AppointmentAt"], []string{"2024-01-15T14:30"})
	// With AM/PM
	require.Contains(t, caps["AppointmentAt"], []string{"2024-01-15 2:30pm"})
	// FlightDeparts with timezone
	require.Contains(t, caps["FlightDeparts"], []string{"2024-01-15T14:30:00Z"})
	require.Contains(t, caps["FlightDeparts"], []string{"2024-01-15T14:30:00+05:30"})
	require.Contains(t, caps["FlightDeparts"], []string{"2024-01-15 14:30 Europe/London"})
}

func TestStepMixed(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "step-mixed")
	out := parseDir(t, dir)
	stepMap := buildStepMap(out)

	// Verify patterns contain expected fragments
	require.Contains(t, stepMap["WantColoredVehicle"], "(car|bike)")
	require.Contains(t, stepMap["WantColoredVehicle"], "(?i:")
	require.Contains(t, stepMap["WantColoredVehicle"], `(-?\d+)`)
	require.Contains(t, stepMap["WantColoredVehicle"], `(-?\d*\.?\d+)`)
	require.Contains(t, stepMap["NamedItemWithPriority"], "(?i:")
	require.Contains(t, stepMap["NamedItemWithPriority"], `"([^"]*)"`)
	require.Contains(t, stepMap["SizedItemCount"], `(-?\d+)`)
	require.Contains(t, stepMap["SizedItemCount"], "(?i:")

	// Datetime combo patterns
	require.Contains(t, stepMap["ScheduleRange"], `\d{1,2}:\d{2}`)              // {time}
	require.Contains(t, stepMap["ScheduleRange"], `\d{4}[-/]\d{2}[-/]`)         // {date}
	require.Contains(t, stepMap["DeadlineWithCount"], `(-?\d+)`)                // {int}
	require.Contains(t, stepMap["DeadlineWithCount"], `\d{4}[-/]\d{2}`)         // {date}
	require.Contains(t, stepMap["DeadlineWithCount"], `\d{1,2}:\d{2}`)          // {time}
	require.Contains(t, stepMap["EventAtDateTime"], `"([^"]*)"`)                // {string}
	require.Contains(t, stepMap["EventAtDateTime"], `\d{4}[-/]\d{2}`)           // {datetime}
	require.Contains(t, stepMap["MeetingInTimezone"], `\d{1,2}:\d{2}`)          // {time}
	require.Contains(t, stepMap["MeetingInTimezone"], "UTC")                    // {timezone}
	require.Contains(t, stepMap["ConvertDatetimeToTimezone"], `\d{4}[-/]\d{2}`) // {datetime}
	require.Contains(t, stepMap["ConvertDatetimeToTimezone"], "UTC")            // {timezone}

	// Custom types parsed
	require.Contains(t, out.CustomTypes, "color")
	require.Contains(t, out.CustomTypes, "priority")
	require.Contains(t, out.CustomTypes, "size")

	// All steps match
	caps := assertAllStepsMatch(t, dir)

	// "I want a red car with 4 doors costing 25000.50 dollars"
	// → captures: [red, car, 4, 25000.50]
	require.Contains(t, caps["WantColoredVehicle"], []string{"red", "car", "4", "25000.50"})
	require.Contains(t, caps["WantColoredVehicle"], []string{"Green", "car", "2", "15000"})

	// "I ordered 3 of red apples and some oranges"
	require.Contains(t, caps["QuantityWithAny"], []string{"3", "red apples and some oranges"})
	require.Contains(t, caps["QuantityWithAny"], []string{"100", "random stuff here"})

	// "I have 5 small red boxes"
	require.Contains(t, caps["SizedItemCount"], []string{"5", "small", "red"})

	// Schedule with date and time: "schedule from 2024-01-15 at 9:00 to 2024-01-15 at 17:00"
	// → captures: [date, time, date, time]
	require.Contains(t, caps["ScheduleRange"], []string{"2024-01-15", "9:00", "2024-01-15", "17:00"})

	// Tasks with count, date, and time: "I have 5 tasks due on 2024-01-15 at 17:00"
	require.Contains(t, caps["DeadlineWithCount"], []string{"5", "2024-01-15", "17:00"})

	// Event with name and datetime: "event "Team Meeting" starts at 2024-01-15 10:00"
	require.Contains(t, caps["EventAtDateTime"], []string{"Team Meeting", "2024-01-15 10:00"})

	// Meeting with time and timezone: "meeting at 14:30 in Europe/London"
	require.Contains(t, caps["MeetingInTimezone"], []string{"14:30", "Europe/London"})
	require.Contains(t, caps["MeetingInTimezone"], []string{"18:00", "Asia/Tokyo"})

	// Convert datetime to timezone: "convert 2024-01-15T14:30:00Z to Europe/London"
	require.Contains(t, caps["ConvertDatetimeToTimezone"], []string{"2024-01-15T14:30:00Z", "Europe/London"})
	require.Contains(t, caps["ConvertDatetimeToTimezone"], []string{"2024-01-15T14:30:00Z", "America/New_York"})
}

// ─── new built-in type tests ────────────────────────────────────────────────

func TestStepUUID(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "step-uuid")
	out := parseDir(t, dir)
	stepMap := buildStepMap(out)

	require.Equal(t,
		`^the identifier is ([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})$`,
		stepMap["MatchUUID"])

	caps := assertAllStepsMatch(t, dir)
	require.Contains(t, caps["MatchUUID"], []string{"550e8400-e29b-41d4-a716-446655440000"})
	require.Contains(t, caps["MatchUUID"], []string{"6BA7B810-9DAD-11D1-80B4-00C04FD430C8"})
}

func TestStepIP(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "step-ip")
	out := parseDir(t, dir)
	stepMap := buildStepMap(out)

	require.Equal(t,
		`^the server is at ([0-9a-fA-F.:]+(?:%25[a-zA-Z0-9]+)?)$`,
		stepMap["MatchIP"])

	caps := assertAllStepsMatch(t, dir)
	require.Contains(t, caps["MatchIP"], []string{"192.168.1.1"})
	require.Contains(t, caps["MatchIP"], []string{"::1"})
	require.Contains(t, caps["MatchIP"], []string{"2001:db8::1"})
}

func TestStepHex(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "step-hex")
	out := parseDir(t, dir)
	stepMap := buildStepMap(out)

	require.Equal(t, `^the color code is (0[xX][0-9a-fA-F]+)$`, stepMap["MatchHex"])

	caps := assertAllStepsMatch(t, dir)
	require.Contains(t, caps["MatchHex"], []string{"0xFF"})
	require.Contains(t, caps["MatchHex"], []string{"0x1A2B"})
	require.Contains(t, caps["MatchHex"], []string{"0XDEADBEEF"})
}

func TestStepPath(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "step-path")
	out := parseDir(t, dir)
	stepMap := buildStepMap(out)

	require.Equal(t, `^the file is at ([./~\\][^\s]*)$`, stepMap["MatchPath"])

	caps := assertAllStepsMatch(t, dir)
	require.Contains(t, caps["MatchPath"], []string{"/usr/local/bin"})
	require.Contains(t, caps["MatchPath"], []string{"./config.yaml"})
	require.Contains(t, caps["MatchPath"], []string{"../parent/file.txt"})
}

func TestStepSemver(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "step-semver")
	out := parseDir(t, dir)
	stepMap := buildStepMap(out)

	require.Equal(t,
		`^the version is (\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?)$`,
		stepMap["MatchSemver"])

	caps := assertAllStepsMatch(t, dir)
	require.Contains(t, caps["MatchSemver"], []string{"1.0.0"})
	require.Contains(t, caps["MatchSemver"], []string{"2.1.3-beta"})
	require.Contains(t, caps["MatchSemver"], []string{"1.0.0-alpha.1+build.123"})
}

func TestStepBase64(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "step-base64")
	out := parseDir(t, dir)
	stepMap := buildStepMap(out)

	require.Equal(t, `^the encoded data is ([A-Za-z0-9+/]{4,}={0,2})$`, stepMap["MatchBase64"])

	caps := assertAllStepsMatch(t, dir)
	require.Contains(t, caps["MatchBase64"], []string{"SGVsbG8="})
	require.Contains(t, caps["MatchBase64"], []string{"SGVsbG8gV29ybGQ="})
	require.Contains(t, caps["MatchBase64"], []string{"dGVzdA=="})
}

func TestStepCSV(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "step-csv")
	out := parseDir(t, dir)
	stepMap := buildStepMap(out)

	require.Equal(t, `^the items are ([^,\s]+(?:,[^,\s]+)+)$`, stepMap["MatchCSV"])

	caps := assertAllStepsMatch(t, dir)
	require.Contains(t, caps["MatchCSV"], []string{"a,b,c"})
	require.Contains(t, caps["MatchCSV"], []string{"1,2,3"})
	require.Contains(t, caps["MatchCSV"], []string{"foo,bar,baz"})
}

func TestStepJSON(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "step-json")
	out := parseDir(t, dir)
	stepMap := buildStepMap(out)

	require.Equal(t, `^the payload is (\{[^}]*\}|\[[^\]]*\])$`, stepMap["MatchJSON"])

	caps := assertAllStepsMatch(t, dir)
	require.Contains(t, caps["MatchJSON"], []string{`{"key":"value"}`})
	require.Contains(t, caps["MatchJSON"], []string{"[1,2,3]"})
}

func TestStepPhone(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "step-phone")
	out := parseDir(t, dir)
	stepMap := buildStepMap(out)

	require.Equal(t, `^the contact number is (\+?[\d\s().-]{7,20})$`, stepMap["MatchPhone"])

	caps := assertAllStepsMatch(t, dir)
	require.Contains(t, caps["MatchPhone"], []string{"+1-555-123-4567"})
	require.Contains(t, caps["MatchPhone"], []string{"+44 20 7946 0958"})
	require.Contains(t, caps["MatchPhone"], []string{"555-123-4567"})
}

func TestStepPercent(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "step-percent")
	out := parseDir(t, dir)
	stepMap := buildStepMap(out)

	require.Equal(t, `^the discount is (-?\d*\.?\d+%)$`, stepMap["MatchPercent"])

	caps := assertAllStepsMatch(t, dir)
	require.Contains(t, caps["MatchPercent"], []string{"50%"})
	require.Contains(t, caps["MatchPercent"], []string{"99.9%"})
	require.Contains(t, caps["MatchPercent"], []string{"-10%"})
}

func TestStepBigint(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "step-bigint")
	out := parseDir(t, dir)
	stepMap := buildStepMap(out)

	require.Equal(t, `^the large number is (-?\d+)$`, stepMap["MatchBigint"])

	caps := assertAllStepsMatch(t, dir)
	require.Contains(t, caps["MatchBigint"], []string{"12345678901234567890"})
	require.Contains(t, caps["MatchBigint"], []string{"-99999999999999999999"})
}

func TestStepRegex(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "step-regex")
	out := parseDir(t, dir)
	stepMap := buildStepMap(out)

	require.Equal(t, `^the pattern is (/[^/]+/)$`, stepMap["MatchRegex"])

	caps := assertAllStepsMatch(t, dir)
	require.Contains(t, caps["MatchRegex"], []string{`/^hello.*$/`})
	require.Contains(t, caps["MatchRegex"], []string{`/\d+/`})
	require.Contains(t, caps["MatchRegex"], []string{"/[a-z]+/"})
}

// ─── root testdata (config + hooks) ─────────────────────────────────────────

func TestStepConfig(t *testing.T) {
	dir := filepath.Join(testdataDir(t), "step-config")
	out := parseDir(t, dir)

	require.Len(t, out.ConfigFunctions, 1)
	require.Equal(t, "MyConfig", out.ConfigFunctions[0].FunctionName)
	require.True(t, out.ConfigFunctions[0].IsExported, "config function should be marked as exported")
	require.Len(t, out.HooksFunctions, 1)
	require.Equal(t, "MyHooks", out.HooksFunctions[0].FunctionName)
	require.True(t, out.HooksFunctions[0].IsExported, "hooks function should be marked as exported")

	// No step functions expected in this directory
	require.Empty(t, out.StepFunctions)
}

// ─── duplicate detection ────────────────────────────────────────────────────

func TestDuplicateStepDetection(t *testing.T) {
	t.Run("returns error for duplicate step patterns", func(t *testing.T) {
		dir := filepath.Join(testdataDir(t), "step-duplicate")

		parser := NewGoSourceFileParser()
		_, err := parser.ParseFunctionCommentsOfGoFilesInDirectoryRecursively(
			context.Background(), dir,
		)

		require.NotNil(t, err)
		require.Contains(t, err.Error(), "duplicate step pattern")
		require.Contains(t, err.Error(), "I have")
		require.Contains(t, err.Error(), "items")
		require.Contains(t, err.Error(), "FirstDuplicateStep")
		require.Contains(t, err.Error(), "SecondDuplicateStep")
	})
}

// ─── discovery test ─────────────────────────────────────────────────────────

// TestAllDirectoriesHaveTests verifies that every step-* directory under
// testdata has a corresponding test. If someone adds a new step-* directory
// but forgets to add a test, this will catch it.
func TestAllDirectoriesHaveTests(t *testing.T) {
	base := testdataDir(t)
	entries, err := os.ReadDir(base)
	require.NoError(t, err)

	// Known step-* directories and their test functions
	testedDirs := map[string]string{
		"step-int":       "TestStepInt",
		"step-bool":      "TestStepBool",
		"step-word":      "TestStepWord",
		"step-any":       "TestStepAny",
		"step-float":     "TestStepFloat",
		"step-string":    "TestStepString",
		"step-color":     "TestStepColor",
		"step-priority":  "TestStepPriority",
		"step-time":      "TestStepTime",
		"step-date":      "TestStepDate",
		"step-timezone":  "TestStepTimezone",
		"step-datetime":  "TestStepDatetime",
		"step-mixed":     "TestStepMixed",
		"step-uuid":      "TestStepUUID",
		"step-ip":        "TestStepIP",
		"step-hex":       "TestStepHex",
		"step-path":      "TestStepPath",
		"step-semver":    "TestStepSemver",
		"step-base64":    "TestStepBase64",
		"step-csv":       "TestStepCSV",
		"step-json":      "TestStepJSON",
		"step-phone":     "TestStepPhone",
		"step-percent":   "TestStepPercent",
		"step-bigint":    "TestStepBigint",
		"step-regex":     "TestStepRegex",
		"step-duplicate": "TestDuplicateStepDetection",
		"step-config":    "TestStepConfig",
	}

	var missing []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "step-") {
			continue
		}
		if _, ok := testedDirs[name]; !ok {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		t.Errorf("step-* directories without tests: %s\nAdd a test function for each and register it in testedDirs.", strings.Join(missing, ", "))
	}
}

// ─── unit tests (kept as-is) ────────────────────────────────────────────────

func TestTransformStepPattern(t *testing.T) {
	t.Run("transforms {color} to regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{
			"color": {
				Name:       "Color",
				Underlying: "string",
				Values:     map[string]string{"Red": "red", "Blue": "blue"},
			},
		}

		result, err := transformStepPattern("^I select {color}$", customTypes)
		require.Nil(t, err)
		require.Contains(t, result, "blue")
		require.Contains(t, result, "red")
		require.Contains(t, result, "(")
		require.Contains(t, result, ")")
	})

	t.Run("returns error for unknown type", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		_, err := transformStepPattern("^I select {unknown}$", customTypes)
		require.NotNil(t, err)
		require.Contains(t, err.Error(), "unknown parameter type")
	})

	t.Run("returns error for type with no constants", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{
			"empty": {
				Name:       "Empty",
				Underlying: "string",
				Values:     map[string]string{},
			},
		}

		_, err := transformStepPattern("^I select {empty}$", customTypes)
		require.NotNil(t, err)
		require.Contains(t, err.Error(), "no defined constants")
	})

	t.Run("handles multiple custom types in pattern", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{
			"color": {
				Name:       "Color",
				Underlying: "string",
				Values:     map[string]string{"Red": "red"},
			},
			"size": {
				Name:       "Size",
				Underlying: "string",
				Values:     map[string]string{"Large": "large"},
			},
		}

		result, err := transformStepPattern("^I want {color} and {size}$", customTypes)
		require.Nil(t, err)
		require.Contains(t, result, "red")
		require.Contains(t, result, "large")
	})

	// Built-in parameter type tests
	t.Run("transforms {int} to regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^I have {int} apples$", customTypes)
		require.Nil(t, err)
		require.Equal(t, `^I have (-?\d+) apples$`, result)
	})

	t.Run("transforms {float} to regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^the price is {float}$", customTypes)
		require.Nil(t, err)
		require.Equal(t, `^the price is (-?\d*\.?\d+)$`, result)
	})

	t.Run("transforms {word} to regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^my name is {word}$", customTypes)
		require.Nil(t, err)
		require.Equal(t, `^my name is (\w+)$`, result)
	})

	t.Run("transforms {string} to regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^I say {string}$", customTypes)
		require.Nil(t, err)
		require.Equal(t, `^I say "([^"]*)"$`, result)
	})

	t.Run("transforms {} (empty) to regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^I have {} items$", customTypes)
		require.Nil(t, err)
		require.Equal(t, `^I have (.*) items$`, result)
	})

	t.Run("transforms {any} to regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^I see {any}$", customTypes)
		require.Nil(t, err)
		require.Equal(t, `^I see (.*)$`, result)
	})

	t.Run("transforms {time} to regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^the meeting is at {time}$", customTypes)
		require.Nil(t, err)
		require.Contains(t, result, `\d{1,2}:\d{2}`)
	})

	t.Run("transforms {date} to regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^the event is on {date}$", customTypes)
		require.Nil(t, err)
		require.Contains(t, result, `\d{4}[-/]\d{2}[-/]\d{2}`)
	})

	t.Run("transforms {datetime} to regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^the appointment is at {datetime}$", customTypes)
		require.Nil(t, err)
		require.Contains(t, result, `\d{4}[-/]\d{2}[-/]\d{2}`)
		require.Contains(t, result, `\d{1,2}:\d{2}`)
	})

	t.Run("transforms {timezone} to regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^convert to {timezone}$", customTypes)
		require.Nil(t, err)
		// Should contain patterns for Z, UTC, offset, and IANA names
		require.Contains(t, result, "Z")
		require.Contains(t, result, "UTC")
		require.Contains(t, result, `[+-]\d{2}`)
		require.Contains(t, result, `[A-Za-z_]+/[A-Za-z_]+`)
	})

	t.Run("time pattern includes optional timezone", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^meeting at {time}$", customTypes)
		require.Nil(t, err)
		// Should contain timezone patterns as optional
		require.Contains(t, result, "Z|UTC")
		require.Contains(t, result, `[A-Za-z_]+/[A-Za-z_]+`)
	})

	t.Run("datetime pattern includes optional timezone", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^appointment at {datetime}$", customTypes)
		require.Nil(t, err)
		// Should contain timezone patterns as optional
		require.Contains(t, result, "Z|UTC")
		require.Contains(t, result, `[A-Za-z_]+/[A-Za-z_]+`)
	})

	t.Run("transforms {email} to regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^user {email} logged in$", customTypes)
		require.Nil(t, err)
		require.Equal(t, `^user ([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}) logged in$`, result)
	})

	t.Run("transforms {duration} to regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^wait for {duration}$", customTypes)
		require.Nil(t, err)
		require.Equal(t, `^wait for (-?(?:\d+\.?\d*(?:ns|us|µs|ms|s|m|h))+)$`, result)
	})

	t.Run("transforms {url} to regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{}

		result, err := transformStepPattern("^navigate to {url}$", customTypes)
		require.Nil(t, err)
		require.Equal(t, `^navigate to (https?://[^\s]+)$`, result)
	})

	t.Run("handles mixed built-in and custom types", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{
			"color": {
				Name:       "Color",
				Underlying: "string",
				Values:     map[string]string{"Red": "red"},
			},
		}

		result, err := transformStepPattern("^I have {int} {color} items$", customTypes)
		require.Nil(t, err)
		require.Contains(t, result, `(-?\d+)`)
		require.Contains(t, result, "red")
	})

	t.Run("handles complex pattern with custom type, built-in types, and regex", func(t *testing.T) {
		customTypes := map[string]*generator.CustomType{
			"color": {
				Name:       "Color",
				Underlying: "string",
				Values:     map[string]string{"Red": "red", "Blue": "blue", "Green": "green"},
			},
			"priority": {
				Name:       "Priority",
				Underlying: "int",
				Values:     map[string]string{"Low": "1", "Medium": "2", "High": "3"},
			},
		}

		// Pattern: custom type + word + int + float + string + another custom type
		result, err := transformStepPattern(
			"^I want a {color} (car|bike) with {int} doors costing {float} dollars named {string} at {priority} priority$",
			customTypes,
		)
		require.Nil(t, err)

		// Verify custom type {color} is transformed with case-insensitive matching
		require.Contains(t, result, "(?i:")
		require.Contains(t, result, "red")
		require.Contains(t, result, "blue")
		require.Contains(t, result, "green")

		// Verify normal regex (car|bike) is preserved
		require.Contains(t, result, "(car|bike)")

		// Verify built-in {int} is transformed
		require.Contains(t, result, `(-?\d+)`)

		// Verify built-in {float} is transformed
		require.Contains(t, result, `(-?\d*\.?\d+)`)

		// Verify built-in {string} is transformed
		require.Contains(t, result, `"([^"]*)"`)

		// Verify custom type {priority} is transformed
		require.Contains(t, result, "low")
		require.Contains(t, result, "medium")
		require.Contains(t, result, "high")
		require.Contains(t, result, "1")
		require.Contains(t, result, "2")
		require.Contains(t, result, "3")
	})
}

// ─── unused import guard ────────────────────────────────────────────────────

var _ = fmt.Sprintf // ensure fmt is used
