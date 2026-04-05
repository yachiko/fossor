package tabs

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ahoma/fossor/internal/git"
	"github.com/ahoma/fossor/internal/ui/common"
)

var (
	labelStyle = lipgloss.NewStyle().
			Foreground(common.ColorMuted).
			Width(16)

	valueStyle = lipgloss.NewStyle().
			Foreground(common.ColorWhite)
)

// InfoTab renders the info tab content.
type InfoTab struct{}

func (t InfoTab) View(repo git.RepoInfo, width, height int) string {
	var b strings.Builder

	row := func(label, value string) {
		b.WriteString("  " + labelStyle.Render(label) + valueStyle.Render(value) + "\n")
	}

	b.WriteString("\n")
	row("Name:", repo.Name)
	row("Path:", repo.Path)
	row("Branch:", repo.Branch)
	row("Default Branch:", repo.DefaultBranch)
	row("Remote:", repo.Remote)
	b.WriteString("\n")

	statusColor := common.StatusColor(repo.Status.String())
	statusStyled := lipgloss.NewStyle().Foreground(statusColor).Bold(true).Render(repo.Status.String())
	b.WriteString("  " + labelStyle.Render("Status:") + statusStyled + "\n")

	row("Ahead:", fmt.Sprintf("%d", repo.Ahead))
	row("Behind:", fmt.Sprintf("%d", repo.Behind))
	row("Changes:", fmt.Sprintf("%d", repo.Changes))

	if repo.Error != nil {
		b.WriteString("\n")
		errStyle := lipgloss.NewStyle().Foreground(common.ColorRed)
		b.WriteString("  " + errStyle.Render("Error: "+repo.Error.Error()) + "\n")
	}

	return b.String()
}
