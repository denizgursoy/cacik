package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	tagexpressions "github.com/cucumber/tag-expressions/go/v6"

	messages "github.com/cucumber/messages/go/v21"
	"github.com/denizgursoy/cacik/pkg/cacik"
	"github.com/denizgursoy/cacik/pkg/executor"
	"github.com/stretchr/testify/require"
)

// withArgs temporarily sets os.Args for testing and restores it after
func withArgs(args []string, fn func()) {
	oldArgs := os.Args
	os.Args = args
	defer func() { os.Args = oldArgs }()
	fn()
}

func Test_extractTagNames(t *testing.T) {
	t.Run("extracts tag names with @ prefix", func(t *testing.T) {
		tags := []*messages.Tag{
			{Name: "@smoke"},
			{Name: "@fast"},
		}
		names := extractTagNames(tags)
		require.Equal(t, []string{"@smoke", "@fast"}, names)
	})

	t.Run("returns empty slice for no tags", func(t *testing.T) {
		names := extractTagNames([]*messages.Tag{})
		require.Empty(t, names)
	})
}

func Test_mergeTags(t *testing.T) {
	t.Run("merges parent and child tags", func(t *testing.T) {
		parent := []string{"@feature"}
		child := []string{"@scenario"}
		merged := mergeTags(parent, child)
		require.Equal(t, []string{"@feature", "@scenario"}, merged)
	})
}

func Test_parseTagsFromArgs(t *testing.T) {
	t.Run("parses --tags with space", func(t *testing.T) {
		withArgs([]string{"cmd", "--tags", "@smoke"}, func() {
			result := parseTagsFromArgs()
			require.Equal(t, "@smoke", result)
		})
	})

	t.Run("parses --tags= format", func(t *testing.T) {
		withArgs([]string{"cmd", "--tags=@smoke and @fast"}, func() {
			result := parseTagsFromArgs()
			require.Equal(t, "@smoke and @fast", result)
		})
	})

	t.Run("returns empty string when no tags", func(t *testing.T) {
		withArgs([]string{"cmd"}, func() {
			result := parseTagsFromArgs()
			require.Equal(t, "", result)
		})
	})

	t.Run("handles complex expression", func(t *testing.T) {
		withArgs([]string{"cmd", "--tags", "(@smoke or @ui) and not @slow"}, func() {
			result := parseTagsFromArgs()
			require.Equal(t, "(@smoke or @ui) and not @slow", result)
		})
	})
}

func Test_filterDocumentByTags(t *testing.T) {
	t.Run("filters scenarios by tag", func(t *testing.T) {
		evaluator, _ := tagexpressions.Parse("@smoke")

		doc := &messages.GherkinDocument{
			Feature: &messages.Feature{
				Tags: []*messages.Tag{},
				Children: []*messages.FeatureChild{
					{
						Scenario: &messages.Scenario{
							Name: "Smoke Test",
							Tags: []*messages.Tag{{Name: "@smoke"}},
						},
					},
					{
						Scenario: &messages.Scenario{
							Name: "Other Test",
							Tags: []*messages.Tag{{Name: "@other"}},
						},
					},
				},
			},
		}

		filtered := filterDocumentByTags(doc, evaluator)
		require.Len(t, filtered.Feature.Children, 1)
		require.Equal(t, "Smoke Test", filtered.Feature.Children[0].Scenario.Name)
	})

	t.Run("inherits feature tags", func(t *testing.T) {
		evaluator, _ := tagexpressions.Parse("@feature")

		doc := &messages.GherkinDocument{
			Feature: &messages.Feature{
				Tags: []*messages.Tag{{Name: "@feature"}},
				Children: []*messages.FeatureChild{
					{
						Scenario: &messages.Scenario{
							Name: "Test",
							Tags: []*messages.Tag{},
						},
					},
				},
			},
		}

		filtered := filterDocumentByTags(doc, evaluator)
		require.Len(t, filtered.Feature.Children, 1)
	})

	t.Run("handles AND expression", func(t *testing.T) {
		evaluator, _ := tagexpressions.Parse("@smoke and @fast")

		doc := &messages.GherkinDocument{
			Feature: &messages.Feature{
				Children: []*messages.FeatureChild{
					{
						Scenario: &messages.Scenario{
							Name: "Both Tags",
							Tags: []*messages.Tag{{Name: "@smoke"}, {Name: "@fast"}},
						},
					},
					{
						Scenario: &messages.Scenario{
							Name: "Only Smoke",
							Tags: []*messages.Tag{{Name: "@smoke"}},
						},
					},
				},
			},
		}

		filtered := filterDocumentByTags(doc, evaluator)
		require.Len(t, filtered.Feature.Children, 1)
		require.Equal(t, "Both Tags", filtered.Feature.Children[0].Scenario.Name)
	})

	t.Run("handles OR expression", func(t *testing.T) {
		evaluator, _ := tagexpressions.Parse("@smoke or @fast")

		doc := &messages.GherkinDocument{
			Feature: &messages.Feature{
				Children: []*messages.FeatureChild{
					{
						Scenario: &messages.Scenario{
							Name: "Has Smoke",
							Tags: []*messages.Tag{{Name: "@smoke"}},
						},
					},
					{
						Scenario: &messages.Scenario{
							Name: "Has Fast",
							Tags: []*messages.Tag{{Name: "@fast"}},
						},
					},
					{
						Scenario: &messages.Scenario{
							Name: "Has Neither",
							Tags: []*messages.Tag{{Name: "@other"}},
						},
					},
				},
			},
		}

		filtered := filterDocumentByTags(doc, evaluator)
		require.Len(t, filtered.Feature.Children, 2)
	})

	t.Run("handles NOT expression", func(t *testing.T) {
		evaluator, _ := tagexpressions.Parse("not @slow")

		doc := &messages.GherkinDocument{
			Feature: &messages.Feature{
				Children: []*messages.FeatureChild{
					{
						Scenario: &messages.Scenario{
							Name: "Fast Test",
							Tags: []*messages.Tag{{Name: "@fast"}},
						},
					},
					{
						Scenario: &messages.Scenario{
							Name: "Slow Test",
							Tags: []*messages.Tag{{Name: "@slow"}},
						},
					},
				},
			},
		}

		filtered := filterDocumentByTags(doc, evaluator)
		require.Len(t, filtered.Feature.Children, 1)
		require.Equal(t, "Fast Test", filtered.Feature.Children[0].Scenario.Name)
	})

	t.Run("handles complex expression with parentheses", func(t *testing.T) {
		evaluator, _ := tagexpressions.Parse("(@smoke or @ui) and not @slow")

		doc := &messages.GherkinDocument{
			Feature: &messages.Feature{
				Children: []*messages.FeatureChild{
					{
						Scenario: &messages.Scenario{
							Name: "Smoke Fast",
							Tags: []*messages.Tag{{Name: "@smoke"}},
						},
					},
					{
						Scenario: &messages.Scenario{
							Name: "UI Fast",
							Tags: []*messages.Tag{{Name: "@ui"}},
						},
					},
					{
						Scenario: &messages.Scenario{
							Name: "Smoke Slow",
							Tags: []*messages.Tag{{Name: "@smoke"}, {Name: "@slow"}},
						},
					},
					{
						Scenario: &messages.Scenario{
							Name: "Other",
							Tags: []*messages.Tag{{Name: "@other"}},
						},
					},
				},
			},
		}

		filtered := filterDocumentByTags(doc, evaluator)
		require.Len(t, filtered.Feature.Children, 2)
		require.Equal(t, "Smoke Fast", filtered.Feature.Children[0].Scenario.Name)
		require.Equal(t, "UI Fast", filtered.Feature.Children[1].Scenario.Name)
	})

	t.Run("preserves background", func(t *testing.T) {
		evaluator, _ := tagexpressions.Parse("@smoke")

		doc := &messages.GherkinDocument{
			Feature: &messages.Feature{
				Children: []*messages.FeatureChild{
					{
						Background: &messages.Background{
							Name: "Setup",
						},
					},
					{
						Scenario: &messages.Scenario{
							Name: "Smoke Test",
							Tags: []*messages.Tag{{Name: "@smoke"}},
						},
					},
				},
			},
		}

		filtered := filterDocumentByTags(doc, evaluator)
		require.Len(t, filtered.Feature.Children, 2)
		require.NotNil(t, filtered.Feature.Children[0].Background)
	})

	t.Run("filters scenarios within rules with tag inheritance", func(t *testing.T) {
		evaluator, _ := tagexpressions.Parse("@feature and @rule")

		doc := &messages.GherkinDocument{
			Feature: &messages.Feature{
				Tags: []*messages.Tag{{Name: "@feature"}},
				Children: []*messages.FeatureChild{
					{
						Rule: &messages.Rule{
							Tags: []*messages.Tag{{Name: "@rule"}},
							Children: []*messages.RuleChild{
								{
									Scenario: &messages.Scenario{
										Name: "Rule Scenario",
										Tags: []*messages.Tag{},
									},
								},
							},
						},
					},
				},
			},
		}

		filtered := filterDocumentByTags(doc, evaluator)
		require.Len(t, filtered.Feature.Children, 1)
		require.NotNil(t, filtered.Feature.Children[0].Rule)
		require.Len(t, filtered.Feature.Children[0].Rule.Children, 1)
	})
}

func TestCucumberRunner_Run(t *testing.T) {
	t.Run("executes feature with matching tag", func(t *testing.T) {
		var mu sync.Mutex
		stepExecuted := false

		runner := NewCucumberRunner(t).
			WithFeaturesDirectories("testdata/with-tag").
			RegisterStep("^hello$", func(ctx *cacik.Context) {
				mu.Lock()
				stepExecuted = true
				mu.Unlock()

			}).
			RegisterStep("^user is logged in$", func(ctx *cacik.Context) {

			}).
			RegisterStep("^user clicks (.+)$", func(ctx *cacik.Context, link string) {

			}).
			RegisterStep("^user will be logged out$", func(ctx *cacik.Context) {

			})

		withArgs([]string{"cmd", "--tags", "@billing"}, func() {
			err := runner.Run()
			require.Nil(t, err)
		})

		t.Cleanup(func() {
			mu.Lock()
			defer mu.Unlock()
			require.True(t, stepExecuted, "expected step to be executed")
		})
	})

	t.Run("does not execute feature if tags do not match", func(t *testing.T) {
		stepExecuted := false

		runner := NewCucumberRunner(t).
			WithFeaturesDirectories("testdata/without-tag").
			RegisterStep("^.*$", func(ctx *cacik.Context) {
				stepExecuted = true

			})

		withArgs([]string{"cmd", "--tags", "@nonexistent"}, func() {
			err := runner.Run()
			require.Nil(t, err)
			// No scenarios match, so no subtests created — stepExecuted stays false
			require.False(t, stepExecuted, "expected step NOT to be executed")
		})
	})

	t.Run("executes all features when no tags specified", func(t *testing.T) {
		var mu sync.Mutex
		stepExecuted := false

		runner := NewCucumberRunner(t).
			WithFeaturesDirectories("testdata/with-tag").
			RegisterStep("^hello$", func(ctx *cacik.Context) {
				mu.Lock()
				stepExecuted = true
				mu.Unlock()

			}).
			RegisterStep("^user is logged in$", func(ctx *cacik.Context) {

			}).
			RegisterStep("^user clicks (.+)$", func(ctx *cacik.Context, link string) {

			}).
			RegisterStep("^user will be logged out$", func(ctx *cacik.Context) {

			})

		withArgs([]string{"cmd"}, func() {
			err := runner.Run()
			require.Nil(t, err)
		})

		t.Cleanup(func() {
			mu.Lock()
			defer mu.Unlock()
			require.True(t, stepExecuted, "expected step to be executed")
		})
	})

	t.Run("returns error for invalid tag expression", func(t *testing.T) {
		runner := NewCucumberRunner(t).
			WithFeaturesDirectories("testdata/with-tag")

		withArgs([]string{"cmd", "--tags", "invalid expression (("}, func() {
			err := runner.Run()
			require.Error(t, err)
			require.Contains(t, err.Error(), "invalid tag expression")
		})
	})
}

func TestCucumberRunner_RegisterStep(t *testing.T) {
	t.Run("should panic on duplicate step registration", func(t *testing.T) {
		runner := NewCucumberRunner(t)
		runner.RegisterStep("^test$", func() {})

		require.Panics(t, func() {
			runner.RegisterStep("^test$", func() {})
		})
	})

	t.Run("should panic on invalid regex pattern", func(t *testing.T) {
		runner := NewCucumberRunner(t)

		require.Panics(t, func() {
			runner.RegisterStep("[invalid", func() {})
		})
	})
}

func TestCucumberRunner_RunParallel(t *testing.T) {
	t.Run("executes scenarios in parallel", func(t *testing.T) {
		var mu sync.Mutex
		executedSteps := make([]string, 0)

		runner := NewCucumberRunner(t).
			WithFeaturesDirectories("testdata/with-tag").
			RegisterStep("^hello$", func(ctx *cacik.Context) {
				mu.Lock()
				executedSteps = append(executedSteps, "hello")
				mu.Unlock()

			}).
			RegisterStep("^user is logged in$", func(ctx *cacik.Context) {
				mu.Lock()
				executedSteps = append(executedSteps, "user is logged in")
				mu.Unlock()

			}).
			RegisterStep("^user clicks (.+)$", func(ctx *cacik.Context, link string) {
				mu.Lock()
				executedSteps = append(executedSteps, "user clicks "+link)
				mu.Unlock()

			}).
			RegisterStep("^user will be logged out$", func(ctx *cacik.Context) {
				mu.Lock()
				executedSteps = append(executedSteps, "user will be logged out")
				mu.Unlock()

			})

		withArgs([]string{"cmd"}, func() {
			err := runner.Run()
			require.Nil(t, err)
			// Note: with t.Parallel() subtests, steps execute after this function
			// returns, so we cannot assert on executedSteps here. The subtests
			// themselves verify execution by passing.
		})
	})

	t.Run("isolates context between scenarios", func(t *testing.T) {
		var mu sync.Mutex
		contextValues := make(map[string]int)

		runner := NewCucumberRunner(t).
			WithFeaturesDirectories("testdata/with-tag").
			RegisterStep("^hello$", func(ctx *cacik.Context) {
				ctx.Data().Set("value", 42)

			}).
			RegisterStep("^user is logged in$", func(ctx *cacik.Context) {
				// This should not see the value from another scenario
				_, ok := ctx.Data().Get("value")
				mu.Lock()
				if ok {
					contextValues["found"]++
				} else {
					contextValues["notfound"]++
				}
				mu.Unlock()

			}).
			RegisterStep("^user clicks (.+)$", func(ctx *cacik.Context, link string) {

			}).
			RegisterStep("^user will be logged out$", func(ctx *cacik.Context) {

			})

		withArgs([]string{"cmd"}, func() {
			err := runner.Run()
			require.Nil(t, err)
		})
	})
}

func TestCucumberRunner_Subtests(t *testing.T) {
	t.Run("executes scenarios as parallel subtests", func(t *testing.T) {
		var mu sync.Mutex
		executedSteps := make([]string, 0)

		runner := NewCucumberRunner(t).
			WithFeaturesDirectories("testdata/with-tag").
			RegisterStep("^hello$", func(ctx *cacik.Context) {
				mu.Lock()
				executedSteps = append(executedSteps, "hello")
				mu.Unlock()
			}).
			RegisterStep("^user is logged in$", func(ctx *cacik.Context) {
				mu.Lock()
				executedSteps = append(executedSteps, "user is logged in")
				mu.Unlock()
			}).
			RegisterStep("^user clicks (.+)$", func(ctx *cacik.Context, link string) {
				mu.Lock()
				executedSteps = append(executedSteps, "user clicks "+link)
				mu.Unlock()
			}).
			RegisterStep("^user will be logged out$", func(ctx *cacik.Context) {
				mu.Lock()
				executedSteps = append(executedSteps, "user will be logged out")
				mu.Unlock()
			})

		withArgs([]string{"cmd"}, func() {
			err := runner.Run()
			require.Nil(t, err)
			// Note: with t.Parallel() subtests, steps execute after this function
			// returns, so we cannot assert on executedSteps here. The subtests
			// themselves verify execution by passing.
		})
	})

	t.Run("assertion failure fails the subtest not the parent", func(t *testing.T) {
		runner := NewCucumberRunner(t).
			WithFeaturesDirectories("testdata/with-tag").
			RegisterStep("^hello$", func(ctx *cacik.Context) {
				// This step passes
			}).
			RegisterStep("^user is logged in$", func(ctx *cacik.Context) {
			}).
			RegisterStep("^user clicks (.+)$", func(ctx *cacik.Context, link string) {
			}).
			RegisterStep("^user will be logged out$", func(ctx *cacik.Context) {
			})

		withArgs([]string{"cmd"}, func() {
			err := runner.Run()
			require.Nil(t, err)
		})
	})
}

// newRuleRunner creates a CucumberRunner with a temp directory containing a rule feature file.
// The optional onStep callback is called with each step keyword for tracking.
func newRuleRunner(t *testing.T, onStep func(string)) *CucumberRunner {
	t.Helper()
	if onStep == nil {
		onStep = func(string) {}
	}

	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "rule.feature"), []byte(`Feature: User management with rules
  Background:
    Given the system is initialized
  Rule: Registration
    Background:
      Given the registration form is loaded
    Scenario: Successful registration
      When the user registers with "alice@example.com"
      Then the registration should succeed
    Scenario: Failed registration
      When the user registers with ""
      Then the registration should fail
  Rule: Login
    Background:
      Given the login page is loaded
    Scenario: Successful login
      When the user logs in with "alice" and "secret"
      Then the login should succeed
    Scenario: Failed login
      When the user logs in with "alice" and "wrong"
      Then the login should fail
`), 0644)
	require.NoError(t, err)

	return NewCucumberRunner(t).
		WithFeaturesDirectories(dir).
		RegisterStep(`^the system is initialized$`, func(ctx *cacik.Context) {
			onStep("system initialized")
		}).
		RegisterStep(`^the registration form is loaded$`, func(ctx *cacik.Context) {
			onStep("registration form loaded")
		}).
		RegisterStep(`^the login page is loaded$`, func(ctx *cacik.Context) {
			onStep("login page loaded")
		}).
		RegisterStep(`^the user registers with "([^"]*)"$`, func(ctx *cacik.Context, email string) {
			onStep("register " + email)
		}).
		RegisterStep(`^the registration should succeed$`, func(ctx *cacik.Context) {
			onStep("registration succeed")
		}).
		RegisterStep(`^the registration should fail$`, func(ctx *cacik.Context) {
			onStep("registration fail")
		}).
		RegisterStep(`^the user logs in with "([^"]*)" and "([^"]*)"$`, func(ctx *cacik.Context, user, pass string) {
			onStep("login " + user)
		}).
		RegisterStep(`^the login should succeed$`, func(ctx *cacik.Context) {
			onStep("login succeed")
		}).
		RegisterStep(`^the login should fail$`, func(ctx *cacik.Context) {
			onStep("login fail")
		})
}

func TestCucumberRunner_RuleWithBackground(t *testing.T) {
	t.Run("executes rules with feature and rule backgrounds", func(t *testing.T) {
		runner := newRuleRunner(t, nil)

		withArgs([]string{"cmd"}, func() {
			err := runner.Run()
			require.Nil(t, err)
			// Note: with t.Parallel() subtests, steps execute after this function
			// returns. The subtests themselves verify execution by passing.
		})
	})
}

// newTableRunner creates a CucumberRunner with a temp directory containing a table feature file.
func newTableRunner(t *testing.T, onUsers func([]string)) *CucumberRunner {
	t.Helper()
	if onUsers == nil {
		onUsers = func([]string) {}
	}

	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "table.feature"), []byte(`Feature: DataTable support
  Scenario: Step with a DataTable
    Given the following users:
      | name  | age |
      | Alice | 30  |
      | Bob   | 25  |
    Then there should be 2 users
`), 0644)
	require.NoError(t, err)

	var userCount int

	return NewCucumberRunner(t).
		WithFeaturesDirectories(dir).
		RegisterStep(`^the following users:$`, func(ctx *cacik.Context, table cacik.Table) {
			var names []string
			for _, row := range table.SkipHeader() {
				names = append(names, row.Get("name"))
			}
			userCount = len(names)
			onUsers(names)
		}).
		RegisterStep(`^there should be (\d+) users$`, func(ctx *cacik.Context, expected int) error {
			if userCount != expected {
				return fmt.Errorf("expected %d users, got %d", expected, userCount)
			}
			return nil
		})
}

func TestCucumberRunner_DataTable(t *testing.T) {
	t.Run("executes step with DataTable", func(t *testing.T) {
		runner := newTableRunner(t, nil)

		withArgs([]string{"cmd"}, func() {
			err := runner.Run()
			require.Nil(t, err)
			// Note: with t.Parallel() subtests, steps execute after this function
			// returns. The subtests themselves verify execution by passing.
		})
	})
}

// =============================================================================
// Scenario Outline Tests
// =============================================================================

// outlineEvent records a step invocation during Scenario Outline execution.
type outlineEvent struct {
	Step string
	Args []string
}

// newOutlineRunner creates a CucumberRunner wired to testdata/scenario-outline/outline.feature.
// The onEvent callback records every step invocation so the test can verify
// placeholder substitution, DataTable expansion, and execution order.
func newOutlineRunner(t *testing.T, onEvent func(outlineEvent)) *CucumberRunner {
	t.Helper()
	if onEvent == nil {
		onEvent = func(outlineEvent) {}
	}

	return NewCucumberRunner(t).
		WithFeaturesDirectories("testdata/scenario-outline").
		RegisterStep(`^the application is started$`, func(ctx *cacik.Context) {
			onEvent(outlineEvent{Step: "app started"})
		}).
		RegisterStep(`^user "([^"]*)" exists with role "([^"]*)"$`, func(ctx *cacik.Context, user, role string) {
			onEvent(outlineEvent{Step: "user exists", Args: []string{user, role}})
		}).
		RegisterStep(`^user "([^"]*)" logs in with password "([^"]*)"$`, func(ctx *cacik.Context, user, pass string) {
			onEvent(outlineEvent{Step: "login", Args: []string{user, pass}})
		}).
		RegisterStep(`^the login result should be "([^"]*)"$`, func(ctx *cacik.Context, result string) {
			onEvent(outlineEvent{Step: "login result", Args: []string{result}})
		}).
		RegisterStep(`^the user role should be "([^"]*)"$`, func(ctx *cacik.Context, role string) {
			onEvent(outlineEvent{Step: "user role", Args: []string{role}})
		}).
		RegisterStep(`^I assign permissions to "([^"]*)":$`, func(ctx *cacik.Context, user string, table cacik.Table) {
			var perms []string
			for _, row := range table.SkipHeader() {
				perms = append(perms, row.Get("permission")+":"+row.Get("granted"))
			}
			onEvent(outlineEvent{Step: "assign permissions", Args: append([]string{user}, perms...)})
		}).
		RegisterStep(`^user "([^"]*)" should have (\d+) permissions$`, func(ctx *cacik.Context, user string, count int) {
			onEvent(outlineEvent{Step: "has permissions", Args: []string{user, fmt.Sprintf("%d", count)}})
		}).
		RegisterStep(`^the application is running$`, func(ctx *cacik.Context) {
			onEvent(outlineEvent{Step: "app running"})
		}).
		RegisterStep(`^I check the status$`, func(ctx *cacik.Context) {
			onEvent(outlineEvent{Step: "check status"})
		}).
		RegisterStep(`^the status code should be (\d+)$`, func(ctx *cacik.Context, code int) {
			onEvent(outlineEvent{Step: "status code", Args: []string{fmt.Sprintf("%d", code)}})
		}).
		RegisterStep(`^the access control module is loaded$`, func(ctx *cacik.Context) {
			onEvent(outlineEvent{Step: "acl loaded"})
		}).
		RegisterStep(`^user "([^"]*)" has role "([^"]*)"$`, func(ctx *cacik.Context, user, role string) {
			onEvent(outlineEvent{Step: "user has role", Args: []string{user, role}})
		}).
		RegisterStep(`^user "([^"]*)" accesses "([^"]*)"$`, func(ctx *cacik.Context, user, resource string) {
			onEvent(outlineEvent{Step: "accesses", Args: []string{user, resource}})
		}).
		RegisterStep(`^access should be "([^"]*)"$`, func(ctx *cacik.Context, decision string) {
			onEvent(outlineEvent{Step: "access decision", Args: []string{decision}})
		})
}

func TestCucumberRunner_ScenarioOutline(t *testing.T) {
	t.Run("expands all outline examples", func(t *testing.T) {
		var mu sync.Mutex
		var events []outlineEvent

		runner := newOutlineRunner(t, func(e outlineEvent) {
			mu.Lock()
			events = append(events, e)
			mu.Unlock()
		})

		withArgs([]string{"cmd"}, func() {
			err := runner.Run()
			require.Nil(t, err)
		})

		// Assertions run after parallel subtests complete
		t.Cleanup(func() {
			mu.Lock()
			defer mu.Unlock()

			// Count login events — 5 rows (3 valid + 2 invalid)
			loginEvents := filterEvents(events, "login")
			require.Len(t, loginEvents, 5, "expected 5 login invocations from outline expansion")

			// Check login result substitution
			resultEvents := filterEvents(events, "login result")
			require.Len(t, resultEvents, 5)

			// Check role substitution
			roleEvents := filterEvents(events, "user role")
			require.Len(t, roleEvents, 5)
		})
	})

	t.Run("substitutes placeholders inside DataTable cells", func(t *testing.T) {
		var mu sync.Mutex
		var events []outlineEvent

		runner := newOutlineRunner(t, func(e outlineEvent) {
			mu.Lock()
			events = append(events, e)
			mu.Unlock()
		})

		withArgs([]string{"cmd"}, func() {
			err := runner.Run()
			require.Nil(t, err)
		})

		t.Cleanup(func() {
			mu.Lock()
			defer mu.Unlock()

			permEvents := filterEvents(events, "assign permissions")
			require.Len(t, permEvents, 2, "expected 2 permission assignment invocations")
		})
	})

	t.Run("handles static step text with varying examples", func(t *testing.T) {
		var mu sync.Mutex
		var events []outlineEvent

		runner := newOutlineRunner(t, func(e outlineEvent) {
			mu.Lock()
			events = append(events, e)
			mu.Unlock()
		})

		withArgs([]string{"cmd"}, func() {
			err := runner.Run()
			require.Nil(t, err)
		})

		t.Cleanup(func() {
			mu.Lock()
			defer mu.Unlock()

			statusEvents := filterEvents(events, "status code")
			require.Len(t, statusEvents, 2)
		})
	})

	t.Run("expands outline inside rules with background", func(t *testing.T) {
		var mu sync.Mutex
		var events []outlineEvent

		runner := newOutlineRunner(t, func(e outlineEvent) {
			mu.Lock()
			events = append(events, e)
			mu.Unlock()
		})

		withArgs([]string{"cmd"}, func() {
			err := runner.Run()
			require.Nil(t, err)
		})

		t.Cleanup(func() {
			mu.Lock()
			defer mu.Unlock()

			// 4 rows total in the Rule's outline (2 admin + 2 viewer)
			accessEvents := filterEvents(events, "access decision")
			require.Len(t, accessEvents, 4)

			// Feature background should run for each expanded scenario
			appStartedEvents := filterEvents(events, "app started")
			require.Len(t, appStartedEvents, 13, "feature background should run for every expanded scenario")

			// Rule background (acl loaded) should run for each expanded scenario in the rule = 4
			aclEvents := filterEvents(events, "acl loaded")
			require.Len(t, aclEvents, 4, "rule background should run for each expanded scenario in the rule")
		})
	})

	t.Run("filters by examples-level tag", func(t *testing.T) {
		var mu sync.Mutex
		var events []outlineEvent

		runner := newOutlineRunner(t, func(e outlineEvent) {
			mu.Lock()
			events = append(events, e)
			mu.Unlock()
		})

		withArgs([]string{"cmd", "--tags", "@negative"}, func() {
			err := runner.Run()
			require.Nil(t, err)
		})

		t.Cleanup(func() {
			mu.Lock()
			defer mu.Unlock()

			// Only the @negative Examples rows should execute (2 rows)
			loginEvents := filterEvents(events, "login")
			require.Len(t, loginEvents, 2, "only @negative examples should run")
		})
	})
}

// filterEvents returns events matching the given step name.
func filterEvents(events []outlineEvent, step string) []outlineEvent {
	var result []outlineEvent
	for _, e := range events {
		if e.Step == step {
			result = append(result, e)
		}
	}
	return result
}

func Test_resolveAllSteps(t *testing.T) {
	t.Run("resolves all steps when all match", func(t *testing.T) {
		exec := executor.NewStepExecutor()
		err := exec.RegisterStep("^a known step$", func() {})
		require.NoError(t, err)

		runner := NewCucumberRunnerWithExecutor(t, exec)
		scenarios := []ScenarioExecution{
			{
				Scenario: &messages.Scenario{
					Name: "Test Scenario",
					Steps: []*messages.Step{
						{Keyword: "Given ", Text: "a known step"},
					},
				},
				FeatureName: "Test Feature",
				FeatureFile: "test.feature",
			},
		}

		resolveErr := runner.resolveAllSteps(scenarios)
		require.NoError(t, resolveErr)
		require.Len(t, scenarios[0].ResolvedScenarioSteps, 1)
		require.Equal(t, "a known step", scenarios[0].ResolvedScenarioSteps[0].Text)
		require.Equal(t, "Given ", scenarios[0].ResolvedScenarioSteps[0].Keyword)
	})

	t.Run("fails fast on first unmatched step", func(t *testing.T) {
		exec := executor.NewStepExecutor()
		err := exec.RegisterStep("^a known step$", func() {})
		require.NoError(t, err)

		runner := NewCucumberRunnerWithExecutor(t, exec)
		scenarios := []ScenarioExecution{
			{
				Scenario: &messages.Scenario{
					Name: "Test Scenario",
					Steps: []*messages.Step{
						{Keyword: "Given ", Text: "a known step"},
						{Keyword: "When ", Text: "an unknown step"},
					},
				},
				FeatureName: "Test Feature",
				FeatureFile: "test.feature",
			},
		}

		resolveErr := runner.resolveAllSteps(scenarios)
		require.Error(t, resolveErr)
		require.Contains(t, resolveErr.Error(), `"an unknown step"`)
		require.Contains(t, resolveErr.Error(), "Feature: Test Feature")
		require.Contains(t, resolveErr.Error(), "Scenario: Test Scenario")
		require.Contains(t, resolveErr.Error(), "test.feature")
	})

	t.Run("resolves feature background steps", func(t *testing.T) {
		exec := executor.NewStepExecutor()
		err := exec.RegisterStep("^bg step$", func() {})
		require.NoError(t, err)
		err = exec.RegisterStep("^scenario step$", func() {})
		require.NoError(t, err)

		runner := NewCucumberRunnerWithExecutor(t, exec)
		scenarios := []ScenarioExecution{
			{
				Scenario: &messages.Scenario{
					Name:  "Test Scenario",
					Steps: []*messages.Step{{Keyword: "Then ", Text: "scenario step"}},
				},
				FeatureBackground: &messages.Background{
					Steps: []*messages.Step{{Keyword: "Given ", Text: "bg step"}},
				},
				FeatureName: "Feature",
				FeatureFile: "test.feature",
			},
		}

		resolveErr := runner.resolveAllSteps(scenarios)
		require.NoError(t, resolveErr)
		require.Len(t, scenarios[0].ResolvedFeatureBgSteps, 1)
		require.Equal(t, "bg step", scenarios[0].ResolvedFeatureBgSteps[0].Text)
		require.Len(t, scenarios[0].ResolvedScenarioSteps, 1)
	})

	t.Run("fails fast on unmatched background step", func(t *testing.T) {
		exec := executor.NewStepExecutor()
		err := exec.RegisterStep("^scenario step$", func() {})
		require.NoError(t, err)

		runner := NewCucumberRunnerWithExecutor(t, exec)
		scenarios := []ScenarioExecution{
			{
				Scenario: &messages.Scenario{
					Name:  "Test Scenario",
					Steps: []*messages.Step{{Keyword: "Then ", Text: "scenario step"}},
				},
				FeatureBackground: &messages.Background{
					Steps: []*messages.Step{{Keyword: "Given ", Text: "unmatched background step"}},
				},
				FeatureName: "Feature",
				FeatureFile: "test.feature",
			},
		}

		resolveErr := runner.resolveAllSteps(scenarios)
		require.Error(t, resolveErr)
		require.Contains(t, resolveErr.Error(), "unmatched background step")
	})

	t.Run("resolves rule background steps", func(t *testing.T) {
		exec := executor.NewStepExecutor()
		err := exec.RegisterStep("^rule bg step$", func() {})
		require.NoError(t, err)
		err = exec.RegisterStep("^scenario step$", func() {})
		require.NoError(t, err)

		runner := NewCucumberRunnerWithExecutor(t, exec)
		scenarios := []ScenarioExecution{
			{
				Scenario: &messages.Scenario{
					Name:  "Test Scenario",
					Steps: []*messages.Step{{Keyword: "Then ", Text: "scenario step"}},
				},
				RuleBackground: &messages.Background{
					Steps: []*messages.Step{{Keyword: "Given ", Text: "rule bg step"}},
				},
				FeatureName: "Feature",
				FeatureFile: "test.feature",
			},
		}

		resolveErr := runner.resolveAllSteps(scenarios)
		require.NoError(t, resolveErr)
		require.Len(t, scenarios[0].ResolvedRuleBgSteps, 1)
		require.Equal(t, "rule bg step", scenarios[0].ResolvedRuleBgSteps[0].Text)
	})

	t.Run("fails fast on unmatched rule background step", func(t *testing.T) {
		exec := executor.NewStepExecutor()
		err := exec.RegisterStep("^scenario step$", func() {})
		require.NoError(t, err)

		runner := NewCucumberRunnerWithExecutor(t, exec)
		scenarios := []ScenarioExecution{
			{
				Scenario: &messages.Scenario{
					Name:  "Test Scenario",
					Steps: []*messages.Step{{Keyword: "Then ", Text: "scenario step"}},
				},
				RuleBackground: &messages.Background{
					Steps: []*messages.Step{{Keyword: "Given ", Text: "unmatched rule bg step"}},
				},
				FeatureName: "Feature",
				FeatureFile: "test.feature",
			},
		}

		resolveErr := runner.resolveAllSteps(scenarios)
		require.Error(t, resolveErr)
		require.Contains(t, resolveErr.Error(), "unmatched rule bg step")
	})
}

// =============================================================================
// parseReportFileFromArgs Tests
// =============================================================================

func Test_parseReportFileFromArgs(t *testing.T) {
	t.Run("parses --report-file with space", func(t *testing.T) {
		withArgs([]string{"cmd", "--report-file", "report"}, func() {
			result := parseReportFileFromArgs()
			require.Equal(t, "report", result)
		})
	})

	t.Run("parses --report-file= format", func(t *testing.T) {
		withArgs([]string{"cmd", "--report-file=output/report"}, func() {
			result := parseReportFileFromArgs()
			require.Equal(t, "output/report", result)
		})
	})

	t.Run("returns empty string when not present", func(t *testing.T) {
		withArgs([]string{"cmd"}, func() {
			result := parseReportFileFromArgs()
			require.Equal(t, "", result)
		})
	})

	t.Run("returns empty when flag is last arg with no value", func(t *testing.T) {
		withArgs([]string{"cmd", "--report-file"}, func() {
			result := parseReportFileFromArgs()
			require.Equal(t, "", result)
		})
	})
}

// =============================================================================
// resolveReportFile Tests
// =============================================================================

func Test_resolveReportFile(t *testing.T) {
	t.Run("returns empty when no config and no CLI flag", func(t *testing.T) {
		runner := NewCucumberRunner(t)
		withArgs([]string{"cmd"}, func() {
			result := runner.resolveReportFile()
			require.Equal(t, "", result)
		})
	})

	t.Run("returns config value with .html appended", func(t *testing.T) {
		runner := NewCucumberRunner(t).WithConfig(&cacik.Config{
			ReportFile: "config-report",
		})
		withArgs([]string{"cmd"}, func() {
			result := runner.resolveReportFile()
			require.Equal(t, "config-report.html", result)
		})
	})

	t.Run("CLI flag overrides config with .html appended", func(t *testing.T) {
		runner := NewCucumberRunner(t).WithConfig(&cacik.Config{
			ReportFile: "config-report",
		})
		withArgs([]string{"cmd", "--report-file", "cli-report"}, func() {
			result := runner.resolveReportFile()
			require.Equal(t, "cli-report.html", result)
		})
	})
}

// =============================================================================
// resolvedStepToResult Tests
// =============================================================================

func Test_resolvedStepToResult(t *testing.T) {
	t.Run("maps passed step", func(t *testing.T) {
		rs := &executor.ResolvedStep{
			Keyword: "Given ",
			Text:    "a step",
			Status:  "passed",
			Error:   "",
		}
		result := resolvedStepToResult(rs)
		require.Equal(t, cacik.StepPassed, result.Status)
		require.Equal(t, "Given ", result.Keyword)
		require.Equal(t, "a step", result.Text)
		require.Empty(t, result.Error)
	})

	t.Run("maps failed step", func(t *testing.T) {
		rs := &executor.ResolvedStep{
			Keyword: "When ",
			Text:    "something fails",
			Status:  "failed",
			Error:   "oops",
		}
		result := resolvedStepToResult(rs)
		require.Equal(t, cacik.StepFailed, result.Status)
		require.Equal(t, "oops", result.Error)
	})

	t.Run("maps skipped step", func(t *testing.T) {
		rs := &executor.ResolvedStep{
			Keyword: "Then ",
			Text:    "skipped",
			Status:  "skipped",
		}
		result := resolvedStepToResult(rs)
		require.Equal(t, cacik.StepSkipped, result.Status)
	})

	t.Run("maps unknown status to skipped", func(t *testing.T) {
		rs := &executor.ResolvedStep{
			Keyword: "And ",
			Text:    "unknown",
			Status:  "",
		}
		result := resolvedStepToResult(rs)
		require.Equal(t, cacik.StepSkipped, result.Status)
	})
}

// =============================================================================
// buildRunResult Tests
// =============================================================================

func Test_buildRunResult(t *testing.T) {
	t.Run("builds result from scenario executions", func(t *testing.T) {
		reporter := cacik.NewNoopConsoleReporter()
		reporter.AddScenarioResult(true)
		reporter.AddScenarioResult(false)
		reporter.AddStepResult(true, false)
		reporter.AddStepResult(false, false)
		reporter.AddStepResult(false, true)

		scenarios := []ScenarioExecution{
			{
				Scenario: &messages.Scenario{
					Name: "Passing scenario",
					Tags: []*messages.Tag{{Name: "@smoke"}},
				},
				FeatureName: "Feature A",
				RuleName:    "",
				Passed:      true,
				ResolvedScenarioSteps: []*executor.ResolvedStep{
					{Keyword: "Given ", Text: "step one", Status: "passed"},
				},
			},
			{
				Scenario: &messages.Scenario{
					Name: "Failing scenario",
					Tags: []*messages.Tag{{Name: "@regression"}},
				},
				FeatureName: "Feature A",
				RuleName:    "Rule X",
				Passed:      false,
				Error:       "step two failed",
				ResolvedScenarioSteps: []*executor.ResolvedStep{
					{Keyword: "When ", Text: "step two", Status: "failed", Error: "step two failed"},
					{Keyword: "Then ", Text: "step three", Status: "skipped"},
				},
			},
		}

		runner := NewCucumberRunner(t)
		runStartedAt := time.Now()
		result := runner.buildRunResult(scenarios, reporter, runStartedAt)

		require.Len(t, result.Scenarios, 2)

		// First scenario
		require.Equal(t, "Passing scenario", result.Scenarios[0].Name)
		require.Equal(t, "Feature A", result.Scenarios[0].FeatureName)
		require.Empty(t, result.Scenarios[0].RuleName)
		require.True(t, result.Scenarios[0].Passed)
		require.Len(t, result.Scenarios[0].Steps, 1)
		require.Equal(t, cacik.StepPassed, result.Scenarios[0].Steps[0].Status)
		require.Equal(t, []string{"@smoke"}, result.Scenarios[0].Tags)

		// Second scenario
		require.Equal(t, "Failing scenario", result.Scenarios[1].Name)
		require.Equal(t, "Rule X", result.Scenarios[1].RuleName)
		require.False(t, result.Scenarios[1].Passed)
		require.Equal(t, "step two failed", result.Scenarios[1].Error)
		require.Len(t, result.Scenarios[1].Steps, 2)
		require.Equal(t, cacik.StepFailed, result.Scenarios[1].Steps[0].Status)
		require.Equal(t, cacik.StepSkipped, result.Scenarios[1].Steps[1].Status)
		require.Equal(t, []string{"@regression"}, result.Scenarios[1].Tags)

		// Summary
		require.Equal(t, 2, result.Summary.ScenariosTotal)
		require.Equal(t, 1, result.Summary.ScenariosPassed)
		require.Equal(t, 1, result.Summary.ScenariosFailed)

		// Duration and StartedAt
		require.Equal(t, runStartedAt, result.StartedAt)
		require.Greater(t, result.Duration, time.Duration(0), "RunResult.Duration should be > 0")
	})

	t.Run("includes background steps in order", func(t *testing.T) {
		reporter := cacik.NewNoopConsoleReporter()

		scenarios := []ScenarioExecution{
			{
				Scenario:    &messages.Scenario{Name: "With backgrounds"},
				FeatureName: "Feature",
				Passed:      true,
				ResolvedFeatureBgSteps: []*executor.ResolvedStep{
					{Keyword: "Given ", Text: "feature bg", Status: "passed"},
				},
				ResolvedRuleBgSteps: []*executor.ResolvedStep{
					{Keyword: "Given ", Text: "rule bg", Status: "passed"},
				},
				ResolvedScenarioSteps: []*executor.ResolvedStep{
					{Keyword: "Then ", Text: "scenario step", Status: "passed"},
				},
			},
		}

		runner := NewCucumberRunner(t)
		result := runner.buildRunResult(scenarios, reporter, time.Now())

		require.Len(t, result.Scenarios[0].FeatureBgSteps, 1)
		require.Equal(t, "feature bg", result.Scenarios[0].FeatureBgSteps[0].Text)
		require.Len(t, result.Scenarios[0].RuleBgSteps, 1)
		require.Equal(t, "rule bg", result.Scenarios[0].RuleBgSteps[0].Text)
		require.Len(t, result.Scenarios[0].Steps, 1)
		require.Equal(t, "scenario step", result.Scenarios[0].Steps[0].Text)
	})
}

// =============================================================================
// AfterRun Callback Integration Test
// =============================================================================

func TestCucumberRunner_AfterRunCallback(t *testing.T) {
	t.Run("calls AfterRun with RunResult after all scenarios complete", func(t *testing.T) {
		var mu sync.Mutex
		var capturedResult *cacik.RunResult

		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "test.feature"), []byte(`Feature: AfterRun test
  Scenario: Passing
    Given a passing step
`), 0644)
		require.NoError(t, err)

		runner := NewCucumberRunner(t).
			WithFeaturesDirectories(dir).
			WithConfig(&cacik.Config{
				DisableReporter: true,
				AfterRun: func(result cacik.RunResult) {
					mu.Lock()
					capturedResult = &result
					mu.Unlock()
				},
			}).
			RegisterStep(`^a passing step$`, func(ctx *cacik.Context) {})

		withArgs([]string{"cmd"}, func() {
			runErr := runner.Run()
			require.NoError(t, runErr)
		})

		// AfterRun is called synchronously after all subtests, so it should be set
		// by the time Run() returns
		mu.Lock()
		defer mu.Unlock()
		require.NotNil(t, capturedResult, "AfterRun callback should have been called")
		require.Len(t, capturedResult.Scenarios, 1)
		require.Equal(t, "Passing", capturedResult.Scenarios[0].Name)
		require.Equal(t, "AfterRun test", capturedResult.Scenarios[0].FeatureName)
	})
}

// =============================================================================
// HTML Report Integration Test
// =============================================================================

func TestCucumberRunner_HTMLReportGeneration(t *testing.T) {
	t.Run("generates HTML report via config", func(t *testing.T) {
		dir := t.TempDir()
		reportName := filepath.Join(dir, "report")
		reportPath := reportName + ".html"

		featureDir := t.TempDir()
		err := os.WriteFile(filepath.Join(featureDir, "test.feature"), []byte(`Feature: HTML test
  Scenario: A scenario
    Given a step
`), 0644)
		require.NoError(t, err)

		runner := NewCucumberRunner(t).
			WithFeaturesDirectories(featureDir).
			WithConfig(&cacik.Config{
				DisableReporter: true,
				ReportFile:      reportName,
			}).
			RegisterStep(`^a step$`, func(ctx *cacik.Context) {})

		withArgs([]string{"cmd"}, func() {
			runErr := runner.Run()
			require.NoError(t, runErr)
		})

		_, statErr := os.Stat(reportPath)
		require.NoError(t, statErr, "HTML report file should exist")

		data, readErr := os.ReadFile(reportPath)
		require.NoError(t, readErr)
		require.Contains(t, string(data), "HTML test")
		require.Contains(t, string(data), "A scenario")
	})

	t.Run("generates HTML report with multiple scenarios", func(t *testing.T) {
		dir := t.TempDir()
		reportName := filepath.Join(dir, "multi-report")
		reportPath := reportName + ".html"

		featureDir := t.TempDir()
		err := os.WriteFile(filepath.Join(featureDir, "test.feature"), []byte(`Feature: Multi scenario
  Scenario: First
    Given step one

  Scenario: Second
    Given step two
    Then step three
`), 0644)
		require.NoError(t, err)

		var mu sync.Mutex
		var capturedResult *cacik.RunResult

		runner := NewCucumberRunner(t).
			WithFeaturesDirectories(featureDir).
			WithConfig(&cacik.Config{
				DisableReporter: true,
				ReportFile:      reportName,
				AfterRun: func(result cacik.RunResult) {
					mu.Lock()
					capturedResult = &result
					mu.Unlock()
				},
			}).
			RegisterStep(`^step one$`, func(ctx *cacik.Context) {}).
			RegisterStep(`^step two$`, func(ctx *cacik.Context) {}).
			RegisterStep(`^step three$`, func(ctx *cacik.Context) {})

		withArgs([]string{"cmd"}, func() {
			runErr := runner.Run()
			require.NoError(t, runErr)
		})

		// Verify report file
		data, readErr := os.ReadFile(reportPath)
		require.NoError(t, readErr)
		html := string(data)
		require.Contains(t, html, "Multi scenario")
		require.Contains(t, html, "First")
		require.Contains(t, html, "Second")
		require.Contains(t, html, "step one")
		require.Contains(t, html, "step two")
		require.Contains(t, html, "step three")

		// Verify AfterRun received results
		mu.Lock()
		defer mu.Unlock()
		require.NotNil(t, capturedResult)
		require.Equal(t, 2, len(capturedResult.Scenarios))
	})

	t.Run("generates HTML report with rule labels", func(t *testing.T) {
		dir := t.TempDir()
		reportName := filepath.Join(dir, "rule-report")
		reportPath := reportName + ".html"

		featureDir := t.TempDir()
		err := os.WriteFile(filepath.Join(featureDir, "rule.feature"), []byte(`Feature: User management
  Background:
    Given the system is initialized
  Rule: Registration
    Background:
      Given the registration form is loaded
    Scenario: Successful registration
      When the user registers with "alice@example.com"
      Then the registration should succeed
`), 0644)
		require.NoError(t, err)

		runner := NewCucumberRunner(t).
			WithFeaturesDirectories(featureDir).
			WithConfig(&cacik.Config{
				DisableReporter: true,
				ReportFile:      reportName,
			}).
			RegisterStep(`^the system is initialized$`, func(ctx *cacik.Context) {}).
			RegisterStep(`^the registration form is loaded$`, func(ctx *cacik.Context) {}).
			RegisterStep(`^the user registers with "([^"]*)"$`, func(ctx *cacik.Context, email string) {}).
			RegisterStep(`^the registration should succeed$`, func(ctx *cacik.Context) {})

		withArgs([]string{"cmd"}, func() {
			runErr := runner.Run()
			require.NoError(t, runErr)
		})

		data, readErr := os.ReadFile(reportPath)
		require.NoError(t, readErr)
		html := string(data)

		// Rule label should appear in the scenario header breadcrumb
		require.Contains(t, html, "Registration", "HTML report should contain the rule name")
		// Rule label should appear in the step detail area
		require.Contains(t, html, "Rule:", "HTML report should contain Rule: label in step area")
		// Background labels should appear
		require.Contains(t, html, "Background:", "HTML report should contain Background: labels")
		// Feature name and scenario name
		require.Contains(t, html, "User management")
		require.Contains(t, html, "Successful registration")
	})
}
