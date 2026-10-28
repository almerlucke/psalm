package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"slices"
	"strings"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
	"github.com/almerlucke/psalm/modules/effects/compressor"
	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/tools/rhythm"
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
	bpm       = 68 // a resting heart
	beats     = 4  // to the bar
	endBPM    = 54 // the last bars of the last section slow down to it
	slowBars  = 4
	heartRate = 72
)

// tempo is the beats a minute the song starts at (the test plays it faster)
var tempo = float64(bpm)

// section is a part of the psalm: its bars, how loud it plays (the dynamics of the whole), and the verse printed as it begins
type section struct {
	name  string
	bars  int
	dyn   float64
	verse string
}

var sections = []section{
	{"Thread", 8, 0.75, "O LORD, thou hast searched me, and known me. Thou knowest my downsitting and mine uprising, thou understandest my thought afar off. (Psalm 139:1, 2)"},
	{"Knit", 16, 0.8, "For thou hast possessed my reins: thou hast covered me in my mother's womb. My substance was not hid from thee, when I was made in secret, and curiously wrought in the lowest parts of the earth. (Psalm 139:13, 15)"},
	{"Wonder", 16, 1.0, "I will praise thee; for I am fearfully and wonderfully made: marvellous are thy works; and that my soul knoweth right well. (Psalm 139:14)"},
	{"Shadow", 12, 0.7, "Whither shall I go from thy spirit? or whither shall I flee from thy presence? If I say, Surely the darkness shall cover me; even the night shall be light about me. (Psalm 139:7, 11)"},
	{"Light", 12, 0.95, "Yea, the darkness hideth not from thee; but the night shineth as the day: the darkness and the light are both alike to thee. (Psalm 139:12)"},
	{"Sand", 16, 0.8, "How precious also are thy thoughts unto me, O God! how great is the sum of them! If I should count them, they are more in number than the sand. (Psalm 139:17, 18)"},
	{"Awake", 12, 0.7, "When I awake, I am still with thee. Search me, O God, and know my heart: try me, and know my thoughts: and lead me in the way everlasting. (Psalm 139:18, 23, 24)"},
}

// progressions are the chords of every section: a cycle ("chord*bars", a bar each without) that goes round for the bars of the section. The harmony module of the patch has them
// (what the melodist plays over), and the score reads them from here (shared.progs) for the parts that play the chords
var progressions = map[string]string{
	"Thread": "Dsus2*4 D*2 Dsus2*2",
	"Knit":   "D Bm7 Gmaj7 A",
	"Wonder": "G D/F# Em7 A Bm G A D",
	"Shadow": "Bm G Em7 A",
	"Light":  "D Bm7 Gmaj7 A",
	"Sand":   "D Bm7 Gmaj7 A",
	"Awake":  "Dsus2*4 D*8",
}

// harmonyText is the chords for the harmony module: a line for every section
func harmonyText() string {
	lines := make([]string, len(sections))
	for i, s := range sections {
		lines[i] = s.name + ": " + progressions[s.name]
	}
	return strings.Join(lines, "\n")
}

// sectionsText is the sections of the song for the song module
func sectionsText() string {
	parts := make([]string, len(sections))
	for i, s := range sections {
		parts[i] = fmt.Sprintf("%s %d", s.name, s.bars)
	}
	return strings.Join(parts, ", ")
}

// tempoAt is the tempo in a section, at a bar of it: the last bars of the last section slow down
func tempoAt(index int, bar float64) float64 {
	if index != len(sections)-1 {
		return bpm
	}
	from := float64(sections[index].bars - slowBars)
	p := math.Max(0, math.Min((bar-from)/slowBars, 1))
	return bpm + (endBPM-bpm)*(0.5-0.5*math.Cos(math.Pi*p))
}

// tourSeconds is how long the piece lasts, following the tempo
func tourSeconds() float64 {
	seconds := 0.0
	const step = 1.0 / 64 // of a beat
	for i, s := range sections {
		for b := 0.0; b < float64(s.bars*beats); b += step {
			seconds += step * 60 / tempoAt(i, b/beats)
		}
	}
	return seconds
}

// schedule says when every section begins
func schedule() string {
	var b strings.Builder
	seconds := 0.0
	const step = 1.0 / 64
	for i, s := range sections {
		fmt.Fprintf(&b, "%6.1fs  %-8s %2d bars\n", seconds, s.name, s.bars)
		for beat := 0.0; beat < float64(s.bars*beats); beat += step {
			seconds += step * 60 / tempoAt(i, beat/beats)
		}
	}
	fmt.Fprintf(&b, "%6.1fs  end\n", seconds)
	return b.String()
}

// tempoLane sets the tempo of the patch as the song goes: the last bars of the piece slow down, like a breath let go
type tempoLane struct {
	*psalm.Module
	song *patcher.Song
}

func (t *tempoLane) init() *tempoLane { t.SetSelf(t); return t }

func (t *tempoLane) CtrlTick(tick psalm.CtrlTick) {
	pos, ok := t.song.At(tick, t.Config().PPQ)
	if !ok || pos.Ended {
		return
	}
	want := tempoAt(pos.Section.Index, pos.Progress*float64(pos.Section.Bars))
	if math.Abs(want-t.Env().BPM()) >= 0.01 {
		t.Env().SetBPM(want)
	}
}

// orchestra is where the Virtual Playing Orchestra is (it is not in the repository: without it models of the instruments play)
const orchestra = "assets/Virtual-Playing-Orchestra3"

func hasOrchestra() bool {
	_, err := os.Stat(orchestra + "/Vocals/choir-FEMALE-sustain.sfz")
	return err == nil
}

type options struct {
	just  bool
	a4    float64
	seed  uint64 // the seed of the feel of the parts
	human bool   // the parts play off the grid, like players
	real  bool   // the instruments the orchestra has recorded play its recordings instead of the models
	once  bool   // the song ends with its last section, it doesn't loop
}

// strip is a part of the piece: its script, the instrument that plays it (a recording of the orchestra, or a model: a synth of the kind and the preset) and its place
// on the mixer. The sends are the hall, the church (a convolved room), the heavens (a shimmer) and the echo
type strip struct {
	name, script string
	kind, preset string  // the model
	sfz          string  // the recording (in the orchestra); "" for a part that only has a model or its sounds
	gain         float64 // what the level is multiplied by when the recording plays (they are not as loud as the models)
	model        float64 // what the level is multiplied by when the model plays (0: not at all; found by rendering both and comparing the parts)
	sounds       string  // the sounds of a sampler or a grains synth
	kit          string  // the percussion of a sampler: its recordings, see kits.go
	glass        string  // the folder of recordings named by their note, a sampler with one for each pitch
	settings     map[string]float64
	level        float64
	pan          float64
	hall, church float64
	heavens      float64
	echo         float64
	insert       string             // an effect between the instrument and the mixer (a compressor for the voice)
	insertSet    map[string]float64 // its settings
	direct       bool               // not on the mixer: the voice goes round it, and ducks the music (see duckKind)
}

func parts() []strip {
	ps := []strip{
		{name: "heart", script: "heart", kind: "dx", preset: "timpani", level: 0.07, hall: 0.25, church: 0.25},
		{name: "harp", script: "harp", kind: "waveguide", preset: "celticHarp", sfz: "Strings/harp-sustain.sfz", gain: 1.8, model: 0.53, level: 0.42, pan: -0.3, hall: 0.3, church: 0.25, echo: 0.12},
		{name: "celesta", script: "celesta", kind: "dx", preset: "celesta", sfz: "Keys/celesta.sfz", gain: 1.3, model: 0.074, level: 0.86, pan: 0.35, heavens: 0.45, echo: 0.3},
		{name: "flute", script: "flute", kind: "waveguide", preset: "flute", sfz: "Woodwinds/flute-SOLO-sustain.sfz", gain: 1.2, model: 0.2, level: 0.5, pan: -0.15, hall: 0.25, church: 0.35},
		{name: "alto flute", script: "altoflute", kind: "waveguide", preset: "altoFlute", sfz: "Woodwinds/alto-flute-SOLO-sustain.sfz", gain: 1.2, model: 0.133, level: 0.5, pan: 0.2, hall: 0.25, church: 0.35},
		{name: "violin", script: "violin", kind: "waveguide", preset: "legatoViolin", sfz: "Strings/1st-violin-SOLO-sustain.sfz", gain: 1.4, model: 0.85, level: 0.5, pan: 0.3, hall: 0.25, church: 0.4},
		{name: "cello", script: "cello", kind: "waveguide", preset: "cello", sfz: "Strings/cello-SOLO-sustain.sfz", gain: 1.3, model: 0.45, level: 0.5, pan: -0.3, hall: 0.25, church: 0.35},
		{name: "strings", script: "strings", kind: "waveguide", preset: "slowStrings", sfz: "Strings/viola-SEC-sustain.sfz", gain: 1.3, model: 0.11, level: 0.56, hall: 0.3, church: 0.45},
		{name: "choir", script: "choir", kind: "vocal", preset: "hummingChoir", sfz: "Vocals/choir-FEMALE-sustain.sfz", gain: 1.3, model: 0.17, level: 0.56, hall: 0.1, church: 0.7, heavens: 0.2},
		{name: "piano", script: "piano", kind: "piano", preset: "satie", level: 0.8, pan: 0.1, hall: 0.3, church: 0.15},
		{name: "pizzicato", script: "pizzicato", kind: "dx", preset: "pizzicato", sfz: "Strings/all-strings-SEC-pizzicato.sfz", gain: 1.7, model: 0.1, level: 0.8, pan: -0.2, hall: 0.3, echo: 0.3},
		{name: "shaker", script: "shaker", kind: "sampler", kit: "shaker", level: 0.9, pan: -0.45, hall: 0.25},
		{name: "glass", script: "glass", kind: "sampler", glass: "assets/sounds/glass", level: 0.9, pan: 0.35, hall: 0.35, church: 0.2, echo: 0.2},
		{name: "hands", script: "hands", kind: "sampler", kit: "hands", level: 0.15, pan: -0.2, hall: 0.3, church: 0.15},
		{name: "cymbal", script: "cymbal", kind: "sampler", kit: "cymbal", level: 0.8, hall: 0.4, heavens: 0.3},
	}
	if _, err := os.Stat(voicePath); err == nil { // the recording of the psalm, when it is there: played from the start of the Knit
		ps = append(ps, strip{name: "voice", script: "voice", kind: "streamer", sounds: fmt.Sprintf("36 %s vol=%g name=voice", voicePath, voiceGain), level: 0.8, direct: true,
			insert: "compressor", insertSet: map[string]float64{"threshold": -26, "ratio": 8, "knee": 6, "attack": 2, "release": 180, "makeup": 11}})
	}
	return ps
}

// voicePath is the recording of the psalm that is played once from the start of the Knit (a streamer reads it from the disk: it is long); voiceGain is how much louder than it was
// recorded it plays (dB)
const (
	voicePath = "assets/sounds/psalm139_voice.wav"
	voiceGain = 7.0
	voiceWet  = 0.06 // how much of the voice goes through the church
)

// real is the strip played by a recording of the orchestra: a sampler with the sfz of the part, in place of its model
func (s strip) real() strip {
	if s.sfz == "" {
		return s
	}
	s.kind, s.preset, s.settings = "sampler", "", nil
	s.sounds = "0 " + orchestra + "/" + s.sfz + " sfz"
	s.level *= s.gain
	return s
}

// patchIDs are the modules of the patch to look at: the scripts, and the synths by their part
type patchIDs struct {
	scripts  []string
	synths   map[string]string
	live     []string // the modules of the live parts (see live)
	melodist string
}

type sectionJS struct {
	Name  string  `json:"name"`
	Bars  int     `json:"bars"`
	Dyn   float64 `json:"dyn"`
	Verse string  `json:"verse"`
}

// preludeJS is what the first script gets before its own text: the sections, the tuning, the seed of the random timing, the keys of the kits and the pieces of the cut
// recordings
func preludeJS(opt options) (string, error) {
	var ss []sectionJS
	for _, s := range sections {
		ss = append(ss, sectionJS{s.name, s.bars, s.dyn, s.verse})
	}
	js := func(v any) string { b, _ := json.Marshal(v); return string(b) }
	cutPieces, err := piecesJS()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("shared.beats = %d\nshared.just = %v\nshared.a4 = %v\nshared.seed = %d\nshared.human = %v\nshared.sections = %s\nshared.kits = %s\nshared.progs = %s\n",
		beats, opt.just, opt.a4, opt.seed, opt.human, js(ss), js(kitKeys()), js(progressions)) + cutPieces, nil
}

// buildPatch makes the patch: the song of the psalm, a script that holds the score (the first, so the others find it in `shared`), a script and an instrument for
// every part, on a mixer with four buses: a hall, a church, the heavens (a shimmer) and an echo. It returns the scripts and the instruments, to look at what they say
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
	prelude, err := preludeJS(opt)
	if err != nil {
		return patchIDs{}, err
	}
	pt.Tempo().Set(tempo)
	song := add("song", "", 20, 20)
	score := add("script", "", 20, 140) // first, so it has put the harmony in `shared` before the others ask
	all := parts()
	for i := range all {
		if opt.real {
			all[i] = all[i].real()
		} else if all[i].model > 0 {
			all[i].level *= all[i].model
		}
	}
	type placed struct {
		strip
		script, synth string
	}
	var ps []placed
	for i, s := range all {
		y := 20 + float64(i)*90
		pl := placed{strip: s}
		pl.script = add("script", "", 260, y)
		pl.synth = add(s.kind, s.preset, 520, y)
		ps = append(ps, pl)
	}
	mixer := add("mixer", fmt.Sprintf("%d channels", len(all)), 800, 300)
	hall := add("reverb", "", 1060, 20)
	church := add("hall", "", 1060, 160)
	heavens := add("shimmer", "", 1060, 300)
	echo := add("pingpong", "", 1060, 440)
	if err != nil {
		return patchIDs{}, err
	}
	text(song, "sections", sectionsText())
	text(score, "code", prelude+script("score"))
	ids = patchIDs{scripts: []string{score}, synths: map[string]string{}}
	channels := len(all)
	voice := "" // the voice, when the recording is there: it goes round the mixer
	for i := range ps {
		pl := &ps[i]
		text(pl.script, "code", script(pl.strip.script))
		if pl.kit != "" && err == nil {
			pl.sounds, err = kitLines(pl.kit)
		}
		if pl.glass != "" && err == nil {
			pl.sounds, err = pitched(pl.glass, "glass")
		}
		if pl.sounds != "" {
			text(pl.synth, "sounds", pl.sounds)
		}
		apply(pl.synth, pl.settings)
		connect(patcher.Cable{From: pl.script, To: pl.synth})
		if pl.insert != "" { // the instrument through an effect on its way to the mixer
			fx := add(pl.insert, "", 660, 20+float64(i)*90)
			apply(fx, pl.insertSet)
			connect(patcher.Cable{From: pl.synth, To: fx})
			if pl.direct {
				voice = fx
			} else {
				connect(patcher.Cable{From: fx, To: mixer, In: i})
			}
			ids.synths[pl.name+" "+pl.insert] = fx
		} else {
			connect(patcher.Cable{From: pl.synth, To: mixer, In: i})
		}
		ids.scripts = append(ids.scripts, pl.script)
		ids.synths[pl.name] = pl.synth
	}
	// send A to the hall, B to the church, C to the heavens, D to the echo; each comes back on its return
	for i, fx := range []string{hall, church, heavens, echo} {
		connect(patcher.Cable{From: mixer, Out: 1 + i, To: fx})
		connect(patcher.Cable{From: fx, To: mixer, In: channels + i})
	}
	if voice == "" {
		connect(patcher.Cable{From: mixer, Out: 0, To: "out"})
	} else {
		// the music goes through a duck that the voice drives: while the voice speaks the music is a few decibels lower, so the voice is always on top of it. The voice has a church of
		// its own (the same room as the music's, a little of it)
		duck := add("duck", "", 1320, 440)
		church2 := add("hall", "", 1060, 580)
		apply(duck, map[string]float64{"threshold": -34, "ratio": 8, "knee": 10, "attack": 30, "release": 700, "sideCut": 150})
		apply(church2, map[string]float64{"mix": voiceWet, "gain": 2.2})
		connect(patcher.Cable{From: mixer, Out: 0, To: duck})
		connect(patcher.Cable{From: voice, To: duck, In: 1})
		connect(patcher.Cable{From: voice, To: church2})
		connect(patcher.Cable{From: church2, To: "out"})
		connect(patcher.Cable{From: duck, To: "out"})
		ids.synths["music"], ids.synths["duck"], ids.synths["voice church"] = mixer, duck, church2
	}
	if err == nil {
		err = addLive(pt, &ids, opt, add, connect, text, apply)
	}
	if err != nil {
		return patchIDs{}, err
	}
	m := map[string]float64{"returnA": 1, "returnB": 1, "returnC": 0.8, "returnD": 0.8}
	for i, pl := range ps {
		n := fmt.Sprint(i + 1)
		m["level"+n] = pl.level
		m["pan"+n] = pl.pan
		m["sendA"+n], m["sendB"+n], m["sendC"+n], m["sendD"+n] = pl.hall, pl.church, pl.heavens, pl.echo
	}
	m["level"] = 1.5
	apply(mixer, m)
	apply(hall, map[string]float64{"mix": 1, "decay": 3.4, "size": 0.8, "damping": 0.55})
	apply(church, map[string]float64{"mix": 1, "gain": 2.2})
	apply(heavens, map[string]float64{"shimmer": 0.35, "mix": 1, "decay": 7})
	apply(echo, map[string]float64{"time": 0.75, "feedback": 0.4, "stereo": 1, "highCut": 4500, "lowCut": 200, "mix": 1})
	if opt.once {
		apply(song, map[string]float64{"loop": 0})
	}
	pt.Move("out", 1320, 300)
	if err != nil {
		return patchIDs{}, err
	}
	return ids, nil
}

func b2f(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// handKeys are the keys of the pieces cut from the recording of the hands (the ones loud enough to use), as the script of the hands has them
func handKeys() []int {
	c, ok := cutByName("hands")
	if !ok {
		return nil
	}
	ps, err := c.pieces()
	if err != nil {
		return nil
	}
	var keys []int
	for _, p := range ps {
		if p.DB > c.lowest {
			keys = append(keys, p.Key)
		}
	}
	return keys
}

// addLive adds the parts that are made while the song plays, under the written ones (their scripts are not touched), into the instruments the parts have:
//   - a harmony with the chords of the song, and a melodist that thinks aloud on the glass: a short phrase now and then over the chord of the bar, a question and its answer, a
//     little different every time through a section (the thought of the Wonder comes again in the Light, changed), in the Wonder, the Light and the Sand
//   - ghosts of the hands: a hand drum of three hits in twelve turning a step every pass (so the pattern weaves itself round the bar), softer and later than the written hands, with
//     a lane of five velocities against the twelve (the accents never fall in the same place twice), a laid back groove and a drift, in the Wonder, the Light and the Sand
//   - the sand: in the Sand the eggs of a cellular automaton, a new row every pass (rule 30): the grains that are more in number than the sand
func addLive(pt *patcher.Patcher, ids *patchIDs, opt options, add func(kind, preset string, x, y float64) string, connect func(patcher.Cable), text func(id, key, value string),
	apply func(id string, v map[string]float64)) error {
	glass, hands, shaker := ids.synths["glass"], ids.synths["hands"], ids.synths["shaker"]
	if glass == "" || hands == "" || shaker == "" {
		return fmt.Errorf("the live parts need the glass, the hands and the eggs")
	}
	major := float64(slices.Index(patcher.Scales, "major"))
	harmony := add("harmony", "", 1500, 20)
	melodist := add("melodist", "", 1500, 140)
	gate := add("gate", "", 1740, 140)
	text(harmony, "chords", harmonyText())
	apply(harmony, map[string]float64{"key": 2, "scale": major}) // D major
	text(gate, "name", "thoughts")
	text(gate, "sections", "Wonder, Light, Sand")
	apply(melodist, map[string]float64{"form": 4, "contour": 4, "phrase": 1, "density": 0.1, "rest": 0.75, "steps": 0.8, "chord": 0.8, "low": 74, "high": 92,
		"mutate": 0.35, "seed": 11, "velocity": 0.3, "just": b2f(opt.just)})
	connect(patcher.Cable{From: melodist, To: gate, In: 1})
	connect(patcher.Cable{From: gate, Out: 1, To: glass})

	ghost := func(y float64, synth, pattern, lanes string, keys []int, settings map[string]float64, velocity float64) {
		clock := add("clock", "", 1500, y)
		seq := add("sequence", "", 1740, y)
		notes := add("notes", "", 1980, y)
		text(clock, "pattern", pattern)
		text(clock, "lanes", lanes)
		vals := make([]string, len(keys))
		for i, k := range keys {
			vals[i] = fmt.Sprint(k)
		}
		text(seq, "values", strings.Join(vals, " "))
		apply(clock, settings)
		apply(seq, map[string]float64{"order": 1}) // shuffled: every piece in turn, not the same twice in a row
		apply(notes, map[string]float64{"velocity": velocity, "length": 1.5})
		connect(patcher.Cable{From: clock, To: seq})
		connect(patcher.Cable{From: clock, To: notes})
		connect(patcher.Cable{From: seq, To: notes, In: 1})
		connect(patcher.Cable{From: notes, To: synth})
		ids.live = append(ids.live, clock, seq, notes)
	}
	laidBack := float64(slices.Index(rhythm.GrooveNames(), "laid back"))
	ghost(300, hands, "Wonder: E(2,12,+1)\nLight: E(2,12,1+1)\nSand: E(3,12,2+1)", "vel: 100 55 80 50 70\nprob: 100 100 85", handKeys(),
		map[string]float64{"division": 3, "groove": laidBack, "drift": 0.04, "mutate": 0.12, "seed": 4}, 0.3)
	ghost(420, shaker, "Sand: CA(30,16,+)", "vel: 100 50 70\nshift: 0 6 -4 3 0", kitKeys()["shaker"]["egg"],
		map[string]float64{"division": 3, "groove": laidBack, "drift": 0.03}, 0.2)
	ids.live = append(ids.live, harmony, melodist, gate)
	ids.melodist = melodist
	return nil
}

// duckKind is a compressor of which the sidechain drives the compression (the voice turns the music down), as a kind of the patcher: the sound on its first input, the one that
// turns it down on the second
func duckKind() patcher.Kind {
	return patcher.Kind{Name: "duck", Group: "Effects",
		Ins:  []patcher.Port{{Name: "in", Type: patcher.Audio, Index: 0}, {Name: "sidechain", Type: patcher.Audio, Index: 1}},
		Outs: []patcher.Port{{Name: "out", Type: patcher.Audio}},
		Make: func(string) (*patcher.Made, error) {
			m := compressor.New(compressor.DefaultConfig())
			return &patcher.Made{Module: m, Describe: func(string) params.Set { return compressor.Describe(m.Settings()) }}, nil
		}}
}

// kinds are the kinds of the patcher of the song: those of the catalog and the duck
func kinds(cfg catalog.Config) []patcher.Kind { return append(catalog.Kinds(cfg), duckKind()) }
