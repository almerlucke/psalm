package main

import (
	"embed"
	"fmt"
	"strings"

	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
)

//go:embed scripts/*.js
var scripts embed.FS

func script(name string) string {
	b, err := scripts.ReadFile("scripts/" + name + ".js")
	if err != nil {
		panic(err)
	}
	return string(b)
}

// tempo is the bpm of the piece (a quarter note a second: a bar of four is four seconds, so the bars are the seconds of the recording; the test plays it faster)
var tempo = 60.0

// soundPath is the recording the piece is made for
var soundPath = "local/HeavenlyNoah.wav"

// sections are the parts of the piece, in bars of 4/4 (four seconds each): 96 bars, as long as the recording
var sections = []struct {
	name string
	bars int
}{
	{"Water", 8}, {"Rain", 10}, {"Bells", 29}, {"Beasts", 26}, {"Storm", 12}, {"Voice", 11},
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
	parts := []string{"4/4"}
	for _, s := range sections {
		parts = append(parts, fmt.Sprintf("%s %d", s.name, s.bars))
	}
	return strings.Join(parts, ", ")
}

// gains are how much louder than its preset every part plays: the recording is dense (water, rain, bells), a music at the level of the presets is not heard in it
var gains = map[string]float64{"kalimba": 2, "marimba": 2, "carillon": 3, "bell": 7, "chime": 1.5, "vibe": 3, "bowl": 2.5, "piano": 4}

// patchIDs are the modules of the patch to look at: the scripts, and the synths by their part
type patchIDs struct {
	scripts []string
	synths  map[string]string
}

// buildPatch makes the patch: the song, a score script (the first, so the others find what it puts in `shared`), a script and an instrument for every part, and the
// streamer with the recording, which goes straight to the output. The parts are on a mixer with two buses: a room and a soft echo. Rendering turns the loop of the song
// off so it ends with the last section
func buildPatch(pt *patcher.Patcher, once bool, seed uint64) (ids patchIDs, err error) {
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
				if e := pt.TextError(id, key); e != "" {
					err = fmt.Errorf("%s of %s: %s", key, id, e)
				}
			}
		}
	}
	apply := func(id string, v map[string]float64) {
		if err != nil || len(v) == 0 {
			return
		}
		have := pt.SettingsOf(id).Values()
		for k := range v {
			if _, ok := have[k]; !ok {
				err = fmt.Errorf("%s has no setting %q", id, k)
				return
			}
		}
		pt.SettingsOf(id).Apply(v)
	}
	pt.Tempo().Set(tempo)
	song := add("song", "", 20, 20)
	score := add("script", "", 20, 140)
	type part struct {
		name, kind, preset string
		set                map[string]float64 // the preset made into a sound of its own
		level, pan         float64
		room, echo         float64 // sends
	}
	// the parts, each with its sound: a preset, changed (the settings are those of the synth)
	parts := []part{
		{"kalimba", "modal", "kalimbaSoft", map[string]float64{"drift": 0.35, "spread": 0.2, "cutoff": 7000, "hardness": 0.25}, 1.0, 0.25, 0.35, 0.3},
		{"marimba", "modal", "bellMarimba", map[string]float64{"drift": 0.2, "hardness": 0.1, "cutoff": 4500}, 1.0, -0.3, 0.3, 0.1},
		{"carillon", "modal", "carillon", map[string]float64{"drift": 0.3, "spread": 0.25, "hardness": 0.25, "cutoff": 6500}, 1.0, 0.35, 0.5, 0.3},
		{"bell", "modal", "handbellSoft", map[string]float64{"drift": 0.3, "spread": 0.3, "hardness": 0.3, "cutoff": 8000}, 1.0, -0.25, 0.5, 0.35},
		{"chime", "modal", "tubularSoft", map[string]float64{"drift": 0.3, "spread": 0.3, "hardness": 0.3, "cutoff": 6500}, 1.0, -0.1, 0.5, 0.3},
		{"vibe", "modal", "vibeBell", map[string]float64{"drift": 0.25, "spread": 0.25, "hardness": 0.25, "cutoff": 6500}, 1.0, 0.15, 0.5, 0.3},
		{"bowl", "modal", "tunedBowlHigh", map[string]float64{"hardness": 0.4, "spread": 0.35}, 1.0, 0, 0.5, 0.4},
		{"piano", "piano", "satie", nil, 1.0, 0.05, 0.4, 0.2},
	}
	scriptIDs := make([]string, len(parts))
	synthIDs := make([]string, len(parts))
	for i, p := range parts {
		y := 20 + float64(i)*110
		scriptIDs[i] = add("script", "", 260, y)
		synthIDs[i] = add(p.kind, p.preset, 520, y)
	}
	noahScript := add("script", "", 260, 20+float64(len(parts))*110)
	streamer := add("streamer", "", 520, 20+float64(len(parts))*110)
	mixer := add("mixer", "16 channels", 780, 300)
	room := add("reverb", "", 1040, 20)
	echo := add("pingpong", "", 1040, 160)
	limit := add("compressor", "", 1040, 300)
	if err != nil {
		return patchIDs{}, err
	}
	text(song, "sections", songText())
	text(score, "code", fmt.Sprintf("shared.seed = %d\n", seed)+script("score"))
	text(noahScript, "code", script("noah"))
	text(streamer, "sounds", fmt.Sprintf("36 %s vol=-3 name=noah", soundPath))
	connect(patcher.Cable{From: noahScript, To: streamer})
	connect(patcher.Cable{From: streamer, To: "out"})
	ids = patchIDs{scripts: []string{score, noahScript}, synths: map[string]string{"noah": streamer}}
	m := map[string]float64{"returnA": 1, "returnB": 0.7, "level": 1.6}
	for i, p := range parts {
		text(scriptIDs[i], "code", script(p.name))
		connect(patcher.Cable{From: scriptIDs[i], To: synthIDs[i]})
		connect(patcher.Cable{From: synthIDs[i], To: mixer, In: i})
		apply(synthIDs[i], p.set)
		if err == nil { // the loudness of the part is the level of its synth (the mixer's own level stops at a few decibels)
			apply(synthIDs[i], map[string]float64{"level": pt.SettingsOf(synthIDs[i]).Values()["level"] * gains[p.name]})
		}
		n := fmt.Sprint(i + 1)
		m["level"+n], m["pan"+n], m["sendA"+n], m["sendB"+n] = p.level, p.pan, p.room, p.echo
		ids.scripts = append(ids.scripts, scriptIDs[i])
		ids.synths[p.name] = synthIDs[i]
	}
	for i, fx := range []string{room, echo} {
		connect(patcher.Cable{From: mixer, Out: 1 + i, To: fx})
		connect(patcher.Cable{From: fx, To: mixer, In: 16 + i})
	}
	connect(patcher.Cable{From: mixer, Out: 0, To: limit})
	connect(patcher.Cable{From: limit, To: "out"})
	apply(limit, map[string]float64{"threshold": -10, "ratio": 10, "knee": 6, "attack": 3, "release": 150, "mix": 1})
	apply(mixer, m)
	// the room is a wooden one, not a hall; the echo is a long one with the top and the bottom taken off, a few soft repeats
	apply(room, map[string]float64{"mix": 0.7, "decay": 2.8, "size": 0.6, "damping": 0.6})
	apply(echo, map[string]float64{"time": 1.5, "feedback": 0.45, "stereo": 1, "highCut": 3500, "lowCut": 250, "mix": 1})
	if once {
		apply(song, map[string]float64{"loop": 0})
	}
	pt.Move("out", 1060, 400)
	if err != nil {
		return patchIDs{}, err
	}
	return ids, nil
}
