package main

import (
	"fmt"

	"github.com/almerlucke/genny/bucket"
	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/blocks/oscillators/blosc"
	"github.com/almerlucke/psalm/modules/controls/templ"
	"github.com/almerlucke/psalm/modules/controls/timer"
	"github.com/almerlucke/psalm/modules/controls/timer/steps"
	"github.com/almerlucke/psalm/modules/effects/freeverb"
	"github.com/almerlucke/psalm/modules/effects/pingpong"
	"github.com/almerlucke/psalm/modules/oscillators/lfo"
	"github.com/almerlucke/psalm/synths/vco"
	"github.com/almerlucke/psalm/tools/timing"
)

// A guided tour through all features of the vco voice: every section plays for sectionBeats,
// one after the other, and the whole tour loops. The schedule is printed at start.

const sectionBeats = 16.0 // 4 bars

func seq[T any](values ...T) *bucket.Bucket[T] {
	return bucket.NewLoop(values...).SetMode(bucket.Indexed).Loop(true)
}

// note is a pitch (MIDI, 0 = rest) with a length in beats
type note struct {
	pitch float64
	beats float64
}

const rest = 0.0

// notes builds notes of equal length from pitches
func notes(beats float64, pitches ...float64) []note {
	ns := make([]note, len(pitches))
	for i, p := range pitches {
		ns[i] = note{p, beats}
	}
	return ns
}

// section demonstrates a feature: one or more layers of notes (for chords) played by a synth,
// params are sent with every note and cycle per note, so they can step through settings
type section struct {
	name   string
	cfg    vco.Config
	voices int
	amp    float64
	legato float64
	layers [][]note
	params map[string][]float64
	patch  func(p *psalm.Psalm, synth psalm.Modular) // optional extra modulation
}

func beatsOf(ns []note) float64 {
	total := 0.0
	for _, n := range ns {
		total += n.beats
	}
	return total
}

// schedule surrounds the notes with rests, so the layer only plays during its slot in the tour
func schedule(ns []note, index int, numSections int) []note {
	var out []note
	if index > 0 {
		out = append(out, note{rest, float64(index) * sectionBeats})
	}
	out = append(out, ns...)
	if after := numSections - 1 - index; after > 0 {
		out = append(out, note{rest, float64(after) * sectionBeats})
	}
	return out
}

// play converts notes into a step timer plus matching pitch and duration sequences
func play(legato float64, ns []note) (*timer.Timer, *bucket.Bucket[float64], *bucket.Bucket[float64]) {
	var (
		stps    []any
		pitches []float64
		durs    []float64
		beatSec = 60.0 / float64(psalm.BPM())
	)

	for _, n := range ns {
		if n.pitch == rest {
			stps = append(stps, -n.beats)
			continue
		}

		stps = append(stps, n.beats)
		pitches = append(pitches, n.pitch)
		durs = append(durs, n.beats*beatSec*legato)
	}

	return timer.WithSteps(steps.QuickSteps(stps...)), seq(pitches...), seq(durs...)
}

func (s *section) build(p *psalm.Psalm, index int, numSections int) psalm.Modular {
	synth := vco.NewPoly(s.cfg, s.voices).AddTo(p)

	for _, layer := range s.layers {
		if beatsOf(layer) != sectionBeats {
			panic(fmt.Sprintf("section %q layer lasts %v beats, expected %v", s.name, beatsOf(layer), sectionBeats))
		}

		t, pitches, durs := play(s.legato, schedule(layer, index, numSections))

		msg := template.Template{"pitch": pitches}
		for key, values := range s.params {
			msg[key] = seq(values...)
		}

		synth.ValueIn(templ.NewDurationTrigger(durs, s.amp, msg).ValueIn(t.AddTo(p)))
	}

	if s.patch != nil {
		s.patch(p, synth)
	}

	return synth
}

func main() {
	psalm.PushConfig(psalm.Config{
		BPM:        120,
		PPQ:        240,
		SampleRate: 44100,
		BufferSize: 1024,
	})

	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	const (
		A1 = 33.0
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
		B4 = 71.0
		C5 = 72.0
		D5 = 74.0
		E5 = 76.0
		G5 = 79.0
		A5 = 81.0
	)

	// A plain voice with osc 2 off and an open filter, used to hear the raw oscillator features
	plain := func() vco.Config {
		cfg := vco.DefaultConfig()
		cfg.Osc2Level = 0.0
		cfg.Cutoff = 8000.0
		cfg.Resonance = 0.0
		cfg.FilterEnvAmount = 0.0
		cfg.AmpEnv = *adsr.NewSetting(1.0, 5.0, 0.8, 100.0, 1.0, 150.0)
		return cfg
	}

	sections := []*section{
		{
			name:   "Oscillator waveforms (sine, triangle, square, rectangle, saw, ramp, modified square, trapezoid)",
			cfg:    plain(),
			voices: 2, amp: 0.3, legato: 0.85,
			layers: [][]note{notes(2, A3, E4, A3, E4, A3, E4, A3, E4)},
			params: map[string][]float64{"osc1Waveform": {
				float64(blosc.Sine), float64(blosc.Triangle), float64(blosc.Square), float64(blosc.Rectangle),
				float64(blosc.Sawtooth), float64(blosc.Ramp), float64(blosc.ModifiedSquare), float64(blosc.TrapezoidVariable),
			}},
		},
		{
			name: "Osc 2 detune and octave: unison, slow beating, chorus, fifth, octave",
			cfg: func() vco.Config {
				cfg := plain()
				cfg.Osc1Level, cfg.Osc2Level = 0.4, 0.4
				cfg.Cutoff = 3000.0
				return cfg
			}(),
			voices: 2, amp: 0.3, legato: 0.9,
			layers: [][]note{notes(2, A3, A3, A3, A3, A3, A3, A3, A3)},
			params: map[string][]float64{"osc2Detune": {0, 0.03, 0.08, 0.15, 0.3, 7, 12, 12.1}},
		},
		{
			name: "Pulse width: narrowing pulse, then LFO pulse width modulation",
			cfg: func() vco.Config {
				cfg := plain()
				cfg.Osc1Waveform = blosc.Rectangle
				cfg.LFO1 = vco.LFOConfig{Waveform: vco.LFOTriangle, Rate: 2.0}
				return cfg
			}(),
			voices: 2, amp: 0.3, legato: 0.9,
			layers: [][]note{notes(2, A3, A3, A3, A3, A3, C4, D4, E4)},
			params: map[string][]float64{
				"pulseWidth":     {0.5, 0.3, 0.15, 0.05, 0.5, 0.5, 0.5, 0.5},
				"lfo1PulseWidth": {0, 0, 0, 0, 0.4, 0.4, 0.4, 0.4},
			},
		},
		{
			name: "Sub oscillator and noise: plain, + sub, + white noise, + pink noise",
			cfg: func() vco.Config {
				cfg := vco.BassPreset()
				cfg.Osc2Level = 0.0
				return cfg
			}(),
			voices: 2, amp: 0.3, legato: 0.6,
			layers: [][]note{notes(1, A2, A2, E3, A2, A2, A2, G3, A2, A2, A2, E3, A2, A2, A2, G3, A2)},
			params: map[string][]float64{
				"subLevel":   {0, 0, 0, 0, 0.6, 0.6, 0.6, 0.6, 0.6, 0.6, 0.6, 0.6, 0.6, 0.6, 0.6, 0.6},
				"noiseLevel": {0, 0, 0, 0, 0, 0, 0, 0, 0.3, 0.3, 0.3, 0.3, 0.3, 0.3, 0.3, 0.3},
				"noiseType": {0, 0, 0, 0, 0, 0, 0, 0, float64(vco.WhiteNoise), float64(vco.WhiteNoise),
					float64(vco.WhiteNoise), float64(vco.WhiteNoise), float64(vco.PinkNoise), float64(vco.PinkNoise),
					float64(vco.PinkNoise), float64(vco.PinkNoise)},
			},
		},
		{
			name:   "Ring modulation (ringBell preset)",
			cfg:    vco.RingBellPreset(),
			voices: 8, amp: 0.3, legato: 0.5,
			layers: [][]note{notes(1, A4, C5, E5, G5, A5, G5, E5, C5, A4, C5, D5, E5, G5, E5, D5, C5)},
		},
		{
			name:   "Hard sync (syncLead preset): sweeping the osc 2 pitch against osc 1",
			cfg:    vco.SyncLeadPreset(),
			voices: 2, amp: 0.3, legato: 0.9,
			layers: [][]note{notes(2, A3, A3, A3, A3, A3, A3, A3, A3)},
			params: map[string][]float64{"osc2Detune": {0, 4, 7, 12, 17, 24, 19, 12}},
		},
		{
			name: "FM: osc 1 modulating osc 2 with growing amount",
			cfg: func() vco.Config {
				cfg := plain()
				cfg.Osc1Waveform, cfg.Osc2Waveform = blosc.Sine, blosc.Sine
				cfg.Osc1Level, cfg.Osc2Level = 0.0, 0.5
				cfg.Osc2Detune = 0.0
				return cfg
			}(),
			voices: 2, amp: 0.35, legato: 0.9,
			layers: [][]note{notes(2, A3, A3, A3, A3, A3, A3, A3, A3)},
			params: map[string][]float64{"fmAmount": {0, 0.5, 1, 2, 4, 8, 4, 1}},
		},
		{
			name: "Filter modes (lowpass, bandpass, highpass) and resonance, cutoff swept by an external LFO",
			cfg: func() vco.Config {
				cfg := vco.DefaultConfig()
				cfg.FilterEnvAmount = 0.0
				cfg.AmpEnv = *adsr.NewSetting(1.0, 5.0, 0.8, 100.0, 1.0, 150.0)
				return cfg
			}(),
			voices: 2, amp: 0.22, legato: 0.95,
			layers: [][]note{notes(2, A2, A2, A2, A2, A2, A2, A2, A2)},
			params: map[string][]float64{
				"filterMode": {float64(vco.LowPass), float64(vco.LowPass), float64(vco.BandPass), float64(vco.BandPass),
					float64(vco.HighPass), float64(vco.HighPass), float64(vco.LowPass), float64(vco.LowPass)},
				"resonance": {0.2, 0.85, 0.2, 0.85, 0.2, 0.85, 0.2, 0.9},
			},
			patch: func(p *psalm.Psalm, synth psalm.Modular) {
				synth.ValueIn(lfo.New(true, 1.0, 200.0, 3000.0, lfo.Sine).AddTo(p), 0, 1)
			},
		},
		{
			name:   "Filter envelope amount (pluck preset)",
			cfg:    vco.PluckPreset(),
			voices: 4, amp: 0.3, legato: 0.5,
			layers: [][]note{notes(1, A3, C4, E4, A4, A3, C4, E4, A4, G3, B4, D4, G4, G3, B4, D4, G4)},
			params: map[string][]float64{"filterEnvAmount": {0.5, 1, 2, 3, 4, 5, 6, 5, 4, 3, 2, 1, 2, 3, 4, 5}},
		},
		{
			name: "Glide: alternating jumps and slides (acidBass preset, mono)",
			cfg:  vco.AcidBassPreset(),
			// one voice, so every note glides from the previous one
			voices: 1, amp: 0.25, legato: 0.6,
			layers: [][]note{notes(1, A1, A2, A1, E3, A1, A2, G3, A2, A1, A2, A1, E3, C3, A2, G3, A2)},
			params: map[string][]float64{"glide": {0, 0, 0, 0, 0, 0, 0, 0, 0, 120, 0, 120, 120, 120, 120, 120}},
		},
		{
			name: "LFO pitch and amp: delayed vibrato, then square tremolo",
			cfg: func() vco.Config {
				cfg := vco.LeadPreset()
				cfg.LFO1 = vco.LFOConfig{Waveform: vco.LFOSine, Rate: 5.5, Delay: 400.0, KeySync: true}
				cfg.LFO2 = vco.LFOConfig{Waveform: vco.LFOSquare, Rate: 8.0, KeySync: true}
				return cfg
			}(),
			voices: 2, amp: 0.25, legato: 0.9,
			layers: [][]note{notes(4, E4, D4, C4, A3)},
			params: map[string][]float64{
				"lfo1Pitch": {0.4, 0.4, 0, 0},
				"lfo2Amp":   {0, 0, 0.9, 0.9},
			},
		},
		{
			name: "LFO cutoff: sample & hold random filter, then smooth sine wah",
			cfg: func() vco.Config {
				cfg := vco.DefaultConfig()
				cfg.Cutoff = 600.0
				cfg.Resonance = 0.75
				cfg.FilterEnvAmount = 0.0
				cfg.AmpEnv = *adsr.NewSetting(1.0, 5.0, 0.9, 100.0, 1.0, 200.0)
				cfg.LFO1 = vco.LFOConfig{Waveform: vco.LFOSampleHold, Rate: 8.0}
				cfg.LFO2 = vco.LFOConfig{Waveform: vco.LFOSine, Rate: 1.0, KeySync: true}
				return cfg
			}(),
			voices: 2, amp: 0.3, legato: 0.95,
			layers: [][]note{notes(8, A2, A2)},
			params: map[string][]float64{
				"lfo1Cutoff": {2.0, 0},
				"lfo2Cutoff": {0, 2.0},
			},
		},
		{
			name:   "PWM pad (pwmPad preset): Am F C G chords",
			cfg:    vco.PWMPadPreset(),
			voices: 12, amp: 0.15, legato: 1.0,
			layers: [][]note{
				notes(4, A3, F4-12, C4, G3),
				notes(4, C4, A3, E4, B4-12),
				notes(4, E4, C4, G4, D4),
			},
		},
		{
			name:   "Filter envelope to pitch (zap preset)",
			cfg:    vco.ZapPreset(),
			voices: 4, amp: 0.25, legato: 0.5,
			layers: [][]note{{
				{A3, 1}, {rest, 1}, {A3, 0.5}, {A3, 0.5}, {rest, 1},
				{E4, 1}, {rest, 1}, {A3, 0.5}, {E4, 0.5}, {rest, 1},
				{A3, 1}, {rest, 1}, {A3, 0.5}, {A3, 0.5}, {rest, 1},
				{A4, 0.5}, {E4, 0.5}, {C4, 0.5}, {A3, 0.5}, {rest, 2},
			}},
			// one value per sounding note (13), so they line up on every loop
			params: map[string][]float64{"filterEnvToPitch": {12, 24, 36, 12, 24, 36, 12, 24, 36, 48, 48, 48, 48}},
		},
		{
			name: "Drive: increasing saturation before the filter",
			cfg: func() vco.Config {
				cfg := vco.BassPreset()
				cfg.Cutoff = 600.0
				return cfg
			}(),
			voices: 2, amp: 0.25, legato: 0.6,
			layers: [][]note{notes(1, A2, A2, C3, A2, A2, A2, E3, A2, A2, A2, C3, A2, A2, A2, G3, A2)},
			params: map[string][]float64{"drive": {1, 1, 1, 1, 2, 2, 2, 2, 4, 4, 4, 4, 8, 8, 8, 8}},
		},
		{
			name:   "Noise only through a resonant bandpass (wind preset)",
			cfg:    vco.WindPreset(),
			voices: 2, amp: 0.3, legato: 0.8,
			layers: [][]note{notes(8, C5, G5)},
		},
	}

	// Build the synths and print the schedule
	sectionSec := sectionBeats * 60.0 / float64(psalm.BPM())
	var mix []any

	for i, s := range sections {
		fmt.Printf("%6.1fs  %s\n", float64(i)*sectionSec, s.name)
		mix = append(mix, s.build(p, i, len(sections)), 0, 0)
	}

	fmt.Printf("%6.1fs  (tour loops)\n", float64(len(sections))*sectionSec)

	bpmMs := timing.BPMToMilli(psalm.BPM())

	p.Out(freeverb.New(0.7, 0.6, 0.9, 0.12, 0.8).AddTo(p).In(
		pingpong.New(bpmMs*4.0, bpmMs*0.75, 0.2, 0.15).AddTo(p).In(mix...),
	))

	//err := p.ToSoundFile("vco2.wav", writer.WAV, float64(len(sections))*sectionSec, 44100.0, true)
	//if err != nil {
	//	panic(err)
	//}

	_ = p.Listen()
}
