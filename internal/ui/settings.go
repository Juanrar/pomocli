package ui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/Juanrar/pomocli/internal/cycle"
	"github.com/Juanrar/pomocli/internal/store"
)

// Filas de la pantalla de configuración, en orden.
const (
	settingFocus = iota
	settingShort
	settingLong
	settingEvery
	settingCount
)

const (
	settingMin       = time.Minute
	settingMaxFocus  = 180 * time.Minute
	settingMaxBreak  = 60 * time.Minute
	settingFocusStep = 5 * time.Minute
	settingMaxEvery  = 10
)

// LoadConfig devuelve la configuración guardada en st, o la de fábrica si no hay
// ninguna válida.
func LoadConfig(st *store.Store) cycle.Config {
	s, ok := st.Settings()
	if !ok || s.FocusSeconds <= 0 || s.ShortSeconds <= 0 || s.LongSeconds <= 0 || s.Every < 1 {
		return cycle.DefaultConfig()
	}
	return cycle.Config{
		Focus: time.Duration(s.FocusSeconds) * time.Second,
		Short: time.Duration(s.ShortSeconds) * time.Second,
		Long:  time.Duration(s.LongSeconds) * time.Second,
		Every: s.Every,
	}
}

func saveConfig(st *store.Store, cfg cycle.Config) error {
	return st.SetSettings(store.Settings{
		FocusSeconds: int(cfg.Focus / time.Second),
		ShortSeconds: int(cfg.Short / time.Second),
		LongSeconds:  int(cfg.Long / time.Second),
		Every:        cfg.Every,
	})
}

// adjustSetting suma o resta un paso a la fila field, dentro de sus límites.
func adjustSetting(cfg cycle.Config, field, dir int) cycle.Config {
	switch field {
	case settingFocus:
		cfg.Focus = clampDur(cfg.Focus+time.Duration(dir)*settingFocusStep, settingMin, settingMaxFocus)
	case settingShort:
		cfg.Short = clampDur(cfg.Short+time.Duration(dir)*settingMin, settingMin, settingMaxBreak)
	case settingLong:
		cfg.Long = clampDur(cfg.Long+time.Duration(dir)*settingMin, settingMin, settingMaxBreak)
	case settingEvery:
		cfg.Every = min(max(cfg.Every+dir, 1), settingMaxEvery)
	}
	return cfg
}

func clampDur(d, lo, hi time.Duration) time.Duration {
	return min(max(d, lo), hi)
}

func (m Model) updateSettings(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	cfg := m.cycle.Config()
	switch cmdKey(msg) {
	case "up", "k":
		if m.settingsCursor > 0 {
			m.settingsCursor--
		}
	case "down", "j":
		if m.settingsCursor < settingCount-1 {
			m.settingsCursor++
		}
	case "left", "h", "-":
		m.changeSetting(adjustSetting(cfg, m.settingsCursor, -1))
	case "right", "l", "+":
		m.changeSetting(adjustSetting(cfg, m.settingsCursor, 1))
	case "esc":
		m.screen = screenList
	case "q", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

// changeSetting aplica cfg al próximo foco y lo guarda. El timer no corre en esta
// pantalla, así que el ciclo empieza de cero con la nueva configuración.
func (m *Model) changeSetting(cfg cycle.Config) {
	if cfg == m.cycle.Config() {
		return
	}
	m.cycle = cycle.New(cfg)
	if err := saveConfig(m.store, cfg); err != nil {
		m.fail(err)
	}
}

func settingValue(cfg cycle.Config, field int) string {
	switch field {
	case settingFocus:
		return fmt.Sprintf("%d min", int(cfg.Focus/time.Minute))
	case settingShort:
		return fmt.Sprintf("%d min", int(cfg.Short/time.Minute))
	case settingLong:
		return fmt.Sprintf("%d min", int(cfg.Long/time.Minute))
	default:
		return fmt.Sprintf("%d focos", cfg.Every)
	}
}

var settingLabels = [settingCount]string{
	"Foco",
	"Descanso corto",
	"Descanso largo",
	"Focos antes del descanso largo",
}

func (m Model) viewSettings() string {
	w := m.contentWidth()
	cfg := m.cycle.Config()
	var b strings.Builder
	line := func(s string) {
		b.WriteString(lipgloss.PlaceHorizontal(w, lipgloss.Center, s) + "\n")
	}
	line(titleStyle.Render("Configuración"))
	b.WriteString(faintStyle.Render(strings.Repeat("─", w)) + "\n")
	for i := 0; i < settingCount; i++ {
		prefix := "   "
		if i == m.settingsCursor {
			prefix = " > "
		}
		label := settingLabels[i]
		value := lipgloss.NewStyle().Bold(i == m.settingsCursor).Render(settingValue(cfg, i))
		gap := max(w-lipgloss.Width(prefix+label)-lipgloss.Width(value)-1, 1)
		b.WriteString(prefix + label + strings.Repeat(" ", gap) + value + "\n")
	}
	b.WriteString(faintStyle.Render(strings.Repeat("─", w)) + "\n")
	if m.status != "" {
		line(truncate(m.status, w-2))
	}
	line(keyHelp([2]string{"↑↓", "elegir"}, [2]string{"←→", "cambiar"}, [2]string{"esc", "volver"}))
	return lipgloss.Place(m.viewWidth(), m.height, lipgloss.Center, lipgloss.Center,
		strings.TrimSuffix(b.String(), "\n"))
}
