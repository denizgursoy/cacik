package cacik

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// =============================================================================
// SortHooks Tests
// =============================================================================

func TestSortHooks(t *testing.T) {
	t.Run("sorts hooks by Order ascending", func(t *testing.T) {
		h1 := &Hooks{Order: 3}
		h2 := &Hooks{Order: 1}
		h3 := &Hooks{Order: 2}

		sorted := SortHooks([]*Hooks{h1, h2, h3})
		require.Equal(t, 1, sorted[0].Order)
		require.Equal(t, 2, sorted[1].Order)
		require.Equal(t, 3, sorted[2].Order)
	})

	t.Run("stable sort preserves order for equal Order values", func(t *testing.T) {
		var callOrder []string
		h1 := &Hooks{Order: 0, BeforeAll: func() { callOrder = append(callOrder, "first") }}
		h2 := &Hooks{Order: 0, BeforeAll: func() { callOrder = append(callOrder, "second") }}

		sorted := SortHooks([]*Hooks{h1, h2})
		// Both have Order 0, so original order is preserved
		sorted[0].BeforeAll()
		sorted[1].BeforeAll()
		require.Equal(t, []string{"first", "second"}, callOrder)
	})

	t.Run("does not modify original slice", func(t *testing.T) {
		h1 := &Hooks{Order: 2}
		h2 := &Hooks{Order: 1}
		original := []*Hooks{h1, h2}

		sorted := SortHooks(original)
		require.Equal(t, 2, original[0].Order) // original unchanged
		require.Equal(t, 1, sorted[0].Order)   // sorted is different
	})
}

// =============================================================================
// NewHookExecutor Tests
// =============================================================================

func TestNewHookExecutor(t *testing.T) {
	t.Run("filters out nil hooks", func(t *testing.T) {
		h := &Hooks{Order: 1}
		exec := NewHookExecutor(nil, h, nil)

		// Should only have one hook — verify by calling a hook
		var count int
		h.BeforeAll = func() { count++ }
		exec.ExecuteBeforeAll()
		require.Equal(t, 1, count)
	})

	t.Run("empty hooks list", func(t *testing.T) {
		exec := NewHookExecutor()
		// Should not panic
		exec.ExecuteBeforeAll()
		exec.ExecuteAfterAll()
	})
}

// =============================================================================
// BeforeAll / AfterAll Tests
// =============================================================================

func TestExecuteBeforeAll(t *testing.T) {
	t.Run("calls all BeforeAll hooks in order", func(t *testing.T) {
		var order []int
		h1 := &Hooks{Order: 2, BeforeAll: func() { order = append(order, 2) }}
		h2 := &Hooks{Order: 1, BeforeAll: func() { order = append(order, 1) }}
		h3 := &Hooks{Order: 3, BeforeAll: func() { order = append(order, 3) }}

		exec := NewHookExecutor(h1, h2, h3)
		exec.ExecuteBeforeAll()

		require.Equal(t, []int{1, 2, 3}, order)
	})

	t.Run("skips hooks without BeforeAll", func(t *testing.T) {
		var called bool
		h1 := &Hooks{Order: 1} // no BeforeAll
		h2 := &Hooks{Order: 2, BeforeAll: func() { called = true }}

		exec := NewHookExecutor(h1, h2)
		exec.ExecuteBeforeAll()

		require.True(t, called)
	})
}

func TestExecuteAfterAll(t *testing.T) {
	t.Run("calls all AfterAll hooks in order", func(t *testing.T) {
		var order []int
		h1 := &Hooks{Order: 2, AfterAll: func() { order = append(order, 2) }}
		h2 := &Hooks{Order: 1, AfterAll: func() { order = append(order, 1) }}

		exec := NewHookExecutor(h1, h2)
		exec.ExecuteAfterAll()

		require.Equal(t, []int{1, 2}, order)
	})
}

// =============================================================================
// BeforeScenario / AfterScenario Tests
// =============================================================================

func TestExecuteBeforeScenario(t *testing.T) {
	t.Run("passes correct Scenario to hooks", func(t *testing.T) {
		var received Scenario
		h := &Hooks{
			BeforeScenario: func(s Scenario) { received = s },
		}

		exec := NewHookExecutor(h)
		s := Scenario{
			Name:    "Login test",
			Tags:    []string{"@smoke", "@auth"},
			Keyword: "Scenario",
			Line:    42,
		}
		exec.ExecuteBeforeScenario(s)

		require.Equal(t, "Login test", received.Name)
		require.Equal(t, []string{"@smoke", "@auth"}, received.Tags)
		require.Equal(t, "Scenario", received.Keyword)
		require.Equal(t, int64(42), received.Line)
	})

	t.Run("calls multiple hooks in order", func(t *testing.T) {
		var order []int
		h1 := &Hooks{Order: 2, BeforeScenario: func(s Scenario) { order = append(order, 2) }}
		h2 := &Hooks{Order: 1, BeforeScenario: func(s Scenario) { order = append(order, 1) }}

		exec := NewHookExecutor(h1, h2)
		exec.ExecuteBeforeScenario(Scenario{Name: "test"})

		require.Equal(t, []int{1, 2}, order)
	})

	t.Run("skips hooks without BeforeScenario", func(t *testing.T) {
		var called bool
		h1 := &Hooks{Order: 1} // no BeforeScenario
		h2 := &Hooks{Order: 2, BeforeScenario: func(s Scenario) { called = true }}

		exec := NewHookExecutor(h1, h2)
		exec.ExecuteBeforeScenario(Scenario{Name: "test"})

		require.True(t, called)
	})
}

func TestExecuteAfterScenario(t *testing.T) {
	t.Run("passes Scenario and nil error on success", func(t *testing.T) {
		var receivedScenario Scenario
		var receivedErr error
		h := &Hooks{
			AfterScenario: func(s Scenario, err error) {
				receivedScenario = s
				receivedErr = err
			},
		}

		exec := NewHookExecutor(h)
		s := Scenario{Name: "Passing scenario", Keyword: "Scenario"}
		exec.ExecuteAfterScenario(s, nil)

		require.Equal(t, "Passing scenario", receivedScenario.Name)
		require.NoError(t, receivedErr)
	})

	t.Run("passes Scenario and non-nil error on failure", func(t *testing.T) {
		var receivedErr error
		h := &Hooks{
			AfterScenario: func(s Scenario, err error) { receivedErr = err },
		}

		exec := NewHookExecutor(h)
		stepErr := errors.New("step failed: expected 200 got 500")
		exec.ExecuteAfterScenario(Scenario{Name: "Failing scenario"}, stepErr)

		require.Error(t, receivedErr)
		require.Contains(t, receivedErr.Error(), "step failed")
	})

	t.Run("calls multiple hooks in order", func(t *testing.T) {
		var order []int
		h1 := &Hooks{Order: 3, AfterScenario: func(s Scenario, err error) { order = append(order, 3) }}
		h2 := &Hooks{Order: 1, AfterScenario: func(s Scenario, err error) { order = append(order, 1) }}
		h3 := &Hooks{Order: 2, AfterScenario: func(s Scenario, err error) { order = append(order, 2) }}

		exec := NewHookExecutor(h1, h2, h3)
		exec.ExecuteAfterScenario(Scenario{Name: "test"}, nil)

		require.Equal(t, []int{1, 2, 3}, order)
	})
}

// =============================================================================
// BeforeStep / AfterStep Tests
// =============================================================================

func TestExecuteBeforeStep(t *testing.T) {
	t.Run("passes correct Step to hooks", func(t *testing.T) {
		var received Step
		h := &Hooks{
			BeforeStep: func(s Step) { received = s },
		}

		exec := NewHookExecutor(h)
		step := Step{Keyword: "Given ", Text: "the user is logged in", Line: 10}
		exec.ExecuteBeforeStep(step)

		require.Equal(t, "Given ", received.Keyword)
		require.Equal(t, "the user is logged in", received.Text)
		require.Equal(t, int64(10), received.Line)
	})

	t.Run("calls multiple hooks in order", func(t *testing.T) {
		var order []int
		h1 := &Hooks{Order: 2, BeforeStep: func(s Step) { order = append(order, 2) }}
		h2 := &Hooks{Order: 1, BeforeStep: func(s Step) { order = append(order, 1) }}

		exec := NewHookExecutor(h1, h2)
		exec.ExecuteBeforeStep(Step{Keyword: "When ", Text: "action"})

		require.Equal(t, []int{1, 2}, order)
	})

	t.Run("skips hooks without BeforeStep", func(t *testing.T) {
		var called bool
		h1 := &Hooks{Order: 1} // no BeforeStep
		h2 := &Hooks{Order: 2, BeforeStep: func(s Step) { called = true }}

		exec := NewHookExecutor(h1, h2)
		exec.ExecuteBeforeStep(Step{Keyword: "Then ", Text: "verify"})

		require.True(t, called)
	})
}

func TestExecuteAfterStep(t *testing.T) {
	t.Run("passes Step and nil error on success", func(t *testing.T) {
		var receivedStep Step
		var receivedErr error
		h := &Hooks{
			AfterStep: func(s Step, err error) {
				receivedStep = s
				receivedErr = err
			},
		}

		exec := NewHookExecutor(h)
		step := Step{Keyword: "Given ", Text: "the user exists"}
		exec.ExecuteAfterStep(step, nil)

		require.Equal(t, "Given ", receivedStep.Keyword)
		require.Equal(t, "the user exists", receivedStep.Text)
		require.NoError(t, receivedErr)
	})

	t.Run("passes Step and non-nil error on failure", func(t *testing.T) {
		var receivedStep Step
		var receivedErr error
		h := &Hooks{
			AfterStep: func(s Step, err error) {
				receivedStep = s
				receivedErr = err
			},
		}

		exec := NewHookExecutor(h)
		step := Step{Keyword: "Then ", Text: "the status is 200"}
		stepErr := errors.New("assertion failed: expected 200 got 500")
		exec.ExecuteAfterStep(step, stepErr)

		require.Equal(t, "Then ", receivedStep.Keyword)
		require.Equal(t, "the status is 200", receivedStep.Text)
		require.Error(t, receivedErr)
		require.Contains(t, receivedErr.Error(), "assertion failed")
	})

	t.Run("calls multiple hooks in order", func(t *testing.T) {
		var order []int
		h1 := &Hooks{Order: 3, AfterStep: func(s Step, err error) { order = append(order, 3) }}
		h2 := &Hooks{Order: 1, AfterStep: func(s Step, err error) { order = append(order, 1) }}
		h3 := &Hooks{Order: 2, AfterStep: func(s Step, err error) { order = append(order, 2) }}

		exec := NewHookExecutor(h1, h2, h3)
		exec.ExecuteAfterStep(Step{Keyword: "And ", Text: "verify"}, nil)

		require.Equal(t, []int{1, 2, 3}, order)
	})
}

// =============================================================================
// Mixed Hooks Tests — hooks with partial fields set
// =============================================================================

func TestPartialHooks(t *testing.T) {
	t.Run("hook with only BeforeScenario does not panic on other calls", func(t *testing.T) {
		var called bool
		h := &Hooks{
			BeforeScenario: func(s Scenario) { called = true },
			// All other fields nil
		}

		exec := NewHookExecutor(h)
		exec.ExecuteBeforeAll()
		exec.ExecuteAfterAll()
		exec.ExecuteBeforeScenario(Scenario{Name: "test"})
		exec.ExecuteAfterScenario(Scenario{Name: "test"}, nil)
		exec.ExecuteBeforeStep(Step{Keyword: "Given ", Text: "x"})
		exec.ExecuteAfterStep(Step{Keyword: "Given ", Text: "x"}, nil)

		require.True(t, called)
	})

	t.Run("hook with only AfterStep does not panic on other calls", func(t *testing.T) {
		var receivedErr error
		h := &Hooks{
			AfterStep: func(s Step, err error) { receivedErr = err },
		}

		exec := NewHookExecutor(h)
		exec.ExecuteBeforeAll()
		exec.ExecuteAfterAll()
		exec.ExecuteBeforeScenario(Scenario{Name: "test"})
		exec.ExecuteAfterScenario(Scenario{Name: "test"}, nil)
		exec.ExecuteBeforeStep(Step{Keyword: "When ", Text: "y"})

		stepErr := errors.New("failed")
		exec.ExecuteAfterStep(Step{Keyword: "When ", Text: "y"}, stepErr)

		require.Equal(t, stepErr, receivedErr)
	})
}

// =============================================================================
// Multiple Hooks Integration
// =============================================================================

func TestMultipleHooksIntegration(t *testing.T) {
	t.Run("full lifecycle ordering across multiple hooks", func(t *testing.T) {
		var events []string

		h1 := &Hooks{
			Order:          1,
			BeforeAll:      func() { events = append(events, "h1:beforeAll") },
			AfterAll:       func() { events = append(events, "h1:afterAll") },
			BeforeScenario: func(s Scenario) { events = append(events, "h1:beforeScenario") },
			AfterScenario:  func(s Scenario, err error) { events = append(events, "h1:afterScenario") },
			BeforeStep:     func(s Step) { events = append(events, "h1:beforeStep") },
			AfterStep:      func(s Step, err error) { events = append(events, "h1:afterStep") },
		}

		h2 := &Hooks{
			Order:          2,
			BeforeAll:      func() { events = append(events, "h2:beforeAll") },
			AfterAll:       func() { events = append(events, "h2:afterAll") },
			BeforeScenario: func(s Scenario) { events = append(events, "h2:beforeScenario") },
			AfterScenario:  func(s Scenario, err error) { events = append(events, "h2:afterScenario") },
			BeforeStep:     func(s Step) { events = append(events, "h2:beforeStep") },
			AfterStep:      func(s Step, err error) { events = append(events, "h2:afterStep") },
		}

		exec := NewHookExecutor(h2, h1) // intentionally reversed to verify sorting

		scenario := Scenario{Name: "Login", Tags: []string{"@smoke"}}
		step := Step{Keyword: "Given ", Text: "user exists"}

		exec.ExecuteBeforeAll()
		exec.ExecuteBeforeScenario(scenario)
		exec.ExecuteBeforeStep(step)
		exec.ExecuteAfterStep(step, nil)
		exec.ExecuteAfterScenario(scenario, nil)
		exec.ExecuteAfterAll()

		expected := []string{
			"h1:beforeAll",
			"h2:beforeAll",
			"h1:beforeScenario",
			"h2:beforeScenario",
			"h1:beforeStep",
			"h2:beforeStep",
			"h1:afterStep",
			"h2:afterStep",
			"h1:afterScenario",
			"h2:afterScenario",
			"h1:afterAll",
			"h2:afterAll",
		}
		require.Equal(t, expected, events)
	})
}
