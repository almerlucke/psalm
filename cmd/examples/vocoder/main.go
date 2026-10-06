package main

import (
	"github.com/almerlucke/genny/bucket"
	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/genny/sequence"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/modules/controls/templ"
	"github.com/almerlucke/psalm/modules/controls/timer"
	"github.com/almerlucke/psalm/modules/controls/timer/steps"
	"github.com/almerlucke/psalm/modules/effects/freeverb"
	"github.com/almerlucke/psalm/modules/effects/latency"
	"github.com/almerlucke/psalm/modules/effects/pan"
	"github.com/almerlucke/psalm/modules/effects/pingpong"
	"github.com/almerlucke/psalm/modules/effects/vocoder"
	"github.com/almerlucke/psalm/modules/oscillators/lfo"
	"github.com/almerlucke/psalm/modules/player"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/sndfile"
)

// seq plays the values in order and loops (a bucket would shuffle them)
func seq[T any](values ...T) *sequence.Sequence[T] {
	return sequence.NewLoop(values...)
}

// random picks one of the values at random every time
func random[T any](values ...T) *bucket.Bucket[T] {
	return bucket.NewLoop(values...).SetMode(bucket.Random)
}

// chords plays a chord progression, every chord lasts the given number of beats. The pitch
// generator produces a slice per chord, which makes the template send one note message per pitch
func chords(p *psalm.Psalm, synth psalm.Modular, beats float64, amp float64, progression ...[]float64) {
	beatSec := 60.0 / psalm.BPM()

	synth.ValueIn(templ.NewDurationTrigger(beats*beatSec*0.98, amp, template.Template{
		"pitch": seq(progression...),
	}).ValueIn(timer.WithTicks(beats * float64(psalm.PPQ())).AddTo(p)))
}

// note is a pitch (MIDI, 0 = rest) with a length in beats
type note struct {
	pitch float64
	beats float64
}

// melody plays a single note line, legato scales how long each note sounds relative to its length
func melody(p *psalm.Psalm, synth psalm.Modular, amp float64, legato float64, notes ...note) {
	var (
		stps    []any
		pitches []float64
		durs    []float64
		beatSec = 60.0 / psalm.BPM()
	)

	for _, n := range notes {
		if n.pitch == 0 {
			stps = append(stps, -n.beats)
			continue
		}
		stps = append(stps, n.beats)
		pitches = append(pitches, n.pitch)
		durs = append(durs, n.beats*beatSec*legato)
	}

	synth.ValueIn(templ.NewDurationTrigger(seq(durs...), amp, template.Template{
		"pitch": seq(pitches...),
	}).ValueIn(timer.WithSteps(steps.QuickSteps(stps...)).AddTo(p)))
}

func main() {
	psalm.PushConfig(psalm.Config{
		BPM:        90,
		PPQ:        240,
		SampleRate: 44100,
		BufferSize: 1024,
	})

	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	// Modulator: looping speech
	speech := player.New(player.Config{
		SndFile: sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/John_1-1.wav", 4),
		Loop:    false,
		Speed:   1.0,
		Amp:     0.8,
	}).AddTo(p).ValueIn(timer.WithTicks(float64(psalm.PPQ() * 8)).AddTo(p))

	// Carrier: a bright, harmonically rich chord pad, vocoders need lots of harmonics to shape
	carrierCfg := subtractive.PadPreset()
	carrierCfg.Cutoff = 7000.0
	carrierCfg.FilterEnvAmount = 0.0
	carrierCfg.Resonance = 0.0
	carrierCfg.NoiseLevel = 0.05
	carrierCfg.AmpEnv = *adsr.NewSetting(1.0, 30.0, 1.0, 10.0, 1.0, 300.0)

	carrier := subtractive.NewPoly(carrierCfg, 12).AddTo(p)

	// Am F C G, two bars per chord
	chords(p, carrier, 8, 0.31,
		[]float64{45, 57, 60, 64}, // Am: A2 A3 C4 E4
		[]float64{41, 57, 60, 65}, // F:  F2 A3 C4 F4
		[]float64{48, 55, 60, 64}, // C:  C3 G3 C4 E4
		[]float64{43, 55, 59, 62}, // G:  G2 G3 B3 D4
	)

	// Vocoder: speech on input 0, carrier on input 1
	vcfg := vocoder.DefaultConfig()
	vcfg.Gain = 1.0
	voc := vocoder.New(vcfg)
	voc.AddTo(p)
	voc.In(speech, 0, 0, carrier, 0, 1)

	// Flute: a slow, melancholic single note line over the same progression (8 beats per
	// chord). It opens low, answers an octave up over F, peaks over C and steps back down over G,
	// hanging on the leading note B before the loop returns to the low A minor phrase
	const (
		C4 = 60.0
		D4 = 62.0
		E4 = 64.0
		G4 = 67.0
		A4 = 69.0
		B4 = 71.0
		C5 = 72.0
		D5 = 74.0
		E5 = 76.0
		F5 = 77.0
		G5 = 79.0
	)

	flute := subtractive.NewPoly(subtractive.FlutePreset(), 6).AddTo(p)

	melody(p, flute, 0.145, 0.95,
		note{E4, 3}, note{D4, 1}, note{C4, 4}, // Am, low
		note{F5, 3}, note{E5, 1}, note{C5, 2}, note{A4, 2}, // F, an octave up
		note{E5, 3}, note{D5, 1}, note{E5, 2}, note{G5, 2}, // C, the peak
		note{D5, 4}, note{B4, 3}, note{G4, 1}, // G, stepping back down
	)

	// Pad: soft
	padCfg := subtractive.PWMPadPreset()
	padCfg.Cutoff = 500.0

	pad := subtractive.NewPoly(padCfg, 12).AddTo(p)

	chords(p, pad, 8, 0.124,
		[]float64{57, 64, 72}, // A3 E4 C5
		[]float64{43, 60, 69}, // G2 C4 A4
		[]float64{60, 67, 76}, // C4 G4 E5
		[]float64{55, 62, 71}, // G3 D4 B4
	)

	// Falling stars: delicate, glassy high notes from the A minor pentatonic at random, irregular
	// times. Each star starts above its note and falls onto it while fading (the filter envelope
	// drives the pitch), streaking sideways through the stereo field and rippling away through a
	// pingpong delay
	starCfg := subtractive.GlassKeysPreset()
	starCfg.FMAmount = 1.2
	starCfg.Osc2Level = 0.15
	starCfg.FilterEnvToPitch = 7.0 // overridden per star below
	starCfg.FilterEnv = *adsr.NewSetting(1.0, 1.0, 0.0, 1300.0, 1.0, 400.0)
	starCfg.AmpEnv = *adsr.NewSetting(1.0, 2.0, 0.0, 1400.0, 1.0, 500.0)
	starCfg.LFO1 = subtractive.LFOConfig{Waveform: subtractive.LFOSine, Rate: 7.0, Pitch: 0.06}

	stars := subtractive.NewPoly(starCfg, 8).AddTo(p)

	ppq := float64(psalm.PPQ())
	starTimer := timer.New(random(
		// note spacing in ticks, negative values are pauses so the stars appear irregularly
		ppq, 1.5*ppq, 2*ppq, 3*ppq, 4*ppq, -3*ppq, -4*ppq, -6*ppq, -8*ppq,
	)).AddTo(p)

	// notes last long enough for the whole fall, the envelopes decay to silence by themselves
	stars.ValueIn(templ.NewDurationTrigger(1.6, random(0.0227, 0.0397, 0.0454, 0.0624), template.Template{
		"pitch":            random(81.0, 84.0, 86.0, 88.0, 91.0, 93.0, 96.0), // A5 C6 D6 E6 G6 A6 C7
		"filterEnvToPitch": random(5.0, 7.0, 12.0),                           // how far each star falls (semitones)
	}).ValueIn(starTimer))

	// a steady saw sweep makes every star streak sideways across the stereo field as it falls
	starsPanned := pan.New(0.0).AddTo(p).In(stars)
	starsPanned.ValueIn(lfo.New(true, 0.3, -0.7, 0.7, lfo.Saw).AddTo(p), 0, 0)

	starRipples := pingpong.New(pingpong.Config{Time: 0.75, Feedback: 0.5, Mix: 0.5}).AddTo(p).In(starsPanned)

	// Delay the piano and pad by the vocoder latency so everything lines up
	// Panning: the voice stays in the center, the piano sits to the left and the pad to the right
	voicePanned := pan.New(0.0).AddTo(p).In(voc)
	flutePanned := pan.New(-0.5).AddTo(p).In(flute)
	padPanned := pan.New(0.45).AddTo(p).In(pad)

	accompaniment := latency.New(psalm.Stereo, voc.Latency()).AddTo(p).In(flutePanned, 0, 0, padPanned, 0, 0)

	p.Out(freeverb.New(0.9, 0.5, 0.9, 0.3, 0.6).AddTo(p).In(voicePanned, 0, 0, accompaniment, 0, 0, starRipples, 0, 0))

	//err := p.ToSoundFile("vocoder.wav", writer.WAV, 180, 44100.0, true)
	//if err != nil {
	//	panic(err)
	//}

	_ = p.Listen()
}
