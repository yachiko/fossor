package manageview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/yachiko/fossor/internal/ui/common"
)

// viewStash renders the Stash tab (entry list + stash diff).
func (m *Model) viewStash() string {
	height := m.contentHeight(2 * sepLines) // top + bottom separators

	listWidth := m.width / 3
	if listWidth < 30 {
		listWidth = 30
	}
	diffWidth := m.width - listWidth - 5

	// Stash entry list (left)
	var entryList strings.Builder
	entryList.WriteString(catHeaderStyle.Render(fmt.Sprintf("Stash (%d)", len(m.stashEntries))) + "\n")

	if len(m.stashEntries) == 0 {
		entryList.WriteString(disabledStyle.Render("  (empty)") + "\n")
	} else {
		visible := height - 1
		if visible < 1 {
			visible = 1
		}
		if m.stashCursor < m.stashScroll {
			m.stashScroll = m.stashCursor
		}
		if m.stashCursor >= m.stashScroll+visible {
			m.stashScroll = m.stashCursor - visible + 1
		}
		end := m.stashScroll + visible
		if end > len(m.stashEntries) {
			end = len(m.stashEntries)
		}
		for i := m.stashScroll; i < end; i++ {
			label := truncate(m.stashEntries[i], listWidth-4)
			if i == m.stashCursor {
				sel := lipgloss.NewStyle().Background(common.ColorSurface).Foreground(common.ColorWhite)
				entryList.WriteString(sel.Render("> "+label) + "\n")
			} else {
				entryList.WriteString("  " + label + "\n")
			}
		}
	}

	// Stash diff (right)
	var diffPanel strings.Builder
	diffPanel.WriteString(catHeaderStyle.Render("Stash Diff") + "\n")
	m.stashDiffView.Width = diffWidth
	m.stashDiffView.Height = height - 1
	if m.stashDiffView.Height < 1 {
		m.stashDiffView.Height = 1
	}

	if len(m.stashEntries) == 0 {
		diffPanel.WriteString(disabledStyle.Render("  (no stash)") + "\n")
	} else if !m.stashDiffLoaded {
		diffPanel.WriteString(disabledStyle.Render("  loading...") + "\n")
	} else {
		diffPanel.WriteString(m.stashDiffView.View())
	}

	content := renderTwoColumns(entryList.String(), diffPanel.String(), listWidth, diffWidth, height) + "\n"

	return m.renderWithChrome(content, []string{
		"↑↓", "entries", "pgup/dn", "diff", "p", "pop", "d", "drop", "tab", "switch", "esc", "back", "q", "quit",
	})
}
