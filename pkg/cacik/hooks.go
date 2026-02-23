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

	// BeforeStep runs before each step.
	BeforeStep func()

	// AfterStep runs after each step.
	AfterStep func()
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

// ExecuteBeforeStep executes all BeforeStep hooks in order.
func (e *HookExecutor) ExecuteBeforeStep() {
	for _, h := range e.hooks {
		if h.BeforeStep != nil {
			h.BeforeStep()
		}
	}
}

// ExecuteAfterStep executes all AfterStep hooks in order.
func (e *HookExecutor) ExecuteAfterStep() {
	for _, h := range e.hooks {
		if h.AfterStep != nil {
			h.AfterStep()
		}
	}
}
