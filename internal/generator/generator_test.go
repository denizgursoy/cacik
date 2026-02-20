package generator

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// NOTE: These tests are skipped because StartGenerator uses flag.String inside the function,
// which causes "flag redefined" panics when tests run multiple times.
// TODO: Refactor StartGenerator to accept flags as parameters or use a FlagSet.
func TestStartApplication(t *testing.T) {
	t.Skip("Skipping due to flag redefinition issue - needs refactoring")

	t.Run("should call code parser with the working directory", func(t *testing.T) {
		controller := gomock.NewController(t)
		mockGoCodeParser := NewMockGoCodeParser(controller)

		dir, _ := os.Getwd()
		mockGoCodeParser.
			EXPECT().
			ParseFunctionCommentsOfGoFilesInDirectoryRecursively(gomock.Any(), dir).
			Return(&Output{StepFunctions: []*StepFunctionLocator{}}, nil).
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
				Return(&Output{StepFunctions: []*StepFunctionLocator{}}, nil).
				Times(1)
		}

		err := StartGenerator(context.Background(), mockGoCodeParser)
		require.Nil(t, err)
	})
}
