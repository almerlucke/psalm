package main

import (
	"embed"
	"fmt"
	"math"
	"os"

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

const (
	wavesPath    = "assets/sounds/various/boat_waves-6099.wav"
	seaBirdsPath = "assets/sounds/various/mixkit-sea-waves-with-birds-loop-1185.wav"
	gullsPath    = "assets/sounds/various/seagulls-by-the-sea-7042.wav"
	animalsPath  = "assets/sounds/various/animals.wav"
	childrenPath = "assets/sounds/various/mixkit-laughing-children-indoors-427.wav"
	wordPath     = "assets/sounds/various/John_1-1.wav"
	choirPath    = "assets/sounds/various/mixkit-angelical-choir-654.wav"
)

// tempo is the beats a minute the song starts at (the test plays it faster)
var tempo = float64(bpm)

type options struct {
	just  bool
	a4    float64
	seed  uint64 // the seed of the feel of the parts
	human bool   // the parts play off the grid, like players
	once  bool   // the song ends with its last day, it doesn't loop
	real  bool   // the parts the orchestra has play its recordings instead of the synths (see realParts)
}

// orchestra is where the Virtual Playing Orchestra is (it is not in the repository: without it the synths play)
const orchestra = "assets/Virtual-Playing-Orchestra3"

func hasOrchestra() bool {
	_, err := os.Stat(orchestra + "/Vocals/choir-MIXED-sustain.sfz")
	return err == nil
}

// realParts are the instruments the orchestra has recorded: the sfz file (in the orchestra) of the part, and what its level is multiplied by (the recordings are
// not as loud as the synths were; the levels were found by rendering both and comparing the parts)
var realParts = map[string]struct {
	sfz  string
	gain float64
}{
	"violins":   {"Strings/1st-violin-SEC-sustain.sfz", 1.4},
	"celli":     {"Strings/cello-SEC-sustain.sfz", 1.3},
	"pizzicato": {"Strings/all-strings-SEC-pizzicato.sfz", 1.7},
	"harp":      {"Strings/harp-sustain.sfz", 1.8},
	"timpani":   {"Percussion/timpani-hit.sfz", 1},
	"flute":     {"Woodwinds/flute-SOLO-sustain.sfz", 1.2},
	"oboe":      {"Woodwinds/oboe-SOLO-sustain.sfz", 1},
	"clarinet":  {"Woodwinds/clarinet-SOLO-sustain.sfz", 1},
	"horns":     {"Brass/french-horn-SEC-sustain.sfz", 0.8},
	"horn":      {"Brass/french-horn-SOLO-sustain.sfz", 1},
	"celesta":   {"Keys/celesta.sfz", 1.3},
	"bell pad":  {"Percussion/vibraphone-open.sfz", 0.8},
	"bells":     {"Percussion/tubular-bells.sfz", 6},
	"soprano":   {"Vocals/choir-FEMALE-sustain.sfz", 1.3},
	"choir":     {"Vocals/choir-MIXED-sustain.sfz", 1.3},
}

// real is the strip played by the orchestra: a sampler with the sfz of the part, in place of its synth
func (s strip) real() strip {
	r, ok := realParts[s.name]
	if !ok {
		return s
	}
	s.kind, s.preset, s.settings = "sampler", "", nil
	s.sounds = "0 " + orchestra + "/" + r.sfz + " sfz"
	s.level *= r.gain
	return s
}

// patchIDs are the modules of the patch to look at: the scripts, and the synths by their part
type patchIDs struct {
	mixer   string
	scripts []string
	synths  map[string]string
}

// strip is a part of the piece: its script, the synth that plays it and its place on the mixer. The level is the amplitude of the part in the song of
// Go divided by the velocity its notes have (0.8): the scripts play with the velocity, the level makes it as loud as the part was (the timpani and the bells
// play softer notes than 0.8, the word is the vocoder's, which says how loud it is by the voice, and the void was measured against the song of Go)
type strip struct {
	name, script, kind, preset string
	sounds                     string             // the sounds of a sampler or a grains synth
	kit                        string             // the percussion of a sampler: its recordings, see kits.go
	settings                   map[string]float64 // of the synth, after it is made
	level                      float64
	pan                        float64
	hall, heavens, church      float64 // the sends: the hall, the heavens (a shimmer), the church (a convolved room)
	chain                      string  // "": the synth goes to the strip; else the effect it goes through: "pitchshift", "pingpong", "vocoder"
	automated                  bool    // its level is moved by a cable from its script (a recording faded in and out)
}

// the church: the choir, the organ and the sopranos sing in it (their level is lowered and the room sent in, as the group of the song of Go did: 60% of the sound
// as it is, 40% of the room)
const churchLevel = 0.6 * 0.794
const churchSend = 0.67

func parts() []strip {
	amp := func(a float64) float64 { return a / 0.8 }
	return []strip{
		{name: "void", script: "void", kind: "wavetable", preset: "drone", level: amp(0.111) * 0.7, hall: 0.4},
		{name: "spirit", script: "spirit", kind: "pulsar", preset: "choir", level: amp(0.061) * churchLevel, heavens: 0.2, church: churchSend},
		{name: "the word", script: "word", kind: "wavetable", preset: "pad", chain: "vocoder", level: 0.59, hall: 0.35, heavens: 0.1,
			settings: map[string]float64{"position": 1, "positionLFODepth": 0, "filter": 0, "unison": 3}},
		{name: "violins", script: "violins", kind: "wavetable", preset: "strings", level: amp(0.126), pan: -0.35, hall: 0.45, settings: map[string]float64{"position": 0.8}},
		{name: "celli", script: "celli", kind: "subtractive", preset: "strings", level: amp(0.145), pan: 0.35, hall: 0.4},
		{name: "pizzicato", script: "pizzicato", kind: "dx", preset: "pizzicato", level: amp(0.25), pan: 0.45, hall: 0.3},
		{name: "harp", script: "harp", kind: "dx", preset: "harp", level: amp(0.13), pan: 0.55, hall: 0.5},
		{name: "timpani", script: "timpani", kind: "dx", preset: "timpani", level: amp(0.2) * 1.45, pan: -0.1, hall: 0.45},
		{name: "flute", script: "flute", kind: "dx", preset: "flute", chain: "pitchshift", level: amp(0.21), pan: -0.15, hall: 0.4, heavens: 0.1},
		{name: "oboe", script: "oboe", kind: "dx", preset: "oboe", level: amp(0.2), pan: 0.2, hall: 0.4},
		{name: "clarinet", script: "clarinet", kind: "dx", preset: "clarinet", level: amp(0.25), pan: -0.4, hall: 0.4},
		{name: "horns", script: "horns", kind: "dx", preset: "softBrass", level: amp(0.14), pan: -0.2, hall: 0.45},
		{name: "horn", script: "horn", kind: "dx", preset: "frenchHorn", level: amp(0.33), pan: 0.25, hall: 0.4},
		{name: "celesta", script: "celesta", kind: "dx", preset: "celesta", chain: "pingpong", level: amp(0.217), pan: 0.3, heavens: 0.8},
		{name: "bell pad", script: "bellpad", kind: "dx", preset: "bellPad", level: amp(0.0471), hall: 0.4, heavens: 0.3},
		{name: "bells", script: "bells", kind: "dx", preset: "musicBox", level: amp(0.12) * 1.6, pan: 0.15, hall: 0.5, heavens: 0.3},
		{name: "the deep", script: "deep", kind: "pulsar", preset: "formantPad", level: amp(0.2), hall: 0.5},
		{name: "piano", script: "piano", kind: "piano", preset: "grand", level: amp(0.2), pan: -0.05, hall: 0.35},
		{name: "soprano", script: "soprano", kind: "pulsar", preset: "vowelLead", level: amp(0.34) * churchLevel, heavens: 0.15, church: churchSend},
		{name: "choir", script: "choir", kind: "grains", preset: "choirPad", sounds: "choir " + choirPath + " 60", level: amp(0.11) * churchLevel, church: churchSend},
		{name: "organ", script: "organ", kind: "dx", preset: "churchOrgan", level: amp(0.1) * churchLevel, church: churchSend},
		// the recordings of creation: going round while their note is held, their levels follow the days (a cable from the script of each to its strip)
		{name: "waves", script: "waves", kind: "sampler", sounds: "36 " + wavesPath + " whole loop=1.5 name=waves", hall: 0.15, automated: true},
		{name: "sea and birds", script: "seabirds", kind: "sampler", sounds: "36 " + seaBirdsPath + " whole loop=1.5 name=seabirds", pan: 0.2, hall: 0.15, automated: true},
		{name: "gulls", script: "gulls", kind: "sampler", sounds: "36 " + gullsPath + " whole loop=1.5 name=gulls", pan: -0.3, hall: 0.15, automated: true},
		{name: "animals", script: "animals", kind: "sampler", sounds: "36 " + animalsPath + " whole loop=1.5 name=animals", hall: 0.15, automated: true},
		{name: "children", script: "children", kind: "sampler", sounds: "36 " + childrenPath + " whole name=children", pan: 0.1, hall: 0.25, automated: true},
		// the percussion, played by hand: hand-played recordings on a sampler each, the days' rhythms in their scripts
		{name: "shaker", script: "shaker", kind: "sampler", kit: "shaker", level: 0.8, pan: -0.5, hall: 0.25},
		{name: "tambourine", script: "tambourine", kind: "sampler", kit: "tambourine", level: 0.6, pan: 0.5, hall: 0.3},
		{name: "wood", script: "wood", kind: "sampler", kit: "wood", level: 0.9, pan: 0.3, hall: 0.35},
		{name: "hands", script: "hands", kind: "sampler", kit: "hands", level: 0.45, pan: -0.25, hall: 0.3},
		{name: "cymbal", script: "cymbal", kind: "sampler", kit: "cymbal", level: 0.55, hall: 0.5, heavens: 0.2},
	}
}

// buildPatch makes the patch: the song of the seven days, a script that holds the score (the first, so the others find it in `shared`), a script and a synth
// for every part, on a mixer with three buses: a hall, the heavens (a shimmer) and a church. It returns the scripts and the synths, to look at what they say
func buildPatch(pt *patcher.Patcher, opt options) (ids patchIDs, err error) {
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
	prelude, err := preludeJS(opt.just, opt.a4, opt.seed, opt.human)
	if err != nil {
		return patchIDs{}, err
	}
	pt.Tempo().Set(tempo)
	song := add("song", "", 20, 20)
	score := add("script", "", 20, 140) // first, so it has put the days in `shared` before the others ask
	all := parts()
	if opt.real {
		for i := range all {
			all[i] = all[i].real()
		}
	}
	type placed struct {
		strip
		script, synth, out string
	}
	var ps []placed
	for i, s := range all {
		y := 20 + float64(i)*90
		pl := placed{strip: s}
		pl.script = add("script", "", 260, y)
		pl.synth = add(s.kind, s.preset, 520, y)
		pl.out = pl.synth
		ps = append(ps, pl)
	}
	// the Word: the voice of John 1:1 (a sampler and a script of its own) says itself through the pad of open fifths in a vocoder
	wordVoice := add("sampler", "", 520, 20+float64(len(all))*90)
	wordScript := add("script", "", 260, 20+float64(len(all))*90)
	vocoder := add("vocoder", "", 780, 20+float64(len(all))*90)
	mixer := add("mixer", fmt.Sprintf("%d channels", len(all)), 1080, 300)
	hall := add("reverb", "", 1340, 20)
	heavens := add("shimmer", "", 1340, 160)
	church := add("hall", "", 1340, 300)
	if err != nil {
		return patchIDs{}, err
	}
	text(song, "sections", sectionsText())
	text(score, "code", prelude+script("score"))
	text(wordScript, "code", script("theword"))
	text(wordVoice, "sounds", "36 "+wordPath+" whole name=word vol=0")
	ids = patchIDs{mixer: mixer, scripts: []string{score, wordScript}, synths: map[string]string{}}
	channels := len(all)
	modIn := channels + 4 // the mod input of the mixer: after its channels and its four returns
	for i := range ps {
		pl := &ps[i]
		text(pl.script, "code", script(pl.strip.script))
		if pl.kit != "" && err == nil {
			pl.sounds, err = kitLines(pl.kit)
		}
		if pl.sounds != "" {
			text(pl.synth, "sounds", pl.sounds)
		}
		apply(pl.synth, pl.settings)
		connect(patcher.Cable{From: pl.script, To: pl.synth})
		ids.scripts = append(ids.scripts, pl.script)
		switch pl.chain {
		case "pitchshift": // the birds an octave above, a piccolo shadow of the flute
			fx := add("pitchshift", "", 780, 20+float64(i)*90)
			apply(fx, map[string]float64{"semitones": 12, "mix": 0.15})
			connect(patcher.Cable{From: pl.synth, To: fx})
			pl.out = fx
		case "pingpong": // the stars of the celesta, echoing
			fx := add("pingpong", "", 780, 20+float64(i)*90)
			apply(fx, map[string]float64{"time": 0.833, "feedback": 0.35, "mix": 0.25, "highCut": 6000, "lowCut": 250})
			connect(patcher.Cable{From: pl.synth, To: fx})
			pl.out = fx
		case "vocoder": // the pad is the carrier; the voice of the Word the modulator
			connect(patcher.Cable{From: wordVoice, To: vocoder, In: 0})
			connect(patcher.Cable{From: pl.synth, To: vocoder, In: 1})
			connect(patcher.Cable{From: wordScript, To: wordVoice})
			pl.out = vocoder
			ids.synths["the word voice"], ids.synths["the word pad"] = wordVoice, pl.synth
		}
		connect(patcher.Cable{From: pl.out, To: mixer, In: i})
		ids.synths[pl.name] = pl.out
		switch {
		case pl.automated: // the level follows the days: the mod output of the script (0 to 1 over the range 0 to 2 of the level)
			connect(patcher.Cable{From: pl.script, Out: 4, To: mixer, In: modIn, Key: fmt.Sprintf("level%d", i+1), Lo: 0, Hi: 1})
		case pl.strip.script == "void": // the darkness lifts: the cutoff from 300 Hz to the 1500 Hz of the drone (of 20 to 18000, exponentially)
			lo := math.Log(300.0/20) / math.Log(18000.0/20)
			hi := math.Log(1500.0/20) / math.Log(18000.0/20)
			connect(patcher.Cable{From: pl.script, Out: 4, To: pl.synth, In: 1, Key: "cutoff", Lo: lo, Hi: hi})
		}
	}
	// send A to the hall, B to the heavens, C to the church; each comes back on its return
	for i, fx := range []string{hall, heavens, church} {
		connect(patcher.Cable{From: mixer, Out: 1 + i, To: fx})
		connect(patcher.Cable{From: fx, To: mixer, In: channels + i})
	}
	connect(patcher.Cable{From: mixer, Out: 0, To: "out"})
	if err != nil {
		return patchIDs{}, err
	}
	m := map[string]float64{"returnA": 1, "returnB": 1, "returnC": 1}
	for i, pl := range ps {
		n := fmt.Sprint(i + 1)
		m["level"+n] = pl.level // 0 for a recording: its script moves it
		m["pan"+n] = pl.pan
		m["sendA"+n], m["sendB"+n], m["sendC"+n] = pl.hall, pl.heavens, pl.church
	}
	m["level"] = 0.9 // a little under the song of Go, which a limiter kept in
	apply(mixer, m)
	apply(hall, map[string]float64{"mix": 1, "decay": 4.2, "size": 0.85, "damping": 0.5})
	apply(heavens, map[string]float64{"shimmer": 0.5, "mix": 1, "decay": 8})
	apply(church, map[string]float64{"mix": 1, "gain": 2.2})
	if opt.once {
		apply(song, map[string]float64{"loop": 0})
	}
	pt.Move("out", 1600, 300)
	if err != nil {
		return patchIDs{}, err
	}
	return ids, nil
}
