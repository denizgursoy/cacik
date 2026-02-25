package step_path

import (
	"github.com/denizgursoy/cacik/pkg/cacik"
)

// MatchPath uses built-in {path} type
// @cacik `^the file is at {path}$`
func MatchPath(ctx *cacik.Context, p string) {
	ctx.Logger().Info("path", "p", p)
}
