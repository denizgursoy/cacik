package main

import (
	"github.com/denizgursoy/cacik/internal/app"
	"github.com/denizgursoy/cacik/internal/generator"
	"github.com/denizgursoy/cacik/internal/parser"
)

func main() {
	codeGenerator := generator.NewGoCodeGenerator()
	codeGenerator.Name()
	app.StartApplication(parser.NewGoSourceFileParser(), nil)
}
