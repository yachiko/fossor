package manageview

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yachiko/fossor/internal/git"
)

func TestDiffResultMustMatchCurrentSelectionAndRequest(t *testing.T) {
	m := NewWithCoordinator(nil, git.RepoInfo{Path: "/repo"}, true, nil, 7)
	m.changes = []git.ChangeInfo{{Path: "old.go"}, {Path: "new.go"}}
	m.diffLoaded = true
	m.diffView.SetContent("old preview")

	m.moveFileCursor(1)
	if m.diffLoaded {
		t.Fatal("changing files must clear the previous preview while the new one loads")
	}

	m.HandleInternalMsg(diffLoadedMsg{path: "old.go", diff: "old result", session: 7, request: 0})
	if m.diffLoaded {
		t.Fatal("stale file result replaced the loading preview")
	}

	m.HandleInternalMsg(diffLoadedMsg{path: "new.go", diff: "+new result", session: 7, request: 1})
	if !m.diffLoaded || !strings.Contains(m.diffView.View(), "new result") {
		t.Fatal("current file result was not displayed")
	}
}

func TestStashResultMustMatchCurrentEntryAndRequest(t *testing.T) {
	m := NewWithCoordinator(nil, git.RepoInfo{Path: "/repo"}, true, nil, 9)
	m.stashEntries = []string{"stash@{0}: first", "stash@{1}: second"}
	m.stashDiffLoaded = true
	m.stashDiffView.SetContent("first preview")

	m.moveStashCursor(1)
	if m.stashDiffLoaded {
		t.Fatal("changing stashes must clear the previous preview while the new one loads")
	}

	m.HandleInternalMsg(stashDiffLoadedMsg{entry: "stash@{0}: first", diff: "first result", session: 9, path: "/repo", request: 0})
	if m.stashDiffLoaded {
		t.Fatal("stale stash result replaced the loading preview")
	}

	m.HandleInternalMsg(stashDiffLoadedMsg{entry: "stash@{1}: second", diff: "+second result", session: 9, path: "/repo", request: 1})
	if !m.stashDiffLoaded || !strings.Contains(m.stashDiffView.View(), "second result") {
		t.Fatal("current stash result was not displayed")
	}
}

func TestClosedSessionResultsAreIgnored(t *testing.T) {
	m := NewWithCoordinator(nil, git.RepoInfo{Path: "/repo"}, true, nil, 2)
	m.changesRequest = 1
	m.HandleInternalMsg(changesLoadedMsg{
		changes: []git.ChangeInfo{{Path: "stale.go"}},
		session: 1,
		path:    "/repo",
		request: 1,
	})
	if len(m.changes) != 0 {
		t.Fatal("result from a closed session populated the new model")
	}
}

func TestDeleteAllowsUntrackedStatusInEitherColumn(t *testing.T) {
	m := New(nil, git.RepoInfo{Path: "/repo"}, true)
	m.changes = []git.ChangeInfo{{Unstaged: '?', Path: "untracked"}}

	if cmd := m.updateStatus(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")}); cmd == nil {
		t.Fatal("uppercase X did not start deletion for an untracked path")
	}
}
