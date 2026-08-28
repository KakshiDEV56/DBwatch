package tui

import (
	"fmt"
	"strings"
	"time"
)

// LogField is one labeled value in a log entry's detail view (PID, State,
// Duration, Client, Query, ...).
type LogField struct {
	Key   string
	Value string
}

// LogEntry is one line in a panel's log/event stream. Summary is what's
// shown in the scrolling list; Fields + Query back the detail view and the
// copy action.
type LogEntry struct {
	Time     time.Time
	Severity Severity
	Panel    PanelKind
	Summary  string
	Fields   []LogField
	Query    string
}

// MatchesSearch reports whether q (already lowercased) appears in the
// entry's summary, fields, or query text.
func (e LogEntry) MatchesSearch(q string) bool {
	if strings.Contains(strings.ToLower(e.Summary), q) {
		return true
	}
	if strings.Contains(strings.ToLower(e.Query), q) {
		return true
	}
	for _, f := range e.Fields {
		if strings.Contains(strings.ToLower(f.Value), q) {
			return true
		}
	}
	return false
}

// CopyText renders the entry as plain text suitable for pasting into
// Slack, an issue, or a terminal.
func (e LogEntry) CopyText() string {
	var b strings.Builder
	fmt.Fprintf(&b, "[%s] %s\n", e.Time.Format("15:04:05"), e.Summary)
	for _, f := range e.Fields {
		fmt.Fprintf(&b, "%s: %s\n", f.Key, f.Value)
	}
	if e.Query != "" {
		b.WriteString("\n" + e.Query + "\n")
	}
	return b.String()
}

// logBufferLimit caps how many events a single panel's log stream keeps in
// memory before dropping the oldest.
const logBufferLimit = 500

// LogBuffer is an append-only, capped event stream for one panel.
type LogBuffer struct {
	entries []LogEntry
}

func (b *LogBuffer) Append(e LogEntry) {
	b.entries = append(b.entries, e)
	if len(b.entries) > logBufferLimit {
		b.entries = b.entries[len(b.entries)-logBufferLimit:]
	}
}

func (b *LogBuffer) Entries() []LogEntry {
	return b.entries
}
