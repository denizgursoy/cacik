package parser

import (
	"io"

	gherkin "github.com/cucumber/gherkin/go/v26"
	messages "github.com/cucumber/messages/go/v21"
)

func Parse(reader io.Reader) {
	id := (&messages.Incrementing{}).NewId
	document, err := gherkin.ParseGherkinDocument(reader, id)
	if err != nil {
		return
	}
	println(document)
	return
}
