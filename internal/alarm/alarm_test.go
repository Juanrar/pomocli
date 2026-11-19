package alarm

import (
	"bytes"
	"sync"
	"testing"
	"time"
)

func TestPatternLength(t *testing.T) {
	s := pattern(SampleRate, 3*time.Second)
	buf := make([][2]float64, 512)
	total := 0
	for {
		n, ok := s.Stream(buf)
		total += n
		if !ok {
			break
		}
	}
	if total != 132300 {
		t.Fatalf("muestras = %d, quiero 132300", total)
	}
}

func readAll(t *testing.T, d time.Duration) [][2]float64 {
	t.Helper()
	s := pattern(SampleRate, d)
	buf := make([][2]float64, 512)
	var out [][2]float64
	for {
		n, ok := s.Stream(buf)
		out = append(out, buf[:n]...)
		if !ok {
			return out
		}
	}
}

func TestPatternAmplitude(t *testing.T) {
	max := 0.0
	for _, f := range readAll(t, 3*time.Second) {
		for _, v := range f {
			if v < 0 {
				v = -v
			}
			if v > 0.125 {
				t.Fatalf("muestra %v supera 0.125", v)
			}
			if v > max {
				max = v
			}
		}
	}
	if max <= 0.1 {
		t.Fatalf("pico %v, quiero > 0.1", max)
	}
}

// El arpegio: 4 notas de 110 ms que arrancan cada 120 ms, después silencio hasta 1,2 s.
func TestPatternArpeggioNotesAndSilence(t *testing.T) {
	all := readAll(t, 3*time.Second)
	at := func(ms int) int { return SampleRate.N(time.Duration(ms) * time.Millisecond) }

	for k, mid := range []int{55, 175, 295, 415} {
		loud := false
		for i := at(mid - 20); i < at(mid+20); i++ {
			if all[i][0] != 0 {
				loud = true
				break
			}
		}
		if !loud {
			t.Fatalf("la nota %d (alrededor de %d ms) no suena", k+1, mid)
		}
	}

	silent := func(fromMs, toMs int) {
		t.Helper()
		for i := at(fromMs); i < at(toMs); i++ {
			if all[i] != [2]float64{} {
				t.Fatalf("muestra %d (entre %d y %d ms) no es silencio: %v", i, fromMs, toMs, all[i])
			}
		}
	}
	silent(111, 119)   // hueco entre la primera y la segunda nota
	silent(471, 1199)  // silencio después de la última nota
	silent(1671, 2399) // el ciclo se repite cada 1,2 s
}

type syncBuf struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuf) len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Len()
}

func TestBellRepeatsAndStops(t *testing.T) {
	buf := &syncBuf{}
	b := &bellPlayer{w: buf, interval: 10 * time.Millisecond, maxRing: time.Minute}
	b.Stop() // no-op sin sonar
	b.Start()
	time.Sleep(100 * time.Millisecond)
	if n := buf.len(); n < 3 {
		t.Fatalf("BEL escritos = %d, quiero al menos 3", n)
	}
	b.Stop()
	time.Sleep(30 * time.Millisecond)
	n := buf.len()
	time.Sleep(60 * time.Millisecond)
	if buf.len() != n {
		t.Fatalf("siguió escribiendo después de Stop")
	}
}

func TestBellRestartAndMaxRing(t *testing.T) {
	buf := &syncBuf{}
	b := &bellPlayer{w: buf, interval: 10 * time.Millisecond, maxRing: 50 * time.Millisecond}
	b.Start()
	b.Start() // reinicia, no duplica
	time.Sleep(150 * time.Millisecond)
	n := buf.len()
	time.Sleep(60 * time.Millisecond)
	if buf.len() != n {
		t.Fatalf("siguió escribiendo después de maxRing")
	}
	if n == 0 {
		t.Fatal("no sonó")
	}
}
