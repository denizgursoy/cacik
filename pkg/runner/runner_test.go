package runner

import (
	"context"
	"testing"

	messages "github.com/cucumber/messages/go/v21"
	"github.com/stretchr/testify/require"
)

func Test_includeTags(t *testing.T) {
	t.Run("should return true if tags contains", func(t *testing.T) {
		documentTags := []*messages.Tag{
			{
				Name: "@test",
			},
		}
		userTags := []string{"test"}

		require.True(t, includeTags(documentTags, userTags))
	})
	t.Run("should return false if tags do not contain user tag", func(t *testing.T) {
		documentTags := []*messages.Tag{}
		userTags := []string{"test"}

		require.False(t, includeTags(documentTags, userTags))
	})
}

func TestCucumberRunner_RunWithTags(t *testing.T) {
	t.Run("should execute feature with matching tag", func(t *testing.T) {
		// Track if step was executed
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

		err := runner.RunWithTags("billing")
		require.Nil(t, err)
		require.True(t, stepExecuted, "expected step to be executed")
	})

	t.Run("should not execute feature if tags do not match", func(t *testing.T) {
		stepExecuted := false

		runner := NewCucumberRunner().
			WithFeaturesDirectories("testdata/without-tag").
			RegisterStep("^.*$", func(ctx context.Context) (context.Context, error) {
				stepExecuted = true
				return ctx, nil
			})

		err := runner.RunWithTags("test")
		require.Nil(t, err)
		require.False(t, stepExecuted, "expected step NOT to be executed")
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
