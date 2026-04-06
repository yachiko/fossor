package manageview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/ahoma/fossor/internal/git"
	"github.com/ahoma/fossor/internal/ui/common"
)

// Shared styles

var (
	enabledKeyStyle  = lipgloss.NewStyle().Foreground(common.ColorAccent).Bold(true)
	enabledNameStyle = lipgloss.NewStyle().Foreground(common.ColorWhite)
	dangerKeyStyle   = lipgloss.NewStyle().Foreground(common.ColorRed).Bold(true)
	disabledStyle    = lipgloss.NewStyle().Foreground(common.ColorMuted)
	catHeaderStyle   = lipgloss.NewStyle().Bold(true).Foreground(common.ColorAccent)
	warnStyle        = lipgloss.NewStyle().Foreground(common.ColorRed).Bold(true)
)

// Diff styles

var (
	diffAddStyle  = lipgloss.NewStyle().Foreground(common.ColorGreen)
	diffRemStyle  = lipgloss.NewStyle().Foreground(common.ColorRed)
	diffHunkStyle = lipgloss.NewStyle().Foreground(common.ColorBlue).Bold(true)
	diffLineNum   = lipgloss.NewStyle().Foreground(common.ColorMuted)
	diffCtxStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#a0a0a0"))
	diffFileStyle = lipgloss.NewStyle().Bold(true).Foreground(common.ColorAccent).Underline(true)
)

// colorizeDiff applies syntax highlighting to raw diff output.
func colorizeDiff(raw string) string {
	if raw == "" {
		return ""
	}
	lines := strings.Split(raw, "\n")
	var b strings.Builder
	var oldLine, newLine int

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") {
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

// parseHunkHeader extracts old and new start line numbers from "@@ -old,len +new,len @@".
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

// stagedFilesList returns a comma-separated list of staged file paths.
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

// fileChangeIndicator renders a colored 2-char status indicator for a file change.
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

// truncate shortens a string with "..." if it exceeds maxLen.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
