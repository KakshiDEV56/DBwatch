package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Styles derived from theme. Borders are neutral by design (see
// renderTitledBox) -- color lives in status symbols and values, not in
// bright box outlines. That single change is what separates a "web
// dashboard with rounded cards" look from a native terminal console.
var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(theme.Foreground)
	headingStyle = lipgloss.NewStyle().Bold(true).Foreground(theme.Foreground)
	labelStyle   = lipgloss.NewStyle().Bold(true).Foreground(theme.Secondary)
	dimStyle     = lipgloss.NewStyle().Foreground(theme.Muted)

	okStyle   = lipgloss.NewStyle().Bold(true).Foreground(theme.Healthy)
	infoStyle = lipgloss.NewStyle().Bold(true).Foreground(theme.Info)
	warnStyle = lipgloss.NewStyle().Bold(true).Foreground(theme.Warning)
	critStyle = lipgloss.NewStyle().Bold(true).Foreground(theme.Critical)
	errStyle  = lipgloss.NewStyle().Bold(true).Foreground(theme.Critical)

	liveStyle     = lipgloss.NewStyle().Bold(true).Foreground(theme.Healthy)
	readModeStyle = lipgloss.NewStyle().Bold(true).Foreground(theme.Warning)
	flashStyle    = lipgloss.NewStyle().Bold(true).Foreground(theme.Healthy)
	matchStyle    = lipgloss.NewStyle().Bold(true).Foreground(theme.Warning)
	queryStyle    = lipgloss.NewStyle().Foreground(theme.Blue)

	selectedRowStyle = lipgloss.NewStyle().Bold(true).Foreground(theme.Background).Background(theme.Blue)

	statusBarStyle = lipgloss.NewStyle().Foreground(theme.Secondary).Background(theme.BackgroundDark).Padding(0, 1)
	keyStyle       = lipgloss.NewStyle().Bold(true).Foreground(theme.Info)
)

func borderColorFor(focused bool) lipgloss.Color {
	if focused {
		return theme.Blue
	}
	return theme.Border
}

// renderTitledBox draws a btop-style box: the title is spliced directly
// into the top border ("┌─ Connections ────┐") instead of sitting as a
// first line of the body. The border itself is always neutral
// (borderColorFor) -- state is communicated by the symbols and values
// inside, never by lighting up the whole outline in a status color.
// totalWidth is the box's full rendered width, borders included.
func renderTitledBox(title, body string, totalWidth int, focused bool, contentHeight int) string {
	color := borderColorFor(focused)
	borderStyle := lipgloss.NewStyle().Foreground(color)

	// "┌─ " + title + " " + dashes + "┐" -- title must be clipped to fit
	// this decoration or the top border renders wider than the box body
	// below it, which on a narrow terminal broke the border entirely.
	maxTitleWidth := max(0, totalWidth-5)
	title = truncate(title, maxTitleWidth)
	titleStyled := lipgloss.NewStyle().Bold(true).Foreground(theme.Secondary).Render(title)

	dashes := max(0, totalWidth-lipgloss.Width(title)-5)
	top := borderStyle.Render("┌─ ") + titleStyled + borderStyle.Render(" "+strings.Repeat("─", dashes)+"┐")

	style := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderTop(false).
		BorderForeground(color).
		Padding(0, 1).
		// lipgloss's Width() already accounts for horizontal padding, so
		// this only needs to subtract the border (left+right) -- the
		// actual TEXT budget callers should wrap/truncate to is
		// totalWidth-4 (border + padding), one layer narrower than this.
		Width(totalWidth - 2)
	if contentHeight > 0 {
		style = style.Height(contentHeight)
	}

	return top + "\n" + style.Render(body)
}

// bar renders a compact ASCII meter, e.g. "███████░░░" at the given
// width, filled to pct (0-100).
func bar(pct float64, width int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := int(pct/100*float64(width) + 0.5)
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

// sparkline renders a compact history trend using block characters. It
// only renders what's actually in samples -- callers must not pad with
// fabricated values.
func sparkline(samples []float64, width int) string {
	if len(samples) == 0 {
		return ""
	}
	blocks := []rune("▁▂▃▄▅▆▇█")
	// Use only the most recent `width` samples.
	if len(samples) > width {
		samples = samples[len(samples)-width:]
	}
	min, max := samples[0], samples[0]
	for _, v := range samples {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	span := max - min
	var b strings.Builder
	for _, v := range samples {
		idx := 0
		if span > 0 {
			idx = int((v - min) / span * float64(len(blocks)-1))
		}
		b.WriteRune(blocks[idx])
	}
	return b.String()
}
