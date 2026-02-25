package step_regex

import (
	"regexp"

	"github.com/denizgursoy/cacik/pkg/cacik"
)

// MatchRegex uses built-in {regex} type
// @cacik `^the pattern is {regex}$`
func MatchRegex(ctx *cacik.Context, re *regexp.Regexp) {
	ctx.Logger().Info("regex", "re", re)
}
