// Comando pomocli: pomodoro de terminal con lista de tareas.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/Juanrar/pomocli/internal/alarm"
	"github.com/Juanrar/pomocli/internal/cycle"
	"github.com/Juanrar/pomocli/internal/store"
	"github.com/Juanrar/pomocli/internal/ui"
)

const errInvalidConfig = "-focus, -short y -long tienen que ser mayores que cero, y -every al menos 1"

func newFlagSet(cfg *cycle.Config) *flag.FlagSet {
	def := cycle.DefaultConfig()
	fs := flag.NewFlagSet("pomocli", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.DurationVar(&cfg.Focus, "focus", def.Focus, "duración del foco")
	fs.DurationVar(&cfg.Short, "short", def.Short, "duración del descanso corto")
	fs.DurationVar(&cfg.Long, "long", def.Long, "duración del descanso largo")
	fs.IntVar(&cfg.Every, "every", def.Every, "focos antes de cada descanso largo")
	return fs
}

// parseConfig lee los flags de duración. No imprime nada: devuelve los errores.
func parseConfig(args []string) (cycle.Config, error) {
	var cfg cycle.Config
	fs := newFlagSet(&cfg)
	if err := fs.Parse(args); err != nil {
		return cycle.Config{}, err
	}
	if fs.NArg() > 0 {
		return cycle.Config{}, fmt.Errorf("argumento inesperado %q; usá -focus, -short, -long y -every", fs.Arg(0))
	}
	if cfg.Focus <= 0 || cfg.Short <= 0 || cfg.Long <= 0 || cfg.Every < 1 {
		return cycle.Config{}, errors.New(errInvalidConfig)
	}
	return cfg, nil
}

func main() {
	cfg, err := parseConfig(os.Args[1:])
	if err != nil {
		var dummy cycle.Config
		fs := newFlagSet(&dummy)
		fs.SetOutput(os.Stderr)
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(os.Stderr, "Uso: pomocli [flags]")
			fs.PrintDefaults()
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, "Uso: pomocli [flags]")
		fs.PrintDefaults()
		os.Exit(2)
	}

	path, err := store.DefaultPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	st, err := store.Open(path)
	if err != nil {
		if errors.Is(err, store.ErrCorrupt) {
			fmt.Fprintf(os.Stderr, "No se pudo leer %s: %v. No toqué el archivo.\n", path, err)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}

	al, err := alarm.New()
	status := ""
	if err != nil {
		status = fmt.Sprintf("Sin audio (%v). Uso el bell de la terminal. Conectá los auriculares y reiniciá pomocli.", err)
	}

	if _, err := tea.NewProgram(ui.New(st, cfg, al, time.Now, status)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
