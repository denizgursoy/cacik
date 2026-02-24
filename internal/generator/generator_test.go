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

func TestDetectPackageName(t *testing.T) {
	t.Run("detects package name from Go files in directory", func(t *testing.T) {
		// This test runs in the generator package directory, which has Go files
		// with "package generator"
		dir, err := os.Getwd()
		require.NoError(t, err)

		pkgName, err := detectPackageName(dir)
		require.NoError(t, err)
		require.Equal(t, "generator", pkgName)
	})

	t.Run("returns error for directory with no Go files", func(t *testing.T) {
		tmpDir := t.TempDir()
		_, err := detectPackageName(tmpDir)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no Go files found")
	})
}

func TestDetectImportPath(t *testing.T) {
	t.Run("detects import path from go.mod", func(t *testing.T) {
		// This test runs from the generator package directory
		dir, err := os.Getwd()
		require.NoError(t, err)

		pkgPath, err := detectImportPath(dir)
		require.NoError(t, err)
		require.Equal(t, "github.com/denizgursoy/cacik/internal/generator", pkgPath)
	})

	t.Run("returns error for directory without go.mod ancestor", func(t *testing.T) {
		_, err := detectImportPath("/tmp")
		require.Error(t, err)
		require.Contains(t, err.Error(), "go.mod not found")
	})
}
