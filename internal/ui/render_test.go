package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/yachiko/fossor/internal/git"
	"github.com/yachiko/fossor/internal/ui/common"
	"github.com/yachiko/fossor/internal/ui/uitest"
)

// TestDiscoveryRendersPendingRowsFaint drives the app through the discovery
// messages a real scan produces and inspects the rendered frame at each stage,
// so wiring regressions between discovery and the table show up, not only
// table rendering in isolation.
func TestDiscoveryRendersPendingRowsFaint(t *testing.T) {
	profile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(profile) })
	cacheDir := git.CacheDir
	git.CacheDir = t.TempDir()
	t.Cleanup(func() { git.CacheDir = cacheDir })

	a := NewApp(nil, t.TempDir(), false, false, true, "")
	a.liveRepos = make(map[string]git.RepoInfo)
	a.localRepos = make(map[string]git.RepoInfo)
	a.Update(tea.WindowSizeMsg{Width: 120, Height: 20})
	repo := func(name string) git.RepoInfo {
		return git.RepoInfo{Path: "/" + name, Name: name, Branch: "main", DefaultBranch: "main", Behind: 1, Status: git.StatusBehind}
	}

	// Cached rows are not read yet: muted and labelled stale, not faint.
	for _, name := range []string{"alpha", "beta"} {
		a.mainScreen.AddCachedRepo(repo(name))
	}
	for _, name := range []string{"alpha", "beta"} {
		line := renderedRow(t, a, name)
		if uitest.FaintAt(line, name) || !strings.Contains(line, "stale") {
			t.Errorf("cached %s row = %q, want muted stale row without faint", name, line)
		}
	}

	// Local results arrive while fetches are pending: the whole row is faint,
	// including the live status. alpha is selected, beta is not.
	for _, name := range []string{"alpha", "beta"} {
		r := repo(name)
		a.Update(common.RepoDiscoveredMsg{Repo: r, Path: r.Path, Local: true})
	}
	for _, name := range []string{"alpha", "beta"} {
		line := renderedRow(t, a, name)
		if strings.Contains(line, "stale") || !uitest.FaintAt(line, name) || !uitest.FaintAt(line, "Behind") {
			t.Errorf("pending %s row = %q, want live status rendered faint", name, line)
		}
	}

	// A completed refresh restores normal rendering for that row only.
	alpha := repo("alpha")
	a.Update(common.RepoDiscoveredMsg{Repo: alpha, Path: alpha.Path})
	if line := renderedRow(t, a, "alpha"); uitest.FaintAt(line, "alpha") || uitest.FaintAt(line, "Behind") {
		t.Errorf("refreshed alpha row = %q, want no faint", line)
	}
	if line := renderedRow(t, a, "beta"); !uitest.FaintAt(line, "beta") {
		t.Errorf("still pending beta row = %q, want faint", line)
	}
}

func renderedRow(t *testing.T, a *App, name string) string {
	t.Helper()
	for _, line := range strings.Split(a.View(), "\n") {
		if strings.Contains(line, name) {
			return line
		}
	}
	t.Fatalf("no rendered row contains %q:\n%s", name, a.View())
	return ""
}
