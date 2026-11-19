// Package alarm reproduce una alarma sonora sintetizada en código.
package alarm

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sync"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/speaker"
)

const (
	// MaxRing es lo máximo que suena la alarma si nadie la corta.
	MaxRing = 60 * time.Second
	// SampleRate es la frecuencia de muestreo del parlante.
	SampleRate beep.SampleRate = 44100
	// Amplitude es el pico de la onda (0 a 1). Es bajo a propósito: se escucha con auriculares.
	Amplitude = 0.125

	noteLen  = 110 * time.Millisecond
	noteStep = 120 * time.Millisecond
	cycleLen = 1200 * time.Millisecond
	rampLen  = 5 * time.Millisecond
)

// arpeggioHz son las notas del arpegio ascendente: do, mi, sol y do agudo.
var arpeggioHz = [...]float64{523.25, 659.25, 783.99, 1046.5}

// Player controla la alarma.
type Player interface {
	// Start empieza a sonar; si ya sonaba, reinicia. Se corta sola a los MaxRing.
	Start()
	// Stop corta el sonido; no-op si no suena.
	Stop()
}

// speakerPlayer toca el patrón por el parlante. El parlante queda suspendido
// cuando no suena, para no bloquear el sueño de Windows con un stream abierto.
type speakerPlayer struct {
	mu  sync.Mutex
	gen int // cambia en cada Start y Stop; un Suspend viejo no corta un timbre nuevo
}

func (p *speakerPlayer) Start() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.gen++
	g := p.gen
	_ = speaker.Resume()
	speaker.Clear()
	speaker.Play(beep.Seq(pattern(SampleRate, MaxRing), beep.Callback(func() {
		// En otra goroutine: el parlante tiene su lock tomado dentro del callback.
		go func() {
			p.mu.Lock()
			defer p.mu.Unlock()
			if p.gen == g {
				_ = speaker.Suspend()
			}
		}()
	})))
}

func (p *speakerPlayer) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.gen++
	speaker.Clear()
	_ = speaker.Suspend()
}

// bellPlayer es el respaldo cuando no hay audio: manda BEL a la terminal
// una vez por intervalo hasta Stop o hasta maxRing.
type bellPlayer struct {
	w        io.Writer
	interval time.Duration
	maxRing  time.Duration

	mu   sync.Mutex
	stop chan struct{} // nil si no suena
}

func newBellPlayer() *bellPlayer {
	return &bellPlayer{w: os.Stdout, interval: time.Second, maxRing: MaxRing}
}

func (b *bellPlayer) Start() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.stop != nil {
		close(b.stop)
	}
	stop := make(chan struct{})
	b.stop = stop
	go b.run(stop)
}

func (b *bellPlayer) run(stop chan struct{}) {
	deadline := time.NewTimer(b.maxRing)
	defer deadline.Stop()
	tick := time.NewTicker(b.interval)
	defer tick.Stop()
	b.ring(stop)
	for {
		select {
		case <-stop:
			return
		case <-deadline.C:
			return
		case <-tick.C:
			b.ring(stop)
		}
	}
}

// ring escribe BEL salvo que ya se haya pedido Stop.
func (b *bellPlayer) ring(stop chan struct{}) {
	b.mu.Lock()
	defer b.mu.Unlock()
	select {
	case <-stop:
		return
	default:
	}
	fmt.Fprint(b.w, "")
}

func (b *bellPlayer) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.stop != nil {
		close(b.stop)
		b.stop = nil
	}
}

// New inicializa el parlante y devuelve un Player de audio. Si falla o no hay
// dispositivo de salida, devuelve un Player de respaldo (campana de terminal)
// junto con el error.
func New() (Player, error) {
	if !hasOutputDevice() {
		return newBellPlayer(), errNoDevice
	}
	if err := speaker.Init(SampleRate, SampleRate.N(100*time.Millisecond)); err != nil {
		return newBellPlayer(), err
	}
	_ = speaker.Suspend()
	return &speakerPlayer{}, nil
}

var errNoDevice = errors.New("no hay dispositivo de salida de audio")

// pattern devuelve la alarma: un streamer finito de duración total d.
// Ciclo de 1,2 s: cuatro notas de 110 ms en onda triangular que arrancan cada
// 120 ms (do, mi, sol, do agudo) y después silencio hasta completar el ciclo.
func pattern(sr beep.SampleRate, d time.Duration) beep.Streamer {
	total := sr.N(d)
	cycle := sr.N(cycleLen)
	note := sr.N(noteLen)
	step := sr.N(noteStep)
	ramp := sr.N(rampLen)
	pos := 0

	value := func(i int) float64 {
		c := i % cycle
		k := c / step // número de nota dentro del ciclo
		t := c - k*step
		if k >= len(arpeggioHz) || t >= note {
			return 0
		}
		env := 1.0
		if t < ramp {
			env = float64(t) / float64(ramp)
		} else if t >= note-ramp {
			env = float64(note-1-t) / float64(ramp)
		}
		return Amplitude * env * triangle(arpeggioHz[k]*float64(t)/float64(sr))
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

// triangle es una onda triangular entre -1 y 1; p es la fase medida en ciclos.
func triangle(p float64) float64 {
	return 2*math.Abs(2*(p-math.Floor(p+0.5))) - 1
}
