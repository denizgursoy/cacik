package executor

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	messages "github.com/cucumber/messages/go/v21"
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
	steps       []StepDefinition
	patternSet  map[string]bool            // Track registered patterns for duplicate detection
	customTypes map[string]*CustomTypeInfo // type name -> custom type info
	context     context.Context
}

// NewStepExecutor creates a new StepExecutor
func NewStepExecutor() *StepExecutor {
	return &StepExecutor{
		steps:       make([]StepDefinition, 0),
		patternSet:  make(map[string]bool),
		customTypes: make(map[string]*CustomTypeInfo),
		context:     context.Background(),
	}
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

		converted, err := e.convertArg(arg, paramType)
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
		v, err := parseBool(arg)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(v), nil

	default:
		return reflect.Value{}, fmt.Errorf("unsupported parameter type: %s", targetType.Kind())
	}
}

// parseBool converts a string to a boolean value.
// It supports human-readable values in addition to standard bool strings.
// Truthy values: true, yes, on, enabled, 1
// Falsy values: false, no, off, disabled, 0
// All comparisons are case-insensitive.
func parseBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "true", "yes", "on", "enabled", "1":
		return true, nil
	case "false", "no", "off", "disabled", "0":
		return false, nil
	default:
		return false, fmt.Errorf("cannot parse %q as bool", s)
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
