package app

import (
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
			Return([]FunctionDescriptor{}, nil).
			Times(1)

		StartApplication(mockGoCodeParser, mockGherkinParser)
	})

	t.Run("should get directories from flags", func(t *testing.T) {
		controller := gomock.NewController(t)
		mockGherkinParser := NewMockGherkinParser(controller)
		mockGoCodeParser := NewMockGoCodeParser(controller)

		expectedPath := "/etc,/home"
		os.Args = []string{"x", "--code", expectedPath}

		for _, s := range strings.Split(expectedPath, Seperator) {
			mockGoCodeParser.
				EXPECT().
				ParseFunctionCommentsOfGoFilesInDirectoryRecursively(gomock.Any(), s).
				Return([]FunctionDescriptor{}, nil).
				Times(1)
		}

		StartApplication(mockGoCodeParser, mockGherkinParser)
	})
}
