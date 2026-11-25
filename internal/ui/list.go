package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/Juanrar/pomocli/internal/store"
)

// maxContentW es el ancho del bloque central de la lista.
const maxContentW = 64

// tasks devuelve las tareas de hoy: primero las pendientes y después las hechas,
// cada grupo en su orden original.
func (m Model) tasks() []store.Task {
	all := m.store.Today(m.now())
	out := make([]store.Task, 0, len(all))
	for _, t := range all {
		if !t.Done {
			out = append(out, t)
		}
	}
	for _, t := range all {
		if t.Done {
			out = append(out, t)
		}
	}
	return out
}

// sections cuenta las tareas pendientes y las hechas de hoy.
func (m Model) sections() (pending, done int) {
	for _, t := range m.tasks() {
		if t.Done {
			done++
		} else {
			pending++
		}
	}
	return pending, done
}

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

// cursorTo pone el cursor sobre la tarea id, si sigue en la lista de hoy.
func (m *Model) cursorTo(id string) {
	for i, t := range m.tasks() {
		if t.ID == id {
			m.cursor = i
		}
	}
	m.clampCursor()
}

// contentWidth devuelve el ancho del bloque central.
func (m Model) contentWidth() int { return min(m.viewWidth(), maxContentW) }

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
	key := cmdKey(msg)
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
		cmd := m.input.Focus()
		return m, cmd
	case "e":
		if ok {
			m.mode = listEditing
			m.input.SetValue(cur.Title)
			cmd := m.input.Focus()
			return m, cmd
		}
	case "enter":
		if ok {
			if cur.Done {
				m.status = "Esa tarea ya está hecha."
				return m, nil
			}
			return m.startFocus(cur.ID)
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
			m.cursorTo(cur.ID)
		}
	case "x":
		if ok {
			m.mode = listConfirmDelete
		}
	case "r":
		m.screen = screenSummary
	case "c":
		m.screen = screenSettings
		m.settingsCursor = 0
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
	switch cmdKey(msg) {
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
	w := m.contentWidth()
	now := m.now()
	tasks := m.tasks()
	pending, done := m.sections()
	var b strings.Builder
	line := func(s string) {
		b.WriteString(lipgloss.PlaceHorizontal(w, lipgloss.Center, s) + "\n")
	}
	rule := faintStyle.Render(strings.Repeat("─", w)) + "\n"

	line(titleStyle.Render("pomocli") + "   Hoy, " + formatDay(now))
	b.WriteString(rule)
	line(m.dayBar(w))
	b.WriteString(rule)

	if len(tasks) == 0 {
		line("No hay tareas. Apretá a para agregar una.")
	}
	start, end := m.window()
	if start > 0 {
		line(faintStyle.Render(fmt.Sprintf("↑ %d más", start)))
	}
	for i := start; i < end; i++ {
		if i == 0 || i == pending {
			label, n := "Pendientes", pending
			if i >= pending {
				label, n = "Hechas", done
			}
			b.WriteString(sectionHeader(label, n, w) + "\n")
		}
		b.WriteString(m.taskLine(tasks[i], i == m.cursor, w) + "\n")
	}
	if end < len(tasks) {
		line(faintStyle.Render(fmt.Sprintf("↓ %d más", len(tasks)-end)))
	}
	b.WriteString(rule)

	switch m.mode {
	case listAdding:
		line("Nueva tarea: " + m.input.View())
		line(faintStyle.Render("enter guardar  esc cancelar"))
	case listEditing:
		line("Editar: " + m.input.View())
		line(faintStyle.Render("enter guardar  esc cancelar"))
	case listConfirmDelete:
		title := ""
		if cur, ok := m.current(); ok {
			title = cur.Title
		}
		line(truncate(`¿Borrar "`+title+`"? (s/n)`, w-2))
	}
	if m.status != "" {
		line(truncate(m.status, w-2))
	}
	if wl := m.warningLine(w - 2); wl != "" {
		line(wl)
	}
	line(keyHelp([2]string{"enter", "foco"}, [2]string{"a", "agregar"}, [2]string{"e", "editar"},
		[2]string{"+/-", "estimación"}, [2]string{"d", "hecha"}))
	line(keyHelp([2]string{"x", "borrar"}, [2]string{"r", "resumen"}, [2]string{"c", "configurar"}, [2]string{"q", "salir"}))
	return lipgloss.Place(m.viewWidth(), m.height, lipgloss.Center, lipgloss.Center,
		strings.TrimSuffix(b.String(), "\n"))
}

// keyHelp dibuja pares tecla-acción: la tecla en color y la acción en tenue.
func keyHelp(pairs ...[2]string) string {
	parts := make([]string, len(pairs))
	for i, p := range pairs {
		parts[i] = keyStyle.Render(p[0]) + " " + faintStyle.Render(p[1])
	}
	return strings.Join(parts, "   ")
}

// dayBar dibuja el avance del día: pomodoros hechos sobre la estimación total
// de las tareas de hoy, y el tiempo de foco.
func (m Model) dayBar(w int) string {
	now := m.now()
	sum := m.store.DaySummary(now)
	total := 0
	for _, t := range m.tasks() {
		total += t.Estimate
	}
	const barW = 24
	filled := 0
	count := fmt.Sprintf("%d pomodoros · %d min", sum.Count, sum.FocusMinutes)
	if total > 0 {
		filled = min(sum.Count*barW/total, barW)
		count = fmt.Sprintf("%d de %d pomodoros · %d min", sum.Count, total, sum.FocusMinutes)
	}
	bar := doneDot.Render(strings.Repeat("▓", filled)) +
		faintStyle.Render(strings.Repeat("░", barW-filled))
	return truncate("Día  "+bar+"  "+count, w)
}

// sectionHeader dibuja el título de una sección con su cantidad a la derecha.
func sectionHeader(label string, n, w int) string {
	left := "   " + lipgloss.NewStyle().Bold(true).Render(label)
	right := faintStyle.Render(fmt.Sprint(n)) + " "
	gap := max(w-lipgloss.Width(left)-lipgloss.Width(right), 1)
	return left + strings.Repeat(" ", gap) + right
}

func (m Model) taskLine(t store.Task, selected bool, w int) string {
	prefix := "   "
	if selected {
		prefix = " > "
	}
	check := "[ ] "
	if t.Done {
		check = "[√] "
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
	rest := check + padRight(mid, avail) + " " + right
	if t.Done {
		// Quita los colores de los puntos y de la fecha para que toda la
		// línea quede en gris. El cursor queda sin estilo para seguir visible.
		return prefix + doneTaskStyle.Render(ansi.Strip(rest))
	}
	return prefix + rest
}

// fixedLines cuenta las líneas de la lista que no son filas de tareas:
// título, barra del día, reglas, encabezados de sección, input o confirmación,
// estado, aviso y ayuda.
func (m Model) fixedLines() int {
	n := 4 + 1 + 2 + m.headerLines()
	switch m.mode {
	case listAdding, listEditing:
		n += 2
	case listConfirmDelete:
		n++
	}
	if m.status != "" {
		n++
	}
	if m.audioWarning != "" {
		n++
	}
	return n
}

// headerLines cuenta los encabezados de sección: uno por sección con tareas.
func (m Model) headerLines() int {
	p, d := m.sections()
	n := 0
	if p > 0 {
		n++
	}
	if d > 0 {
		n++
	}
	return n
}

// window devuelve el rango de tareas [start, end) que entra en pantalla,
// con el cursor adentro. Cuando hay filas ocultas, los indicadores "↑ N más"
// y "↓ N más" ocupan una línea cada uno.
func (m Model) window() (start, end int) {
	n := len(m.tasks())
	avail := m.height - m.fixedLines()
	if m.height <= 0 || n <= avail {
		return 0, n
	}
	v := max(avail-2, 1)
	start = max(min(m.scroll, n-v), 0)
	if m.cursor < start {
		start = m.cursor
	}
	if m.cursor >= start+v {
		start = m.cursor - v + 1
	}
	end = min(start+v, n)
	if start == 0 {
		end = min(v+1, n)
	}
	if end == n && start > 0 {
		start = max(n-(v+1), 0)
	}
	return start, end
}

// fixScroll guarda el inicio de la ventana para que siga al cursor.
func (m *Model) fixScroll() {
	m.clampCursor()
	m.scroll, _ = m.window()
}
