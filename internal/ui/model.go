// Package ui implementa la interfaz de terminal con Bubble Tea.
package ui

import (
	"strings"
	"time"
	"unicode/utf8"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/Juanrar/pomocli/internal/alarm"
	"github.com/Juanrar/pomocli/internal/cycle"
	"github.com/Juanrar/pomocli/internal/store"
)

type screen int

const (
	screenList screen = iota
	screenTimer
	screenWaiting
	screenSummary
	screenSettings
)

type listMode int

const (
	listNormal listMode = iota
	listAdding
	listEditing
	listConfirmDelete
)

const defaultWidth = 80

type Model struct {
	store          *store.Store
	cycle          *cycle.Cycle
	alarm          alarm.Player
	now            func() time.Time
	width, height  int
	screen         screen
	mode           listMode
	cursor         int // índice en store.Today(now())
	input          textinput.Model
	status         string // mensaje de una línea; se borra con la siguiente tecla
	audioWarning   string // aviso de audio; las teclas no lo borran
	scroll         int    // primera tarea visible en la lista
	settingsCursor int    // fila elegida en la pantalla de configuración

	activeTaskID string
	focusStart   time.Time
	tickID       int
	ringUntil    time.Time
	ringID       int
	confirmQuit  bool
}

func New(st *store.Store, cfg cycle.Config, al alarm.Player, now func() time.Time, status string) Model {
	in := textinput.New()
	in.Prompt = ""
	in.CharLimit = 200
	m := Model{
		store:        st,
		cycle:        cycle.New(cfg),
		alarm:        al,
		now:          now,
		width:        defaultWidth,
		input:        in,
		audioWarning: status,
	}
	m.setInputWidth()
	return m
}

// setInputWidth deja que el prompt más el texto entren en el bloque central de la lista.
func (m *Model) setInputWidth() {
	m.input.SetWidth(max(m.contentWidth()-len("Nueva tarea: ")-3, 5))
}

// cmdKey devuelve la tecla para comparar comandos; las letras van en minúscula
// para que Caps Lock no rompa los atajos.
func cmdKey(msg tea.KeyPressMsg) string {
	s := msg.String()
	if utf8.RuneCountInString(s) == 1 {
		return strings.ToLower(s)
	}
	return s
}

func (m Model) warningLine(w int) string {
	if m.audioWarning == "" {
		return ""
	}
	return warnStyle.Render(truncate(m.audioWarning, max(w, 1)))
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.update(msg)
	nm := next.(Model)
	nm.fixScroll()
	return nm, cmd
}

func (m Model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.setInputWidth()
		return m, nil
	case ringEndedMsg:
		if msg.id == m.ringID {
			m.ringUntil = time.Time{}
		}
		return m, nil
	case tea.FocusMsg:
		// Clicar la ventana de la terminal silencia la alarma, igual que una tecla.
		if m.ringing() {
			m.silence()
		}
		return m, nil
	case tickMsg:
		return m.handleTick(msg)
	case tea.KeyPressMsg:
		if m.ringing() {
			m.silence()
			return m, nil
		}
		m.status = ""
		switch m.screen {
		case screenSummary:
			return m.updateSummary(msg)
		case screenSettings:
			return m.updateSettings(msg)
		case screenTimer:
			return m.updateTimer(msg)
		case screenWaiting:
			return m.updateWaiting(msg)
		}
		return m.updateList(msg)
	}
	if m.screen == screenList && (m.mode == listAdding || m.mode == listEditing) {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) View() tea.View {
	var s string
	switch m.screen {
	case screenSummary:
		s = m.viewSummary()
	case screenSettings:
		s = m.viewSettings()
	case screenTimer:
		s = m.viewTimer()
	case screenWaiting:
		s = m.viewWaiting()
	default:
		s = m.viewList()
	}
	v := tea.NewView(s)
	v.AltScreen = true
	v.ReportFocus = true
	return v
}

// silence corta la alarma que suena.
func (m *Model) silence() {
	m.alarm.Stop()
	m.ringUntil = time.Time{}
}

func (m Model) viewWidth() int {
	if m.width <= 0 {
		return defaultWidth
	}
	return m.width
}
