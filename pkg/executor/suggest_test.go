package executor

import (
	"regexp"
	"strings"
	"testing"
)

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"kitten", "sitting", 3},
		{"saturday", "sunday", 3},
		{"flaw", "lawn", 2},
	}
	for _, tt := range tests {
		got := levenshtein(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestStripRegexMeta(t *testing.T) {
	tests := []struct {
		pattern string
		want    string
	}{
		{`^I have (\d+) apples$`, "i have _ apples"},
		{`^the user says "([^"]*)"$`, `the user says "_"`},
		{`^I select {color}$`, "i select _"},
		{`^simple text$`, "simple text"},
		{`^I want a {color} (car|bike) with {int} doors$`, "i want a _ _ with _ doors"},
	}
	for _, tt := range tests {
		got := stripRegexMeta(tt.pattern)
		if got != tt.want {
			t.Errorf("stripRegexMeta(%q) = %q, want %q", tt.pattern, got, tt.want)
		}
	}
}

func makeSteps(patterns ...string) []StepDefinition {
	steps := make([]StepDefinition, len(patterns))
	for i, p := range patterns {
		steps[i] = StepDefinition{
			Pattern: regexp.MustCompile(p),
		}
	}
	return steps
}

func TestSuggestClosestStep(t *testing.T) {
	steps := makeSteps(
		`^I have (\d+) apples$`,
		`^I eat (\d+) oranges$`,
		`^the basket contains (\d+) fruits$`,
		`^I go to the "([^"]*)" page$`,
	)

	tests := []struct {
		name      string
		stepText  string
		wantMatch string
		wantFound bool
	}{
		{
			name:      "close match - oranges matches eat pattern",
			stepText:  "I have 5 oranges",
			wantMatch: `^I eat (\d+) oranges$`,
			wantFound: true,
		},
		{
			name:      "close match - apples matches have pattern",
			stepText:  "I eat 3 apples",
			wantMatch: `^I have (\d+) apples$`,
			wantFound: true,
		},
		{
			name:      "totally different text - no match",
			stepText:  "the server responds with status code 200 and a JSON body containing the user profile",
			wantMatch: "",
			wantFound: false,
		},
		{
			name:      "exact text matches a pattern",
			stepText:  "I have 10 apples",
			wantMatch: `^I have (\d+) apples$`,
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := SuggestClosestStep(tt.stepText, steps)
			if found != tt.wantFound {
				t.Errorf("SuggestClosestStep(%q) found=%v, want %v", tt.stepText, found, tt.wantFound)
			}
			if found && got != tt.wantMatch {
				t.Errorf("SuggestClosestStep(%q) = %q, want %q", tt.stepText, got, tt.wantMatch)
			}
		})
	}
}

func TestSuggestClosestStep_EmptySteps(t *testing.T) {
	_, found := SuggestClosestStep("I have 5 apples", nil)
	if found {
		t.Error("expected no suggestion for empty step list")
	}
}

func TestGenerateStepSnippet(t *testing.T) {
	tests := []struct {
		name     string
		keyword  string
		stepText string
		wantHas  []string // substrings that must be present
	}{
		{
			name:     "simple text no params",
			keyword:  "Given ",
			stepText: "I am logged in",
			wantHas: []string{
				`// @cacik ` + "`^I am logged in$`",
				"func GivenIAmLoggedIn(ctx *cacik.Context)",
				"// TODO: implement step",
			},
		},
		{
			name:     "text with integer",
			keyword:  "Given ",
			stepText: "I have 5 apples",
			wantHas: []string{
				`(\d+)`,
				"arg1 int",
				"// @cacik",
			},
		},
		{
			name:     "text with quoted string",
			keyword:  "When ",
			stepText: `I go to the "home" page`,
			wantHas: []string{
				`"([^"]*)"`,
				"arg1 string",
			},
		},
		{
			name:     "text with float",
			keyword:  "Then ",
			stepText: "the price is 3.14 dollars",
			wantHas: []string{
				`([\d.]+)`,
				"arg1 float64",
			},
		},
		{
			name:     "mixed params",
			keyword:  "Given ",
			stepText: `I have 5 "red" apples costing 2.50 each`,
			wantHas: []string{
				`(\d+)`,
				`"([^"]*)"`,
				`([\d.]+)`,
				"arg1 string",  // quoted string detected first
				"arg2 float64", // float detected before int
				"arg3 int",     // remaining integer
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateStepSnippet(tt.keyword, tt.stepText)
			for _, sub := range tt.wantHas {
				if !strings.Contains(got, sub) {
					t.Errorf("GenerateStepSnippet(%q, %q) missing %q\nGot:\n%s",
						tt.keyword, tt.stepText, sub, got)
				}
			}
		})
	}
}

func TestStepTextToFuncName(t *testing.T) {
	tests := []struct {
		keyword string
		text    string
		want    string
	}{
		{"Given ", "I am logged in", "GivenIAmLoggedIn"},
		{"When ", "I click the button", "WhenIClickTheButton"},
		{"Then ", "the result is 42", "ThenTheResultIs42"},
		{"Given ", `I have "some" items`, "GivenIHaveSomeItems"},
		{"", "", "UndefinedStep"},
	}

	for _, tt := range tests {
		got := stepTextToFuncName(tt.keyword, tt.text)
		if got != tt.want {
			t.Errorf("stepTextToFuncName(%q, %q) = %q, want %q", tt.keyword, tt.text, got, tt.want)
		}
	}
}

func TestFormatStepSuggestion(t *testing.T) {
	steps := makeSteps(
		`^I have (\d+) apples$`,
		`^I eat (\d+) oranges$`,
	)

	t.Run("with close match", func(t *testing.T) {
		result := FormatStepSuggestion("Given ", "I have 5 oranges", steps)
		if !strings.Contains(result, "Did you mean?") {
			t.Error("expected 'Did you mean?' section")
		}
		if !strings.Contains(result, "You can implement this step with:") {
			t.Error("expected code snippet section")
		}
		if !strings.Contains(result, "// @cacik") {
			t.Error("expected @cacik annotation in snippet")
		}
	})

	t.Run("without close match", func(t *testing.T) {
		result := FormatStepSuggestion("Given ", "the server returns a complex JSON response with nested objects and arrays", steps)
		if strings.Contains(result, "Did you mean?") {
			t.Error("did not expect 'Did you mean?' for very different text")
		}
		if !strings.Contains(result, "You can implement this step with:") {
			t.Error("expected code snippet even without close match")
		}
	})

	t.Run("no registered steps", func(t *testing.T) {
		result := FormatStepSuggestion("Given ", "anything", nil)
		if strings.Contains(result, "Did you mean?") {
			t.Error("should not suggest when no steps registered")
		}
		if !strings.Contains(result, "You can implement this step with:") {
			t.Error("should still show code snippet")
		}
	})
}
