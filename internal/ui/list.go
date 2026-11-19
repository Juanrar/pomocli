package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Juanrar/pomocli/internal/store"
)

func (m Model) tasks() []store.Task { return m.store.Today(m.now()) }

// current devuelve la tarea bajo el cursor, si hay.
func (m Model) current() (store.Task, bool) {
	tasks := m.tasks()
	if m.cursor < 0 || m.cursor >= len(tasks) {
		return store.Task{}, false
	}
	return tasks[m.cursor], true
}

func (m *Model) clampCursor() {
	m.cursor = max(min(m.cursor, len(m.tasks())-1), 0)
}

func (m *Model) fail(err error) {
	m.status = fmt.Sprintf("No se pudo guardar: %v", err)
}

func (m Model) updateList(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case listAdding, listEditing:
		return m.updateInput(msg)
	case listConfirmDelete:
		return m.updateConfirm(msg)
	}
	cur, ok := m.current()
	key := msg.String()
	switch key {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.tasks())-1 {
			m.cursor++
		}
	case "a":
		m.mode = listAdding
		m.input.Reset()
		return m, m.input.Focus()
	case "e":
		if ok {
			m.mode = listEditing
			m.input.SetValue(cur.Title)
			return m, m.input.Focus()
		}
	case "+", "-":
		if ok {
			n := cur.Estimate + 1
			if key == "-" {
				n = cur.Estimate - 1
			}
			if err := m.store.SetEstimate(cur.ID, n); err != nil {
				m.fail(err)
			}
		}
	case "d":
		if ok {
			if err := m.store.SetDone(cur.ID, !cur.Done, m.now()); err != nil {
				m.fail(err)
			}
		}
	case "x":
		if ok {
			m.mode = listConfirmDelete
		}
	case "r":
		m.screen = screenSummary
	case "q", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) updateInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.mode = listNormal
		m.input.Blur()
		return m, nil
	case "enter":
		var err error
		if m.mode == listAdding {
			var t store.Task
			if t, err = m.store.Add(m.input.Value(), m.now()); err == nil {
				for i, x := range m.tasks() {
					if x.ID == t.ID {
						m.cursor = i
					}
				}
			}
		} else if cur, ok := m.current(); ok {
			err = m.store.Rename(cur.ID, m.input.Value())
		}
		if err == store.ErrEmptyTitle {
			m.status = "El título no puede estar vacío"
			return m, nil
		}
		if err != nil {
			m.fail(err)
		}
		m.mode = listNormal
		m.input.Blur()
		m.clampCursor()
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) updateConfirm(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "s":
		if cur, ok := m.current(); ok {
			if err := m.store.Delete(cur.ID); err != nil {
				m.fail(err)
			}
		}
		m.mode = listNormal
		m.clampCursor()
	case "n", "esc":
		m.mode = listNormal
	case "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) viewList() string {
	w := m.viewWidth()
	now := m.now()
	var b strings.Builder

	left := " " + titleStyle.Render("pomocli") + "   Hoy, " + formatDay(now)
	right := faintStyle.Render(formatDayTotals(m.store.DaySummary(now))) + " "
	gap := max(w-lipgloss.Width(left)-lipgloss.Width(right), 1)
	b.WriteString(left + strings.Repeat(" ", gap) + right + "\n")
	b.WriteString(rule(w) + "\n")

	tasks := m.tasks()
	if len(tasks) == 0 {
		b.WriteString(" No hay tareas. Apretá a para agregar una.\n")
	}
	for i, t := range tasks {
		b.WriteString(m.taskLine(t, i == m.cursor, w) + "\n")
	}
	b.WriteString(rule(w) + "\n")

	switch m.mode {
	case listAdding:
		b.WriteString(" Nueva tarea: " + m.input.View() + "\n")
		b.WriteString(" " + faintStyle.Render("enter guardar  esc cancelar") + "\n")
	case listEditing:
		b.WriteString(" Editar: " + m.input.View() + "\n")
		b.WriteString(" " + faintStyle.Render("enter guardar  esc cancelar") + "\n")
	case listConfirmDelete:
		title := ""
		if cur, ok := m.current(); ok {
			title = cur.Title
		}
		b.WriteString(" " + truncate(`¿Borrar "`+title+`"? (s/n)`, w-2) + "\n")
	}
	if m.status != "" {
		b.WriteString(" " + truncate(m.status, w-2) + "\n")
	}
	b.WriteString(" " + faintStyle.Render("enter foco  a agregar  e editar  +/- estimación  d hecha") + "\n")
	b.WriteString(" " + faintStyle.Render("x borrar  r resumen  q salir"))
	return b.String()
}

func (m Model) taskLine(t store.Task, selected bool, w int) string {
	prefix := "   "
	if selected {
		prefix = " > "
	}
	check := "[ ] "
	if t.Done {
		check = "[x] "
	}
	progress := m.store.Progress(t.ID)
	right := dots(progress, t.Estimate) + fmt.Sprintf("  %d/%d ", progress, t.Estimate)

	now := m.now()
	created := t.CreatedAt.In(now.Location())
	since := ""
	if !sameDate(created, now) && created.Before(now) {
		since = "desde " + formatDay(created)
	}

	avail := w - lipgloss.Width(prefix+check) - lipgloss.Width(right) - 1
	sinceW := 0
	if since != "" {
		sinceW = lipgloss.Width(since) + 2
		if avail-sinceW < 10 {
			since, sinceW = "", 0
		}
	}
	mid := padRight(truncate(t.Title, max(avail-sinceW, 1)), max(avail-sinceW, 1))
	if since != "" {
		mid += "  " + faintStyle.Render(since)
	}
	line := prefix + check + padRight(mid, avail) + " " + right
	if t.Done {
		line = faintStyle.Render(line)
	}
	return line
}
