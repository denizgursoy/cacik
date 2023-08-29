package executor

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"reflect"

	"golang.org/x/tools/go/packages"
)

func ExcuteFunction(fnDec *ast.FuncDecl) {
	// Import path of the package containing the function
	packageImportPath := "github.com/denizgursoy/cacik/pkg/testdata" // Replace with the actual import path

	// Name of the function to call
	functionName := "Method1" // Replace with the actual function name

	// Load the package
	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedSyntax | packages.NeedFiles,
	}
	loadedPackages, err := packages.Load(cfg, packageImportPath)
	if err != nil {
		fmt.Println("Error loading package:", err)
		return
	}

	// Find the function by its name
	var targetFunc reflect.Value
	for _, pkg := range loadedPackages {
		for _, syntax := range pkg.Syntax {
			for _, decl := range syntax.Decls {
				if fd, ok := decl.(*ast.FuncDecl); ok && fd.Name.Name == functionName {
					// Get the type information for the function
					fset := token.NewFileSet()
					conf := types.Config{}
					info := &types.Info{
						Types:  make(map[ast.Expr]types.TypeAndValue),
						Defs:   make(map[*ast.Ident]types.Object),
						Uses:   make(map[*ast.Ident]types.Object),
						Scopes: make(map[ast.Node]*types.Scope),
					}
					_, err := conf.Check(packageImportPath, fset, []*ast.File{syntax}, info)
					if err != nil {
						fmt.Println("Error checking types:", err)
						return
					}

					// Get the function's object
					funcObj := info.Defs[fd.Name]
					if funcObj == nil {
						fmt.Println("Function object not found")
						return
					}

					// Convert the function object to a reflect.Value
					if funcObj, ok := funcObj.(*types.Func); ok {
						targetFunc = reflect.ValueOf(funcObj)
					} else {
						fmt.Println("Invalid function object type")
						return
					}

					break
				}
			}
		}
	}

	// Check if the function was found
	if !targetFunc.IsValid() {
		fmt.Println("Function not found:", functionName)
		return
	}

	// Call the function
	targetFunc.Call(nil)

	//// Import path of the package containing the function
	//packageImportPath := "github.com/denizgursoy/cacik/pkg/testdata" // Replace with the actual import path
	//
	//// Name of the function to call
	//functionName := "Method1" // Replace with the actual function name
	//
	//// Load the plugin dynamically
	//p, err := plugin.Open(packageImportPath + ".so")
	//if err != nil {
	//	fmt.Println("Error:", err)
	//	return
	//}
	//
	//// Lookup the symbol (function) you want to call
	//sym, err := p.Lookup(functionName)
	//if err != nil {
	//	fmt.Println("Error:", err)
	//	return
	//}
	//
	//// Assert the symbol to a function type
	//dynamicFunc, ok := sym.(func())
	//if !ok {
	//	fmt.Println("Error: Symbol is not a function")
	//	return
	//}
	//
	//// Call the dynamic function
	//dynamicFunc()
}
