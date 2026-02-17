package main

import (
	"fmt"

	"github.com/Dhanuzh/dcode/internal/tui/components"
)

func main() {
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("              DCode TUI Components Test")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()

	// Test 1: Syntax Highlighting
	testSyntaxHighlighting()

	// Test 2: Markdown Rendering
	testMarkdownRendering()

	// Test 3: Diff Viewing
	testDiffViewing()

	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("                   All Tests Complete!")
	fmt.Println("═══════════════════════════════════════════════════════════")
}

func testSyntaxHighlighting() {
	fmt.Println("━━━ Test 1: Syntax Highlighting ━━━")
	fmt.Println()

	highlighter := components.NewSyntaxHighlighter("monokai")

	// Go code example
	goCode := `package main

import "fmt"

func main() {
	message := "Hello, World!"
	fmt.Println(message)
}
`

	fmt.Println("Example: Go Code")
	fmt.Println(highlighter.HighlightCodeBlock(goCode, "go"))
	fmt.Println()

	// Python example
	pythonCode := `def fibonacci(n):
    """Calculate Fibonacci number."""
    if n <= 1:
        return n
    return fibonacci(n-1) + fibonacci(n-2)

print(fibonacci(10))
`

	fmt.Println("Example: Python Code")
	fmt.Println(highlighter.HighlightCodeBlock(pythonCode, "python"))
	fmt.Println()

	// JSON example
	jsonCode := `{
  "name": "dcode",
  "version": "1.0.0",
  "features": ["ai", "tui", "plugins"]
}
`

	fmt.Println("Example: JSON")
	fmt.Println(highlighter.HighlightJSON(jsonCode))
	fmt.Println()
}

func testMarkdownRendering() {
	fmt.Println("━━━ Test 2: Markdown Rendering ━━━")
	fmt.Println()

	renderer, err := components.NewMarkdownRenderer(80, "dark")
	if err != nil {
		fmt.Printf("Error creating renderer: %v\n", err)
		return
	}

	// Test heading
	h1 := renderer.RenderHeading("Welcome to DCode", 1)
	h2 := renderer.RenderHeading("Features", 2)
	fmt.Println(h1)
	fmt.Println(h2)
	fmt.Println()

	// Test list
	items := []string{
		"Syntax highlighting for 30+ languages",
		"Beautiful markdown rendering",
		"Git diff visualization",
		"Reusable component library",
	}
	fmt.Println(renderer.RenderList(items))
	fmt.Println()

	// Test quote
	quote := "The best way to predict the future is to implement it."
	fmt.Println(renderer.RenderQuote(quote))
	fmt.Println()

	// Test emphasis
	fmt.Println(renderer.RenderEmphasis("This is italic text", false))
	fmt.Println(renderer.RenderEmphasis("This is bold text", true))
	fmt.Println()

	// Test horizontal rule
	fmt.Println(renderer.RenderHorizontalRule())
	fmt.Println()
}

func testDiffViewing() {
	fmt.Println("━━━ Test 3: Diff Viewing ━━━")
	fmt.Println()

	viewer := components.NewDiffViewer(80, "monokai")

	// Sample git diff
	gitDiff := `diff --git a/main.go b/main.go
index 1234567..abcdefg 100644
--- a/main.go
+++ b/main.go
@@ -1,7 +1,8 @@
 package main

 import (
 	"fmt"
+	"log"
 )

 func main() {
-	fmt.Println("Hello")
+	log.Println("Hello, World!")
 }
`

	fmt.Println("Example: Git Diff (Color Coded)")
	fmt.Println(viewer.RenderSimple(gitDiff))
	fmt.Println()

	// Calculate stats
	stats := components.CalculateStats(gitDiff)
	fmt.Println(viewer.RenderStats(stats))
	fmt.Println()

	// Test inline diff
	fmt.Println("Example: Inline Diff (compact)")
	fmt.Println(viewer.RenderInline(gitDiff))
	fmt.Println()
}