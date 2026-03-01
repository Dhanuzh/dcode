package components

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
)

// MarkdownRenderer renders markdown with syntax highlighting
type MarkdownRenderer struct {
	renderer          *glamour.TermRenderer
	syntaxHighlighter *SyntaxHighlighter
	width             int
	theme             string // preserved for SetWidth recreation
}

// ptr helpers
func boolPtr(b bool) *bool    { return &b }
func uintPtr(u uint) *uint    { return &u }
func strPtr(s string) *string { return &s }

// dcodeStyle builds a clean, readable glamour StyleConfig for dcode.
//
// Goals:
//   - Zero document/paragraph margins so output aligns with our bordered block
//   - Colourful, distinct headings (H1–H3)
//   - Readable bullet/numbered lists with proper indentation
//   - Inline code stands out with a subtle background hint
//   - Paragraphs get a small left margin (2 spaces) for visual breathing room
func dcodeStyle(base ansi.StyleConfig) ansi.StyleConfig {
	s := base

	// ── Document / Paragraph ───────────────────────────────────────────────
	s.Document.Margin = uintPtr(0)
	s.Document.Indent = uintPtr(0)

	s.Paragraph.Margin = uintPtr(0)
	s.Paragraph.Indent = uintPtr(0)

	// ── Headings ────────────────────────────────────────────────────────────
	// H1: bold purple, blank line above
	s.H1.Bold = boolPtr(true)
	s.H1.Color = strPtr("#CBA6F7")
	s.H1.Prefix = "  "
	s.H1.Suffix = ""
	s.H1.BlockPrefix = "\n"
	s.H1.BlockSuffix = "\n"
	s.H1.Margin = uintPtr(0)

	// H2: bold blue
	s.H2.Bold = boolPtr(true)
	s.H2.Color = strPtr("#89B4FA")
	s.H2.Prefix = "  "
	s.H2.BlockPrefix = "\n"
	s.H2.BlockSuffix = "\n"
	s.H2.Margin = uintPtr(0)

	// H3: bold green
	s.H3.Bold = boolPtr(true)
	s.H3.Color = strPtr("#A6E3A1")
	s.H3.Prefix = "  "
	s.H3.BlockPrefix = "\n"
	s.H3.BlockSuffix = "\n"
	s.H3.Margin = uintPtr(0)

	// H4–H6: yellow / red / dim
	s.H4.Bold = boolPtr(true)
	s.H4.Color = strPtr("#F9E2AF")
	s.H4.Prefix = "  "
	s.H4.Margin = uintPtr(0)

	s.H5.Bold = boolPtr(true)
	s.H5.Color = strPtr("#F38BA8")
	s.H5.Prefix = "  "
	s.H5.Margin = uintPtr(0)

	s.H6.Bold = boolPtr(false)
	s.H6.Color = strPtr("#A6ADC8")
	s.H6.Prefix = "  "
	s.H6.Margin = uintPtr(0)

	// ── Lists ────────────────────────────────────────────────────────────────
	// Unordered bullets: use a coloured dot, 2-space indent
	s.List.Indent = uintPtr(2)
	s.List.LevelIndent = 2
	s.Item.Prefix = "• "
	s.Item.Color = strPtr("#CDD6F4") // normal text colour

	// Enumeration (ordered lists)
	s.Enumeration.Color = strPtr("#89B4FA")
	s.Enumeration.Bold = boolPtr(false)

	// ── Inline code ──────────────────────────────────────────────────────────
	// Dim background + accent colour so it stands out from surrounding prose
	s.Code.Color = strPtr("#F38BA8")
	s.Code.Bold = boolPtr(false)

	// ── Emphasis / Strong ────────────────────────────────────────────────────
	s.Emph.Italic = boolPtr(true)
	s.Emph.Color = strPtr("#A6ADC8")
	s.Strong.Bold = boolPtr(true)
	s.Strong.Color = strPtr("#CDD6F4")

	// ── Links ────────────────────────────────────────────────────────────────
	s.Link.Color = strPtr("#89B4FA")
	s.Link.Underline = boolPtr(true)
	s.LinkText.Color = strPtr("#89B4FA")

	// ── Block quote ──────────────────────────────────────────────────────────
	s.BlockQuote.Indent = uintPtr(1)
	s.BlockQuote.IndentToken = strPtr("┃ ")
	s.BlockQuote.Color = strPtr("#A6ADC8")
	s.BlockQuote.Italic = boolPtr(true)
	s.BlockQuote.Margin = uintPtr(0)

	// ── Horizontal rule ──────────────────────────────────────────────────────
	s.HorizontalRule.Color = strPtr("#45475A")
	s.HorizontalRule.Format = "────────────────────────────────────────\n"

	// ── Code block (handled by our renderCodeBlock, so suppress glamour's) ──
	// Keep minimal so if glamour ever renders a code block it's at least readable.
	s.CodeBlock.Margin = uintPtr(0)
	s.CodeBlock.Indent = uintPtr(0)

	return s
}

// NewMarkdownRenderer creates a new markdown renderer
func NewMarkdownRenderer(width int, theme string) (*MarkdownRenderer, error) {
	var renderer *glamour.TermRenderer
	var err error

	buildRenderer := func(base ansi.StyleConfig) (*glamour.TermRenderer, error) {
		s := dcodeStyle(base)
		return glamour.NewTermRenderer(glamour.WithStyles(s), glamour.WithWordWrap(width))
	}

	switch theme {
	case "catppuccin", "dark", "monokai":
		renderer, err = buildRenderer(styles.DarkStyleConfig)
	case "dracula":
		renderer, err = buildRenderer(styles.DraculaStyleConfig)
	case "light", "github":
		renderer, err = buildRenderer(styles.LightStyleConfig)
	case "notty":
		renderer, err = buildRenderer(styles.NoTTYStyleConfig)
	default:
		renderer, err = buildRenderer(styles.DarkStyleConfig)
	}

	if err != nil {
		// Fallback: plain word-wrap only
		renderer, _ = glamour.NewTermRenderer(glamour.WithWordWrap(width))
	}

	return &MarkdownRenderer{
		renderer:          renderer,
		syntaxHighlighter: NewSyntaxHighlighter(theme),
		width:             width,
		theme:             theme,
	}, nil
}

// Render renders markdown to terminal output
func (mr *MarkdownRenderer) Render(markdown string) (string, error) {
	// Use glamour for basic markdown rendering
	rendered, err := mr.renderer.Render(markdown)
	if err != nil {
		return markdown, err
	}

	return rendered, nil
}

// RenderWithHighlighting renders markdown with enhanced code block highlighting
func (mr *MarkdownRenderer) RenderWithHighlighting(markdown string) string {
	// First, highlight code blocks
	highlighted := mr.syntaxHighlighter.HighlightMarkdown(markdown)

	// Then render with glamour
	rendered, err := mr.Render(highlighted)
	if err != nil {
		return markdown
	}

	return rendered
}

// SetWidth updates the renderer width, preserving the original theme and zero-margin setting.
func (mr *MarkdownRenderer) SetWidth(width int) {
	mr.width = width
	updated, err := NewMarkdownRenderer(width, mr.theme)
	if err == nil {
		mr.renderer = updated.renderer
	}
}

// RenderCodeBlock renders a standalone code block
func (mr *MarkdownRenderer) RenderCodeBlock(code, language string) string {
	return mr.syntaxHighlighter.HighlightCodeBlock(code, language)
}

// RenderInline renders inline markdown (no paragraphs)
func (mr *MarkdownRenderer) RenderInline(text string) string {
	// Strip markdown formatting for inline display
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "*", "")
	text = strings.ReplaceAll(text, "`", "")
	text = strings.ReplaceAll(text, "#", "")
	return strings.TrimSpace(text)
}

// RenderList renders a markdown list
func (mr *MarkdownRenderer) RenderList(items []string) string {
	var b strings.Builder
	for i, item := range items {
		prefix := "•"
		if i < 10 {
			prefix = string(rune('①' + i)) // Circled numbers
		}
		b.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A6E3A1")).
			Render(prefix))
		b.WriteString(" ")
		b.WriteString(item)
		b.WriteString("\n")
	}
	return b.String()
}

// RenderTable renders a simple table
func (mr *MarkdownRenderer) RenderTable(headers []string, rows [][]string) string {
	// Simple table rendering
	var b strings.Builder

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#CBA6F7"))

	// Headers
	for _, h := range headers {
		b.WriteString(headerStyle.Render(h))
		b.WriteString("  ")
	}
	b.WriteString("\n")

	// Separator
	b.WriteString(strings.Repeat("─", mr.width))
	b.WriteString("\n")

	// Rows
	for _, row := range rows {
		for _, cell := range row {
			b.WriteString(cell)
			b.WriteString("  ")
		}
		b.WriteString("\n")
	}

	return b.String()
}

// RenderQuote renders a blockquote
func (mr *MarkdownRenderer) RenderQuote(text string) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.Border{Left: "┃"}).
		BorderForeground(lipgloss.Color("#6C7086")).
		PaddingLeft(2).
		Foreground(lipgloss.Color("#A6ADC8")).
		Italic(true)

	return style.Render(text)
}

// RenderHeading renders a heading
func (mr *MarkdownRenderer) RenderHeading(text string, level int) string {
	colors := []lipgloss.Color{
		lipgloss.Color("#CBA6F7"), // H1 - Purple
		lipgloss.Color("#89B4FA"), // H2 - Blue
		lipgloss.Color("#A6E3A1"), // H3 - Green
		lipgloss.Color("#F9E2AF"), // H4 - Yellow
		lipgloss.Color("#F38BA8"), // H5 - Red
		lipgloss.Color("#CDD6F4"), // H6 - Text
	}

	if level < 1 {
		level = 1
	}
	if level > 6 {
		level = 6
	}

	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(colors[level-1])

	prefix := strings.Repeat("#", level) + " "
	return style.Render(prefix + text)
}

// RenderLink renders a hyperlink
func (mr *MarkdownRenderer) RenderLink(text, url string) string {
	linkStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#89B4FA")).
		Underline(true)

	return linkStyle.Render(text) + lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6C7086")).
		Render(" ("+url+")")
}

// RenderEmphasis renders emphasized text
func (mr *MarkdownRenderer) RenderEmphasis(text string, strong bool) string {
	if strong {
		return lipgloss.NewStyle().Bold(true).Render(text)
	}
	return lipgloss.NewStyle().Italic(true).Render(text)
}

// RenderHorizontalRule renders a horizontal divider
func (mr *MarkdownRenderer) RenderHorizontalRule() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6C7086")).
		Render(strings.Repeat("─", mr.width))
}
