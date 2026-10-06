package main

import (
	"fmt"

	"github.com/almerlucke/genny/sequence"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/blocks/filters/biquad"
	"github.com/almerlucke/psalm/blocks/filters/ladder"
	"github.com/almerlucke/psalm/blocks/filters/svf"
	"github.com/almerlucke/psalm/modules/controls/templ"
	"github.com/almerlucke/psalm/modules/controls/timer"
	"github.com/almerlucke/psalm/modules/controls/timer/steps"
	"github.com/almerlucke/psalm/modules/effects/freeverb"
	"github.com/almerlucke/psalm/modules/filters"
	"github.com/almerlucke/psalm/modules/oscillators/lfo"
	"github.com/almerlucke/psalm/synths/subtractive"
)

// A tour of the filter modules: every section plays an unfiltered saw source through one filter,
// swept by an LFO. Sections play one after the other and the tour loops

const sectionBeats = 16.0 // 4 bars

// seq plays the values in order and loops (a bucket would shuffle them)
func seq[T any](values ...T) *sequence.Sequence[T] {
	return sequence.NewLoop(values...)
}

// note is one or more pitches (MIDI, a chord when more than one) with a length in beats, no
// pitches is a rest
type note struct {
	pitches []float64
	beats   float64
}

// notes builds single notes of equal length
func notes(beats float64, pitches ...float64) []note {
	ns := make([]note, len(pitches))
	for i, p := range pitches {
		ns[i] = note{[]float64{p}, beats}
	}
	return ns
}

// chordNotes builds chords of equal length
func chordNotes(beats float64, chords ...[]float64) []note {
	ns := make([]note, len(chords))
	for i, c := range chords {
		ns[i] = note{c, beats}
	}
	return ns
}

type section struct {
	name   string
	filter *filters.Filter
	notes  []note
	legato float64
	amp    float64
	lfo    *lfo.LFO // modulation, connected to filter input lfoIn
	lfoIn  int
}

// play schedules the notes in the slot of the section: rests before and after, so the tour loops.
// Every pitch value is a slice, so the template sends one note message per pitch (chords)
func play(p *psalm.Psalm, synth psalm.Modular, s *section, index int, numSections int) {
	var (
		stps    []any
		pitches [][]float64
		durs    []float64
		beatSec = 60.0 / psalm.BPM()
	)

	if index > 0 {
		stps = append(stps, -float64(index)*sectionBeats)
	}

	total := 0.0
	for _, n := range s.notes {
		total += n.beats
		if len(n.pitches) == 0 {
			stps = append(stps, -n.beats)
			continue
		}
		stps = append(stps, n.beats)
		pitches = append(pitches, n.pitches)
		durs = append(durs, n.beats*beatSec*s.legato)
	}
	if total != sectionBeats {
		panic(fmt.Sprintf("section %q lasts %v beats, expected %v", s.name, total, sectionBeats))
	}

	if after := numSections - 1 - index; after > 0 {
		stps = append(stps, -float64(after)*sectionBeats)
	}

	synth.ValueIn(templ.NewDurationTrigger(seq(durs...), s.amp, template.Template{
		"pitch": seq(pitches...),
	}).ValueIn(timer.WithSteps(steps.QuickSteps(stps...)).AddTo(p)))
}

func main() {
	psalm.PushConfig(psalm.Config{
		BPM:        110,
		PPQ:        240,
		SampleRate: 44100,
		BufferSize: 1024,
	})

	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	sectionSec := sectionBeats * 60.0 / psalm.BPM()

	sections := []*section{
		{
			name:   "SVF lowpass, resonant, cutoff swept by a sine LFO",
			filter: filters.NewSVF(svf.LowPass, 800, 0.7),
			notes: notes(0.5, 33, 45, 33, 45, 36, 48, 31, 43, 33, 45, 33, 45, 36, 48, 40, 43,
				33, 45, 33, 45, 36, 48, 31, 43, 33, 45, 33, 45, 28, 40, 31, 43),
			legato: 0.8, amp: 0.3,
			lfo: lfo.New(true, 2.0/sectionSec, 150, 4000, lfo.Sine), lfoIn: 1,
		},
		{
			name:   "Moog style ladder lowpass 24 dB, high resonance and drive",
			filter: filters.NewLadder(ladder.LowPass24, 600, 0.85, 2.5),
			notes: notes(0.25,
				45, 45, 57, 45, 45, 55, 45, 57, 45, 45, 57, 45, 48, 45, 55, 57,
				45, 45, 57, 45, 45, 55, 45, 57, 45, 45, 57, 45, 43, 45, 52, 55,
				45, 45, 57, 45, 45, 55, 45, 57, 45, 45, 57, 45, 48, 45, 55, 57,
				45, 45, 57, 45, 45, 55, 45, 57, 45, 45, 57, 45, 43, 45, 52, 55),
			legato: 0.6, amp: 0.3,
			lfo: lfo.New(true, 1.0/sectionSec, 200, 3000, lfo.Triangle), lfoIn: 1,
		},
		{
			name:   "Ladder bandpass 12 dB, sample & hold cutoff",
			filter: filters.NewLadder(ladder.BandPass12, 1000, 0.6, 1.0),
			notes:  notes(1, 57, 60, 64, 67, 57, 60, 64, 69, 57, 60, 64, 67, 55, 59, 62, 67),
			legato: 0.9, amp: 0.18,
			lfo: lfo.New(true, 4.0, 400, 4000, lfo.Rand), lfoIn: 1,
		},
		{
			name:   "Biquad peaking EQ (+15 dB, Q 5) swept like a wah over chords",
			filter: filters.NewBiquad(biquad.Peak, 1000, 5, 15),
			notes:  chordNotes(4, []float64{57, 60, 64}, []float64{53, 57, 60}, []float64{60, 64, 67}, []float64{55, 59, 62}),
			legato: 1.0, amp: 0.1,
			lfo: lfo.New(true, 1.5, 400, 3000, lfo.Sine), lfoIn: 1,
		},
		{
			name:   "Formant filter morphing through the vowels a e i o u",
			filter: filters.NewFormant(0, 12, 1.0),
			notes:  chordNotes(8, []float64{45, 57}, []float64{43, 55}),
			legato: 1.0, amp: 0.6,
			lfo: lfo.New(true, 1.0/sectionSec, 0, 4, lfo.Triangle), lfoIn: 3,
		},
	}

	// Open, unfiltered saw source so the filter modules do all the shaping
	sourceCfg := subtractive.DefaultConfig()
	sourceCfg.Osc2Detune = 0.1
	sourceCfg.Cutoff = 18000.0
	sourceCfg.Resonance = 0.0
	sourceCfg.FilterEnvAmount = 0.0
	sourceCfg.KeyTrack = 0.0

	var mix []any

	for i, s := range sections {
		fmt.Printf("%6.1fs  %s\n", float64(i)*sectionSec, s.name)

		source := subtractive.NewPoly(sourceCfg, 8).AddTo(p)
		play(p, source, s, i, len(sections))

		s.filter.AddTo(p)
		s.filter.In(source)
		s.filter.ValueIn(s.lfo.AddTo(p), 0, s.lfoIn)

		mix = append(mix, s.filter, 0, 0)
	}

	fmt.Printf("%6.1fs  (tour loops)\n", float64(len(sections))*sectionSec)

	p.Out(freeverb.New(0.6, 0.5, 0.9, 0.1, 0.9).AddTo(p).In(mix...))

	//err := p.ToSoundFile("filters.wav", writer.WAV, float64(len(sections))*sectionSec, 44100.0, true)
	//if err != nil {
	//	panic(err)
	//}

	_ = p.Listen()
}
