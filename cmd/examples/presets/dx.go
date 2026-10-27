package main

import (
	"slices"
	"strings"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/synths/dx"
)

// dxTour has every preset of the dx synth, each playing what its name suggests (the styles of the subtractive ones), with the operators played 1, 2 or 4 times a
// sample (oversample 0, 1, 2; below 0 the preset's own)
func dxTour(oversample int) []entry {
	names := make([]string, 0, len(dx.Presets))
	for n := range dx.Presets {
		names = append(names, n)
	}

	slices.Sort(names)

	var tour []entry

	for _, name := range names {
		patch := dx.Presets[name]
		st, octave := subtractiveStyle(strings.ToLower(name))
		tour = append(tour, entry{"dx", name, st, octave, func(p *psalm.Psalm) psalm.Modular {
			pt := patch()
			if oversample >= 0 {
				pt.Oversample = oversample
			}

			return dx.NewPoly(pt, 8, psalm.Stereo).AddTo(p)
		}})
	}

	return tour
}
