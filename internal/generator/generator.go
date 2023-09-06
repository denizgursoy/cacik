package generator

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

func StartGenerator(ctx context.Context, codeParser GoCodeParser) error {
	funcSources := make([]string, 0)

	codeFlag := flag.String("code", "", "directories to search for functions seperated by comma")
	flag.Parse()

	if len(strings.TrimSpace(*codeFlag)) == 0 {
		directory, err := os.Getwd()
		if err != nil {
			log.Println(err.Error())
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
		err = recursively.Generate(create)

		if err != nil {
			log.Println(err.Error())
			return err
		}
	}

	return nil
}
