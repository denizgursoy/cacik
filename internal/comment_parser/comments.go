package comment_parser

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/denizgursoy/cacik/internal/generator"
)

const (
	StepPrefix   = "@cacik"
	SpaceAndTick = " `"
)

// supportedPrimitives lists the primitive types that can be used as underlying types for custom types
var supportedPrimitives = map[string]bool{
	"string":  true,
	"int":     true,
	"int8":    true,
	"int16":   true,
	"int32":   true,
	"int64":   true,
	"uint":    true,
	"uint8":   true,
	"uint16":  true,
	"uint32":  true,
	"uint64":  true,
	"float32": true,
	"float64": true,
	"bool":    true,
}

type GoSourceFileParser struct {
}

func NewGoSourceFileParser() *GoSourceFileParser {
	return &GoSourceFileParser{}
}

func (g *GoSourceFileParser) ParseFunctionCommentsOfGoFilesInDirectoryRecursively(ctx context.Context, parentDirectory string) (
	*generator.Output, error) {
	directories := getAllSubDirectories(parentDirectory)
	directories = append(directories, parentDirectory)

	output := &generator.Output{
		ConfigFunction: nil,
		StepFunctions:  make([]*generator.StepFunctionLocator, 0),
		CustomTypes:    make(map[string]*generator.CustomType),
	}

	allPackages := make(map[string]*ast.Package)
	for _, dir := range directories {
		packagesInTheDirectory, err := parser.ParseDir(token.NewFileSet(), dir, nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		mergePackages(allPackages, packagesInTheDirectory)
	}

	// First pass: collect all custom types and their constants
	for _, packageData := range allPackages {
		for filePath, node := range packageData.Files {
			importPath, err := getImportPathOfFuncDecl(filePath)
			if err != nil {
				return nil, err
			}

			// Parse custom type declarations
			parseCustomTypes(node, importPath, output.CustomTypes)
		}
	}

	// Second pass: parse constants for the custom types we found
	for _, packageData := range allPackages {
		for _, node := range packageData.Files {
			parseConstants(node, output.CustomTypes)
		}
	}

	// Third pass: parse functions and transform step patterns
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
						output.ConfigFunction = &generator.FunctionLocator{
							FullPackageName: importPathOfFuncDecl,
							FunctionName:    decl.Name.Name,
						}
					} else if isStepFunction {
						// Transform {param} syntax to regex
						transformedStep, err := transformStepPattern(*step, output.CustomTypes)
						if err != nil {
							return nil, fmt.Errorf("error in function %s: %w", decl.Name.Name, err)
						}

						output.StepFunctions = append(output.StepFunctions, &generator.StepFunctionLocator{
							StepName: transformedStep,
							FunctionLocator: &generator.FunctionLocator{
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

// parseCustomTypes finds type declarations like `type Color string` in a file
func parseCustomTypes(file *ast.File, packagePath string, customTypes map[string]*generator.CustomType) {
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			// Check if the underlying type is a supported primitive
			ident, ok := typeSpec.Type.(*ast.Ident)
			if !ok {
				continue
			}

			if supportedPrimitives[ident.Name] {
				typeName := typeSpec.Name.Name
				customTypes[strings.ToLower(typeName)] = &generator.CustomType{
					Name:        typeName,
					PackagePath: packagePath,
					Underlying:  ident.Name,
					Values:      make(map[string]string),
				}
			}
		}
	}
}

// parseConstants finds constant declarations and associates them with custom types
func parseConstants(file *ast.File, customTypes map[string]*generator.CustomType) {
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}

		var currentType string  // Track the type for iota-style const blocks
		var iotaValue int64 = 0 // Track iota value
		var lastExpr ast.Expr   // Track last expression for implicit values

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			// Determine the type of this constant
			var typeName string
			if valueSpec.Type != nil {
				if ident, ok := valueSpec.Type.(*ast.Ident); ok {
					typeName = ident.Name
					currentType = typeName
				}
			} else {
				// Use the type from previous constant in the block
				typeName = currentType
			}

			// Skip if this isn't a custom type we're tracking
			ct, ok := customTypes[strings.ToLower(typeName)]
			if !ok {
				iotaValue++
				continue
			}

			// Process each name in the const declaration
			for i, name := range valueSpec.Names {
				constName := name.Name
				var constValue string

				// Determine the value
				var expr ast.Expr
				if i < len(valueSpec.Values) {
					expr = valueSpec.Values[i]
					lastExpr = expr
				} else {
					expr = lastExpr // Use previous expression (iota continuation)
				}

				if expr != nil {
					constValue = evaluateConstExpr(expr, iotaValue, ct.Underlying)
				} else {
					// Default to iota for int types
					if isIntType(ct.Underlying) {
						constValue = fmt.Sprintf("%d", iotaValue)
					}
				}

				if constValue != "" {
					ct.Values[constName] = constValue
				}
			}

			iotaValue++
		}
	}
}

// evaluateConstExpr evaluates a constant expression and returns its string value
func evaluateConstExpr(expr ast.Expr, iotaValue int64, underlying string) string {
	switch e := expr.(type) {
	case *ast.BasicLit:
		// String or number literal
		value := e.Value
		// Remove quotes from string literals
		if e.Kind == token.STRING {
			value = strings.Trim(value, `"'`+"`")
		}
		return value

	case *ast.Ident:
		// Could be iota or another identifier
		if e.Name == "iota" {
			return fmt.Sprintf("%d", iotaValue)
		}
		// Could be true/false for bool
		if e.Name == "true" || e.Name == "false" {
			return e.Name
		}
		return ""

	case *ast.BinaryExpr:
		// Handle expressions like iota + 1
		left := evaluateConstExpr(e.X, iotaValue, underlying)
		right := evaluateConstExpr(e.Y, iotaValue, underlying)

		if left != "" && right != "" && isIntType(underlying) {
			leftVal, err1 := strconv.ParseInt(left, 10, 64)
			rightVal, err2 := strconv.ParseInt(right, 10, 64)
			if err1 == nil && err2 == nil {
				var result int64
				switch e.Op {
				case token.ADD:
					result = leftVal + rightVal
				case token.SUB:
					result = leftVal - rightVal
				case token.MUL:
					result = leftVal * rightVal
				case token.QUO:
					if rightVal != 0 {
						result = leftVal / rightVal
					}
				}
				return fmt.Sprintf("%d", result)
			}
		}
		return ""

	case *ast.UnaryExpr:
		// Handle negative numbers
		if e.Op == token.SUB {
			val := evaluateConstExpr(e.X, iotaValue, underlying)
			if val != "" {
				return "-" + val
			}
		}
		return ""

	case *ast.ParenExpr:
		return evaluateConstExpr(e.X, iotaValue, underlying)

	default:
		return ""
	}
}

// isIntType returns true if the type is an integer type
func isIntType(typeName string) bool {
	switch typeName {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64":
		return true
	}
	return false
}

// builtInTypes maps built-in parameter type names to their regex patterns
var builtInTypes = map[string]string{
	"int":    `(-?\d+)`,       // Matches integers (positive/negative)
	"float":  `(-?\d*\.?\d+)`, // Matches floating point numbers
	"word":   `(\w+)`,         // Matches a single word (no whitespace)
	"string": `"([^"]*)"`,     // Matches double-quoted strings (captures content without quotes)
	"":       `(.*)`,          // Empty name matches anything
	"any":    `(.*)`,          // Explicit any matches anything

	// Timezone formats:
	// - Z (UTC)
	// - Offset: +05:30, -08:00, +0530, -0800
	// - IANA names: Europe/London, America/New_York, UTC
	"timezone": `(Z|UTC|[+-]\d{2}:?\d{2}|[A-Za-z_]+/[A-Za-z_]+)`,

	// Time formats: HH:MM, HH:MM:SS, HH:MM:SS.mmm, with optional AM/PM and optional timezone
	// Examples: 14:30, 2:30pm, 14:30:45, 09:15:30.123, 14:30+05:30, 2:30pm Europe/London
	"time": `(\d{1,2}:\d{2}(?::\d{2})?(?:\.\d{1,3})?(?:\s*[AaPp][Mm])?(?:\s*(?:Z|UTC|[+-]\d{2}:?\d{2}|[A-Za-z_]+/[A-Za-z_]+))?)`,

	// Date formats (EU default: DD/MM/YYYY):
	// ISO: 2024-01-15, 2024/01/15
	// EU: 15/01/2024, 15-01-2024, 15.01.2024
	// Written: Jan 15, 2024 / January 15, 2024 / 15 Jan 2024
	"date": `(\d{4}[-/]\d{2}[-/]\d{2}|\d{1,2}[-/\.]\d{1,2}[-/\.]\d{2,4}|(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\.?\s+\d{1,2},?\s+\d{4}|\d{1,2}\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\.?\s+\d{4})`,

	// DateTime formats: combines date and time with separator (space, T, or @), with optional timezone
	// Examples: 2024-01-15 14:30:00, 2024-01-15T14:30:00Z, 15/01/2024 2:30pm Europe/London
	"datetime": `(\d{4}[-/]\d{2}[-/]\d{2}[T\s]\d{1,2}:\d{2}(?::\d{2})?(?:\.\d{1,3})?(?:\s*[AaPp][Mm])?(?:\s*(?:Z|UTC|[+-]\d{2}:?\d{2}|[A-Za-z_]+/[A-Za-z_]+))?|\d{1,2}[-/\.]\d{1,2}[-/\.]\d{2,4}\s+\d{1,2}:\d{2}(?::\d{2})?(?:\s*[AaPp][Mm])?(?:\s*(?:Z|UTC|[+-]\d{2}:?\d{2}|[A-Za-z_]+/[A-Za-z_]+))?)`,

	// Email format: local@domain
	// Examples: user@example.com, user.name+tag@sub.domain.org
	"email": `([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`,

	// Duration format: Go time.Duration strings
	// Examples: 5s, 1h30m, 500ms, 2h45m30s, 1.5h, -30m
	"duration": `(-?(?:\d+\.?\d*(?:ns|us|µs|ms|s|m|h))+)`,

	// URL format: HTTP/HTTPS URLs
	// Examples: http://example.com, https://sub.domain.org/path?query=value#fragment
	"url": `(https?://[^\s]+)`,
}

// transformStepPattern replaces {typename} placeholders with regex patterns
func transformStepPattern(pattern string, customTypes map[string]*generator.CustomType) (string, error) {
	// Find all {word} patterns
	result := pattern
	start := 0

	for {
		openBrace := strings.Index(result[start:], "{")
		if openBrace == -1 {
			break
		}
		openBrace += start

		closeBrace := strings.Index(result[openBrace:], "}")
		if closeBrace == -1 {
			break
		}
		closeBrace += openBrace

		// Extract the type name between braces
		typeName := result[openBrace+1 : closeBrace]
		typeNameLower := strings.ToLower(typeName)

		var regexPattern string

		// First, check if it's a built-in type
		if builtIn, ok := builtInTypes[typeNameLower]; ok {
			regexPattern = builtIn
		} else {
			// Look up the custom type
			ct, ok := customTypes[typeNameLower]
			if !ok {
				return "", fmt.Errorf("unknown parameter type {%s} in step pattern (not a built-in type or custom type)", typeName)
			}

			if len(ct.Values) == 0 {
				return "", fmt.Errorf("custom type %s has no defined constants", ct.Name)
			}

			// Replace {typename} with regex pattern from custom type
			regexPattern = "(" + ct.RegexPattern() + ")"
		}

		result = result[:openBrace] + regexPattern + result[closeBrace+1:]

		// Move start past the replacement
		start = openBrace + len(regexPattern)
	}

	return result, nil
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
