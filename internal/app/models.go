package app

import (
	"io"

	"github.com/dave/jennifer/jen"
)

type (
	FunctionLocator struct {
		PackageName string
		Name        string
		Import      string
	}

	StepFunctionLocator struct {
		StepName string
		*FunctionLocator
	}

	Output struct {
		ConfigFunction *FunctionLocator
		StepFunctions  []*StepFunctionLocator
		Imports        map[string]string
	}
)

func (o *Output) Generate(output Output, writer io.Writer) error {
	mainFile := jen.NewFile("main")
	mainFile.ImportNames(o.Imports)
	functionBody := jen.Id("err").Op(":=").Qual("github.com/denizgursoy/cacik/pkg/runner", "NewCucumberRunner").Call().Id(".").Line()

	if output.ConfigFunction != nil {
		functionBody.Id("SetConfig").Call(jen.Qual(output.ConfigFunction.PackageName, output.ConfigFunction.Name)).Id(".").Line()
	}

	for _, function := range output.StepFunctions {
		functionBody.Id("RegisterStep").Call(jen.Id(function.StepName), jen.Qual(function.Import, function.Name)).Id(".").Line()
	}
	functionBody.Id("Run").Call().Line().Line()
	functionBody.If(jen.Id("err").Op("!=").Nil()).Block(
		jen.Qual("log", "Fatal").Call(jen.Id("err")),
	)

	mainFile.Func().Id("main").Params().Block(functionBody)

	_, err := writer.Write([]byte(mainFile.GoString()))

	return err
}
