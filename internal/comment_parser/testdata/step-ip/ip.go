package step_ip

import (
	"net"

	"github.com/denizgursoy/cacik/pkg/cacik"
)

// MatchIP uses built-in {ip} type
// @cacik `^the server is at {ip}$`
func MatchIP(ctx *cacik.Context, addr net.IP) {
	ctx.Logger().Info("ip", "addr", addr)
}
