package cacik

import (
	"fmt"
	"html"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// tagGroup holds scenarios sharing the same tag combination.
type tagGroup struct {
	TagLabel  string           // e.g. "@smoke, @login" or "untagged"
	Count     int              // number of scenarios in this tag group
	Duration  time.Duration    // sum of scenario durations in this tag group
	Scenarios []ScenarioResult // scenarios in this tag group
}

// statusSection holds a top-level failed/passed section with tag sub-groups.
type statusSection struct {
	Label     string        // "Failed Scenarios" or "Passed Scenarios"
	CSSClass  string        // "failed" or "passed"
	Count     int           // number of scenarios in this section
	Duration  time.Duration // sum of scenario durations in this section
	TagGroups []tagGroup    // sub-groups ordered by tag label
}

// reportData is the view model passed to the HTML template.
type reportData struct {
	Summary       ReporterSummary
	TotalDuration time.Duration
	ExecutedAt    time.Time
	Sections      []statusSection
}

// sumDurations returns the total duration across a slice of scenarios.
func sumDurations(scenarios []ScenarioResult) time.Duration {
	var total time.Duration
	for _, s := range scenarios {
		total += s.Duration
	}
	return total
}

// buildReportData groups and sorts scenarios for the HTML report.
// Order: failed section first (if any), then passed section.
// Within each section, scenarios are grouped by their tag set.
func buildReportData(result RunResult) reportData {
	failed := make([]ScenarioResult, 0)
	passed := make([]ScenarioResult, 0)
	for _, s := range result.Scenarios {
		if s.Passed {
			passed = append(passed, s)
		} else {
			failed = append(failed, s)
		}
	}

	var sections []statusSection
	if len(failed) > 0 {
		sections = append(sections, statusSection{
			Label:     "Failed Scenarios",
			CSSClass:  "failed",
			Count:     len(failed),
			Duration:  sumDurations(failed),
			TagGroups: groupByTags(failed),
		})
	}
	if len(passed) > 0 {
		sections = append(sections, statusSection{
			Label:     "Passed Scenarios",
			CSSClass:  "passed",
			Count:     len(passed),
			Duration:  sumDurations(passed),
			TagGroups: groupByTags(passed),
		})
	}

	return reportData{
		Summary:       result.Summary,
		TotalDuration: result.Duration,
		ExecutedAt:    result.StartedAt,
		Sections:      sections,
	}
}

// groupByTags groups scenarios by their sorted tag set.
// Scenarios with no tags go into an "Untagged" group shown last.
func groupByTags(scenarios []ScenarioResult) []tagGroup {
	groups := make(map[string][]ScenarioResult)
	for _, s := range scenarios {
		key := tagKey(s.Tags)
		groups[key] = append(groups[key], s)
	}

	// Collect keys, sort so output is deterministic.
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Move "Untagged" to the end.
	result := make([]tagGroup, 0, len(keys))
	var untagged *tagGroup
	for _, k := range keys {
		scenarios := groups[k]
		tg := tagGroup{TagLabel: k, Count: len(scenarios), Duration: sumDurations(scenarios), Scenarios: scenarios}
		if k == "Untagged" {
			untagged = &tg
		} else {
			result = append(result, tg)
		}
	}
	if untagged != nil {
		result = append(result, *untagged)
	}
	return result
}

// tagKey builds a deterministic label from a scenario's tags.
func tagKey(tags []string) string {
	if len(tags) == 0 {
		return "Untagged"
	}
	sorted := make([]string, len(tags))
	copy(sorted, tags)
	sort.Strings(sorted)
	return strings.Join(sorted, ", ")
}

// htmlParamColors are the hex colors corresponding to each positional entry in
// the console reporter's colorParams slice. They are used in the HTML report
// to highlight captured step parameters.
var htmlParamColors = []string{
	"#5C92FF", // bright blue
	"#00CED1", // cyan
	"#E5C07B", // gold
	"#C0A0FF", // lavender
	"#98C379", // lime
	"#56B6C2", // teal
	"#E06C75", // rose
	"#D19A66", // amber
	"#7ECEA0", // mint
	"#A993D6", // violet
}

// colorizeStepText returns safe HTML for a step's text with capture-group
// parameters wrapped in colored spans. When MatchLocs is empty the text is
// returned as a single span. For skipped steps all text (including params)
// uses the skipped color class instead of positional colors.
func colorizeStepText(step StepResult) template.HTML {
	escaped := html.EscapeString(step.Text)
	statusCls := stepStatusClass(step.Status)

	if len(step.MatchLocs) == 0 {
		return template.HTML(fmt.Sprintf(`<span class="step-text %s">%s</span>`, statusCls, escaped))
	}

	// MatchLocs is a flat slice of [start0, end0, start1, end1, ...] byte
	// offsets into step.Text for each capture group.
	var b strings.Builder
	cursor := 0
	paramIdx := 0
	for i := 0; i+1 < len(step.MatchLocs); i += 2 {
		start := step.MatchLocs[i]
		end := step.MatchLocs[i+1]
		if start < 0 || end < 0 {
			// Non-participating group — skip.
			continue
		}
		// Plain text before this capture group.
		if cursor < start {
			b.WriteString(fmt.Sprintf(`<span class="step-text %s">%s</span>`, statusCls, html.EscapeString(step.Text[cursor:start])))
		}
		// The captured parameter.
		if step.Status == StepSkipped {
			b.WriteString(fmt.Sprintf(`<span class="step-text skipped">%s</span>`, html.EscapeString(step.Text[start:end])))
		} else {
			colorIdx := paramIdx % len(htmlParamColors)
			b.WriteString(fmt.Sprintf(`<span class="step-param" style="color:%s">%s</span>`, htmlParamColors[colorIdx], html.EscapeString(step.Text[start:end])))
		}
		paramIdx++
		cursor = end
	}
	// Trailing plain text after the last capture group.
	if cursor < len(step.Text) {
		b.WriteString(fmt.Sprintf(`<span class="step-text %s">%s</span>`, statusCls, html.EscapeString(step.Text[cursor:])))
	}
	return template.HTML(b.String())
}

// stepStatusClass returns the CSS class name for a step status.
func stepStatusClass(s StepStatus) string {
	switch s {
	case StepPassed:
		return "passed"
	case StepFailed:
		return "failed"
	case StepSkipped:
		return "skipped"
	default:
		return "unknown"
	}
}

// GenerateHTMLReport writes a self-contained HTML test report to the given path.
// The report includes all scenario and step results with timing data, styled
// with inline CSS. No external dependencies are required to view the report.
func GenerateHTMLReport(path string, result RunResult) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("could not create report directory %q: %w", dir, err)
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("could not create report file %q: %w", path, err)
	}
	defer f.Close()

	tmpl, err := template.New("report").Funcs(template.FuncMap{
		"statusClass": func(s StepStatus) string {
			return stepStatusClass(s)
		},
		"colorizeStepText": colorizeStepText,
		"summaryClass": func(failed int) string {
			if failed > 0 {
				return "has-failures"
			}
			return "all-passed"
		},
		"statusSymbol": func(s StepStatus) string {
			switch s {
			case StepPassed:
				return "\u2713" // ✓
			case StepFailed:
				return "\u2717" // ✗
			case StepSkipped:
				return "\u2013" // –
			default:
				return "?"
			}
		},
		"formatDuration": func(d time.Duration) string {
			if d < time.Millisecond {
				return fmt.Sprintf("%.0fµs", float64(d)/float64(time.Microsecond))
			}
			if d < time.Second {
				return fmt.Sprintf("%.0fms", float64(d)/float64(time.Millisecond))
			}
			return fmt.Sprintf("%.2fs", d.Seconds())
		},
		"scenarioClass": func(passed bool) string {
			if passed {
				return "passed"
			}
			return "failed"
		},
		"formatTime": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			return t.Format("2006-01-02 15:04:05")
		},
	}).Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("could not parse HTML template: %w", err)
	}

	data := buildReportData(result)
	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("could not render HTML report: %w", err)
	}

	return nil
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Test Execution Report</title>
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen,
                 Ubuntu, Cantarell, "Fira Sans", "Droid Sans", "Helvetica Neue", sans-serif;
    background: #f8f9fa; color: #212529; line-height: 1.6; padding: 2rem;
  }
  h1 { font-size: 1.5rem; margin-bottom: 0.25rem; color: #212529; font-weight: 700; }
  .executed-at {
    font-size: 0.8rem; color: #868e96; margin-bottom: 1.5rem;
  }

  /* ── Summary dashboard ── */
  .summary {
    display: flex; gap: 1rem; flex-wrap: wrap;
    margin-bottom: 2rem; padding: 1rem 1.25rem; background: #fff;
    border-radius: 10px; border: 1px solid #e9ecef;
    box-shadow: 0 1px 3px rgba(0,0,0,0.04);
  }
  .summary.all-passed {
    border: 2px solid #2b8a3e; background: #f6fef7;
  }
  .summary.has-failures {
    border: 2px solid #c92a2a; background: #fff5f5;
  }
  .summary-item { text-align: center; min-width: 90px; }
  .summary-item .number { font-size: 1.8rem; font-weight: 700; }
  .summary-item .label {
    font-size: 0.7rem; text-transform: uppercase; letter-spacing: 0.05em;
    color: #868e96;
  }
  .number.green  { color: #2b8a3e; }
  .number.red    { color: #c92a2a; }
  .number.yellow { color: #e67700; }
  .number.blue   { color: #1864ab; }

  /* ── Section (Failed / Passed) ── */
  .section { margin-bottom: 2rem; }
  .section-header {
    font-size: 1.1rem; font-weight: 700; margin-bottom: 0.75rem;
    padding-bottom: 0.4rem; border-bottom: 2px solid #dee2e6;
    display: flex; align-items: center; gap: 0.5rem;
  }
  .section-header .dot {
    width: 10px; height: 10px; border-radius: 50%; display: inline-block;
  }
  .section-header .section-meta {
    font-size: 0.8rem; font-weight: 500; color: #868e96;
  }
  .section.failed .section-header { color: #c92a2a; }
  .section.failed .dot { background: #c92a2a; }
  .section.passed .section-header { color: #2b8a3e; }
  .section.passed .dot { background: #2b8a3e; }

  /* ── Tag group ── */
  .tag-group { margin-bottom: 1.25rem; margin-left: 0.25rem; }
  .tag-group-label {
    font-size: 0.8rem; font-weight: 600; color: #495057;
    margin-bottom: 0.4rem; padding-left: 0.25rem;
    display: flex; align-items: center; gap: 0.4rem;
  }
  .tag-group-label .tag-icon { color: #868e96; }
  .tag-group-meta { font-size: 0.75rem; font-weight: 400; color: #868e96; }

  /* ── Toggle buttons ── */
  .toggle-bar {
    display: flex; gap: 0.5rem; margin-bottom: 1rem;
  }
  .toggle-btn {
    background: #fff; border: 1px solid #dee2e6; border-radius: 6px;
    padding: 0.3rem 0.75rem; font-size: 0.75rem; color: #495057;
    cursor: pointer; font-weight: 500; transition: background 0.15s, border-color 0.15s;
  }
  .toggle-btn:hover { background: #f1f3f5; border-color: #adb5bd; }
  .tag-group-toggle {
    background: none; border: none; font-size: 0.7rem; color: #868e96;
    cursor: pointer; padding: 0; margin-left: auto; font-weight: 500;
  }
  .tag-group-toggle:hover { color: #495057; }
  .section-toggle {
    background: none; border: none; font-size: 0.75rem; color: #868e96;
    cursor: pointer; padding: 0; margin-left: 0.5rem; font-weight: 500;
  }
  .section-toggle:hover { color: #495057; }

  /* ── Scenario card ── */
  .scenario {
    margin-bottom: 0.5rem; background: #fff; border-radius: 8px;
    border-left: 4px solid #ced4da; overflow: hidden;
    border: 1px solid #e9ecef;
    box-shadow: 0 1px 2px rgba(0,0,0,0.03);
  }
  .scenario.passed { border-left: 4px solid #69db7c; }
  .scenario.failed { border-left: 4px solid #ff6b6b; }

  .scenario-header {
    display: flex; justify-content: space-between; align-items: center;
    padding: 0.6rem 1rem; cursor: pointer; user-select: none;
    transition: background 0.15s;
  }
  .scenario-header:hover { background: #f1f3f5; }
  .scenario-name { font-weight: 600; font-size: 0.9rem; color: #212529; }
  .scenario-meta {
    display: flex; gap: 0.75rem; align-items: center;
    font-size: 0.78rem; color: #868e96;
  }
  .feature-label { color: #495057; font-size: 0.78rem; }
  .tag {
    background: #e9ecef; border-radius: 4px; padding: 0.1rem 0.45rem;
    font-size: 0.68rem; color: #495057; font-weight: 500;
  }

  .steps {
    padding: 0.5rem 1rem 0.75rem 1rem; display: none;
    background: #1e1f22; border-radius: 0 0 6px 6px;
  }
  .scenario.open .steps { display: block; }

  .step {
    display: flex; align-items: baseline; gap: 0.5rem;
    padding: 0.2rem 0;
    font-family: "JetBrains Mono", "Fira Code", "Cascadia Code", "SF Mono", monospace;
    font-size: 0.82rem;
  }
  .step-symbol { width: 1.2rem; text-align: center; flex-shrink: 0; font-weight: 700; }
  .step-symbol.passed  { color: #32cd32; }
  .step-symbol.failed  { color: #ff4444; }
  .step-symbol.skipped { color: #e6b800; }
  .step-keyword { color: #CF8E6D; font-weight: 600; white-space: pre; }
  .step-text { color: #BCBEC4; }
  .step-text.skipped { color: #6F737A; }
  .step-keyword.skipped { color: #6F737A; }
  .step-param { font-weight: 600; }
  .step-duration {
    margin-left: auto; color: #6F737A; font-size: 0.72rem; white-space: nowrap;
  }
  .step-error {
    color: #ff4444; background: #2c1a1a; border-radius: 4px;
    padding: 0.3rem 0.5rem; margin: 0.15rem 0 0.15rem 1.7rem;
    font-size: 0.78rem; white-space: pre-wrap; border: 1px solid #4a2020;
  }

  .step-group-label {
    font-family: "JetBrains Mono", "Fira Code", "Cascadia Code", "SF Mono", monospace;
    font-size: 0.82rem; color: #BCBEC4; padding: 0.35rem 0 0.1rem 0;
  }
  .step-group-label.step-group-rule { margin-top: 0.6rem; }
  .step-group-label.step-group-scenario { margin-top: 0.6rem; }
  .step-group-kw { color: #CF8E6D; font-weight: 600; }

  .chevron {
    transition: transform 0.2s; font-size: 0.7rem; color: #adb5bd;
  }
  .scenario.open .chevron { transform: rotate(90deg); }

  .empty-msg {
    color: #868e96; font-style: italic; padding: 1rem 0; text-align: center;
  }

  @media (max-width: 600px) {
    body { padding: 0.75rem; }
    .summary { flex-direction: column; gap: 0.5rem; }
  }
</style>
</head>
<body>
<h1>Test Execution Report</h1>
{{if not .ExecutedAt.IsZero}}<div class="executed-at">Executed at {{formatTime .ExecutedAt}}</div>{{end}}

<div class="summary {{summaryClass .Summary.ScenariosFailed}}">
  <div class="summary-item">
    <div class="number blue">{{.Summary.ScenariosTotal}}</div>
    <div class="label">Scenarios</div>
  </div>
  <div class="summary-item">
    <div class="number green">{{.Summary.ScenariosPassed}}</div>
    <div class="label">Passed</div>
  </div>
  <div class="summary-item">
    <div class="number red">{{.Summary.ScenariosFailed}}</div>
    <div class="label">Failed</div>
  </div>
  <div class="summary-item">
    <div class="number blue">{{.Summary.StepsTotal}}</div>
    <div class="label">Steps</div>
  </div>
  <div class="summary-item">
    <div class="number green">{{.Summary.StepsPassed}}</div>
    <div class="label">Steps Passed</div>
  </div>
  <div class="summary-item">
    <div class="number red">{{.Summary.StepsFailed}}</div>
    <div class="label">Steps Failed</div>
  </div>
  <div class="summary-item">
    <div class="number yellow">{{.Summary.StepsSkipped}}</div>
    <div class="label">Steps Skipped</div>
  </div>
  <div class="summary-item">
    <div class="number blue">{{formatDuration .TotalDuration}}</div>
    <div class="label">Duration</div>
  </div>
</div>

{{if not .Sections}}
<div class="empty-msg">No scenarios were executed.</div>
{{else}}
<div class="toggle-bar">
  <button class="toggle-btn" onclick="expandAll()">Expand All</button>
  <button class="toggle-btn" onclick="collapseAll()">Collapse All</button>
</div>
{{end}}

{{range .Sections}}
<div class="section {{.CSSClass}}">
  <div class="section-header"><span class="dot"></span> {{.Label}} <span class="section-meta">{{.Count}} scenarios, {{formatDuration .Duration}}</span><button class="section-toggle" onclick="expandSection(this)">Expand All</button><button class="section-toggle" onclick="collapseSection(this)">Collapse All</button></div>
  {{range .TagGroups}}
  <div class="tag-group">
    <div class="tag-group-label"><span class="tag-icon">#</span> {{.TagLabel}} <span class="tag-group-meta">({{.Count}} scenarios, {{formatDuration .Duration}})</span><button class="tag-group-toggle" onclick="expandGroup(this)">Expand</button><button class="tag-group-toggle" onclick="collapseGroup(this)">Collapse</button></div>
    {{range .Scenarios}}
    <div class="scenario {{scenarioClass .Passed}}">
      <div class="scenario-header" onclick="this.parentElement.classList.toggle('open')">
        <div>
          <span class="feature-label">{{.FeatureName}}</span>{{if .RuleName}} / <span class="feature-label">{{.RuleName}}</span>{{end}}
          <br>
          <span class="scenario-name">{{.Name}}</span>
          {{range .Tags}}<span class="tag">{{.}}</span> {{end}}
        </div>
        <div class="scenario-meta">
          <span>{{formatDuration .Duration}}</span>
          <span class="chevron">&#9654;</span>
        </div>
      </div>
      <div class="steps">
        {{if .FeatureBgSteps}}
        <div class="step-group-label"><span class="step-group-kw">Background:</span></div>
        {{range .FeatureBgSteps}}
        <div class="step">
          <span class="step-symbol {{statusClass .Status}}">{{statusSymbol .Status}}</span>
          <span class="step-keyword {{statusClass .Status}}">{{.Keyword}}</span>
          {{colorizeStepText .}}
          <span class="step-duration">{{formatDuration .Duration}}</span>
        </div>
        {{if .Error}}<div class="step-error">{{.Error}}</div>{{end}}
        {{end}}
        {{end}}
        {{if .RuleName}}<div class="step-group-label step-group-rule"><span class="step-group-kw">Rule:</span> {{.RuleName}}</div>{{end}}
        {{if .RuleBgSteps}}
        <div class="step-group-label"><span class="step-group-kw">Background:</span></div>
        {{range .RuleBgSteps}}
        <div class="step">
          <span class="step-symbol {{statusClass .Status}}">{{statusSymbol .Status}}</span>
          <span class="step-keyword {{statusClass .Status}}">{{.Keyword}}</span>
          {{colorizeStepText .}}
          <span class="step-duration">{{formatDuration .Duration}}</span>
        </div>
        {{if .Error}}<div class="step-error">{{.Error}}</div>{{end}}
        {{end}}
        {{end}}
        <div class="step-group-label step-group-scenario"><span class="step-group-kw">Scenario:</span> {{.Name}}</div>
        {{range .Steps}}
        <div class="step">
          <span class="step-symbol {{statusClass .Status}}">{{statusSymbol .Status}}</span>
          <span class="step-keyword {{statusClass .Status}}">{{.Keyword}}</span>
          {{colorizeStepText .}}
          <span class="step-duration">{{formatDuration .Duration}}</span>
        </div>
        {{if .Error}}<div class="step-error">{{.Error}}</div>{{end}}
        {{end}}
      </div>
    </div>
    {{end}}
  </div>
  {{end}}
</div>
{{end}}

<script>
// Auto-expand failed scenarios
document.querySelectorAll('.scenario.failed').forEach(function(el) { el.classList.add('open'); });

// Global expand/collapse
function expandAll() {
  document.querySelectorAll('.scenario').forEach(function(el) { el.classList.add('open'); });
}
function collapseAll() {
  document.querySelectorAll('.scenario').forEach(function(el) { el.classList.remove('open'); });
}

// Per section expand/collapse
function expandSection(btn) {
  btn.closest('.section').querySelectorAll('.scenario').forEach(function(el) { el.classList.add('open'); });
}
function collapseSection(btn) {
  btn.closest('.section').querySelectorAll('.scenario').forEach(function(el) { el.classList.remove('open'); });
}

// Per tag-group expand/collapse
function expandGroup(btn) {
  btn.closest('.tag-group').querySelectorAll('.scenario').forEach(function(el) { el.classList.add('open'); });
}
function collapseGroup(btn) {
  btn.closest('.tag-group').querySelectorAll('.scenario').forEach(function(el) { el.classList.remove('open'); });
}
</script>
</body>
</html>
`
