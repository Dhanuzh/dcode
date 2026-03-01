package theme

import "github.com/charmbracelet/lipgloss"

// Extra themes ported from opencode's JSON theme files.
// Each function follows the same pattern as builtin.go.

func Aura() *Theme {
	return &Theme{
		Name:            "aura",
		Description:     "Soft, dreamy dark theme",
		Type:            "dark",
		Primary:         lipgloss.Color("#a277ff"),
		Secondary:       lipgloss.Color("#f694ff"),
		Accent:          lipgloss.Color("#a277ff"),
		Success:         lipgloss.Color("#61ffca"),
		Warning:         lipgloss.Color("#ffca85"),
		Error:           lipgloss.Color("#ff6767"),
		Info:            lipgloss.Color("#a277ff"),
		Text:            lipgloss.Color("#edecee"),
		TextMuted:       lipgloss.Color("#6d6d6d"),
		TextDim:         lipgloss.Color("#3d3d3d"),
		TextBright:      lipgloss.Color("#ffffff"),
		Background:      lipgloss.Color("#0f0f0f"),
		Surface:         lipgloss.Color("#1a1a1a"),
		Border:          lipgloss.Color("#2d2d2d"),
		BorderHighlight: lipgloss.Color("#a277ff"),
		SyntaxTheme:     "monokai",
		MarkdownTheme:   "dark",
	}
}

func Ayu() *Theme {
	return &Theme{
		Name:            "ayu",
		Description:     "A simple theme with bright colors",
		Type:            "dark",
		Primary:         lipgloss.Color("#59C2FF"),
		Secondary:       lipgloss.Color("#D2A6FF"),
		Accent:          lipgloss.Color("#E6B450"),
		Success:         lipgloss.Color("#7FD962"),
		Warning:         lipgloss.Color("#E6B673"),
		Error:           lipgloss.Color("#D95757"),
		Info:            lipgloss.Color("#39BAE6"),
		Text:            lipgloss.Color("#BFBDB6"),
		TextMuted:       lipgloss.Color("#565B66"),
		TextDim:         lipgloss.Color("#3D424D"),
		TextBright:      lipgloss.Color("#F0EAD6"),
		Background:      lipgloss.Color("#0B0E14"),
		Surface:         lipgloss.Color("#13161F"),
		Border:          lipgloss.Color("#6C7380"),
		BorderHighlight: lipgloss.Color("#59C2FF"),
		SyntaxTheme:     "monokai",
		MarkdownTheme:   "dark",
	}
}

func CarbonFox() *Theme {
	return &Theme{
		Name:            "carbonfox",
		Description:     "A dark IBM Carbon Design inspired theme",
		Type:            "dark",
		Primary:         lipgloss.Color("#33b1ff"),
		Secondary:       lipgloss.Color("#78a9ff"),
		Accent:          lipgloss.Color("#ff7eb6"),
		Success:         lipgloss.Color("#25be6a"),
		Warning:         lipgloss.Color("#f1c21b"),
		Error:           lipgloss.Color("#ee5396"),
		Info:            lipgloss.Color("#78a9ff"),
		Text:            lipgloss.Color("#f2f4f8"),
		TextMuted:       lipgloss.Color("#7d848f"),
		TextDim:         lipgloss.Color("#303030"),
		TextBright:      lipgloss.Color("#ffffff"),
		Background:      lipgloss.Color("#161616"),
		Surface:         lipgloss.Color("#282828"),
		Border:          lipgloss.Color("#303030"),
		BorderHighlight: lipgloss.Color("#33b1ff"),
		SyntaxTheme:     "monokai",
		MarkdownTheme:   "dark",
	}
}

func CatppuccinFrappe() *Theme {
	return &Theme{
		Name:            "catppuccin-frappe",
		Description:     "Catppuccin Frappé — cool, soothing dark",
		Author:          "Catppuccin",
		Type:            "dark",
		Primary:         lipgloss.Color("#8da4e2"),
		Secondary:       lipgloss.Color("#ca9ee6"),
		Accent:          lipgloss.Color("#f4b8e4"),
		Success:         lipgloss.Color("#a6d189"),
		Warning:         lipgloss.Color("#e5c890"),
		Error:           lipgloss.Color("#e78284"),
		Info:            lipgloss.Color("#81c8be"),
		Text:            lipgloss.Color("#c6d0f5"),
		TextMuted:       lipgloss.Color("#b5bfe2"),
		TextDim:         lipgloss.Color("#51576d"),
		TextBright:      lipgloss.Color("#f2d5cf"),
		Background:      lipgloss.Color("#303446"),
		Surface:         lipgloss.Color("#414559"),
		Border:          lipgloss.Color("#414559"),
		BorderHighlight: lipgloss.Color("#8da4e2"),
		SyntaxTheme:     "monokai",
		MarkdownTheme:   "dark",
	}
}

func CatppuccinMacchiato() *Theme {
	return &Theme{
		Name:            "catppuccin-macchiato",
		Description:     "Catppuccin Macchiato — warm dark",
		Author:          "Catppuccin",
		Type:            "dark",
		Primary:         lipgloss.Color("#8aadf4"),
		Secondary:       lipgloss.Color("#c6a0f6"),
		Accent:          lipgloss.Color("#f5bde6"),
		Success:         lipgloss.Color("#a6da95"),
		Warning:         lipgloss.Color("#eed49f"),
		Error:           lipgloss.Color("#ed8796"),
		Info:            lipgloss.Color("#8bd5ca"),
		Text:            lipgloss.Color("#cad3f5"),
		TextMuted:       lipgloss.Color("#b8c0e0"),
		TextDim:         lipgloss.Color("#494d64"),
		TextBright:      lipgloss.Color("#f4dbd6"),
		Background:      lipgloss.Color("#24273a"),
		Surface:         lipgloss.Color("#363a4f"),
		Border:          lipgloss.Color("#363a4f"),
		BorderHighlight: lipgloss.Color("#8aadf4"),
		SyntaxTheme:     "monokai",
		MarkdownTheme:   "dark",
	}
}

func Cobalt2() *Theme {
	return &Theme{
		Name:            "cobalt2",
		Description:     "Cobalt 2 — vivid blue dark theme",
		Type:            "dark",
		Primary:         lipgloss.Color("#0088ff"),
		Secondary:       lipgloss.Color("#9a5feb"),
		Accent:          lipgloss.Color("#2affdf"),
		Success:         lipgloss.Color("#9eff80"),
		Warning:         lipgloss.Color("#ffc600"),
		Error:           lipgloss.Color("#ff0088"),
		Info:            lipgloss.Color("#ff9d00"),
		Text:            lipgloss.Color("#ffffff"),
		TextMuted:       lipgloss.Color("#adb7c9"),
		TextDim:         lipgloss.Color("#1f4662"),
		TextBright:      lipgloss.Color("#ffffff"),
		Background:      lipgloss.Color("#193549"),
		Surface:         lipgloss.Color("#1f4662"),
		Border:          lipgloss.Color("#1f4662"),
		BorderHighlight: lipgloss.Color("#0088ff"),
		SyntaxTheme:     "monokai",
		MarkdownTheme:   "dark",
	}
}

func Everforest() *Theme {
	return &Theme{
		Name:            "everforest",
		Description:     "Green-based low-contrast theme",
		Type:            "dark",
		Primary:         lipgloss.Color("#a7c080"),
		Secondary:       lipgloss.Color("#7fbbb3"),
		Accent:          lipgloss.Color("#d699b6"),
		Success:         lipgloss.Color("#a7c080"),
		Warning:         lipgloss.Color("#e69875"),
		Error:           lipgloss.Color("#e67e80"),
		Info:            lipgloss.Color("#83c092"),
		Text:            lipgloss.Color("#d3c6aa"),
		TextMuted:       lipgloss.Color("#7a8478"),
		TextDim:         lipgloss.Color("#4a5248"),
		TextBright:      lipgloss.Color("#e9e8d4"),
		Background:      lipgloss.Color("#2d353b"),
		Surface:         lipgloss.Color("#3d4841"),
		Border:          lipgloss.Color("#859289"),
		BorderHighlight: lipgloss.Color("#a7c080"),
		SyntaxTheme:     "monokai",
		MarkdownTheme:   "dark",
	}
}

func Flexoki() *Theme {
	return &Theme{
		Name:            "flexoki",
		Description:     "Flexoki — warm, inkish theme",
		Type:            "dark",
		Primary:         lipgloss.Color("#DA702C"),
		Secondary:       lipgloss.Color("#4385BE"),
		Accent:          lipgloss.Color("#8B7EC8"),
		Success:         lipgloss.Color("#879A39"),
		Warning:         lipgloss.Color("#DA702C"),
		Error:           lipgloss.Color("#D14D41"),
		Info:            lipgloss.Color("#3AA99F"),
		Text:            lipgloss.Color("#CECDC3"),
		TextMuted:       lipgloss.Color("#6F6E69"),
		TextDim:         lipgloss.Color("#302F2C"),
		TextBright:      lipgloss.Color("#F2F0E5"),
		Background:      lipgloss.Color("#100F0F"),
		Surface:         lipgloss.Color("#282726"),
		Border:          lipgloss.Color("#575653"),
		BorderHighlight: lipgloss.Color("#DA702C"),
		SyntaxTheme:     "monokai",
		MarkdownTheme:   "dark",
	}
}

func Kanagawa() *Theme {
	return &Theme{
		Name:            "kanagawa",
		Description:     "Inspired by the great wave of Kanagawa",
		Type:            "dark",
		Primary:         lipgloss.Color("#7E9CD8"),
		Secondary:       lipgloss.Color("#957FB8"),
		Accent:          lipgloss.Color("#D27E99"),
		Success:         lipgloss.Color("#98BB6C"),
		Warning:         lipgloss.Color("#D7A657"),
		Error:           lipgloss.Color("#E82424"),
		Info:            lipgloss.Color("#76946A"),
		Text:            lipgloss.Color("#DCD7BA"),
		TextMuted:       lipgloss.Color("#727169"),
		TextDim:         lipgloss.Color("#363646"),
		TextBright:      lipgloss.Color("#F9E7C0"),
		Background:      lipgloss.Color("#1F1F28"),
		Surface:         lipgloss.Color("#2A2A37"),
		Border:          lipgloss.Color("#54546D"),
		BorderHighlight: lipgloss.Color("#7E9CD8"),
		SyntaxTheme:     "monokai",
		MarkdownTheme:   "dark",
	}
}

func Matrix() *Theme {
	return &Theme{
		Name:            "matrix",
		Description:     "Enter the Matrix",
		Type:            "dark",
		Primary:         lipgloss.Color("#2eff6a"),
		Secondary:       lipgloss.Color("#00efff"),
		Accent:          lipgloss.Color("#c770ff"),
		Success:         lipgloss.Color("#62ff94"),
		Warning:         lipgloss.Color("#e6ff57"),
		Error:           lipgloss.Color("#ff4b4b"),
		Info:            lipgloss.Color("#30b3ff"),
		Text:            lipgloss.Color("#62ff94"),
		TextMuted:       lipgloss.Color("#8ca391"),
		TextDim:         lipgloss.Color("#1e2a1b"),
		TextBright:      lipgloss.Color("#aaffcc"),
		Background:      lipgloss.Color("#0a0e0a"),
		Surface:         lipgloss.Color("#0f1a10"),
		Border:          lipgloss.Color("#1e2a1b"),
		BorderHighlight: lipgloss.Color("#2eff6a"),
		SyntaxTheme:     "monokai",
		MarkdownTheme:   "dark",
	}
}

func OpenCodeTheme() *Theme {
	return &Theme{
		Name:            "opencode",
		Description:     "opencode's own signature theme",
		Type:            "dark",
		Primary:         lipgloss.Color("#fab283"),
		Secondary:       lipgloss.Color("#5c9cf5"),
		Accent:          lipgloss.Color("#9d7cd8"),
		Success:         lipgloss.Color("#7fd88f"),
		Warning:         lipgloss.Color("#f5a742"),
		Error:           lipgloss.Color("#e06c75"),
		Info:            lipgloss.Color("#56b6c2"),
		Text:            lipgloss.Color("#eeeeee"),
		TextMuted:       lipgloss.Color("#808080"),
		TextDim:         lipgloss.Color("#484848"),
		TextBright:      lipgloss.Color("#ffffff"),
		Background:      lipgloss.Color("#0a0a0a"),
		Surface:         lipgloss.Color("#151515"),
		Border:          lipgloss.Color("#484848"),
		BorderHighlight: lipgloss.Color("#fab283"),
		SyntaxTheme:     "monokai",
		MarkdownTheme:   "dark",
	}
}

func Palenight() *Theme {
	return &Theme{
		Name:            "palenight",
		Description:     "Material Palenight",
		Type:            "dark",
		Primary:         lipgloss.Color("#82aaff"),
		Secondary:       lipgloss.Color("#c792ea"),
		Accent:          lipgloss.Color("#89ddff"),
		Success:         lipgloss.Color("#c3e88d"),
		Warning:         lipgloss.Color("#ffcb6b"),
		Error:           lipgloss.Color("#f07178"),
		Info:            lipgloss.Color("#f78c6c"),
		Text:            lipgloss.Color("#a6accd"),
		TextMuted:       lipgloss.Color("#676e95"),
		TextDim:         lipgloss.Color("#32364a"),
		TextBright:      lipgloss.Color("#f0f0f0"),
		Background:      lipgloss.Color("#292d3e"),
		Surface:         lipgloss.Color("#32364a"),
		Border:          lipgloss.Color("#32364a"),
		BorderHighlight: lipgloss.Color("#82aaff"),
		SyntaxTheme:     "monokai",
		MarkdownTheme:   "dark",
	}
}

func RosePine() *Theme {
	return &Theme{
		Name:            "rose-pine",
		Description:     "All natural pine, faux fur and a bit of soho vibes",
		Type:            "dark",
		Primary:         lipgloss.Color("#9ccfd8"),
		Secondary:       lipgloss.Color("#c4a7e7"),
		Accent:          lipgloss.Color("#ebbcba"),
		Success:         lipgloss.Color("#31748f"),
		Warning:         lipgloss.Color("#f6c177"),
		Error:           lipgloss.Color("#eb6f92"),
		Info:            lipgloss.Color("#9ccfd8"),
		Text:            lipgloss.Color("#e0def4"),
		TextMuted:       lipgloss.Color("#6e6a86"),
		TextDim:         lipgloss.Color("#26233a"),
		TextBright:      lipgloss.Color("#f5e0dc"),
		Background:      lipgloss.Color("#191724"),
		Surface:         lipgloss.Color("#1f1d2e"),
		Border:          lipgloss.Color("#403d52"),
		BorderHighlight: lipgloss.Color("#9ccfd8"),
		SyntaxTheme:     "monokai",
		MarkdownTheme:   "dark",
	}
}

func Synthwave84() *Theme {
	return &Theme{
		Name:            "synthwave84",
		Description:     "A Synthwave-inspired theme",
		Type:            "dark",
		Primary:         lipgloss.Color("#36f9f6"),
		Secondary:       lipgloss.Color("#ff7edb"),
		Accent:          lipgloss.Color("#b084eb"),
		Success:         lipgloss.Color("#72f1b8"),
		Warning:         lipgloss.Color("#fede5d"),
		Error:           lipgloss.Color("#fe4450"),
		Info:            lipgloss.Color("#ff8b39"),
		Text:            lipgloss.Color("#ffffff"),
		TextMuted:       lipgloss.Color("#848bbd"),
		TextDim:         lipgloss.Color("#495495"),
		TextBright:      lipgloss.Color("#f0f0ff"),
		Background:      lipgloss.Color("#262335"),
		Surface:         lipgloss.Color("#2a2139"),
		Border:          lipgloss.Color("#495495"),
		BorderHighlight: lipgloss.Color("#36f9f6"),
		SyntaxTheme:     "monokai",
		MarkdownTheme:   "dark",
	}
}

func TokyoNightStorm() *Theme {
	return &Theme{
		Name:            "tokyonight-storm",
		Description:     "Tokyo Night Storm variant",
		Type:            "dark",
		Primary:         lipgloss.Color("#82aaff"),
		Secondary:       lipgloss.Color("#c099ff"),
		Accent:          lipgloss.Color("#ff966c"),
		Success:         lipgloss.Color("#c3e88d"),
		Warning:         lipgloss.Color("#ff966c"),
		Error:           lipgloss.Color("#ff757f"),
		Info:            lipgloss.Color("#82aaff"),
		Text:            lipgloss.Color("#c8d3f5"),
		TextMuted:       lipgloss.Color("#828bb8"),
		TextDim:         lipgloss.Color("#414868"),
		TextBright:      lipgloss.Color("#ffffff"),
		Background:      lipgloss.Color("#1a1b26"),
		Surface:         lipgloss.Color("#24283b"),
		Border:          lipgloss.Color("#737aa2"),
		BorderHighlight: lipgloss.Color("#82aaff"),
		SyntaxTheme:     "monokai",
		MarkdownTheme:   "dark",
	}
}

func Vercel() *Theme {
	return &Theme{
		Name:            "vercel",
		Description:     "Vercel's clean dark theme",
		Type:            "dark",
		Primary:         lipgloss.Color("#0070F3"),
		Secondary:       lipgloss.Color("#52A8FF"),
		Accent:          lipgloss.Color("#8E4EC6"),
		Success:         lipgloss.Color("#46A758"),
		Warning:         lipgloss.Color("#FFB224"),
		Error:           lipgloss.Color("#E5484D"),
		Info:            lipgloss.Color("#52A8FF"),
		Text:            lipgloss.Color("#EDEDED"),
		TextMuted:       lipgloss.Color("#878787"),
		TextDim:         lipgloss.Color("#1F1F1F"),
		TextBright:      lipgloss.Color("#FFFFFF"),
		Background:      lipgloss.Color("#000000"),
		Surface:         lipgloss.Color("#111111"),
		Border:          lipgloss.Color("#1F1F1F"),
		BorderHighlight: lipgloss.Color("#0070F3"),
		SyntaxTheme:     "monokai",
		MarkdownTheme:   "dark",
	}
}

func Vesper() *Theme {
	return &Theme{
		Name:            "vesper",
		Description:     "Warm, minimal dark theme",
		Type:            "dark",
		Primary:         lipgloss.Color("#FFC799"),
		Secondary:       lipgloss.Color("#99FFE4"),
		Accent:          lipgloss.Color("#FFC799"),
		Success:         lipgloss.Color("#99FFE4"),
		Warning:         lipgloss.Color("#FFC799"),
		Error:           lipgloss.Color("#FF8080"),
		Info:            lipgloss.Color("#FFC799"),
		Text:            lipgloss.Color("#FFFFFF"),
		TextMuted:       lipgloss.Color("#A0A0A0"),
		TextDim:         lipgloss.Color("#282828"),
		TextBright:      lipgloss.Color("#FFFFFF"),
		Background:      lipgloss.Color("#101010"),
		Surface:         lipgloss.Color("#1C1C1C"),
		Border:          lipgloss.Color("#282828"),
		BorderHighlight: lipgloss.Color("#FFC799"),
		SyntaxTheme:     "monokai",
		MarkdownTheme:   "dark",
	}
}

func Zenburn() *Theme {
	return &Theme{
		Name:            "zenburn",
		Description:     "Low contrast, easy on the eyes",
		Type:            "dark",
		Primary:         lipgloss.Color("#8cd0d3"),
		Secondary:       lipgloss.Color("#dc8cc3"),
		Accent:          lipgloss.Color("#93e0e3"),
		Success:         lipgloss.Color("#7f9f7f"),
		Warning:         lipgloss.Color("#f0dfaf"),
		Error:           lipgloss.Color("#cc9393"),
		Info:            lipgloss.Color("#dfaf8f"),
		Text:            lipgloss.Color("#dcdccc"),
		TextMuted:       lipgloss.Color("#9f9f9f"),
		TextDim:         lipgloss.Color("#4f4f4f"),
		TextBright:      lipgloss.Color("#f0f0f0"),
		Background:      lipgloss.Color("#3f3f3f"),
		Surface:         lipgloss.Color("#4f4f4f"),
		Border:          lipgloss.Color("#5f5f5f"),
		BorderHighlight: lipgloss.Color("#8cd0d3"),
		SyntaxTheme:     "monokai",
		MarkdownTheme:   "dark",
	}
}

func OsakaJade() *Theme {
	return &Theme{
		Name:            "osaka-jade",
		Description:     "Osaka Jade dark variant",
		Type:            "dark",
		Primary:         lipgloss.Color("#2DD5B7"),
		Secondary:       lipgloss.Color("#D2689C"),
		Accent:          lipgloss.Color("#549e6a"),
		Success:         lipgloss.Color("#549e6a"),
		Warning:         lipgloss.Color("#E5C736"),
		Error:           lipgloss.Color("#FF5345"),
		Info:            lipgloss.Color("#2DD5B7"),
		Text:            lipgloss.Color("#C1C497"),
		TextMuted:       lipgloss.Color("#53685B"),
		TextDim:         lipgloss.Color("#1f2e27"),
		TextBright:      lipgloss.Color("#eef0c5"),
		Background:      lipgloss.Color("#111c18"),
		Surface:         lipgloss.Color("#1a2b23"),
		Border:          lipgloss.Color("#3d4a44"),
		BorderHighlight: lipgloss.Color("#2DD5B7"),
		SyntaxTheme:     "monokai",
		MarkdownTheme:   "dark",
	}
}

func Mercury() *Theme {
	return &Theme{
		Name:            "mercury",
		Description:     "Mercury — cool-gray dark theme",
		Type:            "dark",
		Primary:         lipgloss.Color("#8da4f5"),
		Secondary:       lipgloss.Color("#a7b6f8"),
		Accent:          lipgloss.Color("#8da4f5"),
		Success:         lipgloss.Color("#77c599"),
		Warning:         lipgloss.Color("#fc9b6f"),
		Error:           lipgloss.Color("#fc92b4"),
		Info:            lipgloss.Color("#77becf"),
		Text:            lipgloss.Color("#dddde5"),
		TextMuted:       lipgloss.Color("#9d9da8"),
		TextDim:         lipgloss.Color("#2b2b38"),
		TextBright:      lipgloss.Color("#f0f0f8"),
		Background:      lipgloss.Color("#171721"),
		Surface:         lipgloss.Color("#22222e"),
		Border:          lipgloss.Color("#2f2f40"),
		BorderHighlight: lipgloss.Color("#8da4f5"),
		SyntaxTheme:     "monokai",
		MarkdownTheme:   "dark",
	}
}
