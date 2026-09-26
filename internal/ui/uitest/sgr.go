// Package uitest holds helpers for asserting on rendered terminal output.
package uitest

import "strings"

// FaintAt reports whether the first occurrence of cell in line is drawn with
// the SGR faint attribute, by replaying the SGR sequences that precede it.
func FaintAt(line, cell string) bool {
	i := strings.Index(line, cell)
	if i < 0 {
		return false
	}
	s, faint := line[:i], false
	for {
		start := strings.Index(s, "\x1b[")
		if start < 0 {
			return faint
		}
		s = s[start+2:]
		end := strings.IndexByte(s, 'm')
		if end < 0 {
			return faint
		}
		params := strings.Split(s[:end], ";")
		for j := 0; j < len(params); j++ {
			switch params[j] {
			case "", "0", "22":
				faint = false
			case "2":
				faint = true
			case "38", "48", "58":
				// Skip extended colour arguments: 5;n or 2;r;g;b.
				if j+1 < len(params) && params[j+1] == "5" {
					j += 2
				} else if j+1 < len(params) && params[j+1] == "2" {
					j += 4
				}
			}
		}
		s = s[end+1:]
	}
}
