package main

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
)

// midiFile is the piece: 15 channels, 66 bpm, four minutes
const midiFile = "assets/midi/genny-temple.mid"

// tempo is the bpm of the piece (the file's)
const tempo = 66.0

// part is a voice of the piece: the channels of the file it plays, the shaper preset that sounds it, and where it is in the mix. The tracks of the file are named after the furnishings of
// the temple of Solomon, and every part is the shaper synth
type part struct {
	name     string
	channels []int
	preset   string
	level    float64 // of the mixer strip
	pan      float64
	hall     float64 // the sends: a hall, an echo and a shimmer
	echo     float64
	shimmer  float64
	tune     map[string]float64 // settings on top of the preset
	// sound is the instrument of the sampler when the orchestra is there: a sfz of the Virtual Playing Orchestra, or the text of the sampler's sounds (a line a sound)
	sound string
	vol   float64 // dB: how much louder the instrument plays (the recordings differ in loudness; `vol=` of the sampler)
}

// orchestra is where the Virtual Playing Orchestra is (it is not in the repository: without it the shaper synths play)
const orchestra = "assets/Virtual-Playing-Orchestra3"

func hasOrchestra() bool {
	_, err := os.Stat(orchestra + "/Vocals/choir-MIXED-sustain.sfz")
	return err == nil
}

// parts are the voices of the piece, by the tracks of the file
var parts = []part{
	{name: "molten sea", channels: []int{1}, preset: "churchBell", vol: -2, sound: "Percussion/vibraphone-open.sfz", level: 0.5, pan: 0, hall: 0.5, shimmer: 0.3},
	{name: "lavers left", channels: []int{2}, preset: "vibes", vol: 12, sound: "Percussion/tubular-bells.sfz", level: 0.5, pan: -0.4, hall: 0.35, echo: 0.2},
	{name: "lavers right", channels: []int{3}, preset: "glassBell", vol: 6, sound: "Keys/celesta.sfz", level: 0.5, pan: 0.4, hall: 0.35, echo: 0.2},
	{name: "jachin", channels: []int{4}, preset: "tubeOrgan", vol: 1, sound: "Brass/french-horn-SEC-sustain.sfz", level: 0.45, pan: -0.3, hall: 0.4},
	{name: "boaz", channels: []int{5}, preset: "organ", sound: "Strings/cello-SEC-sustain.sfz", level: 0.45, pan: 0.3, hall: 0.4},
	{name: "pomegranates", channels: []int{6}, preset: "xmodBell", vol: 14, sound: "Percussion/glockenspiel.sfz", level: 0.45, pan: 0.2, hall: 0.4, shimmer: 0.4},
	{name: "lamps left", channels: []int{7}, preset: "harp", sound: "Strings/harp-sustain.sfz", level: 0.55, pan: -0.5, hall: 0.35, echo: 0.25},
	{name: "lamps right", channels: []int{8}, preset: "sevenLamps", vol: 7, sound: "Strings/harp-dampened.sfz", level: 0.5, pan: 0.5, hall: 0.4, shimmer: 0.3},
	{name: "side chambers 5", channels: []int{9}, preset: "choirPad", vol: 4, sound: "Vocals/choir-MIXED-sustain.sfz", level: 0.4, pan: 0, hall: 0.5},
	{name: "side chambers 6", channels: []int{11}, preset: "vowel", vol: 7, sound: "Vocals/choir-MALE-sustain.sfz", level: 0.4, pan: -0.5, hall: 0.5},
	{name: "side chambers 7", channels: []int{12}, preset: "wideMorph", vol: 6, sound: "Vocals/choir-MIXED-sustain.sfz", level: 0.4, pan: 0.5, hall: 0.5, shimmer: 0.2},
	{name: "altar and room", channels: []int{13}, preset: "darkPad", vol: 4, sound: "Strings/all-strings-SEC-sustain.sfz", level: 0.45, pan: 0, hall: 0.5},
	{name: "wing four", channels: []int{14}, preset: "splitPad", sound: "Woodwinds/all-woodwinds-SEC-sustain.sfz", level: 0.4, pan: 0, hall: 0.5},
	{name: "glory", channels: []int{15}, preset: "crystalSea", vol: 9, sound: "Vocals/choir-FEMALE-sustain.sfz", level: 0.5, pan: 0, hall: 0.6, shimmer: 0.5},
	{name: "offerings", channels: []int{10}, preset: "tick", vol: 6, sound: "37 " + orchestra + "/libs/Other/Samples/Percussion/SSO/wood_block-hi-PB.wav whole 41 39", level: 0.7, pan: 0.15, hall: 0.15, echo: 0.1},
}

// duration is how long the file lasts, in seconds
func duration() float64 {
	f, err := midi.ReadFile(midiFile)
	if err != nil {
		return 0
	}
	return f.Seconds(f.Ticks())
}

// patchIDs are the modules of the patch to look at: the synths by their part
type patchIDs struct {
	player string
	synths map[string]string
}

// buildPatch makes the patch: a midiplayer that plays the file, a shaper synth for each part on the channels it plays, all on a mixer with three buses (a hall, an echo and a
// shimmer)
func buildPatch(pt *patcher.Patcher, real bool) (ids patchIDs, err error) {
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
	pt.Tempo().Set(tempo)
	player := add("midiplayer", "", 20, 20)
	if err == nil {
		err = pt.SetText(player, "file", midiFile)
		if err == nil && pt.TextError(player, "file") != "" {
			err = fmt.Errorf("%s: %s", midiFile, pt.TextError(player, "file"))
		}
	}
	if err != nil {
		return patchIDs{}, err
	}
	pt.SettingsOf(player).Apply(map[string]float64{"play": 1, "lift": 0.3})
	mixer := add("mixer", "16 channels", 780, 300)
	hall := add("reverb", "", 1040, 20)
	echo := add("pingpong", "", 1040, 140)
	shine := add("shimmer", "", 1040, 260)
	if err != nil {
		return patchIDs{}, err
	}
	mix := map[string]float64{"returnA": 1, "returnB": 0.7, "returnC": 0.5, "level": 0.8}
	ids = patchIDs{player: player, synths: map[string]string{}}
	for i, p := range parts {
		n := i + 1
		kind, preset := "shaper", p.preset
		if real && p.sound != "" {
			kind, preset = "sampler", ""
		}
		syn := add(kind, preset, 400, 20+float64(i)*90)
		if err != nil {
			return patchIDs{}, err
		}
		for _, ch := range p.channels {
			connect(patcher.Cable{From: player, Out: ch, To: syn})
		}
		connect(patcher.Cable{From: syn, To: mixer, In: i})
		level := p.level
		if kind == "sampler" {
			text := p.sound
			if !strings.Contains(text, " ") { // a sfz of the orchestra
				text = "0 " + orchestra + "/" + p.sound + " sfz"
			}
			if err == nil {
				err = pt.SetText(syn, "sounds", text)
				if err == nil && pt.TextError(syn, "sounds") != "" {
					err = fmt.Errorf("%s: %s", p.name, pt.TextError(syn, "sounds"))
				}
			}
			if p.vol != 0 { // the gain: the output level of the sampler (up to 4), what is more on the strip
				lin := math.Pow(10, p.vol/20)
				own := min(lin, 4)
				if err == nil {
					pt.Module(syn).ReceiveMessage(psalm.Message{"level": own})
				}
				level *= lin / own
			}
		} else if err == nil && len(p.tune) > 0 {
			pt.SettingsOf(syn).Apply(p.tune)
		}
		mix[fmt.Sprintf("level%d", n)] = level
		mix[fmt.Sprintf("pan%d", n)] = p.pan
		mix[fmt.Sprintf("sendA%d", n)] = p.hall
		mix[fmt.Sprintf("sendB%d", n)] = p.echo
		mix[fmt.Sprintf("sendC%d", n)] = p.shimmer
		ids.synths[p.name] = syn
	}
	for i, fx := range []string{hall, echo, shine} { // each send to its effect, which comes back on a return (after the 16 channels)
		connect(patcher.Cable{From: mixer, Out: 1 + i, To: fx})
		connect(patcher.Cable{From: fx, To: mixer, In: 16 + i})
	}
	connect(patcher.Cable{From: mixer, Out: 0, To: "out"})
	if err != nil {
		return patchIDs{}, err
	}
	pt.SettingsOf(mixer).Apply(mix)
	pt.SettingsOf(hall).Apply(map[string]float64{"mix": 0.7, "decay": 4, "size": 0.8})
	pt.SettingsOf(echo).Apply(map[string]float64{"time": 0.75, "feedback": 0.3, "stereo": 1, "highCut": 5000, "lowCut": 200, "mix": 1})
	pt.SettingsOf(shine).Apply(map[string]float64{"shimmer": 0.3, "semitones": 12, "mix": 1, "decay": 6})
	pt.Move("out", 1060, 400)
	return ids, nil
}
