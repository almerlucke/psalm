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
	"github.com/almerlucke/psalm/synths/pulsar"
)

// A tour of the pulsar synth: every section plays a preset and changes a setting note by note, so
// you hear what the formant, the duty, the windows and masking do, and how a pulse train at a low
// enough pitch turns into a rhythm. A parade of the other presets follows, two bars each. The tour
// loops, the schedule is printed at start

const (
	sectionBeats = 16.0 // 4 bars
	paradeBeats  = 8.0  // 2 bars
)

// seq plays the values in order and loops
func seq[T any](values ...T) *sequence.Sequence[T] {
	return sequence.NewLoop(values...)
}

// note is one or more pitches (a chord when more than one) with a length in beats, no pitches is a rest
type note struct {
	pitches []float64
	beats   float64
}

// notes builds single notes of equal length, a 0 pitch is a rest
func notes(beats float64, pitches ...float64) []note {
	ns := make([]note, len(pitches))
	for i, p := range pitches {
		ns[i] = note{[]float64{p}, beats}
		if p == 0 {
			ns[i].pitches = nil
		}
	}
	return ns
}

// phrase joins groups of notes
func phrase(groups ...[]note) []note {
	var ns []note
	for _, g := range groups {
		ns = append(ns, g...)
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
	name      string
	beats     float64 // length of the section, 0 is sectionBeats
	cfg       pulsar.Config
	voices    int
	amp       float64
	legato    float64
	frequency bool // the notes are frequencies in Hz instead of MIDI pitches
	notes     []note
	params    map[string][]float64
	patch     func(p *psalm.Psalm, synth psalm.Modular)
}

func (s *section) length() float64 {
	if s.beats == 0 {
		return sectionBeats
	}
	return s.beats
}

// build plays the notes of the section in its slot of the looping tour: start beats after the
// start of the tour, which lasts tourBeats
func (s *section) build(p *psalm.Psalm, start float64, tourBeats float64) psalm.Modular {
	var (
		stps    []any
		pitches [][]float64
		durs    []float64
		total   float64
		beatSec = 60.0 / float64(psalm.BPM())
	)
	if start > 0 {
		stps = append(stps, -start)
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
	if total != s.length() {
		panic(fmt.Sprintf("section %q lasts %v beats, expected %v", s.name, total, s.length()))
	}
	if after := tourBeats - start - total; after > 0 {
		stps = append(stps, -after)
	}

	synth := pulsar.NewPoly(s.cfg, s.voices).AddTo(p)

	key := "pitch"
	if s.frequency {
		key = "frequency"
	}
	msg := template.Template{key: seq(pitches...)}
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
	psalm.PushConfig(psalm.Config{BPM: 96, PPQ: 240, SampleRate: 44100, BufferSize: 1024})

	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	const (
		A1 = 33.0
		C2 = 36.0
		D2 = 38.0
		E2 = 40.0
		F2 = 41.0
		G2 = 43.0
		A2 = 45.0
		G3 = 55.0
		A3 = 57.0
		B3 = 59.0
		C4 = 60.0
		D4 = 62.0
		E4 = 64.0
		F4 = 65.0
		G4 = 67.0
		A4 = 69.0
		C5 = 72.0
		E5 = 76.0
	)

	beatSec := 60.0 / float64(psalm.BPM())

	sections := []*section{
		{
			name:   "Vowel lead: the formant changes per note, ah, oh, oo, eh, ee",
			cfg:    pulsar.VowelLeadPreset(),
			voices: 2, amp: 0.6, legato: 0.95,
			notes: notes(2, A3, C4, E4, D4, C4, B3, A3, 0),
			params: map[string][]float64{
				"formant":  {700, 450, 320, 550, 300, 700, 450, 700},
				"formant2": {1150, 800, 800, 1800, 2300, 1150, 800, 1150},
			},
		},
		{
			name:   "Formant pad: Am F C G, an LFO on value input 1 sweeps the formant",
			cfg:    pulsar.FormantPadPreset(),
			voices: 12, amp: 0.4, legato: 1.0,
			notes: chordNotes(4,
				[]float64{A3, C4, E4}, []float64{F4 - 12, A3, C4}, []float64{C4, E4, G4}, []float64{G3, B3, D4},
			),
			patch: func(p *psalm.Psalm, synth psalm.Modular) {
				synth.ValueIn(lfo.New(true, 0.06, 300, 1400, lfo.Sine).AddTo(p), 0, 1)
			},
		},
		{
			name:   "Buzz bass: the duty rises from thin clicks to full pulses",
			cfg:    pulsar.BuzzBassPreset(),
			voices: 2, amp: 0.38, legato: 0.7,
			notes: notes(1, A1, A1, A2, A1, C2, C2, D2, E2, F2, F2, A2, F2, G2, G2, E2, G2),
			params: map[string][]float64{"duty": {
				0.05, 0.08, 0.1, 0.12, 0.15, 0.2, 0.25, 0.3, 0.35, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0,
			}},
		},
		{
			name:   "Windows: the same glass melody with a Hann, Gaussian, percussive and rectangle window",
			cfg:    pulsar.GlassPreset(),
			voices: 8, amp: 0.55, legato: 0.6,
			notes:  notes(1, A4, E4, C5, E4, A4, E4, C5, E5, G4, D4, B3+12, D4, G4, D4, B3+12, D4),
			params: map[string][]float64{"window": {0, 0, 0, 0, 1, 1, 1, 1, 2, 2, 2, 2, 3, 3, 3, 3}},
		},
		{
			name:   "Rhythm to pitch: pulses at 3, 4, 6 and 8 Hz are a rhythm, at 30 Hz and up a tone",
			cfg:    pulsar.RhythmPreset(),
			voices: 2, amp: 0.6, legato: 1.0,
			frequency: true,
			notes:     notes(2, 3, 4, 6, 8, 12, 16, 30, 55),
		},
		{
			name:   "Crackle: masking rises, so the buzz breaks up into ever sparser crackles",
			cfg:    pulsar.CracklePreset(),
			voices: 2, amp: 0.65, legato: 1.0,
			frequency: true,
			notes:     notes(4, 80, 60, 45, 30),
			params:    map[string][]float64{"masking": {0.2, 0.5, 0.8, 0.95}},
		},
		{
			name:   "Laser: falling chirps in an arpeggio",
			cfg:    pulsar.LaserPreset(),
			voices: 6, amp: 0.7, legato: 0.5,
			notes: notes(0.5,
				A3, C4, E4, A4, 0, E4, C4, A3, F4-12, A3, C4, F4, 0, C4, A3, 0,
				C4, E4, G4, C5, 0, G4, E4, C4, G3, B3, D4, G4, 0, D4, B3, 0),
		},
	}

	// the parade of the other presets, two bars each with a phrase that suits the preset
	parade := []*section{
		{
			name:   "Parade, choir: an ooh choir on Am and F",
			cfg:    pulsar.ChoirPreset(),
			voices: 8, amp: 0.4, legato: 1.0,
			notes: chordNotes(4, []float64{A3, C4, E4}, []float64{F4 - 12, A3, C4}),
		},
		{
			name:   "Parade, strings: a slow bowed line",
			cfg:    pulsar.StringsPreset(),
			voices: 4, amp: 0.45, legato: 1.0,
			notes: notes(2, E4, D4, C4, B3),
		},
		{
			name:   "Parade, brass: a fanfare",
			cfg:    pulsar.BrassPreset(),
			voices: 4, amp: 0.5, legato: 0.85,
			notes: phrase(notes(0.5, G3, C4, E4, G4), notes(1, G4, 0, E4, G4, C5, 0)),
		},
		{
			name:   "Parade, mallet: a marimba figure",
			cfg:    pulsar.MalletPreset(),
			voices: 8, amp: 0.5, legato: 0.8,
			notes: notes(0.5, A4, E4, C5, A4, G4, E4, D4, E4, C4, E4, G4, C5, B3+12, G4, D4, G4),
		},
		{
			name:   "Parade, sub bass: a deep, round line",
			cfg:    pulsar.SubBassPreset(),
			voices: 2, amp: 0.8, legato: 0.8,
			notes: notes(1, A1, 0, A1, C2, D2, 0, E2, G2),
		},
		{
			name:   "Parade, wobble bass: the formant LFO talks",
			cfg:    pulsar.WobbleBassPreset(),
			voices: 2, amp: 0.85, legato: 0.95,
			notes: notes(2, A1, C2, D2, E2),
		},
		{
			name:   "Parade, insects: a summer night (45 Hz)",
			cfg:    pulsar.InsectsPreset(),
			voices: 2, amp: 0.7, legato: 1.0,
			frequency: true,
			notes:     notes(8, 45),
		},
		{
			name:   "Parade, raindrops: rain on a window (12 Hz)",
			cfg:    pulsar.RaindropsPreset(),
			voices: 2, amp: 0.8, legato: 1.0,
			frequency: true,
			notes:     notes(8, 12),
		},
		{
			name:   "Parade, motor: revving up and down",
			cfg:    pulsar.MotorPreset(),
			voices: 1, amp: 0.7, legato: 1.0,
			frequency: true,
			notes:     notes(2, 18, 30, 38, 22),
		},
	}
	for _, s := range parade {
		s.beats = paradeBeats
	}
	sections = append(sections, parade...)

	tourBeats := 0.0
	for _, s := range sections {
		tourBeats += s.length()
	}

	var mix []any
	start := 0.0
	for _, s := range sections {
		fmt.Printf("%6.1fs  %s\n", start*beatSec, s.name)
		mix = append(mix, s.build(p, start, tourBeats), 0, 0)
		start += s.length()
	}
	fmt.Printf("%6.1fs  (tour loops)\n", tourBeats*beatSec)

	rcfg := fdnreverb.DefaultConfig()
	rcfg.Decay = 2.5
	rcfg.Mix = 0.2

	p.Out(fdnreverb.New(rcfg).AddTo(p).In(mix...))

	_ = p.Listen()
}
