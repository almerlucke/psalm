package main

import (
	"fmt"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/genny/sequence"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/blocks/filters/ladder"
	"github.com/almerlucke/psalm/blocks/oscillators/blosc"
	"github.com/almerlucke/psalm/modules/controls/templ"
	"github.com/almerlucke/psalm/modules/controls/timer"
	"github.com/almerlucke/psalm/modules/controls/timer/steps"
	"github.com/almerlucke/psalm/modules/effects/freeverb"
	"github.com/almerlucke/psalm/modules/effects/pingpong"
	"github.com/almerlucke/psalm/modules/oscillators/lfo"
	"github.com/almerlucke/psalm/synths/subtractive"
)

// A guided tour through all features of the subtractive voice: every section plays for sectionBeats,
// one after the other, and the whole tour loops. The schedule is printed at start.

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

const rest = 0.0

func single(pitch float64, beats float64) note {
	if pitch == rest {
		return note{nil, beats}
	}
	return note{[]float64{pitch}, beats}
}

// notes builds single notes of equal length from pitches
func notes(beats float64, pitches ...float64) []note {
	ns := make([]note, len(pitches))
	for i, p := range pitches {
		ns[i] = single(p, beats)
	}
	return ns
}

// line builds single notes from pitch, beats pairs
func line(pairs ...float64) []note {
	ns := make([]note, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		ns = append(ns, single(pairs[i], pairs[i+1]))
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

// hits builds 16th note hits of one pitch from bar patterns, 'x' is a hit and '.' a rest
func hits(pitch float64, bars ...string) []note {
	var ns []note
	for _, bar := range bars {
		for _, c := range bar {
			if c == 'x' {
				ns = append(ns, single(pitch, 0.25))
			} else {
				ns = append(ns, single(rest, 0.25))
			}
		}
	}
	return ns
}

// section demonstrates a feature: notes (or chords) played by a synth, params are sent with every
// note and cycle per note, so they can step through settings
type section struct {
	name   string
	cfg    subtractive.Config
	voices int
	amp    float64
	legato float64
	notes  []note
	params map[string][]float64
	patch  func(p *psalm.Psalm, synth psalm.Modular) // optional extra modulation
	with   []*section                                // extra parts playing in the same slot (e.g. a drum kit)
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
		out = append(out, single(rest, float64(index)*sectionBeats))
	}
	out = append(out, ns...)
	if after := numSections - 1 - index; after > 0 {
		out = append(out, single(rest, float64(after)*sectionBeats))
	}
	return out
}

// play converts notes into a step timer plus matching pitch and duration sequences. Every pitch
// value is a slice, so the template sends one note message per pitch: a chord for more than one
func play(legato float64, ns []note) (*timer.Timer, *sequence.Sequence[[]float64], *sequence.Sequence[float64]) {
	var (
		stps    []any
		pitches [][]float64
		durs    []float64
		beatSec = 60.0 / psalm.BPM()
	)

	for _, n := range ns {
		if len(n.pitches) == 0 {
			stps = append(stps, -n.beats)
			continue
		}

		stps = append(stps, n.beats)
		pitches = append(pitches, n.pitches)
		durs = append(durs, n.beats*beatSec*legato)
	}

	return timer.WithSteps(steps.QuickSteps(stps...)), seq(pitches...), seq(durs...)
}

// build creates the synths of the section (and the parts it plays with) and returns them for mixing
func (s *section) build(p *psalm.Psalm, index int, numSections int) []psalm.Modular {
	synth := subtractive.NewPoly(s.cfg, s.voices).AddTo(p)

	if beatsOf(s.notes) != sectionBeats {
		panic(fmt.Sprintf("section %q lasts %v beats, expected %v", s.name, beatsOf(s.notes), sectionBeats))
	}

	t, pitches, durs := play(s.legato, schedule(s.notes, index, numSections))

	msg := template.Template{"pitch": pitches}
	for key, values := range s.params {
		msg[key] = seq(values...)
	}

	synth.ValueIn(templ.NewDurationTrigger(durs, s.amp, msg).ValueIn(t.AddTo(p)))

	if s.patch != nil {
		s.patch(p, synth)
	}

	synths := []psalm.Modular{synth}
	for _, part := range s.with {
		synths = append(synths, part.build(p, index, numSections)...)
	}

	return synths
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
	plain := func() subtractive.Config {
		cfg := subtractive.DefaultConfig()
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
			notes: notes(2, A3, E4, A3, E4, A3, E4, A3, E4),
			params: map[string][]float64{"osc1Waveform": {
				float64(blosc.Sine), float64(blosc.Triangle), float64(blosc.Square), float64(blosc.Rectangle),
				float64(blosc.Sawtooth), float64(blosc.Ramp), float64(blosc.ModifiedSquare), float64(blosc.TrapezoidVariable),
			}},
		},
		{
			name: "Osc 2 detune and octave: unison, slow beating, chorus, fifth, octave",
			cfg: func() subtractive.Config {
				cfg := plain()
				cfg.Osc1Level, cfg.Osc2Level = 0.4, 0.4
				cfg.Cutoff = 3000.0
				return cfg
			}(),
			voices: 2, amp: 0.3, legato: 0.9,
			notes:  notes(2, A3, A3, A3, A3, A3, A3, A3, A3),
			params: map[string][]float64{"osc2Detune": {0, 0.03, 0.08, 0.15, 0.3, 7, 12, 12.1}},
		},
		{
			name: "Pulse width: narrowing pulse, then LFO pulse width modulation",
			cfg: func() subtractive.Config {
				cfg := plain()
				cfg.Osc1Waveform = blosc.Rectangle
				cfg.LFO1 = subtractive.LFOConfig{Waveform: subtractive.LFOTriangle, Rate: 2.0}
				return cfg
			}(),
			voices: 2, amp: 0.3, legato: 0.9,
			notes: notes(2, A3, A3, A3, A3, A3, C4, D4, E4),
			params: map[string][]float64{
				"pulseWidth":     {0.5, 0.3, 0.15, 0.05, 0.5, 0.5, 0.5, 0.5},
				"lfo1PulseWidth": {0, 0, 0, 0, 0.4, 0.4, 0.4, 0.4},
			},
		},
		{
			name: "Sub oscillator and noise: plain, + sub, + white noise, + pink noise",
			cfg: func() subtractive.Config {
				cfg := subtractive.BassPreset()
				cfg.Osc2Level = 0.0
				return cfg
			}(),
			voices: 2, amp: 0.392, legato: 0.6,
			notes: notes(1, A2, A2, E3, A2, A2, A2, G3, A2, A2, A2, E3, A2, A2, A2, G3, A2),
			params: map[string][]float64{
				"subLevel":   {0, 0, 0, 0, 0.6, 0.6, 0.6, 0.6, 0.6, 0.6, 0.6, 0.6, 0.6, 0.6, 0.6, 0.6},
				"noiseLevel": {0, 0, 0, 0, 0, 0, 0, 0, 0.3, 0.3, 0.3, 0.3, 0.3, 0.3, 0.3, 0.3},
				"noiseType": {0, 0, 0, 0, 0, 0, 0, 0, float64(subtractive.WhiteNoise), float64(subtractive.WhiteNoise),
					float64(subtractive.WhiteNoise), float64(subtractive.WhiteNoise), float64(subtractive.PinkNoise), float64(subtractive.PinkNoise),
					float64(subtractive.PinkNoise), float64(subtractive.PinkNoise)},
			},
		},
		{
			name:   "Ring modulation (ringBell preset)",
			cfg:    subtractive.RingBellPreset(),
			voices: 8, amp: 0.398, legato: 0.5,
			notes: notes(1, A4, C5, E5, G5, A5, G5, E5, C5, A4, C5, D5, E5, G5, E5, D5, C5),
		},
		{
			name:   "Hard sync (syncLead preset): sweeping the osc 2 pitch against osc 1",
			cfg:    subtractive.SyncLeadPreset(),
			voices: 2, amp: 0.424, legato: 0.9,
			notes:  notes(2, A3, A3, A3, A3, A3, A3, A3, A3),
			params: map[string][]float64{"osc2Detune": {0, 4, 7, 12, 17, 24, 19, 12}},
		},
		{
			name: "FM: osc 1 modulating osc 2 with growing amount",
			cfg: func() subtractive.Config {
				cfg := plain()
				cfg.Osc1Waveform, cfg.Osc2Waveform = blosc.Sine, blosc.Sine
				cfg.Osc1Level, cfg.Osc2Level = 0.0, 0.5
				cfg.Osc2Detune = 0.0
				return cfg
			}(),
			voices: 2, amp: 0.35, legato: 0.9,
			notes:  notes(2, A3, A3, A3, A3, A3, A3, A3, A3),
			params: map[string][]float64{"fmAmount": {0, 0.5, 1, 2, 4, 8, 4, 1}},
		},
		{
			name: "SVF filter modes (lowpass, bandpass, highpass, notch, peak) and resonance, cutoff swept by an external LFO",
			cfg: func() subtractive.Config {
				cfg := subtractive.DefaultConfig()
				cfg.FilterEnvAmount = 0.0
				cfg.AmpEnv = *adsr.NewSetting(1.0, 5.0, 0.8, 100.0, 1.0, 150.0)
				return cfg
			}(),
			voices: 2, amp: 0.22, legato: 0.95,
			notes: notes(2, A2, A2, A2, A2, A2, A2, A2, A2),
			params: map[string][]float64{
				"filterMode": {float64(subtractive.LowPass), float64(subtractive.LowPass), float64(subtractive.BandPass), float64(subtractive.BandPass),
					float64(subtractive.HighPass), float64(subtractive.HighPass), float64(subtractive.Notch), float64(subtractive.Peak)},
				"resonance": {0.2, 0.85, 0.2, 0.85, 0.2, 0.85, 0.3, 0.6},
			},
			patch: func(p *psalm.Psalm, synth psalm.Modular) {
				synth.ValueIn(lfo.New(true, 1.0, 200.0, 3000.0, lfo.Sine).AddTo(p), 0, 1)
			},
		},
		{
			name:   "Filter envelope amount (pluck preset)",
			cfg:    subtractive.PluckPreset(),
			voices: 4, amp: 0.345, legato: 0.5,
			notes:  notes(1, A3, C4, E4, A4, A3, C4, E4, A4, G3, B4, D4, G4, G3, B4, D4, G4),
			params: map[string][]float64{"filterEnvAmount": {0.5, 1, 2, 3, 4, 5, 6, 5, 4, 3, 2, 1, 2, 3, 4, 5}},
		},
		{
			name: "Glide: alternating jumps and slides (acidBass preset, mono)",
			cfg:  subtractive.AcidBassPreset(),
			// one voice, so every note glides from the previous one
			voices: 1, amp: 0.147, legato: 0.6,
			notes:  notes(1, A1, A2, A1, E3, A1, A2, G3, A2, A1, A2, A1, E3, C3, A2, G3, A2),
			params: map[string][]float64{"glide": {0, 0, 0, 0, 0, 0, 0, 0, 0, 120, 0, 120, 120, 120, 120, 120}},
		},
		{
			name: "LFO pitch and amp: delayed vibrato, then square tremolo",
			cfg: func() subtractive.Config {
				cfg := subtractive.LeadPreset()
				cfg.LFO1 = subtractive.LFOConfig{Waveform: subtractive.LFOSine, Rate: 5.5, Delay: 400.0, KeySync: true}
				cfg.LFO2 = subtractive.LFOConfig{Waveform: subtractive.LFOSquare, Rate: 8.0, KeySync: true}
				return cfg
			}(),
			voices: 2, amp: 0.478, legato: 0.9,
			notes: notes(4, E4, D4, C4, A3),
			params: map[string][]float64{
				"lfo1Pitch": {0.4, 0.4, 0, 0},
				"lfo2Amp":   {0, 0, 0.9, 0.9},
			},
		},
		{
			name: "LFO cutoff: sample & hold random filter, then smooth sine wah",
			cfg: func() subtractive.Config {
				cfg := subtractive.DefaultConfig()
				cfg.Cutoff = 600.0
				cfg.Resonance = 0.75
				cfg.FilterEnvAmount = 0.0
				cfg.AmpEnv = *adsr.NewSetting(1.0, 5.0, 0.9, 100.0, 1.0, 200.0)
				cfg.LFO1 = subtractive.LFOConfig{Waveform: subtractive.LFOSampleHold, Rate: 8.0}
				cfg.LFO2 = subtractive.LFOConfig{Waveform: subtractive.LFOSine, Rate: 1.0, KeySync: true}
				return cfg
			}(),
			voices: 2, amp: 0.3, legato: 0.95,
			notes: notes(8, A2, A2),
			params: map[string][]float64{
				"lfo1Cutoff": {2.0, 0},
				"lfo2Cutoff": {0, 2.0},
			},
		},
		{
			name:   "PWM pad (pwmPad preset): Am F C G chords",
			cfg:    subtractive.PWMPadPreset(),
			voices: 12, amp: 0.372, legato: 1.0,
			notes: chordNotes(4,
				[]float64{A3, C4, E4},      // Am
				[]float64{F4 - 12, A3, C4}, // F
				[]float64{C4, E4, G4},      // C
				[]float64{G3, B4 - 12, D4}, // G
			),
		},
		{
			name:   "Filter envelope to pitch (zap preset)",
			cfg:    subtractive.ZapPreset(),
			voices: 4, amp: 0.298, legato: 0.5,
			notes: line(A3, 1, rest, 1, A3, 0.5, A3, 0.5, rest, 1,
				E4, 1, rest, 1, A3, 0.5, E4, 0.5, rest, 1,
				A3, 1, rest, 1, A3, 0.5, A3, 0.5, rest, 1,
				A4, 0.5, E4, 0.5, C4, 0.5, A3, 0.5, rest, 2),
			// one value per sounding note (13), so they line up on every loop
			params: map[string][]float64{"filterEnvToPitch": {12, 24, 36, 12, 24, 36, 12, 24, 36, 48, 48, 48, 48}},
		},
		{
			name: "Drive: increasing saturation before the filter",
			cfg: func() subtractive.Config {
				cfg := subtractive.BassPreset()
				cfg.Cutoff = 600.0
				return cfg
			}(),
			voices: 2, amp: 0.327, legato: 0.6,
			notes:  notes(1, A2, A2, C3, A2, A2, A2, E3, A2, A2, A2, C3, A2, A2, A2, G3, A2),
			params: map[string][]float64{"drive": {1, 1, 1, 1, 2, 2, 2, 2, 4, 4, 4, 4, 8, 8, 8, 8}},
		},
		{
			name:   "Noise only through a resonant bandpass (wind preset)",
			cfg:    subtractive.WindPreset(),
			voices: 2, amp: 0.259, legato: 0.8,
			notes: notes(8, C5, G5),
		},
		{
			name:   "Reese bass (reeseBass preset): detuned saws with a slow internal cutoff LFO",
			cfg:    subtractive.ReeseBassPreset(),
			voices: 2, amp: 0.48, legato: 0.95,
			notes: notes(4, A1, A1, 29, 31),
		},
		{
			name:   "FM bass (fmBass preset) with changing FM amount",
			cfg:    subtractive.FMBassPreset(),
			voices: 2, amp: 0.584, legato: 0.6,
			notes:  notes(1, A1, A2, A1, A2, 36, C3, 36, 43, A1, A2, A1, A2, 31, 43, 31, 40),
			params: map[string][]float64{"fmAmount": {2, 3, 4, 3}},
		},
		{
			name:   "Square bass with short glide (squareBass preset, mono)",
			cfg:    subtractive.SquareBassPreset(),
			voices: 1, amp: 0.469, legato: 0.7,
			notes: notes(1, A2, A2, E3, A2, 43, A2, C3, 50, A2, A2, E3, A3, G3, E3, 50, C3),
		},
		{
			name:   "LFO wobble bass (lfoWobble preset): wobble rate set per note",
			cfg:    subtractive.LFOWobblePreset(),
			voices: 2, amp: 0.681, legato: 0.95,
			notes:  notes(4, A1, A1, 29, 31),
			params: map[string][]float64{"lfo1Rate": {2, 4, 6, 8}},
		},
		{
			name: "Synth drums (synthKick, synthSnare, synthHat, synthTom presets)",
			cfg:  subtractive.SynthKickPreset(), voices: 2, amp: 0.648, legato: 0.5,
			notes: hits(36,
				"x.....x...x.....", "x.....x...x.....", "x.....x...x.....", "x.....x.x.x....."),
			with: []*section{
				{
					name: "snare", cfg: subtractive.SynthSnarePreset(), voices: 2, amp: 0.295, legato: 0.5,
					notes: hits(50,
						"....x.......x...", "....x.......x...", "....x.......x...", "....x.......x.xx"),
				},
				{
					name: "hat", cfg: subtractive.SynthHatPreset(), voices: 4, amp: 0.123, legato: 0.5,
					notes: hits(60,
						"x.x.x.x.x.x.x.xx", "x.x.x.x.x.x.x.xx", "x.x.x.x.x.x.x.xx", "x.x.x.x.x.x....."),
				},
				{
					name: "tom", cfg: subtractive.SynthTomPreset(), voices: 4, amp: 0.39, legato: 0.5,
					notes: line(rest, 14, G3, 0.5, E3, 0.5, C3, 0.5, A2, 0.5),
				},
			},
		},
		{
			name:   "Flute (flute preset): breath noise and delayed vibrato",
			cfg:    subtractive.FlutePreset(),
			voices: 2, amp: 0.432, legato: 0.95,
			notes: line(E5, 2, D5, 1, C5, 1, A4, 3, rest, 1, C5, 1, D5, 1, E5, 2, G5, 3, rest, 1),
		},
		{
			name:   "Clarinet (clarinet preset)",
			cfg:    subtractive.ClarinetPreset(),
			voices: 2, amp: 0.644, legato: 0.95,
			notes: line(A3, 1, C4, 1, E4, 2, D4, 1, C4, 1, 59, 2, C4, 1, D4, 1, E4, 1, F4, 1, E4, 4),
		},
		{
			name:   "Theremin (theremin preset, mono): long glides and wide vibrato",
			cfg:    subtractive.ThereminPreset(),
			voices: 1, amp: 0.735, legato: 1.0,
			notes: line(A4, 3, E5, 3, D5, 2, C5, 3, A4, 3, rest, 2),
		},
		{
			name:   "Electric piano (ePiano preset): FM tone with tremolo, chords",
			cfg:    subtractive.EPianoPreset(),
			voices: 12, amp: 0.123, legato: 0.9,
			notes: chordNotes(4,
				[]float64{A3, C4, E4}, // Am
				[]float64{D4, F4, A4}, // Dm
				[]float64{G3, 59, D4}, // G
				[]float64{C4, E4, G4}, // C
			),
		},
		{
			name:   "Glass keys (glassKeys preset): FM bell arpeggio",
			cfg:    subtractive.GlassKeysPreset(),
			voices: 8, amp: 0.228, legato: 0.5,
			notes: notes(0.5,
				A4, C5, E5, G5, A5, G5, E5, C5, A4, C5, E5, G5, A5, G5, E5, C5,
				G4, B4, D5, G5, D5, B4, G4, D5, A4, C5, E5, A5, E5, C5, A4, E5),
		},
		{
			name:   "Vibes (vibes preset): mallet tone with tremolo",
			cfg:    subtractive.VibesPreset(),
			voices: 6, amp: 0.35, legato: 0.9,
			notes: notes(2, C5, E5, G5, B4, A4, C5, E5, G5),
		},
		{
			name:   "Riser FX (riser preset): pitch and filter swell",
			cfg:    subtractive.RiserPreset(),
			voices: 2, amp: 0.112, legato: 0.95,
			notes: notes(8, A3, E4),
		},
		{
			name:   "Ladder filter modes (ladderBass preset): lowpass 24, lowpass 12, bandpass 12, highpass 12",
			cfg:    subtractive.LadderBassPreset(),
			voices: 2, amp: 0.265, legato: 0.6,
			notes: notes(1, A1, A2, A1, E3, A1, A2, A1, G3, A1, A2, A1, E3, A1, A2, C3, G3),
			params: map[string][]float64{"ladderMode": {
				float64(ladder.LowPass24), float64(ladder.LowPass24), float64(ladder.LowPass24), float64(ladder.LowPass24),
				float64(ladder.LowPass12), float64(ladder.LowPass12), float64(ladder.LowPass12), float64(ladder.LowPass12),
				float64(ladder.BandPass12), float64(ladder.BandPass12), float64(ladder.BandPass12), float64(ladder.BandPass12),
				float64(ladder.HighPass12), float64(ladder.HighPass12), float64(ladder.HighPass12), float64(ladder.HighPass12),
			}},
		},
		{
			name:   "Ladder lead (ladderLead preset, mono): drive, glide and delayed vibrato",
			cfg:    subtractive.LadderLeadPreset(),
			voices: 1, amp: 0.35, legato: 1.0,
			notes: line(A4, 2, C5, 1, D5, 1, E5, 3, D5, 1, C5, 2, A4, 2, G4, 3, rest, 1),
		},
		{
			name:   "Talking synth (talkingSynth preset): an LFO sweeps the formant filter through the vowels",
			cfg:    subtractive.TalkingSynthPreset(),
			voices: 2, amp: 0.362, legato: 0.9,
			notes: notes(2, A2, A2, C3, A2, 43, A2, C3, E3),
		},
		{
			name:   "Choir pad (choirPad preset): formant filtered saws, Am F C G",
			cfg:    subtractive.ChoirPadPreset(),
			voices: 12, amp: 0.0776, legato: 1.0,
			notes: chordNotes(4,
				[]float64{A3, C4, E4},      // Am
				[]float64{F4 - 12, A3, C4}, // F
				[]float64{C4, E4, G4},      // C
				[]float64{G3, 59, D4},      // G
			),
		},
		{
			name:   "Notch pad (notchPad preset): an LFO sweeps a notch, like a phaser",
			cfg:    subtractive.NotchPadPreset(),
			voices: 8, amp: 0.243, legato: 1.0,
			notes: chordNotes(8, []float64{A3, E4}, []float64{F4 - 12, C4}),
		},
	}

	// Build the synths and print the schedule
	sectionSec := sectionBeats * 60.0 / psalm.BPM()
	var mix []any

	for i, s := range sections {
		fmt.Printf("%6.1fs  %s\n", float64(i)*sectionSec, s.name)
		for _, synth := range s.build(p, i, len(sections)) {
			mix = append(mix, synth, 0, 0)
		}
	}

	fmt.Printf("%6.1fs  (tour loops)\n", float64(len(sections))*sectionSec)

	p.Out(freeverb.New(0.7, 0.6, 0.9, 0.12, 0.8).AddTo(p).In(
		pingpong.New(pingpong.Config{Time: 0.75, Feedback: 0.2, Mix: 0.15}).AddTo(p).In(mix...),
	))

	//err := p.ToSoundFile("subtractive2.wav", writer.WAV, float64(len(sections))*sectionSec, 44100.0, true)
	//if err != nil {
	//	panic(err)
	//}

	_ = p.Listen()
}
