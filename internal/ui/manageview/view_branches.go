package manageview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/ahoma/fossor/internal/ui/common"
	"github.com/ahoma/fossor/internal/ui/components"
)

// viewBranches renders the Branches tab as a table.
func (m *Model) viewBranches() string {
	var b strings.Builder
	sep := separator(m.width)

	// Column widths
	colName := 50
	colAhead := 7
	colBehind := 7
	colMerged := 8
	colDate := 12
	colMsg := m.width - colName - colAhead - colBehind - colMerged - colDate - 8
	if colMsg < 10 {
		colMsg = 10
	}

	// Separator + table header
	b.WriteString(sep + "\n")
	hdrStyle := lipgloss.NewStyle().Bold(true).Foreground(common.ColorMuted)
	b.WriteString(hdrStyle.Render(fmt.Sprintf("  %-*s %-*s %-*s %-*s %-*s %s",
		colName, "Branch",
		colAhead, "Ahead",
		colBehind, "Behind",
		colMerged, "Merged",
		colDate, "Date",
		"Message",
	)) + "\n")

	// Rows
	// Extra chrome: top sep(1) + table header(1) + bottom sep(1)
	tableHeight := m.contentHeight(3*sepLines + 1) // +1 for table header
	if m.branchInputMode {
		tableHeight--
	}
	if tableHeight < 1 {
		tableHeight = 1
	}

	if !m.branchesLoaded {
		b.WriteString(disabledStyle.Render("  loading...") + "\n")
	} else if len(m.branches) == 0 {
		b.WriteString(disabledStyle.Render("  (no branches)") + "\n")
	} else {
		if m.branchCursor < m.branchScroll {
			m.branchScroll = m.branchCursor
		}
		if m.branchCursor >= m.branchScroll+tableHeight {
			m.branchScroll = m.branchCursor - tableHeight + 1
		}
		end := m.branchScroll + tableHeight
		if end > len(m.branches) {
			end = len(m.branches)
		}

		nameCol := lipgloss.NewStyle().Width(colName + 1)
		aheadCol := lipgloss.NewStyle().Width(colAhead + 1)
		behindCol := lipgloss.NewStyle().Width(colBehind + 1)
		mergedCol := lipgloss.NewStyle().Width(colMerged + 1)
		dateCol := lipgloss.NewStyle().Width(colDate + 1)

		for i := m.branchScroll; i < end; i++ {
			br := m.branches[i]

			prefix := "  "
			if i == m.branchCursor {
				prefix = "> "
			}

			name := truncate(br.Name, colName)
			if br.IsCurrent {
				name = lipgloss.NewStyle().Foreground(common.ColorGreen).Bold(true).Render(name)
			}

			aheadStr := ""
			if br.Ahead > 0 {
				aheadStr = lipgloss.NewStyle().Foreground(common.ColorGreen).Render(fmt.Sprintf("%d↑", br.Ahead))
			}

			behindStr := ""
			if br.Behind > 0 {
				behindStr = lipgloss.NewStyle().Foreground(common.ColorRed).Render(fmt.Sprintf("%d↓", br.Behind))
			}

			mergedStr := ""
			if br.Merged && br.Name != m.Repo.DefaultBranch {
				mergedStr = lipgloss.NewStyle().Foreground(common.ColorGreen).Render("✓")
			}

			row := prefix +
				nameCol.Render(name) +
				aheadCol.Render(aheadStr) +
				behindCol.Render(behindStr) +
				mergedCol.Render(mergedStr) +
				dateCol.Render(disabledStyle.Render(br.LastDate)) +
				disabledStyle.Render(truncate(br.LastMsg, colMsg))

			if i == m.branchCursor {
				b.WriteString(lipgloss.NewStyle().Background(common.ColorSurface).Width(m.width).Render(row) + "\n")
			} else {
				b.WriteString(row + "\n")
			}
		}
	}

	// Pad rows
	written := strings.Count(b.String(), "\n") - 2
	for i := written; i < tableHeight; i++ {
		b.WriteString("\n")
	}

	b.WriteString(sep + "\n")

	// Branch input
	if m.branchInputMode {
		label := "New branch"
		if m.branchInputAction == "rename" {
			label = "Rename to"
		}
		b.WriteString("  " + catHeaderStyle.Render(label) + "  " + m.branchInput.View() + "\n")
	}

	// Pad to bottom
	lines := strings.Count(b.String(), "\n")
	target := m.height - headerLines - statusBarLines
	for i := lines; i < target; i++ {
		b.WriteString("\n")
	}

	helpPairs := []string{"↑↓", "select", "↵/s", "switch", "n", "new", "r", "rename", "d", "delete", "D", "force del", "tab", "switch tab", "esc", "back", "q", "quit"}
	b.WriteString(components.StatusBar(m.width, helpPairs, m.statusMsg))

	return b.String()
}

// renderActionGrid renders the categorized action button grid.
func (m *Model) renderActionGrid() string {
	var b strings.Builder
	categories := AllCategories()
	var columns []string
	for _, cat := range categories {
		var col strings.Builder
		col.WriteString(catHeaderStyle.Render(cat.String()) + "\n")
		for _, action := range m.actions {
			if action.Category != cat {
				continue
			}
			enabled := action.Enabled(m.Repo)
			if enabled {
				ks := enabledKeyStyle
				if action.Dangerous {
					ks = dangerKeyStyle
				}
				col.WriteString(fmt.Sprintf("  %s %s\n", ks.Render(action.Key), enabledNameStyle.Render(action.Name)))
			} else {
				col.WriteString(fmt.Sprintf("  %s %s\n", disabledStyle.Render(action.Key), disabledStyle.Render(action.Name)))
			}
		}
		columns = append(columns, col.String())
	}
	colWidth := (m.width - 4) / len(categories)
	if colWidth < 20 {
		colWidth = 20
	}
	styledCols := make([]string, len(columns))
	for i, c := range columns {
		styledCols[i] = lipgloss.NewStyle().Width(colWidth).Render(c)
	}
	b.WriteString("  " + lipgloss.JoinHorizontal(lipgloss.Top, styledCols...) + "\n")
	return b.String()
}
