package tui

import "github.com/charmbracelet/lipgloss"

// Severity is dbwatch's single status system -- every metric, panel, and
// event maps onto exactly these six levels, each with one fixed
// symbol/color/label. Nothing in the UI invents its own ad hoc status
// color; everything goes through this type.
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityOK
	SeverityWarning
	SeverityDegraded
	SeverityCritical
	// SeverityError means the database itself is unreachable
	// ("offline") -- distinct from a warning/critical reading on a
	// database that's still responding.
	SeverityError
)

func (s Severity) Symbol() string {
	switch s {
	case SeverityOK:
		return "✓"
	case SeverityWarning:
		return "⚠"
	case SeverityDegraded:
		return "▲"
	case SeverityCritical:
		return "✕"
	case SeverityError:
		return "○"
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
	case SeverityDegraded:
		return "degraded"
	case SeverityCritical:
		return "critical"
	case SeverityError:
		return "offline"
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
	case SeverityDegraded:
		return lipgloss.NewStyle().Bold(true).Foreground(theme.Degraded)
	case SeverityCritical, SeverityError:
		return critStyle
	default:
		return infoStyle
	}
}

// Render formats "<symbol> text" in the severity's color.
func (s Severity) Render(text string) string {
	return s.Style().Render(s.Symbol() + " " + text)
}
