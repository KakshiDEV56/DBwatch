package tui

import (
	"strings"
	"time"
)

func fmtDuration(d time.Duration) string {
	return d.Round(time.Millisecond).String()
}

// truncate collapses embedded whitespace/newlines (multi-line SQL becomes
// one line) and clips to n runes with an ellipsis.
func truncate(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}
