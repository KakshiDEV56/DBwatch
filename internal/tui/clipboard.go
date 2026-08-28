package tui

import "github.com/muesli/termenv"

// Copy sends text to the terminal's clipboard via an OSC52 escape
// sequence. This works over SSH/tmux without any external clipboard
// binary, since the terminal emulator itself receives the request.
func Copy(text string) {
	termenv.Copy(text)
}
