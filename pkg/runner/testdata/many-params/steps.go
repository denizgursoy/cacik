package many_params

import (
	"fmt"

	"github.com/denizgursoy/cacik/pkg/cacik"
)

// ARecordWith handles a step that captures 10 parameters, exercising every
// color in the reporter's colorParams palette.
// @cacik `^a record with "([^"]*)" aged (\d+) scored ([\d.]+) from "([^"]*)" tagged "([^"]*)" on level (\d+) with code "([^"]*)" rated ([\d.]+) and flag (true|false) in group "([^"]*)"$`
func ARecordWith(ctx *cacik.Context, name string, age int, score float64, city string, tag string, level int, code string, rating float64, flag bool, group string) {
	ctx.Logger().Info("record",
		"name", name,
		"age", age,
		"score", fmt.Sprintf("%.1f", score),
		"city", city,
		"tag", tag,
		"level", level,
		"code", code,
		"rating", fmt.Sprintf("%.1f", rating),
		"flag", flag,
		"group", group,
	)
}
