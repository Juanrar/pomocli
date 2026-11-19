package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestBigDigitsShape(t *testing.T) {
	lines := strings.Split(BigDigits("25:00"), "\n")
	if len(lines) != glyphRows || glyphRows != 6 {
		t.Fatalf("got %d lines, want 6", len(lines))
	}
	want := 4*digitCell + colonCell + 4*glyphGap
	if want != bigDigitsW {
		t.Fatalf("bigDigitsW = %d, want %d", bigDigitsW, want)
	}
	for i, l := range lines {
		if w := lipgloss.Width(l); w != want {
			t.Errorf("line %d width %d, want %d", i, w, want)
		}
	}
}

func TestBigDigitsSameWidthForEveryTime(t *testing.T) {
	for _, s := range []string{"00:00", "11:11", "88:88", "59:59", "07:42"} {
		for i, l := range strings.Split(BigDigits(s), "\n") {
			if w := lipgloss.Width(l); w != bigDigitsW {
				t.Errorf("%s línea %d ancho %d, quiero %d", s, i, w, bigDigitsW)
			}
		}
	}
}

func TestBigDigitsZeroHasSlash(t *testing.T) {
	if !strings.Contains(BigDigits("0"), "██╔██") {
		t.Errorf("el 0 no tiene la barra:\n%s", BigDigits("0"))
	}
}

func TestBigDigitsOnlyBlockAndBoxChars(t *testing.T) {
	for _, r := range BigDigits("00:07") {
		if r == '\n' || strings.ContainsRune(" █╗╔═║╝╚", r) {
			continue
		}
		t.Errorf("carácter inesperado %q", r)
	}
}
