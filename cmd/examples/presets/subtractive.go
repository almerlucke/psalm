package main

import (
	"slices"
	"strings"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/synths/subtractive"
)

// subtractiveTour has every preset of the subtractive synth, each playing what its name suggests: a bass line, a melody, a chord, an arpeggio or single notes
func subtractiveTour() []entry {
	names := make([]string, 0, len(subtractive.Presets))
	for n := range subtractive.Presets {
		names = append(names, n)
	}

	slices.Sort(names)

	var tour []entry

	for _, name := range names {
		cfg := subtractive.Presets[name]
		st, octave := subtractiveStyle(strings.ToLower(name))
		tour = append(tour, entry{"subtractive", name, st, octave, func(p *psalm.Psalm) psalm.Modular {
			return subtractive.NewPoly(cfg(), subtractive.Voices(name, 8)).AddTo(p)
		}})
	}

	return tour
}

// subtractiveStyle is what suits a preset by its name
func subtractiveStyle(name string) (style, int) {
	has := func(parts ...string) bool {
		for _, s := range parts {
			if strings.Contains(name, s) {
				return true
			}
		}

		return false
	}

	switch {
	case has("kick", "snare", "hat", "tom", "zap"):
		return hits, -1
	case has("808", "sub"):
		return held, 0
	case has("wobble", "reese"):
		return held, 0
	case has("bass"):
		return bass, 0
	case has("riser"):
		return drone, -1
	case has("bell", "vibes", "glasskeys", "chime"):
		return hits, 0
	case has("pluck", "arp", "clav", "harpsi", "wurli", "epiano", "keys"):
		return arp, 0
	case has("pad", "string", "choir", "organ", "brass", "huge", "stab"):
		return pad, 0
	case has("wind", "flute", "clarinet", "theremin", "talking", "vowel", "sine"):
		return slow, 0
	default:
		return lead, 0
	}
}
