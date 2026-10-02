package main

import (
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
	"github.com/almerlucke/psalm/synths/drums"
	"github.com/almerlucke/psalm/synths/vco"
	"github.com/almerlucke/psalm/tools/timing"
	"github.com/almerlucke/sndfile"
)

func seq[T any](values ...T) *bucket.Bucket[T] {
	return bucket.NewLoop(values...).SetMode(bucket.Indexed).Loop(true)
}

// note is a pitch (MIDI, 0 = rest) with a length in beats
type note struct {
	pitch float64
	beats float64
}

const rest = 0.0

// phrase converts notes into a step timer plus matching pitch and duration sequences,
// legato scales the sounding duration of each note relative to its length
func phrase(legato float64, notes ...note) (*timer.Timer, *bucket.Bucket[float64], *bucket.Bucket[float64]) {
	var (
		stps    []any
		pitches []float64
		durs    []float64
		beatSec = 60.0 / float64(psalm.BPM())
	)

	for _, n := range notes {
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

// sixteenths builds a 16th note step timer from bars of 16 steps
func sixteenths(bars ...[]any) *timer.Timer {
	var all []any
	for _, bar := range bars {
		all = append(all, bar...)
	}
	return timer.WithSteps(steps.Multiply(steps.QuickSteps(all...), 0.25))
}

func repeatBar(bar []any, n int) [][]any {
	bars := make([][]any, n)
	for i := range bars {
		bars[i] = bar
	}
	return bars
}

func drumTemplate(amp any, speed any, sound any) *templ.Templ {
	return templ.NewTrigger(amp, template.Template{
		"speed": speed,
		"sound": sound,
	})
}

// chord holds the bass root (MIDI), the chord tones and the scale as pitch classes
type chord struct {
	root  float64
	tones []int
	scale []int
}

// bar is one bar of melody over a chord
type bar struct {
	chord chord
	notes []note
}

func pitchClass(pitch float64) int {
	return ((int(pitch) % 12) + 12) % 12
}

func contains(pcs []int, pitch float64) bool {
	pc := pitchClass(pitch)
	for _, v := range pcs {
		if v == pc {
			return true
		}
	}
	return false
}

// harmonize returns a second voice below the melody note: a chord tone 3-5 semitones
// below when there is one, otherwise the diatonic third below
func harmonize(pitch float64, ch chord) float64 {
	for p := pitch - 3; p >= pitch-5; p-- {
		if contains(ch.tones, p) {
			return p
		}
	}

	found := 0
	for p := pitch - 1; p > pitch-12; p-- {
		if contains(ch.scale, p) {
			found++
			if found == 2 {
				return p
			}
		}
	}

	return pitch - 3
}

// bassBar plays a calm root, fifth, octave pattern on the chord root
func bassBar(ch chord) []note {
	r := ch.root
	return []note{{r, 2}, {r + 7, 1}, {r + 12, 1}}
}

// arpBar plays a fast chip-style 16th note arpeggio over the chord (root, 3rd, 5th, octave)
func arpBar(ch chord) []note {
	base := float64(60 + ch.tones[0])
	if base > 64 {
		base -= 12
	}

	voicing := []float64{base}
	for _, tone := range ch.tones[1:] {
		p := voicing[len(voicing)-1] + 1
		for pitchClass(p) != tone {
			p++
		}
		voicing = append(voicing, p)
	}
	voicing = append(voicing, base+12)

	pattern := []int{0, 1, 2, 3, 2, 1}
	notes := make([]note, 16)
	for i := range notes {
		notes[i] = note{voicing[pattern[i%len(pattern)]], 0.25}
	}

	return notes
}

func main() {
	psalm.PushConfig(psalm.Config{
		BPM:        96,
		PPQ:        240,
		SampleRate: 44100,
		BufferSize: 1024,
	})

	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	// Original melancholic theme in D minor (8 bars), chords: Dm Bb Gm A | Dm F A Dm
	const (
		G4  = 67.0
		A4  = 69.0
		Bb4 = 70.0
		C5  = 72.0
		Cs5 = 73.0
		D5  = 74.0
		E5  = 76.0
		F5  = 77.0
		G5  = 79.0
		A5  = 81.0
	)

	var (
		dMinor       = []int{2, 4, 5, 7, 9, 10, 0}
		dHarmonicMin = []int{2, 4, 5, 7, 9, 10, 1}
		Dm           = chord{root: 38, tones: []int{2, 5, 9}, scale: dMinor}
		Bb           = chord{root: 34, tones: []int{10, 2, 5}, scale: dMinor}
		Gm           = chord{root: 31, tones: []int{7, 10, 2}, scale: dMinor}
		A            = chord{root: 33, tones: []int{9, 1, 4}, scale: dHarmonicMin}
		F            = chord{root: 41, tones: []int{5, 9, 0}, scale: dMinor}
	)

	// A falling "sigh" motif repeated down the scale, lifting once in bar 6 before settling home
	theme := []bar{
		{Dm, []note{{F5, 2}, {E5, 1}, {D5, 1}}},
		{Bb, []note{{D5, 2}, {C5, 1}, {Bb4, 1}}},
		{Gm, []note{{Bb4, 2}, {A4, 1}, {G4, 1}}},
		{A, []note{{A4, 3}, {rest, 1}}},
		{Dm, []note{{F5, 2}, {E5, 1}, {D5, 1}}},
		{F, []note{{A5, 2}, {G5, 1}, {F5, 1}}},
		{A, []note{{E5, 2}, {Cs5, 2}}},
		{Dm, []note{{D5, 3}, {rest, 1}}},
	}

	// Derive all parts from the theme so they always follow the chords
	var leadNotes, harmonyNotes, bassNotes, arpNotes []note
	for _, b := range theme {
		for _, n := range b.notes {
			leadNotes = append(leadNotes, n)
			if n.pitch == rest {
				harmonyNotes = append(harmonyNotes, n)
			} else {
				harmonyNotes = append(harmonyNotes, note{harmonize(n.pitch, b.chord), n.beats})
			}
		}
		bassNotes = append(bassNotes, bassBar(b.chord)...)
		arpNotes = append(arpNotes, arpBar(b.chord)...)
	}

	leadTimer, leadPitches, leadDurs := phrase(0.9, leadNotes...)
	harmonyTimer, harmonyPitches, harmonyDurs := phrase(0.85, harmonyNotes...)
	bassTimer, bassPitches, bassDurs := phrase(0.9, bassNotes...)
	arpTimer, arpPitches, arpDurs := phrase(0.5, arpNotes...)

	// Bass: detuned saws, snappy filter envelope
	bassCfg := vco.DefaultConfig()
	bassCfg.Osc2Octave = -1
	bassCfg.Osc2Detune = 0.05
	bassCfg.Cutoff = 200.0
	bassCfg.Resonance = 0.5
	bassCfg.FilterEnvAmount = 3.5
	bassCfg.Drive = 2.0
	bassCfg.FilterEnv = *adsr.NewSetting(1.0, 2.0, 0.1, 150.0, 1.0, 100.0)
	bassCfg.AmpEnv = *adsr.NewSetting(1.0, 2.0, 0.8, 100.0, 1.0, 60.0)

	bass := vco.NewPoly(bassCfg, 4).AddTo(p)

	bass.ValueIn(templ.NewDurationTrigger(bassDurs, 0.45, template.Template{
		"pitch": bassPitches,
	}).ValueIn(bassTimer.AddTo(p)))

	// Lead: square + octave pulse, chip-like with a soft filter sweep
	leadCfg := vco.DefaultConfig()
	leadCfg.Osc1Waveform = blosc.Square
	leadCfg.Osc2Waveform = blosc.Rectangle
	leadCfg.PulseWidth = 0.25
	leadCfg.Osc2Detune = 12.08
	leadCfg.Osc1Level = 0.4
	leadCfg.Osc2Level = 0.2
	leadCfg.Cutoff = 1500.0
	leadCfg.Resonance = 0.3
	leadCfg.FilterEnvAmount = 1.5
	leadCfg.KeyTrack = 0.8
	leadCfg.FilterEnv = *adsr.NewSetting(1.0, 10.0, 0.5, 300.0, 1.0, 200.0)
	leadCfg.AmpEnv = *adsr.NewSetting(1.0, 40.0, 0.7, 400.0, 1.0, 500.0)

	lead := vco.NewPoly(leadCfg, 4).AddTo(p)

	lead.ValueIn(templ.NewDurationTrigger(leadDurs, 0.3, template.Template{
		"pitch": leadPitches,
	}).ValueIn(leadTimer.AddTo(p)))

	// Harmony: softer square a third (or chord tone) below the lead
	harmonyCfg := vco.DefaultConfig()
	harmonyCfg.Osc1Waveform = blosc.Square
	harmonyCfg.Osc1Level = 0.5
	harmonyCfg.Osc2Level = 0.0
	harmonyCfg.Cutoff = 1100.0
	harmonyCfg.Resonance = 0.2
	harmonyCfg.FilterEnvAmount = 1.0
	harmonyCfg.KeyTrack = 0.8
	harmonyCfg.FilterEnv = *adsr.NewSetting(1.0, 15.0, 0.4, 300.0, 1.0, 200.0)
	harmonyCfg.AmpEnv = *adsr.NewSetting(1.0, 12.0, 0.7, 250.0, 1.0, 250.0)

	harmony := vco.NewPoly(harmonyCfg, 4).AddTo(p)

	harmony.ValueIn(templ.NewDurationTrigger(harmonyDurs, 0.18, template.Template{
		"pitch": harmonyPitches,
	}).ValueIn(harmonyTimer.AddTo(p)))

	// Arpeggio: thin chip-style pulse, short plucky envelope
	arpCfg := vco.DefaultConfig()
	arpCfg.Osc1Waveform = blosc.Rectangle
	arpCfg.PulseWidth = 0.125
	arpCfg.Osc1Level = 0.5
	arpCfg.Osc2Level = 0.0
	arpCfg.Cutoff = 2500.0
	arpCfg.Resonance = 0.1
	arpCfg.FilterEnvAmount = 1.0
	arpCfg.KeyTrack = 1.0
	arpCfg.FilterEnv = *adsr.NewSetting(1.0, 1.0, 0.2, 60.0, 1.0, 40.0)
	arpCfg.AmpEnv = *adsr.NewSetting(1.0, 1.0, 0.5, 60.0, 1.0, 40.0)

	arp := vco.NewPoly(arpCfg, 4).AddTo(p)

	arp.ValueIn(templ.NewDurationTrigger(arpDurs, 0.08, template.Template{
		"pitch": arpPitches,
	}).ValueIn(arpTimer.AddTo(p)))

	// Drums: half-time 8 bar loop, kick on 1, rimshot on 3, small fill in the last bar
	soundBank := sndfile.SoundBank[float32]{}
	soundBank["kick"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/kick/boomy-real-acoustic-kick-one-shot_120bpm_C_major.wav", 4)
	soundBank["rim"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/hihat/Cymatics - Orchid Rimshot - Basic.wav", 4)
	soundBank["hihat"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/hihat/Cymatics - Humble Closed Hihat 4.wav", 4)
	soundBank["openhat"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/hihat/Cymatics - Humble Open Hihat 2.wav", 4)
	soundBank["crash"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/hihat/Cymatics - Humble Crash 1.wav", 4)

	drms := drums.NewDrums(soundBank, 16, 2).AddTo(p)

	kickBar := []any{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	kickBar2 := []any{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0}
	// Downbeats at 0.45, the pickup kick in every other bar softer
	kickAmps := []float64{0.45, 0.45, 0.2, 0.45, 0.45, 0.2, 0.45, 0.45, 0.2, 0.45, 0.45}
	drms.ValueIn(drumTemplate(seq(kickAmps...), 1.0, "kick").ValueIn(sixteenths(
		kickBar, kickBar2, kickBar, kickBar2, kickBar, kickBar2, kickBar, kickBar,
	).AddTo(p)))

	rimBar := []any{0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0}
	rimFill := []any{0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 1}
	// 8 hits on beat 3 then 2 quiet ghost notes leading back into the loop
	rimAmps := []float64{0.35, 0.35, 0.35, 0.35, 0.35, 0.35, 0.35, 0.35, 0.12, 0.2}
	drms.ValueIn(drumTemplate(seq(rimAmps...), 1.0, "rim").ValueIn(sixteenths(
		append(repeatBar(rimBar, 7), rimFill)...,
	).AddTo(p)))

	hatBar := []any{1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0}
	hatBarOpen := []any{1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0}
	drms.ValueIn(drumTemplate(seq(0.16, 0.08, 0.12, 0.08), seq(1.0, 1.1, 1.0, 1.05), "hihat").ValueIn(sixteenths(
		hatBar, hatBar, hatBar, hatBarOpen, hatBar, hatBar, hatBar, hatBarOpen,
	).AddTo(p)))

	openBar := []any{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0}
	silentBar := []any{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	drms.ValueIn(drumTemplate(0.12, 1.0, "openhat").ValueIn(sixteenths(
		silentBar, silentBar, silentBar, openBar, silentBar, silentBar, silentBar, openBar,
	).AddTo(p)))

	// Soft crash on the downbeat of every 8 bar loop
	drms.ValueIn(drumTemplate(0.15, 1.0, "crash").ValueIn(timer.WithSteps(steps.QuickSteps(32.0)).AddTo(p)))

	bpmMs := timing.BPMToMilli(psalm.BPM())

	p.Out(freeverb.New(0.8, 0.6, 0.9, 0.08, 0.8).AddTo(p).In(
		bass, 0, 0,
		drms, 0, 0,
		arp, 0, 0,
		pingpong.New(bpmMs*4.0, bpmMs*0.75, 0.25, 0.2).AddTo(p).In(lead, 0, 0, harmony, 0, 0), 0, 0,
	))

	//err := p.ToSoundFile("vco.wav", writer.WAV, 60, 44100.0, true)
	//if err != nil {
	//	panic(err)
	//}

	_ = p.Listen()
}
