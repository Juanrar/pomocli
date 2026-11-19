// Package store guarda las tareas y el historial de pomodoros en un archivo JSON.
package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Task struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Estimate  int        `json:"estimate"`
	Done      bool       `json:"done"`
	CreatedAt time.Time  `json:"createdAt"`
	DoneAt    *time.Time `json:"doneAt"`
}

type Pomodoro struct {
	TaskID       string    `json:"taskId"`
	TaskTitle    string    `json:"taskTitle"`
	Start        time.Time `json:"start"`
	End          time.Time `json:"end"`
	FocusSeconds int       `json:"focusSeconds"`
}

type Data struct {
	Version   int        `json:"version"`
	Tasks     []Task     `json:"tasks"`
	Pomodoros []Pomodoro `json:"pomodoros"`
}

type TaskCount struct {
	TaskID, Title string
	Count         int
}

type Summary struct {
	Count        int
	FocusMinutes int
	ByTask       []TaskCount
}

var (
	ErrCorrupt    = errors.New("archivo de datos corrupto")
	ErrEmptyTitle = errors.New("el título no puede estar vacío")
	ErrNotFound   = errors.New("tarea no encontrada")
)

// Store mantiene los datos en memoria y los guarda en disco en cada cambio.
type Store struct {
	path string
	data Data
}

// DefaultPath devuelve <config del usuario>/pomocli/data.json.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "pomocli", "data.json"), nil
}

// Open carga el archivo, o lo crea vacío si no existe. Si el JSON está roto,
// devuelve un error que envuelve ErrCorrupt y no toca el archivo.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	s := &Store{path: path}
	b, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		s.data = Data{Version: 1}
		if err := s.save(); err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	default:
		if err := json.Unmarshal(b, &s.data); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrCorrupt, err)
		}
		if s.data.Version == 0 {
			s.data.Version = 1
		}
	}
	return s, nil
}

func (s *Store) Path() string { return s.path }

func sameDay(a, b time.Time) bool {
	a = a.In(b.Location())
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// Today devuelve las tareas sin terminar y las terminadas hoy, en orden de creación.
func (s *Store) Today(now time.Time) []Task {
	var out []Task
	for _, t := range s.data.Tasks {
		if !t.Done || (t.DoneAt != nil && sameDay(*t.DoneAt, now)) {
			out = append(out, t)
		}
	}
	return out
}

func (s *Store) Task(id string) (Task, bool) {
	if i := s.index(id); i >= 0 {
		return s.data.Tasks[i], true
	}
	return Task{}, false
}

func (s *Store) Add(title string, now time.Time) (Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Task{}, ErrEmptyTitle
	}
	id, err := newID()
	if err != nil {
		return Task{}, err
	}
	t := Task{ID: id, Title: title, Estimate: 1, CreatedAt: now}
	s.data.Tasks = append(s.data.Tasks, t)
	if err := s.save(); err != nil {
		return Task{}, err
	}
	return t, nil
}

func (s *Store) Rename(id, title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return ErrEmptyTitle
	}
	i := s.index(id)
	if i < 0 {
		return ErrNotFound
	}
	s.data.Tasks[i].Title = title
	return s.save()
}

func (s *Store) SetEstimate(id string, n int) error {
	i := s.index(id)
	if i < 0 {
		return ErrNotFound
	}
	if n < 1 {
		n = 1
	}
	s.data.Tasks[i].Estimate = n
	return s.save()
}

func (s *Store) SetDone(id string, done bool, now time.Time) error {
	i := s.index(id)
	if i < 0 {
		return ErrNotFound
	}
	s.data.Tasks[i].Done = done
	if done {
		s.data.Tasks[i].DoneAt = &now
	} else {
		s.data.Tasks[i].DoneAt = nil
	}
	return s.save()
}

// Delete borra la tarea y deja sus pomodoros en el historial.
func (s *Store) Delete(id string) error {
	i := s.index(id)
	if i < 0 {
		return ErrNotFound
	}
	s.data.Tasks = append(s.data.Tasks[:i], s.data.Tasks[i+1:]...)
	return s.save()
}

func (s *Store) AddPomodoro(p Pomodoro) error {
	s.data.Pomodoros = append(s.data.Pomodoros, p)
	return s.save()
}

// Progress cuenta los pomodoros de la tarea, de todos los días.
func (s *Store) Progress(id string) int {
	n := 0
	for _, p := range s.data.Pomodoros {
		if p.TaskID == id {
			n++
		}
	}
	return n
}

// DaySummary resume los pomodoros cuyo End cae en la fecha local de now.
func (s *Store) DaySummary(now time.Time) Summary {
	var sum Summary
	seconds := 0
	pos := map[string]int{}
	for _, p := range s.data.Pomodoros {
		if !sameDay(p.End, now) {
			continue
		}
		sum.Count++
		seconds += p.FocusSeconds
		i, ok := pos[p.TaskID]
		if !ok {
			i = len(sum.ByTask)
			pos[p.TaskID] = i
			sum.ByTask = append(sum.ByTask, TaskCount{TaskID: p.TaskID})
		}
		sum.ByTask[i].Count++
		sum.ByTask[i].Title = p.TaskTitle
	}
	sum.FocusMinutes = seconds / 60
	return sum
}

func (s *Store) index(id string) int {
	for i, t := range s.data.Tasks {
		if t.ID == id {
			return i
		}
	}
	return -1
}

func newID() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// save escribe a un temporal en la misma carpeta y lo renombra sobre el archivo.
func (s *Store) save() (err error) {
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(s.path), "data-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer func() {
		if err != nil {
			os.Remove(name)
		}
	}()
	if _, err = tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, s.path)
}
