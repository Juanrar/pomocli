package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestBigDigitsShape(t *testing.T) {
	lines := strings.Split(BigDigits("25:00"), "\n")
	if len(lines) != 5 {
		t.Fatalf("got %d lines, want 5", len(lines))
	}
	for i, l := range lines {
		if w := lipgloss.Width(l); w != 25 {
			t.Errorf("line %d width %d, want 25", i, w)
		}
	}
}
