package step_string

import (
	"context"
	"fmt"
)

// UserSays captures a quoted string using {string} built-in type
// @cacik `^the user says {string}$`
func UserSays(ctx context.Context, message string) (context.Context, error) {
	fmt.Printf("The user says: %s\n", message)
	return ctx, nil
}

// ErrorMessageIs checks the error message
// @cacik `^the error message is {string}$`
func ErrorMessageIs(ctx context.Context, errMsg string) (context.Context, error) {
	fmt.Printf("Error message: %s\n", errMsg)
	return ctx, nil
}

// TitleIs checks the title using {word} for single word
// @cacik `^the title is {word}$`
func TitleIs(ctx context.Context, title string) (context.Context, error) {
	fmt.Printf("Title: %s\n", title)
	return ctx, nil
}
