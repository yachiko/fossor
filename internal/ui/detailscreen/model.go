package detailscreen

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ahoma/fossor/internal/git"
	"github.com/ahoma/fossor/internal/ui/detailscreen/tabs"
)

const (
	TabInfo     = 0
	TabCommits  = 1
	TabChanges  = 2
	TabCLI      = 3
	NumTabs     = 4
)

var tabNames = [NumTabs]string{"Info", "Commits", "Changes", "CLI"}

// Model is the detail screen model.
type Model struct {
	Repo       git.RepoInfo
	Git        git.Git
	activeTab  int
	infoTab    tabs.InfoTab
	commitsTab tabs.CommitsTab
	changesTab tabs.ChangesTab
	cliTab     tabs.CLITab
	width      int
	height     int
	statusMsg  string
}

// New creates a new detail screen model.
func New(g git.Git, repo git.RepoInfo) Model {
	ct := tabs.NewCommitsTab()
	cht := tabs.NewChangesTab()
	clit := tabs.NewCLITab()
	clit.SetPath(repo.Path)
	return Model{
		Repo:       repo,
		Git:        g,
		commitsTab: ct,
		changesTab: cht,
		cliTab:     clit,
	}
}

func (m *Model) Init() tea.Cmd {
	return m.loadTabData()
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	tabContentHeight := h - 6
	m.commitsTab.SetSize(w, tabContentHeight)
	m.changesTab.SetSize(w, tabContentHeight)
	m.cliTab.SetSize(w, tabContentHeight)
}

func (m *Model) SetStatus(msg string) {
	m.statusMsg = msg
}

func (m *Model) UpdateRepo(repo git.RepoInfo) {
	m.Repo = repo
}
