package manageview

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/yachiko/fossor/internal/ui/common"
)

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch m.mode {
	case modeConfirm:
		return m.updateConfirm(msg)
	case modeInput:
		return m.updateInput(msg)
	case modeCommit:
		return m.updateCommit(msg)
	default:
		return m.updateNormal(msg)
	}
}

// HandleInternalMsg processes internal messages. Returns true if handled.
func (m *Model) HandleInternalMsg(msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case remoteOperationReadyMsg:
		if msg.session != m.sessionID {
			msg.release()
			return true, nil
		}
		return true, tea.ExecProcess(msg.cmd, func(err error) tea.Msg {
			msg.release()
			return execFinishedMsg{action: msg.action, err: err, session: msg.session}
		})
	case execFinishedMsg:
		if msg.session != m.sessionID {
			return true, nil
		}
		m.lastAction = msg.action
		m.lastErr = msg.err
		if msg.err != nil {
			m.lastOutput = msg.err.Error()
			if msg.action == "pull" || msg.action == "fetch" {
				m.remoteErrorAfterRefresh = true
			}
		} else {
			m.lastOutput = ""
			if msg.action == "pull" || msg.action == "fetch" {
				m.verifyAfterRefresh = true
			}
		}
		reloadCommits := m.commitsLoaded
		reloadBranches := m.branchesLoaded
		m.commitsLoaded = false
		m.branchesLoaded = false
		m.stashDiffLoaded = false
		m.diffLoaded = false
		m.diffView.SetContent("")
		m.stashDiffView.SetContent("")
		cmds := []tea.Cmd{m.refreshRepo(), m.refreshStash(), m.loadChanges()}
		if reloadCommits && m.activeTab == TabHistory {
			cmds = append(cmds, m.loadCommits())
		}
		if reloadBranches && m.activeTab == TabBranches {
			cmds = append(cmds, m.loadBranches())
		}
		return true, tea.Batch(cmds...)
	case stashInfoMsg:
		if !m.matchesRequest(msg.session, msg.path, msg.request, m.stashRequest) {
			return true, nil
		}
		m.stashInfo = msg.info
		m.stashEntries = parseStashEntries(msg.info)
		if m.stashCursor >= len(m.stashEntries) {
			m.stashCursor = max(0, len(m.stashEntries)-1)
		}
		m.stashDiffLoaded = false
		m.stashDiffView.SetContent("")
		if m.activeTab == TabStash && len(m.stashEntries) > 0 {
			return true, m.loadStashDiff(m.stashCursor)
		}
		return true, nil
	case repoRefreshedMsg:
		if !m.matchesRequest(msg.session, msg.path, msg.request, m.repoRequest) {
			return true, nil
		}
		m.Repo = msg.repo
		if msg.verificationKnown {
			m.verified = msg.verified
		}
		return true, func() tea.Msg {
			return common.RepoUpdatedMsg{Repo: msg.repo, Verified: msg.verified, RemoteError: msg.remoteError, Source: common.RepoUpdateUserAction}
		}
	case changesLoadedMsg:
		if !m.matchesRequest(msg.session, msg.path, msg.request, m.changesRequest) {
			return true, nil
		}
		m.changes = msg.changes
		if m.fileCursor >= len(m.changes) {
			m.fileCursor = max(0, len(m.changes)-1)
		}
		if c, ok := m.selectedChange(); ok {
			m.diffLoaded = false
			m.diffView.SetContent("")
			return true, m.loadDiff(c)
		}
		m.diffLoaded = false
		m.diffView.SetContent("")
		return true, nil
	case diffLoadedMsg:
		if !m.matchesRequest(msg.session, m.Repo.Path, msg.request, m.diffRequest) || msg.path != m.selectedFilePath() {
			return true, nil
		}
		m.diffLoaded = true
		m.diffView.SetContent(colorizeDiff(msg.diff))
		m.diffView.GotoTop()
		return true, nil
	case commitsLoadedMsg:
		if !m.matchesRequest(msg.session, msg.path, msg.request, m.commitsRequest) {
			return true, nil
		}
		m.commits = msg.commits
		m.commitsLoaded = true
		m.commitsView.SetContent(renderCommits(msg.commits))
		m.commitsView.GotoTop()
		return true, nil
	case remoteLoadedMsg:
		if !m.matchesRequest(msg.session, msg.path, msg.request, m.remoteRequest) {
			return true, nil
		}
		m.remote = msg.remote
		return true, nil
	case stashDiffLoadedMsg:
		if !m.matchesRequest(msg.session, msg.path, msg.request, m.stashDiffRequest) || msg.entry != m.selectedStashEntry() {
			return true, nil
		}
		m.stashDiffLoaded = true
		m.stashDiffView.SetContent(colorizeDiff(msg.diff))
		m.stashDiffView.GotoTop()
		return true, nil
	case branchesLoadedMsg:
		if !m.matchesRequest(msg.session, msg.path, msg.request, m.branchesRequest) {
			return true, nil
		}
		m.branches = msg.branches
		m.branchesLoaded = true
		if m.branchCursor >= len(m.branches) {
			m.branchCursor = max(0, len(m.branches)-1)
		}
		return true, nil
	case stagedDiffLoadedMsg:
		if !m.matchesRequest(msg.session, msg.path, msg.request, m.stagedDiffRequest) {
			return true, nil
		}
		m.commitDiffView.SetContent(colorizeDiff(msg.diff))
		m.commitDiffView.GotoTop()
		return true, nil
	}
	return false, nil
}

func (m *Model) matchesRequest(session uint64, path string, request, current uint64) bool {
	return session == m.sessionID && path == m.Repo.Path && request == current
}

func (m *Model) updateNormal(msg tea.Msg) tea.Cmd {
	kmsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}

	key := kmsg.String()

	// Global keys
	switch key {
	case "q":
		return tea.Quit
	case "esc":
		return func() tea.Msg { return common.SwitchToMainMsg{} }
	case "tab":
		return m.switchTab((m.activeTab + 1) % NumTabs)
	case "shift+tab":
		return m.switchTab((m.activeTab - 1 + NumTabs) % NumTabs)
	}

	// Tab number keys (only when no action conflict on non-Status tabs)
	if m.activeTab != TabStatus {
		switch key {
		case "1":
			return m.switchTab(TabStatus)
		case "2":
			return m.switchTab(TabHistory)
		case "3":
			return m.switchTab(TabStash)
		case "4":
			return m.switchTab(TabBranches)
		}
	}

	// Per-tab key handling
	if !m.verified && m.activeTab != TabStatus {
		return nil
	}
	switch m.activeTab {
	case TabStatus:
		return m.updateStatus(kmsg)
	case TabHistory:
		return m.updateHistory(kmsg)
	case TabStash:
		return m.updateStash(kmsg)
	case TabBranches:
		return m.updateBranches(kmsg)
	}
	return nil
}

func (m *Model) switchTab(tab int) tea.Cmd {
	m.activeTab = tab
	switch tab {
	case TabHistory:
		if !m.commitsLoaded {
			return m.loadCommits()
		}
	case TabStash:
		if len(m.stashEntries) > 0 && !m.stashDiffLoaded {
			return m.loadStashDiff(m.stashCursor)
		}
	case TabBranches:
		if !m.branchesLoaded {
			return m.loadBranches()
		}
	}
	return nil
}

// Status tab: action keys + file navigation
func (m *Model) updateStatus(msg tea.KeyMsg) tea.Cmd {
	key := msg.String()
	if !m.verified && key != "p" && key != "f" {
		return nil
	}

	switch key {
	case "up":
		return m.moveFileCursor(-1)
	case "down":
		return m.moveFileCursor(1)
	case "pgup", "pgdown":
		var cmd tea.Cmd
		m.diffView, cmd = m.diffView.Update(msg)
		return cmd
	case "x":
		// Restore selected: git checkout -- <path> (tracked, non-submodule files only)
		if c, ok := m.selectedChange(); ok && c.Staged != '?' && !c.IsSubmodule {
			cmd := gitCmd(m.Repo.Path, "checkout", "--", c.Path)
			return tea.ExecProcess(cmd, m.execFinished("restore "+c.Path))
		}
		return nil
	case "X":
		// Delete selected untracked path, including an untracked directory.
		if c, ok := m.selectedChange(); ok && (c.Staged == '?' || c.Unstaged == '?') && !c.IsSubmodule {
			cmd := gitCmd(m.Repo.Path, "clean", "-d", "-f", "--", c.Path)
			return tea.ExecProcess(cmd, m.execFinished("delete "+c.Path))
		}
		return nil
	}

	// Inline commit mode
	if key == "c" && m.Repo.Changes > 0 {
		m.mode = modeCommit
		m.commitInput.Reset()
		m.commitInput.SetWidth(m.width - 8)
		m.commitInput.Focus()
		return m.loadStagedDiff()
	}

	// Action dispatch
	idx, ok := m.keyMap[key]
	if !ok {
		return nil
	}
	action := m.actions[idx]
	if !m.verified && action.Name != "pull" && action.Name != "fetch" {
		return nil
	}
	if !action.Enabled(m.Repo) || action.BuildCmd == nil {
		return nil
	}

	if action.Dangerous {
		m.mode = modeConfirm
		m.pendingIdx = idx
		return nil
	}

	if action.UsesSelected {
		p := m.selectedFilePath()
		if p == "" {
			return nil
		}
		return m.executeAction(action, p)
	}

	if action.NeedsInput {
		m.mode = modeInput
		m.pendingIdx = idx
		m.textInput.Placeholder = action.InputPrompt
		m.textInput.SetValue("")
		m.textInput.Focus()
		return m.textInput.Cursor.BlinkCmd()
	}

	return m.executeAction(action, "")
}

// History tab: viewport scrolling
func (m *Model) updateHistory(msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd
	m.commitsView, cmd = m.commitsView.Update(msg)
	return cmd
}

// Stash tab: entry navigation + diff scrolling + pop/drop
func (m *Model) updateStash(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up":
		return m.moveStashCursor(-1)
	case "down":
		return m.moveStashCursor(1)
	case "pgup", "pgdown":
		var cmd tea.Cmd
		m.stashDiffView, cmd = m.stashDiffView.Update(msg)
		return cmd
	case "p":
		if len(m.stashEntries) > 0 {
			ref := fmt.Sprintf("stash@{%d}", m.stashCursor)
			cmd := gitCmd(m.Repo.Path, "stash", "pop", ref)
			return tea.ExecProcess(cmd, m.execFinished("stash pop "+ref))
		}
	case "d":
		if len(m.stashEntries) > 0 {
			ref := fmt.Sprintf("stash@{%d}", m.stashCursor)
			cmd := gitCmd(m.Repo.Path, "stash", "drop", ref)
			return tea.ExecProcess(cmd, m.execFinished("stash drop "+ref))
		}
	}
	return nil
}

// Branches tab: navigation + actions
func (m *Model) updateBranches(msg tea.KeyMsg) tea.Cmd {
	// Branch input mode (create/rename)
	if m.branchInputMode {
		switch msg.String() {
		case "esc":
			m.branchInputMode = false
			m.branchInput.Blur()
			return nil
		case "enter":
			name := strings.TrimSpace(m.branchInput.Value())
			if name == "" {
				return nil
			}
			m.branchInputMode = false
			m.branchInput.Blur()

			var cmd *exec.Cmd
			var action string
			if m.branchInputAction == "create" {
				cmd = gitRefCmd(m.Repo.Path, []string{"branch"}, name)
				action = "create branch " + name
			} else if m.branchInputAction == "rename" && len(m.branches) > 0 {
				old := m.branches[m.branchCursor].Name
				// `--` separator: both old and new names are user-typed.
				cmd = exec.Command("git", "-C", m.Repo.Path, "branch", "-m", "--", old, name)
				action = "rename " + old + " → " + name
			}
			if cmd != nil {
				return tea.ExecProcess(cmd, m.execFinished(action))
			}
			return nil
		}
		var c tea.Cmd
		m.branchInput, c = m.branchInput.Update(msg)
		return c
	}

	switch msg.String() {
	case "up":
		m.moveBranchCursor(-1)
		return nil
	case "down":
		m.moveBranchCursor(1)
		return nil
	case "enter", "s":
		// Switch to selected branch
		if len(m.branches) > 0 && !m.branches[m.branchCursor].IsCurrent {
			name := m.branches[m.branchCursor].Name
			cmd := gitRefCmd(m.Repo.Path, []string{"switch"}, name)
			return tea.ExecProcess(cmd, m.execFinished("switch "+name))
		}
	case "n":
		// Create new branch
		m.branchInputMode = true
		m.branchInputAction = "create"
		m.branchInput.Placeholder = "New branch name..."
		m.branchInput.SetValue("")
		m.branchInput.Focus()
		return m.branchInput.Cursor.BlinkCmd()
	case "r":
		// Rename selected branch
		if len(m.branches) > 0 {
			m.branchInputMode = true
			m.branchInputAction = "rename"
			m.branchInput.Placeholder = "New name for " + m.branches[m.branchCursor].Name + "..."
			m.branchInput.SetValue("")
			m.branchInput.Focus()
			return m.branchInput.Cursor.BlinkCmd()
		}
	case "d":
		// Delete branch (safe — fails on unmerged)
		if len(m.branches) > 0 && !m.branches[m.branchCursor].IsCurrent {
			name := m.branches[m.branchCursor].Name
			cmd := gitRefCmd(m.Repo.Path, []string{"branch", "-d"}, name)
			return tea.ExecProcess(cmd, m.execFinished("delete "+name))
		}
	case "D":
		// Force delete branch
		if len(m.branches) > 0 && !m.branches[m.branchCursor].IsCurrent {
			name := m.branches[m.branchCursor].Name
			cmd := gitRefCmd(m.Repo.Path, []string{"branch", "-D"}, name)
			return tea.ExecProcess(cmd, m.execFinished("force delete "+name))
		}
	}
	return nil
}

func (m *Model) moveBranchCursor(delta int) {
	if len(m.branches) == 0 {
		return
	}
	m.branchCursor += delta
	if m.branchCursor < 0 {
		m.branchCursor = 0
	}
	if m.branchCursor >= len(m.branches) {
		m.branchCursor = len(m.branches) - 1
	}
}

func (m *Model) moveFileCursor(delta int) tea.Cmd {
	if len(m.changes) == 0 {
		return nil
	}
	prev := m.fileCursor
	m.fileCursor += delta
	if m.fileCursor < 0 {
		m.fileCursor = 0
	}
	if m.fileCursor >= len(m.changes) {
		m.fileCursor = len(m.changes) - 1
	}
	if m.fileCursor != prev {
		if c, ok := m.selectedChange(); ok {
			m.diffLoaded = false
			m.diffView.SetContent("")
			return m.loadDiff(c)
		}
	}
	return nil
}

func (m *Model) moveStashCursor(delta int) tea.Cmd {
	if len(m.stashEntries) == 0 {
		return nil
	}
	prev := m.stashCursor
	m.stashCursor += delta
	if m.stashCursor < 0 {
		m.stashCursor = 0
	}
	if m.stashCursor >= len(m.stashEntries) {
		m.stashCursor = len(m.stashEntries) - 1
	}
	if m.stashCursor != prev {
		m.stashDiffLoaded = false
		m.stashDiffView.SetContent("")
		return m.loadStashDiff(m.stashCursor)
	}
	return nil
}

func (m *Model) selectedStashEntry() string {
	if m.stashCursor < 0 || m.stashCursor >= len(m.stashEntries) {
		return ""
	}
	return m.stashEntries[m.stashCursor]
}

func (m *Model) updateCommit(msg tea.Msg) tea.Cmd {
	kmsg, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		m.commitInput, cmd = m.commitInput.Update(msg)
		return cmd
	}

	switch kmsg.String() {
	case "esc":
		m.mode = modeNormal
		m.commitInput.Blur()
		return nil
	case "ctrl+d":
		message := strings.TrimSpace(m.commitInput.Value())
		if message == "" {
			m.statusMsg = "commit: empty message"
			return nil
		}
		m.mode = modeNormal
		m.commitInput.Blur()
		cmd := gitCmd(m.Repo.Path, "commit", "-m", message)
		return tea.ExecProcess(cmd, m.execFinished("commit"))
	case "pgup", "pgdown":
		var cmd tea.Cmd
		m.commitDiffView, cmd = m.commitDiffView.Update(kmsg)
		return cmd
	}

	var cmd tea.Cmd
	m.commitInput, cmd = m.commitInput.Update(msg)
	return cmd
}

func (m *Model) updateConfirm(msg tea.Msg) tea.Cmd {
	kmsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}
	switch kmsg.String() {
	case "y", "Y":
		action := m.actions[m.pendingIdx]
		m.mode = modeNormal
		if action.NeedsInput {
			m.mode = modeInput
			m.textInput.Placeholder = action.InputPrompt
			m.textInput.SetValue("")
			m.textInput.Focus()
			return m.textInput.Cursor.BlinkCmd()
		}
		return m.executeAction(action, "")
	default:
		m.mode = modeNormal
		return nil
	}
}

func (m *Model) updateInput(msg tea.Msg) tea.Cmd {
	kmsg, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return cmd
	}
	switch kmsg.String() {
	case "esc":
		m.mode = modeNormal
		m.textInput.Blur()
		return nil
	case "enter":
		input := m.textInput.Value()
		action := m.actions[m.pendingIdx]
		m.mode = modeNormal
		m.textInput.Blur()
		if input == "" && action.Key == "k" {
			m.statusMsg = action.Name + ": input required"
			return nil
		}
		return m.executeAction(action, input)
	}
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return cmd
}

func (m *Model) executeAction(action Action, input string) tea.Cmd {
	cmd := action.BuildCmd(m.Repo, input)
	actionName := action.Name
	if action.Category == CatRemote && m.Coordinator != nil {
		coordinator := m.Coordinator
		key := m.Repo.CoordinatorKey()
		return func() tea.Msg {
			_, ran, release, err := coordinator.Acquire(context.Background(), key, true)
			if err != nil || !ran {
				return execFinishedMsg{action: actionName, err: err, session: m.sessionID}
			}
			return remoteOperationReadyMsg{action: actionName, cmd: cmd, release: release, session: m.sessionID}
		}
	}
	return tea.ExecProcess(cmd, m.execFinished(actionName))
}

// parseStashEntries splits stash list output into individual entries.
func parseStashEntries(info string) []string {
	if info == "" {
		return nil
	}
	trimmed := strings.TrimSpace(info)
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}
