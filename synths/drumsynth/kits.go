package drumsynth

// Kits maps kit names to their constructors
var Kits = map[string]func() Kit{
	"808": Kit808,
	"909": Kit909,
}

// Kit808 is the boom of the TR-808: a long, deep kick, the metallic hats, the cowbell
func Kit808() Kit {
	return Kit{Sounds: map[string]Sound{
		"kick":    {Kind: Kick, Pitch: 49, Decay: 1.1, Sweep: 12, SweepTime: 0.025, Snap: 0.15, Drive: 0.1, Level: 0.71},
		"snare":   {Kind: Snare, Pitch: 180, Decay: 0.3, Sweep: 2, SweepTime: 0.02, Tone: 0.4, Snap: 0.5, Level: 0.35},
		"hat":     {Kind: Hat, Decay: 0.08, Tone: 0.6, Snap: 0.1, Level: 0.39, Pan: 0.2, Choke: 1},
		"open":    {Kind: Hat, Decay: 0.6, Tone: 0.6, Snap: 0.1, Level: 0.35, Pan: 0.2, Choke: 1},
		"tomLow":  {Kind: Tom, Pitch: 90, Decay: 0.7, Sweep: 4, SweepTime: 0.06, Snap: 0.1, Level: 0.42, Pan: -0.3},
		"tomMid":  {Kind: Tom, Pitch: 130, Decay: 0.6, Sweep: 4, SweepTime: 0.05, Snap: 0.1, Level: 0.51, Pan: 0},
		"tomHigh": {Kind: Tom, Pitch: 180, Decay: 0.5, Sweep: 4, SweepTime: 0.04, Snap: 0.1, Level: 0.43, Pan: 0.3},
		"clap":    {Kind: Clap, Decay: 0.35, Tone: 0.3, Snap: 0.5, Level: 0.23},
		"rim":     {Kind: Rim, Pitch: 1667, Decay: 0.05, Snap: 0.3, Level: 0.24, Pan: -0.1},
		"cowbell": {Kind: Cowbell, Pitch: 540, Decay: 0.35, Level: 0.16, Pan: 0.1},
		"cymbal":  {Kind: Cymbal, Decay: 2, Tone: 0.4, Snap: 0.3, Level: 0.32, Pan: -0.2},
	}}
}

// Kit909 is the punch of the TR-909: a short kick with a click, a bright snare, noisy hats
func Kit909() Kit {
	return Kit{Sounds: map[string]Sound{
		"kick":    {Kind: Kick, Pitch: 52, Decay: 0.5, Sweep: 24, SweepTime: 0.012, Snap: 0.6, Drive: 0.5, Level: 0.71},
		"snare":   {Kind: Snare, Pitch: 200, Decay: 0.25, Sweep: 5, SweepTime: 0.01, Tone: 0.7, Snap: 0.9, Drive: 0.2, Level: 0.55},
		"hat":     {Kind: Hat, Pitch: 1.1, Decay: 0.06, Tone: 0.8, Snap: 0.6, Level: 0.42, Pan: 0.2, Choke: 1},
		"open":    {Kind: Hat, Pitch: 1.1, Decay: 0.4, Tone: 0.8, Snap: 0.6, Level: 0.31, Pan: 0.2, Choke: 1},
		"tomLow":  {Kind: Tom, Pitch: 100, Decay: 0.45, Sweep: 9, SweepTime: 0.04, Snap: 0.3, Level: 0.43, Pan: -0.3},
		"tomMid":  {Kind: Tom, Pitch: 145, Decay: 0.4, Sweep: 9, SweepTime: 0.035, Snap: 0.3, Level: 0.50, Pan: 0},
		"tomHigh": {Kind: Tom, Pitch: 200, Decay: 0.35, Sweep: 9, SweepTime: 0.03, Snap: 0.3, Level: 0.40, Pan: 0.3},
		"clap":    {Kind: Clap, Decay: 0.3, Tone: 0.5, Snap: 0.7, Level: 0.26},
		"rim":     {Kind: Rim, Pitch: 1800, Decay: 0.04, Snap: 0.5, Level: 0.22, Pan: -0.1},
		"cowbell": {Kind: Cowbell, Pitch: 560, Decay: 0.3, Level: 0.17, Pan: 0.1},
		"cymbal":  {Kind: Cymbal, Pitch: 1.2, Decay: 1.5, Tone: 0.6, Snap: 0.6, Level: 0.23, Pan: -0.2},
	}}
}
