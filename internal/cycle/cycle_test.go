package cycle

import (
	"testing"
	"time"
)

var (
	t0  = time.Date(2026, 11, 19, 10, 0, 0, 0, time.Local)
	cfg = Config{Focus: 25 * time.Minute, Short: 5 * time.Minute, Long: 15 * time.Minute, Every: 4}
)

func TestDefaultConfig(t *testing.T) {
	if DefaultConfig() != cfg {
		t.Fatalf("DefaultConfig() = %+v, quiero %+v", DefaultConfig(), cfg)
	}
}

func TestStartFocus(t *testing.T) {
	c := New(cfg)
	if err := c.StartFocus(t0); err != nil {
		t.Fatal(err)
	}
	if c.Phase() != Focus {
		t.Fatalf("fase = %v", c.Phase())
	}
	if got := c.Remaining(t0); got != 25*time.Minute {
		t.Fatalf("Remaining(t0) = %v", got)
	}
	if got := c.Remaining(t0.Add(10 * time.Minute)); got != 15*time.Minute {
		t.Fatalf("Remaining(t0+10m) = %v", got)
	}
	if c.Position() != 1 {
		t.Fatalf("Position = %d", c.Position())
	}
}

func TestStartFocusBusy(t *testing.T) {
	c := New(cfg)
	_ = c.StartFocus(t0)
	if err := c.StartFocus(t0); err != ErrBusy {
		t.Fatalf("err = %v, quiero ErrBusy", err)
	}
}

func TestFocusCompletesToShortBreak(t *testing.T) {
	c := New(cfg)
	_ = c.StartFocus(t0)
	now := t0.Add(25 * time.Minute)
	if ev := c.Tick(now); ev != FocusCompleted {
		t.Fatalf("ev = %v", ev)
	}
	if c.Phase() != ShortBreak {
		t.Fatalf("fase = %v", c.Phase())
	}
	if got := c.Remaining(now); got != 5*time.Minute {
		t.Fatalf("Remaining = %v", got)
	}
}

func TestFourthFocusIsLongBreak(t *testing.T) {
	c := New(cfg)
	now := t0
	for i := 1; i <= 4; i++ {
		if err := c.StartFocus(now); err != nil {
			t.Fatal(err)
		}
		now = now.Add(25 * time.Minute)
		if ev := c.Tick(now); ev != FocusCompleted {
			t.Fatalf("foco %d: ev = %v", i, ev)
		}
		if i < 4 {
			if c.Phase() != ShortBreak {
				t.Fatalf("foco %d: fase = %v", i, c.Phase())
			}
			c.SkipBreak()
		}
	}
	if c.Phase() != LongBreak {
		t.Fatalf("fase = %v, quiero LongBreak", c.Phase())
	}
	if got := c.Remaining(now); got != 15*time.Minute {
		t.Fatalf("Remaining = %v", got)
	}
	if c.Position() != 1 {
		t.Fatalf("Position = %d", c.Position())
	}
}

func TestBreakCompletesToWaiting(t *testing.T) {
	c := New(cfg)
	_ = c.StartFocus(t0)
	now := t0.Add(25 * time.Minute)
	c.Tick(now)
	now = now.Add(5 * time.Minute)
	if ev := c.Tick(now); ev != BreakCompleted {
		t.Fatalf("ev = %v", ev)
	}
	if c.Phase() != Waiting {
		t.Fatalf("fase = %v", c.Phase())
	}
	if c.Remaining(now) != 0 {
		t.Fatalf("Remaining = %v", c.Remaining(now))
	}
	if c.Position() != 2 {
		t.Fatalf("Position = %d", c.Position())
	}
}

func TestPauseResumeNoDrift(t *testing.T) {
	c := New(cfg)
	_ = c.StartFocus(t0)
	c.Pause(t0.Add(10 * time.Minute))
	if !c.Paused() {
		t.Fatal("deberia estar pausado")
	}
	if got := c.Remaining(t0.Add(20 * time.Minute)); got != 15*time.Minute {
		t.Fatalf("Remaining pausado = %v", got)
	}
	if ev := c.Tick(t0.Add(40 * time.Minute)); ev != None {
		t.Fatalf("ev = %v", ev)
	}
	c.Resume(t0.Add(40 * time.Minute))
	if c.Paused() {
		t.Fatal("no deberia estar pausado")
	}
	if got := c.Remaining(t0.Add(40 * time.Minute)); got != 15*time.Minute {
		t.Fatalf("Remaining tras reanudar = %v", got)
	}
	if ev := c.Tick(t0.Add(55 * time.Minute)); ev != FocusCompleted {
		t.Fatalf("ev = %v", ev)
	}
}

func TestCancelDoesNotCount(t *testing.T) {
	c := New(cfg)
	_ = c.StartFocus(t0)
	c.Cancel()
	if c.Phase() != Idle {
		t.Fatalf("fase = %v", c.Phase())
	}
	if c.Position() != 1 {
		t.Fatalf("Position = %d", c.Position())
	}
	_ = c.StartFocus(t0)
	c.Tick(t0.Add(25 * time.Minute))
	if c.Phase() != ShortBreak {
		t.Fatalf("fase = %v", c.Phase())
	}
}

func TestSkipBreak(t *testing.T) {
	c := New(cfg)
	_ = c.StartFocus(t0)
	now := t0.Add(25 * time.Minute)
	c.Tick(now)
	c.SkipBreak()
	if c.Phase() != Waiting {
		t.Fatalf("fase = %v", c.Phase())
	}
	if ev := c.Tick(now.Add(time.Hour)); ev != None {
		t.Fatalf("ev = %v", ev)
	}
}

func TestTickAfterLongGapCompletesOnce(t *testing.T) {
	c := New(cfg)
	_ = c.StartFocus(t0)
	now := t0.Add(2 * time.Hour)
	if ev := c.Tick(now); ev != FocusCompleted {
		t.Fatalf("ev = %v", ev)
	}
	if c.Phase() != ShortBreak {
		t.Fatalf("fase = %v", c.Phase())
	}
	if got := c.Remaining(now); got != 5*time.Minute {
		t.Fatalf("Remaining = %v", got)
	}
	if ev := c.Tick(now); ev != None {
		t.Fatalf("segundo tick ev = %v", ev)
	}
}

func TestRemainingNeverNegative(t *testing.T) {
	c := New(cfg)
	_ = c.StartFocus(t0)
	if got := c.Remaining(t0.Add(30 * time.Minute)); got != 0 {
		t.Fatalf("Remaining = %v", got)
	}
}
