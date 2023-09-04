package main

import (
	"context"
	"os"

	"github.com/denizgursoy/cacik/internal/app"
	"github.com/denizgursoy/cacik/internal/parser"
)

func main() {
	err := app.StartApplication(context.Background(), parser.NewGoSourceFileParser(), nil)
	if err != nil {
		os.Exit(1)
	}
}
