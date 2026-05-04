package ui

import "github.com/charmbracelet/lipgloss"

// Brand palette
const (
	ColorAccent  = lipgloss.Color("#00D4AA") // teal-green brand color
	ColorSuccess = lipgloss.Color("#22C55E") // green
	ColorError   = lipgloss.Color("#EF4444") // red
	ColorWarn    = lipgloss.Color("#F59E0B") // amber
	ColorInfo    = lipgloss.Color("#60A5FA") // blue
	ColorMuted   = lipgloss.Color("#6B7280") // gray
	ColorBold    = lipgloss.Color("#F9FAFB") // near-white
	ColorDim     = lipgloss.Color("#4B5563") // dark gray
)

// Base styles
var (
	AccentStyle  = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	SuccessStyle = lipgloss.NewStyle().Foreground(ColorSuccess)
	ErrorStyle   = lipgloss.NewStyle().Foreground(ColorError)
	WarnStyle    = lipgloss.NewStyle().Foreground(ColorWarn)
	InfoStyle    = lipgloss.NewStyle().Foreground(ColorInfo)
	MutedStyle   = lipgloss.NewStyle().Foreground(ColorMuted)
	BoldStyle    = lipgloss.NewStyle().Foreground(ColorBold).Bold(true)
	DimStyle     = lipgloss.NewStyle().Foreground(ColorDim).Italic(true)
	HintStyle    = lipgloss.NewStyle().Foreground(ColorMuted).Italic(true)

	// Label styles used in key-value pairs
	KeyStyle   = lipgloss.NewStyle().Foreground(ColorMuted)
	ValueStyle = lipgloss.NewStyle().Foreground(ColorBold).Bold(true)

	// Status badge styles
	StatusQueued     = lipgloss.NewStyle().Foreground(ColorMuted).Bold(true)
	StatusProcessing = lipgloss.NewStyle().Foreground(ColorWarn).Bold(true)
	StatusComplete   = lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true)
	StatusFailed     = lipgloss.NewStyle().Foreground(ColorError).Bold(true)

	// Table styles
	TableHeaderStyle = lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	TableBorderStyle = lipgloss.NewStyle().Foreground(ColorDim)
	TableCellStyle   = lipgloss.NewStyle().Foreground(ColorBold)
	TableMutedCell   = lipgloss.NewStyle().Foreground(ColorMuted)

	// Box for highlighted outputs
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorAccent).
			Padding(0, 1).
			MarginTop(0)
)

// StatusStyle returns the lipgloss style matching the API job status string.
func StatusStyle(status string) lipgloss.Style {
	switch status {
	case "succeeded":
		return StatusComplete
	case "running":
		return StatusProcessing
	case "queued":
		return StatusQueued
	case "failed":
		return StatusFailed
	default:
		return MutedStyle
	}
}
