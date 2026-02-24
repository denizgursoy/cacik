package cacik

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// ANSI color codes
const (
	colorReset    = "\033[0m"
	colorGreen    = "\033[32m"
	colorRed      = "\033[31m"
	colorYellow   = "\033[33m"
	colorCyan     = "\033[36m"
	colorGray     = "\033[90m"
	colorBold     = "\033[1m"
	colorStepText = "\033[38;2;187;181;41m"  // IntelliJ Cucumber yellow (#BBB529)
	colorMatchGrp = "\033[38;2;104;151;187m" // IntelliJ Cucumber param blue (#6897BB)
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
	r.writeln(r.color(colorCyan, "Feature:") + " " + r.color(colorBold, name))
}

// RuleStart prints the rule header
func (r *ConsoleReporter) RuleStart(name string) {
	r.writeln("")
	r.writeln("  " + r.color(colorCyan, "Rule:") + " " + r.color(colorBold, name))
}

// BackgroundStart prints the background header
func (r *ConsoleReporter) BackgroundStart() {
	r.writeln("")
	r.writeln("  " + r.color(colorCyan, "Background:"))
}

// ScenarioStart prints the scenario header
func (r *ConsoleReporter) ScenarioStart(name string) {
	r.writeln("")
	r.writeln("  " + r.color(colorCyan, "Scenario:") + " " + r.color(colorBold, name))
}

// formatStep formats a step with colored keyword and step text.
// If matchLocs is provided, capture group regions are highlighted in a
// distinct color (blue) while the rest of the text uses the step yellow.
// matchLocs contains pairs [start, end] of byte offsets into text for each
// capture group (same layout as FindStringSubmatchIndex minus the full-match
// pair).
func (r *ConsoleReporter) formatStep(keyword, text string, matchLocs []int) string {
	coloredKeyword := r.color(colorCyan, keyword)
	coloredText := r.colorizeStepText(text, matchLocs)
	return fmt.Sprintf("    %s%s", coloredKeyword, coloredText)
}

// colorizeStepText applies the step-text yellow to the entire text, but
// overrides capture-group regions with the match-group blue when matchLocs
// is non-nil.
func (r *ConsoleReporter) colorizeStepText(text string, matchLocs []int) string {
	if !r.useColors || len(matchLocs) < 2 {
		return r.color(colorStepText, text)
	}

	var b strings.Builder
	prev := 0
	for i := 0; i+1 < len(matchLocs); i += 2 {
		start, end := matchLocs[i], matchLocs[i+1]
		if start < 0 || end < 0 || start > len(text) || end > len(text) || start >= end {
			continue
		}
		// Text before the capture group — step yellow
		if start > prev {
			b.WriteString(colorStepText)
			b.WriteString(text[prev:start])
			b.WriteString(colorReset)
		}
		// Capture group — match blue + bold
		b.WriteString(colorMatchGrp)
		b.WriteString(colorBold)
		b.WriteString(text[start:end])
		b.WriteString(colorReset)
		prev = end
	}
	// Remaining text after last capture group
	if prev < len(text) {
		b.WriteString(colorStepText)
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

// StepSkipped prints a skipped step with yellow dash
func (r *ConsoleReporter) StepSkipped(keyword, text string) {
	step := r.formatStep(keyword, text, nil)
	symbol := r.color(colorYellow, symbolSkip)
	r.writeln(fmt.Sprintf("%-60s %s", step, symbol))
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
func (r *noopReporter) AddScenarioResult(passed bool)                                   {}
func (r *noopReporter) AddStepResult(passed bool, skipped bool)                         {}
func (r *noopReporter) Flush()                                                          {}
