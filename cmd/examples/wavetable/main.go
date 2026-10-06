package main

import (
	"fmt"
	"path/filepath"

	"github.com/almerlucke/genny/sequence"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/modules/controls/templ"
	"github.com/almerlucke/psalm/modules/controls/timer"
	"github.com/almerlucke/psalm/modules/controls/timer/steps"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/oscillators/lfo"
	"github.com/almerlucke/psalm/synths/wavetable"
)

// A tour of the wavetable synth on the tables in assets/wavetables2048: every section plays a
// preset on a table that suits it and changes a setting note by note (the position, its envelope
// and LFO, unison, the filter). A parade of the other presets follows, two bars each. The tour
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
	name   string
	beats  float64 // length of the section, 0 is sectionBeats
	cfg    wavetable.Config
	voices int
	amp    float64
	legato float64
	notes  []note
	params map[string][]float64
	patch  func(p *psalm.Psalm, synth psalm.Modular)
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
		beatSec = 60.0 / psalm.BPM()
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

	synth := wavetable.NewPoly(s.cfg, s.voices).AddTo(p)

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
	psalm.PushConfig(psalm.Config{BPM: 100, PPQ: 240, SampleRate: 44100, BufferSize: 1024})

	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	// tables are loaded once and shared
	tables := map[string]*wavetable.Table{}
	load := func(name string) *wavetable.Table {
		if t, ok := tables[name]; ok {
			return t
		}
		t := wavetable.MustLoad(filepath.Join("assets/wavetables2048", name+".wav"), 2048)
		tables[name] = t
		return t
	}

	// with sets a setting on a preset
	with := func(cfg wavetable.Config, f func(cfg *wavetable.Config)) wavetable.Config {
		f(&cfg)
		return cfg
	}

	const (
		A1 = 33.0
		C2 = 36.0
		D2 = 38.0
		E2 = 40.0
		F2 = 41.0
		G2 = 43.0
		A2 = 45.0
		E3 = 52.0
		G3 = 55.0
		A3 = 57.0
		B3 = 59.0
		C4 = 60.0
		D4 = 62.0
		E4 = 64.0
		F4 = 65.0
		G4 = 67.0
		A4 = 69.0
		B4 = 71.0
		C5 = 72.0
		D5 = 74.0
		E5 = 76.0
	)

	am := []float64{A3, C4, E4}
	fm := []float64{F4 - 12, A3, C4}
	cm := []float64{C4, E4, G4}
	gm := []float64{G3, B3, D4}

	beatSec := 60.0 / psalm.BPM()

	sections := []*section{
		{
			name:   "Pad on DN Harm: chords drifting slowly through 256 dark harmonic frames",
			cfg:    wavetable.PadPreset(load("DN Harm")),
			voices: 16, amp: 0.6, legato: 1.0,
			notes: chordNotes(4, am, fm, cm, gm),
		},
		{
			name:   "Sweep on OPFoldFMFB: the position envelope travels from a soft fold to a harsh one",
			cfg:    wavetable.SweepPreset(load("OPFoldFMFB")),
			voices: 8, amp: 0.6, legato: 0.75,
			notes: chordNotes(8, []float64{A2, E3, A3}, []float64{F2, C4 - 12, A3}),
		},
		{
			name:   "Lead on VS Morph3: every note at another position",
			cfg:    wavetable.LeadPreset(load("VS Morph3")),
			voices: 2, amp: 0.7, legato: 0.95,
			notes:  phrase(notes(1, E4, G4, A4, 0, A4, C5, B4, G4), notes(2, A4, E4), notes(1, D4, E4, 0, 0)),
			params: map[string][]float64{"position": {0, 0.1, 0.2, 0.3, 0.45, 0.6, 0.75, 0.9, 1, 0.5, 0.2, 0}},
		},
		{
			name:   "Pluck on Terra Bend+PD: the position envelope reaches further up the table every bar",
			cfg:    wavetable.PluckPreset(load("Terra Bend+PD")),
			voices: 8, amp: 1.0, legato: 0.8,
			notes: notes(0.5,
				A3, C4, E4, A4, E4, C4, A4, E4, F4-12, A3, C4, F4, C4, A3, F4, C4,
				C4, E4, G4, C5, G4, E4, C5, G4, G3, B3, D4, G4, D4, B3, G4, D4),
			params: map[string][]float64{"positionEnvAmount": {
				0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.1, 0.35, 0.35, 0.35, 0.35, 0.35, 0.35, 0.35, 0.35,
				0.6, 0.6, 0.6, 0.6, 0.6, 0.6, 0.6, 0.6, 0.85, 0.85, 0.85, 0.85, 0.85, 0.85, 0.85, 0.85,
			}},
		},
		{
			name:   "Bass on MS 20 Pulse2: the cutoff opens note by note",
			cfg:    wavetable.BassPreset(load("MS 20 Pulse2 PS1")),
			voices: 2, amp: 0.9, legato: 0.7,
			notes: notes(1, A1, A1, A2, A1, C2, C2, D2, E2, F2, F2, A2, F2, G2, G2, E2, G2),
			params: map[string][]float64{"cutoff": {
				150, 180, 220, 260, 300, 350, 400, 470, 550, 650, 750, 850, 1000, 1200, 1400, 1700,
			}},
		},
		{
			name:   "Growl on MSelfFB02: the position LFO speeds up, 1, 2, 4 and 8 Hz",
			cfg:    wavetable.GrowlPreset(load("MSelfFB02")),
			voices: 2, amp: 1.0, legato: 0.95,
			notes:  notes(4, A1, C2, D2, E2),
			params: map[string][]float64{"positionLFORate": {1, 2, 4, 8}},
		},
		{
			name:   "Keys on Melda IcarusBell: a bell melody",
			cfg:    wavetable.KeysPreset(load("Melda CustumWave3 IcarusBell")),
			voices: 8, amp: 0.8, legato: 0.9,
			notes: phrase(notes(1, A4, E5, C5, A4, B4, D5, B4, G4), notes(1, A4, C5, E5, D5), notes(2, C5, A4)),
		},
		{
			name:   "Anthem on Add Synth3 Icarus: unison grows from 1 to 3, 5 and 7 oscillators",
			cfg:    wavetable.AnthemPreset(load("Add Synth3 Icarus")),
			voices: 16, amp: 0.7, legato: 0.95,
			notes:  chordNotes(4, am, fm, cm, gm),
			params: map[string][]float64{"unison": {1, 3, 5, 7}},
		},
		{
			name: "Scan on FMAdditive1: an LFO on value input 1 sweeps the position through 127 frames",
			cfg: with(wavetable.PadPreset(load("FMAdditive1")), func(cfg *wavetable.Config) {
				cfg.PositionLFODepth = 0
				cfg.AmpEnv.AttackDuration = 300
			}),
			voices: 8, amp: 0.7, legato: 1.0,
			notes: chordNotes(8, []float64{A2, E3, A3, C4}, []float64{F2, C4 - 12, A3, E4}),
			patch: func(p *psalm.Psalm, synth psalm.Modular) {
				synth.ValueIn(lfo.New(true, 0.125, 0, 1, lfo.Triangle).AddTo(p), 0, 1)
			},
		},
		{
			name: "Hits on KaossDrum2: short notes, every one at another position of a drum table",
			cfg: with(wavetable.KeysPreset(load("KaossDrum2")), func(cfg *wavetable.Config) {
				cfg.PositionEnvAmount = 0
				cfg.AmpEnv.DecayDuration = 180
			}),
			voices: 6, amp: 1.0, legato: 0.5,
			notes: notes(0.5,
				A2, 0, A2, A3, 0, A2, E3, 0, A2, 0, A2, A3, 0, G3, E3, 50,
				A2, 0, A2, A3, 0, A2, E3, 0, A2, A3, A2, A3, G3, E3, 50, C4-12),
			params: map[string][]float64{"position": {
				0.05, 0.4, 0.8, 0.2, 0.6, 0.95, 0.3, 0.7, 0.05, 0.5, 0.85, 0.15,
				0.05, 0.4, 0.8, 0.2, 0.6, 0.95, 0.3, 0.7, 0.05, 0.5, 0.85, 0.15, 0.65, 0.35, 0.9,
			}},
		},
	}

	// the parade of the other presets, two bars each on a table that suits them
	rest := func(beats float64) note { return note{nil, beats} }
	parade := []*section{
		{
			name:   "Parade, strings on DN Harm: a slow bowed Am and F",
			cfg:    wavetable.StringsPreset(load("DN Harm")),
			voices: 8, amp: 0.8, legato: 1.0,
			notes: chordNotes(4, am, fm),
		},
		{
			name:   "Parade, vox on VS Morph3: a sung line",
			cfg:    wavetable.VoxPreset(load("VS Morph3")),
			voices: 2, amp: 0.7, legato: 0.95,
			notes: phrase(notes(1, E4, G4, A4, G4, E4, D4), notes(2, E4)),
		},
		{
			name:   "Parade, brass on OPFoldFMFB: a fanfare",
			cfg:    wavetable.BrassPreset(load("OPFoldFMFB")),
			voices: 8, amp: 0.8, legato: 0.85,
			notes: phrase(chordNotes(1.5, cm), chordNotes(0.5, cm), chordNotes(2, fm), chordNotes(4, gm)),
		},
		{
			name:   "Parade, organ on FMAdditive1: Am F C G",
			cfg:    wavetable.OrganPreset(load("FMAdditive1")),
			voices: 8, amp: 0.65, legato: 0.95,
			notes: chordNotes(2, am, fm, cm, gm),
		},
		{
			name:   "Parade, reese on MSelfFB02: a beating, phasing bass",
			cfg:    wavetable.ReesePreset(load("MSelfFB02")),
			voices: 2, amp: 0.9, legato: 0.95,
			notes: notes(2, A1, C2, D2, A1),
		},
		{
			name:   "Parade, acid on MS 20 Pulse2: a squelching line",
			cfg:    wavetable.AcidPreset(load("MS 20 Pulse2 PS1")),
			voices: 1, amp: 0.9, legato: 0.9,
			notes: notes(0.5, A1, A2, A1, C2, A1, A2, E2, A1, G2, A1, A2, C2, A1, E2, D2, C2),
		},
		{
			name:   "Parade, stab on Terra Bend+PD: offbeat chord stabs",
			cfg:    wavetable.StabPreset(load("Terra Bend+PD")),
			voices: 12, amp: 0.8, legato: 0.8,
			notes: []note{
				{am, 0.75}, rest(0.75), {am, 0.5}, rest(1),
				{fm, 0.75}, rest(0.75), {fm, 0.5}, rest(1),
				{cm, 0.75}, rest(1.25),
			},
		},
		{
			name:   "Parade, drone on Add Synth3 Icarus: a slowly breathing fifth",
			cfg:    wavetable.DronePreset(load("Add Synth3 Icarus")),
			voices: 4, amp: 0.9, legato: 1.0,
			notes: chordNotes(8, []float64{A1, E2, A2}),
		},
		{
			name:   "Parade, glass on Add Bell: an airy bell melody",
			cfg:    wavetable.GlassPreset(load("Add Bell")),
			voices: 8, amp: 0.8, legato: 0.9,
			notes: phrase(notes(1, E5, B4, A4, E5, D5, B4, A4), []note{rest(1)}),
		},
		{
			name:   "Parade, riser on OPFoldFMFB: hold a chord and ride it up",
			cfg:    wavetable.RiserPreset(load("OPFoldFMFB")),
			voices: 4, amp: 0.9, legato: 1.0,
			notes: chordNotes(8, []float64{A2, E3, A3}),
		},
	}
	for _, s := range parade {
		s.beats = paradeBeats
	}
	sections = append(sections, parade...)

	start := 0.0
	tourBeats := 0.0
	for _, s := range sections {
		tourBeats += s.length()
	}
	var mix []any
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
