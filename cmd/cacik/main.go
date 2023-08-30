package main

import (
	"github.com/denizgursoy/cacik/internal/app"
	"github.com/denizgursoy/cacik/internal/parser"
)

func main() {
	app.StartApplication(parser.NewGoSourceFileParser(), nil)
}
