package tui

import "github.com/charmbracelet/lipgloss"

var (
	brandColor = lipgloss.Color("39")
	dimColor   = lipgloss.Color("241")
	borderDim  = lipgloss.Color("240")

	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(brandColor)
	headingStyle  = lipgloss.NewStyle().Bold(true).Foreground(brandColor)
	labelStyle    = lipgloss.NewStyle().Bold(true)
	dimStyle      = lipgloss.NewStyle().Foreground(dimColor)
	okStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	warnStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	critStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	errStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	liveStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	readModeStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	flashStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))

	boxStyle        = lipgloss.NewStyle().Padding(0, 1).Border(lipgloss.RoundedBorder()).BorderForeground(borderDim)
	boxFocusedStyle = boxStyle.BorderForeground(brandColor)

	cardTitleStyle = lipgloss.NewStyle().Bold(true)

	selectedRowStyle = lipgloss.NewStyle().Bold(true).Foreground(brandColor)

	statusBarStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Background(lipgloss.Color("236")).Padding(0, 1)
	keyStyle       = lipgloss.NewStyle().Bold(true).Foreground(brandColor)

	helpTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(brandColor)
)

func boxStyleFor(focused bool) lipgloss.Style {
	if focused {
		return boxFocusedStyle
	}
	return boxStyle
}
