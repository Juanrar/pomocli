package ui

import (
	"testing"
	"time"

	"github.com/Juanrar/pomocli/internal/cycle"
)

func TestSettingsChangeAndPersist(t *testing.T) {
	m, st, _ := newTestModel(t)
	m = press(m, key("c"))
	if m.screen != screenSettings {
		t.Fatalf("c no abrió la configuración: screen=%v", m.screen)
	}
	m = press(m, key("right"))
	if got := m.cycle.Config().Focus; got != 55*time.Minute {
		t.Fatalf("foco = %v, quiero 55m", got)
	}
	s, ok := st.Settings()
	if !ok || s.FocusSeconds != 55*60 {
		t.Fatalf("no se guardó el foco: %+v ok=%v", s, ok)
	}
	if c := content(m); !contains(c, "55 min") {
		t.Errorf("la pantalla no muestra 55 min:\n%s", c)
	}
	m = press(m, key("esc"))
	if m.screen != screenList {
		t.Fatalf("esc no volvió a la lista: screen=%v", m.screen)
	}
}

func TestSettingsRowsAndLimits(t *testing.T) {
	m, _, _ := newTestModel(t)
	m = press(m, key("c"))
	m = press(m, key("down"), key("down"), key("down"))
	for i := 0; i < 20; i++ {
		m = press(m, key("right"))
	}
	if got := m.cycle.Config().Every; got != settingMaxEvery {
		t.Errorf("every = %d, quiero %d (tope)", got, settingMaxEvery)
	}
	m = press(m, key("up"), key("up"), key("up"))
	for i := 0; i < 30; i++ {
		m = press(m, key("left"))
	}
	if got := m.cycle.Config().Focus; got != settingMin {
		t.Errorf("foco = %v, quiero el mínimo %v", got, settingMin)
	}
}

func TestLoadConfigFromStore(t *testing.T) {
	m, st, _ := newTestModel(t)
	if got := LoadConfig(st); got != cycle.DefaultConfig() {
		t.Fatalf("sin datos guardados: %+v", got)
	}
	want := cycle.Config{Focus: 40 * time.Minute, Short: 7 * time.Minute, Long: 20 * time.Minute, Every: 3}
	m = press(m, key("c"))
	m.changeSetting(want)
	if got := LoadConfig(st); got != want {
		t.Fatalf("LoadConfig = %+v, quiero %+v", got, want)
	}
}
