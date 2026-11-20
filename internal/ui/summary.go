package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m Model) updateSummary(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch cmdKey(msg) {
	case "esc":
		m.screen = screenList
	case "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) viewSummary() string {
	w := m.viewWidth()
	sum := m.store.DaySummary(m.now())
	var b strings.Builder
	b.WriteString(" " + titleStyle.Render("Resumen de hoy") + "\n")
	b.WriteString(rule(w) + "\n")
	if sum.Count == 0 {
		b.WriteString(" Todavía no hay pomodoros hoy.\n")
	}
	for _, tc := range sum.ByTask {
		right := doneDot.Render(strings.Repeat("●", tc.Count)) + fmt.Sprintf("  %d", tc.Count)
		tw := max(w-2-lipgloss.Width(right)-2, 1)
		b.WriteString(" " + padRight(truncate(tc.Title, tw), tw) + "  " + right + "\n")
	}
	b.WriteString(rule(w) + "\n")
	b.WriteString(" " + formatDayTotals(sum) + "\n")
	b.WriteString(" " + keyHelp([2]string{"esc", "volver"}))
	return b.String()
}
