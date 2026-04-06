package manageview

// viewHistory renders the History tab (scrollable commit log).
func (m *Model) viewHistory() string {
	height := m.contentHeight(2 * sepLines) // top + bottom separators

	m.commitsView.Width = m.width - 4
	m.commitsView.Height = height

	var content string
	if !m.commitsLoaded {
		content = "  Loading commits...\n"
	} else {
		content = m.commitsView.View() + "\n"
	}

	return m.renderWithChrome(content, []string{
		"↑↓", "scroll", "tab", "switch", "esc", "back", "q", "quit",
	})
}
