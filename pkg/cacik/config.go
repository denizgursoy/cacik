package cacik

// Config holds runtime configuration settings for cacik.
// Discovered from a single function returning *cacik.Config.
// CLI flags (--fail-fast, --no-color, --disable-log, --disable-reporter) always override code config.
type Config struct {
	// FailFast stops execution on first scenario failure.
	FailFast bool

	// NoColor disables colored output.
	NoColor bool

	// DisableLog disables the structured logger (ctx.Logger()) used within
	// step functions. When true, a no-op logger that discards all messages
	// is injected instead of the default slog logger.
	// Default: false (logger is enabled).
	DisableLog bool

	// DisableReporter disables the BDD reporter output (feature, scenario,
	// step and summary lines). When true, no reporter output is printed.
	// Default: false (reporter output is enabled).
	DisableReporter bool

	// Logger sets a custom logger. If nil, default slog logger is used.
	Logger Logger

	// ReportFile is the file name (without extension) for the HTML test report.
	// When set, an HTML report is generated after all scenarios complete.
	// The ".html" extension is appended automatically.
	// CLI flag --report-file overrides this value.
	ReportFile string

	// AfterRun is called after all scenarios have executed.
	// Receives the complete run results for custom reporting.
	// This callback runs after the HTML report is generated (if configured)
	// and before Run() returns.
	AfterRun func(result RunResult)
}
