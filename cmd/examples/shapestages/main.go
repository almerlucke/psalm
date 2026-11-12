// Shapestages: a tour of the stages the shaper synth took from genny's newer shapers: wave folders, the tube, dead zones, curves,
// smoothsteps, Casio CZ phase distortion, harmonics, polynomials and funnels, and the two presets made of them (czResonant,
// tubeOrgan), then oscillators with two separate chains: line 2 reads chains of its own (a folded sine against a CZ saw, two pulse widths moving
// opposite ways, a sine bending a saw, a ring modulator of two waves...). Each plays two bars of what suits it; a macro (m1) moves a number of the stage during the note (the shape envelope), so the
// stage is heard working. The chain of each prints as it starts.
//
//	go run ./cmd/examples/shapestages                  all of them
//	go run ./cmd/examples/shapestages -only fold       the ones with fold in their name
//	go run ./cmd/examples/shapestages -render shapestages.wav
package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/blocks/voicefilter"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/shaper"
	"github.com/almerlucke/psalm/tools/harmony"
	"github.com/almerlucke/psalm/tools/render"
)

type (
	S = arrange.Section
	N = score.Note
)

type style int

const (
	lead style = iota
	bass
	pad
	arp
)

// entry is a chain (or a preset) to hear: from is the value of m1 at the start of the note, to where the shape envelope takes it
type entry struct {
	name, chain string
	from, to    float64
	style       style
	preset      func() shaper.Config
	// the second half of the tour: line 2 with chains of its own (chain2 is its chain A), the line and the cross modulation
	chain2 string
	line2  shaper.Line
	xmod   float64
	silent bool    // line 1 only modulates
	level  float64 // the output level (0: the usual)
	// the third part: a different shape on each side (width 1: line 1 left, line 2 right) or one chain made wide (sMacro: the first macro is that much lower on
	// the left and higher on the right, sMorph the same for the morph)
	width, sMorph, sMacro float64
}

var tour = []entry{
	{"fold", "sin | fold m1[1,6]", 0, 1, lead, nil, "", shaper.Line{}, 0, false, 0, 0, 0, 0},
	{"trifold", "sin | trifold m1[1,5]", 0, 1, pad, nil, "", shaper.Line{}, 0, false, 0, 0, 0, 0},
	{"tube", "sin | tube m1[1,8] 0.4", 0, 1, bass, nil, "", shaper.Line{}, 0, false, 0, 0, 0, 0},
	{"softdead", "sin | softdead m1[0,0.8] | mult 1.8", 0, 1, lead, nil, "", shaper.Line{}, 0, false, 0, 0, 0, 0},
	{"curve", "bipolar | curve m1[-0.9,0.9]", 0, 1, bass, nil, "", shaper.Line{}, 0, false, 0, 0, 0, 0},
	{"smootherstep", "tri | smootherstep 0 m1[1,0.15] | bipolar", 0, 1, lead, nil, "", shaper.Line{}, 0, false, 0, 0, 0, 0},
	{"czsaw", "czsaw m1", 0, 1, lead, nil, "", shaper.Line{}, 0, false, 0, 0, 0, 0},
	{"czsquare", "czsquare m1[0,0.95]", 0, 1, bass, nil, "", shaper.Line{}, 0, false, 0, 0, 0, 0},
	{"czpulse", "czpulse m1", 0, 1, arp, nil, "", shaper.Line{}, 0, false, 0, 0, 0, 0},
	{"harm", "bipolar | harm 1:1 3:0.6 5:0.4 8:0.3", 0, 0, pad, nil, "", shaper.Line{}, 0, false, 0, 0, 0, 0},
	{"poly", "bipolar | poly 0 1.6 0 -0.6", 0, 0, lead, nil, "", shaper.Line{}, 0, false, 0, 0, 0, 0},
	{"legendre", "bipolar | legendre 5", 0, 0, pad, nil, "", shaper.Line{}, 0, false, 0, 0, 0, 0},
	{"funnel", "bipolar | funnel -1 -0.6 -0.2 0.2 0.6 1", 0, 0, arp, nil, "", shaper.Line{}, 0, false, 0, 0, 0, 0},
	{"czResonant", "", 0, 0, lead, shaper.CZResonant, "", shaper.Line{}, 0, false, 0, 0, 0, 0},
	{"tubeOrgan", "", 0, 0, pad, shaper.TubeOrgan, "", shaper.Line{}, 0, false, 0, 0, 0, 0},
	{name: "wide morph (stereo morph)", style: pad, preset: shaper.WideMorph},
	// two oscillators with different waves: line 1 the chain, line 2 chain2
	{name: "duo fold+czsaw", chain: "sin | fold m1[1,4]", chain2: "czsaw m1[0.15,0.9]", line2: shaper.Line{Octave: 1, Detune: 7, Level: 0.5}, to: 1, level: 0.8, style: lead},
	{name: "duo opposite PWM", chain: "pulse m1[0.4,0.06] | bipolar", chain2: "pulse m1[0.06,0.4] | bipolar", line2: shaper.Line{Detune: 12, Level: 1}, to: 1, level: 0.3, style: pad},
	{name: "duo sub+saw", chain: "sin | tube 2 0.2", chain2: "bipolar | softclip m1[1,6]", line2: shaper.Line{Octave: 1, Detune: 4, Level: 0.45}, to: 1, level: 0.7, style: bass},
	{name: "duo tri+pulse pad", chain: "tri | bipolar", chain2: "pulse 0.3 | bipolar", line2: shaper.Line{Detune: 9, Level: 0.8}, level: 0.4, style: pad},
	{name: "duo two syncs", chain: "mult m1[1,6] | mod 1 | bipolar", chain2: "mult m1[1.5,4.5] | mod 1 | bipolar", line2: shaper.Line{Detune: 7, Level: 0.6}, to: 1, level: 1.2, style: lead},
	{name: "duo fm: sine mod saw", chain: "sin", chain2: "bipolar | softclip 1.5", line2: shaper.Line{Octave: 1, Detune: 3, Level: 1}, xmod: 0.12, silent: true, style: arp},
	{name: "duo ring: fold x czres", chain: "sin | fold 2", chain2: "czres 0.7 6", line2: shaper.Line{Detune: 13, Level: 1}, xmod: 1, level: 0.4, style: pad},
	{name: "split fold | czsaw", chain: "sin | fold m1[1,4]", chain2: "czsaw m1[0.15,0.9]", line2: shaper.Line{Detune: 5, Level: 0.7}, width: 1, level: 0.8, to: 1, style: lead},
	{name: "split tri | pulse pad", chain: "tri | bipolar", chain2: "pulse 0.3 | bipolar", line2: shaper.Line{Detune: 6, Level: 0.8}, width: 1, level: 0.5, style: pad},
	{name: "split sine | saw bass", chain: "sin", chain2: "bipolar | softclip m1[1,5]", line2: shaper.Line{Level: 0.6}, width: 1, level: 0.7, to: 1, style: bass},
	{name: "wide pwm (stereo m1)", chain: "pulse m1[0.45,0.08] | bipolar", from: 0.5, sMacro: 0.8, level: 0.35, style: pad},
	{name: "wide fold (stereo m1)", chain: "sin | fold m1[1,5]", from: 0.3, to: 0.6, sMacro: 0.7, level: 0.5, style: pad},
	{name: "wide CZ (stereo m1)", chain: "czsaw m1", from: 0.2, to: 0.9, sMacro: 0.6, level: 0.7, style: lead},
	{name: "duo harm+poly pad", chain: "bipolar | harm 1:1 3:0.5 5:0.3", chain2: "bipolar | poly 0 1.6 0 -0.6", line2: shaper.Line{Detune: 6, Level: 0.6}, level: 0.4, style: pad},
}

func env(attack, decay, sustain, release float64) adsr.Setting {
	return *adsr.NewSetting(1, attack, sustain, decay, 1, release)
}

// chain2Of is how the chain of line 2 is written after the chain of line 1
func chain2Of(e entry) string {
	if e.chain2 == "" {
		return ""
	}
	return "   +   line 2: " + e.chain2
}

func config(e entry) shaper.Config {
	if e.preset != nil {
		return e.preset()
	}
	cfg := shaper.Config{
		Chains:  [shaper.NumChains]string{e.chain},
		Chains2: [shaper.NumChains]string{e.chain2},
		Macros:  [4]float64{e.from}, ShapeAmount: e.to - e.from,
		ShapeEnv: env(5, 900, 0.4, 500),
		Line1:    shaper.Line{Level: 1}, Line2: shaper.Line{Detune: 5, Level: 0.4},
		AmpEnv: env(8, 500, 0.75, 350), Width: 0.4, Level: 0.6,
		Filter: voicefilter.Config{Type: voicefilter.Ladder24, Cutoff: 5500, Resonance: 0.05, Env: env(1, 100, 1, 100)},
	}
	cfg.StereoMorph = e.sMorph
	cfg.StereoMacros[0] = e.sMacro
	if e.sMacro != 0 || e.sMorph != 0 {
		cfg.Unison, cfg.Line2, cfg.Width = 1, shaper.Line{}, 0 // one copy of one line, centred: the image is the stereo morph's
		if e.level > 0 {
			cfg.Level = e.level
		}
	}
	if e.chain2 != "" {
		cfg.Line2 = e.line2
		cfg.XMod, cfg.XModSilent = e.xmod, e.silent
		if e.name == "duo ring: fold x czres" {
			cfg.XModType = 1
		}
		cfg.Unison, cfg.Spread = 2, 5
		if e.width > 0 {
			cfg.Width, cfg.Unison, cfg.UniStereo = e.width, 1, 0
		}
		if e.level > 0 {
			cfg.Level = e.level
		}
	}
	return cfg
}

func main() {
	only := flag.String("only", "", "only the ones with this in their name")
	renderTo := flag.String("render", "", "render to this WAV file instead of playing")
	flag.Parse()
	psalm.PushConfig(psalm.Config{BPM: 110, PPQ: 240, SampleRate: 44100, BufferSize: 512})
	var p *psalm.Psalm
	if *renderTo != "" {
		p = psalm.New(nil)
	} else {
		p = psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	}
	defer p.Close()

	var chosen []entry
	for _, e := range tour {
		if *only == "" || strings.Contains(strings.ToLower(e.name), strings.ToLower(*only)) {
			chosen = append(chosen, e)
		}
	}
	if len(chosen) == 0 {
		log.Fatalf("nothing with %q in its name", *only)
	}
	var sections []S
	for _, e := range chosen {
		sections = append(sections, S{Name: e.name + " " + e.chain + chain2Of(e), Bars: 2})
	}
	form := arrange.NewForm(sections...)
	chords := harmony.MustKey("D minor").MustProgression("i VI")

	mix := mixer.New(p)
	hall := fdnreverb.DefaultConfig()
	hall.Decay, hall.Mix = 2.2, 1
	mix.Bus("hall", mixer.StripConfig{}, fdnreverb.New(hall).AddTo(p))
	for _, e := range chosen {
		synth := shaper.NewPoly(config(e), 8).AddTo(p)
		name := e.name + " " + e.chain + chain2Of(e)
		form.Part(p, synth, name, score.Config{Velocity: 0.75}, arrange.Only(name, func(S) []N { return play(e.style, chords) }))
		mix.Channel(name, synth, mixer.StripConfig{Volume: -8, Sends: map[string]float64{"hall": 0.2}})
	}
	arrangement := arrange.NewArrangement(form).PrintSections(true)
	arrangement.AddTo(p)
	p.Out(mix.Master())

	if *renderTo != "" {
		stats, err := render.ToFile(p, *renderTo, render.Options{Seconds: arrangement.Duration(), Tail: 4})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(stats)
		return
	}
	fmt.Println("playing, Ctrl-C to stop")
	_ = p.Listen()
}

func play(st style, chords []harmony.Chord) []N {
	c1, c2 := chords[0], chords[1]
	switch st {
	case pad:
		return []N{score.Chord(c1.Close(57), 3.5), score.Chord(c2.Close(57), 3), score.Rest(1.5)}
	case bass:
		r1, r2 := c1.BassPitch(38), c2.BassPitch(38)
		return score.Shape(score.Notes(0.5, r1, r1, r1+12, r1, 0, r1+7, r1+10, r1+12, r2, r2, r2+12, r2, 0, r2+7, r2+5, 0), score.Hold(0.8))
	case arp:
		var ns []N
		for _, c := range []harmony.Chord{c1, c2} {
			v := c.Close(62)
			ns = append(ns, score.Notes(0.25, v[0], v[1], v[2], v[0]+12, v[2], v[1], v[0], v[1]+12)...)
			ns = append(ns, score.Notes(0.25, v[0], v[1], v[2], v[0]+12, v[2]+12, v[1]+12, v[0]+12, v[2])...)
		}
		return ns
	}
	return score.MustParse(`D5:1 F5:0.5 A5:0.5 C6:1.5 A5:0.5 | Bb5:1 A5:0.5 G5:0.5 F5:1 r:1`)
}
