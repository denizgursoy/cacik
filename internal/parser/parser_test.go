package parser

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
		Parse(strings.NewReader(string(file)))

		require.Nil(t, err)

	})
}
