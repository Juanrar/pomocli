// Package alarm reproduce una alarma sonora sintetizada en código.
package alarm

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/speaker"
)

const (
	// MaxRing es lo máximo que suena la alarma si nadie la corta.
	MaxRing = 60 * time.Second
	// SampleRate es la frecuencia de muestreo del parlante.
	SampleRate beep.SampleRate = 44100
	// Amplitude es la amplitud de la senoidal (0 a 1).
	Amplitude = 0.25

	toneHz   = 880.0
	toneLen  = 150 * time.Millisecond
	gapLen   = 100 * time.Millisecond
	cycleLen = time.Second
	rampLen  = 5 * time.Millisecond
)

// Player controla la alarma.
type Player interface {
	// Start empieza a sonar; si ya sonaba, reinicia. Se corta sola a los MaxRing.
	Start()
	// Stop corta el sonido; no-op si no suena.
	Stop()
}

type speakerPlayer struct{}

func (speakerPlayer) Start() {
	speaker.Clear()
	speaker.Play(pattern(SampleRate, MaxRing))
}

func (speakerPlayer) Stop() { speaker.Clear() }

// bellPlayer es el respaldo cuando no hay audio: manda BEL a la terminal.
type bellPlayer struct{}

func (bellPlayer) Start() { fmt.Fprint(os.Stdout, "\a") }
func (bellPlayer) Stop()  {}

// New inicializa el parlante y devuelve un Player de audio. Si falla,
// devuelve un Player de respaldo (campana de terminal) junto con el error.
func New() (Player, error) {
	if err := speaker.Init(SampleRate, SampleRate.N(100*time.Millisecond)); err != nil {
		return bellPlayer{}, err
	}
	return speakerPlayer{}, nil
}

// pattern devuelve la alarma: un streamer finito de duración total d.
// Ciclo de 1 s: tono 150 ms, silencio 100 ms, tono 150 ms, silencio 600 ms.
func pattern(sr beep.SampleRate, d time.Duration) beep.Streamer {
	total := sr.N(d)
	cycle := sr.N(cycleLen)
	tone := sr.N(toneLen)
	gap := sr.N(gapLen)
	ramp := sr.N(rampLen)
	pos := 0

	value := func(i int) float64 {
		c := i % cycle
		var t int // índice dentro del tono
		switch {
		case c < tone:
			t = c
		case c >= tone+gap && c < 2*tone+gap:
			t = c - tone - gap
		default:
			return 0
		}
		env := 1.0
		if t < ramp {
			env = float64(t) / float64(ramp)
		} else if t >= tone-ramp {
			env = float64(tone-1-t) / float64(ramp)
		}
		return Amplitude * env * math.Sin(2*math.Pi*toneHz*float64(i)/float64(sr))
	}

	return beep.StreamerFunc(func(samples [][2]float64) (int, bool) {
		if pos >= total {
			return 0, false
		}
		n := 0
		for n < len(samples) && pos < total {
			v := value(pos)
			samples[n] = [2]float64{v, v}
			n++
			pos++
		}
		return n, true
	})
}
