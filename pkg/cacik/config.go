package cacik

// Config holds runtime configuration settings for cacik.
// Settings are merged from all discovered config functions (last wins).
// CLI flags (--parallel, --fail-fast, --no-color) always override code config.
type Config struct {
	// Parallel sets the number of parallel workers.
	// 0 = sequential execution (default)
	// >0 = parallel execution with specified number of workers
	Parallel int

	// FailFast stops execution on first scenario failure.
	FailFast bool

	// NoColor disables colored output.
	NoColor bool

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

		if cfg.Parallel != 0 {
			result.Parallel = cfg.Parallel
		}
		if cfg.FailFast {
			result.FailFast = true
		}
		if cfg.NoColor {
			result.NoColor = true
		}
		if cfg.Logger != nil {
			result.Logger = cfg.Logger
		}
	}

	return result
}
