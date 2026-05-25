package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/yachiko/fossor/internal/git"
	"github.com/yachiko/fossor/internal/ui"
)

var (
	recursive     bool
	noFetch       bool
	noAutoRefresh bool
	openCmd       string
)

// Version is set at build time via -ldflags "-X github.com/yachiko/fossor/cmd.Version=...".
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:     "fossor [path]",
	Short:   "Manage multiple git repositories from a single TUI",
	Version: Version,
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rootDir, err := resolveRootDir(args)
		if err != nil {
			return err
		}

		resolvedOpenCmd := openCmd
		if resolvedOpenCmd == "" {
			resolvedOpenCmd = os.Getenv("FOSSOR_OPEN_CMD")
		}

		g := git.NewExecGit()
		app := ui.NewApp(g, rootDir, recursive, noFetch, noAutoRefresh, resolvedOpenCmd)

		p := tea.NewProgram(app, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("error running fossor: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Recursively scan for git repositories")
	rootCmd.Flags().BoolVar(&noFetch, "no-fetch", false, "Skip git fetch during discovery")
	rootCmd.Flags().BoolVar(&noAutoRefresh, "no-auto-refresh", false, "Disable periodic background refresh")
	rootCmd.Flags().StringVar(&openCmd, "open-cmd", "", "Command to open the selected repo's directory (e.g. 'code', 'cursor'). Falls back to $FOSSOR_OPEN_CMD if unset.")
}

func Execute() error {
	return rootCmd.Execute()
}

func resolveRootDir(args []string) (string, error) {
	var dir string
	if len(args) > 0 {
		dir = args[0]
	} else {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("could not get working directory: %w", err)
		}
	}

	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("could not resolve path: %w", err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("path does not exist: %s", abs)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", abs)
	}

	return abs, nil
}
