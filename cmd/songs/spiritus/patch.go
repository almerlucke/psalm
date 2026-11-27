package main

import (
	_ "embed"
	"fmt"
	"strings"

	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
)

//go:embed scripts/harmony.js
var harmonyScript string

//go:embed scripts/chants.js
var chantsScript string

//go:embed scripts/choir.js
var choirScript string

//go:embed scripts/pad.js
var padScript string

//go:embed scripts/cloud.js
var cloudScript string

//go:embed scripts/bells.js
var bellsScript string

//go:embed scripts/sparkle.js
var sparkleScript string

//go:embed scripts/drone.js
var droneScript string

//go:embed scripts/halo.js
var haloScript string

//go:embed scripts/pulse.js
var pulseScript string

//go:embed scripts/breath.js
var breathScript string

// tempo is the bpm of the piece (the test plays it faster)
var tempo = 60.0

// sections are the parts of the piece, in bars
var sections = []struct {
	name string
	bars int
}{
	{"Waiting", 8}, {"Wind", 12}, {"Fire", 16}, {"Indwelling", 20}, {"Glory", 20}, {"Amen", 12},
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

// the chants, each with the pitch of its home note (shared.chants has the places where it rests on it)
const chantSounds = `adte assets/sounds/chants/Ad.te.levavi.VIII.Ton.wav 55.95
factus assets/sounds/chants/Factus.est.repente.VII.Ton.wav 57.66
epistle assets/sounds/chants/Epistle_for_the_Solemn_Mass_of_Easter_Day.wav 53.14
crux assets/sounds/chants/Crux_fidelis_(Graduale_1974).wav 60.27`

// patchIDs are the modules of the patch to look at: the scripts, and the parts that sound by name
type patchIDs struct {
	scripts []string
	synths  map[string]string
}

// buildPatch makes the patch: the song, two scripts that hold the score and the chants, a clock of sixteenths into
// seven scripts that each play a granular synth (two choirs of chants, a cloud of a pad, a sparkle of bells, a drone and
// the flames) or a synth that plays into a grain buffer (the pad and the bells: nobody hears them, the grains do). The
// mixer has one bus, a hall. Rendering turns the loop of the song off so it ends with the last
// section. It returns the scripts, to look at what they say
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
	score := add("script", "", 20, 140) // the score and the chants first, so they have put their lists in `shared` before the others ask
	chants := add("script", "", 20, 260)
	clock := add("clock", "", 20, 380)
	type part struct {
		name, code   string
		kind, preset string
		buffer       string // a part that plays into a grain buffer
		script, mod  string
	}
	// the nine parts: the granular synths that are heard, and the synths that fill the buffers
	parts := []*part{
		{name: "low", code: "const ROLE = \"low\"\n" + choirScript, kind: "grains", preset: "choirPad"},
		{name: "high", code: "const ROLE = \"high\"\n" + choirScript, kind: "grains", preset: "choirPad"},
		{name: "drone", code: droneScript, kind: "grains", preset: "frozenDrone"},
		{name: "halo", code: haloScript, kind: "grains", preset: "glassKeys"},
		{name: "cloud", code: cloudScript, kind: "grains", preset: "frozenDrone"},
		{name: "sparkle", code: sparkleScript, kind: "grains", preset: "choirPad"},
		{name: "pulse", code: pulseScript, kind: "grains", preset: "stutter"},
		{name: "breath", code: breathScript, kind: "grains", preset: "frozenDrone"},
		{name: "pad", code: padScript, kind: "shaper", preset: "airPad", buffer: "pad"},
		{name: "bells", code: bellsScript, kind: "shaper", preset: "churchBell", buffer: "bells"},
	}
	var bufs []string
	for i, s := range parts {
		y := 20 + float64(i)*120
		s.script = add("script", "", 260, y)
		s.mod = add(s.kind, s.preset, 520, y)
		if s.buffer != "" {
			b := add("grainbuffer", "", 760, y)
			text(b, "name", s.buffer)
			bufs = append(bufs, b)
			connect(patcher.Cable{From: s.mod, To: b})
		}
	}
	mixer := add("mixer", "16 channels", 1000, 260)
	hall := add("reverb", "", 1240, 20)
	if err != nil {
		return patchIDs{}, err
	}
	text(song, "sections", songText())
	text(score, "code", harmonyScript)
	text(chants, "code", chantsScript)
	ids = patchIDs{scripts: []string{score, chants}, synths: map[string]string{}}
	for _, s := range parts {
		text(s.script, "code", s.code)
		connect(patcher.Cable{From: clock, To: s.script})
		connect(patcher.Cable{From: s.script, To: s.mod})
		ids.scripts = append(ids.scripts, s.script)
		ids.synths[s.name] = s.mod
	}
	// the choirs and the others that are heard, each to a channel of the mixer; the buffers (the pad, the bells) to the last two,
	// turned all the way down: only the grains of them are heard
	for i, name := range []string{"low", "high", "drone", "halo", "cloud", "sparkle", "pulse", "breath"} {
		connect(patcher.Cable{From: ids.synths[name], To: mixer, In: i})
	}
	connect(patcher.Cable{From: bufs[0], To: mixer, In: 8})
	connect(patcher.Cable{From: bufs[1], To: mixer, In: 9})
	connect(patcher.Cable{From: mixer, Out: 1, To: hall})
	connect(patcher.Cable{From: hall, To: mixer, In: 16}) // the return of the first bus follows the 16 channels
	connect(patcher.Cable{From: mixer, Out: 0, To: "out"})
	if err != nil {
		return patchIDs{}, err
	}
	// the sound of each granular synth: what it reads (source: 0 a recording, 1 a wave, 2 live) and the shape of a note
	text(ids.synths["low"], "sounds", chantSounds)
	text(ids.synths["high"], "sounds", chantSounds)
	text(ids.synths["pulse"], "sounds", "perc assets/sounds/various/66-bpm-uniq-percussions-264333.wav 60")
	text(ids.synths["breath"], "sounds", "cymbal assets/sounds/various/cymbal-big2-83942.wav 60")
	text(ids.synths["cloud"], "buffer", "pad")
	text(ids.synths["sparkle"], "buffer", "bells")
	pt.SettingsOf(ids.synths["low"]).Apply(map[string]float64{"source": 0, "ampAttack": 3500, "ampDecay": 300, "ampSustain": 1, "ampRelease": 5000,
		"jitter": 0.5, "pitchSpread": 0.04, "stereoSpread": 0.9, "window": 0, "reverse": 0})
	pt.SettingsOf(ids.synths["high"]).Apply(map[string]float64{"source": 0, "ampAttack": 4000, "ampDecay": 300, "ampSustain": 1, "ampRelease": 5000,
		"jitter": 0.5, "pitchSpread": 0.04, "stereoSpread": 1, "window": 0, "reverse": 0})
	pt.SettingsOf(ids.synths["drone"]).Apply(map[string]float64{"source": 1, "wave": 0, "ampAttack": 4000, "ampDecay": 300, "ampSustain": 1, "ampRelease": 6000,
		"density": 60, "grainSize": 180, "jitter": 0.2, "window": 1, "spray": 0, "scan": 0, "stereoSpread": 0.3, "pitchSpread": 0.06, "level": 0})
	pt.SettingsOf(ids.synths["halo"]).Apply(map[string]float64{"source": 1, "wave": 0, "ampAttack": 30, "ampDecay": 300, "ampSustain": 1, "ampRelease": 3000,
		"window": 1, "level": 0})
	pt.SettingsOf(ids.synths["cloud"]).Apply(map[string]float64{"source": 2, "ampAttack": 4000, "ampDecay": 300, "ampSustain": 1, "ampRelease": 6000,
		"window": 1, "jitter": 0.6, "level": 0})
	pt.SettingsOf(ids.synths["sparkle"]).Apply(map[string]float64{"source": 2, "ampAttack": 600, "ampDecay": 300, "ampSustain": 1, "ampRelease": 4000,
		"window": 1, "jitter": 0.5, "level": 0})
	pt.SettingsOf(ids.synths["pulse"]).Apply(map[string]float64{"source": 0, "ampAttack": 300, "ampDecay": 300, "ampSustain": 1, "ampRelease": 1500, "level": 0, "reverse": 0})
	pt.SettingsOf(ids.synths["breath"]).Apply(map[string]float64{"source": 0, "ampAttack": 2500, "ampDecay": 300, "ampSustain": 1, "ampRelease": 4000, "level": 0})
	// the mix: low, high, drone, halo, cloud, sparkle, then the two buffers (not heard); the send is the hall
	pt.SettingsOf(mixer).Apply(map[string]float64{
		"level1": 0.7, "pan1": -0.1, "sendA1": 0.35,
		"level2": 0.55, "pan2": 0.1, "sendA2": 0.5,
		"level3": 0.3, "pan3": 0, "sendA3": 0.1,
		"level4": 0.4, "pan4": 0, "sendA4": 0.5,
		"level5": 2, "pan5": 0, "sendA5": 0.4,
		"level6": 4, "pan6": 0, "sendA6": 0.5,
		"level7": 0.6, "pan7": 0.2, "sendA7": 0.5,
		"level8": 4, "pan8": 0, "sendA8": 0.5,
		"level9": 0, "level10": 0,
		"returnA": 1,
	})
	// the hall: a big room, mostly wet
	pt.SettingsOf(hall).Apply(map[string]float64{"mix": 0.55, "decay": 5, "size": 0.75, "damping": 0.6})
	if once {
		pt.SettingsOf(song).Apply(map[string]float64{"loop": 0})
	}
	pt.Move("out", 1260, 320)
	return ids, nil
}

// textError is the error a module shows for a text it could not read
func textError(pt *patcher.Patcher, id, key string) error {
	if e := pt.TextError(id, key); e != "" {
		return fmt.Errorf("%s of %s: %s", key, id, e)
	}
	return nil
}
