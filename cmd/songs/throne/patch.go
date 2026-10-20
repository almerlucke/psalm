package main

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
)

//go:embed scripts/harmony.js
var harmonyScript string

//go:embed scripts/pad.js
var padScript string

//go:embed scripts/elders.js
var eldersScript string

//go:embed scripts/lead.js
var leadScript string

//go:embed scripts/lamps.js
var lampsScript string

//go:embed scripts/fx.js
var fxScript string

//go:embed scripts/harp.js
var harpScript string

//go:embed scripts/sparkle.js
var sparkleScript string

// tempo is the bpm of the piece (the test plays it faster)
var tempo = 84.0

// sections are the parts of the piece, in bars: 7, 12, 21, 12, 24, 24, 28 and 16, together 144
var sections = []struct {
	name string
	bars int
}{
	{"Door", 7}, {"Throne", 12}, {"Lamps", 21}, {"Glass", 12}, {"Creatures", 24}, {"Elders", 24}, {"Holy", 28}, {"Worthy", 16},
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
// sixteenths into seven scripts that each play a shaper synth (the pad, the elders, the lead, the lamps,
// the joins, the harp and the sparkle of bells), all on a mixer with four buses: a hall, an echo of a dotted eighth, a
// granulator and a shimmer. Rendering turns the loop of
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
	// the seven parts: each synth goes straight to a channel of the mixer, which has the effects
	parts := []part{{name: "pad", preset: "throneDrone"}, {name: "elders", preset: "choirPad"},
		{name: "lead", preset: "trumpetVoice"}, {name: "lamps", preset: "sevenLamps"}, {name: "fx", preset: "impact"},
		{name: "harp", preset: "harp"}, {name: "sparkle", preset: "musicBox"}}
	for i := range parts {
		y := 20 + float64(i)*100
		parts[i].script = add("script", "", 260, y)
		parts[i].synth = add("shaper", parts[i].preset, 520, y)
	}
	mixer := add("mixer", fmt.Sprintf("%d channels", len(parts)), 780, 300)
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
	scripts := map[string]string{"pad": padScript, "elders": eldersScript, "lead": leadScript, "lamps": lampsScript, "fx": fxScript,
		"harp": harpScript, "sparkle": sparkleScript}
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
		connect(patcher.Cable{From: fx, To: mixer, In: len(parts) + i})
	}
	connect(patcher.Cable{From: mixer, Out: 0, To: "out"})
	if err != nil {
		return patchIDs{}, err
	}
	// the mix: the pad, the elders, the lead, the lamps, the joins, the harp, the sparkle; the sends are the hall, the echo,
	// the granulator, the shimmer
	pt.SettingsOf(mixer).Apply(map[string]float64{
		"level1": 0.5, "pan1": 0, "sendA1": 0.35, "sendB1": 0.05, "sendD1": 0.22,
		"level2": 0.5, "pan2": 0, "sendA2": 0.45, "sendD2": 0.2,
		"level3": 0.75, "pan3": 0.15, "sendA3": 0.4, "sendB3": 0.4, "sendC3": 0.2, "sendD3": 0.15,
		"level4": 0.55, "pan4": -0.25, "sendA4": 0.5, "sendB4": 0.45, "sendC4": 0.5, "sendD4": 0.18,
		"level5": 0.7, "pan5": 0, "sendA5": 0.7, "sendB5": 0.2, "sendC5": 0.15,
		"level6": 0.6, "pan6": -0.3, "sendA6": 0.45, "sendB6": 0.35, "sendC6": 0.2, "sendD6": 0.18,
		"level7": 0.45, "pan7": 0.35, "sendA7": 0.5, "sendB7": 0.4, "sendC7": 0.5, "sendD7": 0.22,
		"returnA": 1, "returnB": 0.9, "returnC": 0.8, "returnD": 0.3,
	})
	// the hall: a throne room, big and mostly wet
	pt.SettingsOf(hall).Apply(map[string]float64{"mix": 0.56, "decay": 6, "size": 0.85})
	// the echo: a dotted eighth (0.75 of a beat), a few repeats, with the top left on
	pt.SettingsOf(echo).Apply(map[string]float64{"time": 0.75, "feedback": 0.5, "stereo": 1, "highCut": 5000, "lowCut": 150, "mix": 1})
	// the granulator: a cloud of what comes in, grains of a tenth of a second sprayed and an octave up now and then,
	// feeding back on itself: the sea of glass is made of it
	pt.SettingsOf(grains).Apply(map[string]float64{"density": 24, "grainSize": 140, "jitter": 0.6, "spray": 220, "feedback": 0.6,
		"pitchSpread": 3, "octave": 0.25, "stereoSpread": 1, "mix": 1})
	// the shimmer: the reverb with a fifth up in it (seven semitones), all of it wet
	pt.SettingsOf(shine).Apply(map[string]float64{"shimmer": 0.25, "semitones": 7, "mix": 1, "decay": 6})
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
