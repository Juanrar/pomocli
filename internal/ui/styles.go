package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/Juanrar/pomocli/internal/store"
)

var (
	colorAccent = lipgloss.Color("#FF6347")
	colorExtra  = lipgloss.Color("#FFD700")

	titleStyle = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	doneDot    = lipgloss.NewStyle().Foreground(colorAccent)
	extraDot   = lipgloss.NewStyle().Foreground(colorExtra)
	faintStyle = lipgloss.NewStyle().Faint(true)
)

var months = [12]string{"ene", "feb", "mar", "abr", "may", "jun", "jul", "ago", "sep", "oct", "nov", "dic"}

// formatDay devuelve el día y el mes abreviado, por ejemplo "19 nov".
func formatDay(t time.Time) string {
	return fmt.Sprintf("%d %s", t.Day(), months[t.Month()-1])
}

// formatDayTotals devuelve el total del día, por ejemplo "3 pomodoros hoy, 75 min".
func formatDayTotals(s store.Summary) string {
	word := "pomodoros"
	if s.Count == 1 {
		word = "pomodoro"
	}
	return fmt.Sprintf("%d %s hoy, %d min", s.Count, word, s.FocusMinutes)
}

// dots dibuja los pomodoros hechos (rellenos), los que faltan (vacíos) y los
// que exceden la estimación (rellenos de otro color).
func dots(done, estimate int) string {
	filled := min(done, estimate)
	s := doneDot.Render(strings.Repeat("●", filled)) + strings.Repeat("○", estimate-filled)
	if done > estimate {
		s += extraDot.Render(strings.Repeat("●", done-estimate))
	}
	return s
}

// truncate corta s a w columnas de ancho visible y agrega "…" si hizo falta.
func truncate(s string, w int) string {
	if w < 1 {
		return ""
	}
	return ansi.Truncate(s, w, "…")
}

// padRight rellena s con espacios hasta w columnas.
func padRight(s string, w int) string {
	if d := w - lipgloss.Width(s); d > 0 {
		return s + strings.Repeat(" ", d)
	}
	return s
}

func rule(w int) string {
	return " " + faintStyle.Render(strings.Repeat("─", max(w-2, 0)))
}

func sameDate(a, b time.Time) bool {
	a = a.In(b.Location())
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
