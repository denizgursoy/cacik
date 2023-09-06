package main

import (
	"context"
	"os"

	"github.com/denizgursoy/cacik/internal/comment_parser"
	"github.com/denizgursoy/cacik/internal/generator"
)

func main() {
	err := generator.StartGenerator(context.Background(), comment_parser.NewGoSourceFileParser())
	if err != nil {
		os.Exit(1)
	}
}
