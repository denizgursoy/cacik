package step_semver

import (
	"github.com/denizgursoy/cacik/pkg/cacik"
)

// MatchSemver uses built-in {semver} type
// @cacik `^the version is {semver}$`
func MatchSemver(ctx *cacik.Context, ver string) {
	ctx.Logger().Info("semver", "ver", ver)
}
