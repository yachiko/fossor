package tabs

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/ahoma/fossor/internal/git"
	"github.com/ahoma/fossor/internal/ui/common"
)

// ChangesTab renders the staged/unstaged changes.
type ChangesTab struct {
	Viewport viewport.Model
	Changes  []git.ChangeInfo
	diffStat string
	loaded   bool
}

func NewChangesTab() ChangesTab {
	vp := viewport.New(80, 20)
	return ChangesTab{Viewport: vp}
}

func (t *ChangesTab) SetSize(w, h int) {
	t.Viewport.Width = w - 4
	t.Viewport.Height = h
}

func (t *ChangesTab) SetChanges(changes []git.ChangeInfo, diffStat string) {
	t.Changes = changes
	t.diffStat = diffStat
	t.loaded = true
	t.Viewport.SetContent(t.renderChanges())
	t.Viewport.GotoTop()
}

func (t *ChangesTab) renderChanges() string {
	if len(t.Changes) == 0 {
		return "  No changes."
	}

	greenStyle := lipgloss.NewStyle().Foreground(common.ColorGreen)
	redStyle := lipgloss.NewStyle().Foreground(common.ColorRed)
	yellowStyle := lipgloss.NewStyle().Foreground(common.ColorYellow)

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("  Files"))
	b.WriteString("\n\n")

	for _, c := range t.Changes {
		indicator := fmt.Sprintf("%c%c", c.Staged, c.Unstaged)
		var styledIndicator string
		switch {
		case c.Staged == '?' && c.Unstaged == '?':
			styledIndicator = yellowStyle.Render(indicator)
		case c.Staged != ' ':
			styledIndicator = greenStyle.Render(indicator)
		case c.Unstaged != ' ':
			styledIndicator = redStyle.Render(indicator)
		default:
			styledIndicator = indicator
		}
		b.WriteString(fmt.Sprintf("  %s  %s\n", styledIndicator, c.Path))
	}

	if t.diffStat != "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Bold(true).Render("  Diff"))
		b.WriteString("\n\n")
		for _, line := range strings.Split(t.diffStat, "\n") {
			if line != "" {
				b.WriteString("  " + line + "\n")
			}
		}
	}

	return b.String()
}

func (t *ChangesTab) View() string {
	if !t.loaded {
		return "  Loading changes..."
	}
	return t.Viewport.View()
}
