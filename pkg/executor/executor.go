package executor

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strconv"

	messages "github.com/cucumber/messages/go/v21"
)

// StepDefinition holds a compiled regex pattern and its associated function
type StepDefinition struct {
	Pattern  *regexp.Regexp
	Function any
}

// StepExecutor handles matching and executing step definitions
type StepExecutor struct {
	steps      []StepDefinition
	patternSet map[string]bool // Track registered patterns for duplicate detection
	context    context.Context
}

// NewStepExecutor creates a new StepExecutor
func NewStepExecutor() *StepExecutor {
	return &StepExecutor{
		steps:      make([]StepDefinition, 0),
		patternSet: make(map[string]bool),
		context:    context.Background(),
	}
}

// RegisterStep registers a step definition with its regex pattern and function
func (e *StepExecutor) RegisterStep(pattern string, fn any) error {
	// Check for duplicate pattern
	if e.patternSet[pattern] {
		return fmt.Errorf("duplicate step pattern: %s", pattern)
	}

	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid step pattern %q: %w", pattern, err)
	}

	// Validate function signature
	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func {
		return fmt.Errorf("step handler must be a function, got %T", fn)
	}

	e.steps = append(e.steps, StepDefinition{
		Pattern:  compiled,
		Function: fn,
	})
	e.patternSet[pattern] = true
	return nil
}

// Execute runs all scenarios in the gherkin document
func (e *StepExecutor) Execute(document *messages.GherkinDocument) error {
	if document == nil || document.Feature == nil {
		return nil
	}

	var featureBackground *messages.Background

	for _, child := range document.Feature.Children {
		if child.Background != nil {
			featureBackground = child.Background
		} else if child.Rule != nil {
			if err := e.executeRule(child.Rule, featureBackground); err != nil {
				return err
			}
		} else if child.Scenario != nil {
			if err := e.executeScenarioWithBackground(child.Scenario, featureBackground); err != nil {
				return err
			}
		}
	}

	return nil
}

func (e *StepExecutor) executeRule(rule *messages.Rule, featureBackground *messages.Background) error {
	var ruleBackground *messages.Background

	for _, child := range rule.Children {
		if child.Background != nil {
			ruleBackground = child.Background
		} else if child.Scenario != nil {
			if err := e.executeBackground(featureBackground); err != nil {
				return err
			}
			if err := e.executeScenarioWithBackground(child.Scenario, ruleBackground); err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *StepExecutor) executeScenarioWithBackground(scenario *messages.Scenario, background *messages.Background) error {
	if err := e.executeBackground(background); err != nil {
		return err
	}

	for _, step := range scenario.Steps {
		if err := e.executeStep(step.Text); err != nil {
			return fmt.Errorf("step %q failed: %w", step.Text, err)
		}
	}
	return nil
}

func (e *StepExecutor) executeBackground(background *messages.Background) error {
	if background != nil {
		for _, step := range background.Steps {
			if err := e.executeStep(step.Text); err != nil {
				return fmt.Errorf("background step %q failed: %w", step.Text, err)
			}
		}
	}
	return nil
}

// executeStep finds and executes a matching step definition
func (e *StepExecutor) executeStep(stepText string) error {
	for _, stepDef := range e.steps {
		matches := stepDef.Pattern.FindStringSubmatch(stepText)
		if matches == nil {
			continue
		}

		// Extract capture groups (skip the full match at index 0)
		capturedArgs := matches[1:]

		// Invoke the step function with extracted arguments
		newCtx, err := e.invokeStepFunction(stepDef.Function, capturedArgs)
		if err != nil {
			return err
		}

		// Update context if returned
		if newCtx != nil {
			e.context = newCtx
		}
		return nil
	}

	return fmt.Errorf("no matching step definition found for: %s", stepText)
}

// invokeStepFunction calls the step function with proper argument conversion
func (e *StepExecutor) invokeStepFunction(fn any, args []string) (context.Context, error) {
	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()

	// Build argument list
	callArgs, err := e.buildCallArgs(fnType, args)
	if err != nil {
		return nil, err
	}

	// Call the function
	results := fnValue.Call(callArgs)

	// Process return values
	return e.processReturnValues(fnType, results)
}

// buildCallArgs constructs the argument slice for function invocation
func (e *StepExecutor) buildCallArgs(fnType reflect.Type, capturedArgs []string) ([]reflect.Value, error) {
	numParams := fnType.NumIn()
	callArgs := make([]reflect.Value, 0, numParams)

	capturedIndex := 0

	for i := 0; i < numParams; i++ {
		paramType := fnType.In(i)

		// Check if this parameter is context.Context
		if paramType.Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
			callArgs = append(callArgs, reflect.ValueOf(e.context))
			continue
		}

		// Otherwise, consume from captured arguments
		if capturedIndex >= len(capturedArgs) {
			return nil, fmt.Errorf("not enough captured arguments: expected %d more, have %d", numParams-i, len(capturedArgs)-capturedIndex)
		}

		arg := capturedArgs[capturedIndex]
		capturedIndex++

		converted, err := convertArg(arg, paramType)
		if err != nil {
			return nil, fmt.Errorf("failed to convert argument %q to %s: %w", arg, paramType, err)
		}
		callArgs = append(callArgs, converted)
	}

	return callArgs, nil
}

// processReturnValues extracts context and error from function return values
func (e *StepExecutor) processReturnValues(fnType reflect.Type, results []reflect.Value) (context.Context, error) {
	var newCtx context.Context
	var retErr error

	for i := 0; i < len(results); i++ {
		result := results[i]
		resultType := fnType.Out(i)

		// Check for context.Context
		if resultType.Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
			if !result.IsNil() {
				newCtx = result.Interface().(context.Context)
			}
			continue
		}

		// Check for error
		if resultType.Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			if !result.IsNil() {
				retErr = result.Interface().(error)
			}
			continue
		}
	}

	return newCtx, retErr
}

// convertArg converts a string argument to the target type
func convertArg(arg string, targetType reflect.Type) (reflect.Value, error) {
	switch targetType.Kind() {
	case reflect.String:
		return reflect.ValueOf(arg), nil

	case reflect.Int:
		v, err := strconv.Atoi(arg)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(v), nil

	case reflect.Int8:
		v, err := strconv.ParseInt(arg, 10, 8)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(int8(v)), nil

	case reflect.Int16:
		v, err := strconv.ParseInt(arg, 10, 16)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(int16(v)), nil

	case reflect.Int32:
		v, err := strconv.ParseInt(arg, 10, 32)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(int32(v)), nil

	case reflect.Int64:
		v, err := strconv.ParseInt(arg, 10, 64)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(v), nil

	case reflect.Uint:
		v, err := strconv.ParseUint(arg, 10, 0)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(uint(v)), nil

	case reflect.Uint8:
		v, err := strconv.ParseUint(arg, 10, 8)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(uint8(v)), nil

	case reflect.Uint16:
		v, err := strconv.ParseUint(arg, 10, 16)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(uint16(v)), nil

	case reflect.Uint32:
		v, err := strconv.ParseUint(arg, 10, 32)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(uint32(v)), nil

	case reflect.Uint64:
		v, err := strconv.ParseUint(arg, 10, 64)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(v), nil

	case reflect.Float32:
		v, err := strconv.ParseFloat(arg, 32)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(float32(v)), nil

	case reflect.Float64:
		v, err := strconv.ParseFloat(arg, 64)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(v), nil

	case reflect.Bool:
		v, err := strconv.ParseBool(arg)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(v), nil

	default:
		return reflect.Value{}, fmt.Errorf("unsupported parameter type: %s", targetType.Kind())
	}
}

// GetContext returns the current execution context
func (e *StepExecutor) GetContext() context.Context {
	return e.context
}

// SetContext sets the execution context
func (e *StepExecutor) SetContext(ctx context.Context) {
	e.context = ctx
}
