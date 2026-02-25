package step_bigint

import (
	"math/big"

	"github.com/denizgursoy/cacik/pkg/cacik"
)

// MatchBigint uses built-in {bigint} type
// @cacik `^the large number is {bigint}$`
func MatchBigint(ctx *cacik.Context, n *big.Int) {
	ctx.Logger().Info("bigint", "n", n)
}
