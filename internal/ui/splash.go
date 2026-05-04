package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// ModelVersion is the current generation engine version advertised at startup.
const ModelVersion = "v1.5.0"

// PrintSplash renders an animated startup experience similar to Claude Code.
// Line-by-line banner reveal is skipped on non-TTY (CI/pipe) output.
func PrintSplash(workspaceName string) {
	printAnimatedBanner()
	printStartupInfo(workspaceName)
}

// printAnimatedBanner streams the ASCII banner to stderr line-by-line.
// Non-empty lines are delayed slightly to produce a build-up effect on TTY.
func printAnimatedBanner() {
	lines := strings.Split(Banner, "\n")
	tty := isTTY()
	for _, line := range lines {
		fmt.Fprintln(os.Stderr, AccentStyle.Render(line))
		if tty && strings.TrimSpace(line) != "" {
			time.Sleep(18 * time.Millisecond)
		}
	}
}

// printStartupInfo prints the version/workspace info block that follows the banner.
func printStartupInfo(workspaceName string) {
	borderColor := ColorAccent
	border := lipgloss.NewStyle().Foreground(borderColor)
	rule := border.Render(strings.Repeat("─", 54))

	fmt.Fprintln(os.Stderr, "  "+rule)

	// Model line
	modelTag := lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true).
		Render("  model")
	modelVal := lipgloss.NewStyle().
		Foreground(ColorBold).
		Bold(true).
		Render(ModelVersion)
	fmt.Fprintf(os.Stderr, "%s  %s\n", modelTag, modelVal)

	// Workspace line
	wsLabel := MutedStyle.Render("  workspace")
	var wsVal string
	if workspaceName != "" {
		wsVal = BoldStyle.Render(workspaceName)
	} else {
		wsVal = WarnStyle.Render("none — pick one after your first prompt")
	}
	fmt.Fprintf(os.Stderr, "%s  %s\n", wsLabel, wsVal)

	fmt.Fprintln(os.Stderr, "  "+rule)
	fmt.Fprintln(os.Stderr, MutedStyle.Render("  Type a prompt to generate  ·  /help  ·  /quit"))
	fmt.Fprintln(os.Stderr)
}
