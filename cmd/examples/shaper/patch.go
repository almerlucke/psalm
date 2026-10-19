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
	{"Drone", 12}, {"Pulse", 20}, {"Glow", 16}, {"Depth", 24}, {"Hollow", 12}, {"Lift", 20}, {"Bloom", 20}, {"Dissolve", 16},
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
// and the glass), all on a mixer with four buses: a reverb, an echo of a dotted eighth, a granulator and a shimmer. Rendering turns the loop of
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
	type part struct{ name, script, synth, preset string }
	// the five parts and the joins: each synth goes straight to a channel of the mixer, which has the effects
	parts := []part{{name: "seq", preset: "subSine"}, {name: "arp", preset: "marimba"}, {name: "pad", preset: "airPad"},
		{name: "lead", preset: "fluteLead"}, {name: "glass", preset: "churchBell"}, {name: "fx", preset: "impact"}}
	for i := range parts {
		y := 20 + float64(i)*120
		parts[i].script = add("script", "", 260, y)
		parts[i].synth = add("shaper", parts[i].preset, 520, y)
	}
	mixer := add("mixer", "8 channels", 780, 200)
	// the four buses of the mixer: a hall, a dotted eighth echo, a granulator and a shimmer
	hall := add("reverb", "", 1040, 20)
	echo := add("pingpong", "", 1040, 140)
	grains := add("granulator", "", 1040, 260)
	shine := add("shimmer", "", 1040, 380)
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
		connect(patcher.Cable{From: s.synth, To: mixer, In: i})
		ids.scripts = append(ids.scripts, s.script)
		ids.synths[s.name] = s.synth
	}
	// send A to the hall, B to the echo, C to the granulator, D to the shimmer; each comes back on its return
	for i, fx := range []string{hall, echo, grains, shine} {
		connect(patcher.Cable{From: mixer, Out: 1 + i, To: fx})
		connect(patcher.Cable{From: fx, To: mixer, In: 8 + i})
	}
	connect(patcher.Cable{From: mixer, Out: 0, To: "out"})
	if err != nil {
		return patchIDs{}, err
	}
	// the mix: seq, arp, pad, lead, glass, fx; the sends are the hall, the echo, the granulator, the shimmer
	pt.SettingsOf(mixer).Apply(map[string]float64{
		"level1": 0.95, "pan1": 0, "sendA1": 0.12,
		"level2": 0.6, "pan2": -0.25, "sendA2": 0.3, "sendB2": 0.4, "sendC2": 0.25,
		"level3": 0.4, "pan3": 0, "sendA3": 0.3, "sendB3": 0.05, "sendD3": 0.35,
		"level4": 0.8, "pan4": 0.15, "sendA4": 0.4, "sendB4": 0.4, "sendC4": 0.2, "sendD4": 0.25,
		"level5": 0.55, "pan5": 0.3, "sendA5": 0.5, "sendB5": 0.4, "sendC5": 0.6, "sendD5": 0.3,
		"level6": 0.7, "pan6": 0, "sendA6": 0.7, "sendB6": 0.2, "sendC6": 0.15,
		"returnA": 1, "returnB": 0.9, "returnC": 0.8, "returnD": 0.55,
	})
	// the echo: a dotted eighth (0.75 of a beat), a few repeats, with the top left on
	pt.SettingsOf(echo).Apply(map[string]float64{"time": 0.75, "feedback": 0.5, "stereo": 1, "highCut": 5000, "lowCut": 150, "mix": 1})
	// the granulator: a cloud of what comes in, grains of a tenth of a second sprayed and an octave up now and then,
	// feeding back on itself, so what is left of the layers in the last section is their grains
	pt.SettingsOf(grains).Apply(map[string]float64{"density": 24, "grainSize": 140, "jitter": 0.6, "spray": 220, "feedback": 0.6,
		"pitchSpread": 3, "octave": 0.25, "stereoSpread": 1, "mix": 1})
	// the shimmer: the reverb with an octave up in it, all of it wet
	pt.SettingsOf(shine).Apply(map[string]float64{"shimmer": 0.5, "semitones": 12, "mix": 1, "decay": 6})
	if once {
		pt.SettingsOf(song).Apply(map[string]float64{"loop": 0})
	}
	pt.Move("out", 1060, 520)
	return ids, nil
}

// textError is the error a module shows for a text it could not read
func textError(pt *patcher.Patcher, id, key string) error {
	if e := pt.TextError(id, key); e != "" {
		return fmt.Errorf("%s of %s: %s", key, id, e)
	}
	return nil
}
