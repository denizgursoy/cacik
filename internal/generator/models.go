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
		ConfigFunction *FunctionLocator
		StepFunctions  []*StepFunctionLocator
		CustomTypes    map[string]*CustomType // lowercase type name -> CustomType
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
	return strings.Join(parts, "|")
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

func (o *Output) Generate(writer io.Writer) error {
	mainFile := jen.NewFile("main")

	functionBody := jen.Id("err").Op(":=").Qual("github.com/denizgursoy/cacik/pkg/runner", "NewCucumberRunner").Call().Id(".").Line()

	if o.ConfigFunction != nil {
		functionBody.Id("WithConfigFunc").Call(jen.Qual(o.ConfigFunction.FullPackageName, o.ConfigFunction.FunctionName)).Id(".").Line()
	}

	// Register custom types before steps
	for _, ct := range o.CustomTypes {
		// Build the values map literal
		valuesMap := jen.Map(jen.String()).String().Values(jen.DictFunc(func(d jen.Dict) {
			for k, v := range ct.NamesAndValues() {
				d[jen.Lit(k)] = jen.Lit(v)
			}
		}))

		functionBody.Id("RegisterCustomType").Call(
			jen.Lit(ct.Name),
			jen.Lit(ct.Underlying),
			valuesMap,
		).Id(".").Line()
	}

	for _, function := range o.StepFunctions {
		functionBody.Id("RegisterStep").Call(jen.Lit(function.StepName), jen.Qual(function.FullPackageName, function.FunctionName)).Id(".").Line()
	}
	functionBody.Id("RunWithTags").Call().Line().Line()
	functionBody.If(jen.Id("err").Op("!=").Nil()).Block(
		jen.Qual("log", "Fatal").Call(jen.Id("err")),
	)

	mainFile.Func().Id("main").Params().Block(functionBody)

	_, err := writer.Write([]byte(mainFile.GoString()))

	return err
}
