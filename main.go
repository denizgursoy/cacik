package main

import (
	testdata "github.com/denizgursoy/cacik/internal/comment_parser/testdata"
	stepbool "github.com/denizgursoy/cacik/internal/comment_parser/testdata/step-bool"
	stepbuiltin "github.com/denizgursoy/cacik/internal/comment_parser/testdata/step-builtin"
	stepcolor "github.com/denizgursoy/cacik/internal/comment_parser/testdata/step-color"
	stepfloat "github.com/denizgursoy/cacik/internal/comment_parser/testdata/step-float"
	stepint "github.com/denizgursoy/cacik/internal/comment_parser/testdata/step-int"
	stepone "github.com/denizgursoy/cacik/internal/comment_parser/testdata/step-one"
	steppriority "github.com/denizgursoy/cacik/internal/comment_parser/testdata/step-priority"
	stepstring "github.com/denizgursoy/cacik/internal/comment_parser/testdata/step-string"
	steptwo "github.com/denizgursoy/cacik/internal/comment_parser/testdata/step-two"
	runner "github.com/denizgursoy/cacik/pkg/runner"
	"log"
)

func main() {
	err := runner.NewCucumberRunner().
		WithConfigFunc(testdata.Method1).
		RegisterCustomType("Priority", "int", map[string]string{
			"1":      "1",
			"2":      "2",
			"3":      "3",
			"high":   "3",
			"low":    "1",
			"medium": "2",
		}).
		RegisterCustomType("Color", "string", map[string]string{
			"blue":  "blue",
			"green": "green",
			"red":   "red",
		}).
		RegisterStep("^I have (-?\\d+) apples$", stepbuiltin.HaveApples).
		RegisterStep("^the price is (-?\\d*\\.?\\d+)$", stepbuiltin.PriceIs).
		RegisterStep("^my name is (\\w+)$", stepbuiltin.NameIs).
		RegisterStep("^I say \"([^\"]*)\"$", stepbuiltin.Say).
		RegisterStep("^I see (.*)$", stepbuiltin.SeeAnything).
		RegisterStep("^priority is (1|2|3|high|low|medium)$", steppriority.SetPriority).
		RegisterStep("^the priority is (1|2|3|high|low|medium)$", steppriority.PriorityIs).
		RegisterStep("^the user says \"([^\"]*)\"$", stepstring.UserSays).
		RegisterStep("^the error message is \"([^\"]*)\"$", stepstring.ErrorMessageIs).
		RegisterStep("^the title is (\\w+)$", stepstring.TitleIs).
		RegisterStep("^it is (true|false|yes|no|on|off|enabled|disabled)$", stepbool.ItIs).
		RegisterStep("^the feature is (enabled|disabled)$", stepbool.FeatureToggle).
		RegisterStep("^I select (blue|green|red)$", stepcolor.SelectColor).
		RegisterStep("^the color is (blue|green|red)$", stepcolor.ColorIs).
		RegisterStep("^step 2$", steptwo.Step2).
		RegisterStep("^step 1$", stepone.Step1).
		RegisterStep("^the item costs (-?\\d*\\.?\\d+) dollars$", stepfloat.ItemCosts).
		RegisterStep("^the temperature is (-?\\d*\\.?\\d+) degrees$", stepfloat.TemperatureIs).
		RegisterStep("^I have (\\d+) apples$", stepint.IGetApples).
		RunWithTags()

	if err != nil {
		log.Fatal(err)
	}
}
