package step_priority

import (
	"github.com/denizgursoy/cacik/pkg/cacik"
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
func SetPriority(ctx *cacik.Context, p Priority) {
	ctx.Logger().Info("priority set", "priority", p)
}

// PriorityIs checks if the priority matches
// @cacik `^the priority is {priority}$`
func PriorityIs(ctx *cacik.Context, p Priority) {
	ctx.Logger().Info("priority is", "priority", p)
}
