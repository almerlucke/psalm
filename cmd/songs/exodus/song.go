package main

import (
	"fmt"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/tools/harmony"
)

// the drums are tuned to D: a taiko on D2 and A2, the frame drums an octave and a bit above
const (
	dLow, aLow      = 38.0, 45.0
	dFrame, aFrame  = 50.0, 57.0
	tomLow, tomHigh = 41.0, 45.0
	jingle          = 70.0
	bellD, bellA    = 62.0, 69.0
	anvilPitch      = 50.0
)

// level is the overall gain of the parts: the limiter on the master has little to do
const level = 0.45

// pat plays a pattern of a bar: a pitch for a stroke, 0 for a rest, every one for a step
func pat(step float64, pitches ...float64) []N { return score.Notes(step, pitches...) }

func rep(ns []N, n int) []N { return score.Repeat(ns, n) }

// writeParts writes every part and returns how many notes clash with the chords
func writeParts(p *psalm.Psalm, o *orchestra) int {
	tl := timeline()
	clashes := 0
	part := func(synth psalm.Modular, name string, amp, legato float64, m plays) {
		sc := m.score()
		form.Part(p, synth, name, score.Config{Amp: amp * level, Legato: legato, Velocity: 0.8, Feel: feels[name]}, sc)
		issues := harmony.Check(form.Compile(name, sc), tl)
		clashes += len(issues)
		fmt.Print(harmony.Report(name, issues))
	}
	// drums and bells are unpitched or ring through the chords: not checked against them
	loose := func(synth psalm.Modular, name string, amp, legato float64, m plays) {
		form.Part(p, synth, name, score.Config{Amp: amp * level, Legato: legato, Velocity: 0.8, Feel: feels[name]}, m.score())
	}
	root := func(br bar) float64 { return br.root }

	// ---- the pedal, the clarinets and the flute ----

	// the organ pedal: a drone under Egypt, the wilderness and the years, the roots of the chords elsewhere
	part(o.organBass, "organ (pedal)", 0.3, 1.0, plays{
		egypt:      every(func(b int, br bar) []N { return hold([]float64{root(br)}) }),
		night:      every(from(2, func(b int, br bar) []N { return hold([]float64{root(br)}) })),
		exodus:     every(from(4, func(b int, br bar) []N { return []N{score.N(root(br), 2), score.N(root(br)+7, 2)} })),
		pursuit:    every(func(b int, br bar) []N { return rep([]N{score.N(root(br), 1)}, 4) }),
		sea:        every(within(4, 8, func(b int, br bar) []N { return hold([]float64{root(br)}) })),
		miriam:     every(from(4, func(b int, br bar) []N { return []N{score.N(root(br), 2), score.N(root(br)+7, 2)} })),
		wilderness: every(func(b int, br bar) []N { return hold([]float64{root(br)}) }),
		sinai:      every(func(b int, br bar) []N { return hold([]float64{root(br)}) }),
		years:      every(func(b int, br bar) []N { return hold([]float64{root(br)}) }),
		land:       every(func(b int, br bar) []N { return hold([]float64{root(br)}) }),
	})

	part(o.low, "low clarinet", 0.28, 0.95, plays{
		night:      every(within(4, 8, func(b int, br bar) []N { return hold([]float64{br.mid[0]}) })),
		sea:        every(from(8, func(b int, br bar) []N { return hold([]float64{br.mid[0]}) })),
		wilderness: every(from(4, func(b int, br bar) []N { return hold([]float64{br.mid[0]}) })),
		years:      every(func(b int, br bar) []N { return hold([]float64{br.mid[0]}) }),
		land:       every(func(b int, br bar) []N { return hold([]float64{br.mid[0]}) }),
	})

	// the pulse of the march and the dance, on the marimba
	part(o.marimba, "marimba", 0.4, 0.9, plays{
		exodus: every(from(4, func(b int, br bar) []N { return []N{score.N(root(br)+12, 2), score.N(root(br)+19, 2)} })),
		miriam: every(func(b int, br bar) []N { return []N{score.N(root(br)+12, 2), score.N(root(br)+12, 2)} }),
	})

	part(o.solo, "alto flute", 0.3, 0.97, plays{
		egypt:   tune(lament),
		night:   every(within(4, 8, func(b int, br bar) []N { return hold([]float64{br.high[len(br.high)-1]}) })),
		pursuit: every(func(b int, br bar) []N { return arp8(br.high) }),
		years:   tune(memory),
		land:    tuneBars(promised, 0, 4, 16),
	})

	part(o.section, "clarinets", 0.26, 0.97, plays{
		exodus: tune(score.Transpose(march, -12)), // the clarinet's warm low register
		land:   tuneBars(score.Transpose(promised, -12), 4, 16, 16),
	})
	// glass sustains under the tune, rising
	part(o.glass, "glass harmonica", 0.3, 1.0, plays{
		night:   every(within(4, 8, func(b int, br bar) []N { return held(br.high, 0.5+0.1*float64(b-4)) })),
		pursuit: every(func(b int, br bar) []N { return held(br.high, 0.5+0.07*float64(b)) }),
		sea:     every(within(4, 8, func(b int, br bar) []N { return held(br.high, 0.4+0.1*float64(b-4)) })),
		sinai:   every(within(4, 8, func(b int, br bar) []N { return hold(br.high) })),
	})

	// long ringing tines, struck again and again in a roll: the wind rising over the water, the walls of
	// water standing up, the stillness of the crossing, the sea closing over
	part(o.tine, "space tine", 0.4, 1.0, plays{
		sea: every(func(b int, br bar) []N {
			switch {
			case b < 4:
				return roll([]float64{br.high[b%len(br.high)]}, 14+2*b, 4, 0.2, 0.7)
			case b < 8:
				return roll(br.high, 18+2*(b-4), 5, 0.4, 1.0)
			case b == 11:
				return roll(br.high, 12, 3, 0.2, 0.6)
			case b%2 == 0:
				return []N{{Pitches: []float64{br.high[0]}, Beats: 4, Hold: 0.3, Velocity: 0.55}}
			}
			return nil
		}),
		waters: every(func(b int, br bar) []N {
			if b < 3 {
				return roll(append([]float64{root(br) + 12}, br.high...), 24, 3, 0.6+0.1*float64(b), 1.0)
			}
			return []N{{Pitches: []float64{root(br) + 12}, Beats: 4, Hold: 0.3, Velocity: 1.0}}
		}),
	})

	part(o.oud, "oud", 0.7, 1.0, plays{
		egypt: every(from(4, func(b int, br bar) []N {
			return []N{score.N(root(br)+12, 1), score.Rest(1), score.N(br.mid[1], 1), score.Rest(1)}
		})),
		wilderness: every(func(b int, br bar) []N { return arp8(br.mid) }),
		years:      every(within(0, 8, func(b int, br bar) []N { return arp8(br.mid) })),
	})

	part(o.harp, "harp", 0.3, 1.0, plays{
		exodus: every(from(4, func(b int, br bar) []N { return arp8(br.high) })),
		sea: every(func(b int, br bar) []N {
			switch {
			case b < 4: // the wind rises over the water
				return sweep(br.chord, 36, 96, 16, 4)
			case b < 8: // the walls of water stand up, a glissando down
				return sweep(br.chord, 96, 48, 16, 4)
			}
			return arp8(br.high)
		}),
		miriam: every(from(2, func(b int, br bar) []N { return arp8(br.high) })),
		years:  every(from(4, func(b int, br bar) []N { return arp8(br.high) })),
		land:   every(from(4, func(b int, br bar) []N { return arp8(br.high) })),
	})

	// ---- the winds ----

	part(o.bansuri, "bansuri", 0.14, 0.97, plays{
		sea:        tuneBars(score.Join(wonder, crossing), 0, 12, 12),
		wilderness: tune(wanderer),
		land:       tuneBars(score.Transpose(promised, -12), 4, 16, 16),
	})
	part(o.recorder, "recorder", 0.12, 0.9, plays{
		exodus: tuneBars(march, 8, 16, 16),
		miriam: tune(song),
	})
	part(o.clarinet, "bass clarinet", 0.14, 1.0, plays{
		night:      every(within(0, 8, func(b int, br bar) []N { return hold([]float64{br.mid[1]}) })),
		wilderness: every(from(8, func(b int, br bar) []N { return hold([]float64{br.mid[1]}) })),
	})
	part(o.shakuhachi, "shakuhachi", 0.14, 0.97, plays{
		years: tune(memory),
	})

	// ---- the drums ----

	loose(o.frame, "frame drum", 0.4, 0.9, plays{
		// the work of the slaves: three strokes to the bar, the fourth beat left out
		egypt: every(func(b int, br bar) []N { return pat(1, aFrame, 0, aFrame, aFrame) }),
		// the heart: two beats close together, twice a bar, louder as midnight comes
		night: every(func(b int, br bar) []N {
			a := 0.5 + 0.5*float64(b)/8
			beat := func(amp float64) N { return N{Pitches: []float64{dFrame}, Beats: 0.5, Amp: amp} }
			return []N{beat(a), beat(a * 0.7), score.Rest(1), beat(a), beat(a * 0.7), score.Rest(1)}
		}),
		sea:        every(from(8, func(b int, br bar) []N { return pat(2, dFrame, 0) })),
		wilderness: every(from(2, func(b int, br bar) []N { return pat(4, dFrame) })),
		years:      every(func(b int, br bar) []N { return pat(2, dFrame, 0) }),
		miriam:     every(func(b int, br bar) []N { return pat(1, dFrame, 0, dFrame, 0) }),
	})
	loose(o.taiko, "taiko", 0.22, 0.9, plays{
		exodus:  every(from(2, func(b int, br bar) []N { return pat(1, dLow, 0, aLow, 0) })),
		pursuit: every(func(b int, br bar) []N { return rep(score.Join(pat(0.5, dLow), pat(0.25, dLow, dLow)), 4) }),
		sea:     every(within(4, 8, func(b int, br bar) []N { return pat(4, dLow) })),
		waters:  every(within(0, 3, func(b int, br bar) []N { return rep(pat(0.5, dLow, aLow), 4) })),
		miriam:  every(from(4, func(b int, br bar) []N { return pat(2, dLow, 0) })),
		sinai:   every(func(b int, br bar) []N { return pat(2, dLow, 0) }),
		land:    every(within(12, 16, func(b int, br bar) []N { return pat(2, dLow, 0) })),
	})
	loose(o.bodhran, "bodhran", 0.35, 0.9, plays{
		exodus: every(func(b int, br bar) []N {
			return score.Shape(score.Join(pat(1, dFrame), pat(0.5, dFrame, dFrame), pat(1, dFrame), pat(0.5, dFrame, dFrame)), score.Velocities(1, 0.5, 0.6, 0.8, 0.5, 0.6))
		}),
		pursuit: every(func(b int, br bar) []N { return rep(pat(0.5, dFrame, aFrame), 4) }),
		miriam: every(func(b int, br bar) []N {
			return score.Shape(score.Join(pat(0.5, dFrame, 0, aFrame, 0), pat(0.5, dFrame, dFrame, aFrame, 0)), score.Velocities(1, 0.6, 0.8, 0.6))
		}),
	})
	loose(o.tom, "floor tom", 0.22, 0.9, plays{
		night:   every(within(6, 8, func(b int, br bar) []N { return rep(pat(0.5, tomLow, tomHigh), 4) })),
		pursuit: every(from(4, func(b int, br bar) []N { return rep(pat(0.25, tomLow), 16) })),
		sea:     every(within(4, 8, func(b int, br bar) []N { return rep(pat(0.25, tomLow, tomHigh), 8) })),
		waters:  every(within(0, 3, func(b int, br bar) []N { return rep(pat(0.25, tomHigh, tomLow), 8) })),
		sinai:   every(within(3, 4, func(b int, br bar) []N { return rep(pat(0.25, tomLow), 16) })),
	})
	loose(o.tambourine, "tambourine", 0.7, 0.5, plays{
		exodus: every(from(4, func(b int, br bar) []N {
			return score.Shape(pat(0.5, jingle, jingle, jingle, jingle, jingle, jingle, jingle, jingle), score.Velocities(0.9, 0.4, 0.6, 0.4))
		})),
		miriam: every(func(b int, br bar) []N {
			return score.Shape(pat(0.5, jingle, jingle, jingle, jingle, jingle, jingle, jingle, jingle), score.Velocities(1, 0.5, 0.7, 0.5))
		}),
		land: every(within(8, 13, func(b int, br bar) []N {
			return score.Shape(pat(0.5, jingle, jingle, jingle, jingle, jingle, jingle, jingle, jingle), score.Velocities(0.8, 0.4, 0.6, 0.4))
		})),
	})

	// ---- the bells and bars ----

	// anvil: the bricks of Egypt, the hammer on the second and fourth beats
	loose(o.anvil, "anvil", 0.6, 0.9, plays{
		egypt: every(from(2, func(b int, br bar) []N { return pat(1, 0, anvilPitch, 0, anvilPitch) })),
	})
	// midnight: twelve strokes of a bell, two to a bar
	loose(o.tubular, "tubular bells", 0.3, 1.0, plays{
		night: every(within(0, 6, func(b int, br bar) []N { return strokes(bellD, 0, 2) })),
		sinai: every(within(0, 5, func(b int, br bar) []N { // the ten words
			return strokes(bellA, 0, 2)
		})),
		land: every(from(4, func(b int, br bar) []N {
			if b == 15 {
				return []N{{Pitches: []float64{br.root + 24}, Beats: 4, Hold: 12}}
			}
			return strokes(br.root+24, 0)
		})),
	})
	loose(o.bell, "temple bell", 0.35, 1.0, plays{
		night:  every(within(4, 5, func(b int, br bar) []N { return strokes(bellD-12, 0) })),
		waters: every(within(0, 1, func(b int, br bar) []N { return strokes(bellD-12, 0) })),
		land:   every(within(4, 5, func(b int, br bar) []N { return strokes(bellD-12, 0) })),
	})
	loose(o.hand, "handbells", 0.6, 1.0, plays{
		miriam: every(from(8, func(b int, br bar) []N { return pat(1, br.high[len(br.high)-1]+12, 0, 0, 0) })),
		land:   every(from(8, func(b int, br bar) []N { return arp8(br.high) })),
		// the angel bells over the crossing
		sea: every(from(8, func(b int, br bar) []N { return pat(1, br.high[len(br.high)-1], 0, br.high[1], br.high[0]) })),
	})
	part(o.vibe, "vibraphone", 0.3, 1.0, plays{
		sea:    every(from(8, func(b int, br bar) []N { return arp8(br.high) })),
		miriam: every(from(4, func(b int, br bar) []N { return arp8(br.mid) })),
		land:   every(from(8, func(b int, br bar) []N { return arp8(br.mid) })),
	})
	part(o.bowedVibe, "bowed vibraphone", 0.28, 1.0, plays{
		sea:        every(within(0, 8, func(b int, br bar) []N { return hold(br.mid) })),
		wilderness: every(from(4, func(b int, br bar) []N { return hold(br.mid[1:]) })),
		years:      every(from(4, func(b int, br bar) []N { return hold(br.mid[1:]) })),
	})
	loose(o.bowl, "singing bowl", 0.5, 1.0, plays{
		sea: every(within(0, 1, func(b int, br bar) []N { return strokes(62, 0) })),
		wilderness: every(func(b int, br bar) []N {
			if b%4 == 0 {
				return strokes(62, 0)
			}
			return nil
		}),
		years: every(func(b int, br bar) []N {
			if b%4 == 0 {
				return strokes(62, 0)
			}
			return nil
		}),
	})
	loose(o.thunder, "thunder sheet", 0.7, 1.0, plays{
		night: every(within(7, 8, func(b int, br bar) []N { return strokes(40, 0) })),
		sea:   every(within(4, 5, func(b int, br bar) []N { return strokes(40, 0) })),
		sinai: every(func(b int, br bar) []N {
			if b == 0 || b == 3 || b == 6 {
				return strokes(40, 0)
			}
			return nil
		}),
	})
	loose(o.tamtam, "tam-tam", 0.6, 1.0, plays{
		night:  every(within(4, 5, func(b int, br bar) []N { return strokes(38, 0) })),
		waters: every(within(0, 1, func(b int, br bar) []N { return strokes(38, 0) })),
		sinai:  every(within(0, 1, func(b int, br bar) []N { return strokes(38, 0) })),
	})
	loose(o.gong, "gong", 0.4, 1.0, plays{
		pursuit: every(within(7, 8, func(b int, br bar) []N { return strokes(38, 3) })),
		waters:  every(within(0, 1, func(b int, br bar) []N { return strokes(38, 0) })),
		land:    every(within(15, 16, func(b int, br bar) []N { return []N{{Pitches: []float64{38}, Beats: 4, Hold: 12}} })),
	})
	part(o.kalimba, "kalimba", 0.5, 1.0, plays{
		sea: every(from(8, func(b int, br bar) []N { return pat(1, br.high[2], 0, br.high[1], 0) })),
		wilderness: every(from(4, func(b int, br bar) []N {
			return []N{score.Rest(0.5), score.N(br.high[b%3], 0.5), score.Rest(1), score.N(br.high[(b+1)%3], 0.5), score.Rest(0.5), score.N(br.high[(b+2)%3], 1)}
		})),
		years: every(from(2, func(b int, br bar) []N { return pat(1, 0, br.high[b%3], 0, br.high[(b+1)%3]) })),
	})
	part(o.glock, "glockenspiel", 0.45, 1.0, plays{
		sea:   every(from(8, func(b int, br bar) []N { return rep(pat(0.5, br.high[0]+24, 0, br.high[1]+24, 0), 2) })),
		years: every(from(6, func(b int, br bar) []N { return pat(1, br.high[0]+12, 0, 0, 0) })),
		land:  every(from(8, func(b int, br bar) []N { return rep(pat(0.5, br.high[0]+24, 0, br.high[1]+24, 0), 2) })),
	})

	// ---- the keys ----

	part(o.grand, "piano", 0.32, 1.0, plays{
		egypt: every(func(b int, br bar) []N {
			if b%2 == 0 {
				return held([]float64{root(br), root(br) + 12}, 0.9)
			}
			return nil
		}),
		night: every(func(b int, br bar) []N {
			return score.Shape(rep(pat(0.5, root(br), root(br)+7), 4), score.Velocities(0.7, 0.45, 0.5, 0.45), score.Hold(0.55))
		}),
		exodus: every(from(8, func(b int, br bar) []N { return []N{score.Chord(br.mid, 2), score.Chord(br.mid, 2)} })),
		pursuit: every(func(b int, br bar) []N {
			return rep(pat(0.5, root(br)+12, root(br)), 4)
		}),
		sea: every(func(b int, br bar) []N {
			if b < 4 {
				return sweep(br.chord, 40, 100, 16, 4)
			}
			if b < 8 {
				return held(append([]float64{root(br)}, br.mid...), 1)
			}
			return rep(pat(0.5, br.mid[0], br.mid[1], br.mid[2], br.mid[1]), 2)
		}),
		waters: every(within(0, 3, func(b int, br bar) []N { return rep(pat(0.5, root(br)+24, root(br)+12, root(br), root(br)-12), 2) })),
		miriam: every(func(b int, br bar) []N {
			return []N{score.Rest(1), score.Chord(br.mid, 1), score.Rest(1), score.Chord(br.mid, 1)}
		}),
		sinai: every(func(b int, br bar) []N { return held([]float64{root(br), root(br) + 12, br.mid[1]}, 1) }),
		years: every(from(4, func(b int, br bar) []N { return held([]float64{root(br) + 12, br.mid[1]}, 0.7) })),
		land: every(func(b int, br bar) []N {
			if b < 4 {
				return held([]float64{root(br), root(br) + 12}, 0.9)
			}
			return rep(pat(0.5, br.mid[0], br.mid[1], br.mid[2], br.mid[1]), 2)
		}),
	})

	organ := func(synth psalm.Modular, name string, amp float64, m plays) {
		part(synth, name, amp, 1.0, m)
	}
	organ(o.organLow, "organ (cathedral)", 0.24, plays{
		egypt:   every(func(b int, br bar) []N { return hold(br.mid) }),
		night:   every(func(b int, br bar) []N { return hold(br.mid) }),
		pursuit: every(func(b int, br bar) []N { return hold(append([]float64{root(br)}, br.mid...)) }),
		sea:     every(within(4, 12, func(b int, br bar) []N { return hold(append([]float64{root(br)}, br.mid...)) })),
		waters:  every(within(0, 3, func(b int, br bar) []N { return hold(append([]float64{root(br)}, br.mid...)) })),
		sinai:   every(func(b int, br bar) []N { return hold(append([]float64{root(br)}, br.mid...)) }),
	})
	organ(o.organSoft, "organ (soft)", 0.22, plays{
		wilderness: every(func(b int, br bar) []N { return hold(br.mid) }),
		years:      every(func(b int, br bar) []N { return hold(br.mid) }),
		sea:        every(within(0, 4, func(b int, br bar) []N { return hold(br.mid) })),
		land:       every(within(0, 4, func(b int, br bar) []N { return hold(br.mid) })),
	})
	organ(o.organRaw, "organ (strings)", 0.2, plays{
		exodus: every(from(8, func(b int, br bar) []N { return hold(br.mid) })),
		miriam: every(from(4, func(b int, br bar) []N { return hold(br.mid) })),
		sea:    every(from(8, func(b int, br bar) []N { return hold(br.mid) })),
	})
	organ(o.organPlenum, "organ (plenum)", 0.22, plays{
		sinai: every(func(b int, br bar) []N { return hold(append([]float64{root(br)}, br.mid...)) }),
		land:  every(from(4, func(b int, br bar) []N { return hold(append([]float64{root(br)}, br.mid...)) })),
	})
	return clashes
}

var _ = arrange.Silent
