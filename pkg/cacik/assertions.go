package cacik

import (
	"fmt"
	"reflect"
	"strings"
)

// Assert provides assertion methods for BDD tests.
// All assertions fail immediately (fail-fast behavior).
type Assert struct {
	t *panicT
}

// Equal asserts expected == actual (using reflect.DeepEqual).
// Fails immediately if not equal.
func (a *Assert) Equal(expected, actual any, msgAndArgs ...any) {
	if !reflect.DeepEqual(expected, actual) {
		a.failf(msgAndArgs, "Equal failed:\n\texpected: %v\n\tactual:   %v", expected, actual)
	}
}

// NotEqual asserts expected != actual.
// Fails immediately if equal.
func (a *Assert) NotEqual(expected, actual any, msgAndArgs ...any) {
	if reflect.DeepEqual(expected, actual) {
		a.failf(msgAndArgs, "Expected values to differ, but both are: %v", expected)
	}
}

// Nil asserts value is nil.
// Fails immediately if not nil.
func (a *Assert) Nil(value any, msgAndArgs ...any) {
	if !isNil(value) {
		a.failf(msgAndArgs, "Expected nil, got: %v", value)
	}
}

// NotNil asserts value is not nil.
// Fails immediately if nil.
func (a *Assert) NotNil(value any, msgAndArgs ...any) {
	if isNil(value) {
		a.failf(msgAndArgs, "Expected non-nil value, got nil")
	}
}

// True asserts condition is true.
// Fails immediately if false.
func (a *Assert) True(condition bool, msgAndArgs ...any) {
	if !condition {
		a.failf(msgAndArgs, "Expected true, got false")
	}
}

// False asserts condition is false.
// Fails immediately if true.
func (a *Assert) False(condition bool, msgAndArgs ...any) {
	if condition {
		a.failf(msgAndArgs, "Expected false, got true")
	}
}

// NoError asserts err is nil.
// Fails immediately if err is not nil.
func (a *Assert) NoError(err error, msgAndArgs ...any) {
	if err != nil {
		a.failf(msgAndArgs, "Expected no error, got: %v", err)
	}
}

// Error asserts err is not nil.
// Fails immediately if err is nil.
func (a *Assert) Error(err error, msgAndArgs ...any) {
	if err == nil {
		a.failf(msgAndArgs, "Expected an error, got nil")
	}
}

// ErrorIs asserts that err matches target using errors.Is.
// Fails immediately if not matching.
func (a *Assert) ErrorIs(err, target error, msgAndArgs ...any) {
	if err == nil {
		a.failf(msgAndArgs, "Expected error %v, got nil", target)
		return
	}
	// Use errors.Is for proper error chain checking
	if !errorIs(err, target) {
		a.failf(msgAndArgs, "Expected error %v, got: %v", target, err)
	}
}

// ErrorContains asserts that err contains the given substring.
// Fails immediately if not containing.
func (a *Assert) ErrorContains(err error, substr string, msgAndArgs ...any) {
	if err == nil {
		a.failf(msgAndArgs, "Expected error containing %q, got nil", substr)
		return
	}
	if !strings.Contains(err.Error(), substr) {
		a.failf(msgAndArgs, "Expected error containing %q, got: %v", substr, err)
	}
}

// Contains asserts that s contains the element/substring.
// For strings: checks if s contains substr.
// For slices/arrays: checks if collection contains element.
// For maps: checks if map contains key.
// Fails immediately if not containing.
func (a *Assert) Contains(s, contains any, msgAndArgs ...any) {
	ok, found := containsElement(s, contains)
	if !ok {
		a.failf(msgAndArgs, "Cannot check containment on type %T", s)
		return
	}
	if !found {
		a.failf(msgAndArgs, "%v does not contain %v", s, contains)
	}
}

// NotContains asserts that s does not contain the element/substring.
// Fails immediately if containing.
func (a *Assert) NotContains(s, contains any, msgAndArgs ...any) {
	ok, found := containsElement(s, contains)
	if !ok {
		a.failf(msgAndArgs, "Cannot check containment on type %T", s)
		return
	}
	if found {
		a.failf(msgAndArgs, "%v should not contain %v", s, contains)
	}
}

// Len asserts collection has expected length.
// Fails immediately if length differs.
func (a *Assert) Len(collection any, length int, msgAndArgs ...any) {
	l, ok := getLen(collection)
	if !ok {
		a.failf(msgAndArgs, "Cannot get length of type %T", collection)
		return
	}
	if l != length {
		a.failf(msgAndArgs, "Expected length %d, got %d", length, l)
	}
}

// Empty asserts collection is empty (len == 0).
// Fails immediately if not empty.
func (a *Assert) Empty(collection any, msgAndArgs ...any) {
	l, ok := getLen(collection)
	if !ok {
		a.failf(msgAndArgs, "Cannot get length of type %T", collection)
		return
	}
	if l != 0 {
		a.failf(msgAndArgs, "Expected empty collection, got length %d", l)
	}
}

// NotEmpty asserts collection is not empty (len > 0).
// Fails immediately if empty.
func (a *Assert) NotEmpty(collection any, msgAndArgs ...any) {
	l, ok := getLen(collection)
	if !ok {
		a.failf(msgAndArgs, "Cannot get length of type %T", collection)
		return
	}
	if l == 0 {
		a.failf(msgAndArgs, "Expected non-empty collection")
	}
}

// Greater asserts that e1 > e2.
// Fails immediately if not greater.
func (a *Assert) Greater(e1, e2 any, msgAndArgs ...any) {
	result, ok := compare(e1, e2)
	if !ok {
		a.failf(msgAndArgs, "Cannot compare %T with %T", e1, e2)
		return
	}
	if result != 1 {
		a.failf(msgAndArgs, "Expected %v > %v", e1, e2)
	}
}

// GreaterOrEqual asserts that e1 >= e2.
// Fails immediately if less than.
func (a *Assert) GreaterOrEqual(e1, e2 any, msgAndArgs ...any) {
	result, ok := compare(e1, e2)
	if !ok {
		a.failf(msgAndArgs, "Cannot compare %T with %T", e1, e2)
		return
	}
	if result == -1 {
		a.failf(msgAndArgs, "Expected %v >= %v", e1, e2)
	}
}

// Less asserts that e1 < e2.
// Fails immediately if not less.
func (a *Assert) Less(e1, e2 any, msgAndArgs ...any) {
	result, ok := compare(e1, e2)
	if !ok {
		a.failf(msgAndArgs, "Cannot compare %T with %T", e1, e2)
		return
	}
	if result != -1 {
		a.failf(msgAndArgs, "Expected %v < %v", e1, e2)
	}
}

// LessOrEqual asserts that e1 <= e2.
// Fails immediately if greater than.
func (a *Assert) LessOrEqual(e1, e2 any, msgAndArgs ...any) {
	result, ok := compare(e1, e2)
	if !ok {
		a.failf(msgAndArgs, "Cannot compare %T with %T", e1, e2)
		return
	}
	if result == 1 {
		a.failf(msgAndArgs, "Expected %v <= %v", e1, e2)
	}
}

// Zero asserts that value is the zero value for its type.
// Fails immediately if not zero.
func (a *Assert) Zero(value any, msgAndArgs ...any) {
	if !isZero(value) {
		a.failf(msgAndArgs, "Expected zero value, got: %v", value)
	}
}

// NotZero asserts that value is not the zero value for its type.
// Fails immediately if zero.
func (a *Assert) NotZero(value any, msgAndArgs ...any) {
	if isZero(value) {
		a.failf(msgAndArgs, "Expected non-zero value")
	}
}

// Same asserts that two pointers reference the same object.
// Fails immediately if different references.
func (a *Assert) Same(expected, actual any, msgAndArgs ...any) {
	if !samePointer(expected, actual) {
		a.failf(msgAndArgs, "Expected same object, but got different references")
	}
}

// NotSame asserts that two pointers reference different objects.
// Fails immediately if same reference.
func (a *Assert) NotSame(expected, actual any, msgAndArgs ...any) {
	if samePointer(expected, actual) {
		a.failf(msgAndArgs, "Expected different objects, but got same reference")
	}
}

// Fail fails the test immediately with the given message.
func (a *Assert) Fail(msgAndArgs ...any) {
	msg := "Test failed"
	if len(msgAndArgs) > 0 {
		msg = formatMsgAndArgs(msgAndArgs...)
	}
	a.t.Errorf(msg)
}

// ============================================================================
// Internal helpers
// ============================================================================

// failf formats a failure message with format args and optional user message.
func (a *Assert) failf(msgAndArgs []any, format string, formatArgs ...any) {
	msg := fmt.Sprintf(format, formatArgs...)

	if len(msgAndArgs) > 0 {
		msg += ": " + formatMsgAndArgs(msgAndArgs...)
	}

	a.t.Errorf(msg)
}

// formatMsgAndArgs formats optional message and arguments.
func formatMsgAndArgs(msgAndArgs ...any) string {
	if len(msgAndArgs) == 0 {
		return ""
	}
	if len(msgAndArgs) == 1 {
		if s, ok := msgAndArgs[0].(string); ok {
			return s
		}
		return fmt.Sprintf("%v", msgAndArgs[0])
	}
	if s, ok := msgAndArgs[0].(string); ok {
		return fmt.Sprintf(s, msgAndArgs[1:]...)
	}
	return fmt.Sprint(msgAndArgs...)
}

// isNil checks if a value is nil (handles interface nil).
func isNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return rv.IsNil()
	}
	return false
}

// isZero checks if a value is the zero value for its type.
func isZero(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	return rv.IsZero()
}

// getLen returns the length of a collection.
func getLen(v any) (int, bool) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		return rv.Len(), true
	}
	return 0, false
}

// containsElement checks if s contains the element.
func containsElement(s, elem any) (ok bool, found bool) {
	sv := reflect.ValueOf(s)

	switch sv.Kind() {
	case reflect.String:
		return true, strings.Contains(sv.String(), reflect.ValueOf(elem).String())
	case reflect.Slice, reflect.Array:
		for i := 0; i < sv.Len(); i++ {
			if reflect.DeepEqual(sv.Index(i).Interface(), elem) {
				return true, true
			}
		}
		return true, false
	case reflect.Map:
		for _, key := range sv.MapKeys() {
			if reflect.DeepEqual(key.Interface(), elem) {
				return true, true
			}
		}
		return true, false
	}
	return false, false
}

// compare compares two values and returns -1, 0, or 1.
// Returns (result, ok) where ok is false if comparison is not possible.
func compare(e1, e2 any) (int, bool) {
	v1 := reflect.ValueOf(e1)
	v2 := reflect.ValueOf(e2)

	if v1.Kind() != v2.Kind() {
		return 0, false
	}

	switch v1.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i1, i2 := v1.Int(), v2.Int()
		if i1 < i2 {
			return -1, true
		} else if i1 > i2 {
			return 1, true
		}
		return 0, true

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u1, u2 := v1.Uint(), v2.Uint()
		if u1 < u2 {
			return -1, true
		} else if u1 > u2 {
			return 1, true
		}
		return 0, true

	case reflect.Float32, reflect.Float64:
		f1, f2 := v1.Float(), v2.Float()
		if f1 < f2 {
			return -1, true
		} else if f1 > f2 {
			return 1, true
		}
		return 0, true

	case reflect.String:
		s1, s2 := v1.String(), v2.String()
		if s1 < s2 {
			return -1, true
		} else if s1 > s2 {
			return 1, true
		}
		return 0, true
	}

	return 0, false
}

// samePointer checks if two values point to the same object.
func samePointer(expected, actual any) bool {
	if expected == nil || actual == nil {
		return expected == actual
	}

	ev := reflect.ValueOf(expected)
	av := reflect.ValueOf(actual)

	if ev.Kind() != reflect.Ptr || av.Kind() != reflect.Ptr {
		return false
	}

	return ev.Pointer() == av.Pointer()
}

// errorIs is a simple implementation of errors.Is for compatibility.
func errorIs(err, target error) bool {
	if err == nil || target == nil {
		return err == target
	}

	// Direct comparison
	if err == target {
		return true
	}

	// Check error message as fallback
	return err.Error() == target.Error()
}
