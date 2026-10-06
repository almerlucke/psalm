package main

import (
	"fmt"

	"github.com/almerlucke/genny/sequence"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/modules/controls/templ"
	"github.com/almerlucke/psalm/modules/controls/timer"
	"github.com/almerlucke/psalm/modules/controls/timer/steps"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/oscillators/lfo"
	"github.com/almerlucke/psalm/synths/grains"
	"github.com/almerlucke/sndfile"
)

// A tour of the granular sampler: every section plays a preset on one of the recordings in assets
// and changes a setting note by note. The tour loops, the schedule is printed at start

const sectionBeats = 16.0 // 4 bars

// seq plays the values in order and loops
func seq[T any](values ...T) *sequence.Sequence[T] {
	return sequence.NewLoop(values...)
}

// note is one or more pitches (a chord when more than one) with a length in beats, no pitches is a rest
type note struct {
	pitches []float64
	beats   float64
}

func single(pitch float64, beats float64) note {
	if pitch == 0 {
		return note{nil, beats}
	}
	return note{[]float64{pitch}, beats}
}

// notes builds single notes of equal length
func notes(beats float64, pitches ...float64) []note {
	ns := make([]note, len(pitches))
	for i, p := range pitches {
		ns[i] = single(p, beats)
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
	cfg    grains.Config
	voices int
	amp    float64
	legato float64
	notes  []note
	params map[string][]float64
	patch  func(p *psalm.Psalm, synth psalm.Modular)
}

// build plays the notes of the section in its slot of the looping tour
func (s *section) build(p *psalm.Psalm, index int, numSections int) psalm.Modular {
	var (
		stps    []any
		pitches [][]float64
		durs    []float64
		total   float64
		beatSec = 60.0 / psalm.BPM()
	)

	if index > 0 {
		stps = append(stps, -float64(index)*sectionBeats)
	}
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

	synth := grains.NewPoly(s.cfg, s.voices).AddTo(p)

	msg := template.Template{"pitch": seq(pitches...)}
	for key, values := range s.params {
		msg[key] = seq(values...)
	}
	synth.ValueIn(templ.NewDurationTrigger(seq(durs...), s.amp, msg).
		ValueIn(timer.WithSteps(steps.QuickSteps(stps...)).AddTo(p)))

	if s.patch != nil {
		s.patch(p, synth)
	}

	return synth
}

func main() {
	psalm.PushConfig(psalm.Config{BPM: 90, PPQ: 240, SampleRate: 44100, BufferSize: 1024})

	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	load := func(path string) sndfile.SoundFiler[float32] {
		return sndfile.MustMipMapSoundFile[float32](path, 4)
	}

	// at returns the preset with the position moved to where the recording has sound
	at := func(cfg grains.Config, position float64) grains.Config {
		cfg.Position = position
		return cfg
	}

	// The recordings and the MIDI pitch they are played back at (the root note). Grain levels are
	// normalized to each sample's peak, so quiet and loud recordings play at a similar level
	choir := load("assets/sounds/various/mixkit-angelical-choir-654.wav")
	chant := load("assets/sounds/chants/Johannes.Hymnus.wav")
	keys := load("assets/sounds/various/Cymatics - Orchid KEYS Swirl (C).wav")
	sea := load("assets/sounds/various/mixkit-gentle-sea-wave-1204.wav")
	vocal := load("assets/sounds/various/Cymatics - Blurry Vocal - 80 BPM F Min.wav")

	const (
		A2 = 45.0
		C3 = 48.0
		E3 = 52.0
		G3 = 55.0
		A3 = 57.0
		C4 = 60.0
		D4 = 62.0
		E4 = 64.0
		F4 = 65.0
		G4 = 67.0
		A4 = 69.0
		C5 = 72.0
		E5 = 76.0
	)

	sectionSec := sectionBeats * 60.0 / psalm.BPM()

	sections := []*section{
		{
			name:   "Choir pad on a choir recording: Am F C G",
			cfg:    grains.ChoirPad(choir, 60),
			voices: 12, amp: 0.238, legato: 1.0,
			notes: chordNotes(4,
				[]float64{A3, C4, E4}, []float64{F4 - 12, A3, C4}, []float64{C4, E4, G4}, []float64{G3, 59, D4},
			),
		},
		{
			name:   "Frozen drone on a chant: an LFO on value input 1 slowly moves the frozen position",
			cfg:    grains.FrozenDrone(chant, 60),
			voices: 4, amp: 0.315, legato: 1.0,
			notes: chordNotes(8, []float64{A2, E3}, []float64{C3, G3}),
			patch: func(p *psalm.Psalm, synth psalm.Modular) {
				synth.ValueIn(lfo.New(true, 0.08, 0.2, 0.8, lfo.Sine).AddTo(p), 0, 1)
			},
		},
		{
			name:   "Glass keys on a keys sample: velocity rising, so the grains get denser",
			cfg:    at(grains.GlassKeys(keys, 60), 0.03), // the swirl sounds in its first part
			voices: 8, amp: 0.167, legato: 0.8,
			notes: notes(1, A3, C4, E4, A4, G4, E4, C4, E4, F4, A4, C5, A4, G4, E4, D4, E4),
			params: map[string][]float64{"velocity": {
				0.1, 0.15, 0.2, 0.25, 0.3, 0.35, 0.4, 0.5, 0.55, 0.6, 0.7, 0.75, 0.8, 0.9, 0.95, 1.0,
			}},
		},
		{
			name:   "Texture on sea waves: density stepping up from sparse droplets to a dense cloud",
			cfg:    at(grains.Texture(sea, 60), 0.4),
			voices: 4, amp: 0.436, legato: 0.95,
			notes:  notes(4, C4, C4, C4, C4),
			params: map[string][]float64{"density": {4, 12, 30, 80}},
		},
		{
			name:   "Stutter on a vocal: the grain size changes per note",
			cfg:    grains.Stutter(vocal, 65),
			voices: 4, amp: 0.857, legato: 0.9,
			notes:  notes(2, F4, F4, 63, 63, 60, 60, F4, F4),
			params: map[string][]float64{"grainSize": {30, 60, 100, 160, 60, 30, 100, 200}},
		},
		{
			name:   "Shimmer on the choir: the octave chance rises chord by chord",
			cfg:    grains.Shimmer(choir, 60),
			voices: 12, amp: 0.215, legato: 1.0,
			notes: chordNotes(4,
				[]float64{A3, C4, E4}, []float64{F4 - 12, A3, C4}, []float64{C4, E4, G4}, []float64{A3, C4, E4},
			),
			params: map[string][]float64{"octave": {0, 0.2, 0.4, 0.6}},
		},
		{
			name: "Scan on the chant: frozen, forwards, double speed and backwards",
			cfg: func() grains.Config {
				cfg := grains.ChoirPad(chant, 60)
				cfg.AmpEnv.AttackDuration = 200
				return cfg
			}(),
			voices: 4, amp: 0.317, legato: 1.0,
			notes:  notes(4, A3, A3, A3, A3),
			params: map[string][]float64{"scan": {0, 1, 2, -1}},
		},
		{
			name:   "Pluck on the keys sample: an arpeggio",
			cfg:    at(grains.Pluck(keys, 60), 0.02),
			voices: 8, amp: 0.0802, legato: 0.6,
			notes: notes(0.5,
				A3, C4, E4, A4, C5, A4, E4, C4, F4-12, A3, C4, F4, A4, F4, C4, A3,
				C4, E4, G4, C5, E5, C5, G4, E4, G3, 59, D4, G4, 71, G4, D4, 59),
			params: map[string][]float64{"velocity": {1, 0.6, 0.8, 0.6}},
		},
	}

	var mix []any
	for i, s := range sections {
		fmt.Printf("%6.1fs  %s\n", float64(i)*sectionSec, s.name)
		mix = append(mix, s.build(p, i, len(sections)), 0, 0)
	}
	fmt.Printf("%6.1fs  (tour loops)\n", float64(len(sections))*sectionSec)

	rcfg := fdnreverb.DefaultConfig()
	rcfg.Decay = 3.0
	rcfg.Mix = 0.25

	p.Out(fdnreverb.New(rcfg).AddTo(p).In(mix...))

	_ = p.Listen()
}
