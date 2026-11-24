package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Juanrar/pomocli/internal/cycle"
	"github.com/Juanrar/pomocli/internal/store"
)

var spaceKey = tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}

func fa(m Model) *fakeAlarm { return m.alarm.(*fakeAlarm) }

func send(m Model, msg tea.Msg) (Model, tea.Cmd) {
	next, cmd := m.Update(msg)
	return next.(Model), cmd
}

func startFocus(t *testing.T, title string) (Model, *store.Store, *time.Time, store.Task) {
	t.Helper()
	m, st, now := newTestModel(t)
	task := addTask(t, st, *now, title)
	m = press(m, key("enter"))
	return m, st, now, task
}

// completeFocus avanza 50 minutos y entrega el tick vigente.
func completeFocus(m Model, now *time.Time) Model {
	*now = now.Add(50 * time.Minute)
	m, _ = send(m, tickMsg{id: m.tickID})
	return m
}

func TestStartFocusFromList(t *testing.T) {
	m, st, now := newTestModel(t)
	addTask(t, st, *now, "Leer")
	m, cmd := send(m, key("enter"))
	if m.screen != screenTimer {
		t.Fatalf("screen = %v", m.screen)
	}
	if cmd == nil {
		t.Fatal("cmd is nil")
	}
	c := content(m)
	for _, want := range []string{"Leer", "Foco 1 de 4"} {
		if !contains(c, want) {
			t.Errorf("missing %q in:\n%s", want, c)
		}
	}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }

func TestDoneTaskCannotStart(t *testing.T) {
	m, st, now := newTestModel(t)
	task := addTask(t, st, *now, "Leer")
	if err := st.SetDone(task.ID, true, *now); err != nil {
		t.Fatal(err)
	}
	m = press(m, key("enter"))
	if m.screen != screenList {
		t.Fatalf("screen = %v", m.screen)
	}
	if m.status != "Esa tarea ya está hecha." {
		t.Errorf("status = %q", m.status)
	}
}

func TestFocusCompletionSavesAndRings(t *testing.T) {
	m, st, now, task := startFocus(t, "Leer")
	m = completeFocus(m, now)
	if got := st.Progress(task.ID); got != 1 {
		t.Errorf("progress = %d", got)
	}
	if got := st.DaySummary(*now).FocusMinutes; got != 50 {
		t.Errorf("minutes = %d", got)
	}
	if fa(m).starts != 1 {
		t.Errorf("starts = %d", fa(m).starts)
	}
	c := content(m)
	for _, want := range []string{"Descanso corto", "Alarma sonando"} {
		if !contains(c, want) {
			t.Errorf("missing %q in:\n%s", want, c)
		}
	}
}

func TestSilenceKeyIsConsumed(t *testing.T) {
	m, _, now, _ := startFocus(t, "Leer")
	m = completeFocus(m, now)
	m = press(m, key("s"))
	if fa(m).stops != 1 {
		t.Errorf("stops = %d", fa(m).stops)
	}
	if m.cycle.Phase() != cycle.ShortBreak {
		t.Fatalf("phase = %v, want ShortBreak", m.cycle.Phase())
	}
	m = press(m, key("s"))
	if m.screen != screenWaiting {
		t.Errorf("screen = %v", m.screen)
	}
}

func TestBreakCompletionWaits(t *testing.T) {
	m, _, now, _ := startFocus(t, "Leer")
	m = completeFocus(m, now)
	m = press(m, key("x"))
	*now = now.Add(10 * time.Minute)
	m, cmd := send(m, tickMsg{id: m.tickID})
	if m.screen != screenWaiting {
		t.Fatalf("screen = %v", m.screen)
	}
	if fa(m).starts != 2 {
		t.Errorf("starts = %d", fa(m).starts)
	}
	if cmd == nil {
		t.Error("falta el comando que avisa el fin de la alarma")
	}
	if c := content(m); !contains(c, "Próximo: Foco 2 de 4") {
		t.Errorf("content:\n%s", c)
	}
}

func toWaiting(t *testing.T) (Model, *store.Store, *time.Time, store.Task) {
	m, st, now, task := startFocus(t, "Leer")
	m = completeFocus(m, now)
	m = press(m, key("x"), key("s"))
	if m.screen != screenWaiting {
		t.Fatalf("screen = %v", m.screen)
	}
	return m, st, now, task
}

func TestWaitingEnterStartsNextFocus(t *testing.T) {
	m, _, _, task := toWaiting(t)
	id := m.tickID
	m, cmd := send(m, key("enter"))
	if m.screen != screenTimer || m.cycle.Phase() != cycle.Focus {
		t.Fatalf("screen=%v phase=%v", m.screen, m.cycle.Phase())
	}
	if cmd == nil || m.tickID != id+1 || m.activeTaskID != task.ID {
		t.Errorf("cmd=%v tickID=%d active=%q", cmd, m.tickID, m.activeTaskID)
	}
	if c := content(m); !contains(c, "Foco 2 de 4") {
		t.Errorf("content:\n%s", c)
	}
}

func TestWaitingDoneMarksTask(t *testing.T) {
	m, st, _, task := toWaiting(t)
	m = press(m, key("d"))
	if m.screen != screenList || m.cycle.Phase() != cycle.Idle {
		t.Errorf("screen=%v phase=%v", m.screen, m.cycle.Phase())
	}
	if got, _ := st.Task(task.ID); !got.Done {
		t.Error("task not done")
	}
}

func TestStaleTickIgnored(t *testing.T) {
	m, st, now, task := startFocus(t, "Leer")
	m = press(m, key("esc"), key("enter"))
	if m.tickID != 2 {
		t.Fatalf("tickID = %d", m.tickID)
	}
	*now = now.Add(50 * time.Minute)
	m, cmd := send(m, tickMsg{id: 1})
	if cmd != nil || st.Progress(task.ID) != 0 {
		t.Errorf("stale tick acted: cmd=%v progress=%d", cmd, st.Progress(task.ID))
	}
	m, _ = send(m, tickMsg{id: 2})
	if st.Progress(task.ID) != 1 {
		t.Errorf("progress = %d", st.Progress(task.ID))
	}
}

func TestPauseShowsPausa(t *testing.T) {
	m, st, now, task := startFocus(t, "Leer")
	m = press(m, spaceKey)
	if c := content(m); !contains(c, "PAUSA") {
		t.Errorf("content:\n%s", c)
	}
	*now = now.Add(30 * time.Minute)
	m, _ = send(m, tickMsg{id: m.tickID})
	if st.Progress(task.ID) != 0 {
		t.Error("paused focus completed")
	}
	m = press(m, spaceKey)
	if m.cycle.Paused() {
		t.Error("still paused")
	}
}

func TestQuitDuringFocusConfirms(t *testing.T) {
	m, _, _, _ := startFocus(t, "Leer")
	m = press(m, key("q"))
	const prompt = "¿Salir? El foco en curso se pierde. (s/n)"
	if c := content(m); !contains(c, prompt) {
		t.Errorf("content:\n%s", c)
	}
	m = press(m, key("n"))
	if c := content(m); contains(c, prompt) {
		t.Error("prompt still shown")
	}
	m = press(m, key("q"))
	_, cmd := send(m, key("s"))
	if cmd == nil {
		t.Fatal("cmd nil")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("not QuitMsg")
	}
}

func TestSmallTerminalPlainTime(t *testing.T) {
	m, _, _, _ := startFocus(t, "Leer")
	m, _ = send(m, tea.WindowSizeMsg{Width: 20, Height: 8})
	if c := content(m); !contains(c, "50:00") {
		t.Errorf("content:\n%s", c)
	}
}

func TestTimerShowsSaveError(t *testing.T) {
	m, st, now, _ := startFocus(t, "Leer")
	dir := filepath.Dir(st.Path())
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	m = completeFocus(m, now)
	if c := content(m); !contains(c, "No se pudo guardar:") {
		t.Errorf("content:\n%s", c)
	}
}

func TestConfirmQuitClearedWhenFocusEnds(t *testing.T) {
	m, _, now, _ := startFocus(t, "Leer")
	m = press(m, key("q"))
	m = completeFocus(m, now)
	if c := content(m); contains(c, "¿Salir?") {
		t.Errorf("prompt still shown:\n%s", c)
	}
	m = press(m, key("s"))
	m, cmd := send(m, key("s"))
	if cmd != nil {
		t.Error("second s should not quit")
	}
	if m.screen != screenWaiting {
		t.Errorf("screen = %v", m.screen)
	}
}
