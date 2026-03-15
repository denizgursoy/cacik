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

// =============================================================================
// Tagged Hooks Tests
// =============================================================================

func TestNewHookExecutor_PanicsOnInvalidTagExpression(t *testing.T) {
	t.Run("invalid tag expression panics", func(t *testing.T) {
		h := &Hooks{
			Tags:           "@smoke and and",
			BeforeScenario: func(s Scenario) {},
		}
		require.Panics(t, func() {
			NewHookExecutor(h)
		})
	})

	t.Run("valid tag expression does not panic", func(t *testing.T) {
		h := &Hooks{
			Tags:           "@smoke and @fast",
			BeforeScenario: func(s Scenario) {},
		}
		require.NotPanics(t, func() {
			NewHookExecutor(h)
		})
	})

	t.Run("empty tags does not panic", func(t *testing.T) {
		h := &Hooks{
			BeforeScenario: func(s Scenario) {},
		}
		require.NotPanics(t, func() {
			NewHookExecutor(h)
		})
	})
}

func TestTaggedBeforeScenario(t *testing.T) {
	t.Run("hook with @smoke tag fires for @smoke scenario", func(t *testing.T) {
		var called bool
		h := &Hooks{
			Tags:           "@smoke",
			BeforeScenario: func(s Scenario) { called = true },
		}
		exec := NewHookExecutor(h)
		exec.ExecuteBeforeScenario(Scenario{Name: "test", Tags: []string{"@smoke", "@fast"}})
		require.True(t, called)
	})

	t.Run("hook with @smoke tag does NOT fire for non-smoke scenario", func(t *testing.T) {
		var called bool
		h := &Hooks{
			Tags:           "@smoke",
			BeforeScenario: func(s Scenario) { called = true },
		}
		exec := NewHookExecutor(h)
		exec.ExecuteBeforeScenario(Scenario{Name: "test", Tags: []string{"@regression"}})
		require.False(t, called)
	})

	t.Run("hook with @smoke tag does NOT fire for untagged scenario", func(t *testing.T) {
		var called bool
		h := &Hooks{
			Tags:           "@smoke",
			BeforeScenario: func(s Scenario) { called = true },
		}
		exec := NewHookExecutor(h)
		exec.ExecuteBeforeScenario(Scenario{Name: "test"})
		require.False(t, called)
	})

	t.Run("hook with empty Tags fires for all scenarios (backward compat)", func(t *testing.T) {
		var count int
		h := &Hooks{
			BeforeScenario: func(s Scenario) { count++ },
		}
		exec := NewHookExecutor(h)
		exec.ExecuteBeforeScenario(Scenario{Name: "tagged", Tags: []string{"@smoke"}})
		exec.ExecuteBeforeScenario(Scenario{Name: "untagged"})
		require.Equal(t, 2, count)
	})

	t.Run("hook with compound expression @smoke and @fast", func(t *testing.T) {
		var called bool
		h := &Hooks{
			Tags:           "@smoke and @fast",
			BeforeScenario: func(s Scenario) { called = true },
		}
		exec := NewHookExecutor(h)

		// Only @smoke — should NOT fire
		exec.ExecuteBeforeScenario(Scenario{Name: "test", Tags: []string{"@smoke"}})
		require.False(t, called)

		// Both @smoke and @fast — should fire
		exec.ExecuteBeforeScenario(Scenario{Name: "test", Tags: []string{"@smoke", "@fast"}})
		require.True(t, called)
	})

	t.Run("hook with not @slow", func(t *testing.T) {
		var count int
		h := &Hooks{
			Tags:           "not @slow",
			BeforeScenario: func(s Scenario) { count++ },
		}
		exec := NewHookExecutor(h)

		exec.ExecuteBeforeScenario(Scenario{Name: "fast", Tags: []string{"@smoke"}})
		require.Equal(t, 1, count) // fires

		exec.ExecuteBeforeScenario(Scenario{Name: "slow", Tags: []string{"@slow"}})
		require.Equal(t, 1, count) // does NOT fire

		exec.ExecuteBeforeScenario(Scenario{Name: "untagged"})
		require.Equal(t, 2, count) // fires (no @slow)
	})

	t.Run("hook with @smoke or @critical", func(t *testing.T) {
		var count int
		h := &Hooks{
			Tags:           "@smoke or @critical",
			BeforeScenario: func(s Scenario) { count++ },
		}
		exec := NewHookExecutor(h)

		exec.ExecuteBeforeScenario(Scenario{Name: "s1", Tags: []string{"@smoke"}})
		require.Equal(t, 1, count)

		exec.ExecuteBeforeScenario(Scenario{Name: "s2", Tags: []string{"@critical"}})
		require.Equal(t, 2, count)

		exec.ExecuteBeforeScenario(Scenario{Name: "s3", Tags: []string{"@regression"}})
		require.Equal(t, 2, count) // does NOT fire
	})
}

func TestTaggedAfterScenario(t *testing.T) {
	t.Run("hook with @smoke tag fires for matching scenario", func(t *testing.T) {
		var received Scenario
		h := &Hooks{
			Tags:          "@smoke",
			AfterScenario: func(s Scenario, err error) { received = s },
		}
		exec := NewHookExecutor(h)
		exec.ExecuteAfterScenario(Scenario{Name: "match", Tags: []string{"@smoke"}}, nil)
		require.Equal(t, "match", received.Name)
	})

	t.Run("hook with @smoke tag skips non-matching scenario", func(t *testing.T) {
		var called bool
		h := &Hooks{
			Tags:          "@smoke",
			AfterScenario: func(s Scenario, err error) { called = true },
		}
		exec := NewHookExecutor(h)
		exec.ExecuteAfterScenario(Scenario{Name: "nomatch", Tags: []string{"@regression"}}, nil)
		require.False(t, called)
	})
}

func TestTaggedBeforeStep(t *testing.T) {
	t.Run("step hooks use scenario tags from SetScenarioTags", func(t *testing.T) {
		var called bool
		h := &Hooks{
			Tags:       "@smoke",
			BeforeStep: func(s Step) { called = true },
		}
		exec := NewHookExecutor(h)

		// Set scenario tags to include @smoke
		exec.SetScenarioTags([]string{"@smoke", "@fast"})
		exec.ExecuteBeforeStep(Step{Keyword: "Given ", Text: "a user"})
		require.True(t, called)
	})

	t.Run("step hooks do NOT fire when scenario tags do not match", func(t *testing.T) {
		var called bool
		h := &Hooks{
			Tags:       "@smoke",
			BeforeStep: func(s Step) { called = true },
		}
		exec := NewHookExecutor(h)

		exec.SetScenarioTags([]string{"@regression"})
		exec.ExecuteBeforeStep(Step{Keyword: "Given ", Text: "a user"})
		require.False(t, called)
	})

	t.Run("step hooks fire when no scenario tags set and hook has no tag filter", func(t *testing.T) {
		var called bool
		h := &Hooks{
			BeforeStep: func(s Step) { called = true },
		}
		exec := NewHookExecutor(h)
		// No SetScenarioTags call — scenarioTags is nil
		exec.ExecuteBeforeStep(Step{Keyword: "Given ", Text: "a user"})
		require.True(t, called)
	})
}

func TestTaggedAfterStep(t *testing.T) {
	t.Run("step hooks use scenario tags from SetScenarioTags", func(t *testing.T) {
		var called bool
		h := &Hooks{
			Tags:      "@smoke",
			AfterStep: func(s Step, err error) { called = true },
		}
		exec := NewHookExecutor(h)

		exec.SetScenarioTags([]string{"@smoke"})
		exec.ExecuteAfterStep(Step{Keyword: "Then ", Text: "verify"}, nil)
		require.True(t, called)
	})

	t.Run("ClearScenarioTags prevents step hook from firing", func(t *testing.T) {
		var count int
		h := &Hooks{
			Tags:      "@smoke",
			AfterStep: func(s Step, err error) { count++ },
		}
		exec := NewHookExecutor(h)

		exec.SetScenarioTags([]string{"@smoke"})
		exec.ExecuteAfterStep(Step{Keyword: "Then ", Text: "verify"}, nil)
		require.Equal(t, 1, count)

		exec.ClearScenarioTags()
		exec.ExecuteAfterStep(Step{Keyword: "Then ", Text: "verify"}, nil)
		require.Equal(t, 1, count) // did NOT fire after clear
	})
}

func TestBeforeAllAfterAllTagFiltering(t *testing.T) {
	t.Run("BeforeAll fires when at least one scenario matches tag", func(t *testing.T) {
		var called bool
		h := &Hooks{
			Tags:      "@smoke",
			BeforeAll: func() { called = true },
		}
		exec := NewHookExecutor(h)
		exec.SetAllScenarioTags([][]string{
			{"@smoke", "@fast"},
			{"@regression"},
		})
		exec.ExecuteBeforeAll()
		require.True(t, called)
	})

	t.Run("BeforeAll does NOT fire when no scenario matches tag", func(t *testing.T) {
		var called bool
		h := &Hooks{
			Tags:      "@smoke",
			BeforeAll: func() { called = true },
		}
		exec := NewHookExecutor(h)
		exec.SetAllScenarioTags([][]string{
			{"@regression"},
			{"@api"},
		})
		exec.ExecuteBeforeAll()
		require.False(t, called)
	})

	t.Run("AfterAll fires when at least one scenario matches tag", func(t *testing.T) {
		var called bool
		h := &Hooks{
			Tags:     "@smoke",
			AfterAll: func() { called = true },
		}
		exec := NewHookExecutor(h)
		exec.SetAllScenarioTags([][]string{
			{"@smoke"},
		})
		exec.ExecuteAfterAll()
		require.True(t, called)
	})

	t.Run("AfterAll does NOT fire when no scenario matches tag", func(t *testing.T) {
		var called bool
		h := &Hooks{
			Tags:     "@smoke",
			AfterAll: func() { called = true },
		}
		exec := NewHookExecutor(h)
		exec.SetAllScenarioTags([][]string{
			{"@regression"},
		})
		exec.ExecuteAfterAll()
		require.False(t, called)
	})

	t.Run("untagged BeforeAll always fires regardless of scenario tags", func(t *testing.T) {
		var called bool
		h := &Hooks{
			BeforeAll: func() { called = true },
		}
		exec := NewHookExecutor(h)
		// No SetAllScenarioTags — allScenarioTags is nil
		exec.ExecuteBeforeAll()
		require.True(t, called)
	})

	t.Run("untagged AfterAll always fires regardless of scenario tags", func(t *testing.T) {
		var called bool
		h := &Hooks{
			AfterAll: func() { called = true },
		}
		exec := NewHookExecutor(h)
		exec.SetAllScenarioTags([][]string{{"@anything"}})
		exec.ExecuteAfterAll()
		require.True(t, called)
	})

	t.Run("compound tag expression on BeforeAll", func(t *testing.T) {
		var called bool
		h := &Hooks{
			Tags:      "@smoke and @fast",
			BeforeAll: func() { called = true },
		}
		exec := NewHookExecutor(h)

		// No scenario has both @smoke and @fast
		exec.SetAllScenarioTags([][]string{
			{"@smoke"},
			{"@fast"},
		})
		exec.ExecuteBeforeAll()
		require.False(t, called)

		// One scenario has both
		exec.SetAllScenarioTags([][]string{
			{"@smoke", "@fast"},
			{"@regression"},
		})
		exec.ExecuteBeforeAll()
		require.True(t, called)
	})

	t.Run("not expression on BeforeAll", func(t *testing.T) {
		var called bool
		h := &Hooks{
			Tags:      "not @slow",
			BeforeAll: func() { called = true },
		}
		exec := NewHookExecutor(h)

		// All scenarios are @slow
		exec.SetAllScenarioTags([][]string{
			{"@slow"},
			{"@slow", "@regression"},
		})
		exec.ExecuteBeforeAll()
		require.False(t, called)

		// One scenario is not @slow
		exec.SetAllScenarioTags([][]string{
			{"@slow"},
			{"@fast"},
		})
		exec.ExecuteBeforeAll()
		require.True(t, called)
	})
}

func TestTaggedHooksIntegration(t *testing.T) {
	t.Run("mixed tagged and untagged hooks with multiple scenarios", func(t *testing.T) {
		var events []string

		smokeHook := &Hooks{
			Order: 1,
			Tags:  "@smoke",
			BeforeScenario: func(s Scenario) {
				events = append(events, "smoke:beforeScenario:"+s.Name)
			},
			AfterScenario: func(s Scenario, err error) {
				events = append(events, "smoke:afterScenario:"+s.Name)
			},
			BeforeStep: func(s Step) {
				events = append(events, "smoke:beforeStep")
			},
			AfterStep: func(s Step, err error) {
				events = append(events, "smoke:afterStep")
			},
		}

		globalHook := &Hooks{
			Order: 2,
			BeforeScenario: func(s Scenario) {
				events = append(events, "global:beforeScenario:"+s.Name)
			},
			AfterScenario: func(s Scenario, err error) {
				events = append(events, "global:afterScenario:"+s.Name)
			},
			BeforeStep: func(s Step) {
				events = append(events, "global:beforeStep")
			},
			AfterStep: func(s Step, err error) {
				events = append(events, "global:afterStep")
			},
		}

		exec := NewHookExecutor(smokeHook, globalHook)

		// Scenario 1: has @smoke — both hooks fire
		smokeScenario := Scenario{Name: "S1", Tags: []string{"@smoke"}}
		exec.SetScenarioTags(smokeScenario.Tags)
		exec.ExecuteBeforeScenario(smokeScenario)
		exec.ExecuteBeforeStep(Step{Keyword: "Given ", Text: "step1"})
		exec.ExecuteAfterStep(Step{Keyword: "Given ", Text: "step1"}, nil)
		exec.ExecuteAfterScenario(smokeScenario, nil)
		exec.ClearScenarioTags()

		// Scenario 2: no @smoke — only global hook fires
		regressionScenario := Scenario{Name: "S2", Tags: []string{"@regression"}}
		exec.SetScenarioTags(regressionScenario.Tags)
		exec.ExecuteBeforeScenario(regressionScenario)
		exec.ExecuteBeforeStep(Step{Keyword: "When ", Text: "step2"})
		exec.ExecuteAfterStep(Step{Keyword: "When ", Text: "step2"}, nil)
		exec.ExecuteAfterScenario(regressionScenario, nil)
		exec.ClearScenarioTags()

		expected := []string{
			// S1 (@smoke) — both hooks
			"smoke:beforeScenario:S1",
			"global:beforeScenario:S1",
			"smoke:beforeStep",
			"global:beforeStep",
			"smoke:afterStep",
			"global:afterStep",
			"smoke:afterScenario:S1",
			"global:afterScenario:S1",
			// S2 (@regression) — only global
			"global:beforeScenario:S2",
			"global:beforeStep",
			"global:afterStep",
			"global:afterScenario:S2",
		}
		require.Equal(t, expected, events)
	})
}
