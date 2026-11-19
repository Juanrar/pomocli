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
	case tea.KeyPressMsg:
		m.status = ""
		if m.screen == screenSummary {
			return m.updateSummary(msg)
		}
		return m.updateList(msg)
	}
	return m, nil
}

func (m Model) View() tea.View {
	var s string
	switch m.screen {
	case screenSummary:
		s = m.viewSummary()
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
