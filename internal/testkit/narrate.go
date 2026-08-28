package testkit

import (
	"fmt"
	"time"
)

// Narrator prints a scenario's step-by-step progress to stdout, so
// running a scenario explains what it's doing rather than acting as a
// black box (per the harness's own requirement).
type Narrator struct {
	title string
}

func NewNarrator(title string) *Narrator {
	fmt.Printf("\n[%s]\n\n", title)
	return &Narrator{title: title}
}

func (n *Narrator) Step(format string, args ...any) {
	fmt.Printf("%s\n", fmt.Sprintf(format, args...))
}

func (n *Narrator) OK(format string, args ...any) {
	fmt.Printf("✓ %s\n", fmt.Sprintf(format, args...))
}

func (n *Narrator) Warn(format string, args ...any) {
	fmt.Printf("⚠ %s\n", fmt.Sprintf(format, args...))
}

func (n *Narrator) Fail(format string, args ...any) {
	fmt.Printf("✕ %s\n", fmt.Sprintf(format, args...))
}

func (n *Narrator) Wait(format string, args ...any) {
	fmt.Printf("… %s\n", fmt.Sprintf(format, args...))
}

func (n *Narrator) Expect(format string, args ...any) {
	fmt.Printf("\nExpected in dbwatch:\n%s\n", fmt.Sprintf(format, args...))
}

func (n *Narrator) Done(d time.Duration) {
	fmt.Printf("\n[%s] finished in %s\n", n.title, d.Round(time.Millisecond))
}
