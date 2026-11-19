package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var day = time.Date(2026, 11, 19, 9, 0, 0, 0, time.Local)

func tmpPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "sub", "data.json")
}

func mustOpen(t *testing.T, path string) *Store {
	t.Helper()
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return s
}

func mustAdd(t *testing.T, s *Store, title string, now time.Time) Task {
	t.Helper()
	task, err := s.Add(title, now)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	return task
}

func TestOpenCreatesFile(t *testing.T) {
	path := tmpPath(t)
	mustOpen(t, path)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("archivo no existe: %v", err)
	}
	if !strings.Contains(string(b), `"version": 1`) {
		t.Fatalf("contenido inesperado: %s", b)
	}
	for _, want := range []string{`"tasks": []`, `"pomodoros": []`} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("falta %s en: %s", want, b)
		}
	}
}

func TestOpenCorruptNotOverwritten(t *testing.T) {
	path := tmpPath(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{roto"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Open(path)
	if !errors.Is(err, ErrCorrupt) {
		t.Fatalf("esperaba ErrCorrupt, got %v", err)
	}
	b, _ := os.ReadFile(path)
	if string(b) != "{roto" {
		t.Fatalf("archivo modificado: %q", b)
	}
}

func TestAddAndReopen(t *testing.T) {
	path := tmpPath(t)
	s := mustOpen(t, path)
	mustAdd(t, s, "Lección 0007 DOM", day)
	s2 := mustOpen(t, path)
	got := s2.Today(day)
	if len(got) != 1 || got[0].Title != "Lección 0007 DOM" || got[0].Estimate != 1 || got[0].Done {
		t.Fatalf("inesperado: %+v", got)
	}
}

func TestAddEmptyTitle(t *testing.T) {
	s := mustOpen(t, tmpPath(t))
	if _, err := s.Add("   ", day); !errors.Is(err, ErrEmptyTitle) {
		t.Fatalf("esperaba ErrEmptyTitle, got %v", err)
	}
	if len(s.Today(day)) != 0 {
		t.Fatal("Today debería estar vacío")
	}
}

func TestUnicodeTitleRoundTrip(t *testing.T) {
	path := tmpPath(t)
	s := mustOpen(t, path)
	mustAdd(t, s, "Ñandú 🍅 acción", day)
	got := mustOpen(t, path).Today(day)
	if len(got) != 1 || got[0].Title != "Ñandú 🍅 acción" {
		t.Fatalf("inesperado: %+v", got)
	}
}

func TestTodayFilters(t *testing.T) {
	s := mustOpen(t, tmpPath(t))
	a := mustAdd(t, s, "A", day.Add(-48*time.Hour))
	b := mustAdd(t, s, "B", day.Add(-47*time.Hour))
	c := mustAdd(t, s, "C", day.Add(-46*time.Hour))
	if err := s.SetDone(b.ID, true, day.Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := s.SetDone(c.ID, true, day); err != nil {
		t.Fatal(err)
	}
	got := s.Today(day)
	if len(got) != 2 || got[0].ID != a.ID || got[1].ID != c.ID {
		t.Fatalf("inesperado: %+v", got)
	}
}

func TestSetEstimateMinimum(t *testing.T) {
	s := mustOpen(t, tmpPath(t))
	task := mustAdd(t, s, "T", day)
	if err := s.SetEstimate(task.ID, 0); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Task(task.ID); got.Estimate != 1 {
		t.Fatalf("estimate %d", got.Estimate)
	}
	if err := s.SetEstimate(task.ID, 3); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Task(task.ID); got.Estimate != 3 {
		t.Fatalf("estimate %d", got.Estimate)
	}
}

func TestRenameDeleteNotFound(t *testing.T) {
	s := mustOpen(t, tmpPath(t))
	if err := s.Rename("nope", "x"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Rename: %v", err)
	}
	if err := s.Delete("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Delete: %v", err)
	}
}

func TestProgressAndSummary(t *testing.T) {
	s := mustOpen(t, tmpPath(t))
	T := mustAdd(t, s, "T", day)
	U := mustAdd(t, s, "U", day)
	for _, p := range []Pomodoro{
		{TaskID: T.ID, TaskTitle: "T", Start: day.Add(-25 * time.Hour), End: day.Add(-24 * time.Hour), FocusSeconds: 1500},
		{TaskID: T.ID, TaskTitle: "T", Start: day, End: day.Add(time.Hour), FocusSeconds: 1500},
		{TaskID: U.ID, TaskTitle: "U", Start: day.Add(time.Hour), End: day.Add(2 * time.Hour), FocusSeconds: 1500},
	} {
		if err := s.AddPomodoro(p); err != nil {
			t.Fatal(err)
		}
	}
	if got := s.Progress(T.ID); got != 2 {
		t.Fatalf("Progress %d", got)
	}
	sum := s.DaySummary(day)
	if sum.Count != 2 || sum.FocusMinutes != 50 || len(sum.ByTask) != 2 ||
		sum.ByTask[0] != (TaskCount{T.ID, "T", 1}) || sum.ByTask[1] != (TaskCount{U.ID, "U", 1}) {
		t.Fatalf("inesperado: %+v", sum)
	}
}

func TestDeleteKeepsHistory(t *testing.T) {
	s := mustOpen(t, tmpPath(t))
	task := mustAdd(t, s, "Original", day)
	if err := s.AddPomodoro(Pomodoro{TaskID: task.ID, TaskTitle: "Original", Start: day, End: day.Add(time.Hour), FocusSeconds: 1500}); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(task.ID); err != nil {
		t.Fatal(err)
	}
	sum := s.DaySummary(day)
	if sum.Count != 1 || sum.ByTask[0].Title != "Original" {
		t.Fatalf("inesperado: %+v", sum)
	}
}

func TestSaveFailureReturnsError(t *testing.T) {
	path := tmpPath(t)
	s := mustOpen(t, path)
	dir := filepath.Dir(path)
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Add("algo", day); err == nil {
		t.Fatal("esperaba error al guardar")
	}
}
