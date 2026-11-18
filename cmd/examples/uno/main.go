// Uno: a short piece for two Uno organs and nothing else, the second answering the first, in the manner of Clair de lune: Db major, slow, a
// melody over arpeggios, a bass that rings on under them.
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
//	go run ./cmd/examples/uno
//	go run ./cmd/examples/uno -render uno.wav
package main

import (
	"math"

	"github.com/almerlucke/psalm/cmd/examples/tutorial/tutorial"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/subtractive"
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

func main() {
	p := tutorial.New(56)

	form := arrange.NewForm(
		arrange.Section{Name: "Theme", Bars: 8},
		arrange.Section{Name: "Return", Bars: 8, Dynamics: 1.1},
	)

	const preset = "unoOrgan"
	uno := subtractive.NewPoly(subtractive.Presets[preset](), subtractive.Voices(preset, 3)).AddTo(p)
	uno2 := subtractive.NewPoly(subtractive.Presets[preset](), subtractive.Voices(preset, 3)).AddTo(p) // another one, for the answer

	// the arpeggios: the bass rings for the bar (the _4), then seven eighths of the three notes of the chord
	// going up and down, one at a time
	arpeggio := func(s arrange.Section) []score.Note {
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

	// one channel, a hall for the aftertouch to open
	mix := mixer.New(p)
	hall := fdnreverb.DefaultConfig()
	hall.Decay = 5
	hall.Mix = 1
	mix.Bus("hall", mixer.StripConfig{}, fdnreverb.New(hall).AddTo(p))
	channel := mix.Channel("uno", uno, mixer.StripConfig{Volume: -2, Pan: -0.15, Sends: map[string]float64{"hall": baseSend}})
	answerChannel := mix.Channel("uno answer", uno2, mixer.StripConfig{Volume: -6, Pan: 0.3, Sends: map[string]float64{"hall": baseSend}})

	theme, back := form.Span("Theme"), form.Span("Return")
	pressure := arrange.Sections{
		"Theme":  lean(theme, 0.75, map[int]float64{1: 0.5, 3: 0.8, 7: 1}),
		"Return": lean(back, 1, map[int]float64{1: 0.6, 3: 0.9, 5: 0.7, 7: 1}),
	}
	arrangement := arrange.NewArrangement(form).AutomateFunc(pressure.Else(baseSend), func(v float64) {
		channel.SetSend("hall", v)
		answerChannel.SetSend("hall", min(1, v*1.15)) // the answer sits a little further back, in the hall
	})
	arrangement.AddTo(p)

	tutorial.Play(p, mix.Master(), arrangement.Duration())
}
