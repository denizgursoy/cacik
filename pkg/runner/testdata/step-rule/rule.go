package step_rule

import (
	"github.com/denizgursoy/cacik/pkg/cacik"
)

// SystemInitialized sets up the system (feature-level background)
// @cacik `^the system is initialized$`
func SystemInitialized(ctx *cacik.Context) {
	ctx.Logger().Info("system initialized")
}

// RegistrationFormLoaded loads the registration form (rule-level background)
// @cacik `^the registration form is loaded$`
func RegistrationFormLoaded(ctx *cacik.Context) {
	ctx.Logger().Info("registration form loaded")
}

// LoginPageLoaded loads the login page (rule-level background)
// @cacik `^the login page is loaded$`
func LoginPageLoaded(ctx *cacik.Context) {
	ctx.Logger().Info("login page loaded")
}

// UserRegisters handles user registration with an email
// @cacik `^the user registers with {string}$`
func UserRegisters(ctx *cacik.Context, email string) {
	ctx.Logger().Info("user registers", "email", email)
}

// RegistrationSucceed asserts that the registration succeeded
// @cacik `^the registration should succeed$`
func RegistrationSucceed(ctx *cacik.Context) {
	ctx.Logger().Info("registration succeeded")
}

// RegistrationFail asserts that the registration failed
// @cacik `^the registration should fail$`
func RegistrationFail(ctx *cacik.Context) {
	ctx.Logger().Info("registration failed")
}

// UserLogsIn handles user login with credentials
// @cacik `^the user logs in with {string} and {string}$`
func UserLogsIn(ctx *cacik.Context, username string, password string) {
	ctx.Logger().Info("user logs in", "username", username, "password", password)
}

// LoginSucceed asserts that the login succeeded
// @cacik `^the login should succeed$`
func LoginSucceed(ctx *cacik.Context) {
	ctx.Logger().Info("login succeeded")
}

// LoginFail asserts that the login failed
// @cacik `^the login should fail$`
func LoginFail(ctx *cacik.Context) {
	ctx.Logger().Info("login failed")
}
