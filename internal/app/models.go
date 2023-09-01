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
	f := jen.NewFile("main")
	f.ImportNames(o.Imports)
	line := jen.Id("err").Op(":=").Qual("github.com/denizgursoy/cacik/pkg/runner", "NewCucumberRunner").Call().Id(".").Line()

	if output.ConfigFunction != nil {
		line.Id("SetConfig").Call(jen.Qual(output.ConfigFunction.PackageName, output.ConfigFunction.Name)).Id(".").Line()
	}

	for _, function := range output.StepFunctions {
		line.Id("RegisterStep").Call(jen.Id(function.StepName), jen.Qual(function.Import, function.Name)).Id(".").Line()
	}
	line.Id("Run").Call().Line().Line()
	line.If(jen.Id("err").Op("!=").Nil()).Block(
		jen.Qual("log", "Fatal").Call(jen.Id("err")),
	)

	f.Func().Id("main").Params().Block(line)

	_, err := writer.Write([]byte(f.GoString()))

	return err
}
