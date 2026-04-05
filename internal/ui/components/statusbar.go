package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var barStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#626262")).
	Padding(0, 1)

var keyStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#89B4FA"))

var descStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#626262"))

func renderPairs(pairs []string, width int) string {
	var b strings.Builder
	for i := 0; i+1 < len(pairs); i += 2 {
		if b.Len() > 0 {
			b.WriteString("  ")
		}
		b.WriteString(keyStyle.Render(pairs[i]))
		b.WriteString(" ")
		b.WriteString(descStyle.Render(pairs[i+1]))
	}
	return b.String()
}

// StatusBar renders a status bar with help keys and an optional message on a separate line.
// Each rows entry is a []string of key/description pairs.
func StatusBar(width int, rows ...interface{}) string {
	var keyLines []string
	var message string

	for _, row := range rows {
		switch v := row.(type) {
		case []string:
			keyLines = append(keyLines, renderPairs(v, width))
		case string:
			message = v
		}
	}

	msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#DBBD70"))
	var lines []string
	if message != "" {
		lines = append(lines, msgStyle.Render(message))
	} else {
		lines = append(lines, "")
	}
	lines = append(lines, keyLines...)

	result := strings.Join(lines, "\n")
	return barStyle.Width(width).Render(result)
}
