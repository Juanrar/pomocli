package ui

import (
	"fmt"
	"math"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Juanrar/pomocli/internal/alarm"
	"github.com/Juanrar/pomocli/internal/cycle"
	"github.com/Juanrar/pomocli/internal/store"
)

const tickInterval = 250 * time.Millisecond

type tickMsg struct{ id int }

const ringingLine = "Alarma sonando: apretá cualquier tecla para silenciarla"

var (
	focusDigits = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6347"))
	breakDigits = lipgloss.NewStyle().Foreground(lipgloss.Color("#3CB371"))
)

func (m Model) tickCmd() tea.Cmd {
	id := m.tickID
	return tea.Tick(tickInterval, func(time.Time) tea.Msg { return tickMsg{id: id} })
}

// startFocus arranca un foco sobre la tarea id y devuelve el primer tick.
func (m Model) startFocus(id string) (Model, tea.Cmd) {
	now := m.now()
	if err := m.cycle.StartFocus(now); err != nil {
		m.status = err.Error()
		return m, nil
	}
	m.activeTaskID = id
	m.focusStart = now
	m.tickID++
	m.confirmQuit = false
	m.screen = screenTimer
	return m, m.tickCmd()
}

func (m Model) ringing() bool { return m.now().Before(m.ringUntil) }

func (m *Model) startRinging() {
	m.alarm.Start()
	m.ringUntil = m.now().Add(alarm.MaxRing)
}

func (m Model) isBreak() bool {
	p := m.cycle.Phase()
	return p == cycle.ShortBreak || p == cycle.LongBreak
}

func (m Model) handleTick(msg tickMsg) (tea.Model, tea.Cmd) {
	if msg.id != m.tickID {
		return m, nil
	}
	now := m.now()
	switch m.cycle.Tick(now) {
	case cycle.FocusCompleted:
		err := m.store.AddPomodoro(store.Pomodoro{
			TaskID:       m.activeTaskID,
			TaskTitle:    m.activeTitle(),
			Start:        m.focusStart,
			End:          now,
			FocusSeconds: int(m.cycle.Config().Focus.Seconds()),
		})
		if err != nil {
			m.fail(err)
		}
		m.startRinging()
		return m, m.tickCmd()
	case cycle.BreakCompleted:
		m.startRinging()
		m.screen = screenWaiting
		return m, nil
	}
	switch m.cycle.Phase() {
	case cycle.Focus, cycle.ShortBreak, cycle.LongBreak:
		return m, m.tickCmd()
	}
	return m, nil
}

func (m Model) toList() Model {
	m.cycle.Cancel()
	m.confirmQuit = false
	m.screen = screenList
	return m
}

func (m *Model) togglePause(now time.Time) {
	if m.cycle.Paused() {
		m.cycle.Resume(now)
	} else {
		m.cycle.Pause(now)
	}
}

func (m Model) updateTimer(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	now := m.now()
	if m.isBreak() {
		switch key {
		case "space":
			m.togglePause(now)
		case "s":
			m.cycle.SkipBreak()
			m.screen = screenWaiting
		case "esc":
			m = m.toList()
		case "q", "ctrl+c":
			return m, tea.Quit
		}
		return m, nil
	}
	if m.confirmQuit {
		switch key {
		case "s":
			return m, tea.Quit
		case "n", "esc":
			m.confirmQuit = false
		}
		return m, nil
	}
	switch key {
	case "space":
		m.togglePause(now)
	case "esc":
		m = m.toList()
	case "q", "ctrl+c":
		m.confirmQuit = true
	}
	return m, nil
}

func (m Model) updateWaiting(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		return m.startFocus(m.activeTaskID)
	case "d":
		if err := m.store.SetDone(m.activeTaskID, true, m.now()); err != nil {
			m.fail(err)
		}
		m = m.toList()
		m.clampCursor()
	case "esc":
		m = m.toList()
	case "q", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) activeTitle() string {
	if t, ok := m.store.Task(m.activeTaskID); ok {
		return t.Title
	}
	return ""
}

func (m Model) progressLine() string {
	t, ok := m.store.Task(m.activeTaskID)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%d/%d", m.store.Progress(t.ID), t.Estimate)
}

func (m Model) place(lines ...string) string {
	return lipgloss.Place(m.viewWidth(), m.height, lipgloss.Center, lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center, lines...))
}

func (m Model) viewTimer() string {
	secs := int(math.Ceil(m.cycle.Remaining(m.now()).Seconds()))
	text := fmt.Sprintf("%02d:%02d", secs/60, secs%60)

	style := focusDigits
	if m.isBreak() {
		style = breakDigits
	}
	if m.cycle.Paused() {
		style = style.Faint(true)
	}
	digits := text
	if big := BigDigits(text); m.viewWidth() >= lipgloss.Width(big)+4 && m.height >= 12 {
		digits = big
	}
	lines := []string{style.Render(digits)}
	if m.cycle.Paused() {
		lines = append(lines, style.Bold(true).Render("PAUSA"))
	}
	lines = append(lines, "")

	phase := fmt.Sprintf("Foco %d de %d", m.cycle.Position(), m.cycle.Config().Every)
	footer := "espacio pausa  esc cancelar  q salir"
	switch m.cycle.Phase() {
	case cycle.ShortBreak:
		phase, footer = "Descanso corto", "espacio pausa  s saltar  esc lista  q salir"
	case cycle.LongBreak:
		phase, footer = "Descanso largo", "espacio pausa  s saltar  esc lista  q salir"
	}
	lines = append(lines, truncate(m.activeTitle(), max(m.viewWidth()-4, 1)), phase, m.progressLine(), "")
	if m.ringing() {
		lines = append(lines, ringingLine)
	}
	if m.confirmQuit {
		lines = append(lines, "¿Salir? El foco en curso se pierde. (s/n)")
	} else {
		lines = append(lines, faintStyle.Render(footer))
	}
	return m.place(lines...)
}

func (m Model) viewWaiting() string {
	lines := []string{
		titleStyle.Render("Descanso terminado"),
		"",
		truncate(m.activeTitle(), max(m.viewWidth()-4, 1)),
		fmt.Sprintf("Próximo: Foco %d de %d", m.cycle.Position(), m.cycle.Config().Every),
		m.progressLine(),
		"",
	}
	if m.ringing() {
		lines = append(lines, ringingLine)
	}
	lines = append(lines, faintStyle.Render("enter siguiente foco  d tarea hecha  esc lista  q salir"))
	return m.place(lines...)
}
