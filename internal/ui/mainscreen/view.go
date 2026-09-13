package mainscreen

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/yachiko/fossor/internal/git"
	"github.com/yachiko/fossor/internal/ui/common"
	"github.com/yachiko/fossor/internal/ui/components"
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
	colWorktree = 1
	colBranch   = 20
	colAhead    = 7
	colBehind   = 7
	colChanges  = 9
	colStatus   = 14
)

// fixedColumnsWidth = leading indent(2) + spaces between cols(5) + branch + ahead + behind + changes + status
const fixedColumnsWidth = 2 + 5 + colBranch + colAhead + colBehind + colChanges + colStatus

func (m *Model) hasWorktrees() bool {
	for _, repo := range m.Repos {
		if repo.LinkedWorktree {
			return true
		}
	}
	return false
}

func (m *Model) nameColWidth() int {
	fixedWidth := fixedColumnsWidth
	if m.hasWorktrees() {
		fixedWidth += colWorktree + 1
	}
	w := m.width - fixedWidth
	if w < 12 {
		w = 12
	}
	return w
}

func (m *Model) statusCountsView() string {
	counts := make(map[git.RepoStatus]int)
	for _, r := range m.Repos {
		verification := m.Verification(r.Path)
		if verification == Unverified {
			continue
		}
		if verification == RemoteError {
			counts[git.StatusError]++
			continue
		}
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
	left := common.TitleStyle.Render("fossor") + "  " + dirStyle.Render(git.Sanitize(m.RootDir))
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
	b.WriteString("\n")

	// Search + filter line (always reserved)
	searchLeft := "  "
	if m.searching || m.searchText.Value() != "" {
		searchLeft += m.searchText.View()
	}
	filterRight := ""
	if m.filterMode != FilterAll {
		filterRight = lipgloss.NewStyle().Foreground(common.ColorYellow).Render("[" + m.filterMode.String() + "]")
	}
	if filterRight != "" {
		gap := m.width - lipgloss.Width(searchLeft) - lipgloss.Width(filterRight)
		if gap < 1 {
			gap = 1
		}
		b.WriteString(searchLeft + strings.Repeat(" ", gap) + filterRight)
	} else {
		b.WriteString(searchLeft)
	}
	b.WriteString("\n")

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

	var header string
	if m.hasWorktrees() {
		header = fmt.Sprintf("  %-*s %-*s %-*s %*s %*s %*s %-*s",
			colWorktree, "",
			colName, "Name"+sortIndicator(SortName),
			colBranch, "Branch"+sortIndicator(SortBranch),
			colAhead, "Ahead"+sortIndicator(SortAhead),
			colBehind, "Behind"+sortIndicator(SortBehind),
			colChanges, "Changes"+sortIndicator(SortChanges),
			colStatus, "Status"+sortIndicator(SortStatus),
		)
	} else {
		header = fmt.Sprintf("  %-*s %-*s %*s %*s %*s %-*s",
			colName, "Name"+sortIndicator(SortName),
			colBranch, "Branch"+sortIndicator(SortBranch),
			colAhead, "Ahead"+sortIndicator(SortAhead),
			colBehind, "Behind"+sortIndicator(SortBehind),
			colChanges, "Changes"+sortIndicator(SortChanges),
			colStatus, "Status"+sortIndicator(SortStatus),
		)
	}
	b.WriteString(headerStyle.Width(m.width).Render(header))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", m.width))
	b.WriteString("\n")

	// Rows
	rows := m.visibleRows()
	tableHeight := m.TableHeight()

	start := m.scrollOffset
	end := start + tableHeight
	if end > len(rows) {
		end = len(rows)
	}
	if start > len(rows) {
		start = len(rows)
	}

	for vi := start; vi < end; vi++ {
		tableRow := rows[vi]
		if tableRow.repoIndex < 0 {
			marker := m.worktreeMarker(tableRow)
			row := fmt.Sprintf("  %-*s %s worktrees (%d)", colWorktree, marker, git.Sanitize(tableRow.groupName), tableRow.groupSize)
			style := lipgloss.NewStyle().Bold(true).Foreground(common.ColorAccent)
			if vi == m.cursor {
				style = selectedStyle.Bold(true)
			}
			b.WriteString(style.Width(m.width).Render(row))
			b.WriteString("\n")
			continue
		}
		repo := m.Repos[tableRow.repoIndex]
		verification := m.Verification(repo.Path)

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
		switch verification {
		case Unverified:
			statusStr = "stale"
		case RemoteError:
			statusStr = "remote error"
		}
		if repo.Status == 0 {
			statusStr = "..."
		}

		name := git.Sanitize(repo.Name)
		if tableRow.groupKey != "" && !tableRow.groupRoot {
			name = "  " + name
		}
		marker := ""
		if tableRow.groupRoot {
			marker = m.worktreeMarker(tableRow)
		}
		var row string
		if m.hasWorktrees() {
			row = fmt.Sprintf("  %-*s %-*s %-*s %*s %*s %*s %-*s",
				colWorktree, marker,
				colName, truncate(name, colName),
				colBranch, truncate(git.Sanitize(repo.Branch), colBranch),
				colAhead, aheadStr,
				colBehind, behindStr,
				colChanges, changesStr,
				colStatus, statusStr,
			)
		} else {
			row = fmt.Sprintf("  %-*s %-*s %*s %*s %*s %-*s",
				colName, truncate(name, colName),
				colBranch, truncate(git.Sanitize(repo.Branch), colBranch),
				colAhead, aheadStr,
				colBehind, behindStr,
				colChanges, changesStr,
				colStatus, statusStr,
			)
		}

		if verification == Unverified {
			b.WriteString(lipgloss.NewStyle().Foreground(common.ColorMuted).Width(m.width).Render(row))
		} else if verification == RemoteError && vi == m.cursor {
			b.WriteString(lipgloss.NewStyle().Background(common.ColorSurface).Foreground(common.ColorRed).Width(m.width).Render(row))
		} else if vi == m.cursor {
			b.WriteString(selectedStyle.Width(m.width).Render(row))
		} else {
			statusColored := lipgloss.NewStyle().Foreground(common.StatusColor(statusStr)).Render(statusStr)
			var rowNoStatus string
			if m.hasWorktrees() {
				rowNoStatus = fmt.Sprintf("  %-*s %-*s %-*s %*s %*s %*s ",
					colWorktree, marker,
					colName, truncate(name, colName),
					colBranch, truncate(git.Sanitize(repo.Branch), colBranch),
					colAhead, aheadStr,
					colBehind, behindStr,
					colChanges, changesStr,
				)
			} else {
				rowNoStatus = fmt.Sprintf("  %-*s %-*s %*s %*s %*s ",
					colName, truncate(name, colName),
					colBranch, truncate(git.Sanitize(repo.Branch), colBranch),
					colAhead, aheadStr,
					colBehind, behindStr,
					colChanges, changesStr,
				)
			}
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
		"↵", "manage",
		"space", "toggle group",
		"s", "search",
		"t", "filter",
		"1-6", "sort",
		"p/P", "pull",
		"f/F", "fetch",
		"d/D", "default",
	}
	if m.OpenCmd != "" {
		helpPairs = append(helpPairs, "o", "open")
	}
	helpPairs = append(helpPairs, "q", "quit")
	b.WriteString(components.StatusBar(m.width, helpPairs, m.statusMsg))

	return b.String()
}

func (m *Model) worktreeMarker(row tableRow) string {
	if m.collapsed[row.groupKey] {
		return "▶"
	}
	return "▼"
}

func truncate(s string, maxLen int) string {
	if len([]rune(s)) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string([]rune(s)[:maxLen])
	}
	return string([]rune(s)[:maxLen-3]) + "..."
}
