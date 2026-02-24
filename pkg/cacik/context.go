// Package cacik provides the execution context for BDD step functions.
package cacik

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

// T is the test interface used for assertion failures.
// It is satisfied by *testing.T and by the internal panicT fallback.
type T interface {
	Errorf(format string, args ...any)
	FailNow()
	Helper()
	Failed() bool
}

// Logger is the interface for structured logging within step functions.
// Compatible with *slog.Logger and other structured loggers.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// Data provides scenario-scoped state management.
// Use this to store and retrieve values across steps within a scenario.
type Data struct {
	t      T
	values map[string]any
}

// Set stores a value in the scenario-scoped data store.
func (d *Data) Set(key string, value any) {
	d.values[key] = value
}

// Get retrieves a value from the scenario-scoped data store.
// Returns the value and a boolean indicating if the key was found.
func (d *Data) Get(key string) (any, bool) {
	v, ok := d.values[key]
	return v, ok
}

// MustGet retrieves a value or fails the test if not found.
func (d *Data) MustGet(key string) any {
	d.t.Helper()
	v, ok := d.values[key]
	if !ok {
		d.t.Errorf("key %q not found in context data", key)
		d.t.FailNow()
	}
	return v
}

// Context is the execution context passed to all step functions.
// It provides logging, assertions, and state management for BDD tests.
type Context struct {
	ctx      context.Context
	t        T
	logger   Logger
	assert   *Assert
	data     *Data
	reporter Reporter
}

// New creates a new Context with the given options.
func New(opts ...Option) *Context {
	t := &panicT{}
	c := &Context{
		ctx:    context.Background(),
		t:      t,
		assert: &Assert{t: t},
		data:   &Data{t: t, values: make(map[string]any)},
	}
	for _, opt := range opts {
		opt(c)
	}
	// Set defaults if not provided
	if c.logger == nil {
		c.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}
	if c.reporter == nil {
		c.reporter = &noopReporter{}
	}
	return c
}

// Context returns the underlying context.Context for library compatibility.
func (c *Context) Context() context.Context {
	return c.ctx
}

// WithContext updates the underlying context.Context.
// Use this for timeouts, cancellation, or storing values in the standard context.
func (c *Context) WithContext(ctx context.Context) {
	c.ctx = ctx
}

// Logger returns the logger instance.
func (c *Context) Logger() Logger {
	return c.logger
}

// Assert returns the assertion helper for making test assertions.
func (c *Context) Assert() *Assert {
	return c.assert
}

// Data returns the data store for scenario-scoped state management.
func (c *Context) Data() *Data {
	return c.data
}

// Reporter returns the reporter for test output.
func (c *Context) Reporter() Reporter {
	return c.reporter
}

// TestingT returns the T interface used for assertions.
func (c *Context) TestingT() T {
	return c.t
}

// NoopLogger discards all log messages.
type NoopLogger struct{}

func (n *NoopLogger) Debug(msg string, args ...any) {}
func (n *NoopLogger) Info(msg string, args ...any)  {}
func (n *NoopLogger) Warn(msg string, args ...any)  {}
func (n *NoopLogger) Error(msg string, args ...any) {}

// panicT panics on test failure. Used as fallback when *testing.T is not provided.
type panicT struct {
	failed bool
}

func (p *panicT) Errorf(format string, args ...any) {
	p.failed = true
	panic(fmt.Sprintf("test failed: "+format, args...))
}

func (p *panicT) FailNow() {
	p.failed = true
	panic("test failed")
}

func (p *panicT) Helper() {}

func (p *panicT) Failed() bool {
	return p.failed
}
