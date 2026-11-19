// Package ui implementa la interfaz de terminal con Bubble Tea.
package ui

import (
	"time"

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
	store         *store.Store
	cycle         *cycle.Cycle
	alarm         alarm.Player
	now           func() time.Time
	width, height int
	screen        screen
	mode          listMode
	cursor        int // índice en store.Today(now())
	input         textinput.Model
	status        string // mensaje de una línea; se borra con la siguiente tecla

	activeTaskID string
	focusStart   time.Time
	tickID       int
	ringUntil    time.Time
	confirmQuit  bool
}

func New(st *store.Store, cfg cycle.Config, al alarm.Player, now func() time.Time, status string) Model {
	in := textinput.New()
	in.Prompt = ""
	in.CharLimit = 200
	return Model{
		store:  st,
		cycle:  cycle.New(cfg),
		alarm:  al,
		now:    now,
		width:  defaultWidth,
		input:  in,
		status: status,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tickMsg:
		return m.handleTick(msg)
	case tea.KeyPressMsg:
		if m.ringing() {
			m.alarm.Stop()
			m.ringUntil = time.Time{}
			return m, nil
		}
		m.status = ""
		switch m.screen {
		case screenSummary:
			return m.updateSummary(msg)
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
	case screenTimer:
		s = m.viewTimer()
	case screenWaiting:
		s = m.viewWaiting()
	default:
		s = m.viewList()
	}
	v := tea.NewView(s)
	v.AltScreen = true
	return v
}

func (m Model) viewWidth() int {
	if m.width <= 0 {
		return defaultWidth
	}
	return m.width
}
