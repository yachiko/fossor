package common

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestStatusColor(t *testing.T) {
	tests := []struct {
		status string
		want   lipgloss.Color
	}{
		{"Up to date", ColorGreen},
		{"Behind", ColorYellow},
		{"Dirty", ColorYellow},
		{"Diverged", ColorYellow},
		{"Non-default", ColorRed},
		{"Error", ColorRed},
		{"Ahead", ColorBlue},
		{"unknown", ColorMuted},
		{"", ColorMuted},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := StatusColor(tt.status)
			if got != tt.want {
				t.Errorf("StatusColor(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}
