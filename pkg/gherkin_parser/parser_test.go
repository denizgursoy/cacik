package gherkin_parser

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Run("should return feature", func(t *testing.T) {
		file, err := os.ReadFile("testdata/feat.feature")
		if err != nil {
			return
		}
		ParseGherkinFile(strings.NewReader(string(file)))

		require.Nil(t, err)

	})
}

func TestSearchFeatureFilesIn(t *testing.T) {
	t.Run("should return all feature files in a directory", func(t *testing.T) {
		expectedFiles := []string{
			"testdata/feat.feature",
			"testdata/feature-source-1/source-one.feature",
			"testdata/feature-source-2/source-two.feature",
			"testdata/source-three.feature",
		}

		actualFiles, err := SearchFeatureFilesIn([]string{"testdata"})

		require.Nil(t, err)
		require.Equal(t, expectedFiles, actualFiles)
	})
}
