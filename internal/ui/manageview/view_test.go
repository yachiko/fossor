package manageview

import (
	"strings"
	"testing"

	"github.com/ahoma/fossor/internal/git"
)

func TestColorizeDiff(t *testing.T) {
	t.Run("empty input returns empty", func(t *testing.T) {
		got := colorizeDiff("")
		if got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})

	t.Run("diff --git line becomes file header", func(t *testing.T) {
		input := "diff --git a/foo.go b/foo.go"
		got := colorizeDiff(input)
		if !strings.Contains(got, "foo.go") {
			t.Errorf("expected output to contain file name 'foo.go', got %q", got)
		}
	})

	t.Run("add lines are present", func(t *testing.T) {
		input := "@@ -1,3 +1,4 @@\n+added line"
		got := colorizeDiff(input)
		if !strings.Contains(got, "added line") {
			t.Errorf("expected output to contain 'added line', got %q", got)
		}
		if !strings.Contains(got, "+") {
			t.Errorf("expected output to contain '+' marker, got %q", got)
		}
	})

	t.Run("remove lines are present", func(t *testing.T) {
		input := "@@ -1,3 +1,2 @@\n-removed line"
		got := colorizeDiff(input)
		if !strings.Contains(got, "removed line") {
			t.Errorf("expected output to contain 'removed line', got %q", got)
		}
		if !strings.Contains(got, "-") {
			t.Errorf("expected output to contain '-' marker, got %q", got)
		}
	})

	t.Run("hunk header with line numbers", func(t *testing.T) {
		input := "@@ -10,3 +10,5 @@ func foo()"
		got := colorizeDiff(input)
		if !strings.Contains(got, "10:10") {
			t.Errorf("expected hunk header to contain '10:10', got %q", got)
		}
	})

	t.Run("index lines are stripped", func(t *testing.T) {
		input := "index abc123..def456 100644"
		got := colorizeDiff(input)
		if strings.Contains(got, "index") {
			t.Errorf("expected index line to be stripped, got %q", got)
		}
	})

	t.Run("--- and +++ lines are stripped", func(t *testing.T) {
		input := "--- a/file.go\n+++ b/file.go"
		got := colorizeDiff(input)
		if strings.Contains(got, "---") {
			t.Errorf("expected '---' line to be stripped, got %q", got)
		}
		if strings.Contains(got, "+++") {
			t.Errorf("expected '+++' line to be stripped, got %q", got)
		}
	})
}

func TestParseHunkHeader(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantOld int
		wantNew int
	}{
		{
			name:    "standard hunk with counts",
			input:   "@@ -10,3 +10,5 @@",
			wantOld: 10,
			wantNew: 10,
		},
		{
			name:    "single line hunk",
			input:   "@@ -1 +1 @@",
			wantOld: 1,
			wantNew: 1,
		},
		{
			name:    "different start lines",
			input:   "@@ -5,10 +20,15 @@",
			wantOld: 5,
			wantNew: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOld, gotNew := parseHunkHeader(tt.input)
			if gotOld != tt.wantOld {
				t.Errorf("oldStart = %d, want %d", gotOld, tt.wantOld)
			}
			if gotNew != tt.wantNew {
				t.Errorf("newStart = %d, want %d", gotNew, tt.wantNew)
			}
		})
	}
}

func TestFileChangeIndicator(t *testing.T) {
	tests := []struct {
		name   string
		change git.ChangeInfo
		check  func(t *testing.T, result string)
	}{
		{
			name:   "untracked file",
			change: git.ChangeInfo{Staged: '?', Unstaged: '?', Path: "new.txt"},
			check: func(t *testing.T, result string) {
				if !strings.Contains(result, "?") {
					t.Errorf("expected '?' for untracked, got %q", result)
				}
			},
		},
		{
			name:   "staged only",
			change: git.ChangeInfo{Staged: 'M', Unstaged: ' ', Path: "file.go"},
			check: func(t *testing.T, result string) {
				if !strings.Contains(result, "M") {
					t.Errorf("expected 'M' for staged, got %q", result)
				}
			},
		},
		{
			name:   "unstaged only",
			change: git.ChangeInfo{Staged: ' ', Unstaged: 'M', Path: "file.go"},
			check: func(t *testing.T, result string) {
				if !strings.Contains(result, "M") {
					t.Errorf("expected 'M' for unstaged, got %q", result)
				}
			},
		},
		{
			name:   "submodule",
			change: git.ChangeInfo{Staged: 'M', Unstaged: ' ', Path: "sub", IsSubmodule: true},
			check: func(t *testing.T, result string) {
				if !strings.Contains(result, "[sub]") {
					t.Errorf("expected '[sub]' for submodule, got %q", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fileChangeIndicator(tt.change)
			tt.check(t, result)
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "shorter than max unchanged",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "at max unchanged",
			input:  "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "over max truncated with ellipsis",
			input:  "hello world",
			maxLen: 8,
			want:   "hello...",
		},
		{
			name:   "maxLen 3 truncates without ellipsis",
			input:  "hello",
			maxLen: 3,
			want:   "hel",
		},
		{
			name:   "maxLen 2 truncates without ellipsis",
			input:  "hello",
			maxLen: 2,
			want:   "he",
		},
		{
			name:   "maxLen 1 truncates to single char",
			input:  "hello",
			maxLen: 1,
			want:   "h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestParseStashEntries(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "empty string returns nil",
			input: "",
			want:  nil,
		},
		{
			name:  "whitespace only returns nil",
			input: "   \n  \t  ",
			want:  nil,
		},
		{
			name:  "single entry",
			input: "stash@{0}: WIP on main: abc1234 some message",
			want:  []string{"stash@{0}: WIP on main: abc1234 some message"},
		},
		{
			name:  "multiple entries",
			input: "stash@{0}: WIP on main: abc1234 first\nstash@{1}: WIP on main: def5678 second",
			want:  []string{"stash@{0}: WIP on main: abc1234 first", "stash@{1}: WIP on main: def5678 second"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseStashEntries(tt.input)
			if tt.want == nil {
				if got != nil {
					t.Errorf("parseStashEntries(%q) = %v, want nil", tt.input, got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("parseStashEntries(%q) returned %d entries, want %d", tt.input, len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("entry[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
