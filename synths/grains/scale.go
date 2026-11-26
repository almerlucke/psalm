package grains

import "strings"

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
