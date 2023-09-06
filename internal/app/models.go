package app

import (
	"io"

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

	Output struct {
		ConfigFunction *FunctionLocator
		StepFunctions  []*StepFunctionLocator
	}
)

func (o *Output) Generate(writer io.Writer) error {
	mainFile := jen.NewFile("main")

	functionBody := jen.Id("err").Op(":=").Qual("github.com/denizgursoy/cacik/pkg/runner", "NewCucumberRunner").Call().Id(".").Line()

	if o.ConfigFunction != nil {
		functionBody.Id("WithConfigFunc").Call(jen.Qual(o.ConfigFunction.FullPackageName, o.ConfigFunction.FunctionName)).Id(".").Line()
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
