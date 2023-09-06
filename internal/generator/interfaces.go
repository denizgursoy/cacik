//go:generate mockgen -source=interfaces.go -destination=interface_mock.go -package=generator
package generator

import "context"

type (
	GoCodeParser interface {
		ParseFunctionCommentsOfGoFilesInDirectoryRecursively(context.Context, string) (*Output, error)
	}
	GherkinParser interface {
	}
)
