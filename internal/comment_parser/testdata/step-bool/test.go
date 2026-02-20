package step_bool

import (
	"context"
	"fmt"
)

// ItIs handles boolean steps like "it is true" or "it is enabled"
// @cacik `^it is (true|false|yes|no|on|off|enabled|disabled)$`
func ItIs(ctx context.Context, value bool) (context.Context, error) {
	fmt.Printf("it is %v\n", value)
	return ctx, nil
}

// FeatureToggle handles feature state toggling
// @cacik `^the feature is (enabled|disabled)$`
func FeatureToggle(ctx context.Context, enabled bool) (context.Context, error) {
	if enabled {
		fmt.Println("Feature is ON")
	} else {
		fmt.Println("Feature is OFF")
	}
	return ctx, nil
}
