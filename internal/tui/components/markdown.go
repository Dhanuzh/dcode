package components

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// MarkdownRenderer renders markdown with syntax highlighting
type MarkdownRenderer struct {
	renderer       *glamour.TermRenderer
	syntaxHighlighter *SyntaxHighlighter
	width          int
}

// NewMarkdownRenderer creates a new markdown renderer
func NewMarkdownRenderer(width int, theme string) (*MarkdownRenderer, error) {
	// Map theme to glamour style
	var renderer *glamour.TermRenderer
	var err error

	switch theme {
	case "catppuccin", "dark", "monokai":
		renderer, err = glamour.NewTermRenderer(glamour.WithStylePath("dark"), glamour.WithWordWrap(width))
	case "dracula":
		renderer, err = glamour.NewTermRenderer(glamour.WithStylePath("dracula"), glamour.WithWordWrap(width))
	case "light", "github":
		renderer, err = glamour.NewTermRenderer(glamour.WithStylePath("light"), glamour.WithWordWrap(width))
	case "notty":
		renderer, err = glamour.NewTermRenderer(glamour.WithStylePath("notty"), glamour.WithWordWrap(width))
	default:
		renderer, err = glamour.NewTermRenderer(glamour.WithAutoStyle(), glamour.WithWordWrap(width))
	}

	if err != nil {
		// Fallback to basic renderer
		renderer, _ = glamour.NewTermRenderer()
	}

	return &MarkdownRenderer{
		renderer:          renderer,
		syntaxHighlighter: NewSyntaxHighlighter(theme),
		width:             width,
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

// SetWidth updates the renderer width
func (mr *MarkdownRenderer) SetWidth(width int) {
	mr.width = width
	// Recreate renderer with new width
	mr.renderer, _ = glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
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
		Render(" (" + url + ")")
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
