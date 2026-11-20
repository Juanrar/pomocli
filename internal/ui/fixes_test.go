package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Juanrar/pomocli/internal/cycle"
	"github.com/Juanrar/pomocli/internal/store"
)

const testWarning = "Sin audio (prueba). Uso el bell de la terminal."

func newWarnModel(t *testing.T) (Model, *store.Store, *time.Time) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "data.json"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 19, 9, 0, 0, 0, time.Local)
	m := New(st, cycle.DefaultConfig(), &fakeAlarm{}, func() time.Time { return now }, testWarning)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	return next.(Model), st, &now
}

func TestAudioWarningSurvivesKeysAndShowsEverywhere(t *testing.T) {
	m, st, now := newWarnModel(t)
	addTask(t, st, *now, "Leer")
	if c := content(m); !contains(c, testWarning) {
		t.Fatalf("falta el aviso en la lista:\n%s", c)
	}
	m = press(m, key("down"), key("up"), key("j"))
	if c := content(m); !contains(c, testWarning) {
		t.Fatalf("el aviso desapareció después de teclas:\n%s", c)
	}
	m = press(m, key("enter"))
	if m.screen != screenTimer {
		t.Fatalf("screen = %v", m.screen)
	}
	if c := content(m); !contains(c, testWarning) {
		t.Fatalf("falta el aviso en el timer:\n%s", c)
	}
	m.screen = screenWaiting
	if c := content(m); !contains(c, testWarning) {
		t.Fatalf("falta el aviso en la espera:\n%s", c)
	}
}

// ringingWaiting lleva el modelo a la espera con la alarma del descanso sonando.
func ringingWaiting(t *testing.T) (Model, *time.Time) {
	t.Helper()
	m, _, now, _ := startFocus(t, "Leer")
	m = completeFocus(m, now)
	m = press(m, key("x")) // silencia la alarma del foco
	*now = now.Add(10 * time.Minute)
	m, _ = send(m, tickMsg{id: m.tickID})
	if m.screen != screenWaiting || !m.ringing() {
		t.Fatalf("screen=%v ringing=%v", m.screen, m.ringing())
	}
	return m, now
}

func TestRingEndedClearsLineAndKeyIsACommand(t *testing.T) {
	m, now := ringingWaiting(t)
	if !contains(content(m), ringingLine) {
		t.Fatalf("debería sonar:\n%s", content(m))
	}
	*now = now.Add(61 * time.Second)
	m, _ = send(m, ringEndedMsg{id: m.ringID})
	if contains(content(m), ringingLine) {
		t.Fatalf("la línea de alarma sigue:\n%s", content(m))
	}
	m = press(m, key("enter"))
	if m.screen != screenTimer || m.cycle.Phase() != cycle.Focus {
		t.Fatalf("enter no empezó el foco: screen=%v phase=%v", m.screen, m.cycle.Phase())
	}
}

func TestRingEndedStaleIDIgnored(t *testing.T) {
	m, _ := ringingWaiting(t)
	m, _ = send(m, ringEndedMsg{id: m.ringID - 1})
	if m.ringUntil.IsZero() {
		t.Fatal("un id viejo cortó la alarma")
	}
}

func TestAlarmStartReturnsRingEndTick(t *testing.T) {
	m, _, now, _ := startFocus(t, "Leer")
	*now = now.Add(50 * time.Minute)
	m, cmd := send(m, tickMsg{id: m.tickID})
	if cmd == nil || m.ringID == 0 {
		t.Fatalf("cmd=%v ringID=%d", cmd, m.ringID)
	}
}

func TestSkipBreakDoesNotRing(t *testing.T) {
	m, _, now, _ := startFocus(t, "Leer")
	m = completeFocus(m, now)
	m = press(m, key("x")) // silencia
	before := fa(m).starts
	m = press(m, key("s"))
	if m.screen != screenWaiting {
		t.Fatalf("screen = %v", m.screen)
	}
	if fa(m).starts != before {
		t.Errorf("starts = %d, quiero %d", fa(m).starts, before)
	}
}

func TestCapsLockCommands(t *testing.T) {
	m, st, now := newTestModel(t)
	addTask(t, st, *now, "Leer")
	m = press(m, key("x"), key("S"))
	if len(st.Today(*now)) != 0 {
		t.Fatal("x y S no borró la tarea")
	}
	_, cmd := m.Update(key("Q"))
	if cmd == nil {
		t.Fatal("Q no salió")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("Q no devolvió QuitMsg")
	}
	m, st, now = newTestModel(t)
	addTask(t, st, *now, "Leer")
	m = press(m, key("A"))
	if m.mode != listAdding {
		t.Errorf("A no abrió el input, mode = %v", m.mode)
	}
	m = typeText(m, "XyZ")
	if m.input.Value() != "XyZ" {
		t.Errorf("el input perdió mayúsculas: %q", m.input.Value())
	}
}

func TestInputFitsWidth(t *testing.T) {
	m, _, _ := newTestModel(t)
	m, _ = send(m, tea.WindowSizeMsg{Width: 40, Height: 24})
	m = press(m, key("a"))
	m = typeText(m, strings.Repeat("w", 120))
	found := false
	for i, l := range strings.Split(content(m), "\n") {
		if !strings.Contains(l, "Nueva tarea:") {
			continue
		}
		found = true
		if w := lipgloss.Width(l); w > 40 {
			t.Errorf("línea %d ancho %d > 40: %q", i, w, l)
		}
	}
	if !found {
		t.Fatal("no se ve el input")
	}
}

func manyTasks(t *testing.T, n int) (Model, *store.Store, *time.Time) {
	t.Helper()
	m, st, now := newTestModel(t)
	for i := 1; i <= n; i++ {
		addTask(t, st, *now, fmt.Sprintf("Tarea %02d", i))
	}
	m, _ = send(m, tea.WindowSizeMsg{Width: 80, Height: 20})
	return m, st, now
}

func TestListScrollsWithManyTasks(t *testing.T) {
	m, _, _ := manyTasks(t, 40)
	check := func(label string) {
		t.Helper()
		c := content(m)
		if n := len(strings.Split(c, "\n")); n > 20 {
			t.Fatalf("%s: %d líneas > 20:\n%s", label, n, c)
		}
		if !contains(c, "q salir") {
			t.Fatalf("%s: falta la ayuda:\n%s", label, c)
		}
	}
	check("inicio")
	if c := content(m); !contains(c, "más") {
		t.Errorf("falta el indicador de filas ocultas:\n%s", c)
	}
	m = press(m, key("a"))
	check("agregando")
	if !contains(content(m), "Nueva tarea:") {
		t.Fatalf("falta el input:\n%s", content(m))
	}
	m = press(m, key("esc"))
	for i := 0; i < 39; i++ {
		m = press(m, key("down"))
		if c := content(m); !contains(c, fmt.Sprintf("Tarea %02d", i+2)) {
			t.Fatalf("el cursor en %d no se ve:\n%s", i+2, c)
		}
	}
	check("final")
	m = press(m, key("x"))
	check("borrar")
	m = press(m, key("esc"))
	for i := 0; i < 39; i++ {
		m = press(m, key("up"))
	}
	if c := content(m); !contains(c, "Tarea 01") {
		t.Errorf("no volvió arriba:\n%s", c)
	}
}

func TestListScrollAfterDelete(t *testing.T) {
	m, _, _ := manyTasks(t, 30)
	for i := 0; i < 29; i++ {
		m = press(m, key("down"))
	}
	m = press(m, key("x"), key("s"))
	c := content(m)
	if !contains(c, "Tarea 29") || len(strings.Split(c, "\n")) > 20 {
		t.Errorf("después de borrar:\n%s", c)
	}
}
