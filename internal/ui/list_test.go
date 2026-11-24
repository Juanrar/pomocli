package ui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/Juanrar/pomocli/internal/cycle"
	"github.com/Juanrar/pomocli/internal/store"
)

type fakeAlarm struct{ starts, stops int }

func (f *fakeAlarm) Start() { f.starts++ }
func (f *fakeAlarm) Stop()  { f.stops++ }

func newTestModel(t *testing.T) (Model, *store.Store, *time.Time) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "data.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 19, 9, 0, 0, 0, time.Local)
	m := New(st, cycle.DefaultConfig(), &fakeAlarm{}, func() time.Time { return now }, "")
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return next.(Model), st, &now
}

func press(m Model, keys ...tea.KeyPressMsg) Model {
	for _, k := range keys {
		next, _ := m.Update(k)
		m = next.(Model)
	}
	return m
}

func key(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	}
	r := []rune(s)[0]
	return tea.KeyPressMsg{Code: r, Text: s}
}

func typeText(m Model, s string) Model {
	for _, r := range s {
		m = press(m, key(string(r)))
	}
	return m
}

// content devuelve la vista sin secuencias de escape, para comparar texto.
func content(m Model) string { return ansi.Strip(m.View().Content) }

func addTask(t *testing.T, st *store.Store, now time.Time, title string) store.Task {
	t.Helper()
	task, err := st.Add(title, now)
	if err != nil {
		t.Fatal(err)
	}
	return task
}

func TestFormatDay(t *testing.T) {
	if got := formatDay(time.Date(2026, 11, 19, 0, 0, 0, 0, time.Local)); got != "19 nov" {
		t.Errorf("got %q", got)
	}
	if got := formatDay(time.Date(2026, 1, 5, 0, 0, 0, 0, time.Local)); got != "5 ene" {
		t.Errorf("got %q", got)
	}
}

func TestFormatDayTotals(t *testing.T) {
	cases := map[string]store.Summary{
		"3 pomodoros hoy, 75 min": {Count: 3, FocusMinutes: 75},
		"1 pomodoro hoy, 25 min":  {Count: 1, FocusMinutes: 25},
		"0 pomodoros hoy, 0 min":  {},
	}
	for want, s := range cases {
		if got := formatDayTotals(s); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}

func TestAddTask(t *testing.T) {
	m, st, now := newTestModel(t)
	m = press(m, key("a"))
	m = typeText(m, "Leer")
	m = press(m, key("enter"))
	tasks := st.Today(*now)
	if len(tasks) != 1 || tasks[0].Title != "Leer" {
		t.Fatalf("tasks = %+v", tasks)
	}
	c := content(m)
	if !strings.Contains(c, "[ ] Leer") || !strings.Contains(c, "0/1") {
		t.Errorf("content:\n%s", c)
	}
	if m.mode != listNormal {
		t.Errorf("mode = %v", m.mode)
	}
}

func TestAddEmptyShowsError(t *testing.T) {
	m, _, _ := newTestModel(t)
	m = press(m, key("a"), key("enter"))
	if m.mode != listAdding {
		t.Errorf("mode = %v", m.mode)
	}
	if c := content(m); !strings.Contains(c, "El título no puede estar vacío") {
		t.Errorf("content:\n%s", c)
	}
}

func TestEditTask(t *testing.T) {
	m, st, now := newTestModel(t)
	addTask(t, st, *now, "Leer")
	m = press(m, key("e"))
	if m.mode != listEditing {
		t.Fatalf("mode = %v", m.mode)
	}
	m = typeText(m, " mucho")
	m = press(m, key("enter"))
	if got := st.Today(*now)[0].Title; got != "Leer mucho" {
		t.Errorf("title = %q", got)
	}
}

func TestEscCancelsInput(t *testing.T) {
	m, st, now := newTestModel(t)
	m = press(m, key("a"))
	m = typeText(m, "x")
	m = press(m, key("esc"))
	if m.mode != listNormal || len(st.Today(*now)) != 0 {
		t.Errorf("mode = %v", m.mode)
	}
}

func TestEstimatePlusMinus(t *testing.T) {
	m, st, now := newTestModel(t)
	addTask(t, st, *now, "Leer")
	m = press(m, key("+"), key("+"))
	if got := st.Today(*now)[0].Estimate; got != 3 {
		t.Fatalf("estimate = %d", got)
	}
	m = press(m, key("-"), key("-"), key("-"), key("-"))
	if got := st.Today(*now)[0].Estimate; got != 1 {
		t.Errorf("estimate = %d, want 1", got)
	}
	_ = m
}

func TestToggleDone(t *testing.T) {
	m, st, now := newTestModel(t)
	addTask(t, st, *now, "Leer")
	m = press(m, key("d"))
	if c := content(m); !strings.Contains(c, "[✓] Leer") {
		t.Errorf("content:\n%s", c)
	}
	m = press(m, key("d"))
	if c := content(m); !strings.Contains(c, "[ ] Leer") {
		t.Errorf("content:\n%s", c)
	}
}

func TestDeleteConfirm(t *testing.T) {
	m, st, now := newTestModel(t)
	addTask(t, st, *now, "Leer")
	m = press(m, key("x"))
	if c := content(m); !strings.Contains(c, `¿Borrar "Leer"? (s/n)`) {
		t.Errorf("content:\n%s", c)
	}
	m = press(m, key("n"))
	if len(st.Today(*now)) != 1 {
		t.Fatal("n borró la tarea")
	}
	m = press(m, key("x"), key("s"))
	if len(st.Today(*now)) != 0 {
		t.Error("s no borró la tarea")
	}
	if m.cursor != 0 || m.mode != listNormal {
		t.Errorf("cursor=%d mode=%v", m.cursor, m.mode)
	}
}

func TestCursorBounds(t *testing.T) {
	m, st, now := newTestModel(t)
	addTask(t, st, *now, "Uno")
	addTask(t, st, *now, "Dos")
	m = press(m, key("up"), key("k"))
	if m.cursor != 0 {
		t.Errorf("cursor = %d", m.cursor)
	}
	m = press(m, key("down"), key("j"), key("j"))
	if m.cursor != 1 {
		t.Errorf("cursor = %d", m.cursor)
	}
}

func TestCarriedTaskShowsSince(t *testing.T) {
	m, st, now := newTestModel(t)
	addTask(t, st, now.AddDate(0, 0, -2), "Vieja")
	addTask(t, st, *now, "Nueva")
	c := content(m)
	if !strings.Contains(c, "desde 17 nov") {
		t.Errorf("content:\n%s", c)
	}
	if strings.Count(c, "desde") != 1 {
		t.Errorf("solo la vieja lleva desde:\n%s", c)
	}
}

func TestEmptyListKeysAreNoOps(t *testing.T) {
	m, _, _ := newTestModel(t)
	for _, k := range []string{"enter", "d", "x", "e", "+", "-", "up", "down"} {
		m = press(m, key(k))
	}
	if c := content(m); !strings.Contains(c, "No hay tareas.") {
		t.Errorf("content:\n%s", c)
	}
}

func TestLongTitleTruncated(t *testing.T) {
	m, st, now := newTestModel(t)
	addTask(t, st, *now, strings.Repeat("ñandú ", 40))
	addTask(t, st, now.AddDate(0, 0, -1), "Ñ"+strings.Repeat("😀", 100))
	c := content(m)
	for _, line := range strings.Split(c, "\n") {
		if w := lipgloss.Width(line); w > 80 {
			t.Errorf("ancho %d > 80: %q", w, line)
		}
	}
	if !strings.Contains(c, "…") {
		t.Errorf("sin elipsis:\n%s", c)
	}
}

func TestSummaryScreen(t *testing.T) {
	m, st, now := newTestModel(t)
	task := addTask(t, st, *now, "Leer")
	for i := 0; i < 2; i++ {
		err := st.AddPomodoro(store.Pomodoro{
			TaskID: task.ID, TaskTitle: task.Title,
			Start: *now, End: now.Add(25 * time.Minute), FocusSeconds: 1500,
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	m = press(m, key("r"))
	c := content(m)
	if !strings.Contains(c, "Resumen de hoy") || !strings.Contains(c, "2 pomodoros hoy, 50 min") {
		t.Errorf("content:\n%s", c)
	}
	m = press(m, key("esc"))
	if m.screen != screenList || !strings.Contains(content(m), "[ ] Leer") {
		t.Errorf("no volvió a la lista:\n%s", content(m))
	}
}

func TestSummaryEmpty(t *testing.T) {
	m, _, _ := newTestModel(t)
	m = press(m, key("r"))
	if c := content(m); !strings.Contains(c, "Todavía no hay pomodoros hoy.") {
		t.Errorf("content:\n%s", c)
	}
}

func TestQuit(t *testing.T) {
	m, _, _ := newTestModel(t)
	_, cmd := m.Update(key("q"))
	if cmd == nil {
		t.Fatal("cmd nil")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("no devolvió QuitMsg")
	}
}
