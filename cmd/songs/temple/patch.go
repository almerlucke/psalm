package main

import (
	"fmt"

	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
)

// midiFile is the piece: 16 channels, 66 bpm, four minutes
const midiFile = "assets/midi/genny-temple.mid"

// tempo is the bpm of the piece (the file's)
const tempo = 66.0

// part is a voice of the piece: the channels of the file it plays, the synth and preset that sound it, and where it is in the mix. The tracks of the file are named after
// the furnishings of the temple of Solomon
type part struct {
	name     string
	channels []int
	kind     string // the synth of the patcher
	preset   string
	level    float64 // of the mixer strip
	pan      float64
	hall     float64 // the sends: a hall, an echo and a shimmer
	echo     float64
	shimmer  float64
}

// parts are the voices of the piece, by the tracks of the file: the instruments are the ones of the General MIDI programs of the file, as psalm plays them
var parts = []part{
	{name: "molten sea", channels: []int{1}, kind: "modal", preset: "tubularBells", level: 0.45, pan: 0, hall: 0.5, shimmer: 0.3},
	{name: "lavers left", channels: []int{2}, kind: "modal", preset: "vibraphone", level: 0.33, pan: -0.4, hall: 0.35, echo: 0.2},
	{name: "lavers right", channels: []int{3}, kind: "modal", preset: "glockenspiel", level: 0.4, pan: 0.4, hall: 0.35, echo: 0.2},
	{name: "jachin", channels: []int{4}, kind: "organ", preset: "full", level: 0.315, pan: -0.3, hall: 0.4},
	{name: "boaz", channels: []int{5}, kind: "organ", preset: "full", level: 0.203, pan: 0.3, hall: 0.4},
	{name: "gold bells", channels: []int{6}, kind: "modal", preset: "vibraphone", level: 0.42, pan: 0.2, hall: 0.4, shimmer: 0.4},
	{name: "lamps left", channels: []int{7}, kind: "waveguide", preset: "harp", level: 0.347, pan: -0.5, hall: 0.35, echo: 0.25},
	{name: "lamps right", channels: []int{8}, kind: "waveguide", preset: "harp", level: 0.625, pan: 0.5, hall: 0.4, shimmer: 0.3},
	{name: "side chambers 5", channels: []int{9}, kind: "vocal", preset: "aah", level: 0.28, pan: 0, hall: 0.5},
	{name: "side chambers 6", channels: []int{11}, kind: "vocal", preset: "aah", level: 0.268, pan: -0.5, hall: 0.5},
	{name: "side chambers 7", channels: []int{12}, kind: "vocal", preset: "aah", level: 0.28, pan: 0.5, hall: 0.5, shimmer: 0.2},
	{name: "altar and room", channels: []int{13}, kind: "subtractive", preset: "pad", level: 0.27, pan: 0, hall: 0.5},
	{name: "wing four", channels: []int{14}, kind: "vocal", preset: "aah", level: 0.252, pan: 0, hall: 0.5},
	{name: "room and glory", channels: []int{15}, kind: "vocal", preset: "angels", level: 0.5, pan: 0, hall: 0.6, shimmer: 0.5},
	{name: "piano melody", channels: []int{16}, kind: "piano", preset: "grand", level: 0.4, pan: 0, hall: 0.3, echo: 0.1},
	{name: "offerings", channels: []int{10}, kind: "drumsynth", preset: "808", level: 0.7, pan: 0.15, hall: 0.15, echo: 0.1},
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

// buildPatch makes the patch: a midiplayer that plays the file, a synth for each part on the channels it plays, all on a mixer with three buses (a hall, an echo and a
// shimmer)
func buildPatch(pt *patcher.Patcher) (ids patchIDs, err error) {
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
		syn := add(p.kind, p.preset, 400, 20+float64(i)*90)
		if err != nil {
			return patchIDs{}, err
		}
		for _, ch := range p.channels {
			connect(patcher.Cable{From: player, Out: ch, To: syn})
		}
		connect(patcher.Cable{From: syn, To: mixer, In: i})
		level := p.level
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
