package manageview

import (
	"strings"

	"github.com/yachiko/fossor/internal/ui/components"
)

// viewCommit renders the inline commit message editor with staged diff.
func (m *Model) viewCommit() string {
	var b strings.Builder
	sep := separator(m.width)

	b.WriteString(sep + "\n")
	b.WriteString("  " + catHeaderStyle.Render("Commit message") + "\n")
	m.commitInput.SetWidth(m.width - 6)
	for _, line := range strings.Split(m.commitInput.View(), "\n") {
		b.WriteString("  " + line + "\n")
	}
	b.WriteString(sep + "\n")

	// Staged diff below
	taHeight := m.commitInput.Height() + 2
	diffHeight := m.contentHeight(3*sepLines+1+taHeight) - 1 // -1 for "Staged" header
	if diffHeight < 3 {
		diffHeight = 3
	}
	m.commitDiffView.Width = m.width - 8
	m.commitDiffView.Height = diffHeight

	stagedFiles := m.stagedFilesList()
	b.WriteString("  " + catHeaderStyle.Render("Staged") + "  " + disabledStyle.Render(stagedFiles) + "\n")

	for _, line := range strings.Split(m.commitDiffView.View(), "\n") {
		b.WriteString("  " + line + "\n")
	}

	b.WriteString(sep + "\n")

	// Pad
	lines := strings.Count(b.String(), "\n")
	target := m.height - headerLines - statusBarLines
	for i := lines; i < target; i++ {
		b.WriteString("\n")
	}

	helpPairs := []string{"ctrl+d", "commit", "pgup/dn", "scroll diff", "esc", "cancel"}
	b.WriteString(components.StatusBar(m.width, helpPairs, m.statusMsg))

	return b.String()
}
