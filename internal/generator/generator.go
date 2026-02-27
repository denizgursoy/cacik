package generator

import (
	"context"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"golang.org/x/mod/modfile"
)

const (
	Separator             = ","
	defaultFileNamePrefix = "cacik"
)

func StartGenerator(ctx context.Context, codeParser GoCodeParser) error {
	funcSources := make([]string, 0)

	codeFlag := flag.String("code", "", "directories to search for functions seperated by comma")
	outputFlag := flag.String("output", "", "output file prefix (e.g. 'billing' produces billing_test.go with TestBilling)")
	flag.Parse()

	if len(strings.TrimSpace(*codeFlag)) == 0 {
		directory, err := os.Getwd()
		if err != nil {
			log.Println(err.Error())
			return err
		}
		funcSources = append(funcSources, directory)
	} else {
		funcSources = append(funcSources, strings.Split(*codeFlag, Separator)...)
	}

	for _, source := range funcSources {
		recursively, err := codeParser.ParseFunctionCommentsOfGoFilesInDirectoryRecursively(ctx, source)
		if err != nil {
			log.Println(err.Error())
			return err
		}

		// Resolve file name prefix: default or CLI flag
		fileNamePrefix := defaultFileNamePrefix
		if *outputFlag != "" {
			fileNamePrefix = *outputFlag
		}

		outputFile := outputFileFromPrefix(fileNamePrefix)
		recursively.TestFuncName = testFuncNameFromPrefix(fileNamePrefix)

		// Detect package name and full import path for the CWD
		pkgName, pkgPath, detectErr := detectPackage(outputFile)
		if detectErr != nil {
			log.Printf("warning: could not detect package: %v", detectErr)
		}
		if pkgName == "" {
			pkgName = fileNamePrefix + "_test"
		}
		recursively.PackageName = pkgName
		if pkgPath != "" {
			recursively.CurrentPackagePath = pkgPath
		}

		// Validate that all cross-package functions are exported
		if err := validateExportedFunctions(recursively); err != nil {
			return err
		}

		create, err := os.Create(outputFile)
		if err != nil {
			return err
		}
		err = recursively.Generate(create)

		if err != nil {
			log.Println(err.Error())
			return err
		}
	}

	return nil
}

// outputFileFromPrefix returns the generated test file name for a given prefix.
// For example, "billing" -> "billing_test.go".
func outputFileFromPrefix(prefix string) string {
	if prefix == "" {
		prefix = defaultFileNamePrefix
	}
	return prefix + "_test.go"
}

// testFuncNameFromPrefix returns the test function name for a given prefix.
// Segments separated by underscores are title-cased and joined.
// For example, "billing" -> "TestBilling", "my_feature" -> "TestMyFeature".
func testFuncNameFromPrefix(prefix string) string {
	if prefix == "" {
		prefix = defaultFileNamePrefix
	}
	segments := strings.Split(prefix, "_")
	var b strings.Builder
	b.WriteString("Test")
	for _, seg := range segments {
		if seg == "" {
			continue
		}
		runes := []rune(seg)
		runes[0] = unicode.ToUpper(runes[0])
		b.WriteString(string(runes))
	}
	return b.String()
}

// detectPackage detects the Go package name from Go files in the current
// directory and the full import path by combining the module path from go.mod
// with the relative directory. The outputFile parameter is the name of the
// generated test file so it can be skipped during package detection.
func detectPackage(outputFile string) (pkgName string, pkgPath string, err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("cannot get working directory: %w", err)
	}

	// 1. Detect package name from Go files in CWD
	pkgName, err = detectPackageName(cwd, outputFile)
	if err != nil {
		return "", "", err
	}

	// 2. Detect full import path from go.mod
	pkgPath, err = detectImportPath(cwd)
	if err != nil {
		return pkgName, "", err
	}

	return pkgName, pkgPath, nil
}

// detectPackageName detects the Go package name for the given directory by
// reading the package clause from existing Go files. If no Go files provide
// a package name, it returns "" so the caller can apply its own default.
// The outputFile parameter is the name of the generated test file so it can
// be skipped.
func detectPackageName(dir string, outputFile string) (string, error) {
	fset := token.NewFileSet()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("cannot read directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") {
			continue
		}
		// Skip the files we generate
		if name == outputFile {
			continue
		}

		filePath := filepath.Join(dir, name)
		f, parseErr := parser.ParseFile(fset, filePath, nil, parser.PackageClauseOnly)
		if parseErr != nil {
			continue
		}
		if f.Name != nil && f.Name.Name != "" {
			return f.Name.Name, nil
		}
	}

	return "", nil
}

// validateExportedFunctions checks that all discovered step, config, and hooks
// functions that live in a different package from the generated test file are
// exported (start with an uppercase letter). Unexported functions from other
// packages would cause a compilation error in the generated code.
// Functions in the same package as currentPackagePath are allowed to be unexported.
func validateExportedFunctions(output *Output) error {
	var items []string

	checkLocator := func(fl *FunctionLocator, kind string) {
		if fl.IsExported {
			return
		}
		if output.isSamePackage(fl.FullPackageName) {
			return
		}
		items = append(items, fmt.Sprintf(
			"  - %s function %q in package %q",
			kind, fl.FunctionName, fl.FullPackageName,
		))
	}

	for _, cf := range output.ConfigFunctions {
		checkLocator(cf, "config")
	}
	for _, hf := range output.HooksFunctions {
		checkLocator(hf, "hooks")
	}
	for _, sf := range output.StepFunctions {
		checkLocator(sf.FunctionLocator, "step")
	}

	if len(items) > 0 {
		return fmt.Errorf("the following functions are not exported (functions must start with an uppercase letter to be accessible from the generated test file):\n%s",
			strings.Join(items, "\n"))
	}
	return nil
}

// detectImportPath walks up from dir looking for go.mod, then computes the
// full import path as module_path + relative_directory.
func detectImportPath(dir string) (string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	// Walk up looking for go.mod
	current := absDir
	for {
		goModPath := filepath.Join(current, "go.mod")
		data, readErr := os.ReadFile(goModPath)
		if readErr == nil {
			modFile, parseErr := modfile.Parse(goModPath, data, nil)
			if parseErr != nil {
				return "", fmt.Errorf("cannot parse go.mod: %w", parseErr)
			}

			modulePath := modFile.Module.Mod.Path
			rel, relErr := filepath.Rel(current, absDir)
			if relErr != nil {
				return "", relErr
			}

			if rel == "." {
				return modulePath, nil
			}
			return modulePath + "/" + filepath.ToSlash(rel), nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("go.mod not found in any parent of %s", dir)
		}
		current = parent
	}
}
