package cacik

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGenerateHTMLReport(t *testing.T) {
	t.Run("creates file with valid HTML", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "report.html")

		startedAt := time.Date(2026, 2, 28, 14, 30, 0, 0, time.UTC)
		result := RunResult{
			Summary: ReporterSummary{
				ScenariosTotal:  2,
				ScenariosPassed: 1,
				ScenariosFailed: 1,
				StepsTotal:      5,
				StepsPassed:     3,
				StepsFailed:     1,
				StepsSkipped:    1,
			},
			StartedAt: startedAt,
			Scenarios: []ScenarioResult{
				{
					FeatureName: "Login",
					Name:        "Successful login",
					Tags:        []string{"@smoke"},
					Passed:      true,
					Duration:    150 * time.Millisecond,
					StartedAt:   time.Now(),
					Steps: []StepResult{
						{Keyword: "Given ", Text: "a registered user", Status: StepPassed, Duration: 50 * time.Millisecond},
						{Keyword: "When ", Text: "they enter valid credentials", Status: StepPassed, Duration: 80 * time.Millisecond},
						{Keyword: "Then ", Text: "they see the dashboard", Status: StepPassed, Duration: 20 * time.Millisecond},
					},
				},
				{
					FeatureName: "Login",
					Name:        "Failed login",
					Passed:      false,
					Error:       "expected 200 got 401",
					Duration:    200 * time.Millisecond,
					StartedAt:   time.Now(),
					Steps: []StepResult{
						{Keyword: "Given ", Text: "a registered user", Status: StepPassed, Duration: 50 * time.Millisecond},
						{Keyword: "When ", Text: "they enter wrong password", Status: StepFailed, Error: "expected 200 got 401", Duration: 100 * time.Millisecond},
						{Keyword: "Then ", Text: "they see an error", Status: StepSkipped},
					},
				},
			},
		}

		err := GenerateHTMLReport(path, result)
		require.NoError(t, err)

		data, err := os.ReadFile(path)
		require.NoError(t, err)

		html := string(data)
		require.True(t, strings.HasPrefix(html, "<!DOCTYPE html>"), "should start with doctype")
		require.Contains(t, html, "Test Execution Report")
		require.Contains(t, html, "Successful login")
		require.Contains(t, html, "Failed login")
		require.Contains(t, html, "a registered user")
		require.Contains(t, html, "expected 200 got 401")
		require.Contains(t, html, "@smoke")
		require.Contains(t, html, "Login")
		require.Contains(t, html, "Executed at 2026-02-28 14:30:00")
		// Has failures → red summary panel
		require.Contains(t, html, `summary has-failures`)
	})

	t.Run("creates parent directories", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "nested", "deep", "report.html")

		result := RunResult{
			Summary: ReporterSummary{ScenariosTotal: 0},
		}

		err := GenerateHTMLReport(path, result)
		require.NoError(t, err)

		_, err = os.Stat(path)
		require.NoError(t, err, "report file should exist")
	})

	t.Run("handles empty results", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "empty.html")

		result := RunResult{}

		err := GenerateHTMLReport(path, result)
		require.NoError(t, err)

		data, err := os.ReadFile(path)
		require.NoError(t, err)

		html := string(data)
		require.Contains(t, html, "Test Execution Report")
		require.Contains(t, html, "0") // zero scenarios
		// No failures → green summary panel
		require.Contains(t, html, `summary all-passed`)
	})

	t.Run("includes rule name when present", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "report.html")

		result := RunResult{
			Summary: ReporterSummary{ScenariosTotal: 1, ScenariosPassed: 1},
			Scenarios: []ScenarioResult{
				{
					FeatureName: "Cart",
					RuleName:    "Pricing",
					Name:        "Discount applied",
					Passed:      true,
					Duration:    10 * time.Millisecond,
				},
			},
		}

		err := GenerateHTMLReport(path, result)
		require.NoError(t, err)

		data, err := os.ReadFile(path)
		require.NoError(t, err)

		html := string(data)
		require.Contains(t, html, "Pricing")
	})

	t.Run("formats durations correctly", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "report.html")

		result := RunResult{
			Summary:  ReporterSummary{ScenariosTotal: 1, ScenariosPassed: 1, StepsTotal: 3, StepsPassed: 3},
			Duration: 3 * time.Second,
			Scenarios: []ScenarioResult{
				{
					FeatureName: "Timing",
					Name:        "Various durations",
					Passed:      true,
					Duration:    2500 * time.Millisecond,
					Steps: []StepResult{
						{Keyword: "Given ", Text: "fast step", Status: StepPassed, Duration: 500 * time.Microsecond},
						{Keyword: "When ", Text: "medium step", Status: StepPassed, Duration: 150 * time.Millisecond},
						{Keyword: "Then ", Text: "slow step", Status: StepPassed, Duration: 2 * time.Second},
					},
				},
			},
		}

		err := GenerateHTMLReport(path, result)
		require.NoError(t, err)

		data, err := os.ReadFile(path)
		require.NoError(t, err)

		html := string(data)
		// Microsecond formatting
		require.Contains(t, html, "500µs")
		// Millisecond formatting
		require.Contains(t, html, "150ms")
		// Second formatting
		require.Contains(t, html, "2.00s")
		// Total duration in summary header
		require.Contains(t, html, "3.00s")
		require.Contains(t, html, "Duration")
	})

	t.Run("returns error for invalid path", func(t *testing.T) {
		// /dev/null/impossible is not writable
		err := GenerateHTMLReport("/dev/null/impossible/report.html", RunResult{})
		require.Error(t, err)
	})

	t.Run("groups scenarios by status with failed first", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "report.html")

		result := RunResult{
			Summary: ReporterSummary{ScenariosTotal: 3, ScenariosPassed: 2, ScenariosFailed: 1},
			Scenarios: []ScenarioResult{
				{FeatureName: "F1", Name: "Passing A", Passed: true, Duration: 100 * time.Millisecond, Tags: []string{"@smoke"}},
				{FeatureName: "F1", Name: "Failing B", Passed: false, Duration: 250 * time.Millisecond, Tags: []string{"@smoke"}},
				{FeatureName: "F1", Name: "Passing C", Passed: true, Duration: 50 * time.Millisecond},
			},
		}

		err := GenerateHTMLReport(path, result)
		require.NoError(t, err)

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		html := string(data)

		// Failed section should appear before passed section
		failedIdx := strings.Index(html, "Failed Scenarios")
		passedIdx := strings.Index(html, "Passed Scenarios")
		require.Greater(t, failedIdx, 0, "should contain Failed Scenarios section")
		require.Greater(t, passedIdx, 0, "should contain Passed Scenarios section")
		require.Less(t, failedIdx, passedIdx, "Failed section should come before Passed section")

		// Section headers should include count and duration
		require.Contains(t, html, "1 scenarios, 250ms")
		require.Contains(t, html, "2 scenarios, 150ms")
	})

	t.Run("groups scenarios by tags within status sections", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "report.html")

		result := RunResult{
			Summary: ReporterSummary{ScenariosTotal: 3, ScenariosPassed: 3},
			Scenarios: []ScenarioResult{
				{FeatureName: "F", Name: "S1", Passed: true, Tags: []string{"@api"}, Duration: 120 * time.Millisecond},
				{FeatureName: "F", Name: "S2", Passed: true, Tags: []string{"@api"}, Duration: 180 * time.Millisecond},
				{FeatureName: "F", Name: "S3", Passed: true, Duration: 60 * time.Millisecond},
			},
		}

		err := GenerateHTMLReport(path, result)
		require.NoError(t, err)

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		html := string(data)

		// Should contain tag group labels
		require.Contains(t, html, "@api")
		require.Contains(t, html, "Untagged")

		// Untagged should appear after @api
		apiIdx := strings.Index(html, "tag-group-label\"><span class=\"tag-icon\">#</span> @api")
		untaggedIdx := strings.Index(html, "tag-group-label\"><span class=\"tag-icon\">#</span> Untagged")
		require.Greater(t, apiIdx, 0)
		require.Greater(t, untaggedIdx, 0)
		require.Less(t, apiIdx, untaggedIdx, "Untagged should appear after tagged groups")

		// Tag group labels should include count and duration inline
		require.Contains(t, html, "(2 scenarios, 300ms)", "tag group @api should show count and summed duration")
		require.Contains(t, html, "(1 scenarios, 60ms)", "tag group Untagged should show count and duration")
	})

	t.Run("shows green summary panel when all tests pass", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "report.html")

		result := RunResult{
			Summary: ReporterSummary{ScenariosTotal: 2, ScenariosPassed: 2, ScenariosFailed: 0},
			Scenarios: []ScenarioResult{
				{FeatureName: "F", Name: "S1", Passed: true, Duration: 50 * time.Millisecond},
				{FeatureName: "F", Name: "S2", Passed: true, Duration: 80 * time.Millisecond},
			},
		}

		err := GenerateHTMLReport(path, result)
		require.NoError(t, err)

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		html := string(data)

		require.Contains(t, html, `summary all-passed`)
		require.NotContains(t, html, `summary has-failures`)
	})

	t.Run("shows red summary panel when one scenario fails", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "report.html")

		result := RunResult{
			Summary: ReporterSummary{ScenariosTotal: 2, ScenariosPassed: 1, ScenariosFailed: 1},
			Scenarios: []ScenarioResult{
				{FeatureName: "F", Name: "S1", Passed: true, Duration: 50 * time.Millisecond},
				{FeatureName: "F", Name: "S2", Passed: false, Duration: 80 * time.Millisecond},
			},
		}

		err := GenerateHTMLReport(path, result)
		require.NoError(t, err)

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		html := string(data)

		require.Contains(t, html, `summary has-failures`)
		require.NotContains(t, html, `summary all-passed`)
	})

	t.Run("shows red summary panel when multiple scenarios fail", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "report.html")

		result := RunResult{
			Summary: ReporterSummary{ScenariosTotal: 3, ScenariosPassed: 0, ScenariosFailed: 3},
			Scenarios: []ScenarioResult{
				{FeatureName: "F", Name: "S1", Passed: false, Duration: 50 * time.Millisecond},
				{FeatureName: "F", Name: "S2", Passed: false, Duration: 80 * time.Millisecond},
				{FeatureName: "F", Name: "S3", Passed: false, Duration: 30 * time.Millisecond},
			},
		}

		err := GenerateHTMLReport(path, result)
		require.NoError(t, err)

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		html := string(data)

		require.Contains(t, html, `summary has-failures`)
	})
}

func TestTagKey(t *testing.T) {
	t.Run("returns Untagged for empty tags", func(t *testing.T) {
		require.Equal(t, "Untagged", tagKey(nil))
		require.Equal(t, "Untagged", tagKey([]string{}))
	})

	t.Run("returns single tag as-is", func(t *testing.T) {
		require.Equal(t, "@smoke", tagKey([]string{"@smoke"}))
	})

	t.Run("sorts multiple tags alphabetically", func(t *testing.T) {
		require.Equal(t, "@api, @smoke", tagKey([]string{"@smoke", "@api"}))
	})

	t.Run("does not mutate input slice", func(t *testing.T) {
		tags := []string{"@z", "@a", "@m"}
		_ = tagKey(tags)
		require.Equal(t, []string{"@z", "@a", "@m"}, tags)
	})
}

func TestGroupByTags(t *testing.T) {
	t.Run("groups scenarios by tag set with count and duration", func(t *testing.T) {
		scenarios := []ScenarioResult{
			{Name: "S1", Tags: []string{"@smoke"}, Duration: 100 * time.Millisecond},
			{Name: "S2", Tags: []string{"@smoke"}, Duration: 200 * time.Millisecond},
			{Name: "S3", Tags: []string{"@api"}, Duration: 50 * time.Millisecond},
		}

		groups := groupByTags(scenarios)
		require.Len(t, groups, 2)
		require.Equal(t, "@api", groups[0].TagLabel)
		require.Equal(t, 1, groups[0].Count)
		require.Equal(t, 50*time.Millisecond, groups[0].Duration)
		require.Len(t, groups[0].Scenarios, 1)
		require.Equal(t, "@smoke", groups[1].TagLabel)
		require.Equal(t, 2, groups[1].Count)
		require.Equal(t, 300*time.Millisecond, groups[1].Duration)
		require.Len(t, groups[1].Scenarios, 2)
	})

	t.Run("puts untagged last", func(t *testing.T) {
		scenarios := []ScenarioResult{
			{Name: "S1"},
			{Name: "S2", Tags: []string{"@smoke"}},
			{Name: "S3"},
		}

		groups := groupByTags(scenarios)
		require.Len(t, groups, 2)
		require.Equal(t, "@smoke", groups[0].TagLabel)
		require.Equal(t, "Untagged", groups[1].TagLabel)
		require.Equal(t, 2, groups[1].Count)
		require.Len(t, groups[1].Scenarios, 2)
	})

	t.Run("handles all untagged", func(t *testing.T) {
		scenarios := []ScenarioResult{
			{Name: "S1"},
			{Name: "S2"},
		}

		groups := groupByTags(scenarios)
		require.Len(t, groups, 1)
		require.Equal(t, "Untagged", groups[0].TagLabel)
		require.Equal(t, 2, groups[0].Count)
		require.Len(t, groups[0].Scenarios, 2)
	})

	t.Run("multi-tag scenarios grouped together", func(t *testing.T) {
		scenarios := []ScenarioResult{
			{Name: "S1", Tags: []string{"@smoke", "@api"}},
			{Name: "S2", Tags: []string{"@api", "@smoke"}}, // same tags, different order
			{Name: "S3", Tags: []string{"@smoke"}},
		}

		groups := groupByTags(scenarios)
		require.Len(t, groups, 2)
		// "@api, @smoke" comes before "@smoke" alphabetically
		require.Equal(t, "@api, @smoke", groups[0].TagLabel)
		require.Equal(t, 2, groups[0].Count)
		require.Len(t, groups[0].Scenarios, 2)
		require.Equal(t, "@smoke", groups[1].TagLabel)
		require.Equal(t, 1, groups[1].Count)
		require.Len(t, groups[1].Scenarios, 1)
	})
}

func TestColorizeStepText(t *testing.T) {
	t.Run("no match locs returns single span", func(t *testing.T) {
		step := StepResult{Text: "a registered user", Status: StepPassed}
		got := string(colorizeStepText(step))
		require.Equal(t, `<span class="step-text passed">a registered user</span>`, got)
	})

	t.Run("no match locs with failed status", func(t *testing.T) {
		step := StepResult{Text: "it should fail", Status: StepFailed}
		got := string(colorizeStepText(step))
		require.Equal(t, `<span class="step-text failed">it should fail</span>`, got)
	})

	t.Run("no match locs with skipped status", func(t *testing.T) {
		step := StepResult{Text: "it should be skipped", Status: StepSkipped}
		got := string(colorizeStepText(step))
		require.Equal(t, `<span class="step-text skipped">it should be skipped</span>`, got)
	})

	t.Run("single capture group", func(t *testing.T) {
		// Text: "the price is 42 dollars"
		// Capture group on "42" at positions [13, 15]
		step := StepResult{
			Text:      "the price is 42 dollars",
			Status:    StepPassed,
			MatchLocs: []int{13, 15},
		}
		got := string(colorizeStepText(step))
		require.Contains(t, got, `<span class="step-text passed">the price is </span>`)
		require.Contains(t, got, `<span class="step-param" style="color:#5C92FF">42</span>`)
		require.Contains(t, got, `<span class="step-text passed"> dollars</span>`)
	})

	t.Run("two capture groups use different colors", func(t *testing.T) {
		// Text: "user alice has 5 items"
		// Capture group 0: "alice" at [5, 10]
		// Capture group 1: "5" at [15, 16]
		step := StepResult{
			Text:      "user alice has 5 items",
			Status:    StepPassed,
			MatchLocs: []int{5, 10, 15, 16},
		}
		got := string(colorizeStepText(step))
		require.Contains(t, got, `style="color:#5C92FF">alice</span>`)
		require.Contains(t, got, `style="color:#00CED1">5</span>`)
	})

	t.Run("capture group at start of text", func(t *testing.T) {
		// Text: "42 is the answer"
		// Capture group on "42" at [0, 2]
		step := StepResult{
			Text:      "42 is the answer",
			Status:    StepPassed,
			MatchLocs: []int{0, 2},
		}
		got := string(colorizeStepText(step))
		require.True(t, strings.HasPrefix(got, `<span class="step-param" style="color:#5C92FF">42</span>`))
		require.Contains(t, got, `<span class="step-text passed"> is the answer</span>`)
	})

	t.Run("capture group at end of text", func(t *testing.T) {
		// Text: "the answer is 42"
		// Capture group on "42" at [14, 16]
		step := StepResult{
			Text:      "the answer is 42",
			Status:    StepPassed,
			MatchLocs: []int{14, 16},
		}
		got := string(colorizeStepText(step))
		require.Contains(t, got, `<span class="step-text passed">the answer is </span>`)
		require.True(t, strings.HasSuffix(got, `<span class="step-param" style="color:#5C92FF">42</span>`))
	})

	t.Run("skipped step params use skipped color", func(t *testing.T) {
		step := StepResult{
			Text:      "the price is 42 dollars",
			Status:    StepSkipped,
			MatchLocs: []int{13, 15},
		}
		got := string(colorizeStepText(step))
		// Params should NOT have step-param class, they should use skipped
		require.NotContains(t, got, "step-param")
		require.Contains(t, got, `<span class="step-text skipped">the price is </span>`)
		require.Contains(t, got, `<span class="step-text skipped">42</span>`)
		require.Contains(t, got, `<span class="step-text skipped"> dollars</span>`)
	})

	t.Run("wraps color index for more than 10 params", func(t *testing.T) {
		// Build text with 11 capture groups so index 10 wraps to color 0
		// Text: "a b c d e f g h i j k"
		// 11 single-char capture groups
		step := StepResult{
			Text:   "a b c d e f g h i j k",
			Status: StepPassed,
			MatchLocs: []int{
				0, 1, // a -> color 0
				2, 3, // b -> color 1
				4, 5, // c -> color 2
				6, 7, // d -> color 3
				8, 9, // e -> color 4
				10, 11, // f -> color 5
				12, 13, // g -> color 6
				14, 15, // h -> color 7
				16, 17, // i -> color 8
				18, 19, // j -> color 9
				20, 21, // k -> color 0 (wraps)
			},
		}
		got := string(colorizeStepText(step))
		// First param (a) should use color 0 (#5C92FF)
		require.Contains(t, got, `style="color:#5C92FF">a</span>`)
		// 11th param (k) should also use color 0 (#5C92FF) due to wrapping
		require.Contains(t, got, `style="color:#5C92FF">k</span>`)
		// 10th param (j) should use color 9 (#A993D6)
		require.Contains(t, got, `style="color:#A993D6">j</span>`)
	})

	t.Run("HTML-escapes special characters", func(t *testing.T) {
		step := StepResult{
			Text:      `the value is <b>"42"</b>`,
			Status:    StepPassed,
			MatchLocs: []int{13, 24}, // <b>"42"</b>
		}
		got := string(colorizeStepText(step))
		require.Contains(t, got, `&lt;b&gt;&#34;42&#34;&lt;/b&gt;`)
		require.NotContains(t, got, "<b>")
	})

	t.Run("non-participating group is skipped", func(t *testing.T) {
		// A group with -1,-1 means it didn't participate in the match
		step := StepResult{
			Text:      "the price is 42 dollars",
			Status:    StepPassed,
			MatchLocs: []int{-1, -1, 13, 15},
		}
		got := string(colorizeStepText(step))
		// The -1,-1 group should be skipped, "42" gets the first participating color
		require.Contains(t, got, `style="color:#5C92FF">42</span>`)
	})

	t.Run("empty match locs treated same as nil", func(t *testing.T) {
		step := StepResult{Text: "hello world", Status: StepPassed, MatchLocs: []int{}}
		got := string(colorizeStepText(step))
		require.Equal(t, `<span class="step-text passed">hello world</span>`, got)
	})

	t.Run("integration with HTML report", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "report.html")

		result := RunResult{
			Summary: ReporterSummary{ScenariosTotal: 1, ScenariosPassed: 1, StepsTotal: 1, StepsPassed: 1},
			Scenarios: []ScenarioResult{
				{
					FeatureName: "Pricing",
					Name:        "Item price",
					Passed:      true,
					Duration:    50 * time.Millisecond,
					Steps: []StepResult{
						{
							Keyword:   "Given ",
							Text:      "the price is 42 dollars",
							Status:    StepPassed,
							Duration:  50 * time.Millisecond,
							MatchLocs: []int{13, 15},
						},
					},
				},
			},
		}

		err := GenerateHTMLReport(path, result)
		require.NoError(t, err)

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		html := string(data)

		// Should contain the param color span
		require.Contains(t, html, `style="color:#5C92FF">42</span>`)
		// Should contain normal text spans
		require.Contains(t, html, `the price is `)
		require.Contains(t, html, ` dollars`)
	})
}

func TestBuildReportData(t *testing.T) {
	t.Run("separates failed and passed into sections", func(t *testing.T) {
		result := RunResult{
			Scenarios: []ScenarioResult{
				{Name: "Pass1", Passed: true, Duration: 100 * time.Millisecond},
				{Name: "Fail1", Passed: false, Duration: 200 * time.Millisecond},
				{Name: "Pass2", Passed: true, Duration: 300 * time.Millisecond},
			},
		}

		data := buildReportData(result)
		require.Len(t, data.Sections, 2)
		require.Equal(t, "Failed Scenarios", data.Sections[0].Label)
		require.Equal(t, "failed", data.Sections[0].CSSClass)
		require.Equal(t, 1, data.Sections[0].Count)
		require.Equal(t, 200*time.Millisecond, data.Sections[0].Duration)
		require.Equal(t, "Passed Scenarios", data.Sections[1].Label)
		require.Equal(t, "passed", data.Sections[1].CSSClass)
		require.Equal(t, 2, data.Sections[1].Count)
		require.Equal(t, 400*time.Millisecond, data.Sections[1].Duration)
	})

	t.Run("omits empty sections", func(t *testing.T) {
		allPassed := RunResult{
			Scenarios: []ScenarioResult{
				{Name: "Pass1", Passed: true},
			},
		}
		data := buildReportData(allPassed)
		require.Len(t, data.Sections, 1)
		require.Equal(t, "Passed Scenarios", data.Sections[0].Label)

		allFailed := RunResult{
			Scenarios: []ScenarioResult{
				{Name: "Fail1", Passed: false},
			},
		}
		data = buildReportData(allFailed)
		require.Len(t, data.Sections, 1)
		require.Equal(t, "Failed Scenarios", data.Sections[0].Label)
	})

	t.Run("empty scenarios produces no sections", func(t *testing.T) {
		data := buildReportData(RunResult{})
		require.Len(t, data.Sections, 0)
	})

	t.Run("preserves summary, total duration, and executed at", func(t *testing.T) {
		startedAt := time.Date(2026, 2, 28, 14, 30, 0, 0, time.UTC)
		result := RunResult{
			Summary:   ReporterSummary{ScenariosTotal: 42, StepsTotal: 100},
			Duration:  5 * time.Second,
			StartedAt: startedAt,
		}
		data := buildReportData(result)
		require.Equal(t, 42, data.Summary.ScenariosTotal)
		require.Equal(t, 100, data.Summary.StepsTotal)
		require.Equal(t, 5*time.Second, data.TotalDuration)
		require.Equal(t, startedAt, data.ExecutedAt)
	})

	t.Run("failed section has tag sub-groups", func(t *testing.T) {
		result := RunResult{
			Scenarios: []ScenarioResult{
				{Name: "Fail1", Passed: false, Tags: []string{"@api"}},
				{Name: "Fail2", Passed: false, Tags: []string{"@api"}},
				{Name: "Fail3", Passed: false},
			},
		}

		data := buildReportData(result)
		require.Len(t, data.Sections, 1)
		groups := data.Sections[0].TagGroups
		require.Len(t, groups, 2)
		require.Equal(t, "@api", groups[0].TagLabel)
		require.Len(t, groups[0].Scenarios, 2)
		require.Equal(t, "Untagged", groups[1].TagLabel)
		require.Len(t, groups[1].Scenarios, 1)
	})
}
