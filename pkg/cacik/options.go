package cacik

import "context"

// Option configures a Context.
type Option func(*Context)

// WithLogger sets the logger for the context.
func WithLogger(logger Logger) Option {
	return func(c *Context) {
		c.logger = logger
	}
}

// WithContext sets the underlying context.Context.
func WithContext(ctx context.Context) Option {
	return func(c *Context) {
		c.ctx = ctx
	}
}

// WithData sets initial data for the context.
func WithData(data map[string]any) Option {
	return func(c *Context) {
		c.data.values = data
	}
}

// WithReporter sets the reporter for test output.
func WithReporter(reporter Reporter) Option {
	return func(c *Context) {
		c.reporter = reporter
	}
}

// WithTestingT sets the T interface (typically *testing.T) for assertions.
// When set, assertion failures use t.Fatalf() instead of panicking.
func WithTestingT(t T) Option {
	return func(c *Context) {
		c.t = t
		c.assert.t = t
		c.data.t = t
	}
}
