package tui

import "github.com/charmbracelet/lipgloss"

// Theme centralizes every color dbwatch uses. Nothing outside this file
// should reference a raw hex/ANSI color -- change the palette here and it
// propagates everywhere.
type Theme struct {
	Background     lipgloss.Color
	BackgroundDark lipgloss.Color
	Foreground     lipgloss.Color
	Secondary      lipgloss.Color
	Muted          lipgloss.Color

	Healthy  lipgloss.Color
	Info     lipgloss.Color
	Blue     lipgloss.Color
	Warning  lipgloss.Color
	Degraded lipgloss.Color
	Critical lipgloss.Color
	Magenta  lipgloss.Color

	Border    lipgloss.Color
	BorderDim lipgloss.Color
}

// gruvbox is dbwatch's palette -- a Gruvbox-inspired dark theme, the same
// visual language btop's default theme draws from.
var gruvbox = Theme{
	Background:     lipgloss.Color("#1d2021"),
	BackgroundDark: lipgloss.Color("#282828"),
	Foreground:     lipgloss.Color("#ebdbb2"),
	Secondary:      lipgloss.Color("#bdae93"),
	Muted:          lipgloss.Color("#928374"),

	Healthy:  lipgloss.Color("#b8bb26"),
	Info:     lipgloss.Color("#8ec07c"),
	Blue:     lipgloss.Color("#83a598"),
	Warning:  lipgloss.Color("#fabd2f"),
	Degraded: lipgloss.Color("#fe8019"),
	Critical: lipgloss.Color("#fb4934"),
	Magenta:  lipgloss.Color("#d3869b"),

	Border:    lipgloss.Color("#665c54"),
	BorderDim: lipgloss.Color("#504945"),
}

var theme = gruvbox
