package ui

import (
	"fmt"
	"os"
	"time"

	"github.com/briandowns/spinner"
)

// Spinner wraps briandowns/spinner with start/stop/fail semantics.
type Spinner struct {
	s       *spinner.Spinner
	message string
}

// NewSpinner creates a new styled spinner ready to start.
func NewSpinner(message string) *Spinner {
	s := spinner.New(spinner.CharSets[14], 80*time.Millisecond, spinner.WithWriter(os.Stderr))
	s.Prefix = " "
	s.Color("cyan", "bold") //nolint:errcheck
	return &Spinner{s: s, message: message}
}

// Start begins the spinner animation with the given message.
func (sp *Spinner) Start() {
	sp.s.Suffix = fmt.Sprintf("  %s", MutedStyle.Render(sp.message))
	sp.s.Start()
}

// UpdateMessage changes the spinner's label while it's running.
func (sp *Spinner) UpdateMessage(msg string) {
	sp.message = msg
	sp.s.Suffix = fmt.Sprintf("  %s", MutedStyle.Render(msg))
}

// Stop clears the spinner and prints a success line to stderr.
func (sp *Spinner) Stop(doneMsg string) {
	sp.s.Stop()
	fmt.Fprintf(os.Stderr, "  %s  %s\n",
		SuccessStyle.Render("✓"),
		MutedStyle.Render(doneMsg),
	)
}

// Fail clears the spinner and prints a failure line to stderr.
func (sp *Spinner) Fail(errMsg string) {
	sp.s.Stop()
	fmt.Fprintf(os.Stderr, "  %s  %s\n",
		ErrorStyle.Render("✗"),
		ErrorStyle.Render(errMsg),
	)
}
