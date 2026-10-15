package dx

import "github.com/almerlucke/psalm/blocks/modmatrix"

// More patches in the style of the DX7's factory banks and the TX81Z's: electric pianos, basses,
// brass, strings and pads, mallets, bells, plucks, organs, winds, leads, percussion and effects.
// They are made of the classic values (an algorithm, ratios by coarse and fine, output levels, 4 rate
// 4 level envelopes, feedback, key scaling), and a few of the additions of dx: the waves of the TX81Z
// (Operator.Wave), the filter, the glide and vibrato, a routing of its own and the modulation matrix
// (those are not part of the SysEx data)

func init() {
	for name, preset := range dxPresets {
		Presets[name] = preset
	}
}

// eg is an envelope: the rates 1-4 then the levels 1-4
type eg struct{ r, l [4]int }

func g(r1, r2, r3, r4, l1, l2, l3, l4 int) eg {
	return eg{[4]int{r1, r2, r3, r4}, [4]int{l1, l2, l3, l4}}
}

// x makes an operator: the coarse ratio, the fine ratio, the output level and the envelope
func x(coarse, fine, level int, e eg) Operator {
	o := opr(coarse, level, e.r, e.l)
	o.Fine = fine
	return o
}

func (o Operator) vel(n int) Operator    { o.VelocitySens = n; return o }
func (o Operator) det(n int) Operator    { o.Detune = n; return o }
func (o Operator) wav(n int) Operator    { o.Wave = n; return o }
func (o Operator) rscale(n int) Operator { o.RateScaling = n; return o }
func (o Operator) amod(n int) Operator   { o.AmpModSens = n; return o }

// fixed makes the frequency fixed: the decade (0-3 for 1 Hz to 1 kHz) and the fine value (a factor 1 to 10)
func (o Operator) fixed(decade, fine int) Operator {
	o.FixedFreq, o.Coarse, o.Fine = true, decade, fine
	return o
}

// kscale sets the keyboard level scaling: the break point, the depths and the curves of both sides
func (o Operator) kscale(bp, ld, rd, lc, rc int) Operator {
	o.BreakPoint, o.LeftDepth, o.RightDepth, o.LeftCurve, o.RightCurve = bp, ld, rd, lc, rc
	return o
}

// dxp makes a patch of an algorithm (1-32), feedback and the operators from operator 1 on (the others are silent)
func dxp(name string, alg, fb int, ops ...Operator) Patch {
	p := InitVoice()
	p.Name = name
	p.Algorithm = alg - 1
	p.Feedback = fb
	for i := range p.Ops {
		if i < len(ops) {
			p.Ops[i] = ops[i]
		} else {
			p.Ops[i].OutputLevel = 0
		}
	}
	return p
}

// lfo sets the LFO: speed, delay, pitch depth, amplitude depth, wave and the pitch sensitivity
func (p *Patch) lfo(speed, delay, pitch, amp, wave, sens int) {
	p.LFOSpeed, p.LFODelay, p.LFOPitchDepth, p.LFOAmpDepth, p.LFOWaveform, p.PitchModSens = speed, delay, pitch, amp, wave, sens
}

// pitchEG sets the pitch envelope: its rates and levels (the fourth level is where it starts)
func (p *Patch) pitchEG(r1, r2, r3, r4, l1, l2, l3, l4 int) {
	p.PitchEGRates, p.PitchEGLevels = [4]int{r1, r2, r3, r4}, [4]int{l1, l2, l3, l4}
}

// flt sets the filter: type (1 low pass, 2 band pass, 3 high pass, 4 notch, 5 ladder 24, 6 ladder 12),
// cutoff, resonance, envelope amount, key tracking and the attack, decay, sustain and release
func (p *Patch) flt(typ, cutoff, res, env, key, a, d, s, r int) {
	p.Filter = Filter{Type: typ, Cutoff: cutoff, Resonance: res, EnvAmount: env, KeyTrack: key, Attack: a, Decay: d, Sustain: s, Release: r}
	p.FilterSet = true
}

// mod puts a source on a destination in the next free slot of the matrix
func (p *Patch) mod(src modmatrix.Source, dest string, amount float64) {
	for i := range p.Mods {
		if p.Mods[i].Src == modmatrix.None {
			p.Mods[i] = modmatrix.Slot{Src: src, Dest: dest, Amount: amount}
			return
		}
	}
}

// dxPresets are the patches of this file, by name
var dxPresets = map[string]func() Patch{
	// electric pianos
	"tineKeys": TineKeys, "fmPiano": FMPiano, "softPiano": SoftPiano, "dynoKeys": DynoKeys, "wurlyKeys": WurlyKeys, "brightGrand": BrightGrand,
	// basses
	"slapFM": SlapFM, "synthBass": SynthBass, "fretlessFM": FretlessFM, "fmBass": FMBass, "acidBass": AcidBass, "rubberBass": RubberBass, "subBass": SubBass,
	// brass
	"brassSection": BrassSection, "soloTrumpet": SoloTrumpet, "hornSolo": HornSolo, "synthBrass": SynthBrass, "sforzandoBrass": SforzandoBrass,
	// strings and pads
	"warmStrings": WarmStrings, "glassPad": GlassPad, "icePad": IcePad, "evolvingPad": EvolvingPad, "choirPad": ChoirPad, "sweepStrings": SweepStrings,
	// mallets
	"woodMarimba": WoodMarimba, "vibesFM": VibesFM, "thumbKalimba": ThumbKalimba, "steelCelesta": SteelCelesta, "crystalGlock": CrystalGlock, "chanceMallet": ChanceMallet,
	// bells
	"tubeChime": TubeChime, "churchBell": ChurchBell, "musicBoxFM": MusicBoxFM, "gamelanBell": GamelanBell, "bellShimmer": BellShimmer,
	// plucks
	"harpFM": HarpFM, "nylonPick": NylonPick, "kotoPluck": KotoPluck, "banjoTwang": BanjoTwang, "baroqueHarpsi": BaroqueHarpsi, "funkClav": FunkClav,
	// organs
	"fmOrgan": FMOrgan, "pipeOrgan": PipeOrgan, "drawbarOrgan": DrawbarOrgan, "reedOrgan": ReedOrgan,
	// winds
	"airFlute": AirFlute, "woodClarinet": WoodClarinet, "nasalOboe": NasalOboe, "tenorSax": TenorSax, "panPipes": PanPipes,
	// leads
	"syncLead": SyncLead, "pwmLead": PWMLead, "pureSine": PureSine, "fmLead": FMLead, "glideLead": GlideLead,
	// percussion
	"tomFM": TomFM, "kickFM": KickFM, "timpaniFM": TimpaniFM, "snareFM": SnareFM, "hatFM": HatFM, "woodBlockFM": WoodBlockFM,
	// effects
	"riserSweep": RiserSweep, "laserZap": LaserZap, "windGust": WindGust, "vowelAh": VowelAh, "alienBubbles": AlienBubbles,
	// the waves of the TX81Z
	"hollowReed": HollowReed, "logSawBrass": LogSawBrass, "halfSineOrgan": HalfSineOrgan, "evenAbsPad": EvenAbsPad, "pulseClav": PulseClav,
	// the matrix and the filter
	"wheelKeys": WheelKeys, "velocityFM": VelocityFM, "filterPluck": FilterPluck, "wheelPad": WheelPad,
}

// --- electric pianos ---

// TineKeys is an electric piano of a tine: a bright ratio 14 strike that dies in a moment over a warm body
func TineKeys() Patch {
	p := dxp("TINE KEYS", 5, 4,
		x(1, 0, 99, g(99, 40, 22, 60, 99, 80, 0, 0)),
		x(14, 0, 66, g(99, 78, 40, 60, 99, 0, 0, 0)).vel(6),
		x(1, 0, 90, g(99, 35, 20, 60, 99, 70, 0, 0)).det(8),
		x(1, 0, 62, g(99, 55, 30, 60, 99, 25, 0, 0)).vel(5),
		x(2, 0, 70, g(99, 50, 25, 60, 99, 40, 0, 0)),
		x(1, 0, 58, g(99, 60, 30, 60, 99, 20, 0, 0)))
	p.Level = 0.9365 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// FMPiano is the electric piano of the factory banks: carriers of the same pitch, their modulators
// at ratio 1 that bark and settle, a high strike on top
func FMPiano() Patch {
	p := dxp("FM PIANO", 5, 5,
		x(1, 0, 99, g(99, 30, 22, 55, 99, 85, 0, 0)),
		x(1, 0, 80, g(99, 52, 30, 55, 99, 30, 0, 0)).vel(7),
		x(1, 0, 92, g(99, 28, 20, 55, 99, 80, 0, 0)).det(9),
		x(1, 0, 74, g(99, 58, 30, 55, 99, 20, 0, 0)).vel(6),
		x(1, 0, 80, g(99, 32, 20, 55, 99, 75, 0, 0)).det(5),
		x(7, 0, 62, g(99, 75, 40, 55, 99, 0, 0, 0)).vel(5))
	p.Level = 0.9071 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// SoftPiano is a mellow piano of two chains, soft in the attack and long in the decay
func SoftPiano() Patch {
	p := dxp("SOFT PIANO", 4, 3,
		x(1, 0, 99, g(97, 24, 18, 50, 99, 80, 0, 0)),
		x(1, 0, 74, g(97, 40, 25, 50, 99, 40, 0, 0)).vel(4),
		x(1, 0, 60, g(97, 50, 30, 50, 99, 10, 0, 0)).vel(5),
		x(2, 0, 80, g(97, 26, 18, 50, 99, 70, 0, 0)).det(8),
		x(1, 0, 66, g(97, 45, 25, 50, 99, 25, 0, 0)),
		x(3, 0, 58, g(98, 62, 35, 50, 99, 0, 0, 0)).vel(6))
	p.Level = 0.8908 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// DynoKeys is a Rhodes of the 70s with bark: an octave modulator with a hard velocity and the feedback loop of algorithm 6
func DynoKeys() Patch {
	p := dxp("DYNO KEYS", 6, 6,
		x(1, 0, 99, g(99, 32, 20, 55, 99, 82, 0, 0)),
		x(1, 0, 88, g(99, 45, 25, 55, 99, 22, 0, 0)).vel(7),
		x(1, 0, 94, g(99, 30, 18, 55, 99, 80, 0, 0)).det(10),
		x(2, 0, 76, g(99, 50, 28, 55, 99, 15, 0, 0)).vel(7),
		x(1, 0, 70, g(99, 38, 22, 55, 99, 60, 0, 0)).det(4),
		x(1, 0, 62, g(99, 60, 30, 55, 99, 20, 0, 0)).vel(5))
	p.Level = 0.9170 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// WurlyKeys is a reed piano: a sustaining body, a fierce feedback and a soft tremolo
func WurlyKeys() Patch {
	p := dxp("WURLY KEYS", 5, 7,
		x(1, 0, 99, g(99, 36, 20, 60, 99, 78, 20, 0)).amod(1),
		x(2, 0, 70, g(99, 50, 25, 60, 99, 45, 20, 0)).vel(5),
		x(1, 0, 90, g(99, 34, 20, 60, 99, 74, 15, 0)).det(6),
		x(1, 0, 66, g(99, 55, 30, 60, 99, 35, 10, 0)).vel(5),
		x(1, 0, 70, g(99, 40, 20, 60, 99, 60, 0, 0)).det(8),
		x(3, 0, 58, g(99, 70, 40, 60, 99, 15, 0, 0)))
	p.lfo(28, 10, 0, 18, 4, 0)
	p.Level = 0.8880 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// BrightGrand is a piano of four carriers in algorithm 21: hammer partials above the string, strong key scaling
func BrightGrand() Patch {
	p := dxp("BRT GRAND", 21, 3,
		x(1, 0, 99, g(99, 26, 18, 50, 99, 82, 0, 0)).kscale(39, 0, 30, 0, 0),
		x(2, 0, 84, g(99, 34, 20, 50, 99, 70, 0, 0)).det(8),
		x(1, 0, 66, g(99, 52, 28, 50, 99, 20, 0, 0)).vel(7),
		x(3, 0, 70, g(99, 32, 20, 50, 99, 55, 0, 0)).det(6),
		x(4, 0, 62, g(99, 40, 22, 50, 99, 45, 0, 0)),
		x(5, 0, 60, g(99, 80, 45, 50, 99, 0, 0, 0)).vel(6))
	p.Level = 1.0370 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// --- basses ---

// SlapFM is a slapped bass: the pop of a fast modulator over a round body in algorithm 16
func SlapFM() Patch {
	p := dxp("SLAP FM", 16, 5,
		x(1, 0, 99, g(99, 45, 30, 70, 99, 85, 50, 0)),
		x(1, 0, 78, g(99, 70, 40, 70, 99, 20, 0, 0)).vel(7),
		x(2, 0, 70, g(99, 80, 40, 70, 99, 0, 0, 0)).vel(6),
		x(1, 0, 60, g(99, 70, 40, 70, 99, 20, 0, 0)),
		x(3, 0, 66, g(99, 85, 45, 70, 99, 0, 0, 0)).vel(5),
		x(1, 0, 74, g(99, 60, 35, 70, 99, 30, 0, 0)))
	p.Level = 0.8851 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// SynthBass is a saw like bass: a heavy feedback modulator over a sub octave pair (algorithm 1)
func SynthBass() Patch {
	p := dxp("SYNTH BASS", 1, 7,
		x(1, 0, 99, g(99, 55, 40, 75, 99, 90, 85, 0)),
		x(0, 0, 66, g(99, 60, 40, 75, 99, 60, 40, 0)).vel(4),
		x(1, 0, 80, g(99, 60, 40, 75, 99, 85, 75, 0)).det(8),
		x(1, 0, 70, g(99, 75, 40, 75, 99, 50, 40, 0)),
		x(2, 0, 60, g(99, 80, 40, 75, 99, 20, 0, 0)).vel(5),
		x(1, 0, 78, g(99, 70, 40, 75, 99, 40, 30, 0)))
	p.Level = 0.9584 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// FretlessFM is a fretless bass: a slow growl as the modulators settle, a little vibrato
func FretlessFM() Patch {
	p := dxp("FRETLESS", 4, 4,
		x(1, 0, 99, g(92, 35, 28, 65, 99, 88, 60, 0)),
		x(1, 0, 70, g(85, 40, 28, 65, 99, 50, 30, 0)).vel(5),
		x(1, 0, 62, g(80, 45, 28, 65, 99, 30, 15, 0)).vel(5),
		x(1, 0, 78, g(92, 35, 28, 65, 99, 80, 55, 0)).det(9),
		x(2, 0, 60, g(88, 50, 30, 65, 99, 35, 15, 0)),
		x(1, 0, 55, g(85, 55, 30, 65, 99, 40, 20, 0)))
	p.lfo(36, 40, 12, 0, 0, 3)
	p.Level = 0.8957 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// FMBass is the DX7 bass: a sub carrier under a stack that clacks and settles (algorithm 10)
func FMBass() Patch {
	p := dxp("FM BASS", 10, 4,
		x(1, 0, 99, g(99, 50, 35, 70, 99, 88, 70, 0)),
		x(1, 0, 78, g(99, 62, 38, 70, 99, 45, 25, 0)).vel(6),
		x(1, 0, 70, g(99, 70, 40, 70, 99, 20, 0, 0)),
		x(0, 0, 90, g(99, 50, 35, 70, 99, 90, 75, 0)),
		x(1, 0, 70, g(99, 75, 40, 70, 99, 30, 0, 0)).vel(6),
		x(2, 0, 62, g(99, 80, 40, 70, 99, 0, 0, 0)))
	p.Level = 0.9373 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// AcidBass is a resonant bass: a log saw carrier through the 24 dB ladder whose envelope snaps shut
func AcidBass() Patch {
	p := dxp("ACID BASS", 1, 6,
		x(1, 0, 99, g(99, 45, 40, 80, 99, 92, 85, 0)).wav(7),
		x(1, 0, 60, g(99, 55, 40, 80, 99, 50, 40, 0)),
		x(1, 0, 70, g(99, 45, 40, 80, 99, 90, 85, 0)).det(9).wav(6),
		x(1, 0, 40, g(99, 55, 40, 80, 99, 50, 40, 0)),
		x(2, 0, 30, g(99, 55, 40, 80, 99, 40, 30, 0)),
		x(1, 0, 40, g(99, 55, 40, 80, 99, 40, 30, 0)))
	p.flt(5, 28, 72, 62, 40, 0, 40, 18, 35)
	p.Level = 1.1840 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// RubberBass is a rubbery bass: a pitch envelope that snaps down into the note on a sine and a square pair
func RubberBass() Patch {
	p := dxp("RUBBERBASS", 8, 3,
		x(1, 0, 99, g(99, 55, 35, 70, 99, 90, 70, 0)),
		x(1, 0, 64, g(99, 62, 38, 70, 99, 40, 20, 0)).vel(5),
		x(1, 0, 80, g(99, 55, 35, 70, 99, 85, 60, 0)).wav(6).det(5),
		x(2, 0, 62, g(99, 60, 38, 70, 99, 30, 10, 0)),
		x(1, 0, 55, g(99, 70, 38, 70, 99, 40, 0, 0)),
		x(3, 0, 50, g(99, 75, 40, 70, 99, 0, 0, 0)))
	p.pitchEG(80, 99, 99, 99, 50, 50, 50, 64)
	p.Level = 0.9317 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// SubBass is a low sine with just a breath of feedback, an octave down
func SubBass() Patch {
	p := dxp("SUB BASS", 3, 2,
		x(1, 0, 99, g(96, 40, 40, 70, 99, 95, 90, 0)),
		x(1, 0, 44, g(96, 50, 40, 70, 99, 60, 40, 0)).vel(4),
		x(1, 0, 38, g(96, 50, 40, 70, 99, 60, 30, 0)),
		x(1, 0, 60, g(96, 40, 40, 70, 99, 90, 80, 0)),
		x(2, 0, 42, g(96, 60, 40, 70, 99, 30, 0, 0)),
		x(1, 0, 40, g(96, 60, 40, 70, 99, 30, 0, 0)))
	p.Transpose = 12
	p.Level = 0.6974 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// --- brass ---

// BrassSection is a brass section: a modulator that swells over four detuned carriers (algorithm 23)
func BrassSection() Patch {
	p := dxp("BRASS SECT", 23, 6,
		x(1, 0, 99, g(70, 40, 30, 55, 99, 95, 90, 0)),
		x(1, 0, 90, g(68, 40, 30, 55, 99, 93, 88, 0)).det(9),
		x(1, 0, 78, g(62, 40, 30, 55, 99, 85, 75, 0)).vel(4),
		x(1, 0, 90, g(66, 40, 30, 55, 99, 93, 88, 0)).det(5),
		x(1, 0, 88, g(66, 40, 30, 55, 99, 93, 88, 0)).det(11),
		x(1, 0, 78, g(58, 40, 30, 55, 99, 88, 80, 0)).vel(4))
	p.lfo(30, 50, 6, 0, 4, 3)
	p.Level = 0.8964 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// SoloTrumpet is a trumpet: a bright blare that breaks in with the attack, then a vibrato that comes in late
func SoloTrumpet() Patch {
	p := dxp("TRUMPET 1", 27, 3,
		x(1, 0, 99, g(88, 40, 30, 60, 99, 96, 92, 0)),
		x(1, 0, 94, g(88, 40, 30, 60, 99, 95, 90, 0)).det(8),
		x(1, 0, 74, g(84, 45, 30, 60, 99, 80, 70, 0)).vel(5),
		x(1, 0, 92, g(88, 40, 30, 60, 99, 94, 90, 0)).det(6),
		x(1, 0, 78, g(86, 42, 30, 60, 99, 85, 75, 0)).vel(4),
		x(1, 0, 84, g(84, 38, 30, 60, 99, 80, 70, 0)).vel(5))
	p.lfo(34, 60, 12, 0, 0, 3)
	p.Level = 0.8045 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// HornSolo is a horn: mellow, a slow attack, and the modulator on a lower level
func HornSolo() Patch {
	p := dxp("HORN SOLO", 2, 4,
		x(1, 0, 99, g(70, 40, 35, 55, 99, 96, 92, 0)),
		x(1, 0, 70, g(66, 40, 35, 55, 99, 85, 75, 0)).vel(4),
		x(1, 0, 70, g(70, 40, 35, 55, 99, 94, 90, 0)).det(8),
		x(1, 0, 60, g(66, 40, 35, 55, 99, 85, 75, 0)),
		x(1, 0, 62, g(66, 40, 35, 55, 99, 80, 70, 0)).vel(3),
		x(2, 0, 50, g(60, 40, 35, 55, 99, 70, 60, 0)))
	p.lfo(32, 70, 8, 0, 0, 3)
	p.Level = 0.5626 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// SynthBrass is an analogue like brass: a seven feedback modulator on a stack of five detuned carriers (algorithm 24)
func SynthBrass() Patch {
	p := dxp("SYN BRASS", 24, 7,
		x(1, 0, 99, g(80, 50, 40, 55, 99, 96, 94, 0)).det(7),
		x(1, 0, 94, g(80, 50, 40, 55, 99, 95, 92, 0)).det(10),
		x(1, 0, 94, g(80, 50, 40, 55, 99, 95, 92, 0)).det(4),
		x(1, 0, 90, g(78, 50, 40, 55, 99, 95, 92, 0)).det(12),
		x(1, 0, 90, g(78, 50, 40, 55, 99, 95, 92, 0)).det(2),
		x(1, 0, 80, g(76, 50, 40, 55, 99, 90, 85, 0)).vel(4))
	p.Level = 1.0913 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// SforzandoBrass is a stab: the modulators open wide at the start and close again, harder when played hard
func SforzandoBrass() Patch {
	p := dxp("SFORZANDO", 7, 4,
		x(1, 0, 99, g(92, 40, 30, 60, 99, 94, 88, 0)),
		x(1, 0, 90, g(92, 55, 30, 60, 99, 40, 25, 0)).vel(7),
		x(1, 0, 94, g(92, 40, 30, 60, 99, 94, 88, 0)).det(9),
		x(1, 0, 88, g(92, 55, 30, 60, 99, 45, 30, 0)).vel(7),
		x(1, 0, 84, g(92, 50, 30, 60, 99, 40, 28, 0)).vel(7),
		x(1, 0, 70, g(92, 50, 30, 60, 99, 60, 45, 0)))
	p.Level = 0.9467 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// --- strings and pads ---

// WarmStrings is a warm string section: slow attacks, detuned carriers over a soft modulator (algorithm 26)
func WarmStrings() Patch {
	p := dxp("WARM STR", 26, 3,
		x(1, 0, 99, g(48, 40, 30, 45, 99, 96, 92, 0)).det(6),
		x(1, 0, 92, g(46, 40, 30, 45, 99, 95, 90, 0)).det(9),
		x(1, 0, 64, g(44, 40, 30, 45, 99, 85, 80, 0)).vel(3),
		x(1, 0, 90, g(48, 40, 30, 45, 99, 95, 90, 0)).det(4),
		x(1, 0, 60, g(44, 40, 30, 45, 99, 85, 80, 0)),
		x(2, 0, 56, g(44, 40, 30, 45, 99, 80, 75, 0)))
	p.lfo(28, 30, 10, 0, 4, 2)
	p.Level = 1.0000 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// GlassPad is a pad of odd partials: ratios 1, 3 and 5 as pairs, a slow swell and a high shimmer (algorithm 6)
func GlassPad() Patch {
	p := dxp("GLASS PAD", 6, 3,
		x(1, 0, 90, g(40, 35, 25, 40, 99, 92, 85, 0)),
		x(3, 0, 62, g(38, 35, 25, 40, 99, 80, 70, 0)),
		x(1, 0, 86, g(42, 35, 25, 40, 99, 92, 85, 0)).det(11),
		x(5, 0, 58, g(38, 35, 25, 40, 99, 75, 65, 0)),
		x(2, 0, 80, g(40, 35, 25, 40, 99, 90, 82, 0)).det(3),
		x(7, 0, 54, g(36, 35, 25, 40, 99, 70, 55, 0)))
	p.Level = 1.0185 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// IcePad is a cold pad: carriers at 1, 2 and the top (algorithm 28) with high inharmonic modulators
func IcePad() Patch {
	p := dxp("ICE PAD", 28, 4,
		x(1, 0, 92, g(36, 30, 22, 38, 99, 90, 80, 0)),
		x(7, 30, 60, g(34, 30, 22, 38, 99, 70, 55, 0)),
		x(2, 0, 82, g(36, 30, 22, 38, 99, 88, 78, 0)).det(9),
		x(5, 0, 55, g(34, 30, 22, 38, 99, 60, 45, 0)),
		x(11, 0, 50, g(32, 30, 22, 38, 99, 55, 40, 0)),
		x(4, 0, 74, g(36, 30, 22, 38, 99, 85, 70, 0)).det(5))
	p.lfo(22, 25, 0, 12, 4, 0)
	p.Ops[5].AmpModSens = 1
	p.Level = 1.0372 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// EvolvingPad is a pad that changes: modulators that swell and fade at different speeds, a slow LFO
func EvolvingPad() Patch {
	p := dxp("EVOLVING", 14, 5,
		x(1, 0, 94, g(44, 40, 30, 40, 99, 94, 90, 0)),
		x(1, 0, 60, g(30, 14, 30, 40, 40, 90, 50, 0)),
		x(1, 0, 90, g(44, 40, 30, 40, 99, 94, 90, 0)).det(10),
		x(2, 0, 62, g(28, 12, 30, 40, 30, 92, 45, 0)),
		x(3, 0, 54, g(34, 18, 25, 40, 45, 85, 40, 0)).det(4),
		x(1, 0, 70, g(26, 20, 30, 40, 35, 90, 60, 0)))
	p.lfo(8, 40, 4, 20, 4, 1)
	p.Ops[0].AmpModSens = 1
	p.Level = 0.8651 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// ChoirPad is a choir of "ah": detuned voices with a formant like pair on two ratios (algorithm 20)
func ChoirPad() Patch {
	p := dxp("CHOIR PAD", 20, 3,
		x(1, 0, 94, g(50, 40, 30, 45, 99, 94, 88, 0)).det(6),
		x(1, 0, 90, g(52, 40, 30, 45, 99, 94, 88, 0)).det(9),
		x(3, 0, 70, g(46, 40, 30, 45, 99, 85, 75, 0)).vel(3),
		x(1, 0, 92, g(50, 40, 30, 45, 99, 94, 88, 0)).det(4),
		x(2, 0, 66, g(46, 40, 30, 45, 99, 80, 70, 0)),
		x(5, 0, 60, g(46, 40, 30, 45, 99, 70, 60, 0)))
	p.lfo(30, 45, 8, 0, 4, 2)
	p.Level = 0.9307 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// SweepStrings is a string machine: six detuned saw and pulse carriers (algorithm 32) through a low pass that opens and closes
func SweepStrings() Patch {
	p := dxp("SWEEP STR", 32, 2,
		x(1, 0, 90, g(55, 40, 30, 45, 99, 96, 92, 0)).wav(7).det(5),
		x(1, 0, 86, g(55, 40, 30, 45, 99, 96, 92, 0)).wav(7).det(9),
		x(1, 0, 84, g(55, 40, 30, 45, 99, 96, 92, 0)).wav(6).det(3),
		x(2, 0, 70, g(55, 40, 30, 45, 99, 96, 92, 0)).wav(7).det(11),
		x(1, 0, 80, g(55, 40, 30, 45, 99, 96, 92, 0)).wav(6).det(7),
		x(1, 0, 78, g(55, 40, 30, 45, 99, 96, 92, 0)).wav(7).det(1))
	p.flt(6, 38, 28, 40, 60, 55, 55, 55, 45)
	p.Level = 1.4027 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// --- mallets ---

// WoodMarimba is a marimba: a sine with a short ratio 4 knock, and a sub octave warmth (algorithm 5)
func WoodMarimba() Patch {
	p := dxp("WOOD MRMB", 5, 0,
		x(1, 0, 99, g(99, 50, 30, 80, 99, 40, 0, 0)).rscale(2),
		x(4, 0, 74, g(99, 85, 50, 80, 99, 0, 0, 0)).vel(6),
		x(0, 0, 70, g(99, 45, 30, 80, 99, 30, 0, 0)),
		x(1, 0, 40, g(99, 70, 40, 80, 99, 0, 0, 0)),
		x(10, 0, 56, g(99, 90, 60, 80, 99, 0, 0, 0)).vel(7),
		x(1, 0, 20, g(99, 90, 60, 80, 99, 0, 0, 0)))
	p.Level = 1.0000 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// VibesFM is a vibraphone: a long ring with a ratio 4 strike and a motor tremolo (algorithm 5)
func VibesFM() Patch {
	p := dxp("VIBES FM", 5, 0,
		x(1, 0, 99, g(99, 28, 22, 60, 99, 70, 0, 0)).amod(3),
		x(4, 0, 66, g(99, 60, 35, 60, 99, 0, 0, 0)).vel(6),
		x(1, 0, 70, g(99, 28, 22, 60, 99, 50, 0, 0)).det(7).amod(3),
		x(1, 0, 40, g(99, 50, 30, 60, 99, 0, 0, 0)),
		x(3, 50, 40, g(99, 45, 25, 60, 99, 25, 0, 0)).amod(3),
		x(1, 0, 24, g(99, 60, 30, 60, 99, 0, 0, 0)))
	p.lfo(34, 0, 0, 40, 4, 0)
	p.Level = 1.0821 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// ThumbKalimba is a thumb piano: a hollow tone from the even sine, a tick of an inharmonic partial
func ThumbKalimba() Patch {
	p := dxp("THUMB KLMB", 5, 2,
		x(1, 0, 99, g(99, 42, 28, 70, 99, 30, 0, 0)).wav(4),
		x(5, 20, 62, g(99, 80, 45, 70, 99, 0, 0, 0)).vel(6),
		x(1, 0, 70, g(99, 45, 30, 70, 99, 20, 0, 0)).wav(2).det(8),
		x(1, 0, 40, g(99, 70, 40, 70, 99, 0, 0, 0)),
		x(2, 0, 60, g(99, 55, 30, 70, 99, 20, 0, 0)),
		x(1, 0, 30, g(99, 80, 40, 70, 99, 0, 0, 0)))
	p.Level = 0.9127 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// SteelCelesta is a celesta an octave up: soft bell partials, no sustain (algorithm 5)
func SteelCelesta() Patch {
	p := dxp("STL CELEST", 5, 0,
		x(1, 0, 99, g(99, 36, 25, 70, 99, 30, 0, 0)),
		x(7, 0, 54, g(99, 70, 40, 70, 99, 0, 0, 0)).vel(6),
		x(2, 0, 72, g(99, 40, 28, 70, 99, 20, 0, 0)).det(8),
		x(3, 50, 46, g(99, 75, 40, 70, 99, 0, 0, 0)),
		x(4, 0, 60, g(99, 45, 28, 70, 99, 10, 0, 0)),
		x(11, 0, 36, g(99, 85, 50, 70, 99, 0, 0, 0)).vel(6))
	p.Transpose = 36
	p.Level = 0.9798 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// CrystalGlock is a glockenspiel: ratio 3.5 strikes, a ring that lasts (algorithm 31)
func CrystalGlock() Patch {
	p := dxp("CRYST GLCK", 31, 3,
		x(1, 0, 99, g(99, 32, 24, 60, 99, 60, 0, 0)),
		x(3, 50, 72, g(99, 58, 30, 60, 99, 10, 0, 0)).vel(6),
		x(2, 76, 70, g(99, 38, 26, 60, 99, 40, 0, 0)),
		x(5, 40, 64, g(99, 48, 28, 60, 99, 20, 0, 0)),
		x(6, 0, 58, g(99, 50, 30, 60, 99, 10, 0, 0)).vel(5),
		x(1, 0, 50, g(99, 90, 40, 60, 99, 0, 0, 0)).vel(7))
	p.Transpose = 36
	p.Level = 0.9563 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// ChanceMallet is a marimba whose strike changes with every note: the matrix moves the knock by a random value
func ChanceMallet() Patch {
	p := dxp("CHANCE MLT", 5, 1,
		x(1, 0, 99, g(99, 48, 30, 80, 99, 40, 0, 0)),
		x(3, 0, 60, g(99, 82, 50, 80, 99, 0, 0, 0)).vel(5),
		x(1, 0, 74, g(99, 44, 30, 80, 99, 30, 0, 0)).det(8),
		x(4, 0, 44, g(99, 80, 50, 80, 99, 0, 0, 0)),
		x(2, 0, 64, g(99, 55, 32, 80, 99, 15, 0, 0)),
		x(7, 0, 40, g(99, 85, 50, 80, 99, 0, 0, 0)))
	p.LFO2Rate = 0.5
	p.mod(modmatrix.Random, "op2Gain", 0.25)
	p.mod(modmatrix.Random, "op4Gain", 0.2)
	p.mod(modmatrix.Random, "pitch", 0.03)
	p.Level = 1.0378 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// --- bells ---

// TubeChime is a tubular bell: strike partials of 1, 2.76 and 5.4 that ring for seconds (algorithm 5)
func TubeChime() Patch {
	p := dxp("TUBE CHIME", 5, 0,
		x(1, 0, 99, g(99, 18, 14, 40, 99, 80, 0, 0)),
		x(3, 50, 70, g(99, 40, 22, 40, 99, 30, 0, 0)).vel(5),
		x(2, 76, 78, g(99, 22, 16, 40, 99, 60, 0, 0)),
		x(4, 0, 56, g(99, 45, 25, 40, 99, 20, 0, 0)),
		x(5, 40, 70, g(99, 25, 18, 40, 99, 50, 0, 0)),
		x(1, 0, 52, g(99, 60, 30, 40, 99, 20, 0, 0)).vel(6))
	p.Level = 0.9073 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// ChurchBell is a big bell: a low hum, the minor third and a strike of inharmonic partials (algorithm 1)
func ChurchBell() Patch {
	p := dxp("CHURCH BEL", 1, 2,
		x(1, 0, 99, g(99, 10, 8, 35, 99, 85, 0, 0)),
		x(2, 36, 62, g(99, 22, 14, 35, 99, 40, 0, 0)).vel(5),
		x(0, 0, 80, g(99, 12, 10, 35, 99, 70, 0, 0)),
		x(2, 76, 60, g(99, 25, 16, 35, 99, 40, 0, 0)),
		x(3, 14, 56, g(99, 40, 20, 35, 99, 20, 0, 0)),
		x(5, 40, 50, g(99, 55, 25, 35, 99, 10, 0, 0)).vel(6))
	p.Transpose = 12
	p.Level = 0.6656 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// MusicBoxFM is a music box: the tines of a comb, high and plain with a faint ratio 7 sparkle
func MusicBoxFM() Patch {
	p := dxp("MUSIC BOX", 5, 0,
		x(1, 0, 99, g(99, 40, 28, 70, 99, 20, 0, 0)),
		x(7, 0, 50, g(99, 75, 40, 70, 99, 0, 0, 0)).vel(5),
		x(3, 0, 70, g(99, 45, 30, 70, 99, 10, 0, 0)).det(8),
		x(9, 0, 36, g(99, 80, 40, 70, 99, 0, 0, 0)),
		x(5, 0, 54, g(99, 50, 32, 70, 99, 10, 0, 0)),
		x(1, 0, 30, g(99, 80, 40, 70, 99, 0, 0, 0)))
	p.Transpose = 36
	p.Level = 1.0249 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// GamelanBell is a gamelan gong chime: pairs a few cents apart beat slowly (algorithm 4)
func GamelanBell() Patch {
	p := dxp("GAMELAN", 4, 3,
		x(1, 0, 99, g(99, 16, 12, 45, 99, 80, 0, 0)).det(5),
		x(1, 41, 62, g(99, 32, 20, 45, 99, 40, 0, 0)),
		x(2, 33, 60, g(99, 40, 24, 45, 99, 15, 0, 0)).vel(5),
		x(1, 0, 90, g(99, 16, 12, 45, 99, 75, 0, 0)).det(10),
		x(3, 5, 60, g(99, 35, 22, 45, 99, 30, 0, 0)),
		x(1, 83, 56, g(99, 45, 26, 45, 99, 20, 0, 0)))
	p.Level = 0.8309 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// BellShimmer is a pad of bells: slow attacks of inharmonic partials with an LFO that makes them shimmer (algorithm 10)
func BellShimmer() Patch {
	p := dxp("BELL SHMR", 10, 2,
		x(1, 0, 94, g(44, 25, 20, 40, 99, 88, 70, 0)).amod(1),
		x(3, 50, 58, g(40, 25, 20, 40, 99, 70, 50, 0)),
		x(1, 0, 40, g(40, 25, 20, 40, 99, 70, 50, 0)),
		x(2, 0, 84, g(46, 25, 20, 40, 99, 85, 65, 0)).det(9).amod(2),
		x(5, 20, 56, g(40, 25, 20, 40, 99, 60, 40, 0)),
		x(7, 0, 50, g(38, 25, 20, 40, 99, 55, 35, 0)))
	p.lfo(24, 20, 0, 40, 4, 0)
	p.Level = 1.1261 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// --- plucks ---

// HarpFM is a harp: a soft strike, a ring and a rising pitch at the very start (algorithm 5)
func HarpFM() Patch {
	p := dxp("HARP  FM", 5, 1,
		x(1, 0, 99, g(99, 34, 22, 60, 99, 70, 0, 0)),
		x(2, 0, 66, g(99, 60, 35, 60, 99, 5, 0, 0)).vel(6),
		x(2, 0, 70, g(99, 38, 24, 60, 99, 50, 0, 0)).det(8),
		x(3, 0, 50, g(99, 70, 40, 60, 99, 0, 0, 0)),
		x(1, 0, 54, g(99, 40, 24, 60, 99, 30, 0, 0)),
		x(5, 0, 44, g(99, 80, 45, 60, 99, 0, 0, 0)).vel(5))
	p.Level = 0.9086 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// NylonPick is a nylon string guitar: the body of the first partials and a soft pick (algorithm 2)
func NylonPick() Patch {
	p := dxp("NYLON PCK", 2, 3,
		x(1, 0, 99, g(99, 40, 26, 65, 99, 70, 0, 0)),
		x(2, 0, 64, g(99, 62, 36, 65, 99, 10, 0, 0)).vel(6),
		x(1, 0, 70, g(99, 44, 28, 65, 99, 40, 0, 0)).det(7),
		x(1, 0, 60, g(99, 55, 32, 65, 99, 10, 0, 0)),
		x(3, 0, 54, g(99, 75, 40, 65, 99, 0, 0, 0)).vel(5),
		x(1, 0, 60, g(99, 70, 40, 65, 99, 20, 0, 0)))
	p.Level = 0.8824 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// KotoPluck is a koto: a bright pluck that bends down into the pitch, with a bite of the square wave (algorithm 4)
func KotoPluck() Patch {
	p := dxp("KOTO PLCK", 4, 5,
		x(1, 0, 99, g(99, 34, 24, 60, 99, 60, 0, 0)),
		x(1, 0, 78, g(99, 62, 36, 60, 99, 15, 0, 0)).vel(6),
		x(3, 0, 62, g(99, 72, 40, 60, 99, 0, 0, 0)).vel(6),
		x(2, 0, 80, g(99, 40, 26, 60, 99, 40, 0, 0)).wav(2),
		x(1, 0, 66, g(99, 60, 34, 60, 99, 12, 0, 0)),
		x(4, 0, 58, g(99, 78, 40, 60, 99, 0, 0, 0)))
	p.pitchEG(99, 70, 99, 99, 50, 50, 50, 58)
	p.Level = 0.8914 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// BanjoTwang is a banjo: a bright, tinny head, fast to die away (algorithm 3)
func BanjoTwang() Patch {
	p := dxp("BANJO TWG", 3, 5,
		x(1, 0, 99, g(99, 55, 40, 65, 99, 40, 0, 0)),
		x(1, 0, 70, g(99, 70, 40, 65, 99, 20, 0, 0)).vel(6),
		x(2, 0, 62, g(99, 78, 40, 65, 99, 0, 0, 0)),
		x(2, 0, 90, g(99, 50, 38, 65, 99, 30, 0, 0)).det(8),
		x(3, 0, 70, g(99, 70, 40, 65, 99, 10, 0, 0)).vel(7),
		x(5, 0, 64, g(99, 82, 45, 65, 99, 0, 0, 0)).vel(7))
	p.Level = 0.9608 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// BaroqueHarpsi is a harpsichord: plucked plectra of every harmonic, with a release click of the jack (algorithm 4)
func BaroqueHarpsi() Patch {
	p := dxp("HARPSICHRD", 4, 6,
		x(1, 0, 99, g(99, 40, 35, 80, 99, 70, 45, 0)),
		x(1, 0, 74, g(99, 60, 40, 80, 99, 40, 20, 0)),
		x(3, 0, 60, g(99, 70, 40, 80, 99, 20, 0, 0)),
		x(2, 0, 90, g(99, 40, 35, 80, 99, 70, 50, 0)).det(8),
		x(2, 0, 66, g(99, 55, 40, 80, 99, 30, 15, 0)),
		x(5, 0, 60, g(99, 78, 45, 80, 99, 0, 0, 0)))
	p.Level = 0.8778 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// FunkClav is a clavinet: a rasp of the feedback and a pickup that is hit hard, short (algorithm 5)
func FunkClav() Patch {
	p := dxp("FUNK CLAV", 5, 7,
		x(1, 0, 99, g(99, 55, 45, 78, 99, 50, 30, 0)),
		x(3, 0, 78, g(99, 70, 45, 78, 99, 30, 10, 0)).vel(7),
		x(2, 0, 80, g(99, 55, 45, 78, 99, 50, 25, 0)).det(9),
		x(1, 0, 64, g(99, 65, 45, 78, 99, 30, 10, 0)).vel(6),
		x(4, 0, 66, g(99, 62, 45, 78, 99, 40, 20, 0)),
		x(1, 0, 74, g(99, 70, 45, 78, 99, 30, 10, 0)).vel(6))
	p.Level = 0.9159 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// --- organs ---

// FMOrgan is an organ of six drawbars at 1/2, 1, 2, 3, 4 and 6 times the note (algorithm 32)
func FMOrgan() Patch {
	gate := g(99, 99, 99, 80, 99, 99, 99, 0)
	p := dxp("FM ORGAN", 32, 2,
		x(0, 0, 90, gate),
		x(1, 0, 94, gate),
		x(2, 0, 90, gate),
		x(3, 0, 80, gate),
		x(4, 0, 74, gate),
		x(6, 0, 66, gate))
	p.lfo(40, 0, 0, 0, 4, 0)
	p.Level = 0.5966 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// PipeOrgan is a pipe organ: sines with a chiff on the breath of the first moment (algorithm 31)
func PipeOrgan() Patch {
	gate := g(80, 99, 99, 60, 99, 99, 99, 0)
	p := dxp("PIPE ORGAN", 31, 5,
		x(1, 0, 94, gate),
		x(2, 0, 86, g(78, 99, 99, 60, 99, 99, 99, 0)),
		x(4, 0, 78, g(76, 99, 99, 60, 99, 99, 99, 0)),
		x(0, 0, 86, gate),
		x(3, 0, 70, g(74, 99, 99, 60, 99, 99, 99, 0)),
		x(12, 0, 62, g(99, 80, 40, 60, 99, 20, 0, 0)))
	p.Level = 0.5503 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// DrawbarOrgan is a tonewheel organ: three drawbars with the percussion of the third harmonic (algorithm 29)
func DrawbarOrgan() Patch {
	gate := g(99, 99, 99, 85, 99, 99, 99, 0)
	p := dxp("DRAWBAR 1", 29, 3,
		x(1, 0, 94, gate),
		x(2, 0, 84, gate),
		x(1, 0, 86, gate),
		x(3, 0, 56, g(99, 80, 50, 85, 99, 0, 0, 0)).vel(4),
		x(0, 0, 90, gate),
		x(1, 0, 46, g(99, 99, 99, 85, 99, 99, 99, 0)))
	p.lfo(38, 0, 0, 0, 4, 0)
	p.Level = 0.4463 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// ReedOrgan is a reed organ, the harmonium: the square wave of the TX81Z and a soft attack
func ReedOrgan() Patch {
	p := dxp("REED ORGAN", 29, 3,
		x(1, 0, 90, g(80, 99, 99, 70, 99, 99, 99, 0)).wav(6),
		x(1, 0, 60, g(80, 99, 99, 70, 99, 99, 99, 0)),
		x(1, 0, 80, g(78, 99, 99, 70, 99, 99, 99, 0)).wav(2).det(9),
		x(2, 0, 50, g(78, 99, 99, 70, 99, 99, 99, 0)),
		x(2, 0, 70, g(78, 99, 99, 70, 99, 99, 99, 0)).wav(1),
		x(1, 0, 40, g(78, 99, 99, 70, 99, 99, 99, 0)))
	p.lfo(36, 20, 4, 0, 4, 1)
	p.Level = 0.4025 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// --- winds ---

// AirFlute is a flute: a sine with a breathy edge from a feedback modulator that comes in with the attack (algorithm 3)
func AirFlute() Patch {
	p := dxp("AIR FLUTE", 3, 4,
		x(1, 0, 99, g(78, 40, 40, 60, 99, 96, 94, 0)),
		x(1, 0, 40, g(80, 50, 40, 60, 99, 60, 55, 0)).vel(3),
		x(1, 0, 30, g(90, 60, 40, 60, 99, 50, 40, 0)),
		x(2, 0, 50, g(74, 40, 40, 60, 99, 94, 90, 0)).det(8),
		x(14, 0, 50, g(99, 70, 40, 60, 99, 0, 0, 0)).vel(5),
		x(1, 0, 40, g(90, 60, 40, 60, 99, 40, 40, 0)))
	p.lfo(34, 60, 10, 0, 4, 3)
	p.Level = 0.4518 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// WoodClarinet is a clarinet: odd harmonics from the square wave, mellowed by a low modulator
func WoodClarinet() Patch {
	p := dxp("CLARINET", 2, 3,
		x(1, 0, 99, g(80, 40, 40, 60, 99, 96, 94, 0)).wav(6),
		x(1, 0, 52, g(80, 40, 40, 60, 99, 70, 60, 0)).vel(4),
		x(1, 0, 60, g(80, 40, 40, 60, 99, 90, 85, 0)).det(8),
		x(3, 0, 40, g(80, 40, 40, 60, 99, 60, 50, 0)),
		x(1, 0, 30, g(80, 40, 40, 60, 99, 60, 50, 0)),
		x(1, 0, 30, g(80, 40, 40, 60, 99, 60, 50, 0)))
	p.lfo(34, 55, 8, 0, 4, 3)
	p.Level = 0.3093 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// NasalOboe is an oboe: a narrow, reedy tone with partials around 3 and 4 (algorithm 14)
func NasalOboe() Patch {
	p := dxp("NASAL OBOE", 14, 4,
		x(1, 0, 99, g(82, 40, 40, 60, 99, 96, 94, 0)),
		x(1, 0, 74, g(82, 40, 40, 60, 99, 90, 85, 0)).vel(4),
		x(1, 0, 90, g(82, 40, 40, 60, 99, 95, 92, 0)).det(8),
		x(3, 0, 74, g(80, 40, 40, 60, 99, 85, 80, 0)).vel(4),
		x(1, 0, 70, g(80, 40, 40, 60, 99, 80, 75, 0)),
		x(2, 0, 66, g(78, 40, 40, 60, 99, 80, 75, 0)))
	p.lfo(36, 50, 10, 0, 4, 3)
	p.Level = 0.5751 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// TenorSax is a tenor saxophone: a throaty tone with a growl as the feedback modulator is hit (algorithm 3)
func TenorSax() Patch {
	p := dxp("TENOR SAX", 3, 5,
		x(1, 0, 99, g(88, 40, 40, 60, 99, 96, 94, 0)),
		x(1, 0, 76, g(88, 50, 40, 60, 99, 80, 70, 0)).vel(5),
		x(1, 0, 74, g(88, 50, 40, 60, 99, 70, 60, 0)),
		x(1, 0, 90, g(88, 40, 40, 60, 99, 95, 92, 0)).det(9),
		x(2, 0, 70, g(86, 50, 40, 60, 99, 82, 75, 0)).vel(4),
		x(1, 0, 74, g(86, 50, 40, 60, 99, 78, 70, 0)))
	p.lfo(34, 50, 14, 0, 0, 3)
	p.Level = 0.6809 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// PanPipes is a pan flute: sines with a puff of high noise like breath at the start (algorithm 32)
func PanPipes() Patch {
	sus := g(82, 99, 99, 60, 99, 99, 99, 0)
	p := dxp("PAN PIPES", 32, 7,
		x(1, 0, 96, sus),
		x(2, 0, 40, g(80, 99, 99, 60, 99, 99, 99, 0)),
		x(1, 0, 50, sus).det(9),
		x(3, 0, 28, g(78, 99, 99, 60, 99, 99, 99, 0)),
		x(14, 0, 22, g(99, 70, 40, 60, 99, 0, 0, 0)),
		x(11, 50, 40, g(99, 62, 40, 60, 99, 0, 0, 0)).vel(4))
	p.lfo(34, 60, 8, 0, 4, 2)
	p.Level = 0.3772 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// --- leads ---

// SyncLead is a hard sync like lead: the log saw wave with a modulator that gives the tearing sweep (algorithm 1)
func SyncLead() Patch {
	p := dxp("SYNC LEAD", 1, 6,
		x(1, 0, 99, g(99, 60, 50, 75, 99, 96, 90, 0)).wav(7),
		x(1, 0, 70, g(99, 40, 40, 75, 99, 60, 50, 0)).wav(7),
		x(1, 0, 70, g(99, 60, 50, 75, 99, 90, 80, 0)).wav(7).det(9),
		x(3, 0, 60, g(99, 40, 40, 75, 99, 50, 40, 0)).vel(4),
		x(2, 0, 60, g(99, 30, 40, 75, 99, 60, 60, 0)),
		x(1, 0, 70, g(99, 30, 40, 75, 99, 60, 55, 0)))
	p.lfo(36, 40, 10, 0, 0, 3)
	p.Level = 0.6215 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// PWMLead is a pulse width lead: two squares a few cents apart beat to make the width move (algorithm 32)
func PWMLead() Patch {
	sus := g(99, 99, 99, 70, 99, 99, 99, 0)
	p := dxp("PWM LEAD", 32, 0,
		x(1, 0, 90, sus).wav(6).det(5),
		x(1, 0, 90, sus).wav(6).det(9),
		x(1, 0, 60, sus).wav(3).det(7),
		x(0, 0, 50, sus).wav(1))
	p.lfo(36, 30, 6, 0, 4, 3)
	p.Level = 0.3022 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// PureSine is a sine lead, only a vibrato that comes in late
func PureSine() Patch {
	p := dxp("PURE SINE", 32, 0,
		x(1, 0, 99, g(96, 99, 99, 70, 99, 99, 99, 0)))
	p.lfo(36, 70, 20, 0, 4, 3)
	p.Level = 0.2896 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// FMLead is an FM lead with a clean attack: a bell like bite that fades into a singing line (algorithm 5)
func FMLead() Patch {
	p := dxp("FM LEAD", 5, 5,
		x(1, 0, 99, g(99, 50, 40, 70, 99, 96, 92, 0)),
		x(1, 0, 78, g(99, 40, 40, 70, 99, 55, 45, 0)).vel(6),
		x(1, 0, 88, g(99, 50, 40, 70, 99, 95, 90, 0)).det(10),
		x(2, 0, 72, g(99, 40, 40, 70, 99, 50, 40, 0)),
		x(2, 0, 64, g(99, 50, 40, 70, 99, 90, 85, 0)).det(4),
		x(7, 0, 56, g(99, 55, 40, 70, 99, 20, 10, 0)).vel(6))
	p.lfo(36, 55, 12, 0, 0, 3)
	p.Level = 0.7119 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// GlideLead is a mono lead that slides from note to note, with a vibrato of its own
func GlideLead() Patch {
	p := dxp("GLIDE LEAD", 4, 6,
		x(1, 0, 99, g(99, 60, 50, 75, 99, 96, 92, 0)),
		x(1, 0, 74, g(99, 50, 45, 75, 99, 70, 60, 0)).vel(5),
		x(1, 0, 62, g(99, 50, 45, 75, 99, 60, 50, 0)),
		x(2, 0, 90, g(99, 60, 50, 75, 99, 94, 90, 0)).det(9),
		x(1, 0, 60, g(99, 50, 45, 75, 99, 60, 50, 0)),
		x(1, 0, 62, g(99, 50, 45, 75, 99, 55, 50, 0)))
	p.Glide, p.VibratoRate, p.VibratoDepth, p.VibratoDelay = 120, 5.5, 0.3, 400
	p.Level = 0.7191 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// --- percussion ---

// TomFM is a tom: a body that falls in pitch, a click of the stick (algorithm 1)
func TomFM() Patch {
	p := dxp("TOM  FM", 1, 5,
		x(1, 0, 99, g(99, 38, 30, 70, 99, 40, 0, 0)),
		x(1, 0, 60, g(99, 55, 35, 70, 99, 10, 0, 0)).vel(5),
		x(1, 0, 60, g(99, 45, 35, 70, 99, 20, 0, 0)).det(4),
		x(2, 0, 54, g(99, 70, 40, 70, 99, 0, 0, 0)),
		x(7, 0, 50, g(99, 85, 50, 70, 99, 0, 0, 0)).vel(6),
		x(3, 0, 60, g(99, 80, 45, 70, 99, 0, 0, 0)))
	p.pitchEG(99, 62, 99, 99, 50, 50, 50, 68)
	p.Transpose = 12
	p.Level = 0.9025 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// KickFM is a kick drum: a sine that drops fast from a high start with a click on top (algorithm 3)
func KickFM() Patch {
	p := dxp("KICK  FM", 3, 3,
		x(0, 0, 99, g(99, 48, 40, 75, 99, 30, 0, 0)),
		x(1, 0, 50, g(99, 65, 40, 75, 99, 0, 0, 0)).vel(5),
		x(1, 0, 40, g(99, 70, 40, 75, 99, 0, 0, 0)),
		x(1, 0, 70, g(99, 80, 40, 75, 99, 0, 0, 0)),
		x(9, 0, 44, g(99, 92, 60, 75, 99, 0, 0, 0)),
		x(1, 0, 40, g(99, 92, 60, 75, 99, 0, 0, 0)))
	p.pitchEG(99, 55, 99, 99, 50, 50, 50, 78)
	p.Transpose = 12
	p.Level = 1.2844 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// TimpaniFM is a timpani: a harmonic series from 1 to 2.1 that fades at different speeds and a pitch that settles (algorithm 5)
func TimpaniFM() Patch {
	p := dxp("TIMPANI", 5, 2,
		x(1, 0, 99, g(99, 28, 22, 60, 99, 50, 0, 0)),
		x(1, 50, 58, g(99, 50, 30, 60, 99, 10, 0, 0)).vel(6),
		x(1, 50, 80, g(99, 34, 24, 60, 99, 30, 0, 0)),
		x(1, 0, 50, g(99, 55, 30, 60, 99, 0, 0, 0)),
		x(2, 12, 66, g(99, 36, 26, 60, 99, 20, 0, 0)),
		x(2, 0, 30, g(99, 80, 40, 60, 99, 0, 0, 0)))
	p.pitchEG(99, 50, 99, 99, 50, 50, 50, 58)
	p.Level = 0.9485 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// SnareFM is a snare: a body and noise like bursts, a high modulator fed back on itself (algorithm 4)
func SnareFM() Patch {
	p := dxp("SNARE  FM", 4, 7,
		x(1, 50, 90, g(99, 55, 45, 75, 99, 20, 0, 0)),
		x(7, 0, 76, g(99, 62, 45, 75, 99, 20, 0, 0)).vel(5),
		x(11, 36, 70, g(99, 66, 45, 75, 99, 0, 0, 0)),
		x(1, 0, 99, g(99, 48, 35, 75, 99, 0, 0, 0)),
		x(3, 20, 70, g(99, 70, 45, 75, 99, 0, 0, 0)),
		x(14, 33, 78, g(99, 62, 40, 75, 99, 10, 0, 0)))
	p.pitchEG(99, 70, 99, 99, 50, 50, 50, 56)
	p.Level = 0.9063 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// HatFM is a hi hat: inharmonic square waves at fixed frequencies, a short burst (algorithm 32)
func HatFM() Patch {
	p := dxp("HI HAT FM", 32, 0,
		x(1, 0, 90, g(99, 66, 45, 80, 99, 0, 0, 0)).wav(6).fixed(3, 5),
		x(1, 0, 90, g(99, 66, 45, 80, 99, 0, 0, 0)).wav(6).fixed(3, 28),
		x(1, 0, 86, g(99, 66, 45, 80, 99, 0, 0, 0)).wav(6).fixed(3, 56),
		x(1, 0, 86, g(99, 66, 45, 80, 99, 0, 0, 0)).wav(6).fixed(3, 82),
		x(1, 0, 80, g(99, 66, 45, 80, 99, 0, 0, 0)).wav(6).fixed(3, 95),
		x(1, 0, 80, g(99, 66, 45, 80, 99, 0, 0, 0)).wav(6).fixed(3, 99))
	p.flt(3, 80, 10, 0, 0, 0, 30, 0, 20)
	p.Level = 5.2997 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// WoodBlockFM is a wood block: a short dry knock of two inharmonic partials (algorithm 5)
func WoodBlockFM() Patch {
	p := dxp("WOOD BLOCK", 5, 0,
		x(3, 0, 99, g(99, 70, 50, 85, 99, 0, 0, 0)),
		x(2, 41, 64, g(99, 82, 50, 85, 99, 0, 0, 0)),
		x(5, 0, 80, g(99, 74, 50, 85, 99, 0, 0, 0)),
		x(1, 0, 40, g(99, 90, 50, 85, 99, 0, 0, 0)),
		x(4, 40, 40, g(99, 76, 50, 85, 99, 0, 0, 0)),
		x(1, 0, 20, g(99, 90, 50, 85, 99, 0, 0, 0)))
	p.Level = 0.9867 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// --- effects ---

// RiserSweep is a riser: a pitch that climbs over seconds and a filter that opens
func RiserSweep() Patch {
	p := dxp("RISER", 24, 6,
		x(1, 0, 90, g(30, 99, 99, 50, 99, 99, 99, 0)).wav(7),
		x(1, 0, 80, g(30, 99, 99, 50, 99, 99, 99, 0)).wav(7).det(10),
		x(2, 0, 76, g(30, 99, 99, 50, 99, 99, 99, 0)).wav(6).det(4),
		x(3, 0, 60, g(30, 99, 99, 50, 99, 99, 99, 0)),
		x(1, 0, 70, g(30, 99, 99, 50, 99, 99, 99, 0)).det(12),
		x(2, 0, 80, g(40, 99, 99, 50, 30, 99, 99, 0)))
	p.pitchEG(32, 99, 99, 99, 78, 50, 50, 40)
	p.flt(6, 20, 40, 65, 0, 62, 99, 99, 40)
	p.Level = 0.6773 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// LaserZap is a laser: a very high pitch that falls through the sound in a flash (algorithm 5)
func LaserZap() Patch {
	p := dxp("LASER ZAP", 5, 4,
		x(1, 0, 99, g(99, 60, 40, 80, 99, 50, 0, 0)),
		x(1, 0, 80, g(99, 60, 40, 80, 99, 20, 0, 0)),
		x(2, 0, 70, g(99, 62, 40, 80, 99, 20, 0, 0)).wav(6),
		x(3, 0, 70, g(99, 66, 40, 80, 99, 0, 0, 0)),
		x(1, 0, 40, g(99, 70, 40, 80, 99, 0, 0, 0)),
		x(5, 0, 60, g(99, 70, 40, 80, 99, 0, 0, 0)))
	p.pitchEG(99, 62, 99, 99, 50, 50, 50, 99)
	p.Level = 1.0651 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// WindGust is wind: the noise of a high feedback modulator through a band pass that moves at random
func WindGust() Patch {
	p := dxp("WIND GUST", 4, 7,
		x(11, 0, 80, g(55, 45, 60, 50, 99, 90, 85, 0)).fixed(3, 40),
		x(1, 0, 90, g(55, 45, 60, 50, 99, 90, 85, 0)).fixed(3, 70),
		x(1, 0, 70, g(55, 45, 60, 50, 99, 90, 85, 0)),
		x(1, 0, 90, g(55, 45, 60, 50, 99, 90, 85, 0)).fixed(2, 90),
		x(14, 0, 90, g(55, 45, 60, 50, 99, 90, 85, 0)),
		x(31, 99, 90, g(55, 45, 60, 50, 99, 90, 85, 0)))
	p.lfo(8, 0, 0, 0, 5, 0)
	p.flt(2, 46, 55, 14, 20, 40, 60, 60, 60)
	p.LFO2Rate = 0.3
	p.mod(modmatrix.LFO2, "cutoff", 0.5)
	p.mod(modmatrix.Random, "cutoff", 0.25)
	p.Level = 2.0 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// VowelAh is a vowel: a modulator at the pitch of the note drives carriers fixed at the formants of "ah" (algorithm 22)
func VowelAh() Patch {
	p := dxp("VOWEL AH", 22, 2,
		x(1, 0, 0, g(70, 99, 99, 60, 99, 99, 99, 0)),
		x(1, 0, 0, g(70, 99, 99, 60, 99, 99, 99, 0)),
		x(1, 0, 94, g(70, 99, 99, 60, 99, 99, 99, 0)).fixed(2, 85),
		x(1, 0, 86, g(70, 99, 99, 60, 99, 99, 99, 0)).fixed(3, 5),
		x(1, 0, 76, g(70, 99, 99, 60, 99, 99, 99, 0)).fixed(3, 40),
		x(1, 0, 74, g(70, 99, 99, 60, 99, 99, 99, 0)))
	p.lfo(34, 50, 10, 0, 4, 3)
	p.Level = 0.3958 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// AlienBubbles is a bubbling texture: a sample and hold LFO moves the pitch in steps over a metal bell (algorithm 15)
func AlienBubbles() Patch {
	p := dxp("ALIEN BUBB", 15, 3,
		x(1, 0, 94, g(90, 40, 40, 55, 99, 85, 70, 0)),
		x(1, 41, 70, g(90, 40, 40, 55, 99, 70, 55, 0)),
		x(3, 0, 80, g(90, 40, 40, 55, 99, 85, 70, 0)).det(5),
		x(1, 83, 70, g(90, 40, 40, 55, 99, 60, 45, 0)),
		x(5, 0, 60, g(90, 40, 40, 55, 99, 50, 40, 0)),
		x(2, 70, 60, g(90, 40, 40, 55, 99, 50, 40, 0)))
	p.lfo(62, 0, 80, 0, 5, 5)
	p.Level = 1.0647 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// --- the waves of the TX81Z ---

// HollowReed is a hollow reed: the pulse and even waves give the nasal odd partial colour (algorithm 11)
func HollowReed() Patch {
	p := dxp("HOLLOW RD", 11, 2,
		x(1, 0, 94, g(85, 40, 40, 60, 99, 94, 90, 0)).wav(3),
		x(1, 0, 60, g(85, 40, 40, 60, 99, 80, 70, 0)).wav(2),
		x(1, 0, 60, g(85, 40, 40, 60, 99, 60, 50, 0)).wav(1),
		x(2, 0, 88, g(85, 40, 40, 60, 99, 92, 85, 0)).wav(4).det(8),
		x(1, 0, 50, g(85, 40, 40, 60, 99, 60, 50, 0)).wav(5),
		x(3, 0, 46, g(85, 40, 40, 60, 99, 50, 40, 0)).wav(6))
	p.lfo(34, 45, 8, 0, 4, 3)
	p.Level = 1.1591 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// LogSawBrass is a brass of the log saw: it bites, and its modulators swell with the attack (algorithm 22)
func LogSawBrass() Patch {
	p := dxp("LOGSAW BRS", 22, 3,
		x(1, 0, 94, g(75, 40, 35, 55, 99, 95, 90, 0)).wav(7),
		x(1, 0, 64, g(70, 40, 35, 55, 99, 85, 75, 0)).vel(4),
		x(1, 0, 88, g(75, 40, 35, 55, 99, 95, 90, 0)).wav(7).det(9),
		x(1, 0, 84, g(73, 40, 35, 55, 99, 93, 88, 0)).wav(6).det(5),
		x(2, 0, 70, g(73, 40, 35, 55, 99, 92, 85, 0)).wav(7).det(11),
		x(1, 0, 70, g(66, 40, 35, 55, 99, 88, 80, 0)))
	p.lfo(32, 60, 8, 0, 4, 3)
	p.Level = 0.8212 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// HalfSineOrgan is an organ of the half sine: a wire like tone with even partials only
func HalfSineOrgan() Patch {
	gate := g(99, 99, 99, 85, 99, 99, 99, 0)
	p := dxp("HALFSINE O", 32, 0,
		x(1, 0, 90, gate).wav(1),
		x(2, 0, 84, gate).wav(1),
		x(3, 0, 70, gate).wav(1).det(8),
		x(4, 0, 64, gate).wav(2),
		x(0, 0, 90, gate).wav(1),
		x(6, 0, 50, gate).wav(2))
	p.lfo(40, 0, 0, 0, 4, 0)
	p.Level = 0.9844 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// EvenAbsPad is a pad of the double speed waves: sounds an octave above the note with a soft grain
func EvenAbsPad() Patch {
	p := dxp("EVENABS PD", 5, 2,
		x(1, 0, 86, g(50, 40, 30, 45, 99, 94, 88, 0)).wav(5),
		x(1, 0, 54, g(46, 40, 30, 45, 99, 80, 70, 0)).wav(4),
		x(1, 0, 80, g(52, 40, 30, 45, 99, 94, 88, 0)).wav(4).det(10),
		x(2, 0, 50, g(46, 40, 30, 45, 99, 75, 65, 0)).wav(2),
		x(0, 0, 90, g(52, 40, 30, 45, 99, 94, 88, 0)).det(5),
		x(1, 0, 46, g(48, 40, 30, 45, 99, 70, 60, 0)))
	p.lfo(24, 30, 8, 0, 4, 2)
	p.Level = 1.4902 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// PulseClav is a plucked pulse: the pulse sine wave on a short envelope, a funk riff voice
func PulseClav() Patch {
	p := dxp("PULSE CLAV", 4, 3,
		x(1, 0, 99, g(99, 60, 45, 78, 99, 50, 20, 0)).wav(3),
		x(1, 0, 70, g(99, 70, 45, 78, 99, 20, 0, 0)).wav(6).vel(6),
		x(3, 0, 50, g(99, 75, 45, 78, 99, 0, 0, 0)),
		x(1, 0, 86, g(99, 55, 45, 78, 99, 40, 15, 0)).wav(3).det(9),
		x(2, 0, 64, g(99, 65, 45, 78, 99, 20, 0, 0)).wav(2),
		x(2, 0, 40, g(99, 75, 45, 78, 99, 0, 0, 0)))
	p.Level = 0.9496 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// --- the matrix and the filter ---

// WheelKeys is an electric piano that the mod wheel brightens: it opens the modulators and adds a vibrato
func WheelKeys() Patch {
	p := dxp("WHEEL KEYS", 5, 4,
		x(1, 0, 99, g(99, 34, 22, 58, 99, 82, 0, 0)),
		x(1, 0, 66, g(99, 55, 30, 58, 99, 25, 0, 0)).vel(6),
		x(1, 0, 90, g(99, 32, 22, 58, 99, 78, 0, 0)).det(9),
		x(2, 0, 62, g(99, 58, 30, 58, 99, 20, 0, 0)).vel(5),
		x(1, 0, 74, g(99, 36, 22, 58, 99, 70, 0, 0)).det(4),
		x(5, 0, 56, g(99, 70, 40, 58, 99, 0, 0, 0)))
	p.mod(modmatrix.Wheel, "op2Gain", 0.35)
	p.mod(modmatrix.Wheel, "op4Gain", 0.3)
	p.mod(modmatrix.Wheel, "op6Gain", 0.2)
	p.Level = 0.8732 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// VelocityFM is a piano where velocity opens the modulators: soft notes are plain, hard notes ring with bells
func VelocityFM() Patch {
	p := dxp("VELOCITYFM", 5, 3,
		x(1, 0, 99, g(99, 30, 20, 55, 99, 80, 0, 0)),
		x(1, 0, 40, g(99, 50, 30, 55, 99, 25, 0, 0)),
		x(1, 0, 88, g(99, 30, 20, 55, 99, 75, 0, 0)).det(8),
		x(3, 0, 40, g(99, 62, 35, 55, 99, 10, 0, 0)),
		x(2, 0, 70, g(99, 34, 20, 55, 99, 60, 0, 0)),
		x(9, 0, 30, g(99, 75, 40, 55, 99, 0, 0, 0)))
	p.mod(modmatrix.Velocity, "op2Gain", 0.4)
	p.mod(modmatrix.Velocity, "op4Gain", 0.45)
	p.mod(modmatrix.Velocity, "op6Gain", 0.4)
	p.Level = 0.9430 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// FilterPluck is a synthetic pluck: a saw through a resonant low pass whose envelope falls, brighter on hard notes
func FilterPluck() Patch {
	p := dxp("FILT PLUCK", 32, 0,
		x(1, 0, 90, g(99, 60, 40, 70, 99, 70, 60, 0)).wav(7).det(5),
		x(1, 0, 90, g(99, 60, 40, 70, 99, 70, 60, 0)).wav(7).det(9),
		x(0, 0, 80, g(99, 60, 40, 70, 99, 70, 60, 0)).wav(6))
	p.flt(5, 24, 45, 55, 50, 0, 38, 8, 40)
	p.mod(modmatrix.Velocity, "cutoff", 0.3)
	p.Level = 1.4127 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}

// WheelPad is a pad whose filter the mod wheel opens, and a slow LFO that breathes on it
func WheelPad() Patch {
	p := dxp("WHEEL PAD", 24, 2,
		x(1, 0, 90, g(50, 40, 30, 45, 99, 95, 92, 0)).wav(7).det(5),
		x(1, 0, 86, g(50, 40, 30, 45, 99, 95, 92, 0)).wav(7).det(10),
		x(1, 0, 80, g(50, 40, 30, 45, 99, 95, 92, 0)).wav(6).det(3),
		x(2, 0, 70, g(50, 40, 30, 45, 99, 95, 92, 0)).wav(7).det(12),
		x(1, 0, 74, g(50, 40, 30, 45, 99, 95, 92, 0)).det(7),
		x(1, 0, 70, g(50, 40, 30, 45, 99, 95, 92, 0)))
	p.flt(6, 40, 30, 0, 50, 40, 40, 99, 45)
	p.LFO2Rate = 0.4
	p.mod(modmatrix.Wheel, "cutoff", 0.9)
	p.mod(modmatrix.LFO2, "cutoff", 0.15)
	p.Level = 1.2017 // calibrated: -12 LUFS (or a -1 dB peak) at amplitude 1
	return p
}
