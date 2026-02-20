package step_priority

import (
	"context"
	"fmt"
)

// Priority represents a priority level
type Priority int

const (
	Low    Priority = 1
	Medium Priority = 2
	High   Priority = 3
)

// SetPriority sets the priority level
// @cacik `^priority is {priority}$`
func SetPriority(ctx context.Context, p Priority) (context.Context, error) {
	fmt.Printf("Priority set to: %d\n", p)
	return ctx, nil
}

// PriorityIs checks if the priority matches
// @cacik `^the priority is {priority}$`
func PriorityIs(ctx context.Context, p Priority) (context.Context, error) {
	fmt.Printf("The priority is: %d\n", p)
	return ctx, nil
}
