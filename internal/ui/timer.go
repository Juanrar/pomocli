package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Juanrar/pomocli/internal/alarm"
	"github.com/Juanrar/pomocli/internal/cycle"
	"github.com/Juanrar/pomocli/internal/store"
)

const tickInterval = 250 * time.Millisecond

type tickMsg struct{ id int }

const ringingLine = "Alarma sonando: apretá una tecla o clicá la ventana para silenciarla"

type ringEndedMsg struct{ id int }

// Ayudas de la pantalla de foco y de descanso, como pares tecla-acción.
var (
	focusFooter = [][2]string{{"espacio", "pausa"}, {"esc", "cancelar"}, {"q", "salir"}}
	breakFooter = [][2]string{{"espacio", "pausa"}, {"s", "saltar"}, {"esc", "lista"}, {"q", "salir"}}
)

var (
	digitBody   = lipgloss.NewStyle().Foreground(lipgloss.Color("#E0E0E0"))
	digitShadow = lipgloss.NewStyle().Foreground(lipgloss.Color("#6C6C6C"))
	focusPhase  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6347"))
	breakPhase  = lipgloss.NewStyle().Foreground(lipgloss.Color("#3CB371"))
)

// colorBig pinta el cuerpo (█) y la sombra (caja) de los dígitos grandes.
// Los espacios quedan sin estilo.
func colorBig(big string, faint bool) string {
	body, shadow := digitBody.Faint(faint), digitShadow.Faint(faint)
	var b, run strings.Builder
	kind := 0 // 0 espacio, 1 cuerpo, 2 sombra
	flush := func() {
		switch kind {
		case 1:
			b.WriteString(body.Render(run.String()))
		case 2:
			b.WriteString(shadow.Render(run.String()))
		default:
			b.WriteString(run.String())
		}
		run.Reset()
	}
	for _, r := range big {
		k := 2
		switch r {
		case ' ', '\n':
			k = 0
		case '█':
			k = 1
		}
		if k != kind {
			flush()
			kind = k
		}
		run.WriteRune(r)
	}
	flush()
	return b.String()
}

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

// startRinging arranca la alarma y devuelve el comando que avisa cuando se corta sola.
func (m *Model) startRinging() tea.Cmd {
	m.alarm.Start()
	m.ringUntil = m.now().Add(alarm.MaxRing)
	m.ringID++
	id := m.ringID
	return tea.Tick(alarm.MaxRing, func(time.Time) tea.Msg { return ringEndedMsg{id: id} })
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
		m.confirmQuit = false
		ring := m.startRinging()
		return m, tea.Batch(m.tickCmd(), ring)
	case cycle.BreakCompleted:
		ring := m.startRinging()
		m.screen = screenWaiting
		return m, ring
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
	key := cmdKey(msg)
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
	switch cmdKey(msg) {
	case "enter":
		return m.startFocus(m.activeTaskID)
	case "d":
		if err := m.store.SetDone(m.activeTaskID, true, m.now()); err != nil {
			m.fail(err)
		}
		m = m.toList()
		m.cursorTo(m.activeTaskID)
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

	phaseStyle := focusPhase
	if m.isBreak() {
		phaseStyle = breakPhase
	}
	digits := text
	if m.viewWidth() >= bigDigitsW+4 && m.height >= 13 {
		digits = colorBig(BigDigits(text), m.cycle.Paused())
	} else if m.cycle.Paused() {
		digits = faintStyle.Render(text)
	}
	lines := []string{digits}
	if m.cycle.Paused() {
		lines = append(lines, lipgloss.NewStyle().Bold(true).Render("PAUSA"))
	}
	lines = append(lines, "")

	phase := fmt.Sprintf("Foco %d de %d", m.cycle.Position(), m.cycle.Config().Every)
	footer := focusFooter
	switch m.cycle.Phase() {
	case cycle.ShortBreak:
		phase, footer = "Descanso corto", breakFooter
	case cycle.LongBreak:
		phase, footer = "Descanso largo", breakFooter
	}
	lines = append(lines, truncate(m.activeTitle(), max(m.viewWidth()-4, 1)), phaseStyle.Render(phase), m.progressLine(), "")
	if m.ringing() {
		lines = append(lines, ringingLine)
	}
	if m.status != "" {
		lines = append(lines, truncate(m.status, max(m.viewWidth()-4, 1)))
	}
	if w := m.warningLine(m.viewWidth() - 4); w != "" {
		lines = append(lines, w)
	}
	if m.confirmQuit {
		lines = append(lines, "¿Salir? El foco en curso se pierde. (s/n)")
	} else {
		lines = append(lines, keyHelp(footer...))
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
	if m.status != "" {
		lines = append(lines, truncate(m.status, max(m.viewWidth()-4, 1)))
	}
	if w := m.warningLine(m.viewWidth() - 4); w != "" {
		lines = append(lines, w)
	}
	lines = append(lines, keyHelp([2]string{"enter", "siguiente foco"}, [2]string{"d", "tarea hecha"},
		[2]string{"esc", "lista"}, [2]string{"q", "salir"}))
	return m.place(lines...)
}
