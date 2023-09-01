package generator

import (
	_ "embed"
	"log"
	"os"
	"text/template"
)

type (
	GoCodeGenerator struct {
	}
)

var (
	//go:embed testdata/output.go
	templateContent string
)

func NewGoCodeGenerator() *GoCodeGenerator {
	return &GoCodeGenerator{}
}

func (g *GoCodeGenerator) Generate() {
	tmpl := template.Must(template.New("goCode").Parse(templateContent))

	file, err := os.Create("main.go")
	if err != nil {
		log.Fatal(err.Error())
	}
	data := struct {
		Code string
	}{
		Code: "fmt.Println(\"Hello, World!\")",
	}

	err = tmpl.Execute(file, data)
	if err != nil {
		log.Fatal(err.Error())
	}
}
