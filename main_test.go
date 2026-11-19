package main

import (
	"strings"
	"testing"
	"time"

	"github.com/Juanrar/pomocli/internal/cycle"
)

func TestParseConfigDefaults(t *testing.T) {
	got, err := parseConfig(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != cycle.DefaultConfig() {
		t.Fatalf("got %+v, want %+v", got, cycle.DefaultConfig())
	}
}

func TestParseConfigCustom(t *testing.T) {
	got, err := parseConfig([]string{"-focus", "50m", "-short", "10m", "-long", "20m", "-every", "2"})
	if err != nil {
		t.Fatal(err)
	}
	want := cycle.Config{Focus: 50 * time.Minute, Short: 10 * time.Minute, Long: 20 * time.Minute, Every: 2}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestParseConfigInvalid(t *testing.T) {
	for _, args := range [][]string{
		{"-focus", "0s"},
		{"-short", "-5m"},
		{"-long", "0s"},
		{"-every", "0"},
	} {
		if _, err := parseConfig(args); err == nil {
			t.Errorf("parseConfig(%v) no devolvió error", args)
		}
	}
}

func TestParseConfigPositionalArg(t *testing.T) {
	for _, args := range [][]string{{"foo"}, {"-focus", "10m", "extra"}} {
		_, err := parseConfig(args)
		if err == nil || !strings.Contains(err.Error(), "argumento inesperado") {
			t.Errorf("parseConfig(%v) = %v", args, err)
		}
	}
}
