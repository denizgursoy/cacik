package app

type (
	FunctionLocator struct {
		PackageName string
		Name        string
		Import      string
	}

	StepFunctionLocator struct {
		StepName string
		FunctionLocator
	}

	Output struct {
		ConfigFunction *FunctionLocator
		StepFunctions  []*StepFunctionLocator
	}
)
