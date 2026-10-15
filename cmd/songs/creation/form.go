package main

import (
	"fmt"

	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/tools/harmony"
)

const (
	bpm     = 72
	restBPM = 63 // the seventh day
	evening = 6  // bars of every day before the morning
)

// the days
const (
	day1 = "Day 1: light"
	day2 = "Day 2: firmament"
	day3 = "Day 3: land"
	day4 = "Day 4: lights"
	day5 = "Day 5: creatures"
	day6 = "Day 6: mankind"
	day7 = "Day 7: rest"
)

// --- the form: seven days, three beats to the bar ---

var form = func() *arrange.Form {
	verse := func(text string) map[string]any { return map[string]any{"verse": text} }
	f := arrange.NewForm(
		arrange.Section{Name: day1, Bars: 12, Dynamics: 1.3, Params: verse("And God said, Let there be light: and there was light. (Genesis 1:3)")},
		arrange.Section{Name: day2, Bars: 12, Params: verse("And God said, Let there be a firmament in the midst of the waters, and let it divide the waters from the waters. (Genesis 1:6)")},
		arrange.Section{Name: day3, Bars: 12, Params: verse("And God said, Let the earth bring forth grass, the herb yielding seed, and the fruit tree yielding fruit after his kind. (Genesis 1:11)")},
		arrange.Section{Name: day4, Bars: 12, Params: verse("And God said, Let there be lights in the firmament of the heaven to divide the day from the night; and let them be for signs, and for seasons, and for days, and years. (Genesis 1:14)")},
		arrange.Section{Name: day5, Bars: 12, Params: verse("And God said, Let the waters bring forth abundantly the moving creature that hath life, and fowl that may fly above the earth in the open firmament of heaven. (Genesis 1:20)")},
		arrange.Section{Name: day6, Bars: 12, Dynamics: 1.1, Params: verse("So God created man in his own image, in the image of God created he him; male and female created he them. And God saw every thing that he had made, and, behold, it was very good. (Genesis 1:27, 31)")},
		arrange.Section{Name: day7, Bars: 21, Params: verse("And God blessed the seventh day, and sanctified it: because that in it he had rested from all his work which God created and made. (Genesis 2:3)\n" +
			"For by him were all things created, that are in heaven, and that are in earth, visible and invisible: all things were created by him, and for him. " +
			"And, having made peace through the blood of his cross, by him to reconcile all things unto himself. (Colossians 1:16, 20)")},
	)
	f.BeatsPerBar = 3 // the Trinity
	return f
}()

// --- harmony: E flat major, a progression for every day ---

var key = harmony.MustKey("Eb major")

// the chords of every day, one per bar: the evening, then the morning
var days = map[string]string{
	day1: "I5 I5 I5 I5 I5 I5  I IV I V IV I",                    // without form and void (no third), then light
	day2: "vi vi IV IV V V  I V vi iii IV I",                    // the waters divided
	day3: "I I IV IV V V  I IV I IV I IV",                       // the land rises, the plants grow on I and IV
	day4: "vi IV I V vi V  I iii IV I V I",                      // night, then the sun
	day5: "vi vi iii iii IV V  I IV V I IV V",                   // the deep, then the birds
	day6: "I V vi IV I V  I vi IV V IV I",                       // the beasts, then man: very good
	day7: "I vi IV I I iii IV V I V vi I iii I V7 I IV I I I I", // the Doxology, the Amen (IV I) and the rest
}

// bars holds the chord and the voicings of every bar of the song
type bar struct {
	chord harmony.Chord
	high  []float64 // upper voices (violins)
	mid   []float64 // middle voices (choir, organ, horns)
	root  float64   // the bass
}

var bars = func() []bar {
	var bs []bar
	high, mid := []float64(nil), []float64(nil)
	for _, s := range form.Sections {
		chords := key.MustProgression(days[s.Name])
		if len(chords) != s.Bars {
			panic(fmt.Sprintf("%s: %d chords for %d bars", s.Name, len(chords), s.Bars))
		}
		for _, c := range chords {
			if high == nil {
				high, mid = c.Close(67), c.Close(55)
			} else {
				high, mid = harmony.VoiceLead(high, c), harmony.VoiceLead(mid, c)
			}
			bs = append(bs, bar{chord: c, high: high, mid: mid, root: c.BassPitch(34)})
		}
	}
	return bs
}()

// at returns a bar of a section
func at(s arrange.Section, b int) bar { return bars[s.StartBar+b] }

// morning tells whether a bar of a day is in its morning
func morning(b int) bool { return b >= evening }

// final is the bar of the seventh day the last chord comes home on, after the Amen's IV; it rings
// to the end
const final = 17

// timeline lays out the chord of every bar, for the harmony check
func timeline() []harmony.TimedChord {
	var tl []harmony.TimedChord
	for i, b := range bars {
		tl = append(tl, harmony.TimedChord{Beat: float64(i) * form.BeatsPerBar, Beats: form.BeatsPerBar, Chord: b.chord})
	}
	return tl
}
