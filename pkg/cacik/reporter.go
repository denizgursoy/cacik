package cacik

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// ANSI color codes
const (
	colorReset = "\033[0m"
	colorGreen = "\033[32m"
	colorRed   = "\033[31m"

	colorKeyword      = "\033[38;2;207;142;109m" // #CF8E6D — keywords (Feature:, Scenario:, Given, etc.)
	colorText         = "\033[38;2;188;190;196m" // #BCBEC4 — step text, feature/scenario names
	colorParam        = "\033[38;2;92;146;255m"  // #5C92FF — captured step parameters, table data cells
	colorOutlineParam = "\033[38;2;199;125;187m" // #C77DBB — <placeholder> params, table header cells
	colorSkipped      = "\033[38;2;111;115;122m" // #6F737A — skipped step text
	colorYellow       = "\033[33m"               // skipped step symbol
)

// Symbols for step status
const (
	symbolPass = "✓"
	symbolFail = "✗"
	symbolSkip = "-"
)

// Reporter handles test execution output
type Reporter interface {
	// Feature/Scenario lifecycle
	FeatureStart(name string)
	RuleStart(name string)
	BackgroundStart()
	ScenarioStart(name string)

	// Step reporting
	// matchLocs holds pairs of [start, end] byte offsets for each capture group
	// within text (same format as regexp.FindStringSubmatchIndex, but without
	// the full-match pair at the front). Pass nil when match info is unavailable.
	StepPassed(keyword, text string, matchLocs []int)
	StepFailed(keyword, text string, errMsg string, matchLocs []int)
	StepSkipped(keyword, text string)

	// StepDataTable prints a DataTable attached to a step.
	// rows is the raw cell data (including header row).
	// Call this immediately after StepPassed/StepFailed/StepSkipped when the
	// step has an attached DataTable.
	StepDataTable(rows [][]string)

	// Summary
	AddScenarioResult(passed bool)
	AddStepResult(passed bool, skipped bool)

	// Output control
	Flush() // For buffered reporters - prints accumulated output
}

// ReporterSummary tracks test execution statistics
type ReporterSummary struct {
	ScenariosTotal  int
	ScenariosPassed int
	ScenariosFailed int
	StepsTotal      int
	StepsPassed     int
	StepsFailed     int
	StepsSkipped    int
}

// ConsoleReporter prints colored output to stdout
type ConsoleReporter struct {
	useColors bool
	buffer    *strings.Builder
	buffered  bool
	disabled  bool
	mu        sync.Mutex
	summary   ReporterSummary
}

// NewConsoleReporter creates a reporter that prints directly to stdout
func NewConsoleReporter(useColors bool) *ConsoleReporter {
	return &ConsoleReporter{
		useColors: useColors,
		buffered:  false,
	}
}

// NewBufferedReporter creates a reporter that buffers output for atomic printing
// Used for parallel execution to prevent interleaved output
func NewBufferedReporter(useColors bool) *ConsoleReporter {
	return &ConsoleReporter{
		useColors: useColors,
		buffer:    &strings.Builder{},
		buffered:  true,
	}
}

// NewNoopConsoleReporter creates a ConsoleReporter that suppresses all output.
// Summary statistics are still tracked.
func NewNoopConsoleReporter() *ConsoleReporter {
	return &ConsoleReporter{
		disabled: true,
	}
}

func (r *ConsoleReporter) write(s string) {
	if r.disabled {
		return
	}
	if r.buffered {
		r.buffer.WriteString(s)
	} else {
		fmt.Print(s)
	}
}

func (r *ConsoleReporter) writeln(s string) {
	r.write(s + "\n")
}

func (r *ConsoleReporter) color(c, s string) string {
	if r.useColors {
		return c + s + colorReset
	}
	return s
}

// FeatureStart prints the feature header
func (r *ConsoleReporter) FeatureStart(name string) {
	r.writeln("")
	r.writeln(r.color(colorKeyword, "Feature:") + " " + r.color(colorText, name))
}

// RuleStart prints the rule header
func (r *ConsoleReporter) RuleStart(name string) {
	r.writeln("")
	r.writeln("  " + r.color(colorKeyword, "Rule:") + " " + r.color(colorText, name))
}

// BackgroundStart prints the background header
func (r *ConsoleReporter) BackgroundStart() {
	r.writeln("")
	r.writeln("  " + r.color(colorKeyword, "Background:"))
}

// ScenarioStart prints the scenario header.
// Segments enclosed in angle brackets (e.g. <param>) are highlighted with
// the outline-parameter color.
func (r *ConsoleReporter) ScenarioStart(name string) {
	r.writeln("")
	r.writeln("  " + r.color(colorKeyword, "Scenario:") + " " + r.colorizeOutlineParams(name))
}

// colorizeOutlineParams applies the text color to the name while highlighting
// <placeholder> segments with the outline-parameter color.
func (r *ConsoleReporter) colorizeOutlineParams(name string) string {
	if !r.useColors {
		return name
	}

	var b strings.Builder
	prev := 0
	for {
		start := strings.Index(name[prev:], "<")
		if start < 0 {
			break
		}
		start += prev
		end := strings.Index(name[start:], ">")
		if end < 0 {
			break
		}
		end += start + 1 // include the '>'

		// Text before the <param>
		if start > prev {
			b.WriteString(colorText)
			b.WriteString(name[prev:start])
			b.WriteString(colorReset)
		}
		// The <param> itself
		b.WriteString(colorOutlineParam)
		b.WriteString(name[start:end])
		b.WriteString(colorReset)
		prev = end
	}
	if prev == 0 {
		// No angle brackets found — plain text color
		return colorText + name + colorReset
	}
	// Remaining text after last <param>
	if prev < len(name) {
		b.WriteString(colorText)
		b.WriteString(name[prev:])
		b.WriteString(colorReset)
	}
	return b.String()
}

// formatStep formats a step with colored keyword and step text.
// If matchLocs is provided, capture group regions are highlighted with the
// parameter color while the rest of the text uses the text color.
// matchLocs contains pairs [start, end] of byte offsets into text for each
// capture group (same layout as FindStringSubmatchIndex minus the full-match
// pair).
func (r *ConsoleReporter) formatStep(keyword, text string, matchLocs []int) string {
	coloredKeyword := r.color(colorKeyword, keyword)
	coloredText := r.colorizeStepText(text, matchLocs)
	return fmt.Sprintf("    %s%s", coloredKeyword, coloredText)
}

// colorizeStepText applies the text color to the entire step text, but
// overrides capture-group regions with the parameter color when matchLocs
// is non-nil.
func (r *ConsoleReporter) colorizeStepText(text string, matchLocs []int) string {
	if !r.useColors || len(matchLocs) < 2 {
		return r.color(colorText, text)
	}

	var b strings.Builder
	prev := 0
	for i := 0; i+1 < len(matchLocs); i += 2 {
		start, end := matchLocs[i], matchLocs[i+1]
		if start < 0 || end < 0 || start > len(text) || end > len(text) || start >= end {
			continue
		}
		// Text before the capture group
		if start > prev {
			b.WriteString(colorText)
			b.WriteString(text[prev:start])
			b.WriteString(colorReset)
		}
		// Capture group — parameter color
		b.WriteString(colorParam)
		b.WriteString(text[start:end])
		b.WriteString(colorReset)
		prev = end
	}
	// Remaining text after last capture group
	if prev < len(text) {
		b.WriteString(colorText)
		b.WriteString(text[prev:])
		b.WriteString(colorReset)
	}
	return b.String()
}

// StepPassed prints a passed step with green checkmark
func (r *ConsoleReporter) StepPassed(keyword, text string, matchLocs []int) {
	step := r.formatStep(keyword, text, matchLocs)
	symbol := r.color(colorGreen, symbolPass)
	r.writeln(fmt.Sprintf("%-60s %s", step, symbol))
}

// StepFailed prints a failed step with red X and error message
func (r *ConsoleReporter) StepFailed(keyword, text string, errMsg string, matchLocs []int) {
	step := r.formatStep(keyword, text, matchLocs)
	symbol := r.color(colorRed, symbolFail)
	r.writeln(fmt.Sprintf("%-60s %s", step, symbol))

	// Print error message indented
	if errMsg != "" {
		lines := strings.Split(errMsg, "\n")
		for _, line := range lines {
			r.writeln(r.color(colorRed, "      "+line))
		}
	}
}

// StepSkipped prints a skipped step with dimmed text and yellow dash
func (r *ConsoleReporter) StepSkipped(keyword, text string) {
	coloredKeyword := r.color(colorSkipped, keyword)
	coloredText := r.color(colorSkipped, text)
	step := fmt.Sprintf("    %s%s", coloredKeyword, coloredText)
	symbol := r.color(colorYellow, symbolSkip)
	r.writeln(fmt.Sprintf("%-60s %s", step, symbol))
}

// StepDataTable prints a DataTable below the step line with aligned columns.
// The first row is treated as the header (cells colored with the outline-param
// color) and subsequent rows use the step-param color. Pipe delimiters use the
// keyword color.
func (r *ConsoleReporter) StepDataTable(rows [][]string) {
	if len(rows) == 0 {
		return
	}

	// Compute max width per column
	colWidths := make([]int, len(rows[0]))
	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// Print each row with individually colored pipes, header cells, and data cells
	for rowIdx, row := range rows {
		var b strings.Builder
		b.WriteString("      ")
		b.WriteString(r.color(colorKeyword, "|"))
		b.WriteString(" ")
		for i, cell := range row {
			width := 0
			if i < len(colWidths) {
				width = colWidths[i]
			}
			padded := fmt.Sprintf("%-*s", width, cell)
			if rowIdx == 0 {
				b.WriteString(r.color(colorOutlineParam, padded))
			} else {
				b.WriteString(r.color(colorParam, padded))
			}
			b.WriteString(" ")
			b.WriteString(r.color(colorKeyword, "|"))
			b.WriteString(" ")
		}
		r.writeln(b.String())
	}
}

// AddScenarioResult tracks scenario pass/fail for summary
func (r *ConsoleReporter) AddScenarioResult(passed bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.summary.ScenariosTotal++
	if passed {
		r.summary.ScenariosPassed++
	} else {
		r.summary.ScenariosFailed++
	}
}

// AddStepResult tracks step pass/fail/skip for summary
func (r *ConsoleReporter) AddStepResult(passed bool, skipped bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.summary.StepsTotal++
	if skipped {
		r.summary.StepsSkipped++
	} else if passed {
		r.summary.StepsPassed++
	} else {
		r.summary.StepsFailed++
	}
}

// GetSummary returns the current summary statistics
func (r *ConsoleReporter) GetSummary() ReporterSummary {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.summary
}

// PrintSummary prints the final test summary
func (r *ConsoleReporter) PrintSummary() {
	r.mu.Lock()
	summary := r.summary
	r.mu.Unlock()

	r.writeln("")

	// Scenarios summary
	scenarioLine := fmt.Sprintf("%d scenario(s)", summary.ScenariosTotal)
	if summary.ScenariosTotal > 0 {
		parts := []string{}
		if summary.ScenariosPassed > 0 {
			parts = append(parts, r.color(colorGreen, fmt.Sprintf("%d passed", summary.ScenariosPassed)))
		}
		if summary.ScenariosFailed > 0 {
			parts = append(parts, r.color(colorRed, fmt.Sprintf("%d failed", summary.ScenariosFailed)))
		}
		if len(parts) > 0 {
			scenarioLine += " (" + strings.Join(parts, ", ") + ")"
		}
	}
	r.writeln(scenarioLine)

	// Steps summary
	stepLine := fmt.Sprintf("%d step(s)", summary.StepsTotal)
	if summary.StepsTotal > 0 {
		parts := []string{}
		if summary.StepsPassed > 0 {
			parts = append(parts, r.color(colorGreen, fmt.Sprintf("%d passed", summary.StepsPassed)))
		}
		if summary.StepsFailed > 0 {
			parts = append(parts, r.color(colorRed, fmt.Sprintf("%d failed", summary.StepsFailed)))
		}
		if summary.StepsSkipped > 0 {
			parts = append(parts, r.color(colorYellow, fmt.Sprintf("%d skipped", summary.StepsSkipped)))
		}
		if len(parts) > 0 {
			stepLine += " (" + strings.Join(parts, ", ") + ")"
		}
	}
	r.writeln(stepLine)
}

// Flush prints buffered output atomically (for parallel execution)
func (r *ConsoleReporter) Flush() {
	if r.buffered && r.buffer.Len() > 0 {
		r.mu.Lock()
		defer r.mu.Unlock()
		fmt.Fprint(os.Stdout, r.buffer.String())
		r.buffer.Reset()
	}
}

// MergeSummary merges another reporter's summary into this one (for parallel execution)
func (r *ConsoleReporter) MergeSummary(other *ConsoleReporter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	otherSummary := other.GetSummary()
	r.summary.ScenariosTotal += otherSummary.ScenariosTotal
	r.summary.ScenariosPassed += otherSummary.ScenariosPassed
	r.summary.ScenariosFailed += otherSummary.ScenariosFailed
	r.summary.StepsTotal += otherSummary.StepsTotal
	r.summary.StepsPassed += otherSummary.StepsPassed
	r.summary.StepsFailed += otherSummary.StepsFailed
	r.summary.StepsSkipped += otherSummary.StepsSkipped
}

// noopReporter discards all output
type noopReporter struct{}

// NewNoopReporter creates a reporter that discards all output
func NewNoopReporter() Reporter {
	return &noopReporter{}
}

func (r *noopReporter) FeatureStart(name string)                                        {}
func (r *noopReporter) RuleStart(name string)                                           {}
func (r *noopReporter) BackgroundStart()                                                {}
func (r *noopReporter) ScenarioStart(name string)                                       {}
func (r *noopReporter) StepPassed(keyword, text string, matchLocs []int)                {}
func (r *noopReporter) StepFailed(keyword, text string, errMsg string, matchLocs []int) {}
func (r *noopReporter) StepSkipped(keyword, text string)                                {}
func (r *noopReporter) StepDataTable(rows [][]string)                                   {}
func (r *noopReporter) AddScenarioResult(passed bool)                                   {}
func (r *noopReporter) AddStepResult(passed bool, skipped bool)                         {}
func (r *noopReporter) Flush()                                                          {}
