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
			if v > 0.25 {
				t.Fatalf("muestra %v supera 0.25", v)
			}
			if v > max {
				max = v
			}
		}
	}
	if max <= 0.2 {
		t.Fatalf("pico %v, quiero > 0.2", max)
	}
}

func TestPatternSilenceGap(t *testing.T) {
	all := readAll(t, 3*time.Second)
	for i := 7056; i <= 10584; i++ {
		if all[i] != [2]float64{} {
			t.Fatalf("muestra %d no es silencio: %v", i, all[i])
		}
	}
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
