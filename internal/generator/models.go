package generator

import (
	"io"
	"sort"
	"strings"

	"github.com/dave/jennifer/jen"
)

type (
	FunctionLocator struct {
		FullPackageName string
		FunctionName    string
	}

	StepFunctionLocator struct {
		StepName string
		*FunctionLocator
	}

	// CustomType represents a user-defined type like `type Color string`
	// with its associated constant values
	CustomType struct {
		Name        string            // Type name, e.g., "Color"
		PackagePath string            // Full package path
		Underlying  string            // Underlying primitive type: "string", "int", "float64", etc.
		Values      map[string]string // Constant name -> value, e.g., {"Red": "red", "Blue": "blue"}
	}

	Output struct {
		ConfigFunctions    []*FunctionLocator // Functions returning *cacik.Config
		HooksFunctions     []*FunctionLocator // Functions returning *cacik.Hooks
		StepFunctions      []*StepFunctionLocator
		CustomTypes        map[string]*CustomType // lowercase type name -> CustomType
		CurrentPackagePath string                 // Full import path of the package where the test file is generated
		PackageName        string                 // Short package name (e.g., "myapp"); if empty, defaults to "main"
	}
)

// ValuesList returns a sorted list of all constant values for this custom type
func (ct *CustomType) ValuesList() []string {
	values := make([]string, 0, len(ct.Values))
	for _, v := range ct.Values {
		values = append(values, v)
	}
	sort.Strings(values)
	return values
}

// NamesAndValues returns a map of lowercase name/value -> actual value
// This is used for case-insensitive matching at runtime
func (ct *CustomType) NamesAndValues() map[string]string {
	result := make(map[string]string)
	for name, value := range ct.Values {
		// Add lowercase constant name -> value
		result[strings.ToLower(name)] = value
		// Add lowercase value -> value (for direct value matching)
		result[strings.ToLower(value)] = value
	}
	return result
}

// RegexPattern returns a regex pattern that matches any of the constant values or names
// The pattern uses (?i:...) for case-insensitive matching
func (ct *CustomType) RegexPattern() string {
	seen := make(map[string]bool)
	var parts []string

	// Add constant names and values (deduplicated, lowercase for matching)
	for name, value := range ct.Values {
		nameLower := strings.ToLower(name)
		valueLower := strings.ToLower(value)

		if !seen[nameLower] {
			parts = append(parts, regexEscape(nameLower))
			seen[nameLower] = true
		}
		if !seen[valueLower] {
			parts = append(parts, regexEscape(valueLower))
			seen[valueLower] = true
		}
	}

	sort.Strings(parts)
	// Use (?i:...) for case-insensitive matching
	return "(?i:" + strings.Join(parts, "|") + ")"
}

// regexEscape escapes special regex characters in a string
func regexEscape(s string) string {
	special := []string{"\\", ".", "+", "*", "?", "(", ")", "[", "]", "{", "}", "^", "$", "|"}
	result := s
	for _, char := range special {
		result = strings.ReplaceAll(result, char, "\\"+char)
	}
	return result
}

// isSamePackage returns true when the function is in the same package as the
// generated test file and therefore should be called without an import qualifier.
func (o *Output) isSamePackage(fullPkg string) bool {
	return o.CurrentPackagePath != "" && fullPkg == o.CurrentPackagePath
}

// qualOrLocal returns a jen.Statement that either qualifies the function call with
// its package path (for external packages) or calls it directly (for same-package).
func (o *Output) qualOrLocal(fullPkg, funcName string) *jen.Statement {
	if o.isSamePackage(fullPkg) {
		return jen.Id(funcName)
	}
	return jen.Qual(fullPkg, funcName)
}

func (o *Output) Generate(writer io.Writer) error {
	pkgName := o.PackageName
	if pkgName == "" {
		pkgName = "main"
	}
	mainFile := jen.NewFile(pkgName)

	var statements []jen.Code

	// Collect configs: config := cacik.MergeConfigs(...)
	if len(o.ConfigFunctions) > 0 {
		configCalls := make([]jen.Code, 0, len(o.ConfigFunctions))
		for _, cf := range o.ConfigFunctions {
			configCalls = append(configCalls, o.qualOrLocal(cf.FullPackageName, cf.FunctionName).Call())
		}
		statements = append(statements,
			jen.Id("config").Op(":=").Qual("github.com/denizgursoy/cacik/pkg/cacik", "MergeConfigs").Call(configCalls...),
		)
	}

	// Collect hooks: hooks := []*cacik.Hooks{...}
	if len(o.HooksFunctions) > 0 {
		hooksCalls := make([]jen.Code, 0, len(o.HooksFunctions))
		for _, hf := range o.HooksFunctions {
			hooksCalls = append(hooksCalls, o.qualOrLocal(hf.FullPackageName, hf.FunctionName).Call())
		}
		statements = append(statements,
			jen.Id("hooks").Op(":=").Index().Op("*").Qual("github.com/denizgursoy/cacik/pkg/cacik", "Hooks").Values(hooksCalls...),
		)
	}

	// Build runner chain — always pass t to NewCucumberRunner
	runnerChain := jen.Id("err").Op(":=").Qual("github.com/denizgursoy/cacik/pkg/runner", "NewCucumberRunner").Call(jen.Id("t")).Id(".").Line()

	// Add WithConfig if we have configs
	if len(o.ConfigFunctions) > 0 {
		runnerChain.Id("WithConfig").Call(jen.Id("config")).Id(".").Line()
	}

	// Add WithHooks if we have hooks
	if len(o.HooksFunctions) > 0 {
		runnerChain.Id("WithHooks").Call(jen.Id("hooks").Op("...")).Id(".").Line()
	}

	// Register custom types before steps
	for _, ct := range o.CustomTypes {
		// Build the values map literal
		valuesMap := jen.Map(jen.String()).String().Values(jen.DictFunc(func(d jen.Dict) {
			for k, v := range ct.NamesAndValues() {
				d[jen.Lit(k)] = jen.Lit(v)
			}
		}))

		runnerChain.Id("RegisterCustomType").Call(
			jen.Lit(ct.Name),
			jen.Lit(ct.Underlying),
			valuesMap,
		).Id(".").Line()
	}

	// Register steps
	for _, function := range o.StepFunctions {
		runnerChain.Id("RegisterStep").Call(jen.Lit(function.StepName), o.qualOrLocal(function.FullPackageName, function.FunctionName)).Id(".").Line()
	}

	runnerChain.Id("Run").Call()

	statements = append(statements, runnerChain)

	// Error handling: always use t.Fatal(err)
	statements = append(statements,
		jen.If(jen.Id("err").Op("!=").Nil()).Block(
			jen.Id("t").Dot("Fatal").Call(jen.Id("err")),
		),
	)

	// Always generate func TestCacik(t *testing.T) { ... }
	mainFile.Func().Id("TestCacik").Params(
		jen.Id("t").Op("*").Qual("testing", "T"),
	).Block(statements...)

	_, err := writer.Write([]byte(mainFile.GoString()))

	return err
}
