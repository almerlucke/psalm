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

// how every part plays: ensembles never start a chord exactly together, soloists breathe
var feels = map[string]score.Feel{
	"violins":   {Timing: 8, Spread: 12, Velocity: 0.06},
	"celli":     {Timing: 8, Spread: 10, Velocity: 0.06},
	"pizzicato": {Timing: 6, Velocity: 0.1, Accents: []float64{1, 0.8, 0.85}, BeatsPerBar: 3},
	"harp":      {Timing: 5, Velocity: 0.1},
	"timpani":   {Timing: 4, Velocity: 0.08},
	"flute":     {Timing: 10, Push: 6, Velocity: 0.08},
	"oboe":      {Timing: 10, Push: 6, Velocity: 0.08},
	"clarinet":  {Timing: 10, Push: 6, Velocity: 0.08},
	"horns":     {Timing: 10, Spread: 15, Velocity: 0.06},
	"horn":      {Timing: 12, Push: 4, Velocity: 0.08},
	"soprano":   {Timing: 10, Push: 6, Velocity: 0.08},
	"spirit":    {Timing: 10, Spread: 20},
	"choir":     {Timing: 10, Spread: 20},
	"organ":     {Timing: 4, Spread: 6},
	"celesta":   {Timing: 6, Velocity: 0.12},
	"bell pad":  {Strum: 25, Velocity: 0.05},
	"bells":     {Strum: 40},
	"piano":     {Timing: 6, Strum: 18, Velocity: 0.06},
	"the deep":  {Timing: 20},
}

// dark keeps the evening of the first day as quiet as the darkness: the day sounds out (its
// dynamics) when the light comes
const dark = 0.9 / 1.3

// perBar writes a section bar by bar with the chord and voicings of every bar; a bar without notes
// (nil) is a rest
func perBar(s S, f func(b int, br bar) []N) []N {
	return arrange.PerBar(s, func(b int) []N { return f(b, at(s, b)) })
}

// ringing plays a chord in every bar of the seventh day, the last one ringing to the end
func ringing(b int, chord []float64) []N {
	switch {
	case b < final:
		return []N{score.Chord(chord, 3)}
	case b == final:
		return []N{{Pitches: chord, Beats: 12, Hold: 12}}
	}
	return nil // the last chord still rings
}

// writeParts writes every part in a tuning and returns how many notes clash with the chords
func writeParts(p *psalm.Psalm, o *orchestra, tune score.Tuning) int {
	tl := timeline()
	clashes := 0
	// part plays a score; it is checked against the chords first (the void holds its fifth
	// through the chords on purpose)
	part := func(synth psalm.Modular, name string, amp float64, legato float64, sc arrange.Score) {
		form.Part(p, synth, name, score.Config{Amp: amp, Legato: legato, Velocity: 0.8, Tuning: tune, Feel: feels[name]}, sc)
		if name == "void" {
			return
		}
		issues := harmony.Check(form.Compile(name, sc), tl)
		clashes += len(issues)
		fmt.Print(harmony.Report(name, issues))
	}

	// the void: "the earth was without form, and void; and darkness was upon the face of the deep",
	// an open fifth; it returns under the last chord of the seventh day
	part(o.void, "void", 0.111, 1.0, func(s S) []N {
		switch s.Name {
		case day1:
			return score.Join([]N{{Pitches: []float64{39, 46}, Beats: 18, Hold: 19, Amp: dark}}, rest(6))
		case day7:
			return score.Join(rest(final), []N{{Pitches: []float64{39, 46, 51}, Beats: 12, Hold: 12}})
		}
		return nil
	})

	// "and the Spirit of God moved upon the face of the waters": a breathing choir in the darkness,
	// the heaven between the waters on the second morning, the morning stars on the fourth
	part(o.spirit, "spirit", 0.061, 1.0, func(s S) []N {
		switch s.Name {
		case day1:
			return perBar(s, func(b int, br bar) []N {
				if morning(b) {
					return []N{score.Chord(br.mid, 3)}
				}
				return []N{{Pitches: []float64{51, 58}, Beats: 3, Velocity: 0.6, Amp: dark}}
			})
		case day2, day4:
			return perBar(s, func(b int, br bar) []N {
				if !morning(b) {
					return nil
				}
				return []N{score.Chord(br.mid, 3)}
			})
		case day7:
			return perBar(s, func(b int, br bar) []N { return []N{score.Chord(br.mid, 3)} })
		}
		return nil
	})

	// the Word sounds through open fifths, two readings in the darkness of the first evening
	part(o.wordPad, "the word", 0.0976, 1.0, arrange.Only(day1, func(s S) []N {
		return score.Join(rest(2), []N{{Pitches: []float64{39, 46, 51, 58}, Beats: 12, Amp: dark}}, rest(6))
	}))

	part(o.violins, "violins", 0.126, 1.0, func(s S) []N {
		return perBar(s, func(b int, br bar) []N {
			switch s.Name {
			case day1, day4, day6:
				if morning(b) {
					return []N{score.Chord(br.high, 3)}
				}
			case day2:
				return []N{score.Chord(score.Pitches(br.high, 12), 3)} // the waters above, very high
			case day7:
				return ringing(b, br.high)
			}
			return nil
		})
	})

	part(o.celli, "celli", 0.145, 1.0, func(s S) []N {
		return perBar(s, func(b int, br bar) []N {
			switch s.Name {
			case day2:
				return []N{score.Chord([]float64{br.root, br.root + 7}, 3)} // the waters below, very low
			case day3, day5, day6:
				return []N{score.Chord([]float64{br.root + 12, br.root + 19}, 3)}
			case day7:
				return ringing(b, []float64{br.root + 12, br.root + 19})
			}
			return nil
		})
	})

	// the land walks: the dry land on the third evening, the beasts on the sixth
	part(o.pizz, "pizzicato", 0.25, 0.5, func(s S) []N {
		return perBar(s, func(b int, br bar) []N {
			if !arrange.In(s, day3, day6) || morning(b) {
				return nil
			}
			r := br.root + 12
			return score.Notes(1, r, r+7, r+12)
		})
	})

	// the waters flow on the second morning
	part(o.harp, "harp", 0.13, 1.6, arrange.Bars(day2, func(s S, b int) []N {
		if !morning(b) {
			return nil
		}
		br := at(s, b)
		return arp(append([]float64{br.root + 12}, br.mid...))
	}))

	// let there be light; the dry land appears and the beasts tread; the Amen
	part(o.timpani, "timpani", 0.2, 1.0, func(s S) []N {
		return perBar(s, func(b int, br bar) []N {
			switch {
			case s.Name == day1 && b == evening:
				return []N{{Pitches: []float64{39}, Beats: 3, Velocity: 0.9}}
			case arrange.In(s, day3, day6) && !morning(b):
				r := br.root
				for r < 36 {
					r += 12
				}
				return []N{{Pitches: []float64{r}, Beats: 2, Velocity: 0.6}, {Pitches: []float64{r}, Beats: 1, Velocity: 0.4}}
			case s.Name == day7 && b == final:
				return []N{{Pitches: []float64{39}, Beats: 3, Velocity: 0.6}}
			}
			return nil
		})
	})

	// the plants in canon on the third morning, the moon on the fourth evening, the birds on the
	// fifth morning
	part(o.flute, "flute", 0.21, 0.95, func(s S) []N {
		switch s.Name {
		case day3:
			return score.Join(rest(6), seedA, seedB, seedC)
		case day4:
			return score.Join(moon, rest(6))
		case day5:
			return perBar(s, func(b int, br bar) []N {
				if !morning(b) {
					return nil
				}
				return birdsong(br.chord, b%2 == 1)
			})
		}
		return nil
	})

	part(o.oboe, "oboe", 0.2, 0.95, arrange.Only(day3, func(s S) []N {
		return score.Join(rest(8), seedA, seedB)
	}))

	part(o.clarinet, "clarinet", 0.25, 0.95, arrange.Only(day3, func(s S) []N {
		return score.Join(rest(10), score.Transpose(seedA, -12))
	}))

	// light breaks, the sun rises, the beasts
	part(o.horns, "horns", 0.14, 0.95, func(s S) []N {
		return perBar(s, func(b int, br bar) []N {
			if (arrange.In(s, day1, day4) && morning(b)) || (s.Name == day6 && !morning(b)) {
				return []N{score.Chord(score.Pitches(br.mid, -12), 3)}
			}
			return nil
		})
	})

	// Adam answers Eve with the figure of the cross; on the seventh day the cross leads through the
	// Amen into the rest
	part(o.horn, "horn", 0.33, 0.95, func(s S) []N {
		switch s.Name {
		case day6:
			return score.Join(rest(6), answer)
		case day7:
			return score.Join(rest(final-1), cross, rest(3))
		}
		return nil
	})

	// the stars of the first morning and of the fourth night
	part(o.celesta, "celesta", 0.217, 0.9, func(s S) []N {
		return perBar(s, func(b int, br bar) []N {
			if (s.Name == day1 && morning(b)) || (s.Name == day4 && !morning(b)) {
				return stars(br.chord, b)
			}
			return nil
		})
	})

	part(o.bellPad, "bell pad", 0.0471, 1.0, func(s S) []N {
		return perBar(s, func(b int, br bar) []N {
			if (s.Name == day1 && morning(b)) || (s.Name == day4 && !morning(b)) {
				return []N{score.Chord(br.mid, 3)}
			}
			return nil
		})
	})

	// the bells ring once, as the Amen comes home
	part(o.bells, "bells", 0.12, 1.0, arrange.Only(day7, func(s S) []N {
		return score.Join(rest(final), []N{{Pitches: []float64{63, 70, 75}, Beats: 12, Hold: 12, Velocity: 0.5}})
	}))

	// the great creatures of the sea, low in the deep
	part(o.deep, "the deep", 0.2, 1.0, arrange.Bars(day5, func(s S, b int) []N {
		if morning(b) {
			return nil
		}
		return []N{{Pitches: []float64{at(s, b).root}, Beats: 3, Hold: 3.2}}
	}))

	// mankind: the hands of man on the piano
	part(o.grand, "piano", 0.2, 1.0, arrange.Bars(day6, func(s S, b int) []N {
		if !morning(b) {
			return nil
		}
		br := at(s, b)
		return arp(append([]float64{br.root + 12, br.root + 19}, br.mid...))
	}))

	// Eve sings the theme of man; on the seventh day the soprano leads the Doxology
	part(o.soprano, "soprano", 0.34, 0.95, func(s S) []N {
		switch s.Name {
		case day6:
			return score.Join(rest(6), theme)
		case day7:
			return score.Join(doxology, rest(5))
		}
		return nil
	})

	// the choir: the light, the morning stars, man, the Doxology
	part(o.choir, "choir", 0.11, 1.0, func(s S) []N {
		return perBar(s, func(b int, br bar) []N {
			switch {
			case arrange.In(s, day1, day4, day6) && morning(b):
				return []N{score.Chord(br.mid, 3)}
			case s.Name == day7:
				return ringing(b, br.mid)
			}
			return nil
		})
	})

	// the organ plays the Doxology in four parts and holds the Amen
	part(o.organ, "organ", 0.1, 1.0, arrange.Bars(day7, func(s S, b int) []N {
		br := at(s, b)
		return ringing(b, append([]float64{br.root}, br.mid...))
	}))

	return clashes
}
