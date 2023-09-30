package runner

import (
	"bytes"
	"os"
	"testing"

	messages "github.com/cucumber/messages/go/v21"
	"github.com/denizgursoy/cacik/pkg/gherkin_parser"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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
	t.Run("should call executor by tags", func(t *testing.T) {
		controller := gomock.NewController(t)
		defer controller.Finish()
		executor := NewMockExecutor(controller)

		readFile, err := os.ReadFile("testdata/with-tag/a.feature")
		require.Nil(t, err)
		document, err := gherkin_parser.ParseGherkinFile(bytes.NewReader(readFile))
		require.Nil(t, err)

		executor.EXPECT().Execute(document).Times(1)

		runner := NewCucumberRunner(executor).WithFeaturesDirectories("testdata/with-tag")
		err = runner.RunWithTags("test")

		require.Nil(t, err)
	})
	t.Run("should not call executor if tags does not match", func(t *testing.T) {
		controller := gomock.NewController(t)
		defer controller.Finish()
		executor := NewMockExecutor(controller)

		runner := NewCucumberRunner(executor).WithFeaturesDirectories("testdata/without-tag")
		err := runner.RunWithTags("test")

		require.Nil(t, err)
	})
}
