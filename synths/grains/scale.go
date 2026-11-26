package grains

import (
	"slices"
	"strings"
)

// scales are the scales the pitch of grains can be held to, by name: the semitones above the root
var scales = map[string][]float64{
	"major":           {0, 2, 4, 5, 7, 9, 11},
	"minor":           {0, 2, 3, 5, 7, 8, 10},
	"harmonicminor":   {0, 2, 3, 5, 7, 8, 11},
	"dorian":          {0, 2, 3, 5, 7, 9, 10},
	"phrygian":        {0, 1, 3, 5, 7, 8, 10},
	"lydian":          {0, 2, 4, 6, 7, 9, 11},
	"mixolydian":      {0, 2, 4, 5, 7, 9, 10},
	"pentatonic":      {0, 2, 4, 7, 9},
	"minorpentatonic": {0, 3, 5, 7, 10},
	"blues":           {0, 3, 5, 6, 7, 10},
	"wholetone":       {0, 2, 4, 6, 8, 10},
	"diminished":      {0, 2, 3, 5, 6, 8, 9, 11},
	"fifths":          {0, 7},
	"octaves":         {0},
}

// ScaleNames are the names of the scales for Config.Scale and the message "scale"
var ScaleNames = []string{"major", "minor", "harmonicMinor", "dorian", "phrygian", "lydian", "mixolydian", "pentatonic", "minorPentatonic",
	"blues", "wholeTone", "diminished", "fifths", "octaves"}

// ScaleNamed is the scale of a name (see ScaleNames, in any case), nil for none: "off" or a name it does not know
func ScaleNamed(name string) []float64 {
	if s, ok := scales[strings.ToLower(strings.TrimSpace(name))]; ok {
		return append([]float64(nil), s...)
	}
	return nil
}

// ScaleChoices are the choices of the setting "scale": none, then the scales
var ScaleChoices = append([]string{"off"}, ScaleNames...)

// scaleIndex is the choice of a scale (0 when it is none or one of your own)
func scaleIndex(scale []float64) int {
	for i, name := range ScaleNames {
		if slices.Equal(ScaleNamed(name), scale) && len(scale) > 0 {
			return i + 1
		}
	}
	return 0
}

// stacks are the chords a grain can be: the semitones above its pitch, by name
var stacks = []struct {
	name string
	up   []float64
}{
	{"off", nil}, {"octave", []float64{0, 12}}, {"fifth", []float64{0, 7}}, {"power", []float64{0, 7, 12}},
	{"major", []float64{0, 4, 7}}, {"minor", []float64{0, 3, 7}}, {"sus4", []float64{0, 5, 7}}, {"major 7", []float64{0, 4, 7, 11}},
	{"minor 7", []float64{0, 3, 7, 10}}, {"cluster", []float64{0, 1, 2}},
}

// StackChoices are the choices of the setting "stack": none, then chords
var StackChoices = func() []string {
	var n []string
	for _, s := range stacks {
		n = append(n, s.name)
	}
	return n
}()

// StackNamed is the chord of a choice of "stack" (by its index), nil for none
func stackOf(i int) []float64 {
	if i <= 0 || i >= len(stacks) {
		return nil
	}
	return append([]float64(nil), stacks[i].up...)
}

// stackIndex is the choice of a chord (0 when it is none or one of your own)
func stackIndex(up []float64) int {
	for i, s := range stacks {
		if i > 0 && slices.Equal(s.up, up) {
			return i
		}
	}
	return 0
}

// NoteNames are the names of the twelve pitch classes, for the setting "scaleRoot"
var NoteNames = []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}
