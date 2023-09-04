package app

import (
	"context"
	"flag"
	"log"
	"os"
	"strings"
)

const (
	Separator = ","
)

func StartApplication(ctx context.Context, codeParser GoCodeParser, gherkinParser GherkinParser) error {
	funcSources := make([]string, 0)

	codeFlag := flag.String("code", "", "directories to search for functions seperated by comma")
	flag.Parse()

	if len(strings.TrimSpace(*codeFlag)) == 0 {
		directory, err := os.Getwd()
		if err != nil {
			return err
		}
		funcSources = append(funcSources, directory)
	} else {
		funcSources = append(funcSources, strings.Split(*codeFlag, Separator)...)
	}

	for _, source := range funcSources {
		recursively, err := codeParser.ParseFunctionCommentsOfGoFilesInDirectoryRecursively(ctx, source)
		if err != nil {
			log.Println(err.Error())
			return err
		}
		create, err := os.Create("main.go")
		if err != nil {
			return err
		}
		recursively.Generate(create)
	}

	return nil
}
