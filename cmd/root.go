package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/ahoma/fossor/internal/git"
	"github.com/ahoma/fossor/internal/ui"
)

var (
	recursive bool
	noFetch   bool
)

var rootCmd = &cobra.Command{
	Use:   "fossor [path]",
	Short: "Manage multiple git repositories from a single TUI",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rootDir, err := resolveRootDir(args)
		if err != nil {
			return err
		}

		g := git.NewExecGit()
		app := ui.NewApp(g, rootDir, recursive, noFetch)

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
