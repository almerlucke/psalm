package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/synths/sampler"
)

//go:embed scripts/score.js
var scoreScript string

//go:embed scripts/rhythm.js
var rhythmScript string

//go:embed scripts/piano.js
var pianoScript string

//go:embed scripts/glass.js
var glassScript string

//go:embed scripts/harp.js
var harpScript string

//go:embed scripts/tubes.js
var tubesScript string

//go:embed scripts/pad.js
var padScript string

//go:embed scripts/bass.js
var bassScript string

//go:embed scripts/voice.js
var voiceScript string

//go:embed scripts/perc.js
var percScript string

//go:embed scripts/cue.js
var cueScript string

//go:embed scripts/psalm.js
var psalmScript string

//go:embed scripts/cuewhistle.js
var cueWhistleScript string

//go:embed scripts/whistle.js
var whistleScript string

//go:embed scripts/kit.js
var kitScript string

//go:embed scripts/hats.js
var hatsScript string

//go:embed scripts/tamb.js
var tambScript string

//go:embed scripts/shake.js
var shakeScript string

//go:embed scripts/trumpet.js
var trumpetScript string

// nominalTempo is the beats a minute the piece is written for, and tempo the one it plays at (the test plays it faster)
const nominalTempo = 66.0

var tempo = nominalTempo

// sections are the parts of the piece, in bars: the verses of Psalm 150, "Praise ye the LORD"
var sections = []struct {
	name string
	bars int
}{
	{"Sanctuary", 8}, {"Greatness", 12}, {"Dance", 16}, {"Cymbals", 12}, {"Breath", 12}, {"Hallelujah", 6},
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

// the long notes I sang (loofdeheer.wav): from and to in seconds, and the pitch each was sung at; on the keys from 24, five apart
var sungNotes = [][3]float64{
	{0.5, 1.9, 60.0}, {2.6, 4.9, 56.05}, {7.0, 9.6, 53.4}, {10.4, 12.6, 49.1}, {15.0, 17.8, 48.3},
	{20.8, 22.4, 40.95}, {23.5, 25.6, 48.2}, {27.0, 28.4, 53.2}, {29.0, 30.6, 48.2},
}

const (
	sungPath    = "assets/sounds/loofdeheer.wav"
	sungFirst   = 24
	tubesPath   = "assets/sounds/various/wooden-tubes-32138.wav"
	percPath    = "assets/sounds/various/66-bpm-uniq-percussions-264333.wav"
	tambPath    = "assets/sounds/various/tambourine-2-bars-loop-90bpm-44-46139.wav"
	cymbalPath  = "assets/sounds/various/cymbal-big2-83942.wav"
	psalmPath   = "assets/sounds/psalm_150.wav"
	whistlePath = "assets/sounds/whistle_psalm150.wav"
	shakePath   = "assets/sounds/various/spinopel-hit-the-tambourine-393065.wav"
)

// notesLine is the line of a sampler for long notes of a recording, on the keys from first, five apart
func notesLine(first int, path string, notes [][3]float64, name string, vol float64) string {
	line := fmt.Sprintf("%d %s notes", first, path)
	for _, n := range notes {
		line += fmt.Sprintf(" %g %g %g", n[0], n[1], n[2])
	}
	return fmt.Sprintf("%s name=%s vol=%g", line, name, vol)
}

func sungLine() string { return notesLine(sungFirst, sungPath, sungNotes, "voice", 5) }

// the percussion cut at its hits, a piece on each key from 36; the tambourine loop the same; the long shake of a tambourine on a few keys (the higher keys
// play it higher and shorter) and a crash of a cymbal on one
func percLine() string { return fmt.Sprintf("36 %s hits 9 0.08 0 24 name=perc vol=10", percPath) }

// the long recording of percussion holds three kinds of sounds: low hand drums in its first 24 seconds, louder deep drums from 42 to 94, and high ones (shakers,
// hats) from 94 to the end; each kind is cut by itself and has a sampler of its own
func kitLine() string  { return fmt.Sprintf("36 %s hits 9 0.08 42 94 name=kit vol=4", percPath) }
func hatsLine() string { return fmt.Sprintf("36 %s hits 9 0.08 94 130 name=hats vol=18", percPath) }
func tambLine() string { return fmt.Sprintf("36 %s hits 6 0.08 name=tamb vol=14", tambPath) }

// the psalm I sang (psalm_150.wav), played once, as it is, from Greatness on
func psalmLine() string { return "36 " + psalmPath + " whole name=psalm vol=0" }

// the psalm I whistled (whistle_psalm150.wav), played once, as it is, from Dance on
func whistleLine() string { return "36 " + whistlePath + " whole name=whistle vol=0" }

func shakeLine() string {
	return fmt.Sprintf("60 %s whole 76 60 name=shake vol=0\n84 %s whole name=cymbal vol=14", shakePath, cymbalPath)
}

// the wooden tubes cut at their hits, a piece on each key from 24
func tubesLine() string { return fmt.Sprintf("24 %s hits 8 0.15 name=tubes vol=20", tubesPath) }

var pitchName = regexp.MustCompile(`^([A-G])(#?)(\d)_`)

// pitched makes a line of the sampler for every pitch of a folder of recordings named by their note (C#5_03.wav): one recording for each pitch, on its
// key and the keys above it up to the next recording (a little higher). It returns the lines and the pitches
func pitched(dir, name string, vol float64) (string, []int, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.wav"))
	if err != nil {
		return "", nil, err
	}
	slices.Sort(files)
	semis := map[string]int{"C": 0, "D": 2, "E": 4, "F": 5, "G": 7, "A": 9, "B": 11}
	byPitch := map[int]string{}
	var pitches []int
	for _, f := range files {
		m := pitchName.FindStringSubmatch(filepath.Base(f))
		if m == nil {
			continue
		}
		oct, _ := strconv.Atoi(m[3])
		p := 12*(oct+1) + semis[m[1]]
		if m[2] == "#" {
			p++
		}
		if _, ok := byPitch[p]; ok || p > 125 {
			continue
		}
		byPitch[p] = f
		pitches = append(pitches, p)
	}
	if len(pitches) == 0 {
		return "", nil, fmt.Errorf("%s: no recordings named by their note", dir)
	}
	slices.Sort(pitches)
	var lines []string
	for i, p := range pitches {
		hi := p + 2 // the keys up to the next recording are its, a little higher
		if i+1 < len(pitches) {
			hi = pitches[i+1] - 1
		}
		lines = append(lines, fmt.Sprintf("%d %s whole %d %d name=%s vol=%g", p, byPitch[p], hi, p, name, vol))
	}
	return strings.Join(lines, "\n"), pitches, nil
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

// what the scripts know of the sounds: the pieces of the tubes (key, level, seconds), the sung notes (key, pitch, level, seconds), and the pitches of the glass and the harp
func soundsJS(glassPitches, harpPitches []int) (string, error) {
	snd, err := sampler.LoadFile(tubesPath)
	if err != nil {
		return "", err
	}
	cuts := sampler.OnsetCuts(snd, 8, 0.15)
	sr := snd.SampleRate()
	var tubes []map[string]float64
	for i := 0; i+1 < len(cuts) && 24+i <= 127; i++ {
		tubes = append(tubes, map[string]float64{"key": float64(24 + i), "sec": math.Round(float64(cuts[i+1]-cuts[i])/sr*100) / 100,
			"db": math.Round(rmsDB(snd, cuts[i], cuts[i+1]))})
	}
	vocal, err := sampler.LoadFile(sungPath)
	if err != nil {
		return "", err
	}
	var sung []map[string]float64
	for i, n := range sungNotes {
		sr := vocal.SampleRate()
		sung = append(sung, map[string]float64{"key": float64(sungFirst + 5*i), "pitch": math.Round(n[2]), "sec": n[1] - n[0],
			"db": math.Round(rmsDB(vocal, int64(n[0]*sr), int64(n[1]*sr)))})
	}
	perc, err := piecesOf(percPath, 36, 9, 0, 24)
	if err != nil {
		return "", err
	}
	kit, err := piecesOf(percPath, 36, 9, 42, 94)
	if err != nil {
		return "", err
	}
	hats, err := piecesOf(percPath, 36, 9, 94, 130)
	if err != nil {
		return "", err
	}
	tamb, err := piecesOf(tambPath, 36, 6, 0, 0)
	if err != nil {
		return "", err
	}
	js := func(v any) string { b, _ := json.Marshal(v); return string(b) }
	return "shared.sounds = " + js(map[string]any{"tubes": tubes, "sung": sung, "glass": glassPitches, "harp": harpPitches, "perc": perc, "kit": kit, "hats": hats, "tamb": tamb}) + "\n", nil
}

// piecesOf cuts a recording at its hits as the sampler does (the part of it from `from` to `to` seconds, to 0: to the end) and says of each piece the key, the
// seconds and the level
func piecesOf(path string, first int, rise, from, to float64) ([]map[string]float64, error) {
	snd, err := sampler.LoadFile(path)
	if err != nil {
		return nil, err
	}
	sr := snd.SampleRate()
	cuts := sampler.Window(sampler.OnsetCuts(snd, rise, 0.08), sr, from, to, snd.NumFrames())
	var out []map[string]float64
	for i := 0; i+1 < len(cuts) && first+i <= 127; i++ {
		out = append(out, map[string]float64{"key": float64(first + i), "sec": math.Round(float64(cuts[i+1]-cuts[i])/sr*100) / 100,
			"db": math.Round(rmsDB(snd, cuts[i], cuts[i+1]))})
	}
	return out, nil
}

// patchIDs are the modules of the patch to look at: the scripts, and the synths by their part
type patchIDs struct {
	mixer   string
	scripts []string
	synths  map[string]string
}

// buildPatch makes the patch: the song, a score script (the chords, the scale, the layers), a rhythm script that is the clock of the piano, six more scripts
// each with a clock of its own, a division of the beat of its own (eighth triplets, sixteenth triplets, quarters), a synth each, on a mixer with three buses: a hall,
// an echo and a shimmer. Rendering turns the loop of the song off so it ends with the last section. It returns the scripts, to look at what they say
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
				if e := pt.TextError(id, key); e != "" {
					err = fmt.Errorf("%s of %s: %s", key, id, e)
				}
			}
		}
	}
	glassLines, glassPitches, gerr := pitched("assets/sounds/glass", "glass", 12)
	harpLines, harpPitches, herr := pitched("assets/sounds/harp", "harp", 5)
	if gerr != nil || herr != nil {
		return patchIDs{}, fmt.Errorf("%v %v", gerr, herr)
	}
	sounds, err := soundsJS(glassPitches, harpPitches)
	if err != nil {
		return patchIDs{}, err
	}
	if _, serr := os.Stat(sungPath); serr != nil {
		return patchIDs{}, serr
	}
	pt.Tempo().Set(tempo)
	song := add("song", "", 20, 20)
	score := add("script", "", 20, 160) // first, so it has put the chords in `shared` before the others ask
	rhythm := add("script", "", 20, 300)
	cueWhistle := add("script", "", 20, 580) // the same for the whistle
	cue := add("script", "", 20, 440)        // says when the psalm I sang begins (fire), to the script of the sampler that plays it
	type part struct{ name, script, synth, kind, preset, sounds string }
	parts := []part{
		{name: "piano", kind: "piano", preset: "grand"},
		{name: "glass", kind: "sampler", sounds: glassLines},
		{name: "harp", kind: "sampler", sounds: harpLines},
		{name: "tubes", kind: "sampler", sounds: tubesLine()},
		{name: "pad", kind: "organ", preset: "diapason"},
		{name: "bass", kind: "shaper", preset: "subSine"},
		{name: "voice", kind: "sampler", sounds: sungLine()},
		{name: "perc", kind: "sampler", sounds: percLine()},
		{name: "kit", kind: "sampler", sounds: kitLine()},
		{name: "hats", kind: "sampler", sounds: hatsLine()},
		{name: "tamb", kind: "sampler", sounds: tambLine()},
		{name: "shake", kind: "sampler", sounds: shakeLine()},
		{name: "trumpet", kind: "organ", preset: "nasalReed"},
		{name: "psalm", kind: "sampler", sounds: psalmLine()},
		{name: "whistle", kind: "sampler", sounds: whistleLine()},
	}
	for i := range parts {
		y := 20 + float64(i)*100
		parts[i].script = add("script", "", 260, y)
		parts[i].synth = add(parts[i].kind, parts[i].preset, 520, y)
	}
	// the singer: the sound input (a microphone) on a strip of its own after the parts, its dry recorder before the strip, the recorder of the whole piece after the mixer
	voiceIn := add("input", "", 260, 20+float64(len(parts))*100)
	voiceRec := add("recorder", "", 520, 20+float64(len(parts))*100)
	mixRec := add("recorder", "", 1300, 300)
	singer := len(parts) // its strip (counted from 0)
	evener := add("compressor", "", 650, 20+float64(len(parts)-1)*100)
	mixer := add("mixer", fmt.Sprintf("%d channels", len(parts)+1), 780, 300)
	hall := add("reverb", "", 1040, 20)
	echo := add("pingpong", "", 1040, 140)
	shine := add("shimmer", "", 1040, 260)
	if err != nil {
		return patchIDs{}, err
	}
	text(song, "sections", songText())
	text(score, "code", sounds+scoreScript)
	text(rhythm, "code", rhythmScript)
	text(cue, "code", cueScript)
	text(cueWhistle, "code", cueWhistleScript)
	scripts := map[string]string{"piano": pianoScript, "glass": glassScript, "harp": harpScript, "tubes": tubesScript, "pad": padScript,
		"bass": bassScript, "voice": voiceScript, "perc": percScript, "kit": kitScript, "hats": hatsScript, "tamb": tambScript, "shake": shakeScript,
		"trumpet": trumpetScript, "psalm": psalmScript, "whistle": whistleScript}
	for _, s := range parts {
		text(s.script, "code", scripts[s.name])
		if s.sounds != "" {
			text(s.synth, "sounds", s.sounds)
		}
	}
	ids = patchIDs{mixer: mixer, scripts: []string{score, rhythm, cue, cueWhistle}, synths: map[string]string{}}
	for i, s := range parts {
		connect(patcher.Cable{From: s.script, To: s.synth})
		ids.scripts = append(ids.scripts, s.script)
		ids.synths[s.name] = s.synth
		if s.name == "whistle" { // the cue script says when
			connect(patcher.Cable{From: cueWhistle, Out: 3, To: s.script, In: 0})
		}
		if s.name == "psalm" {
			// the cue script says when; a compressor evens out the voice (a loud word brought down more than a soft one, then all of it up)
			connect(patcher.Cable{From: cue, Out: 3, To: s.script, In: 0})
			connect(patcher.Cable{From: s.synth, To: evener})
			connect(patcher.Cable{From: evener, To: mixer, In: i})
			ids.synths[s.name] = evener
		} else {
			connect(patcher.Cable{From: s.synth, To: mixer, In: i})
		}
		if s.name == "piano" { // the rhythm script is the clock of the piano: its triggers come from fire()
			connect(patcher.Cable{From: rhythm, Out: 3, To: s.script, In: 0})
		}
	}
	// send A to the hall, B to the echo, C to the shimmer; each comes back on its return
	for i, fx := range []string{hall, echo, shine} {
		connect(patcher.Cable{From: mixer, Out: 1 + i, To: fx})
		connect(patcher.Cable{From: fx, To: mixer, In: len(parts) + 1 + i})
	}
	connect(patcher.Cable{From: voiceIn, To: voiceRec})
	connect(patcher.Cable{From: voiceRec, To: mixer, In: singer})
	connect(patcher.Cable{From: mixer, Out: 0, To: mixRec})
	connect(patcher.Cable{From: mixRec, To: "out"})
	if err != nil {
		return patchIDs{}, err
	}
	// the mix: piano, glass, harp, tubes, pad, bass, voice, perc, kit, hats, tamb, shake, trumpet, psalm (the one I sang), whistle (the one I whistled), then the singer; the sends are the hall, the echo, the shimmer
	pt.SettingsOf(mixer).Apply(map[string]float64{
		"level1": 1.2, "pan1": -0.2, "sendA1": 0.35, "sendB1": 0.35,
		"level2": 0.6, "pan2": 0.3, "sendA2": 0.5, "sendC2": 0.3,
		"level3": 0.7, "pan3": -0.4, "sendA3": 0.4, "sendB3": 0.2,
		"level4": 0.6, "pan4": 0.4, "sendA4": 0.3, "sendB4": 0.25,
		"level5": 0.06, "pan5": 0, "sendA5": 0.4, "sendC5": 0.2,
		"level6": 0.25, "pan6": 0, "sendA6": 0.1,
		"level7": 0.7, "pan7": 0, "sendA7": 0.5, "sendC7": 0.25,
		"level8": 0.8, "pan8": -0.3, "sendA8": 0.25, "sendB8": 0.2,
		"level9": 0.7, "pan9": -0.1, "sendA9": 0.3, "sendB9": 0.2,
		"level10": 1.2, "pan10": 0.4, "sendA10": 0.2, "sendB10": 0.15,
		"level11": 1.0, "pan11": 0.35, "sendA11": 0.2, "sendB11": 0.15,
		"level12": 0.5, "pan12": 0.2, "sendA12": 0.5, "sendC12": 0.2,
		"level13": 0.12, "pan13": 0, "sendA13": 0.6, "sendB13": 0.3,
		"level14": 1.4, "pan14": 0, "sendA14": 0.3, "sendB14": 0.12, "sendC14": 0.1,
		"level15": 0.22, "pan15": 0, "sendA15": 0.38, "sendB15": 0.42, "sendC15": 0.41,
		"level16": 1, "pan16": 0, "sendA16": 0.5, "sendB16": 0.3, "sendC16": 0.1,
		"returnA": 1, "returnB": 0.7, "returnC": 0.3,
	})
	pt.SettingsOf(evener).Apply(map[string]float64{"threshold": -42, "ratio": 6, "knee": 12, "attack": 15, "release": 350, "makeup": 18, "mix": 1})
	pt.SettingsOf(hall).Apply(map[string]float64{"mix": 0.5, "decay": 3.6, "size": 0.7})
	pt.SettingsOf(echo).Apply(map[string]float64{"time": 0.75, "feedback": 0.4, "stereo": 1, "highCut": 5000, "lowCut": 150, "mix": 1})
	pt.SettingsOf(shine).Apply(map[string]float64{"shimmer": 0.25, "semitones": 7, "mix": 1, "decay": 5})
	if once {
		pt.SettingsOf(song).Apply(map[string]float64{"loop": 0})
	}
	pt.Move("out", 1560, 300)
	return ids, nil
}
