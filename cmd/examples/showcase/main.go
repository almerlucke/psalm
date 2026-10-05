package main

// "Aurora": a slow orchestral piece in D minor, the showcase of psalm. Every synth plays a part
// of the orchestra and every effect has a role:
//
//	violins     wavetable strings on a generated sawtooth table, chorus, an LFO on the position
//	celli       subtractive strings
//	pizzicato   FM upright bass
//	harp        FM harp through freeverb and the granulator (frozen in the interlude)
//	flute       FM flute with a pitch shifted octave above, like a piccolo shadow
//	horns       DX soft brass chords
//	horn solo   FM french horn counter melody
//	soprano     pulsar vowel lead, the theme in the climax
//	choir       grains choir pad on a choir recording and a pulsar choir, in a convolved church
//	celesta     FM celesta through a pingpong delay into a shimmer reverb
//	bell pad    DX bell pad through a flanger, lined up with the vocoder by latency
//	voices      speech vocoding a wavetable pad
//	droplets    pulsar raindrops
//	aurora      wavetable drone on a generated random spectra table
//
// The strings run through a ladder lowpass that opens at dawn (the prelude) and closes at dusk
// (the coda), the whole orchestra sits in an FDN hall and a soft clipper guards the master.
//
// Every part has a score per section of the piece and only plays in its sections. The piece
// loops, the schedule is printed at start

import (
	"fmt"
	"math"

	"github.com/almerlucke/genny/sequence"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/blocks/filters/ladder"
	"github.com/almerlucke/psalm/blocks/filters/svf"
	"github.com/almerlucke/psalm/modules/controls/templ"
	"github.com/almerlucke/psalm/modules/controls/timer"
	"github.com/almerlucke/psalm/modules/controls/timer/steps"
	"github.com/almerlucke/psalm/modules/effects/chorus"
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/effects/flanger"
	"github.com/almerlucke/psalm/modules/effects/freeverb"
	"github.com/almerlucke/psalm/modules/effects/granulator"
	"github.com/almerlucke/psalm/modules/effects/latency"
	"github.com/almerlucke/psalm/modules/effects/pan"
	"github.com/almerlucke/psalm/modules/effects/pingpong"
	"github.com/almerlucke/psalm/modules/effects/pitchshift"
	"github.com/almerlucke/psalm/modules/effects/shimmer"
	"github.com/almerlucke/psalm/modules/effects/vocoder"
	"github.com/almerlucke/psalm/modules/filters"
	"github.com/almerlucke/psalm/modules/oscillators/lfo"
	"github.com/almerlucke/psalm/modules/player"
	"github.com/almerlucke/psalm/synths/dx"
	"github.com/almerlucke/psalm/synths/fm"
	"github.com/almerlucke/psalm/synths/grains"
	"github.com/almerlucke/psalm/synths/pulsar"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
	"github.com/almerlucke/psalm/tools/timing"
	"github.com/almerlucke/sndfile"
)

const bpm = 66

// seq plays the values in order and loops
func seq[T any](values ...T) *sequence.Sequence[T] {
	return sequence.NewLoop(values...)
}

// --- harmony ---

// chord holds the voicings of a chord for the sections of the orchestra
type chord struct {
	root    float64   // pizzicato bass
	celli   []float64 // root and fifth
	choir   []float64 // middle voices, also the horns an octave down
	violins []float64 // upper voices
}

var (
	Dm = chord{38, []float64{50, 57}, []float64{57, 62, 65}, []float64{62, 65, 69, 74}}
	Bb = chord{34, []float64{46, 53}, []float64{58, 62, 65}, []float64{62, 65, 70, 74}}
	Gm = chord{31, []float64{43, 50}, []float64{58, 62, 67}, []float64{62, 67, 70, 74}}
	A  = chord{33, []float64{45, 52}, []float64{57, 61, 64}, []float64{61, 64, 69, 73}}
	F  = chord{41, []float64{41, 48}, []float64{57, 60, 65}, []float64{60, 65, 69, 72}}
	C  = chord{36, []float64{48, 55}, []float64{55, 60, 64}, []float64{60, 64, 67, 72}}
)

// the progression, one chord per bar: Dm Bb Gm A F C Gm A
var progression = []chord{Dm, Bb, Gm, A, F, C, Gm, A}

// the coda comes home: Dm Bb F C Gm A Dm Dm
var homecoming = []chord{Dm, Bb, F, C, Gm, A, Dm, Dm}

// --- the form ---

type section struct {
	name     string
	bars     int
	chords   []chord // one per bar, repeating
	dynamics float64 // the loudness of the whole orchestra in this section
}

func (s section) chord(bar int) chord {
	return s.chords[bar%len(s.chords)]
}

var piece = []section{
	{"Prelude", 8, progression, 0.9},
	{"Theme", 16, progression, 0.85},
	{"Chorale", 8, progression, 1.6},
	{"Interlude", 8, progression, 1.1},
	{"Climax", 16, progression, 0.95},
	{"Coda", 8, homecoming, 1.0},
}

func pieceBars() int {
	n := 0
	for _, s := range piece {
		n += s.bars
	}
	return n
}

// --- scores ---

// ev is a note (or a chord, or a rest without pitches) lasting beats, held for hold beats (0 is
// the whole length times the legato of the part) at a velocity (0 is 0.8)
type ev struct {
	pitches  []float64
	beats    float64
	hold     float64
	velocity float64
}

func rest(beats float64) ev { return ev{beats: beats} }

func n(pitch float64, beats float64) ev { return ev{pitches: []float64{pitch}, beats: beats} }

func ch(pitches []float64, beats float64) ev { return ev{pitches: pitches, beats: beats} }

// transpose returns the pitches moved by semitones
func transpose(pitches []float64, semitones float64) []float64 {
	out := make([]float64, len(pitches))
	for i, p := range pitches {
		out[i] = p + semitones
	}
	return out
}

// perBar builds a section bar by bar
func perBar(s section, f func(bar int, c chord) []ev) []ev {
	var evs []ev
	for bar := range s.bars {
		evs = append(evs, f(bar, s.chord(bar))...)
	}
	return evs
}

// the theme, 16 bars over the progression twice
var theme = [][]ev{
	{n(69, 2), n(65, 1), n(64, 1)},
	{n(62, 3), n(65, 1)},
	{n(67, 2), n(70, 1), n(69, 1)},
	{n(64, 2), n(61, 1), n(64, 1)},
	{n(69, 2), n(72, 1), n(69, 1)},
	{n(67, 2), n(64, 1), n(67, 1)},
	{n(70, 1.5), n(69, 0.5), n(67, 1), n(74, 1)},
	{n(73, 2), n(69, 2)},
	{n(74, 2), n(72, 1), n(69, 1)},
	{n(70, 2), n(69, 1), n(65, 1)},
	{n(67, 1), n(69, 1), n(70, 1), n(74, 1)},
	{n(73, 3), n(76, 1)},
	{n(77, 2), n(76, 1), n(72, 1)},
	{n(76, 2), n(74, 1), n(72, 1)},
	{n(70, 2), n(69, 1), n(67, 1)},
	{n(69, 4)},
}

// the horn counter melody of the climax, two half notes per bar
var counter = [][]float64{
	{53, 57}, {53, 50}, {50, 55}, {52, 49}, {48, 53}, {52, 55}, {58, 57}, {57, 52},
}

// harpArp is an 8th note arpeggio up and down the chord
func harpArp(c chord) []ev {
	tones := []float64{c.celli[0], c.celli[1], c.choir[0], c.choir[1], c.choir[2], c.choir[1], c.choir[0], c.celli[1]}
	evs := make([]ev, len(tones))
	for i, t := range tones {
		evs[i] = ev{pitches: []float64{t}, beats: 0.5, velocity: []float64{0.9, 0.6, 0.7, 0.6, 0.8, 0.6, 0.7, 0.5}[i]}
	}
	return evs
}

// part is an instrument and its score: the events of every section, nil when it doesn't play
type part struct {
	name   string
	synth  psalm.Modular
	amp    float64
	legato float64
	key    string // "pitch" or "frequency"
	score  func(s section) []ev
}

// schedule turns the score of a part into a looping trigger for its synth
func schedule(p *psalm.Psalm, pt part) {
	var (
		stps    []any
		pitches [][]float64
		durs    []float64
		vels    []float64
		amps    []float64
		beatSec = 60.0 / float64(bpm)
	)
	for _, s := range piece {
		evs := pt.score(s)
		if evs == nil {
			stps = append(stps, -float64(s.bars*4))
			continue
		}
		total := 0.0
		for _, e := range evs {
			total += e.beats
			if len(e.pitches) == 0 {
				stps = append(stps, -e.beats)
				continue
			}
			stps = append(stps, e.beats)
			pitches = append(pitches, e.pitches)
			hold := e.hold
			if hold == 0 {
				hold = e.beats * pt.legato
			}
			durs = append(durs, hold*beatSec)
			vel := e.velocity
			if vel == 0 {
				vel = 0.8
			}
			vels = append(vels, vel)
			amps = append(amps, pt.amp*s.dynamics)
		}
		if total != float64(s.bars*4) {
			panic(fmt.Sprintf("%s in the %s lasts %v beats, expected %v", pt.name, s.name, total, s.bars*4))
		}
	}
	key := pt.key
	if key == "" {
		key = "pitch"
	}
	pt.synth.ValueIn(templ.NewDurationTrigger(seq(durs...), seq(amps...), template.Template{
		key:        seq(pitches...),
		"velocity": seq(vels...),
	}).ValueIn(timer.WithSteps(steps.QuickSteps(stps...)).AddTo(p)))
}

// --- arrangement ---

// arrangement follows the bar of the looping piece: it opens the string filter at dawn, closes it
// at dusk and freezes the harp granulator in the second half of the interlude
type arrangement struct {
	*psalm.Module
	strings     *filters.Filter
	gran        *granulator.Granulator
	ticksPerBar int64
	current     int
}

func (a *arrangement) CtrlTick(tick psalm.CtrlTick) {
	t := tick.TickTimestamp - 1
	pos := float64(t%(a.ticksPerBar*int64(pieceBars()))) / float64(a.ticksPerBar) // in bars

	start := 0.0
	index := 0
	for i, s := range piece {
		if pos < start+float64(s.bars) {
			index = i
			break
		}
		start += float64(s.bars)
	}
	s := piece[index]
	progress := (pos - start) / float64(s.bars)

	if index != a.current {
		a.current = index
		fmt.Printf("bar %3d  %s\n", int(start)+1, s.name)
	}

	// the strings filter: dawn and dusk, open in between
	cutoff := 12000.0
	switch s.name {
	case "Prelude":
		cutoff = 250 * math.Pow(12000.0/250, progress)
	case "Coda":
		cutoff = 12000 * math.Pow(400.0/12000, progress)
	}
	a.strings.SetCutoff(cutoff)

	// the granulator only plays in the interlude, frozen in its second half
	mix, freeze := 0.0, 0.0
	if s.name == "Interlude" {
		mix = 0.6
		if progress >= 0.5 {
			freeze = 1
		}
	}
	a.gran.ReceiveMessage(psalm.Message{"mix": mix, "freeze": freeze})
}

// softClip keeps the master below full scale: transparent below 0.7, smoothly rounding above
type softClip struct {
	*psalm.Module
}

func newSoftClip() *softClip {
	c := &softClip{Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo})}
	c.SetSelf(c)
	return c
}

func (c *softClip) Synthesize() {
	const knee = 0.7
	ins := c.Input(0).Channels
	outs := c.Output(0).Channels
	for ch := range outs {
		for i, x := range ins[ch] {
			v := float64(x)
			if a := math.Abs(v); a > knee {
				v = math.Copysign(knee+(1-knee)*math.Tanh((a-knee)/(1-knee)), v)
			}
			outs[ch][i] = float32(v)
		}
	}
}

func main() {
	psalm.PushConfig(psalm.Config{BPM: bpm, PPQ: 240, SampleRate: 44100, BufferSize: 1024})

	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	beatMs := timing.BPMToMilli(bpm)

	// the buses
	var hall, stringBus, choirBus []any
	route := func(bus *[]any, m psalm.Modular) { *bus = append(*bus, m, 0, 0) }

	// --- strings ---

	violinsCfg := wavetable.StringsPreset(gen.Brightness(64))
	violinsCfg.Position = 0.8
	violinsCfg.Unison = 3 // the chorus widens the section
	violins := wavetable.NewPoly(violinsCfg, 16).AddTo(p)
	violins.ValueIn(lfo.New(true, 0.05, 0.7, 0.9, lfo.Sine).AddTo(p), 0, 1) // the bowing breathes
	violinsChorus := chorus.New(0.3, 0.3, 0.4, 0.1, 1.0, 0.35, nil).AddTo(p).In(violins)
	route(&stringBus, pan.New(-0.35).AddTo(p).In(violinsChorus))

	celli := subtractive.NewPoly(subtractive.StringsPreset(), 8).AddTo(p)
	route(&stringBus, pan.New(0.35).AddTo(p).In(celli))

	stringFilter := filters.NewLadder(ladder.LowPass12, 250, 0.1, 1.0)
	stringFilter.AddTo(p)
	stringFilter.In(stringBus...)
	route(&hall, stringFilter)

	pizz := fm.NewPoly(fm.UprightBassPreset(), 2).AddTo(p)
	route(&hall, pan.New(0.45).AddTo(p).In(pizz))

	// --- harp: freeverb, then the granulator ---

	harp := fm.NewPoly(fm.HarpPreset(), 12).AddTo(p)
	harpRoom := freeverb.New(0.5, 0.4, 1.0, 0.25, 0.9).AddTo(p).In(pan.New(0.55).AddTo(p).In(harp))
	gcfg := granulator.DefaultConfig()
	gcfg.Mix = 0
	gcfg.Delay = 300
	gcfg.Spray = 250
	gcfg.GrainSize = 220
	gcfg.Density = 30
	gcfg.Octave = 0.25
	gran := granulator.New(gcfg)
	gran.AddTo(p)
	gran.In(harpRoom)
	route(&hall, gran)

	// --- winds and brass ---

	flute := fm.NewPoly(fm.FlutePreset(), 2).AddTo(p)
	piccolo := pitchshift.New(12, 60, 0.15).AddTo(p).In(flute) // an octave shadow above
	route(&hall, pan.New(-0.1).AddTo(p).In(piccolo))

	horns := dx.NewPoly(dx.SoftBrass(), 12, psalm.Stereo).AddTo(p)
	route(&hall, pan.New(-0.2).AddTo(p).In(horns))

	hornSolo := fm.NewPoly(fm.FrenchHornPreset(), 2).AddTo(p)
	route(&hall, pan.New(0.25).AddTo(p).In(hornSolo))

	// --- choir: grains on a choir recording, a pulsar choir and the soprano, in a church ---

	choirRec := sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/mixkit-angelical-choir-654.wav", 4)
	choir := grains.NewPoly(grains.ChoirPad(choirRec, 60), 12).AddTo(p)
	route(&choirBus, choir)
	ooh := pulsar.NewPoly(pulsar.ChoirPreset(), 12).AddTo(p)
	route(&choirBus, ooh)
	soprano := pulsar.NewPoly(pulsar.VowelLeadPreset(), 2).AddTo(p)
	route(&choirBus, soprano)

	choirClean := filters.NewSVF(svf.HighPass, 120, 0.1)
	choirClean.AddTo(p)
	choirClean.In(choirBus...)
	church := convolver.IRFromSoundFile(sndfile.MustSoundFile[float32]("assets/IR44100/St Nicolaes Church.wav"))
	// large partitions keep the long church response cheap, their latency is a natural pre delay
	choirChurch := convolver.New(church, convolver.Config{PartitionSize: 4096, Mix: 0.4, Gain: 0.15}).AddTo(p).In(choirClean)

	// --- interlude colours: celesta, bell pad, vocoder, droplets ---

	celesta := fm.NewPoly(fm.CelestaPreset(), 8).AddTo(p)
	celestaEcho := pingpong.New(beatMs*2, beatMs*0.75, 0.35, 0.3).AddTo(p).In(pan.New(0.3).AddTo(p).In(celesta))
	scfg := shimmer.DefaultConfig()
	scfg.Reverb.Decay = 7
	scfg.Shimmer = 0.45
	scfg.Mix = 0.45
	celestaSky := shimmer.New(scfg).AddTo(p).In(celestaEcho)

	speech := player.New(player.Config{
		SndFile: sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/John_1-1.wav", 4),
		Loop:    true,
		Speed:   1.0,
		Amp:     0.8,
	}).AddTo(p)
	carrierCfg := wavetable.PadPreset(gen.Brightness(16))
	carrierCfg.Position = 1
	carrierCfg.PositionLFODepth = 0
	carrierCfg.Filter = wavetable.FilterOff
	carrierCfg.Unison = 3
	carrierCfg.AmpEnv.AttackDuration = 200
	carrier := wavetable.NewPoly(carrierCfg, 12).AddTo(p)
	voc := vocoder.New(vocoder.DefaultConfig())
	voc.AddTo(p)
	voc.In(speech, 0, 0, carrier, 0, 1)
	route(&hall, voc)

	bellPad := dx.NewPoly(dx.BellPad(), 12, psalm.Stereo).AddTo(p)
	bellFlange := flanger.New(0.08, 0.5, 0.4, 0.4, nil).AddTo(p).In(bellPad)
	// the vocoder sounds an FFT frame late, the bell pad waits for it
	route(&hall, latency.New(psalm.Stereo, voc.Latency()).AddTo(p).In(bellFlange))

	droplets := pulsar.NewPoly(pulsar.RaindropsPreset(), 2).AddTo(p)
	route(&hall, droplets)

	// --- aurora: a drone on a table of random spectra, at dawn and dusk ---

	aurora := wavetable.NewPoly(wavetable.DronePreset(gen.Random(64, 6, 32, 11)), 4).AddTo(p)
	route(&hall, aurora)

	// --- the scores ---

	only := func(names ...string) func(s section) bool {
		return func(s section) bool {
			for _, n := range names {
				if s.name == n {
					return true
				}
			}
			return false
		}
	}

	parts := []part{
		{"violins", violins, 0.12, 1.0, "", func(s section) []ev {
			if !only("Prelude", "Theme", "Climax", "Coda")(s) {
				return nil
			}
			return perBar(s, func(bar int, c chord) []ev {
				if s.name == "Coda" && bar == s.bars-1 {
					return []ev{{pitches: c.violins, beats: 4, hold: 6}} // the last chord rings
				}
				return []ev{ch(c.violins, 4)}
			})
		}},
		{"celli", celli, 0.08, 1.0, "", func(s section) []ev {
			if !only("Prelude", "Theme", "Chorale", "Interlude", "Climax", "Coda")(s) {
				return nil
			}
			return perBar(s, func(bar int, c chord) []ev {
				if s.name == "Interlude" {
					return []ev{{pitches: c.celli[:1], beats: 4, velocity: 0.5}}
				}
				return []ev{ch(c.celli, 4)}
			})
		}},
		{"pizzicato", pizz, 0.19, 0.5, "", func(s section) []ev {
			if !only("Theme", "Climax")(s) {
				return nil
			}
			return perBar(s, func(bar int, c chord) []ev {
				return []ev{n(c.root, 2), n(c.root+7, 1), n(c.root+12, 1)}
			})
		}},
		{"harp", harp, 0.12, 1.6, "", func(s section) []ev {
			if !only("Prelude", "Theme", "Interlude", "Climax", "Coda")(s) {
				return nil
			}
			return perBar(s, func(bar int, c chord) []ev {
				if s.name == "Prelude" && bar < 4 {
					return []ev{rest(4)}
				}
				if s.name == "Coda" && bar >= 6 {
					if bar == 7 {
						return []ev{rest(4)}
					}
					return harpArp(c) // the final arpeggio
				}
				return harpArp(c)
			})
		}},
		{"flute", flute, 0.35, 0.95, "", func(s section) []ev {
			if s.name != "Theme" {
				return nil
			}
			var evs []ev
			for _, bar := range theme {
				evs = append(evs, bar...)
			}
			return evs
		}},
		{"horns", horns, 0.075, 0.95, "", func(s section) []ev {
			if !only("Chorale", "Climax")(s) {
				return nil
			}
			return perBar(s, func(bar int, c chord) []ev {
				low := transpose(c.choir, -12)
				if s.name == "Chorale" {
					return []ev{ch(low, 2), ch(low, 2)} // a chorale in half notes
				}
				return []ev{ch(low, 4)}
			})
		}},
		{"horn solo", hornSolo, 0.14, 0.95, "", func(s section) []ev {
			if s.name != "Climax" {
				return nil
			}
			return perBar(s, func(bar int, c chord) []ev {
				pair := counter[bar%len(counter)]
				return []ev{n(pair[0], 2), n(pair[1], 2)}
			})
		}},
		{"soprano", soprano, 0.15, 0.95, "", func(s section) []ev {
			if s.name != "Climax" {
				return nil
			}
			var evs []ev
			for _, bar := range theme {
				evs = append(evs, bar...)
			}
			return evs
		}},
		{"choir", choir, 0.25, 1.0, "", func(s section) []ev {
			if !only("Prelude", "Chorale", "Climax", "Coda")(s) {
				return nil
			}
			return perBar(s, func(bar int, c chord) []ev {
				if s.name == "Prelude" && bar%2 == 1 {
					return []ev{rest(4)}
				}
				if s.name == "Prelude" {
					return []ev{{pitches: c.choir, beats: 4, hold: 3.8}} // every other chord, within its bar
				}
				return []ev{ch(c.choir, 4)}
			})
		}},
		{"ooh", ooh, 0.045, 1.0, "", func(s section) []ev {
			if !only("Chorale", "Climax")(s) {
				return nil
			}
			return perBar(s, func(bar int, c chord) []ev {
				return []ev{ch(transpose(c.choir, 12), 4)}
			})
		}},
		{"celesta", celesta, 0.22, 0.9, "", func(s section) []ev {
			if !only("Interlude", "Coda")(s) {
				return nil
			}
			if s.name == "Coda" {
				// the theme returns as a memory (the bars that fit Dm Bb F C), then the last chord
				var evs []ev
				for _, bar := range [][]ev{theme[0], theme[1], theme[4], theme[5]} {
					for _, e := range bar {
						evs = append(evs, ev{pitches: transpose(e.pitches, 12), beats: e.beats, velocity: 0.6})
					}
				}
				evs = append(evs, rest(4), n(74, 2), n(69, 2), n(77, 2), n(74, 2), rest(4))
				return evs
			}
			return perBar(s, func(bar int, c chord) []ev {
				v := transpose(c.violins, 12)
				if bar%2 == 0 {
					return []ev{n(v[3], 1), n(v[2], 0.5), n(v[1], 0.5), n(v[0], 1), rest(1)}
				}
				return []ev{rest(1), n(v[1], 0.5), n(v[2], 0.5), n(v[3], 2)}
			})
		}},
		{"bell pad", bellPad, 0.05, 1.0, "", func(s section) []ev {
			if s.name != "Interlude" {
				return nil
			}
			return perBar(s, func(bar int, c chord) []ev { return []ev{ch(c.choir, 4)} })
		}},
		{"voices", carrier, 0.1, 1.0, "", func(s section) []ev {
			if s.name != "Interlude" {
				return nil
			}
			return perBar(s, func(bar int, c chord) []ev { return []ev{ch(append([]float64{c.celli[0]}, c.choir...), 4)} })
		}},
		{"droplets", droplets, 0.1, 1.0, "frequency", func(s section) []ev {
			if s.name != "Interlude" {
				return nil
			}
			return []ev{n(7, 16), n(9, 16)}
		}},
		{"aurora", aurora, 0.17, 1.0, "", func(s section) []ev {
			if !only("Prelude", "Coda")(s) {
				return nil
			}
			return []ev{{pitches: []float64{38, 45}, beats: 32, hold: 26}}
		}},
	}
	for _, pt := range parts {
		schedule(p, pt)
	}

	// --- the hall, the master and the arrangement ---

	hcfg := fdnreverb.DefaultConfig()
	hcfg.Decay = 3.8
	hcfg.Size = 0.8
	hcfg.Damping = 0.55
	hcfg.PreDelay = 30
	hcfg.Mix = 0.28
	orchestra := fdnreverb.New(hcfg).AddTo(p).In(hall...)

	master := newSoftClip()
	master.AddTo(p)
	master.In(orchestra, 0, 0, choirChurch, 0, 0, celestaSky, 0, 0)

	arr := &arrangement{
		Module:      psalm.NewControl(),
		strings:     stringFilter,
		gran:        gran,
		ticksPerBar: int64(psalm.PPQ() * 4),
		current:     -1,
	}
	arr.SetSelf(arr)
	arr.AddTo(p)

	barSec := 4 * 60.0 / float64(bpm)
	fmt.Println("Aurora")
	start := 0
	for _, s := range piece {
		fmt.Printf("%6.1fs  %-10s %d bars\n", float64(start)*barSec, s.name, s.bars)
		start += s.bars
	}
	fmt.Printf("%6.1fs  (loops)\n", float64(start)*barSec)

	p.Out(master)

	_ = p.Listen()
}
