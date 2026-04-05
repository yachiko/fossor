package tabs

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/ahoma/fossor/internal/git"
	"github.com/ahoma/fossor/internal/ui/common"
)

// CommitsTab renders the commits tab with a scrollable viewport.
type CommitsTab struct {
	Viewport viewport.Model
	Commits  []git.CommitInfo
	loaded   bool
}

func NewCommitsTab() CommitsTab {
	vp := viewport.New(80, 20)
	return CommitsTab{Viewport: vp}
}

func (t *CommitsTab) SetSize(w, h int) {
	t.Viewport.Width = w - 4
	t.Viewport.Height = h
}

func (t *CommitsTab) SetCommits(commits []git.CommitInfo) {
	t.Commits = commits
	t.loaded = true
	t.Viewport.SetContent(t.renderCommits())
	t.Viewport.GotoTop()
}

func (t *CommitsTab) renderCommits() string {
	if len(t.Commits) == 0 {
		return "  No commits found."
	}

	hashStyle := lipgloss.NewStyle().Foreground(common.ColorAccent)
	authorStyle := lipgloss.NewStyle().Foreground(common.ColorMuted)

	var b strings.Builder
	for _, c := range t.Commits {
		b.WriteString(fmt.Sprintf("  %s %s\n",
			hashStyle.Render(c.Short),
			c.Subject,
		))
		b.WriteString(fmt.Sprintf("  %s  %s\n\n",
			authorStyle.Render(c.Author),
			authorStyle.Render(c.Date.Format("2006-01-02 15:04")),
		))
	}
	return b.String()
}

func (t *CommitsTab) View() string {
	if !t.loaded {
		return "  Loading commits..."
	}
	return t.Viewport.View()
}
