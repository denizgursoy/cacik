package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"github.com/denizgursoy/cacik/internal/executor"
)

func GetComments(path string) {
	fileSet := token.NewFileSet()
	dir, err := parser.ParseDir(fileSet, path, nil, parser.ParseComments)
	if err != nil {
		return
	}

	for packageName, packageData := range dir {
		for _, val := range packageData.Files {
			for _, dec := range val.Decls {
				decl, ok := dec.(*ast.FuncDecl)
				if ok {
					if IsConfigFunction(decl, val.Imports) {
						println("found config function")
						executor.ExcuteFunction(decl)
					}
				}
			}
		}
		fmt.Println(packageName, packageData)
	}
}

func IsConfigFunction(fnDecl *ast.FuncDecl, imports []*ast.ImportSpec) bool {
	returnedTypes := fnDecl.Type.Results.List
	if len(returnedTypes) != 1 {
		return false
	}
	fmt.Printf("Analyzing function %s:\n", fnDecl.Name.Name)
	e := returnedTypes[0].Type
	path := analyzeExpr(e, imports)
	return strings.HasSuffix(path, "Config")
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
