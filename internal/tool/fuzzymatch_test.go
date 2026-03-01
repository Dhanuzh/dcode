package tool

import (
	"testing"
)

func TestSimpleReplacer_ExactMatch(t *testing.T) {
	content := "hello world\nfoo bar\nbaz"
	result, err := FuzzyReplace(content, "foo bar", "replaced", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "hello world\nreplaced\nbaz"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSimpleReplacer_NotFound(t *testing.T) {
	content := "hello world"
	_, err := FuzzyReplace(content, "not here", "replaced", false)
	if err == nil {
		t.Fatal("expected error for not-found string")
	}
}

func TestLineTrimmedReplacer_IndentationDifference(t *testing.T) {
	content := "func main() {\n\t\tfmt.Println(\"hello\")\n\t\treturn\n}"
	// LLM provides with spaces instead of tabs
	oldString := "  fmt.Println(\"hello\")\n  return"
	result, err := FuzzyReplace(content, oldString, "\t\tfmt.Println(\"world\")\n\t\treturn", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "func main() {\n\t\tfmt.Println(\"world\")\n\t\treturn\n}" {
		t.Errorf("unexpected result: %q", result)
	}
}

func TestWhitespaceNormalizedReplacer(t *testing.T) {
	content := "hello   world   here"
	// LLM collapses extra spaces
	oldString := "hello world here"
	result, err := FuzzyReplace(content, oldString, "replaced", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "replaced"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestIndentationFlexibleReplacer(t *testing.T) {
	content := "class Foo {\n    func bar() {\n        doSomething()\n    }\n}"
	// LLM provides at wrong nesting level but correct relative indentation
	oldString := "func bar() {\n    doSomething()\n}"
	result, err := FuzzyReplace(content, oldString, "func bar() {\n    doSomethingElse()\n}", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == content {
		t.Error("content was not modified")
	}
}

func TestEscapeNormalizedReplacer(t *testing.T) {
	content := "line1\nline2\nline3"
	// LLM double-escaped the newline
	oldString := "line1\\nline2"
	result, err := FuzzyReplace(content, oldString, "lineA\nlineB", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "lineA\nlineB\nline3"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestTrimmedBoundaryReplacer(t *testing.T) {
	content := "hello world"
	// LLM added extra whitespace around the block
	oldString := "  hello world  "
	result, err := FuzzyReplace(content, oldString, "replaced", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "replaced" {
		t.Errorf("expected 'replaced', got %q", result)
	}
}

func TestMultipleMatches_Error(t *testing.T) {
	content := "foo\nbar\nfoo\nbaz"
	_, err := FuzzyReplace(content, "foo", "replaced", false)
	if err == nil {
		t.Fatal("expected error for multiple matches")
	}
	if err.Error() != "Found multiple matches for oldString. Provide more surrounding lines in oldString to identify the correct match." {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestMultipleMatches_ReplaceAll(t *testing.T) {
	content := "foo\nbar\nfoo\nbaz"
	result, err := FuzzyReplace(content, "foo", "replaced", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "replaced\nbar\nreplaced\nbaz"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestBlockAnchorReplacer_FuzzyMiddle(t *testing.T) {
	content := "func main() {\n\tx := 1\n\ty := 2\n\tz := 3\n\treturn\n}"
	// LLM got first and last lines right, but middle is slightly wrong
	oldString := "func main() {\n\tx := 10\n\ty := 20\n\tz := 30\n\treturn\n}"
	result, err := FuzzyReplace(content, oldString, "func main() {\n\ta := 1\n\treturn\n}", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == content {
		t.Error("content was not modified")
	}
}

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"kitten", "sitting", 3},
	}

	for _, tc := range tests {
		got := levenshtein(tc.a, tc.b)
		if got != tc.expected {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.expected)
		}
	}
}

func TestContextAwareReplacer(t *testing.T) {
	content := "if true {\n\tline1\n\tline2\n\tline3\n}"
	// First and last lines correct, some middle lines wrong but >50% right
	oldString := "if true {\n\tline1\n\tWRONG\n\tline3\n}"
	result, err := FuzzyReplace(content, oldString, "if false {\n}", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == content {
		t.Error("content was not modified")
	}
}
