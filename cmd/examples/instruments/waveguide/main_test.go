package main

import (
	"testing"

	"github.com/almerlucke/psalm/cmd/examples/instruments/audition"
)

// the page of the waveguide: the phrases sound, presets of every model start and play, and presets
// chosen on a page that started on a plucked string are the presets: a bow or a pipe goes on sounding
func TestWaveguidePage(t *testing.T) {
	audition.Check(t, spec(), audition.Checks{
		Start:     []string{"violin", "tom", "flute", "harp"},
		Chosen:    []string{"violin", "altoFlute", "cello", "tom", "guitar"},
		Sustained: []string{"violin", "altoFlute", "cello"},
	})
}
