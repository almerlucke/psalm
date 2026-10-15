package vocal

import "github.com/almerlucke/psalm/blocks/modmatrix"

func init() {
	for name, f := range morePresets {
		Presets[name] = f
	}
}

// morePresets are the registrations of presets_more.go
var morePresets = map[string]func() Config{
	"bassAh":        BassAh,
	"bassOo":        BassOo,
	"tenorOh":       TenorOh,
	"tenorEe":       TenorEe,
	"altoAh":        AltoAh,
	"altoEh":        AltoEh,
	"sopranoAh":     SopranoAh,
	"sopranoEe":     SopranoEe,
	"childOo":       ChildOo,
	"childAh":       ChildAh,
	"bassChoir":     BassChoir,
	"tenorChoir":    TenorChoir,
	"altoChoir":     AltoChoir,
	"sopranoChoir":  SopranoChoir,
	"childrenChoir": ChildrenChoir,
	"humming":       Humming,
	"hummingChoir":  HummingChoir,
	"whisperSoft":   WhisperSoft,
	"whisperBreath": WhisperBreath,
	"chantMantra":   ChantMantra,
	"kyrie":         Kyrie,
	"spokenWords":   SpokenWords,
	"operaTenor":    OperaTenor,
	"operaSoprano":  OperaSoprano,
	"operaBass":     OperaBass,
	"popBreathy":    PopBreathy,
	"popBright":     PopBright,
	"gospelChoir":   GospelChoir,
	"monkDrone":     MonkDrone,
	"throatLow":     ThroatLow,
	"overtoneSweep": OvertoneSweep,
	"padWarm":       PadWarm,
	"padDark":       PadDark,
	"padAir":        PadAir,
	"syllableStab":  SyllableStab,
	"consonantStab": ConsonantStab,
	"diphthongAi":   DiphthongAi,
	"diphthongOu":   DiphthongOu,
	"liquidLa":      LiquidLa,
	"wheelVowel":    WheelVowel,
	"wheelWhisper":  WheelWhisper,
}

// BassAh is one bass singing "a", a natural vibrato late in coming
func BassAh() Config {
	cfg := Solo()
	cfg.Range, cfg.Vowel, cfg.Brightness, cfg.Training = Bass, 0, 0.6, 0.3
	cfg.Level = 0.6572
	return cfg
}

// BassOo is one bass humming a closed "u", dark and close
func BassOo() Config {
	cfg := Solo()
	cfg.Range, cfg.Vowel, cfg.Brightness, cfg.Breath, cfg.VibratoDepth = Bass, 4, 0.2, 0.1, 0.1
	cfg.Level = 1.621
	return cfg
}

// TenorOh is one tenor on a round "o", an LF source
func TenorOh() Config {
	cfg := Solo()
	cfg.Range, cfg.Vowel, cfg.Source, cfg.Rd, cfg.Brightness = Tenor, 3, 1, 0.9, 0.5
	cfg.Level = 1
	return cfg
}

// TenorEe is one tenor on a bright "i", pressed
func TenorEe() Config {
	cfg := Solo()
	cfg.Range, cfg.Vowel, cfg.Brightness, cfg.VibratoDepth = Tenor, 2, 0.85, 0.15
	cfg.Level = 2.173
	return cfg
}

// AltoAh is one alto on "a", warm, with the singer's formant
func AltoAh() Config {
	cfg := Solo()
	cfg.Range, cfg.Vowel, cfg.Training, cfg.Brightness, cfg.Breath = Alto, 0, 0.6, 0.45, 0.15
	cfg.Level = 1
	return cfg
}

// AltoEh is one alto on an open "e", a little breath
func AltoEh() Config {
	cfg := Solo()
	cfg.Range, cfg.Vowel, cfg.Breath, cfg.Brightness = Alto, 1, 0.12, 0.55
	cfg.Level = 0.6593
	return cfg
}

// SopranoAh is one soprano on "a", the first formant following the pitch up: the open head voice
func SopranoAh() Config {
	cfg := Solo()
	cfg.Range, cfg.Vowel, cfg.Training, cfg.Tuning, cfg.NaturalVibrato, cfg.Brightness = Soprano, 0.3, 0.9, 1, 0.6, 0.3
	cfg.VibratoDepth = 0
	cfg.Level = 2.579
	return cfg
}

// SopranoEe is one soprano on a thin, high "i"
func SopranoEe() Config {
	cfg := Solo()
	cfg.Range, cfg.Vowel, cfg.Brightness, cfg.VibratoDepth = Soprano, 2, 0.35, 0.18
	cfg.Level = 2.453
	return cfg
}

// ChildOo is a child singing "u": small, pure, almost without edge
func ChildOo() Config {
	cfg := Solo()
	cfg.Range, cfg.Vowel, cfg.Brightness, cfg.Breath, cfg.VibratoDepth = ChildVoice, 4, 0.3, 0.08, 0.05
	cfg.Level = 0.9433
	return cfg
}

// ChildAh is a child shouting "a": bright, open, the voice a little pressed
func ChildAh() Config {
	cfg := Solo()
	cfg.Range, cfg.Vowel, cfg.Brightness, cfg.Source, cfg.Rd = ChildVoice, 0, 0.9, 1, 0.5
	cfg.VibratoDepth = 0
	cfg.Level = 1
	return cfg
}

// BassChoir is six basses on "o", slow in and slow out
func BassChoir() Config {
	return Config{Range: Bass, Vowel: 3, VowelGlide: 0.4, Brightness: 0.4, Breath: 0.06, Attack: 0.35, Release: 0.9,
		VibratoRate: 5, VibratoDepth: 0.12, VibratoDelay: 0.3, Singers: 6, Detune: 20, Spread: 0.7, Jitter: 0.005,
		Shimmer: 0.03, Level: 1.087}
}

// TenorChoir is five tenors on "e" with an LF source, forward and wide
func TenorChoir() Config {
	return Config{Range: Tenor, Vowel: 1.2, VowelGlide: 0.3, Brightness: 0.85, Breath: 0.06, Attack: 0.2, Release: 0.6,
		VibratoRate: 5.5, VibratoDepth: 0.2, VibratoDelay: 0.2, Singers: 5, Detune: 16, Spread: 0.9, Jitter: 0.005,
		Shimmer: 0.03, Training: 0.4, Source: 1, Rd: 0.7, Level: 3.331}
}

// AltoChoir is eight altos on "e": a thick middle voice
func AltoChoir() Config {
	return Config{Range: Alto, Vowel: 1, VowelGlide: 0.3, Brightness: 0.5, Breath: 0.08, Attack: 0.25, Release: 0.7,
		VibratoRate: 5.2, VibratoDepth: 0.15, VibratoDelay: 0.25, Singers: 8, Detune: 22, Spread: 0.8, Jitter: 0.005,
		Shimmer: 0.03, Level: 0.7105}
}

// SopranoChoir is ten sopranos on "i" with the singer's formant: a high, silvery section
func SopranoChoir() Config {
	return Config{Range: Soprano, Vowel: 2, VowelGlide: 0.3, Brightness: 0.55, Breath: 0.05, Attack: 0.3, Release: 0.8,
		VibratoRate: 5.8, VibratoDepth: 0.2, VibratoDelay: 0.3, Singers: 10, Detune: 20, Spread: 1, Jitter: 0.004,
		Shimmer: 0.02, Training: 0.8, Tuning: 1, Level: 1.913}
}

// ChildrenChoir is twelve children on "u", pure and a little out of tune
func ChildrenChoir() Config {
	return Config{Range: ChildVoice, Vowel: 4, VowelGlide: 0.25, Brightness: 0.35, Breath: 0.1, Attack: 0.2, Release: 0.5,
		VibratoRate: 5, VibratoDepth: 0.05, Singers: 12, Detune: 35, Spread: 1, Jitter: 0.008, Shimmer: 0.04, Level: 0.7269}
}

// Humming is one voice with the lips closed: the nose open, an "m" with a pitch
func Humming() Config {
	cfg := Solo()
	cfg.Vowel, cfg.Nasal, cfg.Brightness, cfg.Breath, cfg.Subglottal = 4, 1, 0.3, 0.03, 0.4
	cfg.Level = 1.032
	return cfg
}

// HummingChoir is eight voices humming together, close and warm
func HummingChoir() Config {
	cfg := Choir()
	cfg.Range, cfg.Vowel, cfg.Nasal, cfg.Brightness, cfg.Attack, cfg.Release = Alto, 3.5, 0.9, 0.25, 0.5, 1
	cfg.Level = 1.532
	return cfg
}

// WhisperSoft is a quiet whisper of an "a": air and no pitch
func WhisperSoft() Config {
	return Config{Range: Alto, Vowel: 0, Whisper: 1, Attack: 0.1, Release: 0.3, Singers: 1, Steady: true, Level: 1.297}
}

// WhisperBreath is a half whispered "o": the voice under a sigh of air
func WhisperBreath() Config {
	cfg := Solo()
	cfg.Vowel, cfg.Whisper, cfg.Breath, cfg.Brightness = 3, 0.55, 0.4, 0.3
	cfg.Level = 1.248
	return cfg
}

// ChantMantra sings "mo ma ni pa me u" one syllable a note, low and even
func ChantMantra() Config {
	cfg := Solo()
	cfg.Range = Bass
	cfg.Word = mustWord("mo ma ni pa me u")
	cfg.Release, cfg.Steady, cfg.Brightness, cfg.VibratoDepth = 0.25, true, 0.4, 0
	cfg.ConsonantLevel = 0.2
	cfg.Level = 1.121
	return cfg
}

// Kyrie sings "ki ri e e le i so ne" slowly, a tenor on the beat
func Kyrie() Config {
	cfg := Solo()
	cfg.Range = Tenor
	cfg.Word = mustWord("ki ri e e le i so ne")
	cfg.WordSync, cfg.Release, cfg.Attack, cfg.VibratoDepth, cfg.Brightness = 2, 0.2, 0.04, 0.12, 0.55
	cfg.Training = 0.5
	cfg.ConsonantLevel = 0.25
	cfg.Level = 1.756
	return cfg
}

// SpokenWords talks "ne lo wa ta ma ni": an alto speaking on the beat, no vibrato, no song
func SpokenWords() Config {
	cfg := Solo()
	cfg.Range = Alto
	cfg.Word = mustWord("ne lo wa ta ma ni")
	cfg.WordSync, cfg.Steady, cfg.VibratoDepth, cfg.Attack, cfg.Release = 0.5, true, 0, 0.02, 0.08
	cfg.Brightness, cfg.Coarticulation = 0.7, 0.06
	cfg.ConsonantLevel = 0.5
	cfg.Level = 0.9461
	return cfg
}

// OperaTenor is a trained tenor: the singer's formant, a wide natural vibrato, a full LF source
func OperaTenor() Config {
	cfg := Solo()
	cfg.Range, cfg.Vowel, cfg.Training, cfg.Source, cfg.Rd = Tenor, 0, 1, 1, 0.7
	cfg.NaturalVibrato, cfg.VibratoDepth, cfg.Brightness = 1, 0, 0.7
	cfg.Level = 1.289
	return cfg
}

// OperaSoprano is a trained soprano whose first formant follows the note: an open, ringing "a" with a natural vibrato
func OperaSoprano() Config {
	cfg := Solo()
	cfg.Range, cfg.Vowel, cfg.Training, cfg.Tuning, cfg.Source, cfg.Rd = Soprano, 0, 1, 1, 1, 0.4
	cfg.NaturalVibrato, cfg.VibratoDepth, cfg.Brightness = 1, 0, 0.75
	cfg.Level = 1.083
	return cfg
}

// OperaBass is a trained bass, "o" ringing in the singer's formant, a slow wide vibrato
func OperaBass() Config {
	cfg := Solo()
	cfg.Range, cfg.Vowel, cfg.Training, cfg.Source, cfg.Rd = Bass, 3, 1, 1, 0.6
	cfg.NaturalVibrato, cfg.VibratoDepth, cfg.Brightness, cfg.Release = 1, 0, 0.8, 0.5
	cfg.Level = 1.889
	return cfg
}

// PopBreathy is a close, intimate pop voice: an LF source with a high Rd and a lot of air
func PopBreathy() Config {
	cfg := Solo()
	cfg.Range, cfg.Vowel, cfg.Source, cfg.Rd, cfg.Breath = Alto, 1.5, 1, 2.2, 0.4
	cfg.NaturalVibrato, cfg.VibratoDepth, cfg.Brightness, cfg.Subglottal = 0.3, 0, 0.25, 0.5
	cfg.Level = 1.942
	return cfg
}

// PopBright is a forward, pressed pop voice, an LF source at a low Rd
func PopBright() Config {
	cfg := Solo()
	cfg.Range, cfg.Vowel, cfg.Source, cfg.Rd = Alto, 0.5, 1, 0.4
	cfg.VibratoDepth, cfg.Brightness, cfg.Breath = 0.08, 0.95, 0.02
	cfg.Level = 2.573
	return cfg
}

// GospelChoir is six voices wide open on "a", bright and loud, a fast shouting vibrato
func GospelChoir() Config {
	return Config{Range: Alto, Vowel: 0.3, VowelGlide: 0.2, Brightness: 0.9, Breath: 0.04, Attack: 0.1, Release: 0.5,
		VibratoRate: 6.5, VibratoDepth: 0.35, VibratoDelay: 0.1, Singers: 6, Detune: 25, Spread: 0.9, Jitter: 0.006,
		Shimmer: 0.04, Training: 0.7, Source: 1, Rd: 0.5, Level: 1.674}
}

// MonkDrone is four monks on a closed "o", a pitch held still for a long time
func MonkDrone() Config {
	return Config{Range: Bass, Vowel: 3.3, VowelGlide: 0.6, Brightness: 0.3, Breath: 0.03, Attack: 1, Release: 1.5,
		Singers: 4, Detune: 6, Spread: 0.5, Steady: true, Jitter: 0.002, Shimmer: 0.01, Level: 2.493}
}

// ThroatLow is a giant's throat: the formants shifted far down, a rough, pressed source
func ThroatLow() Config {
	return Config{Range: Bass, Vowel: 3.5, VowelGlide: 0.5, Brightness: 1, Breath: 0.1, Attack: 0.5, Release: 1,
		FormantShift: 0.6, Singers: 3, Detune: 15, Spread: 0.8, Jitter: 0.02, Shimmer: 0.15, Steady: true,
		OpenQuotient: 0.35, Level: 1.025}
}

// OvertoneSweep is a low voice whose vowel sweeps slowly up and down: the overtones whistle out of the drone
func OvertoneSweep() Config {
	return Config{Range: Bass, Vowel: 2, VowelGlide: 0.1, VowelRate: 0.25, VowelDepth: 2, Brightness: 0.85,
		Attack: 0.8, Release: 1.5, Singers: 2, Detune: 5, Spread: 0.6, Steady: true, Training: 0.5, Level: 1.514}
}

// PadWarm is ten voices on "o" opening slowly to "a": a warm, wide pad
func PadWarm() Config {
	return Config{Range: Alto, Vowel: 3, VowelGlide: 1.2, VowelRate: 0.08, VowelDepth: 1.5, Brightness: 0.35, Breath: 0.1,
		Attack: 1.5, Release: 2, VibratoRate: 4.8, VibratoDepth: 0.1, VibratoDelay: 0.8, Singers: 10, Detune: 24,
		Spread: 1, Jitter: 0.005, Shimmer: 0.03, Source: 1, Rd: 1.2, Level: 1.98}
}

// PadDark is a dark pad of low "u" voices, slow
func PadDark() Config {
	return Config{Range: Tenor, Vowel: 4, VowelGlide: 1, Brightness: 0.1, Breath: 0.12, Attack: 2.5, Release: 3,
		Singers: 12, Detune: 30, Spread: 1, Jitter: 0.006, Shimmer: 0.03, FormantShift: 0.85, Level: 2.09}
}

// PadAir is a pad of breath: a whisper of "e" voices, wide, with a faint tone under it
func PadAir() Config {
	return Config{Range: Soprano, Vowel: 1, VowelGlide: 0.8, VowelRate: 0.15, VowelDepth: 1, Whisper: 0.75, Breath: 0.3,
		Attack: 1.2, Release: 2, Singers: 6, Detune: 12, Spread: 1, Level: 3.304}
}

// SyllableStab is a short "ta" stabbed on the note: a tenor, a hard attack, gone at once
func SyllableStab() Config {
	cfg := Solo()
	cfg.Range = Tenor
	cfg.Word = mustWord("ta")
	cfg.Attack, cfg.Release, cfg.Steady, cfg.Brightness, cfg.VibratoDepth = 0.003, 0.08, true, 0.85, 0
	cfg.ConsonantLevel = 0.6
	cfg.Level = 0.4914
	return cfg
}

// ConsonantStab is a choir spitting "ke po": hard stops before an "o", short
func ConsonantStab() Config {
	cfg := Choir()
	cfg.Range = Bass
	cfg.Word = mustWord("ko pu")
	cfg.Attack, cfg.Release, cfg.Brightness, cfg.Singers, cfg.Detune = 0.005, 0.12, 0.8, 5, 20
	cfg.ConsonantLevel = 0.8
	cfg.Level = 0.7712
	return cfg
}

// DiphthongAi glides each note from "a" to "i", slowly: a long, open "ai"
func DiphthongAi() Config {
	cfg := Solo()
	cfg.Range, cfg.Phoneme, cfg.PhonemeTo = Soprano, 6, 1
	cfg.DiphthongTime, cfg.DiphthongHold, cfg.Training = 0.6, 0.2, 0.5
	cfg.Level = 1.632
	return cfg
}

// DiphthongOu glides each note from "o" to "u", a choir closing its mouth
func DiphthongOu() Config {
	cfg := Choir()
	cfg.Range, cfg.Phoneme, cfg.PhonemeTo = Tenor, 8, 10
	cfg.DiphthongTime, cfg.DiphthongHold, cfg.Singers, cfg.Brightness = 0.9, 0.1, 5, 0.3
	cfg.Level = 1.771
	return cfg
}

// LiquidLa sings "la le li lo lu ra re": liquids and glides moving into their vowels, one a note
func LiquidLa() Config {
	cfg := Solo()
	cfg.Range = ChildVoice
	cfg.Word = mustWord("la ye li wo lu ra")
	cfg.Release, cfg.Attack, cfg.VowelGlide, cfg.Brightness, cfg.Coarticulation = 0.12, 0.02, 0.04, 0.6, 0.08
	cfg.Level = 0.8254
	return cfg
}

// WheelVowel sings an "a" whose vowel the mod wheel moves toward "u" through the vowels between
func WheelVowel() Config {
	cfg := Solo()
	cfg.Range, cfg.Vowel, cfg.VowelGlide, cfg.Brightness, cfg.Training, cfg.VibratoDepth = Tenor, 0.8, 0.08, 0.5, 0.2, 0.1
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "vowel", Amount: 1}
	cfg.Mods[1] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "brightness", Amount: -0.4}
	cfg.Level = 0.6995
	return cfg
}

// WheelWhisper is a sung "e" the mod wheel turns into a whisper
func WheelWhisper() Config {
	cfg := Solo()
	cfg.Range, cfg.Vowel, cfg.Brightness, cfg.Breath = Soprano, 2.5, 0.6, 0.2
	cfg.Mods[0] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "whisper", Amount: 1}
	cfg.Mods[1] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "breath", Amount: 0.5}
	cfg.Level = 1.194
	return cfg
}
