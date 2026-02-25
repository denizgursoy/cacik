package cacik

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// mockLogger is a mock implementation of Logger for testing
type mockLogger struct {
	debugMessages []string
	infoMessages  []string
	warnMessages  []string
	errorMessages []string
}

func (m *mockLogger) Debug(msg string, args ...any) {
	m.debugMessages = append(m.debugMessages, msg)
}

func (m *mockLogger) Info(msg string, args ...any) {
	m.infoMessages = append(m.infoMessages, msg)
}

func (m *mockLogger) Warn(msg string, args ...any) {
	m.warnMessages = append(m.warnMessages, msg)
}

func (m *mockLogger) Error(msg string, args ...any) {
	m.errorMessages = append(m.errorMessages, msg)
}

// =============================================================================
// Context Tests
// =============================================================================

func TestNew(t *testing.T) {
	t.Run("creates context with defaults", func(t *testing.T) {
		ctx := New()
		require.NotNil(t, ctx)
		require.NotNil(t, ctx.Context())
		require.NotNil(t, ctx.Logger())
		require.NotNil(t, ctx.Assert())
		require.NotNil(t, ctx.Data())
		require.NotEmpty(t, ctx.ID())
	})

	t.Run("creates context with custom logger", func(t *testing.T) {
		logger := &mockLogger{}
		ctx := New(WithLogger(logger))

		ctx.Logger().Info("test message")
		require.Len(t, logger.infoMessages, 1)
		require.Equal(t, "test message", logger.infoMessages[0])
	})

	t.Run("creates context with custom context.Context", func(t *testing.T) {
		stdCtx := context.WithValue(context.Background(), "key", "value")
		ctx := New(WithContext(stdCtx))
		require.Equal(t, "value", ctx.Context().Value("key"))
	})

	t.Run("creates context with initial data", func(t *testing.T) {
		data := map[string]any{"key": "value"}
		ctx := New(WithData(data))
		v, ok := ctx.Data().Get("key")
		require.True(t, ok)
		require.Equal(t, "value", v)
	})
}

func TestData_SetGet(t *testing.T) {
	t.Run("set and get values", func(t *testing.T) {
		ctx := New()
		ctx.Data().Set("name", "Alice")
		ctx.Data().Set("count", 42)

		name, ok := ctx.Data().Get("name")
		require.True(t, ok)
		require.Equal(t, "Alice", name)

		count, ok := ctx.Data().Get("count")
		require.True(t, ok)
		require.Equal(t, 42, count)
	})

	t.Run("get returns false for missing key", func(t *testing.T) {
		ctx := New()
		_, ok := ctx.Data().Get("missing")
		require.False(t, ok)
	})

	t.Run("MustGet returns value", func(t *testing.T) {
		ctx := New()
		ctx.Data().Set("key", "value")
		v := ctx.Data().MustGet("key")
		require.Equal(t, "value", v)
	})

	t.Run("MustGet panics for missing key", func(t *testing.T) {
		ctx := New()
		require.Panics(t, func() {
			ctx.Data().MustGet("missing")
		})
	})
}

func TestContext_WithContext(t *testing.T) {
	t.Run("updates underlying context", func(t *testing.T) {
		ctx := New()
		require.NotNil(t, ctx.Context())

		newStdCtx := context.WithValue(context.Background(), "key", "value")
		ctx.WithContext(newStdCtx)

		require.Equal(t, "value", ctx.Context().Value("key"))
	})
}

func TestContext_ID(t *testing.T) {
	t.Run("returns a valid UUID string", func(t *testing.T) {
		ctx := New()
		id := ctx.ID()
		require.NotEmpty(t, id)
		// UUID v4 format: 8-4-4-4-12 hex chars
		require.Regexp(t, `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`, id)
	})

	t.Run("each context gets a unique ID", func(t *testing.T) {
		ctx1 := New()
		ctx2 := New()
		ctx3 := New()
		require.NotEqual(t, ctx1.ID(), ctx2.ID())
		require.NotEqual(t, ctx2.ID(), ctx3.ID())
		require.NotEqual(t, ctx1.ID(), ctx3.ID())
	})

	t.Run("ID is stable across multiple calls", func(t *testing.T) {
		ctx := New()
		require.Equal(t, ctx.ID(), ctx.ID())
	})
}

// =============================================================================
// Assertion Tests - All assertions panic on failure
// =============================================================================

func TestAssertions_Equal(t *testing.T) {
	t.Run("passes for equal values", func(t *testing.T) {
		ctx := New()
		require.NotPanics(t, func() {
			ctx.Assert().Equal(42, 42)
			ctx.Assert().Equal("hello", "hello")
			ctx.Assert().Equal([]int{1, 2, 3}, []int{1, 2, 3})
		})
	})

	t.Run("panics for unequal values", func(t *testing.T) {
		ctx := New()
		require.Panics(t, func() {
			ctx.Assert().Equal(42, 43)
		})
	})
}

func TestAssertions_NotEqual(t *testing.T) {
	t.Run("passes for unequal values", func(t *testing.T) {
		ctx := New()
		require.NotPanics(t, func() {
			ctx.Assert().NotEqual(42, 43)
			ctx.Assert().NotEqual("hello", "world")
		})
	})

	t.Run("panics for equal values", func(t *testing.T) {
		ctx := New()
		require.Panics(t, func() {
			ctx.Assert().NotEqual(42, 42)
		})
	})
}

func TestAssertions_Nil(t *testing.T) {
	t.Run("passes for nil values", func(t *testing.T) {
		ctx := New()
		require.NotPanics(t, func() {
			ctx.Assert().Nil(nil)
			var p *int
			ctx.Assert().Nil(p)
			var s []int
			ctx.Assert().Nil(s)
		})
	})

	t.Run("panics for non-nil values", func(t *testing.T) {
		ctx := New()
		require.Panics(t, func() {
			ctx.Assert().Nil(42)
		})
	})
}

func TestAssertions_NotNil(t *testing.T) {
	t.Run("passes for non-nil values", func(t *testing.T) {
		ctx := New()
		require.NotPanics(t, func() {
			ctx.Assert().NotNil(42)
			ctx.Assert().NotNil("hello")
			ctx.Assert().NotNil([]int{1, 2, 3})
		})
	})

	t.Run("panics for nil values", func(t *testing.T) {
		ctx := New()
		require.Panics(t, func() {
			ctx.Assert().NotNil(nil)
		})
	})
}

func TestAssertions_TrueFalse(t *testing.T) {
	t.Run("True passes for true", func(t *testing.T) {
		ctx := New()
		require.NotPanics(t, func() {
			ctx.Assert().True(true)
			ctx.Assert().True(1 == 1)
		})
	})

	t.Run("True panics for false", func(t *testing.T) {
		ctx := New()
		require.Panics(t, func() {
			ctx.Assert().True(false)
		})
	})

	t.Run("False passes for false", func(t *testing.T) {
		ctx := New()
		require.NotPanics(t, func() {
			ctx.Assert().False(false)
			ctx.Assert().False(1 == 2)
		})
	})

	t.Run("False panics for true", func(t *testing.T) {
		ctx := New()
		require.Panics(t, func() {
			ctx.Assert().False(true)
		})
	})
}

func TestAssertions_Error(t *testing.T) {
	t.Run("NoError passes for nil", func(t *testing.T) {
		ctx := New()
		require.NotPanics(t, func() {
			ctx.Assert().NoError(nil)
		})
	})

	t.Run("NoError panics for error", func(t *testing.T) {
		ctx := New()
		require.Panics(t, func() {
			ctx.Assert().NoError(errors.New("some error"))
		})
	})

	t.Run("Error passes for error", func(t *testing.T) {
		ctx := New()
		require.NotPanics(t, func() {
			ctx.Assert().Error(errors.New("some error"))
		})
	})

	t.Run("Error panics for nil", func(t *testing.T) {
		ctx := New()
		require.Panics(t, func() {
			ctx.Assert().Error(nil)
		})
	})
}

func TestAssertions_ErrorContains(t *testing.T) {
	t.Run("passes when error contains substring", func(t *testing.T) {
		ctx := New()
		require.NotPanics(t, func() {
			ctx.Assert().ErrorContains(errors.New("connection refused"), "refused")
		})
	})

	t.Run("panics when error does not contain substring", func(t *testing.T) {
		ctx := New()
		require.Panics(t, func() {
			ctx.Assert().ErrorContains(errors.New("connection refused"), "timeout")
		})
	})

	t.Run("panics when error is nil", func(t *testing.T) {
		ctx := New()
		require.Panics(t, func() {
			ctx.Assert().ErrorContains(nil, "anything")
		})
	})
}

func TestAssertions_Contains(t *testing.T) {
	t.Run("passes for string containing substring", func(t *testing.T) {
		ctx := New()
		require.NotPanics(t, func() {
			ctx.Assert().Contains("hello world", "world")
		})
	})

	t.Run("passes for slice containing element", func(t *testing.T) {
		ctx := New()
		require.NotPanics(t, func() {
			ctx.Assert().Contains([]int{1, 2, 3}, 2)
		})
	})

	t.Run("passes for map containing key", func(t *testing.T) {
		ctx := New()
		require.NotPanics(t, func() {
			ctx.Assert().Contains(map[string]int{"a": 1}, "a")
		})
	})

	t.Run("panics when not containing", func(t *testing.T) {
		ctx := New()
		require.Panics(t, func() {
			ctx.Assert().Contains("hello", "world")
		})
	})
}

func TestAssertions_NotContains(t *testing.T) {
	t.Run("passes when not containing", func(t *testing.T) {
		ctx := New()
		require.NotPanics(t, func() {
			ctx.Assert().NotContains("hello", "world")
			ctx.Assert().NotContains([]int{1, 2, 3}, 4)
		})
	})

	t.Run("panics when containing", func(t *testing.T) {
		ctx := New()
		require.Panics(t, func() {
			ctx.Assert().NotContains("hello world", "world")
		})
	})
}

func TestAssertions_Len(t *testing.T) {
	t.Run("passes for correct length", func(t *testing.T) {
		ctx := New()
		require.NotPanics(t, func() {
			ctx.Assert().Len([]int{1, 2, 3}, 3)
			ctx.Assert().Len("hello", 5)
			ctx.Assert().Len(map[string]int{"a": 1, "b": 2}, 2)
		})
	})

	t.Run("panics for incorrect length", func(t *testing.T) {
		ctx := New()
		require.Panics(t, func() {
			ctx.Assert().Len([]int{1, 2, 3}, 2)
		})
	})
}

func TestAssertions_EmptyNotEmpty(t *testing.T) {
	t.Run("Empty passes for empty collection", func(t *testing.T) {
		ctx := New()
		require.NotPanics(t, func() {
			ctx.Assert().Empty([]int{})
			ctx.Assert().Empty("")
			ctx.Assert().Empty(map[string]int{})
		})
	})

	t.Run("Empty panics for non-empty collection", func(t *testing.T) {
		ctx := New()
		require.Panics(t, func() {
			ctx.Assert().Empty([]int{1})
		})
	})

	t.Run("NotEmpty passes for non-empty collection", func(t *testing.T) {
		ctx := New()
		require.NotPanics(t, func() {
			ctx.Assert().NotEmpty([]int{1})
			ctx.Assert().NotEmpty("hello")
		})
	})

	t.Run("NotEmpty panics for empty collection", func(t *testing.T) {
		ctx := New()
		require.Panics(t, func() {
			ctx.Assert().NotEmpty([]int{})
		})
	})
}

func TestAssertions_Comparisons(t *testing.T) {
	t.Run("Greater passes when greater", func(t *testing.T) {
		ctx := New()
		require.NotPanics(t, func() {
			ctx.Assert().Greater(5, 3)
			ctx.Assert().Greater(3.14, 2.71)
			ctx.Assert().Greater("b", "a")
		})
	})

	t.Run("Greater panics when not greater", func(t *testing.T) {
		ctx := New()
		require.Panics(t, func() {
			ctx.Assert().Greater(3, 5)
		})
	})

	t.Run("GreaterOrEqual passes when greater or equal", func(t *testing.T) {
		ctx := New()
		require.NotPanics(t, func() {
			ctx.Assert().GreaterOrEqual(5, 3)
			ctx.Assert().GreaterOrEqual(5, 5)
		})
	})

	t.Run("Less passes when less", func(t *testing.T) {
		ctx := New()
		require.NotPanics(t, func() {
			ctx.Assert().Less(3, 5)
			ctx.Assert().Less(2.71, 3.14)
		})
	})

	t.Run("LessOrEqual passes when less or equal", func(t *testing.T) {
		ctx := New()
		require.NotPanics(t, func() {
			ctx.Assert().LessOrEqual(3, 5)
			ctx.Assert().LessOrEqual(5, 5)
		})
	})
}

func TestAssertions_ZeroNotZero(t *testing.T) {
	t.Run("Zero passes for zero values", func(t *testing.T) {
		ctx := New()
		require.NotPanics(t, func() {
			ctx.Assert().Zero(0)
			ctx.Assert().Zero("")
			ctx.Assert().Zero(nil)
			var p *int
			ctx.Assert().Zero(p)
		})
	})

	t.Run("Zero panics for non-zero values", func(t *testing.T) {
		ctx := New()
		require.Panics(t, func() {
			ctx.Assert().Zero(42)
		})
	})

	t.Run("NotZero passes for non-zero values", func(t *testing.T) {
		ctx := New()
		require.NotPanics(t, func() {
			ctx.Assert().NotZero(42)
			ctx.Assert().NotZero("hello")
		})
	})

	t.Run("NotZero panics for zero values", func(t *testing.T) {
		ctx := New()
		require.Panics(t, func() {
			ctx.Assert().NotZero(0)
		})
	})
}

func TestAssertions_SameNotSame(t *testing.T) {
	t.Run("Same passes for same pointer", func(t *testing.T) {
		ctx := New()
		x := &struct{ value int }{value: 42}
		require.NotPanics(t, func() {
			ctx.Assert().Same(x, x)
		})
	})

	t.Run("Same panics for different pointers", func(t *testing.T) {
		ctx := New()
		a := &struct{ value int }{value: 1}
		b := &struct{ value int }{value: 2}
		require.Panics(t, func() {
			ctx.Assert().Same(a, b)
		})
	})

	t.Run("NotSame passes for different pointers", func(t *testing.T) {
		ctx := New()
		a := &struct{ value int }{value: 1}
		b := &struct{ value int }{value: 2}
		require.NotPanics(t, func() {
			ctx.Assert().NotSame(a, b)
		})
	})

	t.Run("NotSame panics for same pointer", func(t *testing.T) {
		ctx := New()
		x := &struct{ value int }{value: 42}
		require.Panics(t, func() {
			ctx.Assert().NotSame(x, x)
		})
	})
}

func TestAssertions_Fail(t *testing.T) {
	t.Run("Fail panics", func(t *testing.T) {
		ctx := New()
		require.Panics(t, func() {
			ctx.Assert().Fail("custom message")
		})
	})
}

func TestAssertions_WithMessage(t *testing.T) {
	t.Run("assertions accept custom messages", func(t *testing.T) {
		ctx := New()
		require.NotPanics(t, func() {
			ctx.Assert().Equal(1, 1, "numbers should match")
			ctx.Assert().True(true, "condition should be true")
			ctx.Assert().NoError(nil, "should have no error")
			ctx.Assert().Equal("hello", "hello", "expected %s", "hello")
		})
	})
}

// =============================================================================
// Default implementations tests
// =============================================================================

func TestDefaultLogger(t *testing.T) {
	t.Run("uses slog by default and does not panic", func(t *testing.T) {
		ctx := New() // Uses slog by default
		require.NotPanics(t, func() {
			ctx.Logger().Debug("test")
			ctx.Logger().Info("test")
			ctx.Logger().Warn("test")
			ctx.Logger().Error("test")
		})
	})
}

func TestPanicT(t *testing.T) {
	t.Run("panics on failure", func(t *testing.T) {
		ctx := New() // Uses panicT by default
		require.Panics(t, func() {
			ctx.Assert().True(false)
		})
	})
}

func TestWithTestingT(t *testing.T) {
	t.Run("uses provided T for assertions", func(t *testing.T) {
		ctx := New(WithTestingT(t))
		require.NotNil(t, ctx.TestingT())
		// Assertions should work without panicking when using *testing.T
		ctx.Assert().Equal(1, 1)
		ctx.Assert().True(true)
	})

	t.Run("TestingT returns the T interface", func(t *testing.T) {
		ctx := New(WithTestingT(t))
		require.Equal(t, t, ctx.TestingT())
	})

	t.Run("default TestingT returns panicT", func(t *testing.T) {
		ctx := New()
		require.NotNil(t, ctx.TestingT())
		_, ok := ctx.TestingT().(*panicT)
		require.True(t, ok)
	})
}
