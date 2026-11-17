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
)

// tempo is the beats a minute the song starts at (the test plays it faster)
var tempo = float64(bpm)

type options struct {
	just  bool
	a4    float64
	seed  uint64 // the seed of the feel of the parts
	human bool   // the parts play off the grid, swung, like players
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
	"flute":        {"Woodwinds/flute-SOLO-sustain.sfz", 1.2},
	"upright bass": {"Strings/bass-SOLO-pizzicato.sfz", 2},
	"vibes":        {"Percussion/vibraphone-auto-damp.sfz", 8.4},
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

// patchIDs are the modules of the patch to look at: the scripts, and the synths by their part (index: the place of the part on the mixer)
type patchIDs struct {
	mixer   string
	scripts []string
	synths  map[string]string
	index   map[string]int
}

// strip is a part of the piece: its script, the synth that plays it and its place on the mixer. The level is the amplitude of the part divided by the velocity
// its notes have (0.8): the scripts play with the velocity, the level makes it as loud as it was meant (the word is the vocoder's, which says how loud it is by the voice)
type strip struct {
	name, script, kind, preset string
	sounds                     string             // the sounds of a sampler
	kit                        string             // the percussion of a sampler: its recordings, see kits.go
	settings                   map[string]float64 // of the synth, after it is made
	level                      float64
	pan                        float64
	hall, heavens              float64 // the sends: the hall and the heavens (a shimmer)
	chain                      string  // "": the synth goes to the strip; else the effect it goes through: "vocoder"
	automated                  bool    // its level is moved by a cable from its script (a recording faded in and out)
}

func parts() []strip {
	amp := func(a float64) float64 { return a / 0.8 }
	return []strip{
		{name: "void", script: "void", kind: "wavetable", preset: "drone", level: amp(0.111) * 0.7, hall: 0.4},
		{name: "the word", script: "word", kind: "wavetable", preset: "pad", chain: "vocoder", level: 0.85, hall: 0.6, heavens: 0.05,
			settings: map[string]float64{"position": 1, "positionLFODepth": 0, "filter": 0, "unison": 3}},
		// the waters of the intro, going round while their note is held, their level follows the bars (a cable from the script to the strip)
		{name: "waves", script: "waves", kind: "sampler", sounds: "36 " + wavesPath + " whole loop=1.5 name=waves", hall: 0.3, automated: true},
		// the rest of the recordings of creation play like percussion: cut at their hits, a piece on each key, played on euclidean rhythms (see kits.go)
		{name: "sea and birds", script: "seabirds", kind: "sampler", kit: "sea and birds", level: 0.45, pan: 0.25, hall: 0.5, heavens: 0.125},
		{name: "animals", script: "animals", kind: "sampler", kit: "animals", level: 0.4, pan: -0.2, hall: 0.4, heavens: 0.05},
		{name: "gulls", script: "gulls", kind: "sampler", kit: "gulls", level: 0.75, pan: -0.35, hall: 0.5, heavens: 0.15},
		{name: "children", script: "children", kind: "sampler", kit: "children", level: 0.5, pan: 0.15, hall: 0.4, heavens: 0.075},
		// the combo, in the pure ratios of the rest: a walking bass, a piano comping, a vibraphone, a flute leading with a steel pan after it, and the percussion keeping the waltz
		{name: "upright bass", script: "jazzbass", kind: "dx", preset: "woodBass", level: amp(0.25) * 0.65, pan: -0.15, hall: 0.12},
		{name: "piano", script: "comp", kind: "piano", preset: "grand", level: amp(0.2) * 0.8, pan: -0.05, hall: 0.25},
		{name: "vibes", script: "vibes", kind: "dx", preset: "mallet", level: amp(0.05), pan: 0.2, hall: 0.35, heavens: 0.05},
		{name: "bells", script: "bells", kind: "dx", preset: "musicBox", level: amp(0.12) * 1.5, pan: 0.15, hall: 0.45, heavens: 0.05},
		{name: "marimba", script: "answer", kind: "modal", preset: "marimba", level: 0.5, pan: -0.25, hall: 0.35, heavens: 0.05},
		{name: "kalimba", script: "kalimba", kind: "modal", preset: "kalimba", level: 0.5, pan: 0.3, hall: 0.4, heavens: 0.05},
		{name: "flute", script: "flute", kind: "dx", preset: "flute", level: amp(0.21), pan: -0.15, hall: 0.4, heavens: 0.05},
		{name: "steel pan", script: "steelpan", kind: "modal", preset: "steelPan", level: 0.3, pan: -0.4, hall: 0.35, heavens: 0.075},
		// the percussion of the islands, played by hand: a sampler each with the hand-played recordings of its instrument, in the ratios of the days' rhythms
		{name: "darbuka", script: "darbuka", kind: "sampler", kit: "darbuka", level: 0.55, pan: 0.05, hall: 0.2},
		{name: "congas", script: "congas", kind: "sampler", kit: "congas", level: 0.6, pan: 0.3, hall: 0.25},
		{name: "bongos", script: "bongos", kind: "sampler", kit: "bongos", level: 0.45, pan: -0.3, hall: 0.25},
		{name: "claves", script: "claves", kind: "sampler", kit: "claves", level: 0.4, pan: 0.45, hall: 0.2},
		{name: "shakers", script: "shakers", kind: "sampler", kit: "shakers", level: 0.7, pan: -0.5, hall: 0.25, heavens: 0.075},
	}
}

// buildPatch makes the patch: the song of the seven days, a script that holds the score (the first, so the others find it in `shared`), a script and a synth
// for every part, on a mixer with two buses: a hall and the heavens (a shimmer). It returns the scripts and the synths, to look at what they say
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
	cutPieces, err := piecesJS() // the pieces of the recordings that play like percussion
	if err != nil {
		return patchIDs{}, err
	}
	prelude += cutPieces
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
	if err != nil {
		return patchIDs{}, err
	}
	text(song, "sections", sectionsText())
	text(score, "code", prelude+script("score"))
	text(wordScript, "code", script("theword"))
	syllables, kerr := kitLines("word") // the whole verse on a key of its own (35), the syllables of it from 36
	if kerr != nil {
		return patchIDs{}, kerr
	}
	text(wordVoice, "sounds", "35 "+wordPath+" whole name=wordall vol=0\n"+syllables)
	ids = patchIDs{mixer: mixer, scripts: []string{score, wordScript}, synths: map[string]string{}, index: map[string]int{}}
	channels := len(all)
	modIn := channels + 4 // the mod input of the mixer: after its channels and its four returns (three buses are used)
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
		case "vocoder": // the pad is the carrier; the voice of the Word the modulator
			connect(patcher.Cable{From: wordVoice, To: vocoder, In: 0})
			connect(patcher.Cable{From: pl.synth, To: vocoder, In: 1})
			connect(patcher.Cable{From: wordScript, To: wordVoice})
			// an echo on the dotted eighth (at 92 beats a minute) behind the voice, so the syllables trail off into the hall
			echo := add("pingpong", "", 1040, 20+float64(len(all))*90)
			apply(echo, map[string]float64{"time": 0.489, "feedback": 0.35, "mix": 0.3, "highCut": 5000, "lowCut": 250})
			connect(patcher.Cable{From: vocoder, To: echo})
			pl.out = echo
			ids.synths["the word voice"], ids.synths["the word pad"] = wordVoice, pl.synth
		}
		connect(patcher.Cable{From: pl.out, To: mixer, In: i})
		ids.synths[pl.name] = pl.out
		ids.index[pl.name] = i
		switch {
		case pl.automated: // the level follows the days: the mod output of the script (0 to 1 over the range 0 to 2 of the level)
			connect(patcher.Cable{From: pl.script, Out: 4, To: mixer, In: modIn, Key: fmt.Sprintf("level%d", i+1), Lo: 0, Hi: 1})
		case pl.strip.script == "void" && pl.kind == "wavetable": // the darkness lifts: the cutoff from 300 Hz to the 1500 Hz of the drone (of 20 to 18000, exponentially)
			lo := math.Log(300.0/20) / math.Log(18000.0/20)
			hi := math.Log(1500.0/20) / math.Log(18000.0/20)
			connect(patcher.Cable{From: pl.script, Out: 4, To: pl.synth, In: 1, Key: "cutoff", Lo: lo, Hi: hi})
		}
	}
	// send A to the hall, B to the heavens; each comes back on its return
	for i, fx := range []string{hall, heavens} {
		connect(patcher.Cable{From: mixer, Out: 1 + i, To: fx})
		connect(patcher.Cable{From: fx, To: mixer, In: channels + i})
	}
	// the master: a low shelf and a gentle compressor glue the mix together and make it louder, before the output
	masterEQ, masterComp := add("eq", "", 1600, 160), add("compressor", "", 1840, 160)
	connect(patcher.Cable{From: mixer, Out: 0, To: masterEQ})
	connect(patcher.Cable{From: masterEQ, To: masterComp})
	connect(patcher.Cable{From: masterComp, To: "out"})
	apply(masterEQ, map[string]float64{"lowFreq": 110, "lowGain": 3, "lowCut": 25})
	apply(masterComp, map[string]float64{"threshold": -20, "ratio": 2.5, "knee": 8, "attack": 25, "release": 250, "makeup": 5, "mix": 1})
	if err != nil {
		return patchIDs{}, err
	}
	m := map[string]float64{"returnA": 1, "returnB": 0.4}
	for i, pl := range ps {
		n := fmt.Sprint(i + 1)
		m["level"+n] = pl.level // 0 for a recording: its script moves it
		m["pan"+n] = pl.pan
		m["sendA"+n], m["sendB"+n] = pl.hall, pl.heavens
	}
	m["level"] = 0.9 // a little under the song of Go, which a limiter kept in
	apply(mixer, m)
	apply(hall, map[string]float64{"mix": 1, "decay": 4.2, "size": 0.85, "damping": 0.5})
	apply(heavens, map[string]float64{"shimmer": 0.35, "mix": 1, "decay": 6})
	if opt.once {
		apply(song, map[string]float64{"loop": 0})
	}
	pt.Move("out", 2080, 300)
	if err != nil {
		return patchIDs{}, err
	}
	return ids, nil
}
