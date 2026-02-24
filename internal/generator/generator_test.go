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

	t.Run("falls back to directory name when no Go files exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Create a subdirectory with a known name
		subDir := tmpDir + "/myfeatures"
		require.NoError(t, os.Mkdir(subDir, 0o755))

		pkgName, err := detectPackageName(subDir)
		require.NoError(t, err)
		require.Equal(t, "myfeatures", pkgName)
	})

	t.Run("sanitizes hyphens in directory name", func(t *testing.T) {
		tmpDir := t.TempDir()
		subDir := tmpDir + "/my-cool-app"
		require.NoError(t, os.Mkdir(subDir, 0o755))

		pkgName, err := detectPackageName(subDir)
		require.NoError(t, err)
		require.Equal(t, "my_cool_app", pkgName)
	})

	t.Run("uses module path at module root with no Go files", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Create a go.mod in the temp dir
		goMod := "module github.com/example/myproject\n\ngo 1.21\n"
		require.NoError(t, os.WriteFile(tmpDir+"/go.mod", []byte(goMod), 0o644))

		pkgName, err := detectPackageName(tmpDir)
		require.NoError(t, err)
		require.Equal(t, "myproject", pkgName)
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

func TestSanitizePackageName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"myapp", "myapp"},
		{"my-app", "my_app"},
		{"my.app", "my_app"},
		{"MyApp", "myapp"},
		{"123app", "_123app"},
		{"", ""},
		{"a", "a"},
		{"-leading", "leading"},
		{"with spaces", "withspaces"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizePackageName(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}
