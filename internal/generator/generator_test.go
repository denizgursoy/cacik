package generator

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestStartApplication(t *testing.T) {
	t.Run("should call code parser with the working directory", func(t *testing.T) {
		controller := gomock.NewController(t)
		mockGoCodeParser := NewMockGoCodeParser(controller)

		dir, _ := os.Getwd()
		mockGoCodeParser.
			EXPECT().
			ParseFunctionCommentsOfGoFilesInDirectoryRecursively(gomock.Any(), dir).
			Return([]FunctionLocator{}, nil).
			Times(1)

		err := StartGenerator(context.Background(), mockGoCodeParser)
		require.Nil(t, err)
	})

	t.Run("should get directories from flags", func(t *testing.T) {
		controller := gomock.NewController(t)
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

		err := StartGenerator(context.Background(), mockGoCodeParser)
		require.Nil(t, err)
	})
}
