package manageview

import (
	"errors"
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
		t.Fatal("changing files must clear the previous preview")
	}
	m.HandleInternalMsg(diffLoadedMsg{path: "old.go", diff: "old", session: 7, request: 0})
	if m.diffLoaded {
		t.Fatal("stale file result replaced the loading preview")
	}
	m.HandleInternalMsg(diffLoadedMsg{path: "new.go", diff: "+new", session: 7, request: 1})
	if !m.diffLoaded {
		t.Fatal("current file result was not displayed")
	}
}

func TestDeleteAllowsUntrackedStatusInEitherColumn(t *testing.T) {
	m := New(nil, git.RepoInfo{Path: "/repo"}, true)
	m.changes = []git.ChangeInfo{{Unstaged: '?', Path: "untracked"}}
	if cmd := m.updateStatus(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")}); cmd == nil {
		t.Fatal("uppercase X did not start deletion for an untracked path")
	}
}

func TestLoaderErrorsDoNotAppearAsEmptyData(t *testing.T) {
	m := New(nil, git.RepoInfo{Path: "/repo"}, true)
	m.SetContext(m.ctx, 3)
	m.changesRequest = 1
	m.HandleInternalMsg(changesLoadedMsg{err: errors.New("status failed"), session: 3, path: "/repo", request: 1})
	if m.changesErr == nil || len(m.changes) != 0 {
		t.Fatalf("changes error was not retained: err=%v changes=%v", m.changesErr, m.changes)
	}

	m.changes = []git.ChangeInfo{{Path: "old.go"}, {Path: "new.go"}}
	m.diffRequest = 2
	m.fileCursor = 1
	m.HandleInternalMsg(diffLoadedMsg{path: "old.go", diff: "old", session: 3, request: 1})
	if m.diffLoaded {
		t.Fatal("stale diff populated the current selection")
	}
	m.HandleInternalMsg(diffLoadedMsg{path: "new.go", err: errors.New("diff failed"), session: 3, request: 2})
	if m.diffErr == nil || m.diffLoaded {
		t.Fatal("failed diff was treated as a loaded empty diff")
	}
}

func TestClosedSessionLoaderResultsAreIgnored(t *testing.T) {
	m := New(nil, git.RepoInfo{Path: "/repo"}, true)
	m.SetContext(m.ctx, 2)
	m.commitsRequest = 1
	m.HandleInternalMsg(commitsLoadedMsg{commits: []git.CommitInfo{{Subject: "stale"}}, session: 1, path: "/repo", request: 1})
	if len(m.commits) != 0 {
		t.Fatal("closed-session result populated the model")
	}
}

func TestBranchComparisonErrorIsVisible(t *testing.T) {
	m := New(nil, git.RepoInfo{Path: "/repo"}, true)
	m.SetSize(120, 30)
	m.activeTab = TabBranches
	m.branchesLoaded = true
	m.branches = []git.BranchInfo{{Name: "feature", ComparisonError: errors.New("missing default branch")}}
	if view := m.View(); !strings.Contains(view, "?") {
		t.Fatal("unavailable comparison rendered as zero")
	}
}
