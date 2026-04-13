package executor

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// levenshtein computes the Levenshtein edit distance between two strings.
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	// Use a single row + prev-value approach to save memory.
	prev := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		cur := make([]int, lb+1)
		cur[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			ins := cur[j-1] + 1
			del := prev[j] + 1
			sub := prev[j-1] + cost
			cur[j] = min3(ins, del, sub)
		}
		prev = cur
	}
	return prev[lb]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// captureGroupRe matches common regex capture groups: (\d+), ([^"]*), (true|false), etc.
var captureGroupRe = regexp.MustCompile(`\([^()]*\)`)

// placeholderTypeRe matches {type} placeholders like {int}, {string}, {float}.
var placeholderTypeRe = regexp.MustCompile(`\{[a-zA-Z]+\}`)

// stripRegexMeta converts a regex pattern to a human-readable form for
// comparison purposes. It removes anchors (^, $), replaces capture groups and
// {type} placeholders with a generic placeholder word, and collapses whitespace.
func stripRegexMeta(pattern string) string {
	s := pattern
	// Remove anchors
	s = strings.TrimPrefix(s, "^")
	s = strings.TrimSuffix(s, "$")
	// Replace capture groups with a placeholder
	s = captureGroupRe.ReplaceAllString(s, "_")
	// Replace {type} placeholders
	s = placeholderTypeRe.ReplaceAllString(s, "_")
	// Collapse multiple underscores/spaces
	s = strings.Join(strings.Fields(s), " ")
	return strings.ToLower(s)
}

// SuggestClosestStep finds the registered step definition whose pattern is
// most similar to the unmatched step text. Returns the original pattern string
// and true if a match within the similarity threshold is found.
func SuggestClosestStep(stepText string, steps []StepDefinition) (string, bool) {
	if len(steps) == 0 {
		return "", false
	}

	normalizedText := strings.ToLower(stepText)
	bestDist := -1
	bestPattern := ""

	for _, sd := range steps {
		pat := sd.Pattern.String()
		normalizedPat := stripRegexMeta(pat)
		dist := levenshtein(normalizedText, normalizedPat)
		if bestDist < 0 || dist < bestDist {
			bestDist = dist
			bestPattern = pat
		}
	}

	// Only suggest if the distance is within 50% of the step text length.
	// For very short texts, use a minimum threshold of 5.
	threshold := len(normalizedText) / 2
	if threshold < 5 {
		threshold = 5
	}
	if bestDist <= threshold {
		return bestPattern, true
	}
	return "", false
}

// numberRe matches standalone integers in step text (e.g., "5", "100").
var numberRe = regexp.MustCompile(`\b\d+\b`)

// floatRe matches standalone float numbers in step text (e.g., "3.14", "0.5").
var floatRe = regexp.MustCompile(`\b\d+\.\d+\b`)

// quotedStringRe matches double-quoted strings in step text.
var quotedStringRe = regexp.MustCompile(`"[^"]*"`)

// GenerateStepSnippet generates a Go function stub with a // @cacik annotation
// for an unmatched step. It auto-detects parameters (numbers, quoted strings)
// in the step text and produces appropriate capture groups and typed function
// parameters.
func GenerateStepSnippet(keyword, stepText string) string {
	pattern := stepText
	var params []param

	// Order matters: detect floats before ints so "3.14" isn't split.
	// Detect quoted strings first.

	// 1. Quoted strings → "([^"]*)" with string param
	pattern = quotedStringRe.ReplaceAllStringFunc(pattern, func(match string) string {
		params = append(params, param{name: fmt.Sprintf("arg%d", len(params)+1), typ: "string"})
		return `"([^"]*)"`
	})

	// 2. Float numbers → ([\d.]+) with float64 param
	pattern = floatRe.ReplaceAllStringFunc(pattern, func(match string) string {
		params = append(params, param{name: fmt.Sprintf("arg%d", len(params)+1), typ: "float64"})
		return `([\d.]+)`
	})

	// 3. Integer numbers → (\d+) with int param
	pattern = numberRe.ReplaceAllStringFunc(pattern, func(match string) string {
		params = append(params, param{name: fmt.Sprintf("arg%d", len(params)+1), typ: "int"})
		return `(\d+)`
	})

	// Add regex anchors
	pattern = "^" + pattern + "$"

	// Build function name from step text words (PascalCase)
	funcName := stepTextToFuncName(keyword, stepText)

	// Build parameter list
	paramList := "ctx *cacik.Context"
	for _, p := range params {
		paramList += fmt.Sprintf(", %s %s", p.name, p.typ)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("// %s\n", funcName))
	sb.WriteString(fmt.Sprintf("// @cacik `%s`\n", pattern))
	sb.WriteString(fmt.Sprintf("func %s(%s) {\n", funcName, paramList))
	sb.WriteString("    // TODO: implement step\n")
	sb.WriteString("}")

	return sb.String()
}

type param struct {
	name string
	typ  string
}

// stepTextToFuncName converts a Gherkin step keyword and text into a PascalCase
// Go function name. For example: keyword="Given ", text="I have 5 apples"
// → "IHave5Apples".
func stepTextToFuncName(keyword, text string) string {
	// Combine keyword (trimmed) and text
	combined := strings.TrimSpace(keyword) + " " + text

	// Remove quotes and special characters, keep alphanumeric and spaces
	var cleaned strings.Builder
	for _, r := range combined {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cleaned.WriteRune(r)
		} else if r == ' ' || r == '_' || r == '-' {
			cleaned.WriteRune(' ')
		}
	}

	// Split into words and PascalCase them
	words := strings.Fields(cleaned.String())
	var name strings.Builder
	for _, w := range words {
		if len(w) == 0 {
			continue
		}
		// If the word starts with a digit, just append it as-is
		if unicode.IsDigit(rune(w[0])) {
			name.WriteString(w)
		} else {
			// Capitalize first letter
			runes := []rune(w)
			runes[0] = unicode.ToUpper(runes[0])
			name.WriteString(string(runes))
		}
	}

	result := name.String()
	if result == "" {
		return "UndefinedStep"
	}

	// If the name starts with a digit, prefix with "Step"
	if unicode.IsDigit(rune(result[0])) {
		return "Step" + result
	}
	return result
}

// FormatStepSuggestion builds the full suggestion message to append to an
// "undefined step" error. It includes both a "Did you mean?" section (if a
// close match exists) and a "You can implement this step with:" code snippet.
func FormatStepSuggestion(keyword, stepText string, steps []StepDefinition) string {
	var sb strings.Builder

	// "Did you mean?" section
	if closest, found := SuggestClosestStep(stepText, steps); found {
		sb.WriteString("\nDid you mean?\n")
		sb.WriteString(fmt.Sprintf("    %s\n", closest))
	}

	// Code snippet section
	snippet := GenerateStepSnippet(keyword, stepText)
	sb.WriteString("\nYou can implement this step with:\n\n")
	for _, line := range strings.Split(snippet, "\n") {
		sb.WriteString(fmt.Sprintf("    %s\n", line))
	}

	return sb.String()
}
