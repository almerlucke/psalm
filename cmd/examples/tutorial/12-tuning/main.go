// Step 12: tuning.
//
// Notes are written in MIDI pitches; a tuning decides which frequency each pitch sounds at. Without
// one the synths use equal temperament: twelve even semitones, the tuning of a piano, in which every
// key is equally (slightly) out of tune. Give a part a tuning and it plays in that instead:
//
//	score.Config{Tuning: harmony.EqualTemperament(440)}          the piano tuning, A4 at 440 Hz
//	score.Config{Tuning: harmony.MustKey("C major").Just(440)}   just intonation in C
//
// Just intonation tunes every note to a pure ratio of the tonic: the third 5/4, the fifth 3/2, the
// sixth 5/3. The chords on I, IV, V, vi and iii then sound still and pure, without the slow beating
// you hear in equal temperament (listen to the sustained chords below, first equal, then just).
// The chord on ii is a comma out of tune, so the progression avoids it.
//
// A tuning is a plain function from a pitch to a frequency (score.Tuning), so you can write your
// own: func(pitch float64) float64 { return 432 * math.Exp2((pitch-69)/12) } is equal temperament
// with A4 at 432 Hz.
//
//	go run ./cmd/examples/tutorial/12-tuning
package main

import (
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/cmd/examples/tutorial/tutorial"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/dx"
	"github.com/almerlucke/psalm/tools/harmony"
)

func main() {
	p := tutorial.New(60)

	form := arrange.NewForm(
		arrange.Section{Name: "Equal temperament", Bars: 4},
		arrange.Section{Name: "Just intonation", Bars: 4},
	)
	key := harmony.MustKey("C major")
	chords := key.MustProgression("I IV V I")

	organ := dx.NewPoly(dx.ChurchOrganPreset(), 12, psalm.Stereo).AddTo(p)

	// long chords: a low root and a close chord, a bar each
	chordsOf := func(s arrange.Section, bar int) []score.Note {
		c := chords[bar%4]
		return []score.Note{score.Chord(append([]float64{c.BassPitch(36)}, c.Close(55)...), 4)}
	}
	// the same chords twice, a part per tuning, each in its own section
	form.Part(p, organ, "equal", score.Config{Amp: 0.25, Legato: 0.97, Tuning: harmony.EqualTemperament(440)},
		arrange.Bars("Equal temperament", chordsOf))
	form.Part(p, organ, "just", score.Config{Amp: 0.25, Legato: 0.97, Tuning: key.Just(440)},
		arrange.Bars("Just intonation", chordsOf))

	arrangement := arrange.NewArrangement(form).PrintSections(true)
	arrangement.AddTo(p)

	mix := mixer.New(p)
	hall := fdnreverb.DefaultConfig()
	hall.Mix = 1
	mix.Bus("hall", mixer.StripConfig{}, fdnreverb.New(hall).AddTo(p))
	mix.Channel("organ", organ, mixer.StripConfig{Sends: map[string]float64{"hall": 0.3}})
	tutorial.Play(p, mix.Master(), arrangement.Duration())
}
