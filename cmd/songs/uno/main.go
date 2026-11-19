// Uno: a short piece for two Uno organs and a piano, in the manner of Clair de lune: Db major, slow, a melody over
// arpeggios, a bass that rings on under them. It goes round three times, each time with more: the first Uno alone, then
// the second Uno answering it, then a piano dancing around the melody: a counterpoint that moves against it and a
// rolling figure under it, both loose and off the beat, with the pedal changed at every bar. Then everything plays
// and the theme is varied three times: its notes filled in with runs, the same tune syncopated over arpeggios twice as
// fast, and at last the theme high and broad, the piano's bass under it, ending in a held chord.
//
// The Uno Synth Pro is paraphonic with three voices, and the piece is written for that: a held bass, an
// arpeggio of one note at a time and the melody make three, and where a fourth note comes (the last bar)
// a voice is stolen and slides, the legato glide. The sound is the unoOrgan preset of the subtractive synth,
// played with its three voices (subtractive.Voices).
//
// The aftertouch to the hall of the keyboard example (-hall-aftertouch) is played here by the arrangement:
// the send to the hall of the one channel swells in the second half of a bar where a note is held, as the
// hand would lean on the key, and lets go at the next bar.
//
//	go run ./cmd/songs/uno                          play it
//	go run ./cmd/songs/uno -render uno.wav -seed 7  render it once, with the tail
package main

import (
	"flag"
	"fmt"
	"math"

	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/piano"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/tools/render"
)

const baseSend = 0.25 // the send to the hall without any pressure

// lean is the aftertouch: in the bars it names (from 0) the pressure rises over the second half of the bar to
// the amount (0-1) and is gone at the next bar; the send to the hall goes from baseSend up to peak with it
func lean(bars arrange.Span, peak float64, pressed map[int]float64) arrange.Curve {
	return func(progress float64) float64 {
		b := progress * float64(bars)
		bar := int(math.Floor(b))
		amount, ok := pressed[bar]
		if !ok {
			return baseSend
		}
		x := max(0, min((b-float64(bar)-0.3)/0.65, 1))
		return baseSend + (peak-baseSend)*amount*(0.5-0.5*math.Cos(math.Pi*x))
	}
}

// the chords of the eight bars, the bass and the three notes of the chord going up
var chords = [8][4]float64{{49, 56, 61, 65}, {46, 58, 61, 65}, {42, 54, 58, 61}, {44, 56, 60, 63}, {49, 56, 61, 65}, {42, 54, 58, 61}, {44, 56, 60, 66}, {49, 56, 61, 65}}

// quick is the arpeggio twice as fast: the bass rings for the bar, then the notes of the chord in sixteenths,
// one at a time, up and down
func quick() []score.Note {
	var notes []score.Note
	for _, c := range chords {
		bass := score.N(c[0], 0.25)
		bass.Hold = 4
		notes = append(notes, bass)
		for i := range 15 {
			notes = append(notes, score.N([]float64{c[1], c[2], c[3], c[2]}[i%4], 0.25))
		}
	}
	return notes
}

// roll is the chord rolled by the piano in sixteenths, in a wave: three notes up, three back, a rest
func roll() []score.Note {
	var notes []score.Note
	for _, c := range chords {
		low := []float64{c[1] - 12, c[2] - 12, c[3] - 12}
		wave := []float64{low[0], low[1], low[2], low[1], low[0], low[1], low[2], low[1], low[0], low[1], low[2], low[1], low[0], 0, low[1], low[2]}
		notes = append(notes, score.Shape(score.Notes(0.25, wave...), score.Velocities(0.6, 0.35, 0.4, 0.35))...)
	}
	return notes
}

func main() {
	renderTo := flag.String("render", "", "render the piece once to this WAV file instead of playing it")
	seed := flag.Uint64("seed", 0, "seed for everything random (the feel of the parts): the same seed plays the same")
	flag.Parse()
	if *seed != 0 {
		psalm.Seed(*seed)
	}
	psalm.PushConfig(psalm.Config{BPM: 56, PPQ: 240, SampleRate: 44100, BufferSize: 512})
	var p *psalm.Psalm
	if *renderTo != "" {
		p = psalm.New(nil)
	} else {
		p = psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	}
	defer p.Close()

	form := arrange.NewForm(
		arrange.Section{Name: "Theme", Bars: 8},                       // the Uno alone
		arrange.Section{Name: "Answer", Bars: 8},                      // the second Uno comes in
		arrange.Section{Name: "Counterpoint", Bars: 8, Dynamics: 1.1}, // and the piano
		arrange.Section{Name: "Variation 1", Bars: 8, Dynamics: 1.15}, // everything plays: the theme with runs
		arrange.Section{Name: "Variation 2", Bars: 8, Dynamics: 1.2},  // syncopated, over arpeggios of sixteenths
		arrange.Section{Name: "Variation 3", Bars: 8, Dynamics: 1.3},  // high and broad, the ending
	)

	const preset = "unoOrgan"
	uno := subtractive.NewPoly(subtractive.Presets[preset](), subtractive.Voices(preset, 3)).AddTo(p)
	keys := piano.NewPoly(piano.FeltPreset(), 8).AddTo(p)
	uno2 := subtractive.NewPoly(subtractive.Presets[preset](), subtractive.Voices(preset, 3)).AddTo(p) // another one, for the answer

	// the arpeggios: the bass rings for the bar (the _4), then seven eighths of the three notes of the chord
	// going up and down, one at a time
	arpeggio := func(s arrange.Section) []score.Note {
		if s.Name == "Variation 2" {
			return quick() // twice as fast
		}
		return score.MustParse(`
			Db3:.5_4 Ab3:.5 Db4 F4 Db4 Ab3 Db4 F4 |
			Bb2:.5_4 Bb3:.5 Db4 F4 Db4 Bb3 Db4 F4 |
			Gb2:.5_4 Gb3:.5 Bb3 Db4 Bb3 Gb3 Bb3 Db4 |
			Ab2:.5_4 Ab3:.5 C4 Eb4 C4 Ab3 C4 Eb4 |
			Db3:.5_4 Ab3:.5 Db4 F4 Db4 Ab3 Db4 F4 |
			Gb2:.5_4 Gb3:.5 Bb3 Db4 Bb3 Gb3 Bb3 Db4 |
			Ab2:.5_4 Ab3:.5 C4 Gb4 C4 Ab3 C4 Gb4 |
			Db3:.5_6 Ab3:.5 Db4 F4 Db4 Ab3 Db4:.5 Ab3:.5`)
	}
	melody := func(s arrange.Section) []score.Note {
		switch s.Name {
		case "Variation 1": // the notes of the theme filled in with runs
			return score.MustParse(`
				F5:1 Ab5:.5 F5:.5 Ab5:1 Gb5:.5 F5:.5 |
				Db5:.5 F5:.5 Ab5:1 Bb5:1 Ab5:.5 F5:.5 |
				Gb5:1 Bb5:1 F5:.5 Eb5:.5 F5:.5 Eb5:.5 |
				Eb5:1 F5:.5 Eb5:.5 C5:1 Eb5:.5 Ab5:.5 |
				F5:.5 Ab5:.5 Db6:1 Ab5:1 F5:1 |
				Bb5:1 Db6:.5 Bb5:.5 Ab5:1 Gb5:.5 Ab5:.5 |
				Eb5:.5 F5:.5 Gb5:1 Ab5:1 Gb5:.5 F5:.5 |
				F5:2 Eb5:1 F5:1`)
		case "Variation 2": // the same tune syncopated, the notes pushed a half beat early
			return score.MustParse(`
				r:.5 F5:1.5 Ab5:1 Gb5:1 |
				Db5:.5 F5:.5 Ab5:1.5 F5:.5 Db5:1 |
				r:.5 Gb5:1.5 F5:1 Eb5:1 |
				Eb5:1.5 C5:.5 Eb5:1 Ab5:1 |
				r:.5 F5:1.5 Ab5:1 Db6:1 |
				Bb5:1.5 Ab5:.5 Gb5:1 Eb5:1 |
				Eb5:1 F5:1 Gb5:1 Ab5:1 |
				F5:2 Ab5:2`)
		case "Variation 3": // high and broad, the ending
			return score.MustParse(`
				F5:1 Ab5:1 Db6:2 |
				Db6:1 Bb5:1 Ab5:2 |
				Gb5:1 Bb5:1 Db6:2 |
				C6:2 Eb6:1 C6:1 |
				Db6:1.5 Ab5:.5 F5:1 Db6:1 |
				Bb5:1 Db6:1 Gb6:2 |
				Ab5:1 Gb5:1 F5:1 Eb5:1 |
				Db6:4_7`)
		}
		return score.MustParse(`
			F5:2 Ab5:1 Gb5:1 |
			Db5:1 F5:1 Ab5:2 |
			Gb5:2 F5:1 Eb5:1 |
			Eb5:2 C5:1 Eb5:1 |
			F5:2 Ab5:1 Db6:1 |
			Bb5:2 Ab5:1 Gb5:1 |
			Eb5:2 F5:1 Gb5:1 |
			F5:4_5`)
	}
	form.Part(p, uno, "arpeggio", score.Config{Amp: 0.3, Legato: 0.9, Velocity: 0.6}, arpeggio)
	form.Part(p, uno, "melody", score.Config{Amp: 0.4, Legato: 0.97, Velocity: 0.7}, melody)

	// the answer, on a second Uno an octave up and softer: it comes in on the third beat of every bar, where the
	// melody has just stated its phrase, and one note at a time (three voices are plenty)
	answer := func(s arrange.Section) []score.Note {
		switch s.Name {
		case "Variation 1", "Variation 3": // the answer in eighths
			return score.MustParse(`
				r:2 Ab6:.5 Bb6:.5 Ab6:.5 F6:.5 |
				r:2 F6:.5 Ab6:.5 F6:.5 Db6:.5 |
				r:2 Bb6:.5 Db7:.5 Bb6:.5 Gb6:.5 |
				r:2 C6:.5 Eb6:.5 C6:.5 Ab5:.5 |
				r:2 F6:.5 Ab6:.5 F6:.5 Ab6:.5 |
				r:2 Db6:.5 Bb5:.5 Db6:.5 Gb6:.5 |
				r:2 C6:.5 Eb6:.5 Gb6:.5 Eb6:.5 |
				r:2 Ab6:1 F6:1_3`)
		case "Variation 2": // long notes that come in on the second beat
			return score.MustParse(`
				r:1 Ab6:1.5 F6:1.5 |
				r:1 F6:1.5 Db6:1.5 |
				r:1 Bb6:1.5 Gb6:1.5 |
				r:1 C6:1.5 Eb6:1.5 |
				r:1 Ab6:1.5 F6:1.5 |
				r:1 Bb6:1.5 Db7:1.5 |
				r:1 C6:1.5 Gb6:1.5 |
				r:1 Ab6:3`)
		}
		return score.MustParse(`
			r:2 Ab6:1 F6:1 |
			r:2 F6:1 Db6:1 |
			r:2 Bb6:1 Gb6:1 |
			r:2 C6:1 Eb6:1 |
			r:2 F6:1 Ab6:1 |
			r:2 Db6:1 Bb6:1 |
			r:2 C6:1 Gb6:1 |
			r:1 Ab6:3_5`)
	}
	form.Part(p, uno2, "answer", score.Config{Amp: 0.14, Legato: 0.95, Velocity: 0.5}, answer)

	// the piano dances around the melody in two lines that never fall on the beat together: a low one that rolls
	// the chord up and back in uneven steps (a dotted figure, a rest where the melody moves, a leap), and a high one,
	// the counterpoint, that comes in between the notes of the melody and goes down where it goes up
	dance := func(s arrange.Section) []score.Note {
		if s.Name == "Variation 2" { // rolling the chord, sixteenths in a wave
			return roll()
		}
		return score.MustParse(`
			r:.5 F3:.5 Ab3:.5 Db4:1.5 Ab3:.5 F3:.5 |
			Bb2:.75 Db3:.75 F3:.5 Bb3:1 F3:.5 Db3:.5 |
			r:1 Gb3:.5 Bb3:.5 Db4:1 Bb3:.5 Gb3:.5 |
			Eb3:.5 Ab3:1 C4:.5 r:.5 Ab3:.5 Eb3:1 |
			r:.5 Ab3:.5 F4:.75 Db4:.75 Ab3:.5 F3:1 |
			Db3:.5 Gb3:.5 Bb3:1 r:.5 Db4:.5 Bb3:.5 Gb3:.5 |
			Eb3:.5 Gb3:.5 C4:1 Eb4:.5 C4:.5 Gb3:1 |
			Db3:1 Ab3:.5 F4:.5 Db4:2_4`)
	}
	counterpoint := func(s arrange.Section) []score.Note {
		return score.MustParse(`
			r:1.5 Db5:.5 Bb4:1 Ab4:1 |
			Bb4:.5 Ab4:.5 r:.5 F5:.5 Db5:1 Bb4:1 |
			r:.5 Bb4:1 Db5:.5 F5:.5 Db5:1 Bb4:.5 |
			Ab4:1.5 Eb5:.5 C5:1 r:.5 Ab4:.5 |
			r:1 Db5:.5 Ab4:.5 F4:1 Ab4:.5 Db5:.5 |
			Gb5:1.5 Db5:.5 Bb4:1 r:1 |
			r:.5 C5:.5 Bb4:1 Db5:.5 Gb4:.5 Bb4:1 |
			r:1 Ab4:1 Db5:2_3`)
	}
	// played a little loose: a light swing on the eighths, the notes a few milliseconds off, laid back
	loose := score.Feel{Swing: 0.57, Timing: 7, Push: 5, Velocity: 0.12}
	form.Part(p, keys, "dance", score.Config{Amp: 0.5, Legato: 0.97, Velocity: 0.55}, dance).SetFeel(loose)
	form.Part(p, keys, "counterpoint", score.Config{Amp: 0.4, Legato: 0.98, Velocity: 0.62}, counterpoint).SetFeel(loose)

	// the piano's bass, in the last variation only: the root of every bar, two octaves under the Uno's
	bass := func(s arrange.Section) []score.Note {
		if s.Name != "Variation 3" {
			return []score.Note{score.Rest(float64(s.Bars) * 4)}
		}
		var notes []score.Note
		for i, root := range []float64{37, 34, 42, 44, 37, 42, 44, 37} {
			n := score.N(root, 4)
			n.Hold = 4
			if i == 7 {
				n.Hold = 7
			}
			notes = append(notes, n)
		}
		return notes
	}
	form.Part(p, keys, "bass", score.Config{Amp: 0.45, Legato: 1, Velocity: 0.6}, bass)

	// a channel for each, and a hall for the aftertouch to open
	mix := mixer.New(p)
	hall := fdnreverb.DefaultConfig()
	hall.Decay = 5
	hall.Mix = 1
	mix.Bus("hall", mixer.StripConfig{}, fdnreverb.New(hall).AddTo(p))
	channel := mix.Channel("uno", uno, mixer.StripConfig{Volume: -2, Pan: -0.15, Sends: map[string]float64{"hall": baseSend}})
	answerChannel := mix.Channel("uno answer", uno2, mixer.StripConfig{Volume: -6, Pan: 0.3, Sends: map[string]float64{"hall": baseSend}})

	pianoChannel := mix.Channel("piano", keys, mixer.StripConfig{Volume: -3, Pan: 0.1, Sends: map[string]float64{"hall": 0.3}})

	// the leaning grows with every time round
	pressure := arrange.Sections{
		"Theme":        lean(form.Span("Theme"), 0.6, map[int]float64{3: 0.6, 7: 1}),
		"Answer":       lean(form.Span("Answer"), 0.8, map[int]float64{1: 0.5, 3: 0.8, 7: 1}),
		"Counterpoint": lean(form.Span("Counterpoint"), 1, map[int]float64{1: 0.6, 3: 0.9, 5: 0.7, 7: 1}),
		"Variation 1":  lean(form.Span("Variation 1"), 1, map[int]float64{1: 0.7, 3: 1, 5: 0.7, 7: 1}),
		"Variation 2":  lean(form.Span("Variation 2"), 1, map[int]float64{1: 0.8, 3: 1, 5: 0.8, 7: 1}),
		"Variation 3":  lean(form.Span("Variation 3"), 1, map[int]float64{0: 0.5, 1: 0.8, 3: 1, 5: 0.9, 7: 1}),
	}
	arrangement := arrange.NewArrangement(form).AutomateFunc(pressure.Else(baseSend), func(v float64) {
		channel.SetSend("hall", v)
		answerChannel.SetSend("hall", min(1, v*1.15)) // the answer sits a little further back, in the hall
	})
	pedal := arrange.Sections{}
	for _, name := range []string{"Counterpoint", "Variation 1", "Variation 2", "Variation 3"} {
		pedal[name] = form.Span(name).Pedal(0, 8)
	}
	arrangement.AutomateMessage(pedal, keys, "sustain")
	arrangement.Play(answerChannel, "Answer", "Counterpoint", "Variation 1", "Variation 2", "Variation 3")
	arrangement.Play(pianoChannel, "Counterpoint", "Variation 1", "Variation 2", "Variation 3")
	arrangement.AddTo(p)

	p.Out(mix.Master())
	if *renderTo == "" {
		fmt.Println("playing, Ctrl-C to stop")
		if err := p.Listen(); err != nil {
			panic(err)
		}
		return
	}
	stats, err := render.ToFile(p, *renderTo, render.Options{Seconds: arrangement.Duration(), Tail: 8})
	if err != nil {
		panic(err)
	}
	fmt.Println(stats)
}
