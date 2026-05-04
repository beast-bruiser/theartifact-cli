package ui

import (
	"fmt"
	"os"
	"strings"
)

// Banner is the ASCII art logo printed for help and welcome.
const Banner = `
  ████████╗██╗  ██╗███████╗     █████╗ ██████╗ ████████╗██╗███████╗ █████╗  ██████╗████████╗
     ██╔══╝██║  ██║██╔════╝    ██╔══██╗██╔══██╗╚══██╔══╝██║██╔════╝██╔══██╗██╔════╝╚══██╔══╝
     ██║   ███████║█████╗      ███████║██████╔╝   ██║   ██║█████╗  ███████║██║        ██║   
     ██║   ██╔══██║██╔══╝      ██╔══██║██╔══██╗   ██║   ██║██╔══╝  ██╔══██║██║        ██║   
     ██║   ██║  ██║███████╗    ██║  ██║██║  ██║   ██║   ██║██║     ██║  ██║╚██████╗   ██║   
     ╚═╝   ╚═╝  ╚═╝╚══════╝    ╚═╝  ╚═╝╚═╝  ╚═╝   ╚═╝   ╚═╝╚═╝     ╚═╝  ╚═╝ ╚═════╝   ╚═╝   
`

// PrintBanner prints the styled ASCII art banner to stderr.
func PrintBanner() {
	fmt.Fprintln(os.Stderr, AccentStyle.Render(Banner))
	fmt.Fprintln(os.Stderr, MutedStyle.Render("  AI-powered asset generation for creators · theartifact.art"))
	fmt.Fprintln(os.Stderr)
}

// PrintSuccess prints a green ✓ success line to stdout.
func PrintSuccess(msg string) {
	fmt.Printf("  %s  %s\n", SuccessStyle.Render("✓"), BoldStyle.Render(msg))
}

// PrintError prints a red ✗ error line to stderr.
func PrintError(msg string) {
	fmt.Fprintf(os.Stderr, "  %s  %s\n", ErrorStyle.Render("✗"), ErrorStyle.Render(msg))
}

// PrintWarn prints an amber ⚠ warning line to stderr.
func PrintWarn(msg string) {
	fmt.Fprintf(os.Stderr, "  %s  %s\n", WarnStyle.Render("⚠"), WarnStyle.Render(msg))
}

// PrintInfo prints a cyan ● info line to stderr.
func PrintInfo(msg string) {
	fmt.Fprintf(os.Stderr, "  %s  %s\n", InfoStyle.Render("●"), MutedStyle.Render(msg))
}

// PrintHint prints a dimmed italic hint/suggestion to stderr.
func PrintHint(msg string) {
	fmt.Fprintf(os.Stderr, "\n  %s\n", HintStyle.Render(msg))
}

// PrintKeyValue prints a styled "key  value" pair to stdout.
func PrintKeyValue(key, value string) {
	fmt.Printf("  %s  %s\n", KeyStyle.Render(key+":"), ValueStyle.Render(value))
}

// PrintSectionHeader prints a bold section divider.
func PrintSectionHeader(title string) {
	fmt.Println()
	fmt.Println(" " + AccentStyle.Render("▸ "+title))
}

// MaskKey masks an API key like tak_live_*****1234.
func MaskKey(key string) string {
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:8] + strings.Repeat("*", len(key)-12) + key[len(key)-4:]
}

// PrintTable renders a simple aligned table with a styled header.
// headers: column names, rows: 2D string slice of values.
func PrintTable(headers []string, rows [][]string) {
	if len(rows) == 0 {
		fmt.Println(MutedStyle.Render("  (no results)"))
		return
	}

	// Compute column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Header row
	fmt.Println()
	header := " "
	separator := " "
	for i, h := range headers {
		padded := fmt.Sprintf("%-*s", widths[i], h)
		header += TableHeaderStyle.Render(padded)
		separator += TableBorderStyle.Render(strings.Repeat("─", widths[i]))
		if i < len(headers)-1 {
			header += TableBorderStyle.Render("  │  ")
			separator += TableBorderStyle.Render("──┼──")
		}
	}
	fmt.Println(header)
	fmt.Println(separator)

	// Data rows — alternate muted/normal for readability
	for rowIdx, row := range rows {
		line := " "
		for i := range headers {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			padded := fmt.Sprintf("%-*s", widths[i], cell)
			if rowIdx%2 == 0 {
				line += TableCellStyle.Render(padded)
			} else {
				line += TableMutedCell.Render(padded)
			}
			if i < len(headers)-1 {
				line += TableBorderStyle.Render("  │  ")
			}
		}
		fmt.Println(line)
	}
	fmt.Println()
}
