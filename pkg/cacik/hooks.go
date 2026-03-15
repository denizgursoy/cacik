package cacik

import (
	"fmt"
	"sort"

	tagexpressions "github.com/cucumber/tag-expressions/go/v6"
)

// Hooks holds lifecycle hooks for test execution.
// All discovered hook functions are executed, sorted by Order.
type Hooks struct {
	// Order determines execution order (lower = runs first).
	// Default is 0. Hooks with same Order run in discovery order.
	Order int

	// Tags is an optional Cucumber tag expression that filters when this
	// hook fires. When non-empty, all hook types are filtered:
	// BeforeScenario/AfterScenario and BeforeStep/AfterStep fire only
	// for matching scenarios; BeforeAll/AfterAll fire only if at least
	// one scenario in the run matches the tag expression.
	//
	// Examples:
	//   Tags: "@smoke"              — fires only for scenarios tagged @smoke
	//   Tags: "@smoke and @fast"    — requires both tags
	//   Tags: "not @slow"           — skips scenarios tagged @slow
	//   Tags: "@smoke or @critical" — fires for either tag
	Tags string

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

// hookEntry pairs a Hooks with its pre-parsed tag evaluator (nil = no filter).
type hookEntry struct {
	hooks     *Hooks
	evaluator tagexpressions.Evaluatable // nil when Tags is empty
}

// HookExecutor manages execution of multiple hooks.
type HookExecutor struct {
	entries         []hookEntry // sorted by Order
	scenarioTags    []string    // tags of the currently executing scenario
	allScenarioTags [][]string  // tags of all scenarios in the run (for BeforeAll/AfterAll filtering)
}

// NewHookExecutor creates a new HookExecutor with sorted hooks.
// Panics if any hook has an invalid tag expression.
func NewHookExecutor(hooks ...*Hooks) *HookExecutor {
	// Filter out nil hooks
	validHooks := make([]*Hooks, 0, len(hooks))
	for _, h := range hooks {
		if h != nil {
			validHooks = append(validHooks, h)
		}
	}

	sorted := SortHooks(validHooks)

	entries := make([]hookEntry, len(sorted))
	for i, h := range sorted {
		var eval tagexpressions.Evaluatable
		if h.Tags != "" {
			var err error
			eval, err = tagexpressions.Parse(h.Tags)
			if err != nil {
				panic(fmt.Sprintf("cacik: invalid tag expression %q on Hooks (Order=%d): %v", h.Tags, h.Order, err))
			}
		}
		entries[i] = hookEntry{hooks: h, evaluator: eval}
	}

	return &HookExecutor{
		entries: entries,
	}
}

// SetScenarioTags stores the current scenario's tags for step-level hook filtering.
// Must be called before ExecuteBeforeScenario.
func (e *HookExecutor) SetScenarioTags(tags []string) {
	e.scenarioTags = tags
}

// ClearScenarioTags removes the stored scenario tags after scenario completion.
func (e *HookExecutor) ClearScenarioTags() {
	e.scenarioTags = nil
}

// SetAllScenarioTags stores the tag sets for every scenario in the run.
// Used by ExecuteBeforeAll/ExecuteAfterAll to decide whether a tagged hook
// should fire. Must be called before ExecuteBeforeAll.
func (e *HookExecutor) SetAllScenarioTags(tags [][]string) {
	e.allScenarioTags = tags
}

// matchesTags returns true if the hook should fire for the given tags.
// Hooks with no tag expression always match.
func (he *hookEntry) matchesTags(tags []string) bool {
	if he.evaluator == nil {
		return true
	}
	return he.evaluator.Evaluate(tags)
}

// matchesAnyScenario returns true if the hook's tag expression matches at
// least one scenario in the provided tag sets. Hooks with no tag expression
// always return true.
func (he *hookEntry) matchesAnyScenario(allTags [][]string) bool {
	if he.evaluator == nil {
		return true
	}
	for _, tags := range allTags {
		if he.evaluator.Evaluate(tags) {
			return true
		}
	}
	return false
}

// ExecuteBeforeAll executes BeforeAll hooks in order.
// Tagged hooks only fire if at least one scenario in the run matches the
// tag expression. Hooks with no Tags always fire.
func (e *HookExecutor) ExecuteBeforeAll() {
	for _, he := range e.entries {
		if he.hooks.BeforeAll != nil && he.matchesAnyScenario(e.allScenarioTags) {
			he.hooks.BeforeAll()
		}
	}
}

// ExecuteAfterAll executes AfterAll hooks in order.
// Tagged hooks only fire if at least one scenario in the run matches the
// tag expression. Hooks with no Tags always fire.
func (e *HookExecutor) ExecuteAfterAll() {
	for _, he := range e.entries {
		if he.hooks.AfterAll != nil && he.matchesAnyScenario(e.allScenarioTags) {
			he.hooks.AfterAll()
		}
	}
}

// ExecuteBeforeScenario executes matching BeforeScenario hooks in order.
func (e *HookExecutor) ExecuteBeforeScenario(scenario Scenario) {
	for _, he := range e.entries {
		if he.hooks.BeforeScenario != nil && he.matchesTags(scenario.Tags) {
			he.hooks.BeforeScenario(scenario)
		}
	}
}

// ExecuteAfterScenario executes matching AfterScenario hooks in order.
func (e *HookExecutor) ExecuteAfterScenario(scenario Scenario, err error) {
	for _, he := range e.entries {
		if he.hooks.AfterScenario != nil && he.matchesTags(scenario.Tags) {
			he.hooks.AfterScenario(scenario, err)
		}
	}
}

// ExecuteBeforeStep executes matching BeforeStep hooks in order.
// Uses the scenario tags set by SetScenarioTags for tag filtering.
func (e *HookExecutor) ExecuteBeforeStep(step Step) {
	for _, he := range e.entries {
		if he.hooks.BeforeStep != nil && he.matchesTags(e.scenarioTags) {
			he.hooks.BeforeStep(step)
		}
	}
}

// ExecuteAfterStep executes matching AfterStep hooks in order.
// Uses the scenario tags set by SetScenarioTags for tag filtering.
func (e *HookExecutor) ExecuteAfterStep(step Step, err error) {
	for _, he := range e.entries {
		if he.hooks.AfterStep != nil && he.matchesTags(e.scenarioTags) {
			he.hooks.AfterStep(step, err)
		}
	}
}
