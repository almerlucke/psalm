package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/almerlucke/genny/sequence"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/modules/controls/templ"
	"github.com/almerlucke/psalm/modules/controls/timer"
	"github.com/almerlucke/psalm/modules/controls/timer/steps"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
)

// Generated wavetables: every section plays a table made by one of the generators in
// synths/wavetable/gen, the position rising note by note from the start to the end of the table, so
// you hear what the generator sweeps, the first eight for four bars, the others for two. The tour
// loops, the schedule is printed at start.
//
// Pass a directory to also save the tables there as WAV files of 2048 sample frames:
//
//	go run ./cmd/examples/wavegen tables

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

// mustSound slices a recording into a table
func mustSound(path string, numFrames int) *wavetable.Table {
	t, err := gen.LoadSound(path, numFrames)
	if err != nil {
		panic(err)
	}
	return t
}

// rising returns n positions from 0 to 1
func rising(n int) []float64 {
	ps := make([]float64, n)
	for i := range ps {
		ps[i] = float64(i) / float64(n-1)
	}
	return ps
}

func main() {
	psalm.PushConfig(psalm.Config{BPM: 100, PPQ: 240, SampleRate: 44100, BufferSize: 1024})

	tables := []struct {
		name  string
		table *wavetable.Table
	}{
		{"brightness", gen.Brightness(64)},
		{"pwm", gen.PWM(64, 0.03)},
		{"sync", gen.Sync(64, 8)},
		{"fm", gen.FM(64, 3, 6)},
		{"fold", gen.Fold(64, 6, 0.2)},
		{"vowels", gen.Vowels(64, 130)},
		{"drawbars", gen.Drawbars(64, 16)},
		{"random", gen.Random(64, 6, 48, 7)},
		{"filter", gen.Filter(64, 6)},
		{"resonance", gen.Resonance(64, 12)},
		{"oddeven", gen.OddEven(32)},
		{"comb", gen.Comb(64, 0.5)},
		{"saturate", gen.Saturate(32, 10)},
		{"quantize", gen.Quantize(64, 3)},
		{"steps", gen.Steps(64, 6)},
		{"crossfade", gen.Crossfade(64, gen.Sine, gen.Triangle, gen.Saw, gen.Square)},
		{"choir", mustSound("assets/sounds/various/mixkit-angelical-choir-654.wav", 64)},
		{"vocal", mustSound("assets/sounds/various/Cymatics - Blurry Vocal - 80 BPM F Min.wav", 64)},
	}
	table := map[string]*wavetable.Table{}
	for _, t := range tables {
		table[t.name] = t.table
	}

	if len(os.Args) > 1 {
		dir := os.Args[1]
		if err := os.MkdirAll(dir, 0o755); err != nil {
			panic(err)
		}
		for _, t := range tables {
			path := filepath.Join(dir, t.name+".wav")
			if err := t.table.Save(path); err != nil {
				panic(err)
			}
			fmt.Printf("saved %s\n", path)
		}
	}

	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

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
		C3 = 48.0
		D3 = 50.0
		E3 = 52.0
		F3 = 53.0
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
			name:   "Brightness: a lead from a sine to a full sawtooth",
			cfg:    with(wavetable.LeadPreset(table["brightness"]), func(c *wavetable.Config) { c.Filter = wavetable.FilterOff }),
			voices: 2, amp: 0.884, legato: 0.9,
			notes:  notes(1, A3, C4, E4, A4, G4, E4, D4, E4, A3, C4, E4, A4, B4, G4, E4, 0),
			params: map[string][]float64{"position": rising(15)},
		},
		{
			name:   "PWM: organ chords, the pulse narrowing from a square to 3 percent",
			cfg:    wavetable.OrganPreset(table["pwm"]),
			voices: 8, amp: 0.559, legato: 0.95,
			notes:  chordNotes(2, am, am, fm, fm, cm, cm, gm, gm),
			params: map[string][]float64{"position": rising(8)},
		},
		{
			name:   "Sync: a lead tearing open as the synced sawtooth rises to 8 times",
			cfg:    wavetable.LeadPreset(table["sync"]),
			voices: 2, amp: 0.884, legato: 0.9,
			notes:  phrase(notes(0.5, A3, C4, E4, A4, E4, C4, A3, C4), notes(1, D4, E4, G4, E4), notes(0.5, A3, C4, E4, A4, C5, A4, E4, C4), notes(4, A3)),
			params: map[string][]float64{"position": rising(21)},
		},
		{
			name:   "FM: bells, the modulation index rising from a sine to metal (ratio 3)",
			cfg:    wavetable.KeysPreset(table["fm"]),
			voices: 8, amp: 0.656, legato: 0.9,
			notes:  phrase(notes(1, E5, B4, A4, E4, A4, B4, C5, B4, A4, E4, D4, E4), notes(2, A4, 0)),
			params: map[string][]float64{"position": rising(13)},
		},
		{
			name:   "Fold: a bass folding harder and harder (bias 0.2 for even harmonics)",
			cfg:    with(wavetable.BassPreset(table["fold"]), func(c *wavetable.Config) { c.Cutoff = 1500; c.FilterEnvAmount = 1.5 }),
			voices: 2, amp: 1.12, legato: 0.8,
			notes:  notes(1, A1, A1, A2, A1, C2, C2, D2, E2, F2, F2, A2, F2, G2, G2, E2, G2),
			params: map[string][]float64{"position": rising(16)},
		},
		{
			name:   "Vowels: a voice around 130 Hz singing a, e, i, o, u",
			cfg:    with(wavetable.LeadPreset(table["vowels"]), func(c *wavetable.Config) { c.Filter = wavetable.FilterOff; c.Unison = 1 }),
			voices: 2, amp: 1.26, legato: 0.95,
			notes:  phrase(notes(3, C3, E3, G3, F3, E3), notes(1, 0)),
			params: map[string][]float64{"position": {0, 0.25, 0.5, 0.75, 1}},
		},
		{
			name:   "Drawbars: organ chords opening harmonic by harmonic up to 16",
			cfg:    wavetable.OrganPreset(table["drawbars"]),
			voices: 8, amp: 0.798, legato: 0.95,
			notes:  chordNotes(2, am, am, fm, fm, cm, cm, gm, gm),
			params: map[string][]float64{"position": rising(8)},
		},
		{
			name:   "Random (seed 7): a pad crossfading through six random spectra",
			cfg:    with(wavetable.PadPreset(table["random"]), func(c *wavetable.Config) { c.PositionLFODepth = 0.1 }),
			voices: 16, amp: 0.586, legato: 1.0,
			notes:  chordNotes(4, am, fm, cm, gm),
			params: map[string][]float64{"position": {0.1, 0.35, 0.65, 0.9}},
		},
	}

	// two bars for each of the other generators and two tables sliced from recordings
	more := []*section{
		{
			name:   "Filter: a resonant lowpass opening over a sawtooth (q 6)",
			cfg:    with(wavetable.BassPreset(table["filter"]), func(c *wavetable.Config) { c.Filter = wavetable.FilterOff }),
			voices: 2, amp: 1.12, legato: 0.8,
			notes:  notes(0.5, A1, A2, A1, C2, A1, A2, E2, A1, G2, A1, A2, C2, A1, E2, D2, C2),
			params: map[string][]float64{"position": rising(16)},
		},
		{
			name:   "Resonance: the phase distortion resonance rising to 12 times",
			cfg:    with(wavetable.LeadPreset(table["resonance"]), func(c *wavetable.Config) { c.Filter = wavetable.FilterOff }),
			voices: 2, amp: 0.884, legato: 0.9,
			notes:  notes(1, A3, C4, E4, G4, A4, G4, E4, D4),
			params: map[string][]float64{"position": rising(8)},
		},
		{
			name:   "OddEven: organ chords from odd harmonics only to even harmonics only",
			cfg:    wavetable.OrganPreset(table["oddeven"]),
			voices: 8, amp: 0.665, legato: 0.95,
			notes:  chordNotes(2, am, fm, cm, gm),
			params: map[string][]float64{"position": rising(4)},
		},
		{
			name:   "Comb: a plucked arpeggio, the comb delay growing to half a cycle (an octave up)",
			cfg:    wavetable.PluckPreset(table["comb"]),
			voices: 8, amp: 0.724, legato: 0.8,
			notes:  notes(0.5, A3, C4, E4, A4, E4, C4, A4, E4, F4-12, A3, C4, F4, C4, A3, F4, C4),
			params: map[string][]float64{"positionEnvAmount": {0}, "position": rising(16)},
		},
		{
			name:   "Saturate: a bass driven from a sine into soft clipping",
			cfg:    with(wavetable.BassPreset(table["saturate"]), func(c *wavetable.Config) { c.Filter = wavetable.FilterOff }),
			voices: 2, amp: 0.98, legato: 0.8,
			notes:  notes(1, A1, A1, C2, C2, D2, D2, E2, G2),
			params: map[string][]float64{"position": rising(8)},
		},
		{
			name:   "Quantize: keys from 8 bits down to 3 levels",
			cfg:    wavetable.KeysPreset(table["quantize"]),
			voices: 8, amp: 0.656, legato: 0.9,
			notes:  notes(1, E5, B4, A4, E4, A4, B4, C5, B4),
			params: map[string][]float64{"positionEnvAmount": {0}, "position": rising(8)},
		},
		{
			name:   "Steps: a lead stepping from 256 down to 6 steps per cycle",
			cfg:    with(wavetable.LeadPreset(table["steps"]), func(c *wavetable.Config) { c.Filter = wavetable.FilterOff }),
			voices: 2, amp: 0.758, legato: 0.9,
			notes:  notes(0.5, A3, C4, E4, A4, E4, C4, A3, C4, D4, E4, G4, E4, D4, C4, B3, C4),
			params: map[string][]float64{"position": rising(16)},
		},
		{
			name:   "Crossfade: sine, triangle, sawtooth, square",
			cfg:    with(wavetable.LeadPreset(table["crossfade"]), func(c *wavetable.Config) { c.Filter = wavetable.FilterOff }),
			voices: 2, amp: 0.758, legato: 0.9,
			notes:  notes(1, A3, C4, E4, A4, G4, E4, D4, E4),
			params: map[string][]float64{"position": rising(8)},
		},
		{
			name: "Sliced choir recording: low chords on slices of the choir, no lowpass (slices are bright)",
			cfg: with(wavetable.PadPreset(table["choir"]), func(c *wavetable.Config) {
				c.Filter = wavetable.FilterOff
				c.Level = 1.0
			}),
			voices: 16, amp: 0.7, legato: 1.0,
			notes:  chordNotes(4, []float64{A2, C3, E3}, []float64{F2, A2, C3}),
			params: map[string][]float64{"position": {0.3, 0.7}},
		},
		{
			name:   "Sliced vocal recording: a low line moving through the phrase",
			cfg:    with(wavetable.LeadPreset(table["vocal"]), func(c *wavetable.Config) { c.Filter = wavetable.FilterOff; c.Level = 1.0 }),
			voices: 2, amp: 0.7, legato: 0.95,
			notes:  notes(1, A2, C3, E3, D3, C3, B3-12, A2, 0),
			params: map[string][]float64{"position": rising(7)},
		},
	}
	for _, s := range more {
		s.beats = 8
	}
	sections = append(sections, more...)

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
