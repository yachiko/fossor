package common

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	ColorGreen   = lipgloss.Color("#04B575")
	ColorYellow  = lipgloss.Color("#DBBD70")
	ColorRed     = lipgloss.Color("#FF5F56")
	ColorBlue    = lipgloss.Color("#61AFEF")
	ColorMuted   = lipgloss.Color("#626262")
	ColorWhite   = lipgloss.Color("#FAFAFA")
	ColorBg      = lipgloss.Color("#1E1E2E")
	ColorAccent  = lipgloss.Color("#89B4FA")
	ColorSurface = lipgloss.Color("#313244")

	// Styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAccent).
			Padding(0, 1)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Padding(0, 1)

	ActiveTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorBg).
			Background(ColorAccent).
			Padding(0, 2)

	InactiveTabStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Padding(0, 2)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(ColorAccent)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)
)

// StatusColor returns the appropriate color for a repo status string.
func StatusColor(status string) lipgloss.Color {
	switch status {
	case "Up to date":
		return ColorGreen
	case "Behind", "Dirty", "Diverged":
		return ColorYellow
	case "Non-default", "Error":
		return ColorRed
	case "Ahead":
		return ColorBlue
	default:
		return ColorMuted
	}
}
