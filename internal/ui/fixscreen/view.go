package fixscreen

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/ahoma/fossor/internal/git"
	"github.com/ahoma/fossor/internal/ui/common"
	"github.com/ahoma/fossor/internal/ui/components"
)

var (
	enabledKeyStyle  = lipgloss.NewStyle().Foreground(common.ColorAccent).Bold(true)
	enabledNameStyle = lipgloss.NewStyle().Foreground(common.ColorWhite)
	dangerKeyStyle   = lipgloss.NewStyle().Foreground(common.ColorRed).Bold(true)
	disabledStyle    = lipgloss.NewStyle().Foreground(common.ColorMuted)
	catHeaderStyle   = lipgloss.NewStyle().Bold(true).Foreground(common.ColorAccent)
	warnStyle        = lipgloss.NewStyle().Foreground(common.ColorRed).Bold(true)
)

func (m *Model) View() string {
	if m.width == 0 {
		return ""
	}

	var b strings.Builder

	// Line 1: repo name
	b.WriteString(common.TitleStyle.Render(m.Repo.Name))
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

	// Tab bar
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
	sep := strings.Repeat("─", m.width-4)

	// === Bottom section (rendered first to measure height) ===

	// Last action result
	if m.lastAction != "" {
		actionLabel := lipgloss.NewStyle().Foreground(common.ColorAccent).Render(m.lastAction)
		if m.lastErr != nil {
			bottom.WriteString(fmt.Sprintf("  %s: %s\n", actionLabel, lipgloss.NewStyle().Foreground(common.ColorRed).Render(m.lastErr.Error())))
		} else {
			bottom.WriteString(fmt.Sprintf("  %s: %s\n", actionLabel, lipgloss.NewStyle().Foreground(common.ColorGreen).Render("done")))
		}
	}

	// Confirm / input overlays
	switch m.mode {
	case modeConfirm:
		action := m.actions[m.pendingIdx]
		bottom.WriteString(warnStyle.Render(fmt.Sprintf("  Execute %q? This is destructive. (y to confirm, any key to cancel)", action.Name)) + "\n")
	case modeInput:
		action := m.actions[m.pendingIdx]
		bottom.WriteString(fmt.Sprintf("  %s\n", action.InputPrompt))
		bottom.WriteString("  " + m.textInput.View() + "\n")
	}

	// Action grid
	bottom.WriteString(m.renderActionGrid())

	// Status bar
	helpPairs := []string{"↑↓", "files", "pgup/dn", "diff", "x", "restore", "X", "delete", "tab", "switch", "esc", "back", "q", "quit"}
	bottom.WriteString(components.StatusBar(m.width, helpPairs, m.statusMsg))

	bottomStr := bottom.String()
	bottomHeight := strings.Count(bottomStr, "\n")

	// === Top section (panels fill remaining space) ===

	top.WriteString("  " + sep + "\n")

	// Panel height: remaining space between header+sep and bottom
	// header(3) + sep_top(1) + sep_bottom(1) = 5 fixed top chrome
	panelHeight := m.height - 5 - bottomHeight
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
	fileHeader := catHeaderStyle.Render(fmt.Sprintf("Changes (%d)", len(m.changes)))
	fileList.WriteString(fileHeader + "\n")

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
				nameWidth -= 6 // space for " [sub]"
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

	// Join panels — use lipgloss.Height to ensure both columns match
	leftStr := strings.TrimRight(fileList.String(), "\n")
	rightStr := strings.TrimRight(diffPanel.String(), "\n")
	leftCol := lipgloss.NewStyle().Width(fileListWidth).Height(panelHeight).Render(leftStr)
	rightCol := lipgloss.NewStyle().Width(diffWidth).Height(panelHeight).Render(rightStr)
	top.WriteString("  " + lipgloss.JoinHorizontal(lipgloss.Top, leftCol, " ", rightCol))
	top.WriteString("\n  " + sep + "\n")

	return top.String() + bottomStr
}

// viewCommit renders the inline commit message editor with staged diff.
func (m *Model) viewCommit() string {
	var b strings.Builder
	sep := strings.Repeat("─", m.width-4)

	b.WriteString("  " + sep + "\n")
	b.WriteString("  " + catHeaderStyle.Render("Commit message") + "\n")
	m.commitInput.SetWidth(m.width - 6)
	for _, line := range strings.Split(m.commitInput.View(), "\n") {
		b.WriteString("  " + line + "\n")
	}
	b.WriteString("  " + sep + "\n")

	// Staged diff below
	// header(3) + sep(1) + label(1) + textarea(~7) + sep(1) + "Staged"(1) + sep(1) + statusbar(2)
	taHeight := m.commitInput.Height() + 2 // rendered lines + border
	diffHeight := m.height - 9 - taHeight
	if diffHeight < 3 {
		diffHeight = 3
	}
	m.commitDiffView.Width = m.width - 8
	m.commitDiffView.Height = diffHeight

	// Staged files header
	stagedFiles := m.stagedFilesList()
	b.WriteString("  " + catHeaderStyle.Render("Staged") + "  " + disabledStyle.Render(stagedFiles) + "\n")

	for _, line := range strings.Split(m.commitDiffView.View(), "\n") {
		b.WriteString("  " + line + "\n")
	}

	b.WriteString("  " + sep + "\n")

	// Pad
	lines := strings.Count(b.String(), "\n")
	for i := lines; i < m.height-5; i++ {
		b.WriteString("\n")
	}

	helpPairs := []string{"ctrl+d", "commit", "pgup/dn", "scroll diff", "esc", "cancel"}
	b.WriteString(components.StatusBar(m.width, helpPairs, m.statusMsg))

	return b.String()
}

// viewHistory renders the History tab (scrollable commit log).
func (m *Model) viewHistory() string {
	var b strings.Builder
	sep := strings.Repeat("─", m.width-4)

	b.WriteString("  " + sep + "\n")

	// header(3: name+info+tabs) + sep(2) + statusbar(2)
	contentHeight := m.height - 7
	if contentHeight < 3 {
		contentHeight = 3
	}
	m.commitsView.Width = m.width - 4
	m.commitsView.Height = contentHeight

	if !m.commitsLoaded {
		b.WriteString("  Loading commits...\n")
	} else {
		b.WriteString(m.commitsView.View())
	}

	// Pad
	lines := strings.Count(b.String(), "\n")
	for i := lines; i < m.height-5; i++ {
		b.WriteString("\n")
	}

	b.WriteString("  " + sep + "\n")

	helpPairs := []string{"↑↓", "scroll", "tab", "switch", "esc", "back", "q", "quit"}
	b.WriteString(components.StatusBar(m.width, helpPairs, m.statusMsg))

	return b.String()
}

// viewStash renders the Stash tab (entry list + stash diff).
func (m *Model) viewStash() string {
	var b strings.Builder
	sep := strings.Repeat("─", m.width-4)

	b.WriteString("  " + sep + "\n")

	contentHeight := m.height - 7 // header(3: name+info+tabs) + sep(2) + statusbar(2)
	if contentHeight < 3 {
		contentHeight = 3
	}

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
		visible := contentHeight - 1
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
	entryLines := strings.Count(entryList.String(), "\n")
	for i := entryLines; i < contentHeight; i++ {
		entryList.WriteString("\n")
	}

	// Stash diff (right)
	var diffPanel strings.Builder
	diffPanel.WriteString(catHeaderStyle.Render("Stash Diff") + "\n")
	m.stashDiffView.Width = diffWidth
	m.stashDiffView.Height = contentHeight - 1
	if m.stashDiffView.Height < 1 {
		m.stashDiffView.Height = 1
	}

	if len(m.stashEntries) == 0 {
		diffPanel.WriteString(disabledStyle.Render("  (no stash)") + "\n")
		for i := 1; i < contentHeight-1; i++ {
			diffPanel.WriteString("\n")
		}
	} else if !m.stashDiffLoaded {
		diffPanel.WriteString(disabledStyle.Render("  loading...") + "\n")
		for i := 1; i < contentHeight-1; i++ {
			diffPanel.WriteString("\n")
		}
	} else {
		diffPanel.WriteString(m.stashDiffView.View())
	}

	leftCol := lipgloss.NewStyle().Width(listWidth).Render(strings.TrimRight(entryList.String(), "\n"))
	rightCol := lipgloss.NewStyle().Width(diffWidth).Render(strings.TrimRight(diffPanel.String(), "\n"))
	b.WriteString("  " + lipgloss.JoinHorizontal(lipgloss.Top, leftCol, " ", rightCol))
	b.WriteString("\n  " + sep + "\n")

	// Pad
	lines := strings.Count(b.String(), "\n")
	for i := lines; i < m.height-4; i++ {
		b.WriteString("\n")
	}

	helpPairs := []string{"↑↓", "entries", "pgup/dn", "diff", "p", "pop", "d", "drop", "tab", "switch", "esc", "back", "q", "quit"}
	b.WriteString(components.StatusBar(m.width, helpPairs, m.statusMsg))

	return b.String()
}

// viewBranches renders the Branches tab as a table.
func (m *Model) viewBranches() string {
	var b strings.Builder
	sep := strings.Repeat("─", m.width-4)

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

	// Table header
	hdrStyle := lipgloss.NewStyle().Bold(true).Foreground(common.ColorMuted)
	b.WriteString(hdrStyle.Render(fmt.Sprintf("  %-*s %*s %*s %-*s %-*s %s",
		colName, "Branch",
		colAhead, "Ahead",
		colBehind, "Behind",
		colMerged, "Merged",
		colDate, "Date",
		"Message",
	)) + "\n")
	b.WriteString("  " + sep + "\n")

	// Rows
	// header(3: name+info+tabs) + table_header(1) + sep(2) + input(0-1) + statusbar(2)
	tableHeight := m.height - 8
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
		for i := m.branchScroll; i < end; i++ {
			br := m.branches[i]

			// Build row with manual padding to avoid ANSI alignment issues
			var row strings.Builder
			if i == m.branchCursor {
				row.WriteString("> ")
			} else {
				row.WriteString("  ")
			}

			// Name column
			name := truncate(br.Name, colName)
			if br.IsCurrent {
				row.WriteString(lipgloss.NewStyle().Foreground(common.ColorGreen).Bold(true).Render(name))
			} else {
				row.WriteString(name)
			}
			padCol(&row, colName, len(name))

			// Ahead column
			if br.Ahead > 0 {
				aStr := fmt.Sprintf("%d↑", br.Ahead)
				row.WriteString(lipgloss.NewStyle().Foreground(common.ColorGreen).Render(aStr))
				padCol(&row, colAhead, lipgloss.Width(aStr))
			} else {
				padCol(&row, colAhead, 0)
			}

			// Behind column
			if br.Behind > 0 {
				bStr := fmt.Sprintf("%d↓", br.Behind)
				row.WriteString(lipgloss.NewStyle().Foreground(common.ColorRed).Render(bStr))
				padCol(&row, colBehind, lipgloss.Width(bStr))
			} else {
				padCol(&row, colBehind, 0)
			}

			// Merged column
			if br.Merged && br.Name != m.Repo.DefaultBranch {
				row.WriteString(lipgloss.NewStyle().Foreground(common.ColorGreen).Render("✓"))
				padCol(&row, colMerged, 1)
			} else {
				padCol(&row, colMerged, 0)
			}

			// Date column
			row.WriteString(disabledStyle.Render(fmt.Sprintf("%-*s", colDate, br.LastDate)) + " ")

			// Message column
			row.WriteString(disabledStyle.Render(truncate(br.LastMsg, colMsg)))

			rowStr := row.String()
			if i == m.branchCursor {
				b.WriteString(lipgloss.NewStyle().Background(common.ColorSurface).Width(m.width).Render(rowStr) + "\n")
			} else {
				b.WriteString(rowStr + "\n")
			}
		}
	}

	// Pad rows
	written := strings.Count(b.String(), "\n") - 2 // minus header + sep
	for i := written; i < tableHeight; i++ {
		b.WriteString("\n")
	}

	b.WriteString("  " + sep + "\n")

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
	for i := lines; i < m.height-5; i++ {
		b.WriteString("\n")
	}

	helpPairs := []string{"↑↓", "select", "↵/s", "switch", "n", "new", "r", "rename", "d", "delete", "D", "force del", "tab", "switch tab", "esc", "back", "q", "quit"}
	b.WriteString(components.StatusBar(m.width, helpPairs, m.statusMsg))

	return b.String()
}

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


// Diff colorization

var (
	diffAddStyle  = lipgloss.NewStyle().Foreground(common.ColorGreen)
	diffRemStyle  = lipgloss.NewStyle().Foreground(common.ColorRed)
	diffHunkStyle = lipgloss.NewStyle().Foreground(common.ColorBlue).Bold(true)
	diffLineNum   = lipgloss.NewStyle().Foreground(common.ColorMuted)
	diffCtxStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#a0a0a0"))
)

var diffFileStyle = lipgloss.NewStyle().Bold(true).Foreground(common.ColorAccent).Underline(true)

func colorizeDiff(raw string) string {
	if raw == "" {
		return ""
	}
	lines := strings.Split(raw, "\n")
	var b strings.Builder
	var oldLine, newLine int

	for _, line := range lines {
		// Extract file path from diff header
		if strings.HasPrefix(line, "diff --git") {
			// "diff --git a/path b/path" → extract "path"
			parts := strings.SplitN(line, " b/", 2)
			if len(parts) == 2 {
				b.WriteString("\n" + diffFileStyle.Render(parts[1]) + "\n")
			}
			continue
		}
		if strings.HasPrefix(line, "index ") ||
			strings.HasPrefix(line, "--- ") ||
			strings.HasPrefix(line, "+++ ") ||
			strings.HasPrefix(line, "new file") ||
			strings.HasPrefix(line, "old mode") ||
			strings.HasPrefix(line, "new mode") {
			continue
		}
		if strings.HasPrefix(line, "@@") {
			old, new := parseHunkHeader(line)
			oldLine = old
			newLine = new
			ctx := ""
			if idx := strings.Index(line[2:], "@@"); idx >= 0 {
				ctx = strings.TrimSpace(line[2+idx+2:])
			}
			hunkLabel := fmt.Sprintf("─── %d:%d ", oldLine, newLine)
			if ctx != "" {
				hunkLabel += diffCtxStyle.Render(ctx) + " "
			}
			b.WriteString(diffHunkStyle.Render(hunkLabel) + "\n")
			continue
		}
		if strings.HasPrefix(line, "+") {
			gutter := diffLineNum.Render(fmt.Sprintf("%4s %4d ", "", newLine))
			b.WriteString(gutter + diffAddStyle.Render("+ "+line[1:]) + "\n")
			newLine++
		} else if strings.HasPrefix(line, "-") {
			gutter := diffLineNum.Render(fmt.Sprintf("%4d %4s ", oldLine, ""))
			b.WriteString(gutter + diffRemStyle.Render("- "+line[1:]) + "\n")
			oldLine++
		} else if strings.HasPrefix(line, "\\") {
			b.WriteString(diffLineNum.Render("          ") + diffCtxStyle.Render(line) + "\n")
		} else {
			content := line
			if len(line) > 0 && line[0] == ' ' {
				content = line[1:]
			}
			gutter := diffLineNum.Render(fmt.Sprintf("%4d %4d ", oldLine, newLine))
			b.WriteString(gutter + diffCtxStyle.Render("  "+content) + "\n")
			oldLine++
			newLine++
		}
	}
	return b.String()
}

func parseHunkHeader(line string) (oldStart, newStart int) {
	oldStart, newStart = 1, 1
	parts := strings.Fields(line)
	for _, p := range parts {
		if strings.HasPrefix(p, "-") && len(p) > 1 && p[1] >= '0' && p[1] <= '9' {
			fmt.Sscanf(p, "-%d", &oldStart)
		}
		if strings.HasPrefix(p, "+") && len(p) > 1 && p[1] >= '0' && p[1] <= '9' {
			fmt.Sscanf(p, "+%d", &newStart)
		}
	}
	return
}

func (m *Model) stagedFilesList() string {
	var paths []string
	for _, c := range m.changes {
		if c.Staged != ' ' && c.Staged != 0 && c.Staged != '?' {
			paths = append(paths, c.Path)
		}
	}
	if len(paths) == 0 {
		return "(no staged files)"
	}
	return strings.Join(paths, ", ")
}

func fileChangeIndicator(c git.ChangeInfo) string {
	subStyle := lipgloss.NewStyle().Foreground(common.ColorBlue)
	staged := lipgloss.NewStyle().Foreground(common.ColorGreen)
	unstaged := lipgloss.NewStyle().Foreground(common.ColorRed)
	untracked := lipgloss.NewStyle().Foreground(common.ColorYellow)

	if c.IsSubmodule {
		s := string(c.Staged)
		u := string(c.Unstaged)
		if c.Staged == ' ' || c.Staged == 0 {
			s = " "
		}
		if c.Unstaged == ' ' || c.Unstaged == 0 {
			u = " "
		}
		return subStyle.Render(s+u) + " " + subStyle.Render("[sub]")
	}

	if c.Staged == '?' {
		return untracked.Render("??")
	}
	s := string(c.Staged)
	u := string(c.Unstaged)
	if c.Staged == ' ' || c.Staged == 0 {
		s = " "
	} else {
		s = staged.Render(s)
	}
	if c.Unstaged == ' ' || c.Unstaged == 0 {
		u = " "
	} else {
		u = unstaged.Render(u)
	}
	return s + u
}

// padCol writes spaces to fill a column to its target width.
func padCol(b *strings.Builder, colWidth, contentWidth int) {
	pad := colWidth - contentWidth + 1
	if pad < 1 {
		pad = 1
	}
	b.WriteString(strings.Repeat(" ", pad))
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
