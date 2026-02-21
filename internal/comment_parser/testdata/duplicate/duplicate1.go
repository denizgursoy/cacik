package step_duplicate

import (
	"context"
	"fmt"
)

// FirstDuplicateStep is the first definition of a duplicate step
// @cacik `^I have (\d+) items$`
func FirstDuplicateStep(ctx context.Context, count int) (context.Context, error) {
	fmt.Printf("First: I have %d items\n", count)
	return ctx, nil
}
