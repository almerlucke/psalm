package main

import (
	"testing"

	"github.com/almerlucke/psalm/cmd/examples/instruments/audition"
)

// the page of the modal synth: the phrases sound, presets of every family start and play, and presets
// chosen on a page that started on the vibraphone are the presets
func TestModalPage(t *testing.T) {
	audition.Check(t, spec(), audition.Checks{
		Start:  []string{"marimba", "tubularBells", "gong", "singingBowl", "bowedVibraphone"},
		Chosen: []string{"marimba", "gong", "bowedVibraphone", "churchBell", "cymbal", "vibraphone"},
	})
}
