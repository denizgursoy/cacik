package step_int

import (
	"context"
	"fmt"
)

// IGetApples
// @cacik `^I have (\d+) apples$`
func IGetApples(ctx context.Context, appleCount int) (context.Context, error) {
	fmt.Printf("I have %d apples", appleCount)

	return ctx, nil
}
