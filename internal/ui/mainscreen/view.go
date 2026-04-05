package mainscreen

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/ahoma/fossor/internal/git"
	"github.com/ahoma/fossor/internal/ui/common"
	"github.com/ahoma/fossor/internal/ui/components"
)

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(common.ColorAccent)

	selectedStyle = lipgloss.NewStyle().
			Background(common.ColorSurface).
			Foreground(common.ColorWhite)
)

// Fixed column widths (Name is dynamic)
const (
	colBranch  = 20
	colAhead   = 7
	colBehind  = 7
	colChanges = 9
	colStatus  = 14
)

// fixedColumnsWidth = leading indent(2) + spaces between cols(5) + branch + ahead + behind + changes + status
const fixedColumnsWidth = 2 + 5 + colBranch + colAhead + colBehind + colChanges + colStatus

func (m *Model) nameColWidth() int {
	w := m.width - fixedColumnsWidth
	if w < 12 {
		w = 12
	}
	return w
}

func (m *Model) statusCountsView() string {
	counts := make(map[git.RepoStatus]int)
	for _, r := range m.Repos {
		counts[r.Status]++
	}

	// Display order: most urgent first
	order := []git.RepoStatus{
		git.StatusError,
		git.StatusNonDefault,
		git.StatusDiverged,
		git.StatusBehind,
		git.StatusAhead,
		git.StatusDirty,
		git.StatusUpToDate,
	}

	var parts []string
	for _, s := range order {
		label := strings.ToLower(s.String())
		style := lipgloss.NewStyle().Foreground(common.StatusColor(s.String()))
		parts = append(parts, style.Render(fmt.Sprintf("%d %s", counts[s], label)))
	}
	return strings.Join(parts, "  ")
}

func (m *Model) View() string {
	if m.width == 0 {
		return ""
	}

	colName := m.nameColWidth()

	var b strings.Builder

	// Title with status counts
	dirStyle := lipgloss.NewStyle().Foreground(common.ColorMuted)
	left := common.TitleStyle.Render("fossor") + "  " + dirStyle.Render(m.RootDir)
	right := m.statusCountsView()
	if right != "" {
		gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
		if gap < 2 {
			gap = 2
		}
		b.WriteString(left + strings.Repeat(" ", gap) + right)
	} else {
		b.WriteString(left)
	}
	b.WriteString("\n\n")

	// Search bar
	if m.searching || m.searchText.Value() != "" {
		b.WriteString("  " + m.searchText.View() + "\n\n")
	}

	// Filter label
	if m.filterMode != FilterAll {
		filterLabel := lipgloss.NewStyle().Foreground(common.ColorYellow).Render("[Filter: " + m.filterMode.String() + "]")
		b.WriteString("  " + filterLabel + "\n")
	}

	// Header
	sortIndicator := func(col SortColumn) string {
		if m.sortCol == col {
			if m.sortAsc {
				return " ▲"
			}
			return " ▼"
		}
		return ""
	}

	header := fmt.Sprintf("  %-*s %-*s %*s %*s %*s %-*s",
		colName, "Name"+sortIndicator(SortName),
		colBranch, "Branch"+sortIndicator(SortBranch),
		colAhead, "Ahead"+sortIndicator(SortAhead),
		colBehind, "Behind"+sortIndicator(SortBehind),
		colChanges, "Changes"+sortIndicator(SortChanges),
		colStatus, "Status"+sortIndicator(SortStatus),
	)
	b.WriteString(headerStyle.Width(m.width).Render(header))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", m.width))
	b.WriteString("\n")

	// Rows
	indices := m.visibleIndices()
	tableHeight := m.TableHeight()

	start := m.scrollOffset
	end := start + tableHeight
	if end > len(indices) {
		end = len(indices)
	}
	if start > len(indices) {
		start = len(indices)
	}

	for vi := start; vi < end; vi++ {
		idx := indices[vi]
		repo := m.Repos[idx]

		aheadStr := "-"
		behindStr := "-"
		changesStr := "-"
		if repo.Status != 0 { // not unknown
			aheadStr = fmt.Sprintf("%d", repo.Ahead)
			behindStr = fmt.Sprintf("%d", repo.Behind)
			if repo.Changes > 0 {
				changesStr = fmt.Sprintf("%d", repo.Changes)
			} else {
				changesStr = "-"
			}
		}

		statusStr := repo.Status.String()
		if repo.Status == 0 {
			statusStr = "..."
		}

		row := fmt.Sprintf("  %-*s %-*s %*s %*s %*s %-*s",
			colName, truncate(repo.Name, colName),
			colBranch, truncate(repo.Branch, colBranch),
			colAhead, aheadStr,
			colBehind, behindStr,
			colChanges, changesStr,
			colStatus, statusStr,
		)

		if vi == m.cursor {
			b.WriteString(selectedStyle.Width(m.width).Render(row))
		} else {
			statusColored := lipgloss.NewStyle().Foreground(common.StatusColor(statusStr)).Render(statusStr)
			rowNoStatus := fmt.Sprintf("  %-*s %-*s %*s %*s %*s ",
				colName, truncate(repo.Name, colName),
				colBranch, truncate(repo.Branch, colBranch),
				colAhead, aheadStr,
				colBehind, behindStr,
				colChanges, changesStr,
			)
			fullRow := rowNoStatus + statusColored
			padding := m.width - lipgloss.Width(fullRow)
			if padding > 0 {
				fullRow += strings.Repeat(" ", padding)
			}
			b.WriteString(fullRow)
		}
		b.WriteString("\n")
	}

	// Pad empty rows
	for i := end - start; i < tableHeight; i++ {
		b.WriteString("\n")
	}

	// Separator
	b.WriteString(strings.Repeat("─", m.width))
	b.WriteString("\n")

	// Status bar
	helpPairs := []string{
		"↵", "details",
		"s", "search",
		"t", "filter",
		"1-6", "sort",
		"p/P", "pull",
		"f/F", "fetch",
		"d/D", "default",
		"q", "quit",
	}
	b.WriteString(components.StatusBar(m.width, helpPairs, m.statusMsg))

	return b.String()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
