package alarm

import (
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
