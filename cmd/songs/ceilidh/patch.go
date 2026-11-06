package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
)

//go:embed scripts/score.js
var scoreScript string

//go:embed scripts/flute.js
var fluteScript string

//go:embed scripts/whistle.js
var whistleScript string

//go:embed scripts/mandolin.js
var mandolinScript string

//go:embed scripts/bodhran.js
var bodhranScript string

//go:embed scripts/drums.js
var drumsScript string

//go:embed scripts/kick.js
var kickScript string

//go:embed scripts/bass.js
var bassScript string

//go:embed scripts/pad.js
var padScript string

//go:embed scripts/drone.js
var droneScript string

//go:embed scripts/arp.js
var arpScript string

//go:embed scripts/lead.js
var leadScript string

//go:embed scripts/fx.js
var fxScript string

// tempo is the bpm of the piece, in quarter notes (the test plays it faster)
var tempo = 150.0

// jigs are the files the piece is made of
var jigs = []string{"assets/midi/jig-1.mid", "assets/midi/jig-2.mid", "assets/midi/jig-3.mid"}

// sections are the parts of the piece, in bars of 6/8 (three beats)
var sections = []struct {
	name string
	bars int
}{
	{"Hearth", 8}, {"First", 32}, {"Break", 8}, {"Second", 32}, {"Bridge", 8}, {"Third", 48}, {"Outro", 8},
}

// tourSeconds is how long the piece lasts
func tourSeconds() float64 {
	bars := 0
	for _, s := range sections {
		bars += s.bars
	}
	return float64(bars) * 3 * 60 / tempo
}

func songText() string {
	parts := []string{"3/4"}
	for _, s := range sections {
		parts = append(parts, fmt.Sprintf("%s %d", s.name, s.bars))
	}
	return strings.Join(parts, ", ")
}

// tunes reads the jigs and gives the JavaScript that the score script starts with: `const TUNES = {jig1: [[start, pitch,
// length, velocity], ...], ...}`, start and length in eighth notes from the start of the file (a bar of the 6/8 is six)
func tunes(files []string) (string, error) {
	all := map[string][][4]float64{}
	for i, path := range files {
		f, err := midi.ReadFile(path)
		if err != nil {
			return "", err
		}
		if len(f.Tracks) == 0 || f.Division <= 0 {
			return "", fmt.Errorf("%s: no notes", path)
		}
		eighth := float64(f.Division) / 2
		for _, e := range f.Tracks[0].Events {
			if beats, unit, ok := e.TimeSignature(); ok && (beats != 6 || unit != 8) {
				return "", fmt.Errorf("%s: %d/%d, not 6/8", path, beats, unit)
			}
		}
		var open = map[int]int{} // key -> index in the list of the note that sounds
		var list [][4]float64
		for _, e := range f.Tracks[0].Events {
			m := e.Message
			if !m.IsChannel() {
				continue
			}
			switch m.Kind() {
			case midi.NoteOn:
				open[m.Note()] = len(list)
				list = append(list, [4]float64{float64(e.Tick) / eighth, float64(m.Note()), 0, float64(m.Velocity())})
			case midi.NoteOff:
				if j, ok := open[m.Note()]; ok {
					list[j][2] = float64(e.Tick)/eighth - list[j][0]
					delete(open, m.Note())
				}
			}
		}
		for j := range list {
			if list[j][2] <= 0 {
				return "", fmt.Errorf("%s: a note that does not end", path)
			}
		}
		all[fmt.Sprintf("jig%d", i+1)] = list
	}
	data, err := json.Marshal(all)
	if err != nil {
		return "", err
	}
	return "const TUNES = " + string(data) + "\n", nil
}

// patchIDs are the modules of the patch to look at: the scripts, and the synths by their part
type patchIDs struct {
	scripts []string
	synths  map[string]string
}

// buildPatch makes the patch: the song, a score script that holds the notes of the jigs, the chords and the layers
// of the sections, and a clock of eighths into twelve scripts that each play a synth, all on a mixer with three buses:
// a hall, an echo and a shimmer. Rendering turns the loop of the song off so it ends with the last section. It returns
// the scripts, to look at what they say
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
	data, err := tunes(jigs)
	if err != nil {
		return patchIDs{}, err
	}
	pt.Tempo().Set(tempo)
	song := add("song", "", 20, 20)
	score := add("script", "", 20, 160) // first, so it has put the chords in `shared` before the others ask
	clock := add("clock", "", 20, 300)
	type part struct {
		name, script, kind, preset string
	}
	// the parts: each script plays a synth, each synth goes straight to a channel of the mixer
	parts := []part{
		{"flute", fluteScript, "shaper", "fluteLead"},
		{"whistle", whistleScript, "waveguide", "tinWhistle"},
		{"mandolin", mandolinScript, "waveguide", "mandolin"},
		{"bodhran", bodhranScript, "waveguide", "bodhran"},
		{"drums", drumsScript, "drumsynth", "electro"},
		{"bass", bassScript, "shaper", "subSine"},
		{"pad", padScript, "shaper", "warmPad"},
		{"drone", droneScript, "waveguide", "hurdyGurdy"},
		{"arp", arpScript, "shaper", "pluck"},
		{"lead", leadScript, "shaper", "supersawLead"},
		{"fx", fxScript, "shaper", "impact"},
		{"kick", kickScript, "drumsynth", "electro"},
	}
	kickAt := len(parts) - 1
	// the sidechain: the kick is on a synth of its own, and pulls down the parts that have a compressor, each by its ratio (the
	// bass the most). Their sound goes through the compressor on its way to the mixer
	ducked := map[string]float64{"bass": 2.2, "pad": 2, "arp": 1.6, "mandolin": 1.3}
	scripts := make([]string, len(parts))
	synths := make([]string, len(parts))
	for i, p := range parts {
		y := 20 + float64(i)*110
		scripts[i] = add("script", "", 260, y)
		synths[i] = add(p.kind, p.preset, 520, y)
	}
	mixer := add("mixer", "16 channels", 780, 300)
	// the three buses of the mixer: a hall, an echo of a dotted eighth and a shimmer
	hall := add("reverb", "", 1040, 20)
	echo := add("pingpong", "", 1040, 140)
	shine := add("shimmer", "", 1040, 260)
	// playing live there is also the sound input of the computer, to record a voice for the piece: the input (its side panel
	// has the choice of the device), a recorder after it (the switch `record` writes a WAV to recordings/) and a channel of the
	// mixer, which is off until you want to hear yourself (headphones: the speakers sing back into the microphone)
	var mic, rec string
	if !once {
		y := 20 + float64(len(parts))*110
		mic = add("input", "", 260, y)
		rec = add("recorder", "", 520, y)
	}
	if err != nil {
		return patchIDs{}, err
	}
	text(song, "sections", songText())
	text(score, "code", data+scoreScript)
	ids = patchIDs{scripts: []string{score}, synths: map[string]string{}}
	for i, p := range parts {
		text(scripts[i], "code", p.script)
		connect(patcher.Cable{From: clock, To: scripts[i]})
		connect(patcher.Cable{From: scripts[i], To: synths[i]})
		if ratio, ok := ducked[p.name]; ok {
			comp := add("compressor", "", 780, 20+float64(i)*110)
			connect(patcher.Cable{From: synths[i], To: comp})
			connect(patcher.Cable{From: synths[kickAt], To: comp, In: 1})
			connect(patcher.Cable{From: comp, To: mixer, In: i})
			ids.synths[p.name] = comp // the stem of a ducked part is its sound after the compressor
			if err == nil {
				pt.SettingsOf(comp).Apply(map[string]float64{"threshold": -24, "ratio": ratio, "knee": 6, "attack": 2, "release": 180, "sideCut": 20, "mix": 1})
			}
		} else {
			connect(patcher.Cable{From: synths[i], To: mixer, In: i})
			ids.synths[p.name] = synths[i]
		}
		ids.scripts = append(ids.scripts, scripts[i])
	}
	// send A to the hall, B to the echo, C to the shimmer; each comes back on its return (after the 16 channels)
	for i, fx := range []string{hall, echo, shine} {
		connect(patcher.Cable{From: mixer, Out: 1 + i, To: fx})
		connect(patcher.Cable{From: fx, To: mixer, In: 16 + i})
	}
	connect(patcher.Cable{From: mixer, Out: 0, To: "out"})
	if !once {
		connect(patcher.Cable{From: mic, To: rec})
		connect(patcher.Cable{From: rec, To: mixer, In: len(parts)})
	}
	if err != nil {
		return patchIDs{}, err
	}
	// the clock counts eighths (a bar is six)
	pt.SettingsOf(clock).Apply(map[string]float64{"division": 1})
	// the mix: flute, whistle, mandolin, bodhran, drums, bass, pad, drone, arp, lead, fx, kick
	pt.SettingsOf(mixer).Apply(map[string]float64{
		"level1": 1.5, "pan1": -0.1, "sendA1": 0.2, "sendB1": 0.08,
		"level2": 0.5, "pan2": 0.25, "sendA2": 0.3, "sendB2": 0.2, "sendC2": 0.12,
		"level3": 0.45, "pan3": -0.35, "sendA3": 0.15, "sendB3": 0.08,
		"level4": 0.75, "pan4": 0.3, "sendA4": 0.1,
		"level5": 0.6, "pan5": 0, "sendA5": 0.08,
		"level6": 1.0, "pan6": 0,
		"level7": 0.5, "pan7": 0, "sendA7": 0.3, "sendC7": 0.3,
		"level8": 0.4, "pan8": 0, "sendA8": 0.3,
		"level9": 0.6, "pan9": 0.4, "sendA9": 0.2, "sendB9": 0.35,
		"level10": 0.6, "pan10": -0.2, "sendA10": 0.2, "sendB10": 0.3,
		"level11": 0.6, "pan11": 0, "sendA11": 0.6,
		"level12": 0.9, "pan12": 0,
		"returnA": 1, "returnB": 0.8, "returnC": 0.5, "level": 0.7,
	})
	// the drums: the kit has its snare, hats and clap far below the kick, a modern mix has them up
	pt.SettingsOf(synths[len(synths)-1]).Apply(map[string]float64{"kickLevel": 0.95, "kickDecay": 0.55})
	pt.SettingsOf(synths[4]).Apply(map[string]float64{"snareLevel": 0.7, "hatLevel": 0.7, "openLevel": 0.55, "clapLevel": 0.45})
	// the hall: a room, not a cathedral
	pt.SettingsOf(hall).Apply(map[string]float64{"mix": 0.6, "decay": 2.2, "size": 0.55})
	// the echo: a dotted eighth (0.75 of a beat), a few repeats, with the top left on
	pt.SettingsOf(echo).Apply(map[string]float64{"time": 0.75, "feedback": 0.4, "stereo": 1, "highCut": 5000, "lowCut": 200, "mix": 1})
	// the shimmer: the reverb with an octave up in it, all of it wet
	pt.SettingsOf(shine).Apply(map[string]float64{"shimmer": 0.3, "semitones": 12, "mix": 1, "decay": 4})
	if !once {
		pt.SettingsOf(mic).Apply(map[string]float64{"gain": 0, "mono": 1})
		// the microphone: off until you want to hear it
		n := strconv.Itoa(len(parts) + 1)
		pt.SettingsOf(mixer).Apply(map[string]float64{"level" + n: 0, "pan" + n: 0, "sendA" + n: 0.2})
	}
	if once {
		pt.SettingsOf(song).Apply(map[string]float64{"loop": 0})
	}
	pt.Move("out", 1060, 400)
	return ids, nil
}

// textError is the error a module shows for a text it could not read
func textError(pt *patcher.Patcher, id, key string) error {
	if e := pt.TextError(id, key); e != "" {
		return fmt.Errorf("%s of %s: %s", key, id, e)
	}
	return nil
}
