package tui

import "testing"

func TestSeverityLabelAndSymbol(t *testing.T) {
	tests := []struct {
		sev    Severity
		label  string
		symbol string
	}{
		{SeverityInfo, "info", "●"},
		{SeverityOK, "healthy", "✓"},
		{SeverityWarning, "warning", "⚠"},
		{SeverityDegraded, "degraded", "▲"},
		{SeverityCritical, "critical", "✕"},
		{SeverityError, "offline", "○"},
	}
	for _, tt := range tests {
		if got := tt.sev.Label(); got != tt.label {
			t.Errorf("%v.Label() = %q, want %q", tt.sev, got, tt.label)
		}
		if got := tt.sev.Symbol(); got != tt.symbol {
			t.Errorf("%v.Symbol() = %q, want %q", tt.sev, got, tt.symbol)
		}
	}
}
