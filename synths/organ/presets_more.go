package organ

import (
	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm/blocks/modmatrix"
	"github.com/almerlucke/psalm/blocks/voicefilter"
)

func init() {
	for name, f := range morePresets {
		Presets[name] = f
	}
}

// morePresets are the registrations of presets_more.go
var morePresets = map[string]func() Config{
	"jimmySmith":     JimmySmith,
	"funkyStab":      FunkyStab,
	"gospelFast":     GospelFast,
	"gospelSlow":     GospelSlow,
	"rockOverdrive":  RockOverdrive,
	"bluesGrit":      BluesGrit,
	"brightSolo":     BrightSolo,
	"softFlute":      SoftFlute,
	"nasalReed":      NasalReed,
	"softPad":        SoftPad,
	"theatreTibia":   TheatreTibia,
	"churchPlenum":   ChurchPlenum,
	"principalChoir": PrincipalChoir,
	"diapason":       Diapason,
	"stringOrgan":    StringOrgan,
	"cathedral":      Cathedral,
	"chorusV1":       ChorusV1,
	"warmChorus":     WarmChorus,
	"deepVibrato":    DeepVibrato,
	"leslieStopped":  LeslieStopped,
	"leslieFast":     LeslieFast,
	"subBass":        SubBass,
	"pipeFlute":      PipeFlute,
	"glassOrgan":     GlassOrgan,
	"carnival":       Carnival,
	"horror":         Horror,
	"ballpark":       Ballpark,
	"mellowFull":     MellowFull,
	"wheelLeslie":    WheelLeslie,
	"wheelDrive":     WheelDrive,
	"wahOrgan":       WahOrgan,
	"swellFilter":    SwellFilter,
}

// JimmySmith is 884002000 with a long third harmonic percussion, a little drive and the Leslie
// turning up to the chorale: the soul jazz trio organ
func JimmySmith() Config {
	return preset("884002000", func(c *Config) {
		c.Percussion, c.Vibrato, c.Drive, c.Click = PercThird, C3, 0.25, 0.7
		c.Leslie, c.LeslieSpeed, c.Tone = 0.8, 0.8, 0.4
		c.Level = 2.431
	})
}

// FunkyStab is the 8', 4' and 2 2/3' with the fast second percussion and a lot of click, dry: a
// clipped, clavinet like chop
func FunkyStab() Config {
	return preset("006800000", func(c *Config) {
		c.Percussion, c.PercFast, c.Click, c.ClickRand, c.Tone = PercSecond, true, 0.9, 0.5, 0.65
		c.Level = 3.113
	})
}

// GospelFast is 888800000 with the deep chorus into a Leslie at tremolo: the church band at full gallop
func GospelFast() Config {
	return preset("888800000", func(c *Config) {
		c.Vibrato, c.Drive = C3, 0.3
		c.Leslie, c.LeslieSpeed = 0.9, 1
		c.Level = 2.253
	})
}

// GospelSlow is 886000000 on the light chorus and the Leslie at its chorale: a sweet, round shout
func GospelSlow() Config {
	return preset("886000000", func(c *Config) {
		c.Vibrato, c.Click = C1, 0.3
		c.Leslie, c.LeslieSpeed, c.LesliePosition = 0.7, 0.3, 0.8
		c.Level = 2.553
	})
}

// RockOverdrive is 838808000 into a preamp driven hard and bright, the Leslie at tremolo, all the click
func RockOverdrive() Config {
	return preset("838808000", func(c *Config) {
		c.Drive, c.Click, c.Tone = 0.9, 0.9, 0.7
		c.Leslie, c.LeslieSpeed = 0.6, 1
		c.Level = 3.915
	})
}

// BluesGrit is 868000000 with a bit of drive, the light chorus and a slow Leslie: a smoky club organ
func BluesGrit() Config {
	return preset("868000000", func(c *Config) {
		c.Vibrato, c.Drive, c.Click, c.Leakage = C1, 0.5, 0.7, 0.6
		c.Leslie, c.LeslieSpeed, c.Tone = 0.5, 0.4, 0.4
		c.Level = 2.265
	})
}

// BrightSolo is the high drawbars of a lead line, 008888000 with the second percussion and a bright preamp
func BrightSolo() Config {
	return preset("008888000", func(c *Config) {
		c.Percussion, c.PercFast, c.Vibrato, c.Tone = PercSecond, true, V2, 0.75
		c.Leslie, c.LeslieSpeed = 0.4, 0.6
		c.Level = 2.543
	})
}

// SoftFlute is the 8', 4' and 2' very softly, 004840000, with the vibrato of a flute stop
func SoftFlute() Config {
	return preset("004840000", func(c *Config) {
		c.Vibrato, c.Click, c.Tone, c.Tonewheel = V3, 0.05, 0.3, 0.3
		c.Level = 1.848
	})
}

// NasalReed is 008060606: the odd mixtures over the 8', a hollow, oboe like reed
func NasalReed() Config {
	return preset("008060606", func(c *Config) {
		c.Vibrato, c.Tone, c.Click = C1, 0.6, 0.2
		c.Level = 2.416
	})
}

// SoftPad is a slow pad, 808000000, in a chorale Leslie
func SoftPad() Config {
	return preset("808000000", func(c *Config) {
		c.Vibrato, c.Click, c.Tone = C2, 0.05, 0.3
		c.Leslie, c.LeslieSpeed, c.LesliePosition = 0.9, 0.5, 1
		c.Level = 1.679
	})
}

// TheatreTibia is a heavy 8' tibia with the 16' and a deep slow vibrato: the cinema organ
func TheatreTibia() Config {
	return preset("808300000", func(c *Config) {
		c.Vibrato, c.VibratoScale, c.VibratoHz, c.Click = V3, 1.6, 6, 0.1
		c.Tone = 0.4
		c.Level = 1.705
	})
}

// ChurchPlenum is the whole chorus of a church organ, the 8' and 4' strong, the mixtures filling in, dry
func ChurchPlenum() Config {
	return preset("578765432", func(c *Config) {
		c.Click, c.Tone, c.Leakage = 0.2, 0.55, 0.3
		c.Level = 1.894
	})
}

// PrincipalChoir is the principals, 004865432: the 4' leading with no 16' under it
func PrincipalChoir() Config {
	return preset("004865432", func(c *Config) {
		c.Click, c.Tone = 0.15, 0.6
		c.Level = 2.487
	})
}

// Diapason is the foundation of a church organ, 006866420: round, full and without the 16'
func Diapason() Config {
	return preset("006866420", func(c *Config) {
		c.Click, c.Tone = 0.2, 0.35
		c.Level = 1.618
	})
}

// StringOrgan is the keen high drawbars of an organ string stop, 003003868, with the chorus
func StringOrgan() Config {
	return preset("003003868", func(c *Config) {
		c.Vibrato, c.Tone, c.Click = C3, 0.6, 0.1
		c.Level = 2.253
	})
}

// Cathedral is a low, dark registration, 863000600, a long way off: the highs rolled away
func Cathedral() Config {
	return preset("863000600", func(c *Config) {
		c.Tone, c.Click, c.Tonewheel = 0.15, 0.1, 1
		c.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 1800, Resonance: 0.1, KeyTrack: 0.5, Env: voicefilter.Default().Env}
		c.Level = 1.268
	})
}

// ChorusV1 is 338600000 with the lightest vibrato, V1, and no Leslie: the plain scanner
func ChorusV1() Config {
	return preset("338600000", func(c *Config) {
		c.Vibrato = V1
		c.Leslie = 0
		c.Level = 2.523
	})
}

// WarmChorus is 848000000 on the middle chorus, C2, straight into the amp
func WarmChorus() Config {
	return preset("848000000", func(c *Config) {
		c.Vibrato, c.Tone = C2, 0.4
		c.Level = 1.705
	})
}

// DeepVibrato is 008060000 on the deepest vibrato, V3 doubled, without the dry sound
func DeepVibrato() Config {
	return preset("008060000", func(c *Config) {
		c.Vibrato, c.VibratoScale, c.Click = V3, 2, 0.1
		c.Level = 2.569
	})
}

// LeslieStopped is 884400000 with the rotors parked: the cabinet's tone with no movement
func LeslieStopped() Config {
	return preset("884400000", func(c *Config) {
		c.Leslie, c.LeslieSpeed, c.LesliePosition = 1, 0, 0.2
		c.Level = 2.818
	})
}

// LeslieFast is 888800000 with the rotors at tremolo, the microphones far apart
func LeslieFast() Config {
	return preset("888800000", func(c *Config) {
		c.Leslie, c.LeslieSpeed, c.LesliePosition = 1, 1, 1
		c.Level = 1.733
	})
}

// SubBass is the 16' and the 5 1/3' alone, 870000000, dry and with plenty of click
func SubBass() Config {
	return preset("870000000", func(c *Config) {
		c.Click, c.Tone = 0.7, 0.2
		c.Level = 1.345
	})
}

// PipeFlute is a single 4', through a low pass: a hollow wooden pipe
func PipeFlute() Config {
	return preset("000800000", func(c *Config) {
		c.Vibrato, c.Click, c.Tonewheel = V1, 0.05, 0
		c.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 900, KeyTrack: 0.8, Env: voicefilter.Default().Env}
		c.Level = 3.717
	})
}

// GlassOrgan is only the highest drawbars, 000008806: a glassy, whistling organ
func GlassOrgan() Config {
	return preset("000008806", func(c *Config) {
		c.Vibrato, c.Tone, c.Click, c.Tonewheel = V2, 0.85, 0.05, 0
		c.Level = 1.256
	})
}

// Carnival is the fairground organ, 008806000 on the fast vibrato at 8 Hz, a little bright
func Carnival() Config {
	return preset("008806000", func(c *Config) {
		c.Vibrato, c.VibratoHz, c.VibratoScale, c.Tone = C3, 8.5, 1.5, 0.65
		c.Level = 2.802
	})
}

// Horror is a drooping 808000600 with the scanner at 3 Hz, deep: a haunted house organ
func Horror() Config {
	return preset("808000600", func(c *Config) {
		c.Vibrato, c.VibratoHz, c.VibratoScale, c.Tone, c.Leakage = V3, 3, 3, 0.25, 1
		c.Leslie, c.LeslieSpeed = 0.6, 0.15
		c.Level = 1.471
	})
}

// Ballpark is 838380000 with the fast second percussion and the tremolo Leslie: the stadium organ
func Ballpark() Config {
	return preset("838380000", func(c *Config) {
		c.Percussion, c.PercFast, c.Vibrato, c.Tone = PercSecond, true, V1, 0.65
		c.Leslie, c.LeslieSpeed = 0.7, 1
		c.Level = 3.225
	})
}

// MellowFull is every drawbar out, through a low pass: all the weight and none of the edge
func MellowFull() Config {
	return preset("888888888", func(c *Config) {
		c.Vibrato, c.Tone = C1, 0.2
		c.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 1100, Resonance: 0.15, KeyTrack: 0.4, Env: voicefilter.Default().Env}
		c.Leslie, c.LeslieSpeed = 0.5, 0.4
		c.Level = 1.082
	})
}

// WheelLeslie is 888800000 in the Leslie: the mod wheel speeds the rotors from the chorale up to the tremolo
func WheelLeslie() Config {
	return preset("888800000", func(c *Config) {
		c.Vibrato = C2
		c.Leslie, c.LeslieSpeed = 1, 0
		c.Mods[0] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "leslieSpeed", Amount: 1}
		c.Level = 2.057
	})
}

// WheelDrive is 808800000 clean until the mod wheel pushes the preamp into saturation
func WheelDrive() Config {
	return preset("808800000", func(c *Config) {
		c.Leslie, c.LeslieSpeed = 0.5, 0.7
		c.Mods[0] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "drive", Amount: 1}
		c.Mods[1] = modmatrix.Slot{Src: modmatrix.Wheel, Dest: "tone", Amount: 0.3}
		c.Level = 2.311
	})
}

// WahOrgan sweeps a resonant low pass over 888600000 with a slow LFO: a wah organ
func WahOrgan() Config {
	return preset("888600000", func(c *Config) {
		c.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 900, Resonance: 0.5, KeyTrack: 0.3, Env: voicefilter.Default().Env}
		c.LFO2Rate = 0.6
		c.Mods[0] = modmatrix.Slot{Src: modmatrix.LFO2, Dest: "cutoff", Amount: 0.5}
		c.Level = 1.728
	})
}

// SwellFilter opens a low pass over 888888888 with the first key of a phrase, like a swell pedal
func SwellFilter() Config {
	return preset("888888888", func(c *Config) {
		c.Vibrato = C2
		c.Filter = voicefilter.Config{Type: voicefilter.LowPass, Cutoff: 300, Resonance: 0.2, EnvAmount: 3, KeyTrack: 0.3,
			Env: *adsr.NewSetting(1.0, 800.0, 0.9, 600.0, 1.0, 400.0)}
		c.Leslie, c.LeslieSpeed = 0.5, 0.5
		c.Level = 1.829
	})
}
