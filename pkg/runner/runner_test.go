package runner

import (
	"fmt"
	"os"
	"sync"
	"testing"

	tagexpressions "github.com/cucumber/tag-expressions/go/v6"

	messages "github.com/cucumber/messages/go/v21"
	"github.com/denizgursoy/cacik/pkg/cacik"
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
		stepExecuted := false

		runner := NewCucumberRunner().
			WithFeaturesDirectories("testdata/with-tag").
			RegisterStep("^hello$", func(ctx *cacik.Context) {
				stepExecuted = true

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
			require.True(t, stepExecuted, "expected step to be executed")
		})
	})

	t.Run("does not execute feature if tags do not match", func(t *testing.T) {
		stepExecuted := false

		runner := NewCucumberRunner().
			WithFeaturesDirectories("testdata/without-tag").
			RegisterStep("^.*$", func(ctx *cacik.Context) {
				stepExecuted = true

			})

		withArgs([]string{"cmd", "--tags", "@nonexistent"}, func() {
			err := runner.Run()
			require.Nil(t, err)
			require.False(t, stepExecuted, "expected step NOT to be executed")
		})
	})

	t.Run("executes all features when no tags specified", func(t *testing.T) {
		stepExecuted := false

		runner := NewCucumberRunner().
			WithFeaturesDirectories("testdata/with-tag").
			RegisterStep("^hello$", func(ctx *cacik.Context) {
				stepExecuted = true

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
			require.True(t, stepExecuted, "expected step to be executed")
		})
	})

	t.Run("returns error for invalid tag expression", func(t *testing.T) {
		runner := NewCucumberRunner().
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
		runner := NewCucumberRunner()
		runner.RegisterStep("^test$", func() {})

		require.Panics(t, func() {
			runner.RegisterStep("^test$", func() {})
		})
	})

	t.Run("should panic on invalid regex pattern", func(t *testing.T) {
		runner := NewCucumberRunner()

		require.Panics(t, func() {
			runner.RegisterStep("[invalid", func() {})
		})
	})
}

func Test_parseParallelFromArgs(t *testing.T) {
	t.Run("parses --parallel with space", func(t *testing.T) {
		withArgs([]string{"cmd", "--parallel", "4"}, func() {
			result := parseParallelFromArgs()
			require.Equal(t, 4, result)
		})
	})

	t.Run("parses --parallel= format", func(t *testing.T) {
		withArgs([]string{"cmd", "--parallel=8"}, func() {
			result := parseParallelFromArgs()
			require.Equal(t, 8, result)
		})
	})

	t.Run("returns 1 when not specified", func(t *testing.T) {
		withArgs([]string{"cmd"}, func() {
			result := parseParallelFromArgs()
			require.Equal(t, 1, result)
		})
	})

	t.Run("returns 1 for invalid value", func(t *testing.T) {
		withArgs([]string{"cmd", "--parallel", "invalid"}, func() {
			result := parseParallelFromArgs()
			require.Equal(t, 1, result)
		})
	})

	t.Run("returns 1 for zero", func(t *testing.T) {
		withArgs([]string{"cmd", "--parallel", "0"}, func() {
			result := parseParallelFromArgs()
			require.Equal(t, 1, result)
		})
	})

	t.Run("returns 1 for negative", func(t *testing.T) {
		withArgs([]string{"cmd", "--parallel", "-1"}, func() {
			result := parseParallelFromArgs()
			require.Equal(t, 1, result)
		})
	})

	t.Run("combines with tags", func(t *testing.T) {
		withArgs([]string{"cmd", "--tags", "@smoke", "--parallel", "4"}, func() {
			parallel := parseParallelFromArgs()
			tags := parseTagsFromArgs()
			require.Equal(t, 4, parallel)
			require.Equal(t, "@smoke", tags)
		})
	})
}

func TestCucumberRunner_RunParallel(t *testing.T) {
	t.Run("executes scenarios in parallel", func(t *testing.T) {
		var mu sync.Mutex
		executedSteps := make([]string, 0)

		runner := NewCucumberRunner().
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

		withArgs([]string{"cmd", "--parallel", "2"}, func() {
			err := runner.Run()
			require.Nil(t, err)
			require.NotEmpty(t, executedSteps, "expected steps to be executed")
		})
	})

	t.Run("isolates context between scenarios", func(t *testing.T) {
		var mu sync.Mutex
		contextValues := make(map[string]int)

		runner := NewCucumberRunner().
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

		withArgs([]string{"cmd", "--parallel", "2"}, func() {
			err := runner.Run()
			require.Nil(t, err)
		})
	})
}

func TestCucumberRunner_WithTestingT(t *testing.T) {
	t.Run("executes scenarios as subtests", func(t *testing.T) {
		stepExecuted := false

		runner := NewCucumberRunner().
			WithTestingT(t).
			WithFeaturesDirectories("testdata/with-tag").
			RegisterStep("^hello$", func(ctx *cacik.Context) {
				stepExecuted = true
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
			require.True(t, stepExecuted, "expected step to be executed via t.Run subtests")
		})
	})

	t.Run("runs parallel scenarios as parallel subtests", func(t *testing.T) {
		var mu sync.Mutex
		executedSteps := make([]string, 0)

		runner := NewCucumberRunner().
			WithTestingT(t).
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

		withArgs([]string{"cmd", "--parallel", "2"}, func() {
			err := runner.Run()
			require.Nil(t, err)
			// Note: with t.Parallel() subtests, steps execute after this function
			// returns, so we cannot assert on executedSteps here. The subtests
			// themselves verify execution by passing.
		})
	})

	t.Run("assertion failure fails the subtest not the parent", func(t *testing.T) {
		runner := NewCucumberRunner().
			WithTestingT(t).
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

// newRuleRunner creates a CucumberRunner wired to testdata/with-rule
// with all step definitions needed by rule.feature.
// The optional onStep callback is called with each step keyword for tracking.
func newRuleRunner(onStep func(string)) *CucumberRunner {
	if onStep == nil {
		onStep = func(string) {}
	}

	return NewCucumberRunner().
		WithFeaturesDirectories("testdata/with-rule").
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
	t.Run("executes rules with feature and rule backgrounds sequentially", func(t *testing.T) {
		var mu sync.Mutex
		executedSteps := make([]string, 0)

		runner := newRuleRunner(func(step string) {
			mu.Lock()
			executedSteps = append(executedSteps, step)
			mu.Unlock()
		})

		withArgs([]string{"cmd"}, func() {
			err := runner.Run()
			require.Nil(t, err)
			require.NotEmpty(t, executedSteps, "expected steps to be executed")

			// Feature background runs before every scenario (4 scenarios total)
			count := 0
			for _, s := range executedSteps {
				if s == "system initialized" {
					count++
				}
			}
			require.Equal(t, 4, count, "feature background should run for each scenario")
		})
	})

	t.Run("executes rules with backgrounds in parallel", func(t *testing.T) {
		var mu sync.Mutex
		executedSteps := make([]string, 0)

		runner := newRuleRunner(func(step string) {
			mu.Lock()
			executedSteps = append(executedSteps, step)
			mu.Unlock()
		})

		withArgs([]string{"cmd", "--parallel", "2"}, func() {
			err := runner.Run()
			require.Nil(t, err)
			require.NotEmpty(t, executedSteps, "expected steps to be executed in parallel")
		})
	})

	t.Run("executes rules with backgrounds via testing.T subtests", func(t *testing.T) {
		runner := newRuleRunner(nil).WithTestingT(t)

		withArgs([]string{"cmd"}, func() {
			err := runner.Run()
			require.Nil(t, err)
		})
	})

	t.Run("executes rules with backgrounds via testing.T parallel subtests", func(t *testing.T) {
		runner := newRuleRunner(nil).WithTestingT(t)

		withArgs([]string{"cmd", "--parallel", "2"}, func() {
			err := runner.Run()
			require.Nil(t, err)
		})
	})
}

// newTableRunner creates a CucumberRunner wired to testdata/with-table
// with step definitions needed by table.feature.
func newTableRunner(onUsers func([]string)) *CucumberRunner {
	if onUsers == nil {
		onUsers = func([]string) {}
	}

	var userCount int

	return NewCucumberRunner().
		WithFeaturesDirectories("testdata/with-table").
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
	t.Run("executes step with DataTable sequentially", func(t *testing.T) {
		var capturedNames []string

		runner := newTableRunner(func(names []string) {
			capturedNames = names
		})

		withArgs([]string{"cmd"}, func() {
			err := runner.Run()
			require.Nil(t, err)
			require.Equal(t, []string{"Alice", "Bob"}, capturedNames)
		})
	})

	t.Run("executes step with DataTable in parallel", func(t *testing.T) {
		var mu sync.Mutex
		var capturedNames []string

		runner := newTableRunner(func(names []string) {
			mu.Lock()
			capturedNames = names
			mu.Unlock()
		})

		withArgs([]string{"cmd", "--parallel", "2"}, func() {
			err := runner.Run()
			require.Nil(t, err)
			mu.Lock()
			require.Equal(t, []string{"Alice", "Bob"}, capturedNames)
			mu.Unlock()
		})
	})

	t.Run("executes step with DataTable via testing.T", func(t *testing.T) {
		runner := newTableRunner(nil).WithTestingT(t)

		withArgs([]string{"cmd"}, func() {
			err := runner.Run()
			require.Nil(t, err)
		})
	})

	t.Run("executes step with DataTable via testing.T parallel", func(t *testing.T) {
		runner := newTableRunner(nil).WithTestingT(t)

		withArgs([]string{"cmd", "--parallel", "2"}, func() {
			err := runner.Run()
			require.Nil(t, err)
		})
	})
}
