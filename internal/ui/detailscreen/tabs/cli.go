package tabs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ahoma/fossor/internal/ui/common"
)

// CLICmdMsg is sent when the user submits a command in the CLI tab.
// The detail model handles actual execution.
type CLICmdMsg struct {
	Args  []string
	Raw   string
	IsGit bool
}

// CLITab provides a pseudo-CLI for running commands in the repo directory.
type CLITab struct {
	Input     textinput.Model
	Output    viewport.Model
	History   []string
	histIdx   int
	outputBuf strings.Builder
	path      string
	focused   bool
}

func NewCLITab() CLITab {
	ti := textinput.New()
	ti.Placeholder = "command (e.g. git status, ls -la, make build)"
	ti.CharLimit = 200
	ti.Prompt = "$ "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(common.ColorAccent)

	vp := viewport.New(80, 20)
	return CLITab{
		Input:   ti,
		Output:  vp,
		histIdx: -1,
	}
}

func (t *CLITab) SetSize(w, h int) {
	t.Input.Width = w - 10
	t.Output.Width = w - 4
	t.Output.Height = h - 3
}

func (t *CLITab) SetPath(path string) {
	t.path = path
}

func (t *CLITab) Focus() tea.Cmd {
	t.focused = true
	t.Input.Focus()
	return t.Input.Cursor.BlinkCmd()
}

func (t *CLITab) Blur() {
	t.focused = false
	t.Input.Blur()
}

func (t *CLITab) IsFocused() bool {
	return t.focused
}

func (t *CLITab) AppendOutput(cmdLine, output string, err error) {
	if t.outputBuf.Len() > 0 {
		t.outputBuf.WriteString("\n")
	}

	promptStyle := lipgloss.NewStyle().Foreground(common.ColorAccent)
	t.outputBuf.WriteString(promptStyle.Render("$ "+cmdLine) + "\n")

	if err != nil {
		errStyle := lipgloss.NewStyle().Foreground(common.ColorRed)
		t.outputBuf.WriteString(errStyle.Render(fmt.Sprintf("error: %v", err)) + "\n")
	} else if output != "" {
		t.outputBuf.WriteString(output + "\n")
	}

	t.Output.SetContent(t.outputBuf.String())
	t.Output.GotoBottom()
}

func (t *CLITab) Update(msg tea.Msg) tea.Cmd {
	if !t.focused {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			raw := strings.TrimSpace(t.Input.Value())
			if raw == "" {
				return nil
			}
			t.History = append(t.History, raw)
			t.histIdx = len(t.History)
			t.Input.SetValue("")

			args := strings.Fields(raw)
			isGit := false
			if len(args) > 0 && args[0] == "git" {
				isGit = true
				args = args[1:] // strip "git" prefix — RunCommand already adds it
			}
			return func() tea.Msg {
				return CLICmdMsg{Args: args, Raw: raw, IsGit: isGit}
			}

		case "tab":
			t.handleCompletion()
			return nil

		case "up":
			if len(t.History) > 0 && t.histIdx > 0 {
				t.histIdx--
				t.Input.SetValue(t.History[t.histIdx])
				t.Input.CursorEnd()
			}
			return nil

		case "down":
			if t.histIdx < len(t.History)-1 {
				t.histIdx++
				t.Input.SetValue(t.History[t.histIdx])
				t.Input.CursorEnd()
			} else {
				t.histIdx = len(t.History)
				t.Input.SetValue("")
			}
			return nil
		}
	}

	var cmd tea.Cmd
	t.Input, cmd = t.Input.Update(msg)
	return cmd
}

func (t *CLITab) handleCompletion() {
	value := t.Input.Value()
	pos := t.Input.Position()
	if pos == 0 {
		return
	}

	// Get text up to cursor
	textToCursor := value[:pos]
	afterCursor := value[pos:]

	// Find the last word (the token being completed)
	lastSpace := strings.LastIndex(textToCursor, " ")
	var prefix string
	if lastSpace >= 0 {
		prefix = textToCursor[lastSpace+1:]
	} else {
		prefix = textToCursor
	}
	if prefix == "" {
		return
	}

	// Resolve directory and base for completion
	dir := t.path
	base := prefix
	if i := strings.LastIndex(prefix, "/"); i >= 0 {
		dir = filepath.Join(t.path, prefix[:i])
		base = prefix[i+1:]
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	var matches []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(strings.ToLower(name), strings.ToLower(base)) {
			if e.IsDir() {
				matches = append(matches, name+"/")
			} else {
				matches = append(matches, name)
			}
		}
	}

	if len(matches) == 0 {
		return
	}

	beforeWord := textToCursor[:lastSpace+1]

	if len(matches) == 1 {
		// Single match — auto-complete
		completion := matches[0]
		if i := strings.LastIndex(prefix, "/"); i >= 0 {
			completion = prefix[:i+1] + completion
		}
		newValue := beforeWord + completion + afterCursor
		t.Input.SetValue(newValue)
		t.Input.SetCursor(len(beforeWord) + len(completion))
	} else {
		// Multiple matches — complete to common prefix and show options
		cp := commonPrefix(matches)
		if len(cp) > len(base) {
			completion := cp
			if i := strings.LastIndex(prefix, "/"); i >= 0 {
				completion = prefix[:i+1] + completion
			}
			newValue := beforeWord + completion + afterCursor
			t.Input.SetValue(newValue)
			t.Input.SetCursor(len(beforeWord) + len(completion))
		}
		// Show matches in output
		hintStyle := lipgloss.NewStyle().Foreground(common.ColorMuted)
		t.outputBuf.WriteString(hintStyle.Render(strings.Join(matches, "  ")) + "\n")
		t.Output.SetContent(t.outputBuf.String())
		t.Output.GotoBottom()
	}
}

func commonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	prefix := strs[0]
	for _, s := range strs[1:] {
		for i := 0; i < len(prefix) && i < len(s); i++ {
			if prefix[i] != s[i] {
				prefix = prefix[:i]
				break
			}
		}
		if len(s) < len(prefix) {
			prefix = prefix[:len(s)]
		}
	}
	return prefix
}

func (t *CLITab) View() string {
	var b strings.Builder
	b.WriteString(t.Output.View())
	b.WriteString("\n")
	b.WriteString("  " + t.Input.View())
	return b.String()
}
