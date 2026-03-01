package theme

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Theme represents a color scheme for the TUI
type Theme struct {
	Name        string
	Description string
	Author      string
	Type        string // "dark" or "light"

	// Core colors
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Accent    lipgloss.Color
	Success   lipgloss.Color
	Warning   lipgloss.Color
	Error     lipgloss.Color
	Info      lipgloss.Color

	// Text colors
	Text       lipgloss.Color
	TextMuted  lipgloss.Color
	TextDim    lipgloss.Color
	TextBright lipgloss.Color

	// UI colors
	Background      lipgloss.Color
	Surface         lipgloss.Color
	Border          lipgloss.Color
	BorderHighlight lipgloss.Color

	// Semantic colors
	User      lipgloss.Color
	Assistant lipgloss.Color
	System    lipgloss.Color
	Tool      lipgloss.Color

	// Syntax highlighting theme name
	SyntaxTheme string

	// Markdown theme name
	MarkdownTheme string
}

// Registry holds all available themes
type Registry struct {
	themes  map[string]*Theme
	current string
}

// NewRegistry creates a new theme registry with builtin themes
func NewRegistry() *Registry {
	r := &Registry{
		themes:  make(map[string]*Theme),
		current: "catppuccin-mocha",
	}

	// Register all builtin themes
	r.registerBuiltinThemes()

	return r
}

// Get returns a theme by name
func (r *Registry) Get(name string) (*Theme, error) {
	theme, ok := r.themes[name]
	if !ok {
		return nil, fmt.Errorf("theme not found: %s", name)
	}
	return theme, nil
}

// Current returns the currently active theme
func (r *Registry) Current() *Theme {
	theme, _ := r.Get(r.current)
	return theme
}

// SetCurrent sets the current theme
func (r *Registry) SetCurrent(name string) error {
	_, ok := r.themes[name]
	if !ok {
		return fmt.Errorf("theme not found: %s", name)
	}
	r.current = name
	return nil
}

// List returns all available theme names
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.themes))
	for name := range r.themes {
		names = append(names, name)
	}
	return names
}

// ListByType returns themes filtered by type (dark/light)
func (r *Registry) ListByType(themeType string) []string {
	names := make([]string, 0)
	for name, theme := range r.themes {
		if theme.Type == themeType {
			names = append(names, name)
		}
	}
	return names
}

// Register registers a custom theme
func (r *Registry) Register(theme *Theme) {
	r.themes[theme.Name] = theme
}

// registerBuiltinThemes registers all builtin themes
func (r *Registry) registerBuiltinThemes() {
	// Dark themes
	r.Register(CatppuccinMocha())
	r.Register(Dracula())
	r.Register(TokyoNight())
	r.Register(Nord())
	r.Register(Gruvbox())
	r.Register(OneDark())
	r.Register(Monokai())
	r.Register(SolarizedDark())
	r.Register(MaterialDark())
	r.Register(NightOwl())

	// Light themes
	r.Register(CatppuccinLatte())
	r.Register(SolarizedLight())
	r.Register(GithubLight())
	r.Register(MaterialLight())
	r.Register(OneLight())

	// Extra themes (from builtin_extra.go)
	r.Register(Aura())
	r.Register(Ayu())
	r.Register(CarbonFox())
	r.Register(CatppuccinFrappe())
	r.Register(CatppuccinMacchiato())
	r.Register(Cobalt2())
	r.Register(Everforest())
	r.Register(Flexoki())
	r.Register(Kanagawa())
	r.Register(Matrix())
	r.Register(OpenCodeTheme())
	r.Register(Palenight())
	r.Register(RosePine())
	r.Register(Synthwave84())
	r.Register(TokyoNightStorm())
	r.Register(Vercel())
	r.Register(Vesper())
	r.Register(Zenburn())
	r.Register(OsakaJade())
	r.Register(Mercury())
}

// Default returns the default theme (Catppuccin Mocha)
func Default() *Theme {
	return CatppuccinMocha()
}
