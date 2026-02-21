package step_duplicate

import (
	"context"
	"fmt"
)

// SecondDuplicateStep is the second definition of the same step pattern
// @cacik `^I have (\d+) items$`
func SecondDuplicateStep(ctx context.Context, count int) (context.Context, error) {
	fmt.Printf("Second: I have %d items\n", count)
	return ctx, nil
}
