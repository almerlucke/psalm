package vocal

// Presets maps preset names to their config constructors
var Presets = map[string]func() Config{
	"solo":    Solo,
	"choir":   Choir,
	"ooh":     Ooh,
	"aah":     Aah,
	"breathy": Breathy,
	"monks":   Monks,
	"angels":  Angels,
	"robot":   Robot,

	"talkbox":     Talkbox,
	"wahChoir":    WahChoir,
	"throatDrone": ThroatDrone,
	"whisper":     Whisper,
	"alien":       Alien,
	"babble":      Babble,
	"chant":       Chant,
	"giant":       Giant,
	"child":       Child,
}

// Solo is one singer with a natural vibrato
func Solo() Config {
	return Config{Vowel: 0, VowelGlide: 0.15, Brightness: 0.5, Breath: 0.05, Attack: 0.08, Release: 0.3,
		VibratoRate: 5.5, VibratoDepth: 0.25, VibratoDelay: 0.3, Singers: 1, Level: 1.3}
}

// Choir is a section of eight singing "a", spread wide
func Choir() Config {
	return Config{Vowel: 0, VowelGlide: 0.3, Brightness: 0.4, Breath: 0.08, Attack: 0.25, Release: 0.6,
		VibratoRate: 5.2, VibratoDepth: 0.15, VibratoDelay: 0.2, Singers: 8, Detune: 18, Spread: 0.8, Level: 1.4}
}

// Ooh is a soft choir on "u"
func Ooh() Config {
	cfg := Choir()
	cfg.Vowel, cfg.Brightness, cfg.Attack, cfg.Release = 4, 0.2, 0.4, 0.8
	cfg.Level = 3.1
	return cfg
}

// Aah is a bright, full choir on "a"
func Aah() Config {
	cfg := Choir()
	cfg.Brightness, cfg.Singers, cfg.Detune = 0.65, 12, 22
	cfg.Level = 1.25
	return cfg
}

// Breathy is a close, airy solo voice on "o"
func Breathy() Config {
	cfg := Solo()
	cfg.Vowel, cfg.Brightness, cfg.Breath, cfg.VibratoDepth = 3, 0.15, 0.35, 0.15
	cfg.Level = 2.25
	return cfg
}

// Monks are low men's voices on "o", without vibrato
func Monks() Config {
	return Config{Range: Bass, Vowel: 3, VowelGlide: 0.4, Brightness: 0.35, Breath: 0.05, Attack: 0.3, Release: 0.8,
		Singers: 6, Detune: 12, Spread: 0.6, Level: 2.2}
}

// Angels are high voices on "i" morphing slowly, wide and airy
func Angels() Config {
	return Config{Range: Soprano, Vowel: 2, VowelGlide: 0.8, Brightness: 0.3, Breath: 0.15, Attack: 0.6, Release: 1.2,
		VibratoRate: 5, VibratoDepth: 0.12, VibratoDelay: 0.4, Singers: 8, Detune: 14, Spread: 1, Level: 1}
}

// Robot is a steady, bright voice with no vibrato and no drift: a vocoder's voice
func Robot() Config {
	return Config{Range: Tenor, Vowel: 1, VowelGlide: 0.05, Brightness: 0.9, Attack: 0.01, Release: 0.1,
		Singers: 1, Steady: true, Level: 1.2}
}

// Talkbox jumps to a new vowel every sixteenth note, with the tempo: a bright, steady voice talking
// in rhythm
func Talkbox() Config {
	return Config{Range: Tenor, Vowel: 2, VowelGlide: 0.03, VowelSync: 0.25, VowelDepth: 2, VowelSteps: true,
		Brightness: 0.9, Attack: 0.01, Release: 0.08, Singers: 1, Steady: true, Level: 1.9}
}

// WahChoir is a choir whose vowel sweeps from "a" to "u" and back, slowly
func WahChoir() Config {
	return Config{Vowel: 2, VowelGlide: 0.1, VowelRate: 0.4, VowelDepth: 2, Brightness: 0.5, Breath: 0.08,
		Attack: 0.3, Release: 0.7, VibratoRate: 5, VibratoDepth: 0.12, VibratoDelay: 0.3, Singers: 8, Detune: 16,
		Spread: 0.9, Level: 1.63}
}

// ThroatDrone is sixteen low voices, far apart in pitch, on a dark vowel that drifts: a huge drone
func ThroatDrone() Config {
	return Config{Range: Bass, Vowel: 3.5, VowelGlide: 0.5, VowelRate: 0.07, VowelDepth: 0.5, Brightness: 0.75,
		Breath: 0.05, Attack: 1.2, Release: 2, Singers: 16, Detune: 35, Spread: 1, Level: 3.31}
}

// Whisper is all air: four voices breathing an "e", no tone of their own to speak of
func Whisper() Config {
	return Config{Vowel: 1, VowelGlide: 0.2, Brightness: 0, Breath: 1, Attack: 0.15, Release: 0.5, Singers: 4,
		Detune: 20, Spread: 0.8, Steady: true, Level: 1.86}
}

// Alien sings with a soprano's throat whatever the note, a fast wide vibrato, three voices far apart
func Alien() Config {
	return Config{Range: Soprano, Vowel: 2, VowelGlide: 0.2, VowelRate: 1.3, VowelDepth: 1, Brightness: 0.8,
		Attack: 0.05, Release: 0.4, VibratoRate: 9, VibratoDepth: 0.6, Singers: 3, Detune: 60, Spread: 1, Level: 1.42}
}

// Babble is a crowd: twelve voices, each note jumping to a new vowel three times a second
func Babble() Config {
	return Config{Vowel: 2, VowelGlide: 0.12, VowelRate: 3, VowelDepth: 2, VowelSteps: true, Brightness: 0.45,
		Breath: 0.15, Attack: 0.2, Release: 0.6, VibratoRate: 5, VibratoDepth: 0.1, Singers: 12, Detune: 30,
		Spread: 1, Level: 1.98}
}

func mustWord(text string) []Syllable {
	w, err := ParseWord(text)
	if err != nil {
		panic(err)
	}
	return w
}

// Chant sings "ta ke so fu": every note the next syllable of the word, looping
func Chant() Config {
	cfg := Solo()
	cfg.Word = mustWord("ta ke so fu")
	cfg.Release, cfg.Steady, cfg.Brightness = 0.15, true, 0.7
	cfg.ConsonantLevel, cfg.Level = 0.15, 1.2
	return cfg
}

// Giant is a deep slow voice, the formants shifted down, speaking "sho-ku-ta" on the beat
func Giant() Config {
	cfg := Monks()
	cfg.Word = mustWord("sho ku ta")
	cfg.FormantShift = 0.75
	cfg.Singers, cfg.Detune = 3, 8
	cfg.WordSync = 1
	cfg.Level = 0.7
	return cfg
}

// Child is a small bright voice (the formants shifted up) on "i"
func Child() Config {
	cfg := Solo()
	cfg.Vowel, cfg.FormantShift, cfg.Brightness = 2, 1.3, 0.6
	cfg.Level = 1.9
	return cfg
}
