package theme

import "github.com/charmbracelet/lipgloss"

// ═══════════════════════════════════════════════════════════════════════════════
// DARK THEMES
// ═══════════════════════════════════════════════════════════════════════════════

// CatppuccinMocha returns the Catppuccin Mocha theme (default)
func CatppuccinMocha() *Theme {
	return &Theme{
		Name:        "catppuccin-mocha",
		Description: "Soothing pastel theme (dark)",
		Author:      "Catppuccin",
		Type:        "dark",

		Primary:   lipgloss.Color("#CBA6F7"), // Mauve
		Secondary: lipgloss.Color("#89B4FA"), // Blue
		Accent:    lipgloss.Color("#F5C2E7"), // Pink
		Success:   lipgloss.Color("#A6E3A1"), // Green
		Warning:   lipgloss.Color("#F9E2AF"), // Yellow
		Error:     lipgloss.Color("#F38BA8"), // Red
		Info:      lipgloss.Color("#89DCEB"), // Sky

		Text:       lipgloss.Color("#CDD6F4"), // Text
		TextMuted:  lipgloss.Color("#A6ADC8"), // Subtext
		TextDim:    lipgloss.Color("#6C7086"), // Overlay
		TextBright: lipgloss.Color("#F5E0DC"), // Rosewater

		Background:      lipgloss.Color("#1E1E2E"), // Base
		Surface:         lipgloss.Color("#313244"), // Surface0
		Border:          lipgloss.Color("#6C7086"), // Overlay0
		BorderHighlight: lipgloss.Color("#CBA6F7"), // Mauve

		User:      lipgloss.Color("#89B4FA"), // Blue
		Assistant: lipgloss.Color("#CDD6F4"), // Text
		System:    lipgloss.Color("#6C7086"), // Overlay
		Tool:      lipgloss.Color("#F9E2AF"), // Yellow

		SyntaxTheme:   "monokai",
		MarkdownTheme: "dark",
	}
}

// Dracula returns the Dracula theme
func Dracula() *Theme {
	return &Theme{
		Name:        "dracula",
		Description: "Dark theme with vibrant colors",
		Author:      "Dracula",
		Type:        "dark",

		Primary:   lipgloss.Color("#BD93F9"), // Purple
		Secondary: lipgloss.Color("#8BE9FD"), // Cyan
		Accent:    lipgloss.Color("#FF79C6"), // Pink
		Success:   lipgloss.Color("#50FA7B"), // Green
		Warning:   lipgloss.Color("#F1FA8C"), // Yellow
		Error:     lipgloss.Color("#FF5555"), // Red
		Info:      lipgloss.Color("#8BE9FD"), // Cyan

		Text:       lipgloss.Color("#F8F8F2"), // Foreground
		TextMuted:  lipgloss.Color("#6272A4"), // Comment
		TextDim:    lipgloss.Color("#44475A"), // Current line
		TextBright: lipgloss.Color("#FFFFFF"), // White

		Background:      lipgloss.Color("#282A36"), // Background
		Surface:         lipgloss.Color("#44475A"), // Selection
		Border:          lipgloss.Color("#6272A4"), // Comment
		BorderHighlight: lipgloss.Color("#BD93F9"), // Purple

		User:      lipgloss.Color("#8BE9FD"), // Cyan
		Assistant: lipgloss.Color("#F8F8F2"), // Foreground
		System:    lipgloss.Color("#6272A4"), // Comment
		Tool:      lipgloss.Color("#F1FA8C"), // Yellow

		SyntaxTheme:   "dracula",
		MarkdownTheme: "dracula",
	}
}

// TokyoNight returns the Tokyo Night theme
func TokyoNight() *Theme {
	return &Theme{
		Name:        "tokyo-night",
		Description: "A clean, dark theme inspired by Tokyo",
		Author:      "Tokyo Night",
		Type:        "dark",

		Primary:   lipgloss.Color("#BB9AF7"), // Purple
		Secondary: lipgloss.Color("#7AA2F7"), // Blue
		Accent:    lipgloss.Color("#F7768E"), // Red
		Success:   lipgloss.Color("#9ECE6A"), // Green
		Warning:   lipgloss.Color("#E0AF68"), // Yellow
		Error:     lipgloss.Color("#F7768E"), // Red
		Info:      lipgloss.Color("#7DCFFF"), // Cyan

		Text:       lipgloss.Color("#C0CAF5"), // Foreground
		TextMuted:  lipgloss.Color("#565F89"), // Comment
		TextDim:    lipgloss.Color("#414868"), // Dark
		TextBright: lipgloss.Color("#FFFFFF"), // White

		Background:      lipgloss.Color("#1A1B26"), // Background
		Surface:         lipgloss.Color("#24283B"), // Black
		Border:          lipgloss.Color("#414868"), // Dark
		BorderHighlight: lipgloss.Color("#BB9AF7"), // Purple

		User:      lipgloss.Color("#7AA2F7"), // Blue
		Assistant: lipgloss.Color("#C0CAF5"), // Foreground
		System:    lipgloss.Color("#565F89"), // Comment
		Tool:      lipgloss.Color("#E0AF68"), // Yellow

		SyntaxTheme:   "monokai",
		MarkdownTheme: "dark",
	}
}

// Nord returns the Nord theme
func Nord() *Theme {
	return &Theme{
		Name:        "nord",
		Description: "Arctic, north-bluish color palette",
		Author:      "Nord",
		Type:        "dark",

		Primary:   lipgloss.Color("#88C0D0"), // Frost 2
		Secondary: lipgloss.Color("#81A1C1"), // Frost 3
		Accent:    lipgloss.Color("#B48EAD"), // Aurora 4
		Success:   lipgloss.Color("#A3BE8C"), // Aurora 2
		Warning:   lipgloss.Color("#EBCB8B"), // Aurora 1
		Error:     lipgloss.Color("#BF616A"), // Aurora 0
		Info:      lipgloss.Color("#8FBCBB"), // Frost 1

		Text:       lipgloss.Color("#ECEFF4"), // Snow 2
		TextMuted:  lipgloss.Color("#D8DEE9"), // Snow 1
		TextDim:    lipgloss.Color("#4C566A"), // Polar 3
		TextBright: lipgloss.Color("#ECEFF4"), // Snow 2

		Background:      lipgloss.Color("#2E3440"), // Polar 0
		Surface:         lipgloss.Color("#3B4252"), // Polar 1
		Border:          lipgloss.Color("#4C566A"), // Polar 3
		BorderHighlight: lipgloss.Color("#88C0D0"), // Frost 2

		User:      lipgloss.Color("#81A1C1"), // Frost 3
		Assistant: lipgloss.Color("#ECEFF4"), // Snow 2
		System:    lipgloss.Color("#4C566A"), // Polar 3
		Tool:      lipgloss.Color("#EBCB8B"), // Aurora 1

		SyntaxTheme:   "nord",
		MarkdownTheme: "dark",
	}
}

// Gruvbox returns the Gruvbox theme
func Gruvbox() *Theme {
	return &Theme{
		Name:        "gruvbox",
		Description: "Retro groove color scheme",
		Author:      "Gruvbox",
		Type:        "dark",

		Primary:   lipgloss.Color("#D3869B"), // Purple
		Secondary: lipgloss.Color("#83A598"), // Blue
		Accent:    lipgloss.Color("#FE8019"), // Orange
		Success:   lipgloss.Color("#B8BB26"), // Green
		Warning:   lipgloss.Color("#FABD2F"), // Yellow
		Error:     lipgloss.Color("#FB4934"), // Red
		Info:      lipgloss.Color("#8EC07C"), // Aqua

		Text:       lipgloss.Color("#EBDBB2"), // FG
		TextMuted:  lipgloss.Color("#A89984"), // FG4
		TextDim:    lipgloss.Color("#665C54"), // BG3
		TextBright: lipgloss.Color("#FBF1C7"), // FG0

		Background:      lipgloss.Color("#282828"), // BG
		Surface:         lipgloss.Color("#3C3836"), // BG1
		Border:          lipgloss.Color("#665C54"), // BG3
		BorderHighlight: lipgloss.Color("#D3869B"), // Purple

		User:      lipgloss.Color("#83A598"), // Blue
		Assistant: lipgloss.Color("#EBDBB2"), // FG
		System:    lipgloss.Color("#665C54"), // BG3
		Tool:      lipgloss.Color("#FABD2F"), // Yellow

		SyntaxTheme:   "monokai",
		MarkdownTheme: "dark",
	}
}

// OneDark returns the One Dark theme
func OneDark() *Theme {
	return &Theme{
		Name:        "one-dark",
		Description: "Atom's iconic One Dark theme",
		Author:      "Atom",
		Type:        "dark",

		Primary:   lipgloss.Color("#C678DD"), // Purple
		Secondary: lipgloss.Color("#61AFEF"), // Blue
		Accent:    lipgloss.Color("#E06C75"), // Red
		Success:   lipgloss.Color("#98C379"), // Green
		Warning:   lipgloss.Color("#E5C07B"), // Yellow
		Error:     lipgloss.Color("#E06C75"), // Red
		Info:      lipgloss.Color("#56B6C2"), // Cyan

		Text:       lipgloss.Color("#ABB2BF"), // Foreground
		TextMuted:  lipgloss.Color("#5C6370"), // Comment
		TextDim:    lipgloss.Color("#4B5263"), // Gutter
		TextBright: lipgloss.Color("#FFFFFF"), // White

		Background:      lipgloss.Color("#282C34"), // Background
		Surface:         lipgloss.Color("#2C323C"), // Selection
		Border:          lipgloss.Color("#4B5263"), // Gutter
		BorderHighlight: lipgloss.Color("#C678DD"), // Purple

		User:      lipgloss.Color("#61AFEF"), // Blue
		Assistant: lipgloss.Color("#ABB2BF"), // Foreground
		System:    lipgloss.Color("#5C6370"), // Comment
		Tool:      lipgloss.Color("#E5C07B"), // Yellow

		SyntaxTheme:   "monokai",
		MarkdownTheme: "dark",
	}
}

// Monokai returns the Monokai theme
func Monokai() *Theme {
	return &Theme{
		Name:        "monokai",
		Description: "Classic Monokai color scheme",
		Author:      "Monokai",
		Type:        "dark",

		Primary:   lipgloss.Color("#AE81FF"), // Purple
		Secondary: lipgloss.Color("#66D9EF"), // Cyan
		Accent:    lipgloss.Color("#F92672"), // Pink
		Success:   lipgloss.Color("#A6E22E"), // Green
		Warning:   lipgloss.Color("#E6DB74"), // Yellow
		Error:     lipgloss.Color("#F92672"), // Pink
		Info:      lipgloss.Color("#66D9EF"), // Cyan

		Text:       lipgloss.Color("#F8F8F2"), // Foreground
		TextMuted:  lipgloss.Color("#75715E"), // Comment
		TextDim:    lipgloss.Color("#49483E"), // Selection
		TextBright: lipgloss.Color("#FFFFFF"), // White

		Background:      lipgloss.Color("#272822"), // Background
		Surface:         lipgloss.Color("#3E3D32"), // Line highlight
		Border:          lipgloss.Color("#75715E"), // Comment
		BorderHighlight: lipgloss.Color("#AE81FF"), // Purple

		User:      lipgloss.Color("#66D9EF"), // Cyan
		Assistant: lipgloss.Color("#F8F8F2"), // Foreground
		System:    lipgloss.Color("#75715E"), // Comment
		Tool:      lipgloss.Color("#E6DB74"), // Yellow

		SyntaxTheme:   "monokai",
		MarkdownTheme: "dark",
	}
}

// SolarizedDark returns the Solarized Dark theme
func SolarizedDark() *Theme {
	return &Theme{
		Name:        "solarized-dark",
		Description: "Precision colors for machines and people",
		Author:      "Solarized",
		Type:        "dark",

		Primary:   lipgloss.Color("#268BD2"), // Blue
		Secondary: lipgloss.Color("#2AA198"), // Cyan
		Accent:    lipgloss.Color("#D33682"), // Magenta
		Success:   lipgloss.Color("#859900"), // Green
		Warning:   lipgloss.Color("#B58900"), // Yellow
		Error:     lipgloss.Color("#DC322F"), // Red
		Info:      lipgloss.Color("#2AA198"), // Cyan

		Text:       lipgloss.Color("#839496"), // Base0
		TextMuted:  lipgloss.Color("#586E75"), // Base01
		TextDim:    lipgloss.Color("#073642"), // Base02
		TextBright: lipgloss.Color("#EEE8D5"), // Base2

		Background:      lipgloss.Color("#002B36"), // Base03
		Surface:         lipgloss.Color("#073642"), // Base02
		Border:          lipgloss.Color("#586E75"), // Base01
		BorderHighlight: lipgloss.Color("#268BD2"), // Blue

		User:      lipgloss.Color("#268BD2"), // Blue
		Assistant: lipgloss.Color("#839496"), // Base0
		System:    lipgloss.Color("#586E75"), // Base01
		Tool:      lipgloss.Color("#B58900"), // Yellow

		SyntaxTheme:   "solarized-dark",
		MarkdownTheme: "dark",
	}
}

// MaterialDark returns the Material Dark theme
func MaterialDark() *Theme {
	return &Theme{
		Name:        "material-dark",
		Description: "Material Design dark theme",
		Author:      "Material",
		Type:        "dark",

		Primary:   lipgloss.Color("#C792EA"), // Purple
		Secondary: lipgloss.Color("#82AAFF"), // Blue
		Accent:    lipgloss.Color("#F07178"), // Red
		Success:   lipgloss.Color("#C3E88D"), // Green
		Warning:   lipgloss.Color("#FFCB6B"), // Yellow
		Error:     lipgloss.Color("#F07178"), // Red
		Info:      lipgloss.Color("#89DDFF"), // Cyan

		Text:       lipgloss.Color("#EEFFFF"), // Foreground
		TextMuted:  lipgloss.Color("#546E7A"), // Comment
		TextDim:    lipgloss.Color("#37474F"), // Selection
		TextBright: lipgloss.Color("#FFFFFF"), // White

		Background:      lipgloss.Color("#263238"), // Background
		Surface:         lipgloss.Color("#2E3C43"), // Selection
		Border:          lipgloss.Color("#546E7A"), // Comment
		BorderHighlight: lipgloss.Color("#C792EA"), // Purple

		User:      lipgloss.Color("#82AAFF"), // Blue
		Assistant: lipgloss.Color("#EEFFFF"), // Foreground
		System:    lipgloss.Color("#546E7A"), // Comment
		Tool:      lipgloss.Color("#FFCB6B"), // Yellow

		SyntaxTheme:   "monokai",
		MarkdownTheme: "dark",
	}
}

// NightOwl returns the Night Owl theme
func NightOwl() *Theme {
	return &Theme{
		Name:        "night-owl",
		Description: "A theme for the night owls out there",
		Author:      "Night Owl",
		Type:        "dark",

		Primary:   lipgloss.Color("#C792EA"), // Purple
		Secondary: lipgloss.Color("#82AAFF"), // Blue
		Accent:    lipgloss.Color("#F78C6C"), // Orange
		Success:   lipgloss.Color("#ADDB67"), // Green
		Warning:   lipgloss.Color("#ECC48D"), // Yellow
		Error:     lipgloss.Color("#EF5350"), // Red
		Info:      lipgloss.Color("#80CBC4"), // Cyan

		Text:       lipgloss.Color("#D6DEEB"), // Foreground
		TextMuted:  lipgloss.Color("#637777"), // Comment
		TextDim:    lipgloss.Color("#1D3B53"), // Selection
		TextBright: lipgloss.Color("#FFFFFF"), // White

		Background:      lipgloss.Color("#011627"), // Background
		Surface:         lipgloss.Color("#0B2942"), // Line highlight
		Border:          lipgloss.Color("#1D3B53"), // Selection
		BorderHighlight: lipgloss.Color("#C792EA"), // Purple

		User:      lipgloss.Color("#82AAFF"), // Blue
		Assistant: lipgloss.Color("#D6DEEB"), // Foreground
		System:    lipgloss.Color("#637777"), // Comment
		Tool:      lipgloss.Color("#ECC48D"), // Yellow

		SyntaxTheme:   "monokai",
		MarkdownTheme: "dark",
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// LIGHT THEMES
// ═══════════════════════════════════════════════════════════════════════════════

// CatppuccinLatte returns the Catppuccin Latte theme
func CatppuccinLatte() *Theme {
	return &Theme{
		Name:        "catppuccin-latte",
		Description: "Soothing pastel theme (light)",
		Author:      "Catppuccin",
		Type:        "light",

		Primary:   lipgloss.Color("#8839EF"), // Mauve
		Secondary: lipgloss.Color("#1E66F5"), // Blue
		Accent:    lipgloss.Color("#EA76CB"), // Pink
		Success:   lipgloss.Color("#40A02B"), // Green
		Warning:   lipgloss.Color("#DF8E1D"), // Yellow
		Error:     lipgloss.Color("#D20F39"), // Red
		Info:      lipgloss.Color("#04A5E5"), // Sky

		Text:       lipgloss.Color("#4C4F69"), // Text
		TextMuted:  lipgloss.Color("#6C6F85"), // Subtext
		TextDim:    lipgloss.Color("#9CA0B0"), // Overlay
		TextBright: lipgloss.Color("#DC8A78"), // Rosewater

		Background:      lipgloss.Color("#EFF1F5"), // Base
		Surface:         lipgloss.Color("#E6E9EF"), // Mantle
		Border:          lipgloss.Color("#ACB0BE"), // Overlay0
		BorderHighlight: lipgloss.Color("#8839EF"), // Mauve

		User:      lipgloss.Color("#1E66F5"), // Blue
		Assistant: lipgloss.Color("#4C4F69"), // Text
		System:    lipgloss.Color("#9CA0B0"), // Overlay
		Tool:      lipgloss.Color("#DF8E1D"), // Yellow

		SyntaxTheme:   "github",
		MarkdownTheme: "light",
	}
}

// SolarizedLight returns the Solarized Light theme
func SolarizedLight() *Theme {
	return &Theme{
		Name:        "solarized-light",
		Description: "Precision colors for machines and people",
		Author:      "Solarized",
		Type:        "light",

		Primary:   lipgloss.Color("#268BD2"), // Blue
		Secondary: lipgloss.Color("#2AA198"), // Cyan
		Accent:    lipgloss.Color("#D33682"), // Magenta
		Success:   lipgloss.Color("#859900"), // Green
		Warning:   lipgloss.Color("#B58900"), // Yellow
		Error:     lipgloss.Color("#DC322F"), // Red
		Info:      lipgloss.Color("#2AA198"), // Cyan

		Text:       lipgloss.Color("#657B83"), // Base00
		TextMuted:  lipgloss.Color("#93A1A1"), // Base1
		TextDim:    lipgloss.Color("#EEE8D5"), // Base2
		TextBright: lipgloss.Color("#002B36"), // Base03

		Background:      lipgloss.Color("#FDF6E3"), // Base3
		Surface:         lipgloss.Color("#EEE8D5"), // Base2
		Border:          lipgloss.Color("#93A1A1"), // Base1
		BorderHighlight: lipgloss.Color("#268BD2"), // Blue

		User:      lipgloss.Color("#268BD2"), // Blue
		Assistant: lipgloss.Color("#657B83"), // Base00
		System:    lipgloss.Color("#93A1A1"), // Base1
		Tool:      lipgloss.Color("#B58900"), // Yellow

		SyntaxTheme:   "solarized-light",
		MarkdownTheme: "light",
	}
}

// GithubLight returns the GitHub Light theme
func GithubLight() *Theme {
	return &Theme{
		Name:        "github-light",
		Description: "GitHub's light color scheme",
		Author:      "GitHub",
		Type:        "light",

		Primary:   lipgloss.Color("#6F42C1"), // Purple
		Secondary: lipgloss.Color("#0366D6"), // Blue
		Accent:    lipgloss.Color("#E36209"), // Orange
		Success:   lipgloss.Color("#22863A"), // Green
		Warning:   lipgloss.Color("#B08800"), // Yellow
		Error:     lipgloss.Color("#D73A49"), // Red
		Info:      lipgloss.Color("#0366D6"), // Blue

		Text:       lipgloss.Color("#24292E"), // Text
		TextMuted:  lipgloss.Color("#6A737D"), // Gray
		TextDim:    lipgloss.Color("#D1D5DA"), // Border
		TextBright: lipgloss.Color("#000000"), // Black

		Background:      lipgloss.Color("#FFFFFF"), // White
		Surface:         lipgloss.Color("#F6F8FA"), // Gray light
		Border:          lipgloss.Color("#E1E4E8"), // Border
		BorderHighlight: lipgloss.Color("#6F42C1"), // Purple

		User:      lipgloss.Color("#0366D6"), // Blue
		Assistant: lipgloss.Color("#24292E"), // Text
		System:    lipgloss.Color("#6A737D"), // Gray
		Tool:      lipgloss.Color("#B08800"), // Yellow

		SyntaxTheme:   "github",
		MarkdownTheme: "light",
	}
}

// MaterialLight returns the Material Light theme
func MaterialLight() *Theme {
	return &Theme{
		Name:        "material-light",
		Description: "Material Design light theme",
		Author:      "Material",
		Type:        "light",

		Primary:   lipgloss.Color("#7C4DFF"), // Deep Purple
		Secondary: lipgloss.Color("#2196F3"), // Blue
		Accent:    lipgloss.Color("#FF5252"), // Red
		Success:   lipgloss.Color("#91B859"), // Green
		Warning:   lipgloss.Color("#F6A434"), // Orange
		Error:     lipgloss.Color("#E53935"), // Red
		Info:      lipgloss.Color("#39ADB5"), // Cyan

		Text:       lipgloss.Color("#272727"), // Text
		TextMuted:  lipgloss.Color("#90A4AE"), // Gray
		TextDim:    lipgloss.Color("#CCD7DA"), // Border
		TextBright: lipgloss.Color("#000000"), // Black

		Background:      lipgloss.Color("#FAFAFA"), // Background
		Surface:         lipgloss.Color("#FFFFFF"), // White
		Border:          lipgloss.Color("#CCD7DA"), // Border
		BorderHighlight: lipgloss.Color("#7C4DFF"), // Deep Purple

		User:      lipgloss.Color("#2196F3"), // Blue
		Assistant: lipgloss.Color("#272727"), // Text
		System:    lipgloss.Color("#90A4AE"), // Gray
		Tool:      lipgloss.Color("#F6A434"), // Orange

		SyntaxTheme:   "github",
		MarkdownTheme: "light",
	}
}

// OneLight returns the One Light theme
func OneLight() *Theme {
	return &Theme{
		Name:        "one-light",
		Description: "Atom's iconic One Light theme",
		Author:      "Atom",
		Type:        "light",

		Primary:   lipgloss.Color("#A626A4"), // Purple
		Secondary: lipgloss.Color("#4078F2"), // Blue
		Accent:    lipgloss.Color("#E45649"), // Red
		Success:   lipgloss.Color("#50A14F"), // Green
		Warning:   lipgloss.Color("#C18401"), // Yellow
		Error:     lipgloss.Color("#E45649"), // Red
		Info:      lipgloss.Color("#0184BC"), // Cyan

		Text:       lipgloss.Color("#383A42"), // Foreground
		TextMuted:  lipgloss.Color("#A0A1A7"), // Comment
		TextDim:    lipgloss.Color("#E5E5E6"), // Gutter
		TextBright: lipgloss.Color("#000000"), // Black

		Background:      lipgloss.Color("#FAFAFA"), // Background
		Surface:         lipgloss.Color("#FFFFFF"), // White
		Border:          lipgloss.Color("#E5E5E6"), // Gutter
		BorderHighlight: lipgloss.Color("#A626A4"), // Purple

		User:      lipgloss.Color("#4078F2"), // Blue
		Assistant: lipgloss.Color("#383A42"), // Foreground
		System:    lipgloss.Color("#A0A1A7"), // Comment
		Tool:      lipgloss.Color("#C18401"), // Yellow

		SyntaxTheme:   "github",
		MarkdownTheme: "light",
	}
}
