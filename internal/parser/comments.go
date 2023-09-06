package parser

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/denizgursoy/cacik/internal/app"
)

const (
	StepPrefix   = "@cacik"
	SpaceAndTick = " `"
)

type GoSourceFileParser struct {
}

func NewGoSourceFileParser() *GoSourceFileParser {
	return &GoSourceFileParser{}
}

func (g *GoSourceFileParser) ParseFunctionCommentsOfGoFilesInDirectoryRecursively(ctx context.Context, parentDirectory string) (
	*app.Output, error) {
	directories := getAllSubDirectories(parentDirectory)
	directories = append(directories, parentDirectory)

	output := &app.Output{
		ConfigFunction: nil,
		StepFunctions:  make([]*app.StepFunctionLocator, 0),
	}

	allPackages := make(map[string]*ast.Package)
	for _, dir := range directories {
		packagesInTheDirectory, err := parser.ParseDir(token.NewFileSet(), dir, nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		mergePackages(allPackages, packagesInTheDirectory)
	}

	for _, packageData := range allPackages {
		for filePath, node := range packageData.Files {
			for _, dec := range node.Decls {
				decl, ok := dec.(*ast.FuncDecl)
				if ok {
					importPathOfFuncDecl, err := getImportPathOfFuncDecl(filePath)
					if err != nil {
						return nil, err
					}

					step, isStepFunction := IsStepFunction(decl)
					if IsConfigFunction(decl, node.Imports) {
						output.ConfigFunction = &app.FunctionLocator{
							FullPackageName: importPathOfFuncDecl,
							FunctionName:    decl.Name.Name,
						}
					} else if isStepFunction {
						output.StepFunctions = append(output.StepFunctions, &app.StepFunctionLocator{
							StepName: *step,
							FunctionLocator: &app.FunctionLocator{
								FullPackageName: importPathOfFuncDecl,
								FunctionName:    decl.Name.Name,
							},
						})
					}
				}
			}
		}

	}

	return output, nil
}

func IsConfigFunction(fnDecl *ast.FuncDecl, imports []*ast.ImportSpec) bool {
	if fnDecl.Type.Results == nil {
		return false
	}
	returnedTypes := fnDecl.Type.Results.List
	if len(returnedTypes) != 1 {
		return false
	}
	fmt.Printf("Analyzing function %s:\n", fnDecl.Name.Name)
	e := returnedTypes[0].Type
	path := analyzeExpr(e, imports)
	return strings.HasSuffix(path, "Config")
}

func IsStepFunction(decl *ast.FuncDecl) (*string, bool) {
	with := GetCommentLineStartingWith(StepPrefix, decl)
	if with != nil {
		return with, true
	}
	return nil, false
}

func GetCommentLineStartingWith(keyword string, fnDecl *ast.FuncDecl) *string {
	if fnDecl.Doc != nil {
		for _, comment := range fnDecl.Doc.List {
			text := comment.Text
			prefix := fmt.Sprintf("// %s", keyword)
			if strings.HasPrefix(text, prefix) {
				// include empty space and `
				startIndex := len(prefix) + len(SpaceAndTick)
				if len(text)-startIndex > 2 {
					stepDefinition := text[startIndex : len(text)-1]
					return &stepDefinition
				}
			}
		}
	}
	return nil
}

func analyzeExpr(expr ast.Expr, imports []*ast.ImportSpec) string {
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.SelectorExpr:
		return fmt.Sprintf("%s.%s", analyzeExpr(expr.X, imports), expr.Sel.Name)
	case *ast.StarExpr:
		return "*" + analyzeExpr(expr.X, imports)
	case *ast.ParenExpr:
		return "(" + analyzeExpr(expr.X, imports) + ")"
	case *ast.ArrayType:
		return "[]" + analyzeExpr(expr.Elt, imports)
	case *ast.MapType:
		return "map[" + analyzeExpr(expr.Key, imports) + "]" + analyzeExpr(expr.Value, imports)
	case *ast.StructType:
		return "struct{}"
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.FuncType:
		return "func()"
	case *ast.ChanType:
		dir := "chan "
		if expr.Dir == ast.RECV {
			dir = "<-chan "
		} else if expr.Dir == ast.SEND {
			dir = "chan<- "
		}
		return dir + analyzeExpr(expr.Value, imports)
	case *ast.CompositeLit:
		return analyzeExpr(expr.Type, imports)
	default:
		return "unknown"
	}
}

func getImportPathOfFuncDecl(filename string) (string, error) {
	// RunWithTags "go list" command to get module path.
	cmd := exec.Command("go", "list")
	cmd.Dir = filepath.Dir(filename)
	modulePathBytes, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(modulePathBytes)), nil // FuncDecl not found in the file.
}

func getAllSubDirectories(dirPath string) []string {
	var subdirectories []string

	// Walk the directory.
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Println(err)
			return err
		}
		// Check if it's a directory (excluding the root directory).
		if info.IsDir() && path != dirPath {
			subdirectories = append(subdirectories, path)
		}
		return nil
	})

	if err != nil {
		fmt.Println(err)
	}

	return subdirectories
}

func mergePackages(m1 map[string]*ast.Package, m2 map[string]*ast.Package) {
	for k, v := range m2 {
		m1[k] = v
	}
}
