package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/synths/sampler"
)

//go:embed scripts/score.js
var scoreScript string

//go:embed scripts/tabla.js
var tablaScript string

//go:embed scripts/perc.js
var percScript string

//go:embed scripts/chant.js
var chantScript string

//go:embed scripts/color.js
var colorScript string

//go:embed scripts/bed.js
var bedScript string

//go:embed scripts/pad.js
var padScript string

//go:embed scripts/harp.js
var harpScript string

//go:embed scripts/horn.js
var hornScript string

//go:embed scripts/choir.js
var choirScript string

// nominalTempo is the beats a minute the piece is written for: the pieces of the sounds are put on the sixteenths of this tempo (the test
// plays it faster, the pieces then follow the steps)
const nominalTempo = 90.0

// tempo is the bpm of the piece
var tempo = nominalTempo

// orchestra is where the Virtual Playing Orchestra is (it is not in the repository: without it shaper synths play the horn and the choir)
const orchestra = "assets/Virtual-Playing-Orchestra3"

func hasOrchestra() bool {
	_, err := os.Stat(orchestra + "/Vocals/choir-MIXED-sustain.sfz")
	return err == nil
}

// sections are the six parts of the psalm, in bars: every verse is four
var sections = []struct {
	name string
	bars int
}{
	{"Thanks", 12}, {"Creation", 24}, {"Egypt", 24}, {"Wilderness", 28}, {"Remembered", 12}, {"Heaven", 4},
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

// a sound is a file cut into pieces: the first key, how (hits, change, even), with what
type sound struct {
	name, path, mode string
	first            int
	args             []float64
	extra            string // vol=, pan=
}

func (s sound) line() string {
	args := ""
	for _, a := range s.args {
		args += fmt.Sprintf(" %g", a)
	}
	return fmt.Sprintf("%d %s %s%s name=%s %s", s.first, s.path, s.mode, args, s.name, s.extra)
}

// cuts finds the pieces of the sound, as the sampler will: the same function with the same numbers
func (s sound) cuts() ([]int64, sampler.Sound, error) {
	snd, err := sampler.LoadFile(s.path)
	if err != nil {
		return nil, nil, err
	}
	switch s.mode {
	case "hits":
		return sampler.OnsetCuts(snd, s.args[0], s.args[1]), snd, nil
	case "change":
		return sampler.ChangeCuts(snd, s.args[0], s.args[1]), snd, nil
	}
	return nil, snd, fmt.Errorf("%s: no pieces", s.name)
}

// the sounds that are cut: the tabla loop at its hits, the hymn at its syllables, the wooden tubes and the birds at their hits
var (
	tabla = sound{name: "tabla", path: "assets/sounds/various/tabla_loop_4bars_90bpm-86005.wav", mode: "hits", first: 36, args: []float64{9, 0.08}, extra: "vol=10"}
	chant = sound{name: "chant", path: "assets/sounds/chants/Johannes.Hymnus.wav", mode: "change", first: 36, args: []float64{1.1, 0.3}, extra: "vol=-2"}
	tubes = sound{name: "tubes", path: "assets/sounds/various/wooden-tubes-32138.wav", mode: "hits", first: 24, args: []float64{8, 0.15}, extra: "vol=12"}
	birds = sound{name: "birds", path: "assets/sounds/various/birdfish-happy-loop-6199.wav", mode: "hits", first: 0, args: []float64{8, 0.1}, extra: "vol=6"}
)

// the congas and the rest of the percussion, whole, a key each
const perc = `# the congas kit: a hit on a key each
36 assets/sounds/world_percussion/Conga/v2_01_01.wav whole name=open
37 assets/sounds/world_percussion/Conga/v2_02_01.wav whole name=open2
38 assets/sounds/world_percussion/LowConga/v3_01_01.wav whole name=low
40 assets/sounds/world_percussion/HighConga/v2_01_01.wav whole name=high
42 assets/sounds/world_percussion/MutedConga/High_v2_01_01.wav whole name=muted
44 assets/sounds/world_percussion/MutedLowConga/v3_01_01.wav whole name=mutedlow
46 assets/sounds/world_percussion/Darbuka/doom_01_01.wav whole name=doom
48 assets/sounds/world_percussion/EggShaker/fast_01.wav whole name=shaker vol=-6
50 assets/sounds/world_percussion/Claves/01.wav whole name=claves vol=-4
120 assets/sounds/various/shofar.wav whole name=shofar vol=-6
`

// slicesJS is what the scripts know of the pieces: for each cut sound, the key of every piece, where it begins (in sixteenths of the
// nominal tempo) and how long it is. The scripts put the pieces in time with it
func slicesJS() (string, error) {
	step := 60 / nominalTempo / 4
	out := map[string][]map[string]float64{}
	for _, s := range []sound{tabla, chant, tubes, birds} {
		cuts, snd, err := s.cuts()
		if err != nil {
			return "", err
		}
		sr := snd.SampleRate()
		for i := 0; i+1 < len(cuts) && s.first+i <= 127; i++ { // the pieces that have a key
			out[s.name] = append(out[s.name], map[string]float64{
				"key": float64(s.first + i), "at": math.Round(float64(cuts[i])/sr/step*100) / 100, "len": math.Round(float64(cuts[i+1]-cuts[i])/sr/step*100) / 100,
				"db": math.Round(rmsDB(snd, cuts[i], cuts[i+1])),
			})
		}
	}
	b, err := json.Marshal(out)
	return "shared.slices = " + string(b) + "\n", err
}

// rmsDB is the level of a stretch of a sound, in dB, the channels together
func rmsDB(snd sampler.Sound, from, to int64) float64 {
	sum, n := 0.0, 0.0
	for c := range snd.NumChannels() {
		for _, v := range snd.Buffer(c, 0)[from:to] {
			sum += float64(v) * float64(v)
			n++
		}
	}
	if n == 0 || sum == 0 {
		return -120
	}
	return 10 * math.Log10(sum/n)
}

// patchIDs are the modules of the patch to look at: the scripts, and the synths by their part
type patchIDs struct {
	scripts []string
	synths  map[string]string
}

// buildPatch makes the patch: the song, a score script (the verses, the pieces of the sounds, the remixes), a clock of sixteenths into nine
// scripts that each play a synth, on a mixer with three buses: a hall, an echo and a shimmer. Rendering turns the loop of the song
// off so it ends with the last section. It returns the scripts, to look at what they say
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
	slices, err := slicesJS()
	if err != nil {
		return patchIDs{}, err
	}
	pt.Tempo().Set(tempo)
	song := add("song", "", 20, 20)
	score := add("script", "", 20, 160) // first, so it has put the verses in `shared` before the others ask
	clock := add("clock", "", 20, 300)
	vpo := hasOrchestra()
	type part struct{ name, script, synth, kind, preset, sounds string }
	hornPart := part{name: "horn", kind: "shaper", preset: "trumpetVoice"}
	choirPart := part{name: "choir", kind: "shaper", preset: "choirPad"}
	if vpo {
		hornPart = part{name: "horn", kind: "sampler", sounds: "0 " + orchestra + "/Brass/french-horn-SOLO-sustain.sfz sfz"}
		choirPart = part{name: "choir", kind: "sampler", sounds: "0 " + orchestra + "/Vocals/choir-MIXED-sustain.sfz sfz"}
	}
	parts := []part{
		{name: "tabla", kind: "sampler", sounds: tabla.line()},
		{name: "perc", kind: "sampler", sounds: perc},
		{name: "chant", kind: "sampler", sounds: chant.line()},
		{name: "color", kind: "sampler", sounds: tubes.line() + "\n" + birds.line()},
		{name: "bed", kind: "grains", preset: "choirPad", sounds: "chant " + chant.path + " 56"},
		{name: "pad", kind: "shaper", preset: "throneDrone"},
		{name: "harp", kind: "shaper", preset: "harp"},
		hornPart, choirPart,
	}
	for i := range parts {
		y := 20 + float64(i)*100
		parts[i].script = add("script", "", 260, y)
		parts[i].synth = add(parts[i].kind, parts[i].preset, 520, y)
	}
	mixer := add("mixer", fmt.Sprintf("%d channels", len(parts)), 780, 300)
	hall := add("reverb", "", 1040, 20)
	echo := add("pingpong", "", 1040, 140)
	shine := add("shimmer", "", 1040, 260)
	if err != nil {
		return patchIDs{}, err
	}
	text(song, "sections", songText())
	text(score, "code", slices+scoreScript)
	scripts := map[string]string{"tabla": tablaScript, "perc": percScript, "chant": chantScript, "color": colorScript, "bed": bedScript,
		"pad": padScript, "harp": harpScript, "horn": hornScript, "choir": choirScript}
	for _, s := range parts {
		text(s.script, "code", scripts[s.name])
		if s.sounds != "" {
			text(s.synth, "sounds", s.sounds)
		}
	}
	ids = patchIDs{scripts: []string{score}, synths: map[string]string{}}
	for i, s := range parts {
		connect(patcher.Cable{From: clock, To: s.script})
		connect(patcher.Cable{From: s.script, To: s.synth})
		connect(patcher.Cable{From: s.synth, To: mixer, In: i})
		ids.scripts = append(ids.scripts, s.script)
		ids.synths[s.name] = s.synth
	}
	// send A to the hall, B to the echo, C to the shimmer; each comes back on its return
	for i, fx := range []string{hall, echo, shine} {
		connect(patcher.Cable{From: mixer, Out: 1 + i, To: fx})
		connect(patcher.Cable{From: fx, To: mixer, In: len(parts) + i})
	}
	connect(patcher.Cable{From: mixer, Out: 0, To: "out"})
	if err != nil {
		return patchIDs{}, err
	}
	// the mix: tabla, perc, chant, color, bed, pad, harp, horn, choir; the sends are the hall, the echo, the shimmer
	pt.SettingsOf(mixer).Apply(map[string]float64{
		"level1": 0.6, "pan1": 0, "sendA1": 0.2,
		"level2": 0.4, "pan2": 0.2, "sendA2": 0.25, "sendB2": 0.15,
		"level3": 0.7, "pan3": -0.1, "sendA3": 0.45, "sendB3": 0.25,
		"level4": 0.45, "pan4": 0.3, "sendA4": 0.4, "sendB4": 0.3, "sendC4": 0.15,
		"level5": 0.4, "pan5": 0, "sendA5": 0.5, "sendC5": 0.25,
		"level6": 0.4, "pan6": 0, "sendA6": 0.3,
		"level7": 0.5, "pan7": -0.3, "sendA7": 0.4, "sendB7": 0.3,
		"level8": 0.9, "pan8": 0.1, "sendA8": 0.5, "sendB8": 0.2,
		"level9": 0.9, "pan9": 0, "sendA9": 0.5, "sendC9": 0.2,
		"returnA": 1, "returnB": 0.8, "returnC": 0.3,
	})
	pt.SettingsOf(hall).Apply(map[string]float64{"mix": 0.5, "decay": 3.6, "size": 0.7})
	// the echo: a dotted eighth (0.75 of a beat), a few repeats, with the top left on
	pt.SettingsOf(echo).Apply(map[string]float64{"time": 0.75, "feedback": 0.45, "stereo": 1, "highCut": 5000, "lowCut": 150, "mix": 1})
	pt.SettingsOf(shine).Apply(map[string]float64{"shimmer": 0.25, "semitones": 7, "mix": 1, "decay": 5})
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
