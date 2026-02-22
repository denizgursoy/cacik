package step_bool

import (
	"github.com/denizgursoy/cacik/pkg/cacik"
)

// ItIs handles boolean steps like "it is true" or "it is enabled"
// @cacik `^it is (true|false|yes|no|on|off|enabled|disabled)$`
func ItIs(ctx *cacik.Context, value bool) {
	ctx.Logger().Info("it is", "value", value)
}

// FeatureToggle handles feature state toggling
// @cacik `^the feature is (enabled|disabled)$`
func FeatureToggle(ctx *cacik.Context, enabled bool) {
	ctx.Logger().Info("feature toggled", "enabled", enabled)
}
