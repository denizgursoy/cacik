package main

import (
	"log"

	"github.com/denizgursoy/cacik/pkg/runner"
)

func main() {
	err := runner.
		NewCucumberRunner().
		{{if .ConfigFunction}}SetConfig({{.ConfigFunction.PackageName}}.{{ConfigFunction.Name}}).{{end}}
		{{range .StepFunctions}}RegisterStep({{.PackageName}}.{{.Name}}).{{end}}
		Run()

	if err != nil {
		log.Fatal(err)
	}
}
