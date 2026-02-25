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

	"golang.org/x/mod/modfile"
)

const (
	Separator = ","
)

func StartGenerator(ctx context.Context, codeParser GoCodeParser) error {
	funcSources := make([]string, 0)

	codeFlag := flag.String("code", "", "directories to search for functions seperated by comma")
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

		// Detect package name and full import path for the CWD
		pkgName, pkgPath, detectErr := detectPackage()
		if detectErr != nil {
			log.Printf("warning: could not detect package: %v", detectErr)
		}
		if pkgName != "" {
			recursively.PackageName = pkgName
		}
		if pkgPath != "" {
			recursively.CurrentPackagePath = pkgPath
		}

		create, err := os.Create("cacik_test.go")
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

// detectPackage detects the Go package name from Go files in the current
// directory and the full import path by combining the module path from go.mod
// with the relative directory.
func detectPackage() (pkgName string, pkgPath string, err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("cannot get working directory: %w", err)
	}

	// 1. Detect package name from Go files in CWD
	pkgName, err = detectPackageName(cwd)
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

// detectPackageName detects the Go package name for the given directory.
// It first tries to read the package clause from existing Go files.
// If no Go files exist, it falls back to deriving the name from the directory
// path (or the module path for the module root).
func detectPackageName(dir string) (string, error) {
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
		if name == "cacik_test.go" {
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

	// No Go files found — derive package name from directory or module path.
	return packageNameFromDir(dir)
}

// packageNameFromDir derives a valid Go package name from the directory path.
// At the module root it uses the last segment of the module path from go.mod.
// Otherwise it uses the directory name, sanitising characters that are invalid
// in Go identifiers (hyphens, dots, etc.).
func packageNameFromDir(dir string) (string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	// Try to use the module path when we're at the module root.
	goModPath := filepath.Join(absDir, "go.mod")
	if data, readErr := os.ReadFile(goModPath); readErr == nil {
		modFile, parseErr := modfile.Parse(goModPath, data, nil)
		if parseErr == nil && modFile.Module != nil {
			base := filepath.Base(modFile.Module.Mod.Path)
			if name := sanitizePackageName(base); name != "" {
				return name, nil
			}
		}
	}

	// Fall back to the directory name.
	base := filepath.Base(absDir)
	if name := sanitizePackageName(base); name != "" {
		return name, nil
	}

	return "", fmt.Errorf("cannot derive package name from directory %s", dir)
}

// sanitizePackageName turns a raw name (directory segment or module path
// segment) into a valid Go package name. Invalid characters such as hyphens
// and dots are replaced with underscores, and leading digits are prefixed
// with an underscore.
func sanitizePackageName(raw string) string {
	if raw == "" || raw == "." || raw == "/" {
		return ""
	}

	var b strings.Builder
	for i, r := range raw {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			// Go package names are conventionally lowercase.
			b.WriteRune(r - 'A' + 'a')
		case r == '-' || r == '.':
			if i == 0 {
				continue // drop leading separator
			}
			b.WriteRune('_')
		default:
			// Drop other characters.
		}
	}

	name := b.String()
	if name == "" {
		return ""
	}
	// A package name must not start with a digit.
	if name[0] >= '0' && name[0] <= '9' {
		name = "_" + name
	}
	return name
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
