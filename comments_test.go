package cacik

import (
	"testing"

	"github.com/denizgursoy/cacik/internal/parser"
)

func TestGetComments(t *testing.T) {
	t.Run("", func(t *testing.T) {
		parser.GetComments("/home/dgursoy/projects/go/src/cacik/pkg/testdata/")
	})
}
