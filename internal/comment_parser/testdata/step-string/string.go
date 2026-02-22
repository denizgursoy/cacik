package step_string

import (
	"github.com/denizgursoy/cacik/pkg/cacik"
)

// UserSays captures a quoted string using {string} built-in type
// @cacik `^the user says {string}$`
func UserSays(ctx *cacik.Context, message string) {
	ctx.Logger().Info("user says", "message", message)
}

// ErrorMessageIs checks the error message
// @cacik `^the error message is {string}$`
func ErrorMessageIs(ctx *cacik.Context, errMsg string) {
	ctx.Logger().Info("error message is", "message", errMsg)
}

// TitleIs checks the title using {word} for single word
// @cacik `^the title is {word}$`
func TitleIs(ctx *cacik.Context, title string) {
	ctx.Logger().Info("title is", "title", title)
}
