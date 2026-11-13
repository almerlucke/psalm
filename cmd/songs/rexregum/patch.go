package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
)

//go:embed scripts/score.js
var scoreScript string

//go:embed scripts/pedal.js
var pedalScript string

//go:embed scripts/low.js
var lowScript string

//go:embed scripts/high.js
var highScript string

//go:embed scripts/pulse.js
var pulseScript string

//go:embed scripts/fx.js
var fxScript string

// tempo is the bpm of the piece, in quarter notes (the test plays it faster)
var tempo = 66.0

// midiFile is the piece: 14 tracks in 3/4, D major after a G minor start, 49 bars
const midiFile = "assets/midi/rexregum.mid"

// sections are the parts of the piece, in bars of 3/4. The file has a new chord every seven bars (the choir holds it), and
// the entrances fall on the seams: the lament at bar 7, the drums and the chimes at 14, the theme at 21, the canon at 28
var sections = []struct {
	name string
	bars int
}{
	{"Dawn", 7}, {"Lament", 7}, {"Gathering", 7}, {"Theme", 7}, {"Canon", 7}, {"Glory", 7}, {"Throne", 7},
}

// roles are the parts of the score: the channels of the file, played by a synth each. The six choir parts are two layers, the
// three lower voices and the three higher
var roles = map[int]string{
	2: "pedal", 4: "low", 5: "low", 6: "low", 7: "high", 8: "high", 9: "high",
	14: "celesta", 3: "lament", 11: "timpani", 1: "theme", 13: "canon", 12: "chimes",
}

// sixteenth is the grid of the file in ticks (480 to a quarter note): every note starts and ends on it
const sixteenth = 120

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

// score reads the file and gives the JavaScript the score script starts with: `const SCORE = {pedal: [[start, pitch, length,
// velocity], ...]}` (only the pedal: the pulse is made from it, the other parts are played from the file by the midiplayer) with start and length in sixteenths from the start (a bar of the 3/4 is twelve) and the velocity 0-127, and
// `const SECTIONS = [[name, bars], ...]`. A note that starts off the grid, or a file that is not as long as the sections, is an error
func score(path string) (string, error) {
	f, err := midi.ReadFile(path)
	if err != nil {
		return "", err
	}
	if f.Division != 4*sixteenth {
		return "", fmt.Errorf("%s: %d ticks to a quarter note, not %d", path, f.Division, 4*sixteenth)
	}
	type open struct {
		role  string
		index int
	}
	var (
		all     = map[string][][4]float64{}
		sounds  = map[[2]int][]open{} // channel, key -> the notes that sound, the first struck first (a key struck again before it is let go: each let go ends the first)
		endTick int64
	)
	for _, t := range f.Tracks {
		for _, e := range t.Events {
			m := e.Message
			if !m.IsChannel() {
				continue
			}
			role, ok := roles[m.Channel()]
			if !ok {
				continue
			}
			switch m.Kind() {
			case midi.NoteOn:
				if e.Tick%sixteenth != 0 {
					return "", fmt.Errorf("%s: a note of %s at tick %d is off the sixteenth grid", path, role, e.Tick)
				}
				k := [2]int{m.Channel(), m.Note()}
				sounds[k] = append(sounds[k], open{role, len(all[role])})
				all[role] = append(all[role], [4]float64{float64(e.Tick / sixteenth), float64(m.Note()), 0, float64(m.Velocity())})
			case midi.NoteOff:
				k := [2]int{m.Channel(), m.Note()}
				if q := sounds[k]; len(q) > 0 {
					n := &all[q[0].role][q[0].index]
					n[2] = float64(e.Tick)/sixteenth - n[0] // the length may be a part of a sixteenth (the staccato of the theme)
					sounds[k] = q[1:]
					endTick = max(endTick, e.Tick)
				}
			}
		}
	}
	for _, q := range sounds {
		if len(q) > 0 {
			return "", fmt.Errorf("%s: a note that does not end", path)
		}
	}
	bars := 0
	for _, s := range sections {
		bars += s.bars
	}
	if want := int64(bars * 12 * sixteenth); endTick != want {
		return "", fmt.Errorf("%s: the notes end at tick %d, the sections are %d long", path, endTick, want)
	}
	for role, list := range all {
		for _, n := range list {
			if n[2] <= 0 {
				return "", fmt.Errorf("%s: a note of %s with no length", path, role)
			}
		}
	}
	// the notes are played from the file by the midiplayer; the script only holds the pedal, which the pulse follows
	data, err := json.Marshal(map[string][][4]float64{"pedal": all["pedal"]})
	if err != nil {
		return "", err
	}
	secs := make([][2]any, len(sections))
	for i, s := range sections {
		secs[i] = [2]any{s.name, s.bars}
	}
	sd, _ := json.Marshal(secs)
	return "const SCORE = " + string(data) + "\nconst SECTIONS = " + string(sd) + "\n", nil
}

// patchIDs are the modules of the patch to look at: the scripts, and the synths by their part
type patchIDs struct {
	scripts []string
	synths  map[string]string
	looper  string // the audio looper of the live patch ("" when rendering)
	sources string // the source mixer in front of it
}

// orchestra is where the Virtual Playing Orchestra is (it is not in the repository: without it the synths play)
const orchestra = "assets/Virtual-Playing-Orchestra3"

func hasOrchestra() bool {
	_, err := os.Stat(orchestra + "/Vocals/choir-MIXED-sustain.sfz")
	return err == nil
}

// fadedIn are the parts that come in slowly at the start of the piece, over fadeBars bars: the pedal and the choir (the recordings and the
// pads sound at once, and the piece began heavy)
var fadedIn = map[string]bool{"pedal": true, "low": true, "high": true}

// realPart is a recording of the orchestra (a sfz file, in the orchestra) for a part, and the level it plays at (the recordings
// differ in loudness)
type realPart struct {
	sfz  string
	gain float64
	// a modelled instrument in place of a recording: the kind of synth and its preset (a waveguide flute), with no sfz
	kind, preset string
}

// realParts are the parts that play recordings of real instruments (a sampler) when the orchestra is there, each the instrument that
// fits what the file asks of it. The joins (fx: a hit and a riser) have no instrument, they stay a synth
//
//	pedal      the double basses, sustained: the low voice of twelve seconds and more
//	low        the mixed choir, for the lower three voices (G2 to E5)
//	high       the sopranos, for the higher three (D4 to E6)
//	celesta    the celesta, which the file asks for
//	lament     the english horn, the voice of a lament: one line, slow, C4 to A4
//	timpani    the timpani, struck
//	theme      the harp, plucked: the tune of the King
//	canon      the first violins, staccato: the answer a fifth and an octave above
//	chimes     the glockenspiel, for the bells that ring over the Gathering and on
//	pulse      the double basses, short: the eighths under the pulse
var realParts = map[string]realPart{
	"pedal":   {sfz: "Strings/bass-SEC-sustain.sfz", gain: 0.75},
	"low":     {sfz: "Vocals/choir-MIXED-sustain.sfz", gain: 1.5},
	"high":    {sfz: "Vocals/choir-FEMALE-sustain.sfz", gain: 1.1},
	"celesta": {sfz: "Keys/celesta.sfz", gain: 2},
	"lament":  {sfz: "Woodwinds/english-horn-SOLO-sustain.sfz", gain: 2.3},
	"timpani": {sfz: "Percussion/timpani-hit.sfz", gain: 1.1},
	"theme":   {sfz: "Strings/harp-sustain.sfz", gain: 1.1},
	"canon":   {sfz: "Strings/1st-violin-SEC-staccato.sfz", gain: 0.9},
	"chimes":  {sfz: "Percussion/glockenspiel.sfz", gain: 4.5},
	"pulse":   {sfz: "Strings/bass-SEC-staccato.sfz", gain: 2},
}

// tuning is where each synth starts from its preset: the settings of the sound of the piece, by the keys of the synth's page (the real parts, recordings,
// have none). high also moves its cutoff section by section (its script)
var tuning = map[string]map[string]float64{
	"pedal": {"ampAttack": 2200, "ampDecay": 400, "ampSustain": 1, "ampRelease": 3200, "sub": 0.3, "trim": -3, "mono": 0},
	"low": {"ampAttack": 2800, "ampDecay": 400, "ampSustain": 1, "ampRelease": 3600, "unison": 5, "spread": 22, "uniStereo": 0.9, "drift": 0.4,
		"trim": -4, "mono": 0},
	"high": {"ampAttack": 3500, "ampDecay": 400, "ampSustain": 1, "ampRelease": 4500, "position": 0.35, "positionLFORate": 0.08, "positionLFODepth": 0.35,
		"unison": 5, "detune": 16, "spread": 0.9, "resonance": 0.15, "filterEnvAmount": 0.4, "cutoff": 1700, "level": 0.16},
	"celesta": {"ampRelease": 3000, "trim": -2, "mono": 0},
	// a triangle, a slide of ninety milliseconds, a vibrato that comes in after most of a second, a filter half closed
	"lament": {"osc1Waveform": 2, "osc1Level": 1, "osc2Level": 0, "subLevel": 0, "noiseLevel": 0, "ringModLevel": 0, "fmAmount": 0, "sync": 0, "drive": 0,
		"unison": 2, "unisonDetune": 8, "glide": 90, "filterType": 0, "cutoff": 1800, "resonance": 0.1, "filterEnvAmount": 0.6, "keyTrack": 0.3,
		"lfo1Waveform": 0, "lfo1Rate": 5, "lfo1Pitch": 0.14, "lfo1Delay": 700, "lfo1KeySync": 1, "ampAttack": 140, "ampDecay": 300, "ampSustain": 0.8,
		"ampRelease": 1400, "level": 0.5},
	// a sine thump that falls in pitch, a long round tail
	"timpani": {"osc1Waveform": 0, "osc1Level": 1, "osc2Level": 0, "subLevel": 0, "noiseLevel": 0, "ringModLevel": 0, "fmAmount": 0, "sync": 0, "drive": 0,
		"unison": 1, "glide": 0, "filterEnvToPitch": 7, "cutoff": 900, "resonance": 0, "filterEnvAmount": 0, "ampAttack": 3, "ampDecay": 700, "ampSustain": 0,
		"ampRelease": 500, "level": 0.4},
	// a triangle with a sine an octave under it, a little wide, a filter that opens softly on every note
	"theme": {"osc1Waveform": 2, "osc1Level": 1, "osc2Level": 0, "subLevel": 0.25, "noiseLevel": 0, "ringModLevel": 0, "fmAmount": 0, "sync": 0, "drive": 0,
		"unison": 3, "unisonDetune": 9, "drift": 5, "cutoff": 3200, "resonance": 0.1, "filterEnvAmount": 0.8, "keyTrack": 0.5, "ampAttack": 12,
		"ampDecay": 250, "ampSustain": 0.7, "ampRelease": 350, "level": 0.45},
	// a sine with a second sine an octave up and a little apart
	"canon": {"osc1Waveform": 0, "osc1Level": 1, "osc2Waveform": 0, "osc2Level": 0.35, "osc2Octave": 1, "osc2Detune": 0.08, "subLevel": 0, "noiseLevel": 0,
		"ringModLevel": 0, "fmAmount": 0, "sync": 0, "drive": 0, "unison": 2, "unisonDetune": 10, "cutoff": 6000, "resonance": 0, "filterEnvAmount": 0.3,
		"ampAttack": 8, "ampDecay": 300, "ampSustain": 0.5, "ampRelease": 500, "level": 0.4},
	"chimes": {"ampRelease": 2500, "level": 0.4},
	"pulse":  {"cutoff": 420, "resonance": 0.25, "filterEnvAmount": 2, "ampRelease": 160},
}

// buildPatch makes the patch: the song, a score script that holds the notes of the file, and a clock of sixteenths into eleven
// scripts that each play a synth, all on a mixer with three buses: a hall, an echo and a shimmer. The timpani pull down the
// pulse and the pads (a sidechain). Rendering turns the loop of the song off so it ends with the last section. It returns the
// scripts, to look at what they say
func buildPatch(pt *patcher.Patcher, once, real bool) (ids patchIDs, err error) {
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
	data, err := score(midiFile)
	if err != nil {
		return patchIDs{}, err
	}
	pt.Tempo().Set(tempo)
	song := add("song", "", 20, 20)
	scoreID := add("script", "", 20, 160) // first, so it has put the score in `shared` before the others ask
	clock := add("clock", "", 20, 300)
	player := add("midiplayer", "", 20, 440) // plays the file in step with the song, a channel to a synth
	type part struct {
		name, script, kind, preset string
	}
	// the parts: each script plays a synth, each synth goes straight to a channel of the mixer. The presets are where each sound
	// starts; the scripts tune them (set) to the sounds of the piece
	parts := []part{
		{"pedal", pedalScript, "shaper", "throneDrone"},
		{"low", lowScript, "shaper", "crystalSea"},
		{"high", highScript, "wavetable", "evolvingPad"},
		{"celesta", "", "shaper", "glassBell"},
		{"lament", "", "subtractive", "pad"},
		{"timpani", "", "subtractive", "pad"},
		{"theme", "", "subtractive", "pad"},
		{"canon", "", "subtractive", "pad"},
		{"chimes", "", "subtractive", "ePiano"},
		{"pulse", pulseScript, "subtractive", "pulseBass"},
		{"fx", fxScript, "shaper", "impact"},
	}
	index := map[string]int{}
	for i, p := range parts {
		index[p.name] = i
	}
	timpani := index["timpani"]
	// the sidechain: the timpani pull down the parts that have a compressor, each by its ratio (the pulse the most). Their sound
	// goes through the compressor on its way to the mixer
	ducked := map[string]float64{"pulse": 2.4, "high": 1.5, "low": 1.3}
	// the low cut (Hz) of the parts that carry an offset or more low than they need
	lowCut := map[string]float64{"high": 40, "pulse": 35, "pedal": 28}
	// the mix, by the channels in the order of the parts: pedal, low, high, celesta, lament, timpani, theme, canon, chimes, pulse, fx
	mix := map[string]float64{
		"level1": 0.55, "pan1": 0, "sendA1": 0.3,
		"level2": 0.4, "pan2": -0.15, "sendA2": 0.35, "sendC2": 0.25,
		"level3": 0.32, "pan3": 0.15, "sendA3": 0.35, "sendC3": 0.35,
		"level4": 0.6, "pan4": 0.3, "sendA4": 0.4, "sendB4": 0.3,
		"level5": 0.6, "pan5": -0.2, "sendA5": 0.4, "sendB5": 0.25,
		"level6": 0.9, "pan6": 0, "sendA6": 0.25,
		"level7": 0.95, "pan7": -0.1, "sendA7": 0.3, "sendB7": 0.2,
		"level8": 0.8, "pan8": 0.3, "sendA8": 0.3, "sendB8": 0.4, "sendC8": 0.2,
		"level9": 0.6, "pan9": 0.2, "sendA9": 0.4, "sendC9": 0.35,
		"level10": 0.6, "pan10": 0,
		"level11": 0.6, "pan11": 0, "sendA11": 0.6,
		"returnA": 1, "returnB": 0.7, "returnC": 0.5, "level": 0.7,
	}
	if real { // the recordings are not as loud as the synths: each by its gain
		for i, p := range parts {
			if rp, ok := realParts[p.name]; ok {
				mix[fmt.Sprintf("level%d", i+1)] *= rp.gain
			}
		}
	}
	scripts := make([]string, len(parts))
	synths := make([]string, len(parts))
	for i, p := range parts {
		y := 20 + float64(i)*110
		if p.script != "" {
			scripts[i] = add("script", "", 260, y)
		}
		if rp, ok := realParts[p.name]; ok && real {
			parts[i].kind, parts[i].preset = "sampler", ""
			if rp.kind != "" {
				parts[i].kind, parts[i].preset = rp.kind, rp.preset
			}
			p = parts[i]
		}
		synths[i] = add(p.kind, p.preset, 520, y)
		if _, rec := realParts[p.name]; err == nil && !(rec && real) { // the synth is tuned to the sound of the piece; a recording has no settings
			pt.SettingsOf(synths[i]).Apply(tuning[p.name])
		}
	}
	mixer := add("mixer", "16 channels", 780, 300)
	// the three buses of the mixer: a hall, an echo of a dotted eighth and a shimmer
	hall := add("reverb", "", 1040, 20)
	echo := add("pingpong", "", 1040, 140)
	shine := add("shimmer", "", 1040, 260)
	if err != nil {
		return patchIDs{}, err
	}
	text(song, "sections", songText())
	text(player, "file", midiFile)
	if err == nil {
		pt.SettingsOf(player).Apply(map[string]float64{"sync": 1, "lift": 0.35})
	}
	text(scoreID, "code", data+scoreScript)
	ids = patchIDs{scripts: []string{scoreID}, synths: map[string]string{}}
	for i, p := range parts {
		if p.script != "" {
			ids.scripts = append(ids.scripts, scripts[i])
		}
		code := p.script
		if rp, ok := realParts[p.name]; ok && real {
			// a recording has no settings to tune: the scripts only play it (`tune` does nothing; with a synth it is `set`); its level is the gain
			code = "const tune = () => {}\n" + code
			if rp.sfz != "" {
				text(synths[i], "sounds", "0 "+orchestra+"/"+rp.sfz+" sfz")
			}
		}
		if code == p.script {
			code = "const tune = set\n" + code
		}
		if fadedIn[p.name] { // the drones come in over the first bars: the script moves the level of its strip, up to what the mix says
			code = fmt.Sprintf("const BASE = %v\n", mix[fmt.Sprintf("level%d", i+1)]) + code
		}
		if p.script != "" {
			text(scripts[i], "code", code)
			connect(patcher.Cable{From: clock, To: scripts[i]})
			connect(patcher.Cable{From: scripts[i], To: synths[i]})
		}
		for _, ch := range slices.Sorted(maps.Keys(roles)) { // the notes of the part come from its channels of the file
			if roles[ch] == p.name {
				connect(patcher.Cable{From: player, Out: ch, To: synths[i]})
			}
		}
		// on its way to the mixer a sound may go through an eq (a low cut that takes out what is too low to hear, and the offset
		// of the wave) and a compressor (the sidechain)
		last := synths[i]
		if hz, ok := lowCut[p.name]; ok {
			eq := add("eq", "", 650, 20+float64(i)*110)
			connect(patcher.Cable{From: last, To: eq})
			if err == nil {
				pt.SettingsOf(eq).Apply(map[string]float64{"lowCut": hz})
			}
			last = eq
		}
		if ratio, ok := ducked[p.name]; ok {
			comp := add("compressor", "", 780, 20+float64(i)*110)
			connect(patcher.Cable{From: last, To: comp})
			connect(patcher.Cable{From: synths[timpani], To: comp, In: 1})
			last = comp
			if err == nil {
				pt.SettingsOf(comp).Apply(map[string]float64{"threshold": -26, "ratio": ratio, "knee": 8, "attack": 4, "release": 260, "sideCut": 30, "mix": 1})
			}
		}
		connect(patcher.Cable{From: last, To: mixer, In: i})
		if fadedIn[p.name] { // the level of the strip follows the mod output of the script (0 to 1 over the range 0 to 2 of the level)
			connect(patcher.Cable{From: scripts[i], Out: 4, To: mixer, In: 16 + 4, Key: fmt.Sprintf("level%d", i+1), Lo: 0, Hi: 1})
		}
		ids.synths[p.name] = last // the stem of a part is its sound as it goes to the mixer
	}
	// send A to the hall, B to the echo, C to the shimmer; each comes back on its return (after the 16 channels)
	for i, fx := range []string{hall, echo, shine} {
		connect(patcher.Cable{From: mixer, Out: 1 + i, To: fx})
		connect(patcher.Cable{From: fx, To: mixer, In: 16 + i})
	}
	connect(patcher.Cable{From: mixer, Out: 0, To: "out"})
	if err != nil {
		return patchIDs{}, err
	}
	// the clock counts sixteenths (a bar of the 3/4 is twelve): its default
	_ = clock
	for i, p := range parts { // the faded parts start silent: the script raises the level (BASE is what the mix says)
		if fadedIn[p.name] {
			mix[fmt.Sprintf("level%d", i+1)] = 0
		}
	}
	pt.SettingsOf(mixer).Apply(mix)
	// the hall: a big one, for a piece of this size
	pt.SettingsOf(hall).Apply(map[string]float64{"mix": 0.7, "decay": 4.5, "size": 0.8})
	// the echo: a dotted eighth (0.75 of a beat), a few repeats, with the top left on
	pt.SettingsOf(echo).Apply(map[string]float64{"time": 0.75, "feedback": 0.35, "stereo": 1, "highCut": 5000, "lowCut": 200, "mix": 1})
	// the shimmer: the reverb with an octave up in it, all of it wet
	pt.SettingsOf(shine).Apply(map[string]float64{"shimmer": 0.3, "semitones": 12, "mix": 1, "decay": 6})
	if once {
		pt.SettingsOf(song).Apply(map[string]float64{"loop": 0})
	}
	if !once {
		// playing live there is a looper to record on top of the piece: a source mixer chooses what is recorded (its strip 1 is the piece, its strip 2 the
		// sound input of the computer: the side panel of the input has the choice of the device), the looper records it into a loop of bars on the beat of
		// the patch, layer on layer, and plays the layers on. The sound that goes through the looper (its input level) is what you hear of the source: 0, so
		// the microphone does not sing back from the speakers; with headphones turn it up to hear yourself
		y := 20 + float64(len(parts))*110
		mic := add("input", "", 260, y)
		src := add("mixer", "4 channels", 520, y)
		loop := add("audiolooper", "", 780, y)
		connect(patcher.Cable{From: mixer, Out: 0, To: src, In: 0})
		connect(patcher.Cable{From: mic, To: src, In: 1})
		connect(patcher.Cable{From: src, Out: 0, To: loop})
		connect(patcher.Cable{From: loop, To: "out"})
		if err != nil {
			return patchIDs{}, err
		}
		pt.SettingsOf(src).Apply(map[string]float64{"level1": 0, "level2": 1, "pan1": 0, "pan2": 0})
		pt.SettingsOf(loop).Apply(map[string]float64{"bars": 3, "dry": 0})
		ids.looper, ids.sources = loop, src
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
