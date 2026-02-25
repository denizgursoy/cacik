package cacik

import "sort"

// Hooks holds lifecycle hooks for test execution.
// All discovered hook functions are executed, sorted by Order.
type Hooks struct {
	// Order determines execution order (lower = runs first).
	// Default is 0. Hooks with same Order run in discovery order.
	Order int

	// BeforeAll runs once before all scenarios.
	BeforeAll func()

	// AfterAll runs once after all scenarios.
	AfterAll func()

	// BeforeScenario runs before each scenario.
	// The Scenario argument contains the scenario metadata (name, tags, etc.).
	BeforeScenario func(Scenario)

	// AfterScenario runs after each scenario.
	// The error is nil when the scenario passed, non-nil on failure.
	AfterScenario func(Scenario, error)

	// BeforeStep runs before each step.
	// The Step argument contains the step metadata (keyword, text, etc.).
	BeforeStep func(Step)

	// AfterStep runs after each step.
	// The error is nil when the step passed, non-nil on failure.
	AfterStep func(Step, error)
}

// SortHooks sorts hooks by Order (ascending).
// Hooks with the same Order maintain their relative order (stable sort).
func SortHooks(hooks []*Hooks) []*Hooks {
	sorted := make([]*Hooks, len(hooks))
	copy(sorted, hooks)

	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Order < sorted[j].Order
	})

	return sorted
}

// HookExecutor manages execution of multiple hooks.
type HookExecutor struct {
	hooks []*Hooks // sorted by Order
}

// NewHookExecutor creates a new HookExecutor with sorted hooks.
func NewHookExecutor(hooks ...*Hooks) *HookExecutor {
	// Filter out nil hooks
	validHooks := make([]*Hooks, 0, len(hooks))
	for _, h := range hooks {
		if h != nil {
			validHooks = append(validHooks, h)
		}
	}

	return &HookExecutor{
		hooks: SortHooks(validHooks),
	}
}

// ExecuteBeforeAll executes all BeforeAll hooks in order.
func (e *HookExecutor) ExecuteBeforeAll() {
	for _, h := range e.hooks {
		if h.BeforeAll != nil {
			h.BeforeAll()
		}
	}
}

// ExecuteAfterAll executes all AfterAll hooks in order (same as BeforeAll).
func (e *HookExecutor) ExecuteAfterAll() {
	for _, h := range e.hooks {
		if h.AfterAll != nil {
			h.AfterAll()
		}
	}
}

// ExecuteBeforeScenario executes all BeforeScenario hooks in order.
func (e *HookExecutor) ExecuteBeforeScenario(scenario Scenario) {
	for _, h := range e.hooks {
		if h.BeforeScenario != nil {
			h.BeforeScenario(scenario)
		}
	}
}

// ExecuteAfterScenario executes all AfterScenario hooks in order.
func (e *HookExecutor) ExecuteAfterScenario(scenario Scenario, err error) {
	for _, h := range e.hooks {
		if h.AfterScenario != nil {
			h.AfterScenario(scenario, err)
		}
	}
}

// ExecuteBeforeStep executes all BeforeStep hooks in order.
func (e *HookExecutor) ExecuteBeforeStep(step Step) {
	for _, h := range e.hooks {
		if h.BeforeStep != nil {
			h.BeforeStep(step)
		}
	}
}

// ExecuteAfterStep executes all AfterStep hooks in order.
func (e *HookExecutor) ExecuteAfterStep(step Step, err error) {
	for _, h := range e.hooks {
		if h.AfterStep != nil {
			h.AfterStep(step, err)
		}
	}
}
