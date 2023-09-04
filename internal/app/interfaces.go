//go:generate mockgen -source=interfaces.go -destination=interface_mock.go -package=app
package app

import "context"

type (
	GoCodeParser interface {
		ParseFunctionCommentsOfGoFilesInDirectoryRecursively(context.Context, string) (*Output, error)
	}
	GherkinParser interface {
	}
)
