package drumsynth

// Kits maps kit names to their constructors
var Kits = map[string]func() Kit{
	"808": Kit808,
	"909": Kit909,

	"electro":    KitElectro,
	"trap":       KitTrap,
	"industrial": KitIndustrial,
}

// Kit808 is the boom of the TR-808: a long, deep kick, the metallic hats, the cowbell
func Kit808() Kit {
	return withPercussion(Kit{Sounds: map[string]Sound{
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
	}}, 0, 1)
}

// Kit909 is the punch of the TR-909: a short kick with a click, a bright snare, noisy hats
func Kit909() Kit {
	return withPercussion(Kit{Sounds: map[string]Sound{
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
	}}, 0.1, 1.05)
}

// KitElectro is electro funk: kicks and toms that zap down from high up, a crisp snare, tight high hats
func KitElectro() Kit {
	return withPercussion(Kit{Sounds: map[string]Sound{
		"kick":    {Kind: Kick, Pitch: 55, Decay: 0.45, Sweep: 30, SweepTime: 0.04, Snap: 0.3, Drive: 0.3, Level: 0.71},
		"snare":   {Kind: Snare, Pitch: 220, Decay: 0.18, Sweep: 12, SweepTime: 0.02, Tone: 0.9, Snap: 0.9, Level: 0.27},
		"hat":     {Kind: Hat, Pitch: 1.6, Decay: 0.04, Tone: 1, Snap: 0.2, Level: 0.34, Pan: 0.25, Choke: 1},
		"open":    {Kind: Hat, Pitch: 1.6, Decay: 0.25, Tone: 1, Snap: 0.2, Level: 0.32, Pan: 0.25, Choke: 1},
		"tomLow":  {Kind: Tom, Pitch: 150, Decay: 0.35, Sweep: 24, SweepTime: 0.08, Snap: 0.2, Level: 0.38, Pan: -0.4},
		"tomMid":  {Kind: Tom, Pitch: 220, Decay: 0.3, Sweep: 24, SweepTime: 0.07, Snap: 0.2, Level: 0.48, Pan: 0},
		"tomHigh": {Kind: Tom, Pitch: 330, Decay: 0.25, Sweep: 24, SweepTime: 0.06, Snap: 0.2, Level: 0.38, Pan: 0.4},
		"clap":    {Kind: Clap, Decay: 0.2, Tone: 0.8, Snap: 0.8, Level: 0.17},
		"rim":     {Kind: Rim, Pitch: 2400, Decay: 0.03, Snap: 0.4, Level: 0.21, Pan: -0.2},
		"cowbell": {Kind: Cowbell, Pitch: 800, Decay: 0.2, Level: 0.15, Pan: 0.15},
		"cymbal":  {Kind: Cymbal, Pitch: 1.5, Decay: 1, Tone: 0.9, Snap: 0.4, Level: 0.26, Pan: -0.25},
	}}, 0, 1.25)
}

// KitTrap is the trap kit: a long, driven 808 boom to play bass lines on, tight hats for rolls, a
// big clap
func KitTrap() Kit {
	return withPercussion(Kit{Sounds: map[string]Sound{
		"kick":    {Kind: Kick, Pitch: 43, Decay: 2.5, Sweep: 7, SweepTime: 0.03, Snap: 0.2, Drive: 0.6, Level: 0.71},
		"snare":   {Kind: Snare, Pitch: 190, Decay: 0.22, Sweep: 4, SweepTime: 0.015, Tone: 0.6, Snap: 0.8, Drive: 0.3, Level: 0.56},
		"hat":     {Kind: Hat, Pitch: 1.3, Decay: 0.035, Tone: 0.9, Snap: 0.5, Level: 0.42, Pan: 0.2, Choke: 1},
		"open":    {Kind: Hat, Pitch: 1.3, Decay: 0.3, Tone: 0.9, Snap: 0.5, Level: 0.37, Pan: 0.2, Choke: 1},
		"tomLow":  {Kind: Tom, Pitch: 70, Decay: 1, Sweep: 3, SweepTime: 0.05, Snap: 0.1, Drive: 0.3, Level: 0.42, Pan: -0.3},
		"tomMid":  {Kind: Tom, Pitch: 95, Decay: 0.9, Sweep: 3, SweepTime: 0.05, Snap: 0.1, Drive: 0.3, Level: 0.50},
		"tomHigh": {Kind: Tom, Pitch: 120, Decay: 0.8, Sweep: 3, SweepTime: 0.05, Snap: 0.1, Drive: 0.3, Level: 0.42, Pan: 0.3},
		"clap":    {Kind: Clap, Decay: 0.45, Tone: 0.55, Snap: 0.9, Level: 0.16},
		"rim":     {Kind: Rim, Pitch: 1700, Decay: 0.04, Snap: 0.4, Level: 0.23, Pan: -0.1},
		"cowbell": {Kind: Cowbell, Pitch: 587, Decay: 0.3, Level: 0.16, Pan: 0.1},
		"cymbal":  {Kind: Cymbal, Decay: 1.8, Tone: 0.5, Snap: 0.4, Level: 0.28, Pan: -0.2},
	}}, 0.1, 1)
}

// KitIndustrial is everything driven into distortion: a kick that tears down from up high, a
// snare like a hammer on sheet metal, low clanging metal for hats and cymbal
func KitIndustrial() Kit {
	return withPercussion(Kit{Sounds: map[string]Sound{
		"kick":    {Kind: Kick, Pitch: 40, Decay: 0.6, Sweep: 36, SweepTime: 0.02, Snap: 1, Drive: 1, Level: 0.71},
		"snare":   {Kind: Snare, Pitch: 150, Decay: 0.4, Sweep: 24, SweepTime: 0.015, Tone: 1, Snap: 1, Drive: 1, Level: 0.56},
		"hat":     {Kind: Hat, Pitch: 0.7, Decay: 0.07, Tone: 0.3, Snap: 0.8, Drive: 0.8, Level: 0.26, Pan: 0.3, Choke: 1},
		"open":    {Kind: Hat, Pitch: 0.7, Decay: 0.6, Tone: 0.3, Snap: 0.8, Drive: 0.8, Level: 0.26, Pan: 0.3, Choke: 1},
		"tomLow":  {Kind: Tom, Pitch: 80, Decay: 0.5, Sweep: 12, SweepTime: 0.03, Snap: 0.8, Drive: 1, Level: 0.42, Pan: -0.3},
		"tomMid":  {Kind: Tom, Pitch: 115, Decay: 0.45, Sweep: 12, SweepTime: 0.03, Snap: 0.8, Drive: 1, Level: 0.50},
		"tomHigh": {Kind: Tom, Pitch: 160, Decay: 0.4, Sweep: 12, SweepTime: 0.03, Snap: 0.8, Drive: 1, Level: 0.42, Pan: 0.3},
		"clap":    {Kind: Clap, Decay: 0.6, Tone: 0.2, Snap: 1, Drive: 1, Level: 0.50},
		"rim":     {Kind: Rim, Pitch: 900, Decay: 0.08, Snap: 1, Drive: 1, Level: 0.28, Pan: -0.2},
		"cowbell": {Kind: Cowbell, Pitch: 320, Decay: 0.6, Drive: 1, Level: 0.22, Pan: 0.2},
		"cymbal":  {Kind: Cymbal, Pitch: 0.6, Decay: 3, Tone: 0.2, Snap: 1, Drive: 0.7, Level: 0.21, Pan: -0.3},
	}}, 0.9, 0.85)
}

// withPercussion gives a kit the rest of the General MIDI percussion, in the kit's character: how
// driven it is, how high its drums are tuned (a factor). The sounds of the kit itself stay
func withPercussion(kit Kit, drive, tune float64) Kit {
	d := func(x float64) float64 { return min(1, x+drive) }
	more := map[string]Sound{
		"crash":         {Kind: Cymbal, Pitch: 1.1 * tune, Decay: 2.2, Tone: 0.55, Snap: 0.5, Drive: drive, Level: 0.3, Pan: -0.3},
		"ride":          {Kind: Cymbal, Pitch: 1.45 * tune, Decay: 1.4, Tone: 0.75, Snap: 0.15, Drive: drive, Level: 0.24, Pan: 0.3},
		"rideBell":      {Kind: Cowbell, Pitch: 1180 * tune, Decay: 0.9, Drive: drive, Level: 0.18, Pan: 0.3},
		"tambourine":    {Kind: Shaker, Pitch: tune, Decay: 0.25, Tone: 0.8, Snap: 0.8, Drive: drive, Level: 0.3, Pan: 0.35},
		"shaker":        {Kind: Shaker, Decay: 0.09, Tone: 0.65, Drive: drive, Level: 0.3, Pan: -0.35},
		"bongoHigh":     {Kind: Tom, Pitch: 420 * tune, Decay: 0.18, Sweep: 2, SweepTime: 0.01, Snap: d(0.4), Drive: drive, Level: 0.35, Pan: 0.4},
		"bongoLow":      {Kind: Tom, Pitch: 300 * tune, Decay: 0.22, Sweep: 2, SweepTime: 0.01, Snap: d(0.4), Drive: drive, Level: 0.38, Pan: 0.3},
		"congaMute":     {Kind: Tom, Pitch: 330 * tune, Decay: 0.07, Sweep: 1, SweepTime: 0.01, Snap: d(0.6), Drive: drive, Level: 0.35, Pan: -0.2},
		"congaHigh":     {Kind: Tom, Pitch: 290 * tune, Decay: 0.32, Sweep: 1.5, SweepTime: 0.015, Snap: d(0.3), Drive: drive, Level: 0.4, Pan: -0.2},
		"congaLow":      {Kind: Tom, Pitch: 200 * tune, Decay: 0.4, Sweep: 1.5, SweepTime: 0.015, Snap: d(0.3), Drive: drive, Level: 0.42, Pan: -0.3},
		"timbaleHigh":   {Kind: Snare, Pitch: 520 * tune, Decay: 0.35, Sweep: 1, SweepTime: 0.01, Tone: 0.8, Snap: 0.1, Drive: drive, Level: 0.32, Pan: 0.25},
		"timbaleLow":    {Kind: Snare, Pitch: 390 * tune, Decay: 0.4, Sweep: 1, SweepTime: 0.01, Tone: 0.8, Snap: 0.1, Drive: drive, Level: 0.34, Pan: 0.15},
		"agogoHigh":     {Kind: Cowbell, Pitch: 920 * tune, Decay: 0.4, Drive: drive, Level: 0.13, Pan: 0.2},
		"agogoLow":      {Kind: Cowbell, Pitch: 690 * tune, Decay: 0.45, Drive: drive, Level: 0.14, Pan: 0.1},
		"clave":         {Kind: Rim, Pitch: 2500 * tune, Decay: 0.12, Snap: 0.1, Drive: drive, Level: 0.22, Pan: 0.2},
		"woodblockHigh": {Kind: Rim, Pitch: 1500 * tune, Decay: 0.09, Snap: 0.3, Drive: drive, Level: 0.24, Pan: -0.15},
		"woodblockLow":  {Kind: Rim, Pitch: 1050 * tune, Decay: 0.1, Snap: 0.3, Drive: drive, Level: 0.25, Pan: -0.25},
		"triangle":      {Kind: Rim, Pitch: 4300 * tune, Decay: 1.6, Drive: drive, Level: 0.2, Pan: 0.4, Choke: 2},
		"triangleMute":  {Kind: Rim, Pitch: 4300 * tune, Decay: 0.12, Drive: drive, Level: 0.2, Pan: 0.4, Choke: 2},
	}
	for name, snd := range more {
		if _, ok := kit.Sounds[name]; !ok {
			kit.Sounds[name] = snd
		}
	}
	return kit
}
