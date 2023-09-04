package app

import (
	"context"
	"os"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"
)

func TestStartApplication(t *testing.T) {
	t.Run("should call code parser with the working directory", func(t *testing.T) {
		controller := gomock.NewController(t)
		mockGherkinParser := NewMockGherkinParser(controller)
		mockGoCodeParser := NewMockGoCodeParser(controller)

		dir, _ := os.Getwd()
		mockGoCodeParser.
			EXPECT().
			ParseFunctionCommentsOfGoFilesInDirectoryRecursively(gomock.Any(), dir).
			Return([]FunctionLocator{}, nil).
			Times(1)

		StartApplication(context.Background(), mockGoCodeParser, mockGherkinParser)
	})

	t.Run("should get directories from flags", func(t *testing.T) {
		controller := gomock.NewController(t)
		mockGherkinParser := NewMockGherkinParser(controller)
		mockGoCodeParser := NewMockGoCodeParser(controller)

		expectedPath := "/etc,/home"
		os.Args = []string{"x", "--code", expectedPath}

		for _, s := range strings.Split(expectedPath, Separator) {
			mockGoCodeParser.
				EXPECT().
				ParseFunctionCommentsOfGoFilesInDirectoryRecursively(gomock.Any(), s).
				Return([]FunctionLocator{}, nil).
				Times(1)
		}

		StartApplication(context.Background(), mockGoCodeParser, mockGherkinParser)
	})
}
