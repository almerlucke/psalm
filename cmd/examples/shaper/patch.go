package main

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
)

//go:embed scripts/lead.js
var leadScript string

//go:embed scripts/bass.js
var bassScript string

//go:embed scripts/pad.js
var padScript string

const tempo = 100

// sections are the parts of the tour, in bars
var sections = []struct {
	name string
	bars int
}{
	{"Glass", 8}, {"Fold", 8}, {"Sync", 8}, {"Pulse", 8}, {"Morph", 8}, {"Metal", 8},
	{"Vowel", 8}, {"Crush", 8}, {"Rewrite", 8}, {"Chance", 8}, {"Bells", 8}, {"Finale", 8},
}

// tourSeconds is how long the tour lasts
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

// buildPatch makes the patch: a song, a clock of sixteenths into three scripts, each playing a
// shaper synth, all on a mixer with a reverb and a ping pong echo. Rendering turns the loop of the
// song off so it ends with the last section. It returns the scripts, to look at what they say
func buildPatch(pt *patcher.Patcher, once bool) (scripts []string, err error) {
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
	clock := add("clock", "", 20, 200)
	lead := add("script", "", 260, 20)
	bass := add("script", "", 260, 200)
	pad := add("script", "", 260, 380)
	leadSynth := add("shaper", "glass", 520, 20)
	bassSynth := add("shaper", "subFold", 520, 200)
	padSynth := add("shaper", "morphPad", 520, 380)
	mixer := add("mixer", "4 channels", 780, 140)
	hall := add("reverb", "", 780, 380)
	echo := add("pingpong", "", 780, 480)
	if err != nil {
		return nil, err
	}
	text(song, "sections", songText())
	text(lead, "code", leadScript)
	text(bass, "code", bassScript)
	text(pad, "code", padScript)
	for _, s := range []string{lead, bass, pad} {
		connect(patcher.Cable{From: clock, To: s})
	}
	connect(patcher.Cable{From: lead, To: leadSynth})
	connect(patcher.Cable{From: bass, To: bassSynth})
	connect(patcher.Cable{From: pad, To: padSynth})
	for i, s := range []string{leadSynth, bassSynth, padSynth} {
		connect(patcher.Cable{From: s, To: mixer, In: i})
	}
	connect(patcher.Cable{From: mixer, Out: 1, To: hall})
	connect(patcher.Cable{From: hall, To: mixer, In: 4})
	connect(patcher.Cable{From: mixer, Out: 2, To: echo})
	connect(patcher.Cable{From: echo, To: mixer, In: 5})
	connect(patcher.Cable{From: mixer, Out: 0, To: "out"})
	if err != nil {
		return nil, err
	}
	pt.SettingsOf(mixer).Apply(map[string]float64{
		"level1": 0.85, "sendA1": 0.3, "sendB1": 0.25, "pan1": -0.15,
		"level2": 0.9, "sendA2": 0.05, "pan2": 0,
		"level3": 0.7, "sendA3": 0.4, "pan3": 0.1,
	})
	if once {
		pt.SettingsOf(song).Apply(map[string]float64{"loop": 0})
	}
	pt.Move("out", 1040, 200)
	return []string{lead, bass, pad}, nil
}

// textError is the error a module shows for a text it could not read
func textError(pt *patcher.Patcher, id, key string) error {
	if e := pt.TextError(id, key); e != "" {
		return fmt.Errorf("%s of %s: %s", key, id, e)
	}
	return nil
}
