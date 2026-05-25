package manageview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/yachiko/fossor/internal/ui/common"
	"github.com/yachiko/fossor/internal/ui/components"
)

// View is the main entry point — renders shared header then delegates to the active tab.
func (m *Model) View() string {
	if m.width == 0 {
		return ""
	}

	var b strings.Builder

	// Line 1: repo name
	b.WriteString(" " + common.TitleStyle.Render(m.Repo.Name))
	b.WriteString("\n")

	// Line 2: repo info
	statusColor := common.StatusColor(m.Repo.Status.String())
	statusStyled := lipgloss.NewStyle().Foreground(statusColor).Bold(true).Render(m.Repo.Status.String())
	remote := m.remote
	if remote == "" {
		remote = "–"
	}
	b.WriteString(fmt.Sprintf("  %s  %s → %s  remote:%s  ahead:%d  behind:%d  changes:%d  stash:%d\n",
		statusStyled, m.Repo.Branch, m.Repo.DefaultBranch, remote,
		m.Repo.Ahead, m.Repo.Behind, m.Repo.Changes, len(m.stashEntries),
	))

	// Line 3: tab bar
	var tabViews []string
	for i, name := range tabNames {
		if i == m.activeTab {
			tabViews = append(tabViews, common.ActiveTabStyle.Render(name))
		} else {
			tabViews = append(tabViews, common.InactiveTabStyle.Render(name))
		}
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, tabViews...))
	b.WriteString("\n")

	// Tab content
	switch m.activeTab {
	case TabStatus:
		b.WriteString(m.viewStatus())
	case TabHistory:
		b.WriteString(m.viewHistory())
	case TabStash:
		b.WriteString(m.viewStash())
	case TabBranches:
		b.WriteString(m.viewBranches())
	}

	return b.String()
}

// viewStatus renders the Status tab (file panels + actions).
func (m *Model) viewStatus() string {
	if m.mode == modeCommit {
		return m.viewCommit()
	}

	var top, bottom strings.Builder
	sep := separator(m.width)

	// === Bottom section (rendered first to measure height) ===

	if m.lastAction != "" {
		actionLabel := lipgloss.NewStyle().Foreground(common.ColorAccent).Render(m.lastAction)
		if m.lastErr != nil {
			bottom.WriteString(fmt.Sprintf("  %s: %s\n", actionLabel, lipgloss.NewStyle().Foreground(common.ColorRed).Render(m.lastErr.Error())))
		} else {
			bottom.WriteString(fmt.Sprintf("  %s: %s\n", actionLabel, lipgloss.NewStyle().Foreground(common.ColorGreen).Render("done")))
		}
	}

	switch m.mode {
	case modeConfirm:
		action := m.actions[m.pendingIdx]
		bottom.WriteString(warnStyle.Render(fmt.Sprintf("  Execute %q? This is destructive. (y to confirm, any key to cancel)", action.Name)) + "\n")
	case modeInput:
		action := m.actions[m.pendingIdx]
		bottom.WriteString(fmt.Sprintf("  %s\n", action.InputPrompt))
		bottom.WriteString("  " + m.textInput.View() + "\n")
	}

	bottom.WriteString(m.renderActionGrid())

	helpPairs := []string{"↑↓", "files", "pgup/dn", "diff", "x", "restore", "X", "delete", "tab", "switch", "esc", "back", "q", "quit"}
	bottom.WriteString(components.StatusBar(m.width, helpPairs, m.statusMsg))

	bottomStr := bottom.String()
	bottomHeight := strings.Count(bottomStr, "\n")

	// === Top section (panels fill remaining space) ===

	top.WriteString(sep + "\n")

	// Panel height: total minus header, separators, padding, and bottom section
	panelHeight := m.height - headerLines - 2*sepLines - 1 - bottomHeight
	if panelHeight < 3 {
		panelHeight = 3
	}

	fileListWidth := m.width / 3
	if fileListWidth < 25 {
		fileListWidth = 25
	}
	diffWidth := m.width - fileListWidth - 5

	// File list panel (left)
	var fileList strings.Builder
	fileList.WriteString(catHeaderStyle.Render(fmt.Sprintf("Changes (%d)", len(m.changes))) + "\n")

	if len(m.changes) == 0 {
		fileList.WriteString(disabledStyle.Render("  (clean)") + "\n")
	} else {
		visibleFiles := panelHeight - 1
		if visibleFiles < 1 {
			visibleFiles = 1
		}
		if m.fileCursor < m.fileScroll {
			m.fileScroll = m.fileCursor
		}
		if m.fileCursor >= m.fileScroll+visibleFiles {
			m.fileScroll = m.fileCursor - visibleFiles + 1
		}
		end := m.fileScroll + visibleFiles
		if end > len(m.changes) {
			end = len(m.changes)
		}
		for i := m.fileScroll; i < end; i++ {
			c := m.changes[i]
			indicator := fileChangeIndicator(c)
			nameWidth := fileListWidth - 6
			if c.IsSubmodule {
				nameWidth -= 6
			}
			name := truncate(c.Path, nameWidth)
			if i == m.fileCursor {
				sel := lipgloss.NewStyle().Background(common.ColorSurface).Foreground(common.ColorWhite)
				fileList.WriteString(sel.Render(fmt.Sprintf("> %s %s", indicator, name)) + "\n")
			} else {
				fileList.WriteString(fmt.Sprintf("  %s %s\n", indicator, name))
			}
		}
	}

	// Diff panel (right)
	var diffPanel strings.Builder
	diffPanel.WriteString(catHeaderStyle.Render("Diff") + "\n")
	m.diffView.Width = diffWidth
	m.diffView.Height = panelHeight - 1
	if m.diffView.Height < 1 {
		m.diffView.Height = 1
	}
	if len(m.changes) == 0 {
		diffPanel.WriteString(disabledStyle.Render("  (no files)") + "\n")
	} else if !m.diffLoaded {
		diffPanel.WriteString(disabledStyle.Render("  loading...") + "\n")
	} else {
		diffPanel.WriteString(m.diffView.View())
	}

	top.WriteString(renderTwoColumns(fileList.String(), diffPanel.String(), fileListWidth, diffWidth, panelHeight))
	top.WriteString("\n" + sep + "\n")

	return top.String() + bottomStr
}
