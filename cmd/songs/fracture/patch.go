package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/almerlucke/psalm/modules/controls/panel/chordmodel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/tools/rhythm"
)

const (
	bpm   = 124
	beats = 4
)

// section is a part of the piece: its bars
type section struct {
	name string
	bars int
}

var sections = []section{{"Static", 8}, {"Fracture", 16}, {"Machine", 24}, {"Collapse", 8}, {"Drift", 16}, {"Residue", 8}}

func sectionsText() string {
	parts := make([]string, len(sections))
	for i, s := range sections {
		parts[i] = fmt.Sprintf("%s %d", s.name, s.bars)
	}
	return strings.Join(parts, ", ")
}

// harmonyText is the chords of the sections that have their own: the Machine and the Drift have none, they get the chords the chord model draws, new every time the song leaves a
// section (the others line of the harmony)
const harmonyText = `Static: Csus2*2 C5*2 Dbsus2*2 Cm7b5*2
Fracture: Cm7b5 Dbmaj7 Fdim7 Gbsus4 Bbm7 Ab7b9
Collapse: Cdim7*2 Dbsus2*2 Gbmaj7*2 Fm7b5*2
Residue: Csus2*4 C5*4`

// ids are the modules of the patch that tests look at
type ids struct {
	clocks    map[string]string // the clocks of the rhythm parts, by name
	notes     map[string]string // the notes modules that follow them
	chords    string
	toybox    []string // the melodists of the toy box
	grooves   []string // the parts of the groove that builds
	harmony   string
	synths    map[string]string
	musicOnly []string
}

// strip is a voice of the piece: an instrument at a place on the mixer; an insert is an effect between the two
type strip struct {
	name, kind, preset string
	level, pan         float64
	reverb, echo, grit float64 // the sends: the room, the ping pong, the granulator
	air                float64 // the shimmer
	insert             string
	insertSet          map[string]float64
	cut                *cutSound // a sampler that plays the pieces of a recording
}

func strips() []strip {
	all := []strip{
		// the drums go through an eq that takes the sizzle off the top
		{name: "drums", kind: "drumsynth", preset: "industrial", level: 0.3, reverb: 0.08, grit: 0.06, insert: "eq", insertSet: map[string]float64{"highFreq": 4500, "highGain": -9, "highCut": 9000}},
		{name: "glitch", kind: "drumsynth", preset: "electro", level: 1.0, pan: 0.15, echo: 0.3, grit: 0.35, insert: "crusher", insertSet: map[string]float64{"bits": 8, "rate": 9000, "mix": 0.55, "output": 6}},
		{name: "clicks", kind: "pulsar", preset: "crackle", level: 0.5, pan: -0.25, reverb: 0.15, echo: 0.2},
		{name: "bass", kind: "pulsar", preset: "buzzBass", level: 0.45, grit: 0.05},
		{name: "pad", kind: "subtractive", preset: "warmPad", level: 0.3, reverb: 0.35, air: 0.4},
		{name: "toybox", kind: "dx", preset: "musicBox", level: 0.3, pan: 0.25, reverb: 0.45, echo: 0.5, air: 0.3},
		{name: "groove", kind: "drumsynth", preset: "808", level: 0.55, pan: -0.1, reverb: 0.12, echo: 0.12},
		{name: "metal", kind: "modal", preset: "anvil", level: 0.35, pan: -0.35, reverb: 0.3, echo: 0.35, air: 0.2},
	}
	// the samplers: a blurry vocal cut into pieces that stutter and fall over each other, and the strokes of a tabla loop cut up and played in any order, both through the crusher
	for _, c := range []*cutSound{&vocalCut, &tablaCut} {
		if !c.there() {
			continue
		}
		s := strip{name: c.name, kind: "sampler", cut: c, level: 0.7, reverb: 0.2, echo: 0.35, grit: 0.4, insert: "crusher", insertSet: map[string]float64{"bits": 9, "rate": 18000, "mix": 0.5}}
		if c == &tablaCut {
			s.pan, s.level = -0.15, 0.6
		} else {
			s.pan, s.level = 0.3, 1.1 // the vocal chops, louder
		}
		all = append(all, s)
	}
	return all
}

// buildPatch makes the piece. Nothing in it repeats: the rhythms are made while it plays (a euclidean rhythm that turns every pass, cellular automata that run on, a Markov rhythm drawn
// again every so many passes), mutate takes hits away and puts others in, lanes of other lengths than the bar (5, 7, 3 steps against 16) drift against each other, the chords are drawn
// again by a model each time the song leaves a section, and a random source moves the bit depth of the crusher and the feedback of the echo
func buildPatch(pt *patcher.Patcher, seed int) (out ids, err error) {
	out = ids{clocks: map[string]string{}, notes: map[string]string{}, synths: map[string]string{}}
	add := func(kind, preset string, x, y float64) string {
		if err != nil {
			return ""
		}
		var id string
		id, err = pt.Add(kind, preset, x, y)
		return id
	}
	connect := func(c patcher.Cable) string {
		if err != nil {
			return ""
		}
		var id string
		id, err = pt.Connect(c)
		return id
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
	pt.Tempo().Set(bpm)

	// the song, its chords and the model that draws more
	song := add("song", "", 20, 20)
	harm := add("harmony", "", 20, 130)
	model := add("chordmodel", "", 20, 260)
	text(song, "sections", sectionsText())
	text(harm, "chords", harmonyText)
	phrygian := float64(slices.Index(patcher.Scales, "phrygian"))
	apply(harm, map[string]float64{"key": 0, "scale": phrygian}) // C phrygian
	out.harmony = harm
	out.chords = model
	if err == nil {
		if m, ok := pt.Module(model).(*chordmodel.Module); ok {
			m.SetKey(0, 5) // C, phrygian
			m.Preset("minor")
			m.SetTarget("")        // the lines that are no section's: the Machine and the Drift
			m.SetLive(true, 0)     // drawn again every time the song leaves a section
			m.Reroll(uint64(seed)) // not the same progression as the last time
			m.Bake()
		}
	}

	// the instruments on a mixer with four buses: a room, a ping pong, a granulator and a shimmer
	all := strips()
	synth := map[string]string{}
	for i, s := range all {
		synth[s.name] = add(s.kind, s.preset, 1100, 20+float64(i)*90)
		out.synths[s.name] = synth[s.name]
		if s.cut != nil {
			text(synth[s.name], "sounds", s.cut.line())
		}
	}
	channels := 16
	mixer := add("mixer", fmt.Sprintf("%d channels", channels), 1500, 300)
	room := add("reverb", "", 1760, 20)
	echo := add("pingpong", "", 1760, 160)
	grit := add("granulator", "", 1760, 300)
	air := add("shimmer", "", 1760, 440)
	if err != nil {
		return out, err
	}
	m := map[string]float64{"returnA": 0.9, "returnB": 0.7, "returnC": 0.6, "returnD": 0.6, "level": 0.8}
	for i, s := range all {
		n := fmt.Sprint(i + 1)
		var to string = synth[s.name]
		if s.insert != "" {
			fx := add(s.insert, "", 1300, 20+float64(i)*90)
			apply(fx, s.insertSet)
			connect(patcher.Cable{From: synth[s.name], To: fx})
			to = fx
			out.synths[s.name+" "+s.insert] = fx
		}
		connect(patcher.Cable{From: to, To: mixer, In: i})
		m["level"+n], m["pan"+n] = s.level, s.pan
		m["sendA"+n], m["sendB"+n], m["sendC"+n], m["sendD"+n] = s.reverb, s.echo, s.grit, s.air
	}
	for i, fx := range []string{room, echo, grit, air} {
		connect(patcher.Cable{From: mixer, Out: 1 + i, To: fx})
		connect(patcher.Cable{From: fx, To: mixer, In: channels + i})
	}
	connect(patcher.Cable{From: mixer, Out: 0, To: "out"})
	apply(mixer, m)
	apply(room, map[string]float64{"mix": 1, "decay": 2.2, "size": 0.7})
	apply(echo, map[string]float64{"time": 0.75, "feedback": 0.45, "stereo": 1, "highCut": 6000, "lowCut": 250, "mix": 1})
	apply(grit, map[string]float64{"density": 40, "grainSize": 60, "jitter": 0.7, "spray": 120, "pitchSpread": 5, "mix": 1})
	apply(air, map[string]float64{"shimmer": 0.5, "mix": 1, "decay": 4})

	// the sources of change: a random value moves the bit depth of the crusher, a slow wave the feedback of the echo
	rnd := add("random", "", 1300, 700)
	slow := add("lfo", "", 1300, 780)
	apply(rnd, map[string]float64{"beats": 3, "smooth": 0.1})
	apply(slow, map[string]float64{"beats": 23})
	if id := connect(patcher.Cable{Type: patcher.Mod, From: rnd, To: out.synths["glitch crusher"], In: 1, Key: "bits"}); id != "" {
		err = pt.SetRange(id, "", 0.2, 0.55)
	}
	if id := connect(patcher.Cable{Type: patcher.Mod, From: slow, To: echo, In: 1, Key: "feedback"}); id != "" && err == nil {
		err = pt.SetRange(id, "", 0.25, 0.9)
	}

	// the rhythm parts: a clock, a notes module and, for the ones that play different sounds, a sequence that says which
	division := map[string]float64{"1/16": 3, "1/32": 5}
	dilla := float64(slices.Index(rhythm.GrooveNames(), "dilla"))
	rhythmPart := func(name, to string, y float64, div, pattern, lanes string, keys []int, order float64, settings map[string]float64, velocity float64) {
		clock := add("clock", "", 300, y)
		notes := add("notes", "", 780, y)
		text(clock, "pattern", pattern)
		if lanes != "" {
			text(clock, "lanes", lanes)
		}
		set := map[string]float64{"division": division[div], "seed": float64(seed % 1000)}
		for k, v := range settings {
			set[k] = v
		}
		apply(clock, set)
		apply(notes, map[string]float64{"velocity": velocity, "length": 0.8})
		connect(patcher.Cable{From: clock, To: notes})
		if len(keys) > 1 {
			seq := add("sequence", "", 540, y)
			vals := make([]string, len(keys))
			for i, k := range keys {
				vals[i] = fmt.Sprint(k)
			}
			text(seq, "values", strings.Join(vals, " "))
			apply(seq, map[string]float64{"order": order})
			connect(patcher.Cable{From: clock, To: seq})
			connect(patcher.Cable{From: seq, To: notes, In: 1})
		} else if len(keys) == 1 {
			apply(notes, map[string]float64{"pitch": float64(keys[0])})
		}
		connect(patcher.Cable{From: notes, To: synth[name]})
		out.clocks[name+" "+to] = clock
		out.notes[name+" "+to] = notes
		out.musicOnly = append(out.musicOnly, clock, notes)
	}
	const shuffled, random = 1, 2
	// the kick: a heart in the Static, a euclidean rhythm that turns a step every pass in the Fracture, a floor of four that mutates (what has a ! stays) in the Machine, an
	// automaton in the Collapse, three turning in the Drift
	rhythmPart("drums", "kick", 20, "1/16", `Static: x . . . . . . . . . . . . . . .
Fracture: E(5,16,+1)
Machine: x! . . . x! . . . x! . . . x! . . .
Collapse: CA(90,16,+)
Drift: E(3,16,+1)`, "vel: 100 80 90", []int{36}, 0, map[string]float64{"mutate": 0.3, "drift": 0.03}, 0.95)
	rhythmPart("drums", "snare", 90, "1/16", `Fracture: . . . . x . . . . . . . x . [x x x]%fill .
Machine: M(. . . . x . . . . . . . x . . x;16;2/2)
Collapse: CA(110,16,+)
Drift: . . . . x%2/4 . . . . . . . x . . .`, "vel: 100 70", []int{38}, 0, map[string]float64{"mutate": 0.25, "groove": dilla}, 0.85)
	rhythmPart("drums", "hats", 160, "1/16", `Fracture: CA(30,16,+)
Machine: E(11,16,+1)
Collapse: CA(30,16,3+1)
Drift: M(x . x . x x . x . x . x x . x .;16;7/3)`, "prob: 100 70 90 50 100\nvel: 100 60 80 50 90 70 60\nshift: 0 8 -5 0 10 -8 0 4 0", []int{42, 42, 42, 42, 44}, random,
		map[string]float64{"mutate": 0.2, "groove": dilla, "drift": 0.06}, 0.55)
	// the glitch: thirty-seconds of an automaton (rule 110, then 30), a euclidean rhythm that turns three steps a pass, a Markov rhythm drawn again every two passes; lanes of 7, 4
	// and 7 steps against the 32; the sounds in any order
	rhythmPart("glitch", "ticks", 230, "1/32", `Fracture: CA(110,32,+)
Machine: CA(30,32,+)
Collapse: E(13,32,+3)
Drift: M(x . . x . x . . . x . . . . x .;32;5/2)`, "prob: 100 35 80 20 90 50 100\nvel: 100 50 70 40\nshift: 0 12 -10 5 0 0 15", []int{37, 39, 42, 46, 48, 50, 56, 60, 63, 75}, random,
		map[string]float64{"mutate": 0.3, "drift": 0.1}, 0.6)
	rhythmPart("clicks", "dust", 300, "1/32", `Static: M(x . . . . . x . . . . . . x . .;32;3/1)
Fracture: M(x . . x . . . x . . . . x . . .;32;11/1)
Residue: M(x . . . . . . . . . . . . x . .;32;13/1)`, "vel: 100 40 70", []int{60, 72, 84, 67}, random, map[string]float64{"mutate": 0.2}, 0.4)
	rhythmPart("metal", "hits", 370, "1/16", `Machine: E(3,16,+1)%2/4
Collapse: E(5,16,+2)
Fracture: CA(90,16,+)%2/3`, "vel: 100 60 80 50 90", []int{60, 63, 66, 69, 61, 72}, shuffled, map[string]float64{"mutate": 0.15, "drift": 0.05}, 0.55)

	// the samplers: the pieces of a recording in any order, a hit and a roll of them (x3) that stutter and fall over each other, lanes of 5 and 3 steps for the chance and the length
	if vocalCut.there() {
		// the vocal chops are played by the chords: a part of notes relative to the chord of the moment (@1 the root, @3 and @5 the tones above it, ' an octave up), each note one of
		// the syllables, tuned to the tone (a sung syllable is on the pitch it is played at), chosen at random; the figure mutates and the chords change as the song goes
		chops := add("part", "", 300, 440)
		text(chops, "notes", `Static: . . . . . . . . . . @5 . . . . .
Fracture: . . @3 . . @5 . @1' . @3'^2 . . @5 . . .
Machine: @1 . @5' . @3 . . @1' . . @5 . @3'^3 . . @1'
Collapse: @5 . . @3' . . @1' . @5 . . . @3 . @5'^2 .
Drift: @3 . . @5' . . @1' . . . @3' . . @5 . .
Residue: . . . . . . . . . . . . . . @1' .`)
		text(chops, "lanes", "prob: 100 60 90 40 80\nvel: 100 60 80\nlen: 100 40 100 60 100 30 100")
		apply(chops, map[string]float64{"division": 3, "velocity": 0.75, "length": 0.5, "register": 43, "mutate": 0.25, "seed": float64(seed % 1000), "groove": dilla, "drift": 0.06})
		connect(patcher.Cable{From: chops, To: synth[vocalCut.name]})
		out.notes["chops part"] = chops
		out.musicOnly = append(out.musicOnly, chops)
	}
	if !tablaCut.there() {
	} else if keys, kerr := tablaCut.keys(); kerr == nil && len(keys) > 1 {
		rhythmPart("slices", "strokes", 510, "1/16", `Fracture: M(. . x . . x . . . x . . x . . .;16;6/2)
Machine: E(9,32,+3)
Collapse: CA(90,32,+)
Drift: M(x . . x . x . . . x . . . . x .;32;2/1)`, "prob: 100 50 80\nvel: 100 70 85 60\nshift: 0 10 -8 0 6", keys, random, map[string]float64{"mutate": 0.3, "groove": dilla, "drift": 0.05}, 0.65)
	} else if kerr != nil {
		err = kerr
	}

	// the bass and the pad play the chords: notes written relative to them (@b the bass, @1 the root, @5 the fifth), a figure that mutates; the pad holds the chord voice led from the
	// one before
	bass := add("part", "", 300, 460)
	pad := add("part", "", 300, 530)
	text(bass, "notes", `Fracture: @b . . @1 . . @b . . . @5 . . @b . @1'
Machine: @b @b . @b . @b^2 . . @1 . @b . @b . @5%2/4 .
Collapse: @b*4 . . . . @b*4 . . . .
Drift: @b . . . @b . . @1 . . @b . . . @5 .`)
	text(bass, "lanes", "pitch: 0 0 0 12 0 -12 0\nvel: 100 70 85 60 90")
	apply(bass, map[string]float64{"division": 3, "velocity": 0.8, "length": 0.6, "register": 36, "bass": 28, "mutate": 0.18, "seed": float64(seed % 1000), "groove": dilla, "drift": 0.04})
	text(pad, "notes", `Static: @v*16
Collapse: @v*16
Drift: @v*16
Residue: @v*16`)
	apply(pad, map[string]float64{"division": 3, "velocity": 0.5, "length": 1, "register": 48})
	connect(patcher.Cable{From: bass, To: synth["bass"]})
	connect(patcher.Cable{From: pad, To: synth["pad"]})
	out.musicOnly = append(out.musicOnly, bass, pad)

	// the toy box: a melancholic melody on a music box, over the chords of the moment (the tones of the chord count most, the scale is the harmony's, phrygian), in three layers that
	// come in and go as the sections go: the tune alone in the Static; a lower answer under it in the Fracture; a glittering high layer on top in the Machine; nothing in the Collapse
	// (the automata alone); the tune and the glitter in the Drift; the tune alone, thinning, in the Residue. Each layer is a melodist: a pass a section, the pass of the section before
	// changed a little (the tune is the tune again and a little other), and a form of a question and its answer (the question ends open, on the third or the fifth of the chord, the answer closes on the root), the tune falling, a sigh, and the answer under it rising
	layer := func(name string, y float64, sections string, set map[string]float64) {
		m := add("melodist", "", 300, y)
		g := add("gate", "", 540, y)
		text(g, "name", name)
		text(g, "sections", sections)
		base := map[string]float64{"form": 3, "contour": 1, "rhythm": 0, "phrase": 2, "steps": 0.85, "chord": 0.85, "velocity": 0.5, "key": 0, "scale": 0}
		for k, v := range set {
			base[k] = v
		}
		apply(m, base)
		connect(patcher.Cable{From: m, To: g, In: 1})
		connect(patcher.Cable{From: g, Out: 1, To: synth["toybox"]})
		out.toybox = append(out.toybox, m)
		out.musicOnly = append(out.musicOnly, m, g)
	}
	layer("tune", 640, "Static, Fracture, Machine, Drift, Residue", map[string]float64{"form": 4, "contour": 3, "steps": 0.92, "velocity": 0.42, "rhythm": 1, "grid": 1.5, "density": 0.1, "rest": 0.4, "low": 70, "high": 88, "mutate": 0.12, "seed": float64(seed % 10000)})
	layer("answer", 700, "Fracture, Machine", map[string]float64{"form": 4, "contour": 2, "steps": 0.92, "rhythm": 1, "grid": 2, "density": 0.1, "rest": 0.55, "low": 60, "high": 79, "mutate": 0.3, "velocity": 0.36, "seed": float64(seed%10000 + 17)})
	layer("glitter", 760, "Machine, Drift", map[string]float64{"grid": 0.5, "density": 0.2, "rest": 0.55, "low": 84, "high": 103, "mutate": 0.6, "velocity": 0.35, "rhythm": 3, "seed": float64(seed%10000 + 31)})
	// the groove that builds the tune up: an 808 kick, hats, a rim and a conga, that come in as one curve rises (a lane of the song: from nothing in the Static to full in the Machine, away
	// in the Collapse, back and fading in the Drift and the Residue). The curve moves the chance of every hit (more and more of the pattern is played) and the velocity (louder), so the
	// groove grows fuller and louder as the sections go
	build := add("lane", "", 300, 840)
	text(build, "curves", `Static: 0 → 0.2 smooth
Fracture: 0.2 → 0.65 smooth
Machine: 0.65 → 1 smooth
Collapse: 1 → 0.3
Drift: 0.3 → 0.7 smooth
Residue: 0.7 → 0 smooth`)
	mpc58 := float64(slices.Index(rhythm.GrooveNames(), "mpc 58"))
	grow := func(part string, chanceLo, chanceHi float64) {
		clock, notes := out.clocks[part], out.notes[part]
		if id := connect(patcher.Cable{Type: patcher.Mod, From: build, Out: 1, To: clock, In: 0, Key: "chance"}); id != "" && err == nil {
			err = pt.SetRange(id, "", chanceLo, chanceHi)
		}
		if id := connect(patcher.Cable{Type: patcher.Mod, From: build, Out: 1, To: notes, In: 4, Key: "velocity"}); id != "" && err == nil {
			err = pt.SetRange(id, "", 0.25, 0.8)
		}
	}
	gset := map[string]float64{"groove": mpc58, "mutate": 0.1, "drift": 0.03}
	rhythmPart("groove", "kick", 900, "1/16", `Static: x . . . . . . . . . . . . . . .
Fracture: x . . . . . . x . . x . . . . .
Machine: x . . . . . x . . . x . . . . x
Drift: x . . . . . . x . . x . . . . .
Residue: x . . . . . . . . . . . . . . .`, "vel: 100 80 90", []int{36}, 0, gset, 0.8)
	rhythmPart("groove", "hats", 960, "1/16", `Fracture: . . x . . . x . . . x . . . x .
Machine: x . x . x . x . x . x . x . x .
Drift: . . x . . . x . . . x . . . x .`, "vel: 100 60 80 50", []int{42, 42, 44}, random, gset, 0.6)
	rhythmPart("groove", "rim", 1020, "1/16", `Fracture: . . . . . . . . x . . . . . . .
Machine: . . . . x . . . . . . . x . . .
Drift: . . . . . . . . x . . . . . . .`, "", []int{37}, 0, gset, 0.7)
	rhythmPart("groove", "congas", 1080, "1/16", `Machine: . . x . . x . . x . . . x . x .
Drift: . . . x . . . . . x . . . . . .`, "vel: 100 70 85", []int{62, 63, 64, 60}, random, gset, 0.6)
	grow("groove kick", 0.5, 1)
	grow("groove hats", 0, 0.9)
	grow("groove rim", 0.1, 0.9)
	grow("groove congas", 0, 0.8)
	out.grooves = []string{"groove kick", "groove hats"}
	return out, err
}
