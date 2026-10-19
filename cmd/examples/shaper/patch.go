package main

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
)

//go:embed scripts/harmony.js
var harmonyScript string

//go:embed scripts/seq.js
var seqScript string

//go:embed scripts/arp.js
var arpScript string

//go:embed scripts/pad.js
var padScript string

//go:embed scripts/lead.js
var leadScript string

//go:embed scripts/glass.js
var glassScript string

//go:embed scripts/fx.js
var fxScript string

// tempo is the bpm of the piece (the test plays it faster)
var tempo = 100.0

// sections are the parts of the piece, in bars
var sections = []struct {
	name string
	bars int
}{
	{"Drone", 12}, {"Pulse", 20}, {"Depth", 20}, {"Hollow", 12}, {"Return", 20}, {"Dissolve", 12},
}

// tourSeconds is how long the piece lasts
func tourSeconds() float64 {
	bars := 0
	for _, s := range sections {
		bars += s.bars
	}
	return float64(bars) * 4 * 60 / tempo
}

func songText() string {
	parts := make([]string, len(sections))
	for i, s := range sections {
		parts[i] = fmt.Sprintf("%s %d", s.name, s.bars)
	}
	return strings.Join(parts, ", ")
}

// patchIDs are the modules of the patch to look at: the scripts, and the synths by their part
type patchIDs struct {
	scripts []string
	synths  map[string]string
}

// buildPatch makes the patch: the song, a harmony script that holds the score, and a clock of
// sixteenths into five scripts that each play a shaper synth (the sequencer, an arpeggio, a pad, the lead
// and the glass), all on a mixer with a reverb and an echo of a dotted eighth. Rendering turns the loop of
// the song off so it ends with the last section. It returns the scripts, to look at what they say
func buildPatch(pt *patcher.Patcher, once bool) (ids patchIDs, err error) {
	add := func(kind, preset string, x, y float64) string {
		if err != nil {
			return ""
		}
		var id string
		id, err = pt.Add(kind, preset, x, y)
		return id
	}
	connect := func(c patcher.Cable) {
		if err == nil {
			_, err = pt.Connect(c)
		}
	}
	text := func(id, key, value string) {
		if err == nil {
			err = pt.SetText(id, key, value)
			if err == nil {
				err = textError(pt, id, key)
			}
		}
	}
	pt.Tempo().Set(tempo)
	song := add("song", "", 20, 20)
	score := add("script", "", 20, 160) // first, so it has put the chords in `shared` before the others ask
	clock := add("clock", "", 20, 300)
	type part struct{ name, script, synth, preset, effect string }
	// the five parts and the joins; each synth goes through an effect of its own before the mixer (none for the joins)
	parts := []part{{name: "seq", preset: "subSine", effect: "saturate"}, {name: "arp", preset: "marimba", effect: "crusher"},
		{name: "pad", preset: "darkPad", effect: "chorus"}, {name: "lead", preset: "fluteLead", effect: "phaser"},
		{name: "glass", preset: "churchBell", effect: "tremolo"}, {name: "fx", preset: "impact"}}
	for i := range parts {
		y := 20 + float64(i)*120
		parts[i].script = add("script", "", 260, y)
		parts[i].synth = add("shaper", parts[i].preset, 520, y)
	}
	effects := map[string]string{}
	for i, s := range parts {
		if s.effect != "" {
			effects[s.name] = add(s.effect, "", 780, 20+float64(i)*120)
		}
	}
	mixer := add("mixer", "8 channels", 1040, 200)
	hall := add("reverb", "", 1040, 560)
	echo := add("pingpong", "", 1040, 660)
	if err != nil {
		return patchIDs{}, err
	}
	text(song, "sections", songText())
	text(score, "code", harmonyScript)
	scripts := map[string]string{"seq": seqScript, "arp": arpScript, "pad": padScript, "lead": leadScript, "glass": glassScript, "fx": fxScript}
	for _, s := range parts {
		text(s.script, "code", scripts[s.name])
	}
	ids = patchIDs{scripts: []string{score}, synths: map[string]string{}}
	for i, s := range parts {
		connect(patcher.Cable{From: clock, To: s.script})
		connect(patcher.Cable{From: s.script, To: s.synth})
		if fx, ok := effects[s.name]; ok {
			connect(patcher.Cable{From: s.synth, To: fx})
			connect(patcher.Cable{From: fx, To: mixer, In: i})
		} else {
			connect(patcher.Cable{From: s.synth, To: mixer, In: i})
		}
		ids.scripts = append(ids.scripts, s.script)
		ids.synths[s.name] = s.synth
	}
	connect(patcher.Cable{From: mixer, Out: 1, To: hall})
	connect(patcher.Cable{From: hall, To: mixer, In: 8})
	connect(patcher.Cable{From: mixer, Out: 2, To: echo})
	connect(patcher.Cable{From: echo, To: mixer, In: 9})
	connect(patcher.Cable{From: mixer, Out: 0, To: "out"})
	if err != nil {
		return patchIDs{}, err
	}
	// the mix: seq, arp, pad, lead, glass, fx; the first send is the reverb, the second the echo
	pt.SettingsOf(mixer).Apply(map[string]float64{
		"level1": 0.95, "pan1": 0, "sendA1": 0.12, "sendB1": 0.0,
		"level2": 0.55, "pan2": -0.2, "sendA2": 0.3, "sendB2": 0.4,
		"level3": 0.45, "pan3": 0, "sendA3": 0.3, "sendB3": 0.05,
		"level4": 0.8, "pan4": 0.15, "sendA4": 0.4, "sendB4": 0.45,
		"level5": 0.5, "pan5": 0.3, "sendA5": 0.6, "sendB5": 0.5,
		"level6": 0.7, "pan6": 0, "sendA6": 0.7, "sendB6": 0.2,
	})
	// the echo: a dotted eighth (0.75 of a beat), dark, a few repeats
	pt.SettingsOf(echo).Apply(map[string]float64{"time": 0.75, "feedback": 0.5, "stereo": 1, "highCut": 2000, "lowCut": 150, "mix": 1})
	// the effects: grit on the sequencer, a slow wide chorus on the pad, a slow phaser on the lead, an auto-pan on the
	// bells, a little lo-fi on the arpeggio
	apply := func(part string, values map[string]float64) {
		if id, ok := effects[part]; ok {
			pt.SettingsOf(id).Apply(values)
		}
	}
	apply("seq", map[string]float64{"drive": 10, "tone": 2600, "output": -6, "mix": 0.55})
	apply("pad", map[string]float64{"rate": 0.22, "depth": 4, "delay": 20, "spread": 1, "feedback": 0, "tone": 5000, "mix": 0.45, "voices": 3})
	apply("lead", map[string]float64{"rate": 0.11, "min": 250, "max": 2200, "feedback": 0.45, "stereo": 1, "mix": 0.4, "stages": 6})
	apply("glass", map[string]float64{"mode": 1, "shape": 0, "rate": 0.2, "depth": 0.65, "stereo": 1})
	apply("arp", map[string]float64{"bits": 11, "rate": 14000, "output": 0, "mix": 0.3})
	if once {
		pt.SettingsOf(song).Apply(map[string]float64{"loop": 0})
	}
	pt.Move("out", 1060, 200)
	return ids, nil
}

// textError is the error a module shows for a text it could not read
func textError(pt *patcher.Patcher, id, key string) error {
	if e := pt.TextError(id, key); e != "" {
		return fmt.Errorf("%s of %s: %s", key, id, e)
	}
	return nil
}
