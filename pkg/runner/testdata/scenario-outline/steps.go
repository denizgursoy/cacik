package scenario_outline

import (
	"fmt"

	"github.com/denizgursoy/cacik/pkg/cacik"
)

// TheApplicationIsStarted initializes the application
// @cacik `^the application is started$`
func TheApplicationIsStarted(ctx *cacik.Context) {
	ctx.Logger().Info("application started")
}

// UserExistsWithRole sets up a user with a given role
// @cacik `^user "([^"]*)" exists with role "([^"]*)"$`
func UserExistsWithRole(ctx *cacik.Context, username, role string) {
	ctx.Data().Set("user:"+username+":role", role)
	ctx.Logger().Info("user exists", "username", username, "role", role)
}

// UserLogsInWithPassword attempts a login
// @cacik `^user "([^"]*)" logs in with password "([^"]*)"$`
func UserLogsInWithPassword(ctx *cacik.Context, username, password string) {
	ctx.Logger().Info("login attempt", "username", username)
}

// TheLoginResultShouldBe verifies the login outcome
// @cacik `^the login result should be "([^"]*)"$`
func TheLoginResultShouldBe(ctx *cacik.Context, result string) {
	ctx.Logger().Info("login result", "result", result)
}

// TheUserRoleShouldBe verifies the user's role
// @cacik `^the user role should be "([^"]*)"$`
func TheUserRoleShouldBe(ctx *cacik.Context, role string) {
	ctx.Logger().Info("user role", "role", role)
}

// IAssignPermissions assigns permissions from a DataTable to a user
// @cacik `^I assign permissions to "([^"]*)":$`
func IAssignPermissions(ctx *cacik.Context, username string, table cacik.Table) {
	for _, row := range table.SkipHeader() {
		perm := row.Get("permission")
		granted := row.Get("granted")
		ctx.Logger().Info("assign permission",
			"user", username,
			"permission", perm,
			"granted", granted,
		)
	}
}

// UserShouldHaveNPermissions verifies permission count
// @cacik `^user "([^"]*)" should have {int} permissions$`
func UserShouldHaveNPermissions(ctx *cacik.Context, username string, count int) {
	ctx.Logger().Info("permission count", "user", username, "expected", count)
}

// TheApplicationIsRunning checks the app is running
// @cacik `^the application is running$`
func TheApplicationIsRunning(ctx *cacik.Context) {
	ctx.Logger().Info("application is running")
}

// ICheckTheStatus performs a status check
// @cacik `^I check the status$`
func ICheckTheStatus(ctx *cacik.Context) {
	ctx.Logger().Info("checking status")
}

// TheStatusCodeShouldBe verifies the HTTP status code
// @cacik `^the status code should be {int}$`
func TheStatusCodeShouldBe(ctx *cacik.Context, code int) {
	ctx.Logger().Info("status code", "code", code)
}

// TheAccessControlModuleIsLoaded initializes the ACL module
// @cacik `^the access control module is loaded$`
func TheAccessControlModuleIsLoaded(ctx *cacik.Context) {
	ctx.Logger().Info("access control module loaded")
}

// UserHasRole sets a user's role for access control
// @cacik `^user "([^"]*)" has role "([^"]*)"$`
func UserHasRole(ctx *cacik.Context, user, role string) {
	ctx.Data().Set("acl:"+user+":role", role)
	ctx.Logger().Info("user has role", "user", user, "role", role)
}

// UserAccessesResource attempts to access a resource
// @cacik `^user "([^"]*)" accesses "([^"]*)"$`
func UserAccessesResource(ctx *cacik.Context, user, resource string) {
	ctx.Logger().Info("access attempt", "user", user, "resource", resource)
}

// AccessShouldBe verifies the access decision
// @cacik `^access should be "([^"]*)"$`
func AccessShouldBe(ctx *cacik.Context, decision string) {
	if decision != "granted" && decision != "denied" {
		panic(fmt.Sprintf("unexpected access decision: %s", decision))
	}
	ctx.Logger().Info("access decision", "decision", decision)
}
