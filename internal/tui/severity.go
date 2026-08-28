package tui

import "github.com/charmbracelet/lipgloss"

// Severity is dbwatch's consistent status level, used everywhere a metric
// or event needs a state. Symbols are shown alongside color so the meaning
// survives a monochrome terminal.
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityOK
	SeverityWarning
	SeverityCritical
	SeverityError
)

func (s Severity) Symbol() string {
	switch s {
	case SeverityOK:
		return "✓"
	case SeverityWarning:
		return "⚠"
	case SeverityCritical:
		return "▲"
	case SeverityError:
		return "✕"
	default:
		return "●"
	}
}

func (s Severity) Label() string {
	switch s {
	case SeverityOK:
		return "healthy"
	case SeverityWarning:
		return "warning"
	case SeverityCritical:
		return "critical"
	case SeverityError:
		return "error"
	default:
		return "info"
	}
}

func (s Severity) Style() lipgloss.Style {
	switch s {
	case SeverityOK:
		return okStyle
	case SeverityWarning:
		return warnStyle
	case SeverityCritical, SeverityError:
		return critStyle
	default:
		return dimStyle
	}
}

// Render formats "<symbol> text" in the severity's color.
func (s Severity) Render(text string) string {
	return s.Style().Render(s.Symbol() + " " + text)
}
