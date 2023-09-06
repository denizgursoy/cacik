package gherkin_parser

import (
	"io"
	"io/fs"
	"log"
	"path/filepath"
	"strings"

	gherkin "github.com/cucumber/gherkin/go/v26"
	messages "github.com/cucumber/messages/go/v21"
)

const (
	FeatureExtension = ".feature"
)

func SearchFeatureFilesIn(directories []string) ([]string, error) {
	featureFiles := make([]string, 0)

	for _, directory := range directories {
		err := filepath.Walk(directory, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				log.Println(err)
				return err
			}
			if !info.IsDir() {
				if strings.HasSuffix(info.Name(), FeatureExtension) {
					featureFiles = append(featureFiles, path)
				}
			}
			return nil
		})

		if err != nil {
			log.Println(err)
			return nil, err
		}
	}
	return featureFiles, nil
}

func ParseGherkinFile(reader io.Reader) (*messages.GherkinDocument, error) {
	id := (&messages.Incrementing{}).NewId
	document, err := gherkin.ParseGherkinDocument(reader, id)
	if err != nil {

		return nil, err
	}
	return document, nil
}
