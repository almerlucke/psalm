package main

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/tools/harmony"
)

const (
	bpm     = 92
	restBPM = 80 // the tag
	endBPM  = 56 // the last bars of the tag slow down to it
	evening = 4  // the bar of the intro where the light comes (the darkness lifts)
	beats   = 3  // to the bar: the Trinity
)

// day is a section of the piece (the name stayed from the days it was first made of): its bars, the dynamics it plays at, its chords (one a bar) and the verse printed
// as it begins. The form is a jazz tune: an intro over the dark waters, the head (the written melody, A A B A), a solo of the steel pan over the changes, a break of
// the percussion, a solo of the flute and the vibraphone, the head out, and a tag that ends at the cross
type day struct {
	name   string
	bars   int
	dyn    float64
	chords string
	verse  string
}

// the changes of a chorus of the head: A is eight bars (I vi ii V, iii vi ii V), B the bridge, the last A ends on the tonic
const (
	chorusA  = "I vi ii V7 iii vi ii V7"
	chorusA2 = "I vi ii V7 iii vi IV I"
	chorusB  = "IV V7 iii vi ii V7 I V7"
)

var days = []day{
	{"1 Intro: in the beginning", 8, 1.2, "I I ii V7 I I ii V7", // the dark waters and the Word, a vamp on the tonic; the light comes in its fifth bar
		"In the beginning God created the heaven and the earth. And the earth was without form, and void; and darkness was upon the face of the deep. (Genesis 1:1, 2)\n" +
			"In the beginning was the Word, and the Word was with God, and the Word was God. All things were made by him. (John 1:1, 3)"},
	{"2 Head: light", 32, 1, chorusA + " " + chorusA + " " + chorusB + " " + chorusA2, // A A B A
		"And God said, Let there be light: and there was light. And God saw the light, that it was good. (Genesis 1:3, 4)"},
	{"3 Solo: the waters and the land", 32, 1, chorusA + " " + chorusA + " " + chorusB + " " + chorusA2, // the steel pan
		"And God said, Let the waters under the heaven be gathered together unto one place, and let the dry land appear. (Genesis 1:9)"},
	{"4 Break: the creatures", 16, 1.05, "ii V7 I vi ii V7 I vi ii V7 I vi ii V7 I I", // the percussion
		"And God said, Let the earth bring forth the living creature after his kind, cattle, and creeping thing, and beast of the earth. (Genesis 1:24)"},
	{"5 Solo: the heavens", 24, 1, chorusA + " " + chorusB + " " + chorusA2, // the flute, then the vibraphone
		"And God said, Let there be lights in the firmament of the heaven to divide the day from the night; and let them be for signs, and for seasons, and for days, and years. (Genesis 1:14)"},
	{"6 Head out: mankind", 32, 1.15, chorusA + " " + chorusA + " " + chorusB + " " + chorusA2, // the head again, the pan an octave above
		"So God created man in his own image, in the image of God created he him; male and female created he them. And God saw every thing that he had made, and, behold, it was very good. (Genesis 1:27, 31)"},
	{"7 Tag: rest", 12, 1, "I vi ii V7 I vi IV V7 IV I I I", // the plagal Amen, the last chord rings
		"And God blessed the seventh day, and sanctified it: because that in it he had rested from all his work which God created and made. (Genesis 2:3)\n" +
			"For by him were all things created, that are in heaven, and that are in earth, visible and invisible: all things were created by him, and for him. " +
			"And, having made peace through the blood of his cross, by him to reconcile all things unto himself. (Colossians 1:16, 20)"},
}

// sectionsText is the sections of the song for the song module: a bar of three beats, the days
func sectionsText() string {
	parts := []string{"3/4"}
	for _, d := range days {
		parts = append(parts, fmt.Sprintf("%s %d", d.name, d.bars))
	}
	return strings.Join(parts, ", ")
}

// barJS is a bar as the scripts have it: the chord (its root, intervals), the voicings of the upper voices, the middle voices and the bass
type barJS struct {
	Chord string    `json:"c"`
	Root  int       `json:"r"`
	Iv    []int     `json:"iv"`
	High  []float64 `json:"high"`
	Mid   []float64 `json:"mid"`
	Bass  float64   `json:"root"`
}

type dayJS struct {
	Name   string  `json:"name"`
	Number int     `json:"number"`
	Bars   int     `json:"bars"`
	Start  int     `json:"start"`
	Dyn    float64 `json:"dyn"`
	Verse  string  `json:"verse"`
}

// preludeJS is what the first script gets before its own text: the days, the bars with their chords voiced and led (close voicings from G4 and G3, each bar
// moved as little as it can from the one before, the bass at or above D1), the tuning, the seed of the random timing (human: whether the parts play off the grid, swung) and the beats to a bar
func preludeJS(just bool, a4 float64, seed uint64, human bool) (string, error) {
	key := harmony.MustKey("Eb major")
	var bars []barJS
	var ds []dayJS
	var high, mid []float64
	for i, d := range days {
		chords := key.MustProgression(d.chords)
		if len(chords) != d.bars {
			return "", fmt.Errorf("%s: %d chords for %d bars", d.name, len(chords), d.bars)
		}
		ds = append(ds, dayJS{Name: d.name, Number: i + 1, Bars: d.bars, Start: len(bars), Dyn: d.dyn, Verse: d.verse})
		for _, c := range chords {
			if high == nil {
				high, mid = c.Close(67), c.Close(55)
			} else {
				high, mid = harmony.VoiceLead(high, c), harmony.VoiceLead(mid, c)
			}
			bars = append(bars, barJS{Chord: c.Name, Root: c.Root, Iv: c.Intervals, High: high, Mid: mid, Bass: c.BassPitch(34)})
		}
	}
	js := func(v any) string { b, _ := json.Marshal(v); return string(b) }
	return fmt.Sprintf("shared.beats = %d\nshared.just = %v\nshared.a4 = %v\nshared.seed = %d\nshared.human = %v\nshared.days = %s\nshared.bars = %s\nshared.kits = %s\n", beats, just, a4, seed, human, js(ds), js(bars), js(kitKeys())), nil
}

// tempoAt is the tempo in a day, at a bar of it (with the part of a bar): the seventh day is slower, and slower still as the last chord rings
func tempoAt(dayIndex int, bar float64) float64 {
	if dayIndex != len(days)-1 {
		return bpm
	}
	from := float64(days[dayIndex].bars - 4)
	p := max(0, min((bar-from)/4, 1))
	return restBPM + (endBPM-restBPM)*(0.5-0.5*math.Cos(math.Pi*p))
}

// tourSeconds is how long the piece lasts, following the tempo
func tourSeconds() float64 {
	seconds := 0.0
	const step = 1.0 / 64 // of a beat
	for i, d := range days {
		for b := 0.0; b < float64(d.bars*beats); b += step {
			seconds += step * 60 / tempoAt(i, b/beats)
		}
	}
	return seconds
}

// schedule says when every day begins
func schedule() string {
	var b strings.Builder
	seconds := 0.0
	const step = 1.0 / 64
	for i, d := range days {
		fmt.Fprintf(&b, "%6.1fs  %-18s %d bars\n", seconds, d.name, d.bars)
		for beat := 0.0; beat < float64(d.bars*beats); beat += step {
			seconds += step * 60 / tempoAt(i, beat/beats)
		}
	}
	fmt.Fprintf(&b, "%6.1fs  end\n", seconds)
	return b.String()
}

// tempoLane sets the tempo of the patch as the song goes: the seventh day is slower, and slower still in its last bars
type tempoLane struct {
	*psalm.Module
	song *patcher.Song
}

func (t *tempoLane) init() *tempoLane { t.SetSelf(t); return t }

func (t *tempoLane) CtrlTick(tick psalm.CtrlTick) {
	pos, ok := t.song.At(tick, t.Config().PPQ)
	if !ok || pos.Ended {
		return
	}
	want := tempoAt(pos.Section.Index, pos.Progress*float64(pos.Section.Bars))
	if math.Abs(want-t.Env().BPM()) >= 0.01 {
		t.Env().SetBPM(want)
	}
}
