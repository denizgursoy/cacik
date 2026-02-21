package runner

import (
	"context"
	"os"
	"testing"

	tagexpressions "github.com/cucumber/tag-expressions/go/v6"

	messages "github.com/cucumber/messages/go/v21"
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
			RegisterStep("^hello$", func(ctx context.Context) (context.Context, error) {
				stepExecuted = true
				return ctx, nil
			}).
			RegisterStep("^user is logged in$", func(ctx context.Context) (context.Context, error) {
				return ctx, nil
			}).
			RegisterStep("^user clicks (.+)$", func(ctx context.Context, link string) (context.Context, error) {
				return ctx, nil
			}).
			RegisterStep("^user will be logged out$", func(ctx context.Context) (context.Context, error) {
				return ctx, nil
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
			RegisterStep("^.*$", func(ctx context.Context) (context.Context, error) {
				stepExecuted = true
				return ctx, nil
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
			RegisterStep("^hello$", func(ctx context.Context) (context.Context, error) {
				stepExecuted = true
				return ctx, nil
			}).
			RegisterStep("^user is logged in$", func(ctx context.Context) (context.Context, error) {
				return ctx, nil
			}).
			RegisterStep("^user clicks (.+)$", func(ctx context.Context, link string) (context.Context, error) {
				return ctx, nil
			}).
			RegisterStep("^user will be logged out$", func(ctx context.Context) (context.Context, error) {
				return ctx, nil
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
