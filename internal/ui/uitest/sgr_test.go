package uitest

import "testing"

func TestFaintAt(t *testing.T) {
	for _, tc := range []struct {
		line string
		want bool
	}{
		{"plain cell", false},
		{"\x1b[2mcell", true},
		{"\x1b[2;38;2;1;2;3mcell", true},
		{"\x1b[38;2;97;2;97mcell", false},
		{"\x1b[38;5;2mcell", false},
		{"\x1b[48;2;2;2;2;2mcell", true},
		{"\x1b[2mx\x1b[0m cell", false},
		{"\x1b[2mx\x1b[22m cell", false},
	} {
		if got := FaintAt(tc.line, "cell"); got != tc.want {
			t.Errorf("FaintAt(%q) = %v, want %v", tc.line, got, tc.want)
		}
	}
}
