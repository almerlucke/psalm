package main

import (
	"fmt"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/tools/harmony"
)

type (
	S = arrange.Section
	N = score.Note
)

// writeParts writes what every instrument plays, section by section, and returns how many notes
// clash with the chords: every part is checked against them (notes a semitone above a chord tone
// that aren't passing notes, and notes held into a chord they clash with)
func writeParts(p *psalm.Psalm, o *orchestra) int {
	timeline := songChords()
	clashes := 0
	// how the orchestra plays: ensembles never start a chord exactly together, soloists breathe,
	// nothing sits exactly on the grid. The feel is drawn again on every loop
	feels := map[string]score.Feel{
		"violins":   {Timing: 8, Spread: 12, Velocity: 0.06},
		"celli":     {Timing: 8, Spread: 10, Velocity: 0.06},
		"pizzicato": {Timing: 6, Velocity: 0.1, Accents: []float64{1, 0.8, 0.9, 0.8}},
		"harp":      {Timing: 5, Velocity: 0.1},
		"flute":     {Timing: 10, Push: 6, Velocity: 0.08}, // a little behind the beat
		"horns":     {Timing: 10, Spread: 15, Velocity: 0.06},
		"horn solo": {Timing: 12, Push: 4, Velocity: 0.08},
		"soprano":   {Timing: 10, Push: 6, Velocity: 0.08},
		"choir":     {Timing: 10, Spread: 20},
		"ooh":       {Timing: 10, Spread: 20},
		"celesta":   {Timing: 6, Velocity: 0.12},
		"bell pad":  {Strum: 25, Velocity: 0.05}, // rolled chords
		"voices":    {Spread: 10},
		"piano":     {Timing: 6, Strum: 18, Velocity: 0.06}, // the hands roll the chords a little
	}

	part := func(synth psalm.Modular, name string, amp float64, legato float64, sc arrange.Score) {
		form.Part(p, synth, name, score.Config{Amp: amp, Legato: legato, Velocity: 0.8, Feel: feels[name]}, sc)
		if name == "dawn drone" {
			return // a pedal: it holds through the chords on purpose
		}
		issues := harmony.Check(form.Compile(name, sc), timeline)
		clashes += len(issues)
		fmt.Print(harmony.Report(name, issues))
	}

	part(o.violins, "violins", 0.126, 1.0, func(s S) []N {
		if !arrange.In(s, "Prelude", "Theme", "Climax", "Coda") {
			return nil
		}
		return arrange.PerBar(s, func(bar int) []N {
			c := chordAt(s, bar)
			if s.Name == "Coda" && bar == s.Bars-1 {
				return []N{{Pitches: c.violins, Beats: 4, Hold: 6}} // the last chord rings
			}
			return []N{score.Chord(c.violins, 4)}
		})
	})

	part(o.celli, "celli", 0.145, 1.0, func(s S) []N {
		if s.Name == "Interlude" {
			return arrange.PerBar(s, func(bar int) []N {
				return []N{{Pitches: chordAt(s, bar).celli[:1], Beats: 4, Velocity: 0.5}}
			})
		}
		return arrange.PerBar(s, func(bar int) []N { return []N{score.Chord(chordAt(s, bar).celli, 4)} })
	})

	part(o.pizz, "pizzicato", 0.355, 0.5, func(s S) []N {
		if !arrange.In(s, "Theme", "Climax") {
			return nil
		}
		return arrange.PerBar(s, func(bar int) []N {
			r := chordAt(s, bar).root
			return score.Join(score.Notes(2, r), score.Notes(1, r+7, r+12))
		})
	})

	part(o.harp, "harp", 0.13, 1.6, func(s S) []N {
		if !arrange.In(s, "Prelude", "Theme", "Interlude", "Climax", "Coda") {
			return nil
		}
		return arrange.PerBar(s, func(bar int) []N {
			if (s.Name == "Prelude" && bar < 4) || (s.Name == "Coda" && bar == s.Bars-1) {
				return nil // the harp enters halfway the prelude and rests on the last chord
			}
			return harpArp(chordAt(s, bar))
		})
	})

	part(o.flute, "flute", 0.525, 0.95, arrange.Only("Theme", func(s S) []N {
		return theme()
	}))

	part(o.horns, "horns", 0.0788, 0.95, func(s S) []N {
		if !arrange.In(s, "Chorale", "Climax") {
			return nil
		}
		return arrange.PerBar(s, func(bar int) []N {
			low := score.Pitches(chordAt(s, bar).choir, -12)
			if s.Name == "Chorale" {
				return score.Chords(2, low, low) // a chorale in half notes
			}
			return []N{score.Chord(low, 4)}
		})
	})

	part(o.hornSolo, "horn solo", 0.33, 0.95, arrange.Only("Climax", func(s S) []N {
		return score.Repeat(counter, 2)
	}))

	part(o.soprano, "soprano", 0.34, 0.95, arrange.Only("Climax", func(s S) []N {
		return theme()
	}))

	part(o.choir, "choir", 0.0793, 1.0, func(s S) []N {
		if !arrange.In(s, "Prelude", "Chorale", "Climax", "Coda") {
			return nil
		}
		return arrange.PerBar(s, func(bar int) []N {
			c := chordAt(s, bar)
			if s.Name == "Prelude" {
				if bar%2 == 1 {
					return nil
				}
				return []N{{Pitches: c.choir, Beats: 4, Hold: 3.8}} // every other chord, within its bar
			}
			return []N{score.Chord(c.choir, 4)}
		})
	})

	part(o.ooh, "ooh", 0.061, 1.0, func(s S) []N {
		if !arrange.In(s, "Chorale", "Climax") {
			return nil
		}
		return arrange.PerBar(s, func(bar int) []N { return []N{score.Chord(score.Pitches(chordAt(s, bar).choir, 12), 4)} })
	})

	part(o.celesta, "celesta", 0.217, 0.9, func(s S) []N {
		switch s.Name {
		case "Interlude":
			return arrange.PerBar(s, func(bar int) []N {
				v := score.Pitches(chordAt(s, bar).violins, 12)
				if bar%2 == 0 {
					return score.Join(score.Notes(1, v[3]), score.Notes(0.5, v[2], v[1]), score.Notes(1, v[0], 0))
				}
				return score.Join(score.Notes(1, 0), score.Notes(0.5, v[1], v[2]), score.Notes(2, v[3]))
			})
		case "Coda":
			// the theme returns as a memory (the bars that fit Dm Bb F C), then the last chord
			memory := score.Join(themeBar(0), themeBar(1), themeBar(4), themeBar(5))
			memory = score.Shape(score.Transpose(memory, 12), score.Velocities(0.6))
			return score.Join(memory, score.MustParse("r:4 D5:1 C#5 A4:2 F5 D5 r:4")) // D leans on the A chord, resolving to C#
		}
		return nil
	})

	part(o.bellPad, "bell pad", 0.0471, 1.0, arrange.Bars("Interlude", func(s S, bar int) []N {
		return []N{score.Chord(chordAt(s, bar).choir, 4)}
	}))

	part(o.carrier, "voices", 0.0976, 1.0, arrange.Bars("Interlude", func(s S, bar int) []N {
		c := chordAt(s, bar)
		return []N{score.Chord(append([]float64{c.celli[0]}, c.choir...), 4)}
	}))

	form.Part(p, o.droplets, "droplets", score.Config{Amp: 0.133, Legato: 1, Key: "frequency"}, arrange.Only("Interlude", func(s S) []N {
		return score.Notes(16, 7, 9) // drops per second
	}))

	part(o.dawnDrone, "dawn drone", 0.111, 1.0, func(s S) []N {
		if !arrange.In(s, "Prelude", "Coda") {
			return nil
		}
		return []N{{Pitches: []float64{38, 45}, Beats: 32, Hold: 26}}
	})

	// the piano: rolled chords at dawn before the harp enters, weight under the horn chorale, and
	// the last cadence under the celesta, its final chord ringing. The left hand plays the root in
	// octaves, the right hand the middle voices
	part(o.grand, "piano", 0.245, 1.0, func(s S) []N {
		voicing := func(c chord) []float64 { return append([]float64{c.root, c.root + 12}, c.choir...) }
		// the soft chords of the prelude and the coda sound with the quiet orchestra there
		const prelude, coda = 4.7, 3.5
		switch s.Name {
		case "Prelude":
			return arrange.PerBar(s, func(bar int) []N {
				if bar >= 4 {
					return nil
				}
				return []N{{Pitches: voicing(chordAt(s, bar)), Beats: 4, Velocity: 0.4, Amp: prelude}}
			})
		case "Chorale":
			return arrange.PerBar(s, func(bar int) []N {
				v := voicing(chordAt(s, bar))
				return []N{{Pitches: v, Beats: 2, Velocity: 0.6}, {Pitches: v, Beats: 2, Velocity: 0.5}}
			})
		case "Coda":
			return arrange.PerBar(s, func(bar int) []N {
				n := N{Pitches: voicing(chordAt(s, bar)), Beats: 4, Velocity: 0.45, Amp: coda}
				if bar == s.Bars-1 {
					n.Hold, n.Velocity = 6, 0.35 // the last chord rings
				}
				return []N{n}
			})
		}
		return nil
	})

	return clashes
}
