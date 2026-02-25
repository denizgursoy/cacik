package cacik

// Config holds runtime configuration settings for cacik.
// Settings are merged from all discovered config functions (last wins).
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
}

// MergeConfigs combines multiple configs into one.
// Later configs override earlier ones (last wins).
func MergeConfigs(configs ...*Config) *Config {
	result := &Config{}

	for _, cfg := range configs {
		if cfg == nil {
			continue
		}

		if cfg.FailFast {
			result.FailFast = true
		}
		if cfg.NoColor {
			result.NoColor = true
		}
		if cfg.DisableLog {
			result.DisableLog = true
		}
		if cfg.DisableReporter {
			result.DisableReporter = true
		}
		if cfg.Logger != nil {
			result.Logger = cfg.Logger
		}
	}

	return result
}
