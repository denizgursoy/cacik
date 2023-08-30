package app

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

const (
	Seperator = ","
)

func StartApplication(codeParser GoCodeParser, gherkinParser GherkinParser) {
	funcSources := make([]string, 0)

	ctx := context.Background()

	codeFlag := flag.String("code", "", "directories to search for functions seperated by comma")
	flag.Parse()

	if len(strings.TrimSpace(*codeFlag)) == 0 {
		dir, err := os.Getwd()
		if err != nil {
			log.Fatal(err.Error())
		}
		funcSources = append(funcSources, dir)
	} else {
		funcSources = append(funcSources, strings.Split(*codeFlag, Seperator)...)
	}

	for _, source := range funcSources {
		recursively, err := codeParser.ParseFunctionCommentsOfGoFilesInDirectoryRecursively(ctx, source)
		if err != nil {
			log.Fatal(err)
			return
		}
		fmt.Println(recursively)
	}

}
