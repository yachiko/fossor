package manageview

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/yachiko/fossor/internal/ui/components"
)

// Layout constants — shared header and chrome line counts.
const (
	headerLines    = 3 // repo name + info + tabs
	statusBarLines = 2 // help keys + message
	sepLines       = 1 // one "───" line
)

// separator returns a full-width horizontal rule with indent.
func separator(width int) string {
	return "  " + strings.Repeat("─", width-4)
}

// contentHeight returns available lines for tab content, given extra chrome lines
// (separators, table headers, etc.) consumed by the specific tab.
func (m *Model) contentHeight(extraChrome int) int {
	h := m.height - headerLines - statusBarLines - extraChrome
	if h < 3 {
		h = 3
	}
	return h
}

// renderWithChrome wraps tab content with top/bottom separators, padding, and status bar.
func (m *Model) renderWithChrome(content string, helpPairs []string) string {
	var b strings.Builder
	sep := separator(m.width)

	b.WriteString(sep + "\n")
	b.WriteString(content)

	// Pad to push bottom separator + status bar to the bottom
	lines := strings.Count(b.String(), "\n")
	target := m.height - headerLines - statusBarLines - sepLines
	for i := lines; i < target; i++ {
		b.WriteString("\n")
	}

	b.WriteString(sep + "\n")
	b.WriteString(components.StatusBar(m.width, helpPairs, m.statusMsg))

	return b.String()
}

// renderTwoColumns joins two content strings side by side at a fixed height.
func renderTwoColumns(left, right string, leftWidth, rightWidth, height int) string {
	leftCol := lipgloss.NewStyle().Width(leftWidth).Height(height).Render(strings.TrimRight(left, "\n"))
	rightCol := lipgloss.NewStyle().Width(rightWidth).Height(height).Render(strings.TrimRight(right, "\n"))
	return "  " + lipgloss.JoinHorizontal(lipgloss.Top, leftCol, " ", rightCol)
}
