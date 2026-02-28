package executor

import (
	"encoding/base64"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	messages "github.com/cucumber/messages/go/v21"
	"github.com/denizgursoy/cacik/pkg/cacik"
)

// Time/Date parsing layouts (EU format default: DD/MM/YYYY)
var (
	// Time layouts (without timezone, timezone handled separately)
	timeLayouts = []string{
		"15:04:05.000",
		"15:04:05",
		"15:04",
		"3:04:05.000pm",
		"3:04:05.000PM",
		"3:04:05pm",
		"3:04:05PM",
		"3:04:05 pm",
		"3:04:05 PM",
		"3:04pm",
		"3:04PM",
		"3:04 pm",
		"3:04 PM",
	}

	// Date layouts (EU format prioritized: DD/MM/YYYY)
	dateLayouts = []string{
		// EU formats (DD/MM/YYYY) - prioritized
		"02/01/2006",
		"02-01-2006",
		"02.01.2006",
		"2/1/2006",
		"2-1-2006",
		"2.1.2006",
		// ISO formats (YYYY-MM-DD)
		"2006-01-02",
		"2006/01/02",
		// Written formats
		"2 Jan 2006",
		"2 January 2006",
		"02 Jan 2006",
		"02 January 2006",
		"Jan 2, 2006",
		"January 2, 2006",
		"Jan 02, 2006",
		"January 02, 2006",
	}

	// Timezone offset pattern
	tzOffsetRegex = regexp.MustCompile(`^([+-])(\d{2}):?(\d{2})$`)
)

// StepDefinition holds a compiled regex pattern and its associated function
type StepDefinition struct {
	Pattern  *regexp.Regexp
	Function any
}

// ResolvedStep holds a Gherkin step paired with its pre-resolved matching step
// definition. Created by ResolveStep during the pre-run validation phase and
// consumed by ExecuteResolvedStep during execution, avoiding redundant pattern
// matching.
type ResolvedStep struct {
	Keyword   string              // Gherkin keyword (Given, When, Then, etc.)
	Text      string              // Step text
	DataTable *messages.DataTable // Optional DataTable
	StepDef   StepDefinition      // The matched step definition
	Args      []string            // Captured regex groups (full match excluded)
	MatchLocs []int               // Byte positions of capture groups for reporter highlighting

	// Execution outcome fields — populated by ExecuteResolvedStep or by the
	// runner for skipped steps.
	StartedAt time.Time     // When step execution started (zero if skipped)
	Duration  time.Duration // Step execution duration (zero if skipped)
	Status    string        // "passed", "failed", or "skipped"
	Error     string        // Error message (empty if passed/skipped)
}

// CustomTypeInfo holds runtime info for custom type validation
type CustomTypeInfo struct {
	Name          string            // Type name, e.g., "Color"
	Underlying    string            // Underlying primitive type: "string", "int", etc.
	AllowedValues map[string]string // lowercase name/value -> actual value
}

// AllowedValuesList returns a list of allowed values for error messages
func (c *CustomTypeInfo) AllowedValuesList() []string {
	seen := make(map[string]bool)
	var values []string
	for _, v := range c.AllowedValues {
		if !seen[v] {
			values = append(values, v)
			seen[v] = true
		}
	}
	return values
}

// StepExecutor handles matching and executing step definitions
type StepExecutor struct {
	steps        []StepDefinition
	patternSet   map[string]bool            // Track registered patterns for duplicate detection
	customTypes  map[string]*CustomTypeInfo // type name -> custom type info
	cacikCtx     *cacik.Context             // Cacik context for step functions
	hookExecutor *cacik.HookExecutor        // Hook executor for BeforeStep/AfterStep
}

// NewStepExecutor creates a new StepExecutor
func NewStepExecutor() *StepExecutor {
	return &StepExecutor{
		steps:       make([]StepDefinition, 0),
		patternSet:  make(map[string]bool),
		customTypes: make(map[string]*CustomTypeInfo),
		cacikCtx:    cacik.New(),
	}
}

// SetCacikContext sets the cacik context for step execution
func (e *StepExecutor) SetCacikContext(ctx *cacik.Context) {
	e.cacikCtx = ctx
}

// GetCacikContext returns the current cacik context
func (e *StepExecutor) GetCacikContext() *cacik.Context {
	return e.cacikCtx
}

// Clone creates a copy of the executor with fresh context but shared step definitions
func (e *StepExecutor) Clone() *StepExecutor {
	return &StepExecutor{
		steps:        e.steps,        // Share step definitions (read-only)
		patternSet:   e.patternSet,   // Share pattern set (read-only)
		customTypes:  e.customTypes,  // Share custom types (read-only)
		cacikCtx:     nil,            // Will be set per-scenario
		hookExecutor: e.hookExecutor, // Share hook executor
	}
}

// ResolveStep finds the first matching step definition for the given step text
// and returns a ResolvedStep with the match results pre-computed. Returns an
// error if no matching step definition is found.
func (e *StepExecutor) ResolveStep(keyword, stepText string, dataTable *messages.DataTable) (*ResolvedStep, error) {
	for _, stepDef := range e.steps {
		matches := stepDef.Pattern.FindStringSubmatch(stepText)
		if matches == nil {
			continue
		}
		var matchLocs []int
		if idxMatches := stepDef.Pattern.FindStringSubmatchIndex(stepText); len(idxMatches) > 2 {
			matchLocs = idxMatches[2:]
		}
		return &ResolvedStep{
			Keyword:   keyword,
			Text:      stepText,
			DataTable: dataTable,
			StepDef:   stepDef,
			Args:      matches[1:],
			MatchLocs: matchLocs,
		}, nil
	}
	return nil, fmt.Errorf("no matching step definition found for: %s", stepText)
}

// ExecuteResolvedStep executes a pre-resolved step without re-scanning patterns.
// The ResolvedStep must have been created by ResolveStep.
func (e *StepExecutor) ExecuteResolvedStep(rs *ResolvedStep) error {
	cacikStep := cacik.Step{Keyword: rs.Keyword, Text: rs.Text}

	// Execute BeforeStep hooks
	if e.hookExecutor != nil {
		e.hookExecutor.ExecuteBeforeStep(cacikStep)
	}

	// Record start time
	rs.StartedAt = time.Now()

	// Execute step with panic recovery and runtime.Goexit detection
	var stepErr error
	var panicMsg string

	var testingT cacik.T
	if e.cacikCtx != nil {
		testingT = e.cacikCtx.TestingT()
	}

	func() {
		defer func() {
			if r := recover(); r != nil {
				panicMsg = fmt.Sprintf("%v", r)
				stepErr = fmt.Errorf("%v", r)
			} else if testingT != nil && testingT.Failed() {
				panicMsg = "assertion failed"
				stepErr = fmt.Errorf("step assertion failed")
			}
		}()
		stepErr = e.invokeStepFunction(rs.StepDef.Function, rs.Args, rs.DataTable)
	}()

	// Record duration
	rs.Duration = time.Since(rs.StartedAt)

	// Set outcome on ResolvedStep
	if stepErr != nil {
		rs.Status = "failed"
		rs.Error = panicMsg
		if rs.Error == "" {
			rs.Error = stepErr.Error()
		}
	} else {
		rs.Status = "passed"
	}

	// Execute AfterStep hooks
	if e.hookExecutor != nil {
		e.hookExecutor.ExecuteAfterStep(cacikStep, stepErr)
	}

	// Report step result
	if e.cacikCtx != nil {
		reporter := e.cacikCtx.Reporter()
		if stepErr != nil {
			reporter.StepFailed(rs.Keyword, rs.Text, rs.Error, rs.MatchLocs)
			reporter.AddStepResult(false, false)
		} else {
			reporter.StepPassed(rs.Keyword, rs.Text, rs.MatchLocs)
			reporter.AddStepResult(true, false)
		}
		if rs.DataTable != nil {
			reporter.StepDataTable(dataTableToRows(rs.DataTable))
		}
	}

	return stepErr
}

// SetHookExecutor sets the hook executor for BeforeStep/AfterStep hooks
func (e *StepExecutor) SetHookExecutor(he *cacik.HookExecutor) {
	e.hookExecutor = he
}

// RegisterCustomType registers a custom type with its allowed values
func (e *StepExecutor) RegisterCustomType(name, underlying string, values map[string]string) {
	e.customTypes[name] = &CustomTypeInfo{
		Name:          name,
		Underlying:    underlying,
		AllowedValues: values,
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
		if err := e.ExecuteStepWithKeyword(step.Keyword, step.Text, step.DataTable); err != nil {
			return fmt.Errorf("step %q failed: %w", step.Text, err)
		}
	}
	return nil
}

func (e *StepExecutor) executeBackground(background *messages.Background) error {
	if background != nil {
		for _, step := range background.Steps {
			if err := e.ExecuteStepWithKeyword(step.Keyword, step.Text, step.DataTable); err != nil {
				return fmt.Errorf("background step %q failed: %w", step.Text, err)
			}
		}
	}
	return nil
}

// ExecuteStep finds and executes a matching step definition (exported for parallel execution)
// Deprecated: Use ExecuteStepWithKeyword instead
func (e *StepExecutor) ExecuteStep(stepText string) error {
	return e.ExecuteStepWithKeyword("", stepText, nil)
}

// ExecuteStepWithKeyword finds and executes a matching step definition with keyword for reporting.
// If the step has a DataTable, pass it as dataTable; otherwise pass nil.
func (e *StepExecutor) ExecuteStepWithKeyword(keyword, stepText string, dataTable *messages.DataTable) error {
	for _, stepDef := range e.steps {
		matches := stepDef.Pattern.FindStringSubmatch(stepText)
		if matches == nil {
			continue
		}

		// Extract capture groups (skip the full match at index 0)
		capturedArgs := matches[1:]

		// Compute capture group byte positions for reporter highlighting.
		// FindStringSubmatchIndex returns [full_start, full_end, grp1_start, grp1_end, ...].
		// We strip the first pair (full match) so matchLocs = [grp1_start, grp1_end, ...].
		var matchLocs []int
		if idxMatches := stepDef.Pattern.FindStringSubmatchIndex(stepText); len(idxMatches) > 2 {
			matchLocs = idxMatches[2:] // skip full-match pair
		}

		// Build Step for hook functions
		cacikStep := cacik.Step{Keyword: keyword, Text: stepText}

		// Execute BeforeStep hooks
		if e.hookExecutor != nil {
			e.hookExecutor.ExecuteBeforeStep(cacikStep)
		}

		// Execute step with panic recovery and runtime.Goexit detection
		var stepErr error
		var panicMsg string

		// Check if we have a *testing.T backing (for runtime.Goexit detection)
		var testingT cacik.T
		if e.cacikCtx != nil {
			testingT = e.cacikCtx.TestingT()
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					// Traditional panic-based failure (panicT or user panic)
					panicMsg = fmt.Sprintf("%v", r)
					stepErr = fmt.Errorf("%v", r)
				} else if testingT != nil && testingT.Failed() {
					// runtime.Goexit() path: t.FailNow() was called.
					// Deferred functions run but recover() returns nil.
					panicMsg = "assertion failed"
					stepErr = fmt.Errorf("step assertion failed")
				}
			}()
			stepErr = e.invokeStepFunction(stepDef.Function, capturedArgs, dataTable)
		}()

		// Execute AfterStep hooks
		if e.hookExecutor != nil {
			e.hookExecutor.ExecuteAfterStep(cacikStep, stepErr)
		}

		// Report step result
		if e.cacikCtx != nil {
			reporter := e.cacikCtx.Reporter()
			if stepErr != nil {
				errMsg := panicMsg
				if errMsg == "" && stepErr != nil {
					errMsg = stepErr.Error()
				}
				reporter.StepFailed(keyword, stepText, errMsg, matchLocs)
				reporter.AddStepResult(false, false)
			} else {
				reporter.StepPassed(keyword, stepText, matchLocs)
				reporter.AddStepResult(true, false)
			}
			if dataTable != nil {
				reporter.StepDataTable(dataTableToRows(dataTable))
			}
		}

		return stepErr
	}

	// No matching step found
	errMsg := fmt.Sprintf("no matching step definition found for: %s", stepText)
	if e.cacikCtx != nil {
		e.cacikCtx.Reporter().StepFailed(keyword, stepText, errMsg, nil)
		e.cacikCtx.Reporter().AddStepResult(false, false)
	}
	return fmt.Errorf("%s", errMsg)
}

// invokeStepFunction calls the step function with proper argument conversion
func (e *StepExecutor) invokeStepFunction(fn any, args []string, dataTable *messages.DataTable) error {
	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()

	// Build argument list
	callArgs, err := e.buildCallArgs(fnType, args, dataTable)
	if err != nil {
		return err
	}

	// Call the function
	results := fnValue.Call(callArgs)

	// Process return values (check for returned error)
	if err := e.processReturnValues(fnType, results); err != nil {
		return err
	}

	return nil
}

// cacikContextType is the reflect type for *cacik.Context
var cacikContextType = reflect.TypeOf((*cacik.Context)(nil))

// tableType is the reflect type for cacik.Table
var tableType = reflect.TypeOf(cacik.Table{})

// dataTableToRows converts a Gherkin DataTable to a [][]string for the reporter.
func dataTableToRows(dt *messages.DataTable) [][]string {
	rows := make([][]string, 0, len(dt.Rows))
	for _, row := range dt.Rows {
		cells := make([]string, 0, len(row.Cells))
		for _, cell := range row.Cells {
			cells = append(cells, cell.Value)
		}
		rows = append(rows, cells)
	}
	return rows
}

// buildCallArgs constructs the argument slice for function invocation
func (e *StepExecutor) buildCallArgs(fnType reflect.Type, capturedArgs []string, dataTable *messages.DataTable) ([]reflect.Value, error) {
	numParams := fnType.NumIn()
	callArgs := make([]reflect.Value, 0, numParams)

	capturedIndex := 0

	for i := 0; i < numParams; i++ {
		paramType := fnType.In(i)

		// Check if this parameter is *cacik.Context
		if paramType == cacikContextType {
			callArgs = append(callArgs, reflect.ValueOf(e.cacikCtx))
			continue
		}

		// Check if this parameter is cacik.Table
		if paramType == tableType {
			if dataTable == nil {
				return nil, fmt.Errorf("step function expects a cacik.Table parameter but the step has no DataTable")
			}
			table := cacik.NewTableFromDataTable(dataTable)
			callArgs = append(callArgs, reflect.ValueOf(table))
			continue
		}

		// Otherwise, consume from captured arguments
		if capturedIndex >= len(capturedArgs) {
			return nil, fmt.Errorf("not enough captured arguments: expected %d more, have %d", numParams-i, len(capturedArgs)-capturedIndex)
		}

		arg := capturedArgs[capturedIndex]
		capturedIndex++

		converted, err := e.convertArg(arg, paramType)
		if err != nil {
			return nil, fmt.Errorf("failed to convert argument %q to %s: %w", arg, paramType, err)
		}
		callArgs = append(callArgs, converted)
	}

	return callArgs, nil
}

// processReturnValues extracts error from function return values
func (e *StepExecutor) processReturnValues(fnType reflect.Type, results []reflect.Value) error {
	for i := 0; i < len(results); i++ {
		result := results[i]
		resultType := fnType.Out(i)

		// Check for error
		if resultType.Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			if !result.IsNil() {
				return result.Interface().(error)
			}
		}
	}

	return nil
}

// convertArg converts a string argument to the target type
func (e *StepExecutor) convertArg(arg string, targetType reflect.Type) (reflect.Value, error) {
	typeName := targetType.Name()
	kindName := targetType.Kind().String()

	// Check for time.Time
	if targetType == reflect.TypeOf(time.Time{}) {
		// Try parsing as datetime first, then date, then time
		if dt, err := parseDateTime(arg); err == nil {
			return reflect.ValueOf(dt), nil
		}
		if d, err := parseDate(arg); err == nil {
			return reflect.ValueOf(d), nil
		}
		if t, err := parseTime(arg); err == nil {
			return reflect.ValueOf(t), nil
		}
		return reflect.Value{}, fmt.Errorf("cannot parse %q as time.Time", arg)
	}

	// Check for *time.Location
	if targetType == reflect.TypeOf((*time.Location)(nil)) {
		loc, err := parseTimezone(arg)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(loc), nil
	}

	// Check for time.Duration
	if targetType == reflect.TypeOf(time.Duration(0)) {
		d, err := time.ParseDuration(arg)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("cannot parse %q as time.Duration: %w", arg, err)
		}
		return reflect.ValueOf(d), nil
	}

	// Check for *url.URL
	if targetType == reflect.TypeOf((*url.URL)(nil)) {
		u, err := url.Parse(arg)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("cannot parse %q as URL: %w", arg, err)
		}
		return reflect.ValueOf(u), nil
	}

	// Check for net.IP
	if targetType == reflect.TypeOf(net.IP{}) {
		ip := net.ParseIP(arg)
		if ip == nil {
			return reflect.Value{}, fmt.Errorf("cannot parse %q as net.IP", arg)
		}
		return reflect.ValueOf(ip), nil
	}

	// Check for []byte (base64-encoded)
	if targetType == reflect.TypeOf([]byte{}) {
		decoded, err := base64.StdEncoding.DecodeString(arg)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("cannot parse %q as base64 []byte: %w", arg, err)
		}
		return reflect.ValueOf(decoded), nil
	}

	// Check for []string (CSV)
	if targetType == reflect.TypeOf([]string{}) {
		parts := strings.Split(arg, ",")
		return reflect.ValueOf(parts), nil
	}

	// Check for *big.Int
	if targetType == reflect.TypeOf((*big.Int)(nil)) {
		bi := new(big.Int)
		if _, ok := bi.SetString(arg, 10); !ok {
			return reflect.Value{}, fmt.Errorf("cannot parse %q as *big.Int", arg)
		}
		return reflect.ValueOf(bi), nil
	}

	// Check for *regexp.Regexp
	if targetType == reflect.TypeOf((*regexp.Regexp)(nil)) {
		// Strip surrounding slashes if present: /pattern/ → pattern
		pattern := arg
		if len(pattern) >= 2 && pattern[0] == '/' && pattern[len(pattern)-1] == '/' {
			pattern = pattern[1 : len(pattern)-1]
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("cannot parse %q as *regexp.Regexp: %w", arg, err)
		}
		return reflect.ValueOf(re), nil
	}

	// Check if this is a custom type (named type that differs from its kind)
	if typeName != "" && typeName != kindName {
		return e.convertCustomType(arg, targetType, typeName)
	}

	// Handle primitive types
	return convertPrimitive(arg, targetType)
}

// convertCustomType handles conversion of custom types like `type Color string`
func (e *StepExecutor) convertCustomType(arg string, targetType reflect.Type, typeName string) (reflect.Value, error) {
	// Look up custom type info for validation
	info, hasInfo := e.customTypes[typeName]

	// Resolve the actual value (handles case-insensitive matching)
	actualValue := arg
	if hasInfo {
		resolved, ok := info.AllowedValues[strings.ToLower(arg)]
		if !ok {
			return reflect.Value{}, fmt.Errorf("invalid %s: %q (allowed: %v)",
				typeName, arg, info.AllowedValuesList())
		}
		actualValue = resolved
	}

	// Create a value of the custom type
	return convertToCustomType(actualValue, targetType)
}

// convertToCustomType creates a value of a custom type from a string
func convertToCustomType(arg string, targetType reflect.Type) (reflect.Value, error) {
	val := reflect.New(targetType).Elem()

	switch targetType.Kind() {
	case reflect.String:
		val.SetString(arg)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(arg, 10, 64)
		if err != nil {
			return reflect.Value{}, err
		}
		val.SetInt(i)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(arg, 10, 64)
		if err != nil {
			return reflect.Value{}, err
		}
		val.SetUint(u)

	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(arg, 64)
		if err != nil {
			return reflect.Value{}, err
		}
		val.SetFloat(f)

	case reflect.Bool:
		b, err := parseBool(arg)
		if err != nil {
			return reflect.Value{}, err
		}
		val.SetBool(b)

	default:
		return reflect.Value{}, fmt.Errorf("unsupported underlying type: %s", targetType.Kind())
	}

	return val, nil
}

// convertPrimitive converts a string to a primitive type
func convertPrimitive(arg string, targetType reflect.Type) (reflect.Value, error) {
	switch targetType.Kind() {
	case reflect.String:
		return reflect.ValueOf(arg), nil

	case reflect.Int:
		v, err := strconv.ParseInt(arg, 0, 0)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(int(v)), nil

	case reflect.Int8:
		v, err := strconv.ParseInt(arg, 0, 8)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(int8(v)), nil

	case reflect.Int16:
		v, err := strconv.ParseInt(arg, 0, 16)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(int16(v)), nil

	case reflect.Int32:
		v, err := strconv.ParseInt(arg, 0, 32)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(int32(v)), nil

	case reflect.Int64:
		v, err := strconv.ParseInt(arg, 0, 64)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(v), nil

	case reflect.Uint:
		v, err := strconv.ParseUint(arg, 0, 0)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(uint(v)), nil

	case reflect.Uint8:
		v, err := strconv.ParseUint(arg, 0, 8)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(uint8(v)), nil

	case reflect.Uint16:
		v, err := strconv.ParseUint(arg, 0, 16)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(uint16(v)), nil

	case reflect.Uint32:
		v, err := strconv.ParseUint(arg, 0, 32)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(uint32(v)), nil

	case reflect.Uint64:
		v, err := strconv.ParseUint(arg, 0, 64)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(v), nil

	case reflect.Float32:
		v, err := strconv.ParseFloat(arg, 32)
		if err != nil {
			// Try stripping trailing % for percent values
			if strings.HasSuffix(arg, "%") {
				pv, perr := strconv.ParseFloat(strings.TrimSuffix(arg, "%"), 32)
				if perr == nil {
					return reflect.ValueOf(float32(pv / 100)), nil
				}
			}
			return reflect.Value{}, err
		}
		return reflect.ValueOf(float32(v)), nil

	case reflect.Float64:
		v, err := strconv.ParseFloat(arg, 64)
		if err != nil {
			// Try stripping trailing % for percent values
			if strings.HasSuffix(arg, "%") {
				pv, perr := strconv.ParseFloat(strings.TrimSuffix(arg, "%"), 64)
				if perr == nil {
					return reflect.ValueOf(pv / 100), nil
				}
			}
			return reflect.Value{}, err
		}
		return reflect.ValueOf(v), nil

	case reflect.Bool:
		v, err := parseBool(arg)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(v), nil

	default:
		return reflect.Value{}, fmt.Errorf("unsupported parameter type: %s", targetType.Kind())
	}
}

// =============================================================================
// Boolean Parsing
// =============================================================================

// parseBool converts a string to a boolean value.
// Supports human-readable values (case-insensitive):
// Truthy: true, yes, on, enabled, 1, t
// Falsy: false, no, off, disabled, 0, f
func parseBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "true", "yes", "on", "enabled", "1", "t":
		return true, nil
	case "false", "no", "off", "disabled", "0", "f":
		return false, nil
	default:
		return false, fmt.Errorf("cannot parse %q as bool", s)
	}
}

// =============================================================================
// Time, Date, DateTime, and Timezone Parsing
// =============================================================================

// parseTimezone parses a timezone string and returns a *time.Location.
// Supports: Z, UTC, +05:30, -08:00, +0530, Europe/London, America/New_York
func parseTimezone(s string) (*time.Location, error) {
	s = strings.TrimSpace(s)

	// Handle Z and UTC
	if s == "Z" || s == "UTC" {
		return time.UTC, nil
	}

	// Handle offset format: +05:30, -08:00, +0530, -0800
	if matches := tzOffsetRegex.FindStringSubmatch(s); matches != nil {
		sign := 1
		if matches[1] == "-" {
			sign = -1
		}
		hours, _ := strconv.Atoi(matches[2])
		minutes, _ := strconv.Atoi(matches[3])
		offsetSeconds := sign * (hours*3600 + minutes*60)
		return time.FixedZone(s, offsetSeconds), nil
	}

	// Handle IANA timezone names: Europe/London, America/New_York
	loc, err := time.LoadLocation(s)
	if err != nil {
		return nil, fmt.Errorf("unknown timezone %q: %w", s, err)
	}
	return loc, nil
}

// extractTimezone extracts timezone suffix from a time/datetime string.
// Returns the string without timezone and the parsed location.
func extractTimezone(s string) (string, *time.Location) {
	s = strings.TrimSpace(s)

	// Check for Z suffix
	if strings.HasSuffix(s, "Z") {
		return strings.TrimSuffix(s, "Z"), time.UTC
	}

	// Check for UTC suffix
	if strings.HasSuffix(s, " UTC") || strings.HasSuffix(s, "UTC") {
		return strings.TrimSuffix(strings.TrimSuffix(s, " UTC"), "UTC"), time.UTC
	}

	// Check for IANA timezone (contains /)
	parts := strings.Split(s, " ")
	if len(parts) >= 2 {
		lastPart := parts[len(parts)-1]
		if strings.Contains(lastPart, "/") {
			loc, err := time.LoadLocation(lastPart)
			if err == nil {
				return strings.TrimSuffix(s, " "+lastPart), loc
			}
		}
	}

	// Check for offset at the end: +05:30, -08:00, +0530, -0800
	// Pattern: last part starts with + or -
	if len(parts) >= 1 {
		lastPart := parts[len(parts)-1]
		if len(lastPart) >= 5 && (lastPart[0] == '+' || lastPart[0] == '-') {
			loc, err := parseTimezone(lastPart)
			if err == nil {
				withoutTz := strings.TrimSuffix(s, lastPart)
				withoutTz = strings.TrimSuffix(withoutTz, " ")
				return withoutTz, loc
			}
		}
	}

	// Check for offset directly attached (no space): 14:30+05:30
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '+' || s[i] == '-' {
			possibleTz := s[i:]
			loc, err := parseTimezone(possibleTz)
			if err == nil {
				return s[:i], loc
			}
			break
		}
	}

	// No timezone found, use Local
	return s, time.Local
}

// parseTime parses a time string and returns time.Time with zero date (0001-01-01).
// Supports: 14:30, 2:30pm, 14:30:45.123, 14:30+05:30, 2:30pm Europe/London
func parseTime(s string) (time.Time, error) {
	// Extract timezone
	timeStr, loc := extractTimezone(s)
	timeStr = strings.TrimSpace(timeStr)

	// Try each time layout
	for _, layout := range timeLayouts {
		t, err := time.ParseInLocation(layout, timeStr, loc)
		if err == nil {
			// Return with zero date (0001-01-01)
			return time.Date(1, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc), nil
		}
	}

	return time.Time{}, fmt.Errorf("cannot parse %q as time", s)
}

// parseDate parses a date string and returns time.Time at midnight (00:00:00) in Local timezone.
// Supports EU format (DD/MM/YYYY) by default, plus ISO and written formats.
func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)

	// Try each date layout
	for _, layout := range dateLayouts {
		t, err := time.ParseInLocation(layout, s, time.Local)
		if err == nil {
			// Return at midnight
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local), nil
		}
	}

	return time.Time{}, fmt.Errorf("cannot parse %q as date", s)
}

// parseDateTime parses a datetime string and returns time.Time with optional timezone.
// Supports: 2024-01-15 14:30:00, 2024-01-15T14:30:00Z, 15/01/2024 2:30pm Europe/London
func parseDateTime(s string) (time.Time, error) {
	// Extract timezone
	dtStr, loc := extractTimezone(s)
	dtStr = strings.TrimSpace(dtStr)

	// Try to split by T or space to separate date and time
	var datePart, timePart string

	if idx := strings.Index(dtStr, "T"); idx != -1 {
		datePart = dtStr[:idx]
		timePart = dtStr[idx+1:]
	} else if idx := strings.LastIndex(dtStr, " "); idx != -1 {
		// Find the space that separates date and time
		// Date could be "15/01/2024" or "15 Jan 2024"
		// We need to find the space before time (which contains :)
		for i := len(dtStr) - 1; i >= 0; i-- {
			if dtStr[i] == ' ' {
				possibleTime := dtStr[i+1:]
				if strings.Contains(possibleTime, ":") {
					datePart = dtStr[:i]
					timePart = possibleTime
					break
				}
			}
		}
		if datePart == "" {
			// Fallback: last space
			datePart = dtStr[:idx]
			timePart = dtStr[idx+1:]
		}
	} else {
		return time.Time{}, fmt.Errorf("cannot parse %q as datetime: no separator found", s)
	}

	// Parse date part
	var parsedDate time.Time
	var dateErr error
	for _, layout := range dateLayouts {
		parsedDate, dateErr = time.ParseInLocation(layout, datePart, loc)
		if dateErr == nil {
			break
		}
	}
	if dateErr != nil {
		return time.Time{}, fmt.Errorf("cannot parse date part %q: %w", datePart, dateErr)
	}

	// Parse time part
	var parsedTime time.Time
	var timeErr error
	for _, layout := range timeLayouts {
		parsedTime, timeErr = time.ParseInLocation(layout, timePart, loc)
		if timeErr == nil {
			break
		}
	}
	if timeErr != nil {
		return time.Time{}, fmt.Errorf("cannot parse time part %q: %w", timePart, timeErr)
	}

	// Combine date and time
	return time.Date(
		parsedDate.Year(), parsedDate.Month(), parsedDate.Day(),
		parsedTime.Hour(), parsedTime.Minute(), parsedTime.Second(), parsedTime.Nanosecond(),
		loc,
	), nil
}
