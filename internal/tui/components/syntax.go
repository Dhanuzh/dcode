package components

import (
	"bytes"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
)

// SyntaxHighlighter provides code syntax highlighting using Chroma
type SyntaxHighlighter struct {
	style     string
	formatter chroma.Formatter
}

// NewSyntaxHighlighter creates a new syntax highlighter
func NewSyntaxHighlighter(themeName string) *SyntaxHighlighter {
	// Map theme names to chroma styles
	styleName := "monokai"
	switch themeName {
	case "catppuccin", "dark":
		styleName = "monokai"
	case "dracula":
		styleName = "dracula"
	case "nord":
		styleName = "nord"
	case "solarized-dark":
		styleName = "solarized-dark"
	case "solarized-light":
		styleName = "solarized-light"
	case "github":
		styleName = "github"
	case "vim":
		styleName = "vim"
	}

	return &SyntaxHighlighter{
		style:     styleName,
		formatter: formatters.Get("terminal256"),
	}
}

// Highlight applies syntax highlighting to code.
// Returns ANSI-escaped lines joined by "\n".
// IMPORTANT: never pass this output directly into a lipgloss Width()-constrained
// Render call — lipgloss miscounts ANSI escape widths and collapses the content
// to a single line. Use RenderCodeBlock or HighlightLines instead.
func (sh *SyntaxHighlighter) Highlight(code, language string) string {
	// Get lexer for language
	lexer := lexers.Get(language)
	if lexer == nil {
		lexer = lexers.Analyse(code)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	// Get style
	style := styles.Get(sh.style)
	if style == nil {
		style = styles.Fallback
	}

	// Format
	var buf bytes.Buffer
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code // Fallback to unhighlighted
	}

	err = sh.formatter.Format(&buf, style, iterator)
	if err != nil {
		return code // Fallback to unhighlighted
	}

	return buf.String()
}

// HighlightLines highlights code and returns individual ANSI-escaped lines.
// Use this when you need to render each line separately to avoid lipgloss
// width-wrapping mangling the ANSI sequences.
func (sh *SyntaxHighlighter) HighlightLines(code, language string) []string {
	highlighted := sh.Highlight(code, language)
	if highlighted == code {
		// fallback — no highlighting applied, split raw
		return strings.Split(code, "\n")
	}
	return strings.Split(highlighted, "\n")
}

// HighlightCodeBlock highlights a code block with optional language hint
func (sh *SyntaxHighlighter) HighlightCodeBlock(code, language string) string {
	highlighted := sh.Highlight(code, language)

	// Add border/frame
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#6C7086")).
		Padding(1, 2)

	return style.Render(highlighted)
}

// DetectLanguage attempts to detect language from filename or content
func DetectLanguage(filename string) string {
	ext := ""
	if idx := strings.LastIndex(filename, "."); idx != -1 {
		ext = filename[idx+1:]
	}

	langMap := map[string]string{
		"go":         "go",
		"js":         "javascript",
		"ts":         "typescript",
		"jsx":        "jsx",
		"tsx":        "tsx",
		"py":         "python",
		"rb":         "ruby",
		"rs":         "rust",
		"c":          "c",
		"cpp":        "cpp",
		"cc":         "cpp",
		"h":          "c",
		"hpp":        "cpp",
		"java":       "java",
		"kt":         "kotlin",
		"swift":      "swift",
		"php":        "php",
		"sh":         "bash",
		"bash":       "bash",
		"zsh":        "bash",
		"fish":       "fish",
		"ps1":        "powershell",
		"sql":        "sql",
		"yaml":       "yaml",
		"yml":        "yaml",
		"json":       "json",
		"xml":        "xml",
		"html":       "html",
		"css":        "css",
		"scss":       "scss",
		"sass":       "sass",
		"md":         "markdown",
		"tex":        "latex",
		"r":          "r",
		"lua":        "lua",
		"vim":        "vim",
		"diff":       "diff",
		"patch":      "diff",
		"makefile":   "make",
		"dockerfile": "docker",
	}

	if lang, ok := langMap[strings.ToLower(ext)]; ok {
		return lang
	}

	return ""
}

// HighlightDiff highlights a git diff
func (sh *SyntaxHighlighter) HighlightDiff(diff string) string {
	return sh.Highlight(diff, "diff")
}

// HighlightJSON highlights JSON data
func (sh *SyntaxHighlighter) HighlightJSON(json string) string {
	return sh.Highlight(json, "json")
}

// ParseMarkdownCodeBlocks finds and highlights code blocks in markdown
type CodeBlock struct {
	Language string
	Code     string
	Start    int
	End      int
}

// FindCodeBlocks extracts code blocks from markdown text
func FindCodeBlocks(text string) []CodeBlock {
	var blocks []CodeBlock
	lines := strings.Split(text, "\n")

	inBlock := false
	var currentBlock CodeBlock
	var codeLines []string

	for i, line := range lines {
		if strings.HasPrefix(line, "```") {
			if !inBlock {
				// Start of code block
				inBlock = true
				currentBlock.Start = i
				currentBlock.Language = strings.TrimPrefix(line, "```")
				currentBlock.Language = strings.TrimSpace(currentBlock.Language)
				codeLines = nil
			} else {
				// End of code block
				inBlock = false
				currentBlock.End = i
				currentBlock.Code = strings.Join(codeLines, "\n")
				blocks = append(blocks, currentBlock)
				currentBlock = CodeBlock{}
			}
		} else if inBlock {
			codeLines = append(codeLines, line)
		}
	}

	return blocks
}

// HighlightMarkdown highlights code blocks within markdown text
func (sh *SyntaxHighlighter) HighlightMarkdown(text string) string {
	blocks := FindCodeBlocks(text)
	if len(blocks) == 0 {
		return text
	}

	// Replace code blocks with highlighted versions
	result := text
	for i := len(blocks) - 1; i >= 0; i-- {
		block := blocks[i]
		highlighted := sh.HighlightCodeBlock(block.Code, block.Language)

		// Find the original block in text
		original := "```" + block.Language + "\n" + block.Code + "\n```"
		result = strings.Replace(result, original, highlighted, 1)
	}

	return result
}

// AvailableStyles returns list of available syntax highlighting styles
func AvailableStyles() []string {
	var styleNames []string
	for _, style := range styles.Registry {
		styleNames = append(styleNames, style.Name)
	}
	return styleNames
}

// AvailableLexers returns list of supported languages
func AvailableLexers() []string {
	var langs []string
	// Get all registered lexers
	allLexers := lexers.GlobalLexerRegistry.Lexers
	for _, lexer := range allLexers {
		config := lexer.Config()
		langs = append(langs, config.Name)
	}
	return langs
}
