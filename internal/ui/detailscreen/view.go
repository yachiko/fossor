package detailscreen

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/ahoma/fossor/internal/ui/common"
	"github.com/ahoma/fossor/internal/ui/components"
)

func (m *Model) View() string {
	if m.width == 0 {
		return ""
	}

	var b strings.Builder

	// Title
	b.WriteString(common.TitleStyle.Render(m.Repo.Name))
	b.WriteString("\n\n")

	// Tabs
	var tabViews []string
	for i, name := range tabNames {
		if i == m.activeTab {
			tabViews = append(tabViews, common.ActiveTabStyle.Render(name))
		} else {
			tabViews = append(tabViews, common.InactiveTabStyle.Render(name))
		}
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, tabViews...))
	b.WriteString("\n\n")

	// Tab content
	tabHeight := m.height - 6
	switch m.activeTab {
	case TabInfo:
		b.WriteString(m.infoTab.View(m.Repo, m.width, tabHeight))
	case TabCommits:
		b.WriteString(m.commitsTab.View())
	case TabChanges:
		b.WriteString(m.changesTab.View())
	case TabCLI:
		b.WriteString(m.cliTab.View())
	}

	// Pad to fill height
	lines := strings.Count(b.String(), "\n")
	for i := lines; i < m.height-2; i++ {
		b.WriteString("\n")
	}

	// Status bar
	helpPairs := []string{
		"tab", "switch tab",
		"1-4", "jump tab",
		"p", "pull",
		"u", "push",
		"esc", "back",
		"q", "quit",
	}
	b.WriteString(components.StatusBar(m.width, helpPairs, m.statusMsg))


	return b.String()
}
