package main

import (
	"strconv"
	"strings"

	"github.com/almerlucke/genny/bucket"
	"github.com/almerlucke/genny/sequence"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/blocks/oscillators/blosc"
	"github.com/almerlucke/psalm/modules/controls/templ"
	"github.com/almerlucke/psalm/modules/controls/timer"
	"github.com/almerlucke/psalm/modules/controls/timer/steps"
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/psalm/modules/effects/gain"
	"github.com/almerlucke/psalm/modules/effects/pingpong"
	"github.com/almerlucke/psalm/modules/oscillators/lfo"
	"github.com/almerlucke/psalm/synths/drums"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/sndfile"
)

// seq plays the values in order and loops (a bucket would shuffle them)
func seq[T any](values ...T) *sequence.Sequence[T] {
	return sequence.NewLoop(values...)
}

// note is a pitch (MIDI, 0 = rest) with a length in beats
type note struct {
	pitch float64
	beats float64
}

const rest = 0.0

// phrase converts notes into a step timer plus matching pitch and duration sequences,
// legato scales the sounding duration of each note relative to its length
func phrase(legato float64, notes ...note) (*timer.Timer, *sequence.Sequence[float64], *sequence.Sequence[float64]) {
	var (
		stps    []any
		pitches []float64
		durs    []float64
		beatSec = 60.0 / psalm.BPM()
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

// acidLine is a 303 style 16th note sequence: notes plus a filter envelope amount, amplitude, gate duration
// and glide time per sounding note
type acidLine struct {
	notes   []note
	envAmts []float64
	amps    []float64
	durs    []float64
	glides  []float64
}

const (
	acidGate  = 0.55 // fraction of a 16th a normal note sounds
	acidGlide = 70.0 // slide time in milliseconds
)

// newAcidLine parses 16th note steps relative to root: "7" is a note 7 semitones up, "7!" is accented,
// "7~" slides into the note from the previous one (which is held until it arrives) and "-" is a rest
func newAcidLine(root float64, amp float64, accentAmp float64, stps ...string) *acidLine {
	var (
		line     = &acidLine{}
		stepSec  = 0.25 * 60.0 / psalm.BPM()
		prevNote = -1 // index into durs of the previous sounding note, when it was on the previous step
	)

	for _, s := range stps {
		if s == "-" {
			line.notes = append(line.notes, note{rest, 0.25})
			prevNote = -1
			continue
		}

		slide := strings.Contains(s, "~")
		accent := strings.Contains(s, "!")
		off, err := strconv.Atoi(strings.Trim(s, "!~"))
		if err != nil {
			panic("invalid acid step: " + s)
		}

		// A slide ties the previous note into this one, so the previous gate stays open
		glide := 0.0
		if slide && prevNote >= 0 {
			line.durs[prevNote] = stepSec * 1.1
			glide = acidGlide
		}

		line.notes = append(line.notes, note{root + float64(off), 0.25})
		line.durs = append(line.durs, stepSec*acidGate)
		line.glides = append(line.glides, glide)
		prevNote = len(line.durs) - 1

		if accent {
			line.envAmts = append(line.envAmts, 5.5)
			line.amps = append(line.amps, accentAmp)
		} else {
			line.envAmts = append(line.envAmts, 2.5)
			line.amps = append(line.amps, amp)
		}
	}

	return line
}

// connect drives a (mono) poly synth with the acid line
func (l *acidLine) connect(p *psalm.Psalm, synth psalm.Modular) {
	t, pitches, _ := phrase(1.0, l.notes...)

	synth.ValueIn(templ.NewDurationTrigger(seq(l.durs...), seq(l.amps...), template.Template{
		"glide":           seq(l.glides...),
		"pitch":           pitches,
		"filterEnvAmount": seq(l.envAmts...),
	}).ValueIn(t.AddTo(p)))
}

// drumLevels maps pattern characters to hit amplitudes
var drumLevels = map[rune]float64{
	'X': 0.6,  // accent
	'x': 0.45, // normal
	'o': 0.3,  // soft
	'g': 0.12, // ghost
}

// drumPattern builds a 16th note step timer and matching amplitude sequence from bars of 16 characters,
// '.' is a rest, any key of drumLevels is a hit
func drumPattern(bars ...string) (*timer.Timer, *sequence.Sequence[float64]) {
	var (
		stps []any
		amps []float64
	)

	for _, b := range bars {
		if len(b) != 16 {
			panic("drum bar must have 16 steps: " + b)
		}
		for _, c := range b {
			if amp, ok := drumLevels[c]; ok {
				stps = append(stps, 0.25)
				amps = append(amps, amp)
			} else {
				stps = append(stps, -0.25)
			}
		}
	}

	return timer.WithSteps(steps.QuickSteps(stps...)), seq(amps...)
}

// repeatBars repeats a group of bars n times
func repeatBars(n int, bars ...string) []string {
	var all []string
	for range n {
		all = append(all, bars...)
	}
	return all
}

func drumTemplate(amp any, speed any, sound any) *templ.Templ {
	return templ.NewTrigger(amp, template.Template{
		"speed": speed,
		"sound": sound,
	})
}

func main() {
	psalm.PushConfig(psalm.Config{
		BPM:        124,
		PPQ:        240,
		SampleRate: 44100,
		BufferSize: 1024,
	})

	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	// Original acid house jam in A minor, everything on one chord in the classic style
	const root = 45.0 // A2

	// Bass 303: 2 bar 16th note line with octave jumps, minor third and flat seventh,
	// accents marked with ! and slides with ~
	bassLine := newAcidLine(root, 0.118, 0.177,
		"0!", "0", "12~", "-", "0", "0!", "3", "-", "0", "12!~", "0", "7", "-", "0", "10", "12!~",
		"0!", "0", "12~", "-", "0", "0!", "3~", "-", "0", "12!", "0~", "7", "3!", "-", "5", "7~",
	)

	// Lead 303: sparser answering line an octave up, square wave like the 303's second waveform
	leadLine := newAcidLine(root+12, 0.0707, 0.106,
		"-", "-", "12!", "15~", "-", "15", "-", "12", "-", "-", "19!", "17~", "-", "-", "15", "-",
		"-", "-", "12!", "15~", "-", "15", "-", "12", "-", "10", "7!~", "-", "-", "-", "3", "-",
	)

	beatHz := psalm.BPM() / 60.0

	// Bass: mono 303 so slides glide from the previous note. A slow triangle LFO, locked to 2 bars,
	// pushes the cutoff up and down on top of the long external sweep
	bassCfg := subtractive.AcidBassPreset()
	bassCfg.LFO1 = subtractive.LFOConfig{Waveform: subtractive.LFOTriangle, Rate: beatHz / 8.0, Cutoff: 0.6}
	bass := subtractive.NewPoly(bassCfg, 1).AddTo(p)
	bassLine.connect(p, bass)

	// Lead: mono square 303 with random sample & hold cutoff jumps every 8th note and a delayed
	// vibrato on longer notes
	leadCfg := subtractive.AcidBassPreset()
	leadCfg.Osc1Waveform = blosc.Square
	leadCfg.Resonance = 0.7 // the ladder self-oscillates close to 1
	leadCfg.KeyTrack = 0.3
	leadCfg.LFO1 = subtractive.LFOConfig{Waveform: subtractive.LFOSampleHold, Rate: beatHz * 2.0, Cutoff: 0.8}
	leadCfg.LFO2 = subtractive.LFOConfig{Waveform: subtractive.LFOSine, Rate: 6.0, Delay: 120.0, KeySync: true, Pitch: 0.2}
	lead := subtractive.NewPoly(leadCfg, 1).AddTo(p)
	leadLine.connect(p, lead)

	// Drums: four on the floor house beat
	soundBank := sndfile.SoundBank[float32]{}
	soundBank["kick"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/kick/Cymatics - Humble Sit Down Kick - D.wav", 4)
	soundBank["clap"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/clap/Cymatics - Humble Divine Clap.wav", 4)
	soundBank["hihat"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/hihat/Cymatics - Humble Closed Hihat 4.wav", 4)
	soundBank["openhat"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/hihat/Cymatics - Humble Open Hihat 2.wav", 4)
	soundBank["crash"] = sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/hihat/Cymatics - Humble Crash 1.wav", 4)

	drms := drums.NewDrums(soundBank, 24, 2).AddTo(p)

	// X accent, x normal, o soft, g ghost, . rest (16ths)
	kickTimer, kickAmps := drumPattern("X...x...X...x...")
	drms.ValueIn(drumTemplate(kickAmps, 1.0, "kick").ValueIn(kickTimer.AddTo(p)))

	clapTimer, clapAmps := drumPattern("....x.......x...")
	drms.ValueIn(drumTemplate(clapAmps, 1.0, "clap").ValueIn(clapTimer.AddTo(p)))

	hatTimer, hatAmps := drumPattern("g.g.g.g.g.g.g.gg")
	drms.ValueIn(drumTemplate(hatAmps, bucket.NewLoop(1.0, 1.05, 1.1), "hihat").ValueIn(hatTimer.AddTo(p)))

	openTimer, openAmps := drumPattern("..o...o...o...o.")
	drms.ValueIn(drumTemplate(openAmps, 1.0, "openhat").ValueIn(openTimer.AddTo(p)))

	// Crash at the start of every 16 bars
	crashTimer, crashAmps := drumPattern(append([]string{"o..............."}, repeatBars(15, "................")...)...)
	drms.ValueIn(drumTemplate(crashAmps, 1.0, "crash").ValueIn(crashTimer.AddTo(p)))

	// The classic acid move: slowly turning the cutoff knob (control rate LFOs on poly input 1)
	barSec := 4.0 * 60.0 / psalm.BPM()

	// Bass opens up over 16 bars and closes again over the next 16
	bass.ValueIn(lfo.New(true, 1.0/(32.0*barSec), 150.0, 1200.0, lfo.Triangle).AddTo(p), 0, 1)

	// Lead sweeps faster, every 8 bars
	lead.ValueIn(lfo.New(true, 1.0/(8.0*barSec), 300.0, 2500.0, lfo.Sine).AddTo(p), 0, 1)

	ir := convolver.SyntheticReverbIR(3.0, 0.6, psalm.SampleRate(), 1)

	p.Out(gain.New(-3).AddTo(p).In(convolver.New(ir, convolver.Config{PartitionSize: 512, Mix: 0.44, Gain: 0.3}).AddTo(p).In(
		bass, 0, 0,
		drms, 0, 0,
		gain.New(-2).AddTo(p).In(pingpong.New(pingpong.Config{Time: 0.75, Feedback: 0.35, Mix: 0.2}).AddTo(p).In(lead)), 0, 0,
	)))

	//err := p.ToSoundFile("subtractive.wav", writer.WAV, 60, 44100.0, true)
	//if err != nil {
	//	panic(err)
	//}

	_ = p.Listen()
}
