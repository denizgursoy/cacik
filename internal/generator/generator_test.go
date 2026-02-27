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

		pkgName, err := detectPackageName(dir, "cacik_test.go")
		require.NoError(t, err)
		require.Equal(t, "generator", pkgName)
	})

	t.Run("returns empty string when no Go files exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		subDir := tmpDir + "/myfeatures"
		require.NoError(t, os.Mkdir(subDir, 0o755))

		pkgName, err := detectPackageName(subDir, "cacik_test.go")
		require.NoError(t, err)
		require.Equal(t, "", pkgName)
	})

	t.Run("skips generated output file when detecting package", func(t *testing.T) {
		tmpDir := t.TempDir()
		subDir := tmpDir + "/myapp"
		require.NoError(t, os.Mkdir(subDir, 0o755))

		// Only Go file is the generated output file — should be skipped, returns ""
		require.NoError(t, os.WriteFile(subDir+"/cacik_test.go", []byte("package myapp"), 0o644))

		pkgName, err := detectPackageName(subDir, "cacik_test.go")
		require.NoError(t, err)
		require.Equal(t, "", pkgName)
	})

	t.Run("reads package from non-output Go file even when output file exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		subDir := tmpDir + "/myapp"
		require.NoError(t, os.Mkdir(subDir, 0o755))

		// Output file should be skipped
		require.NoError(t, os.WriteFile(subDir+"/billing_test.go", []byte("package wrong"), 0o644))
		// Real source file provides the package name
		require.NoError(t, os.WriteFile(subDir+"/app.go", []byte("package myapp"), 0o644))

		pkgName, err := detectPackageName(subDir, "billing_test.go")
		require.NoError(t, err)
		require.Equal(t, "myapp", pkgName)
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

func TestTestFuncNameFromPrefix(t *testing.T) {
	tests := []struct {
		prefix   string
		expected string
	}{
		{"cacik", "TestCacik"},
		{"billing", "TestBilling"},
		{"my_feature", "TestMyFeature"},
		{"a", "TestA"},
		{"", "TestCacik"}, // empty defaults to "cacik"
		{"user_auth_flow", "TestUserAuthFlow"},
		{"x_y_z", "TestXYZ"},
	}

	for _, tt := range tests {
		t.Run(tt.prefix, func(t *testing.T) {
			result := testFuncNameFromPrefix(tt.prefix)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestOutputFileFromPrefix(t *testing.T) {
	tests := []struct {
		prefix   string
		expected string
	}{
		{"cacik", "cacik_test.go"},
		{"billing", "billing_test.go"},
		{"my_feature", "my_feature_test.go"},
		{"", "cacik_test.go"}, // empty defaults to "cacik"
	}

	for _, tt := range tests {
		t.Run(tt.prefix, func(t *testing.T) {
			result := outputFileFromPrefix(tt.prefix)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateExportedFunctions(t *testing.T) {
	t.Run("passes when all functions are exported", func(t *testing.T) {
		output := &Output{
			CurrentPackagePath: "github.com/example/myapp",
			ConfigFunctions: []*FunctionLocator{
				{FullPackageName: "github.com/example/myapp/config", FunctionName: "GetConfig", IsExported: true},
			},
			HooksFunctions: []*FunctionLocator{
				{FullPackageName: "github.com/example/myapp/hooks", FunctionName: "GetHooks", IsExported: true},
			},
			StepFunctions: []*StepFunctionLocator{
				{StepName: "^step$", FunctionLocator: &FunctionLocator{FullPackageName: "github.com/example/myapp/steps", FunctionName: "MyStep", IsExported: true}},
			},
		}

		err := validateExportedFunctions(output)
		require.NoError(t, err)
	})

	t.Run("allows unexported functions in same package", func(t *testing.T) {
		output := &Output{
			CurrentPackagePath: "github.com/example/myapp",
			ConfigFunctions: []*FunctionLocator{
				{FullPackageName: "github.com/example/myapp", FunctionName: "getConfig", IsExported: false},
			},
			HooksFunctions: []*FunctionLocator{
				{FullPackageName: "github.com/example/myapp", FunctionName: "getHooks", IsExported: false},
			},
			StepFunctions: []*StepFunctionLocator{
				{StepName: "^step$", FunctionLocator: &FunctionLocator{FullPackageName: "github.com/example/myapp", FunctionName: "myStep", IsExported: false}},
			},
		}

		err := validateExportedFunctions(output)
		require.NoError(t, err)
	})

	t.Run("returns error for unexported step function in different package", func(t *testing.T) {
		output := &Output{
			CurrentPackagePath: "github.com/example/myapp",
			StepFunctions: []*StepFunctionLocator{
				{StepName: "^step$", FunctionLocator: &FunctionLocator{FullPackageName: "github.com/example/myapp/steps", FunctionName: "myStep", IsExported: false}},
			},
		}

		err := validateExportedFunctions(output)
		require.Error(t, err)
		require.Contains(t, err.Error(), "step function")
		require.Contains(t, err.Error(), "myStep")
		require.Contains(t, err.Error(), "not exported")
	})

	t.Run("returns error for unexported config function in different package", func(t *testing.T) {
		output := &Output{
			CurrentPackagePath: "github.com/example/myapp",
			ConfigFunctions: []*FunctionLocator{
				{FullPackageName: "github.com/example/myapp/config", FunctionName: "getConfig", IsExported: false},
			},
		}

		err := validateExportedFunctions(output)
		require.Error(t, err)
		require.Contains(t, err.Error(), "config function")
		require.Contains(t, err.Error(), "getConfig")
		require.Contains(t, err.Error(), "not exported")
	})

	t.Run("returns error for unexported hooks function in different package", func(t *testing.T) {
		output := &Output{
			CurrentPackagePath: "github.com/example/myapp",
			HooksFunctions: []*FunctionLocator{
				{FullPackageName: "github.com/example/myapp/hooks", FunctionName: "getHooks", IsExported: false},
			},
		}

		err := validateExportedFunctions(output)
		require.Error(t, err)
		require.Contains(t, err.Error(), "hooks function")
		require.Contains(t, err.Error(), "getHooks")
		require.Contains(t, err.Error(), "not exported")
	})

	t.Run("reports multiple unexported functions in a single error", func(t *testing.T) {
		output := &Output{
			CurrentPackagePath: "github.com/example/myapp",
			StepFunctions: []*StepFunctionLocator{
				{StepName: "^step1$", FunctionLocator: &FunctionLocator{FullPackageName: "github.com/example/myapp/steps", FunctionName: "stepOne", IsExported: false}},
				{StepName: "^step2$", FunctionLocator: &FunctionLocator{FullPackageName: "github.com/example/myapp/steps", FunctionName: "stepTwo", IsExported: false}},
			},
		}

		err := validateExportedFunctions(output)
		require.Error(t, err)
		require.Contains(t, err.Error(), "stepOne")
		require.Contains(t, err.Error(), "stepTwo")
	})

	t.Run("passes with no functions", func(t *testing.T) {
		output := &Output{
			CurrentPackagePath: "github.com/example/myapp",
		}

		err := validateExportedFunctions(output)
		require.NoError(t, err)
	})

	t.Run("allows mix of exported cross-package and unexported same-package", func(t *testing.T) {
		output := &Output{
			CurrentPackagePath: "github.com/example/myapp",
			StepFunctions: []*StepFunctionLocator{
				{StepName: "^local$", FunctionLocator: &FunctionLocator{FullPackageName: "github.com/example/myapp", FunctionName: "localStep", IsExported: false}},
				{StepName: "^remote$", FunctionLocator: &FunctionLocator{FullPackageName: "github.com/example/other", FunctionName: "RemoteStep", IsExported: true}},
			},
		}

		err := validateExportedFunctions(output)
		require.NoError(t, err)
	})
}
