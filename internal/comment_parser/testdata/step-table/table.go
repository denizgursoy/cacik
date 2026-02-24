package step_table

import (
	"fmt"

	"github.com/denizgursoy/cacik/pkg/cacik"
)

// TheFollowingUsers receives a DataTable of users
// @cacik `^the following users:$`
func TheFollowingUsers(ctx *cacik.Context, table cacik.Table) {
	for _, row := range table.SkipHeader() {
		ctx.Logger().Info("user", "name", row.Get("name"), "age", row.Get("age"))
	}
}

// ThereShouldBeNUsers asserts the expected user count
// @cacik `^there should be {int} users$`
func ThereShouldBeNUsers(ctx *cacik.Context, expected int) {
	ctx.Logger().Info("checking user count", "expected", expected)
}

// IHaveItems receives a count and a DataTable of items
// @cacik `^I have {int} items:$`
func IHaveItems(ctx *cacik.Context, count int, table cacik.Table) {
	ctx.Logger().Info("items", "count", count)
	for _, row := range table.SkipHeader() {
		ctx.Logger().Info("item", "name", row.Get("item"), "price", row.Get("price"))
	}
}

// Coordinates receives a headerless DataTable of coordinates
// @cacik `^the coordinates are:$`
func Coordinates(table cacik.Table) {
	for _, row := range table.All() {
		x := row.Cell(0)
		y := row.Cell(1)
		fmt.Println(x, y)
	}
}
