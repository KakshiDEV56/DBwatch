package tui

import (
	"fmt"
	"strings"
	"time"
)

func fmtDuration(d time.Duration) string {
	return d.Round(time.Millisecond).String()
}

// humanRate formats a bytes/second rate compactly, e.g. "12.3 KB/s".
func humanRate(bytesPerSec float64) string {
	const unit = 1024.0
	if bytesPerSec < unit {
		return fmt.Sprintf("%.0f B/s", bytesPerSec)
	}
	div, exp := unit, 0
	for n := bytesPerSec / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB/s", bytesPerSec/div, "KMGTPE"[exp])
}

// humanBytes formats an absolute byte count compactly, e.g. "12.3 MiB".
func humanBytes(n int64) string {
	const unit = 1024.0
	f := float64(n)
	if f < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := unit, 0
	for x := f / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", f/div, "KMGTPE"[exp])
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
