// Package cycle implementa la máquina de estados del ciclo pomodoro.
package cycle

import (
	"errors"
	"time"
)

type Phase int

const (
	Idle Phase = iota // sin timer (pantalla de lista)
	Focus
	ShortBreak
	LongBreak
	Waiting // terminó el descanso, espera al usuario
)

type Config struct {
	Focus, Short, Long time.Duration
	Every              int // focos por descanso largo
}

// DefaultConfig devuelve 25m de foco, 5m y 15m de descanso, descanso largo cada 4 focos.
func DefaultConfig() Config {
	return Config{Focus: 25 * time.Minute, Short: 5 * time.Minute, Long: 15 * time.Minute, Every: 4}
}

type Event int

const (
	None           Event = iota
	FocusCompleted       // el foco llegó a cero; la fase ya es ShortBreak o LongBreak
	BreakCompleted       // el descanso llegó a cero; la fase ya es Waiting
)

var ErrBusy = errors.New("cycle: ya hay un timer corriendo")

type Cycle struct {
	cfg       Config
	phase     Phase
	streak    int
	deadline  time.Time
	paused    bool
	remaining time.Duration // restante guardado mientras está pausado
}

func New(cfg Config) *Cycle {
	return &Cycle{cfg: cfg}
}

func (c *Cycle) Config() Config { return c.cfg }

func (c *Cycle) Phase() Phase { return c.phase }

// StartFocus arranca un foco. Solo vale desde Idle o Waiting.
func (c *Cycle) StartFocus(now time.Time) error {
	if c.phase != Idle && c.phase != Waiting {
		return ErrBusy
	}
	c.phase = Focus
	c.paused = false
	c.deadline = now.Add(c.cfg.Focus)
	return nil
}

func (c *Cycle) hasTimer() bool {
	return c.phase == Focus || c.phase == ShortBreak || c.phase == LongBreak
}

func (c *Cycle) Pause(now time.Time) {
	if !c.hasTimer() || c.paused {
		return
	}
	c.remaining = max(c.deadline.Sub(now), 0)
	c.paused = true
}

func (c *Cycle) Resume(now time.Time) {
	if !c.paused {
		return
	}
	c.deadline = now.Add(c.remaining)
	c.paused = false
}

func (c *Cycle) Paused() bool { return c.paused }

func (c *Cycle) Remaining(now time.Time) time.Duration {
	if !c.hasTimer() {
		return 0
	}
	if c.paused {
		return c.remaining
	}
	return max(c.deadline.Sub(now), 0)
}

// Tick avanza el ciclo. Hace a lo sumo una transición por llamada.
func (c *Cycle) Tick(now time.Time) Event {
	if !c.hasTimer() || c.paused || now.Before(c.deadline) {
		return None
	}
	if c.phase == Focus {
		c.streak++
		if c.streak >= c.cfg.Every {
			c.phase = LongBreak
			c.streak = 0
			c.deadline = now.Add(c.cfg.Long)
		} else {
			c.phase = ShortBreak
			c.deadline = now.Add(c.cfg.Short)
		}
		return FocusCompleted
	}
	c.phase = Waiting
	return BreakCompleted
}

// SkipBreak pasa de un descanso a Waiting. No hace nada en otras fases.
func (c *Cycle) SkipBreak() {
	if c.phase == ShortBreak || c.phase == LongBreak {
		c.phase = Waiting
		c.paused = false
	}
}

// Cancel vuelve a Idle desde cualquier fase. El foco en curso no cuenta.
func (c *Cycle) Cancel() {
	c.phase = Idle
	c.paused = false
}

// Position devuelve 1..Every: la posición del foco actual o del próximo.
func (c *Cycle) Position() int { return c.streak + 1 }
