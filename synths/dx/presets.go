package dx

// A few original patches, to play without loading a SysEx bank and as examples of building patches
// in code. Every value uses the classic 0-99 style ranges

// Presets maps preset names to their constructors
var Presets = map[string]func() Patch{
	"glassKeys": GlassKeys,
	"woodBass":  WoodBass,
	"softBrass": SoftBrass,
	"bellPad":   BellPad,
}

// opr builds an operator with a ratio frequency
func opr(coarse int, level int, rates [4]int, levels [4]int) Operator {
	return Operator{
		EGRates:     rates,
		EGLevels:    levels,
		BreakPoint:  39,
		Detune:      7,
		Coarse:      coarse,
		OutputLevel: level,
	}
}

// GlassKeys is a glassy electric piano: three operator pairs, with a high ratio tine on the
// second pair that fades quickly and velocity that brightens the tone
func GlassKeys() Patch {
	p := InitVoice()
	p.Name = "GLASS KEYS"
	p.Algorithm = 4 // algorithm 5: 6>5, 4>3, 2>1
	p.Feedback = 3

	p.Ops[0] = opr(1, 99, [4]int{96, 30, 25, 55}, [4]int{99, 75, 0, 0})
	p.Ops[1] = opr(1, 78, [4]int{96, 45, 25, 55}, [4]int{99, 65, 0, 0})
	p.Ops[1].VelocitySens = 5
	p.Ops[2] = opr(1, 88, [4]int{96, 28, 25, 55}, [4]int{99, 72, 0, 0})
	p.Ops[2].Detune = 9
	p.Ops[3] = opr(14, 62, [4]int{99, 70, 40, 60}, [4]int{99, 0, 0, 0})
	p.Ops[3].VelocitySens = 6
	p.Ops[4] = opr(4, 70, [4]int{96, 40, 25, 55}, [4]int{99, 55, 0, 0})
	p.Ops[5] = opr(3, 60, [4]int{96, 50, 30, 55}, [4]int{99, 40, 0, 0})
	p.Ops[5].VelocitySens = 4
	return p
}

// WoodBass is a round, plucked bass with a sub octave carrier
func WoodBass() Patch {
	p := InitVoice()
	p.Name = "WOOD BASS"
	p.Algorithm = 0 // algorithm 1: 6>5>4>3, 2>1
	p.Feedback = 4

	p.Ops[0] = opr(1, 99, [4]int{99, 60, 35, 70}, [4]int{99, 85, 65, 0})
	p.Ops[1] = opr(1, 80, [4]int{99, 68, 45, 70}, [4]int{99, 65, 45, 0})
	p.Ops[1].VelocitySens = 4
	p.Ops[2] = opr(0, 85, [4]int{99, 55, 35, 70}, [4]int{99, 85, 65, 0})
	p.Ops[3] = opr(1, 70, [4]int{99, 75, 40, 70}, [4]int{99, 50, 0, 0})
	p.Ops[4] = opr(3, 55, [4]int{99, 85, 40, 70}, [4]int{99, 0, 0, 0})
	p.Ops[5] = opr(1, 50, [4]int{99, 80, 40, 70}, [4]int{99, 0, 0, 0})
	return p
}

// SoftBrass is a mellow brass ensemble: one modulator swells into four carriers, with delayed vibrato
func SoftBrass() Patch {
	p := InitVoice()
	p.Name = "SOFT BRASS"
	p.Algorithm = 21 // algorithm 22: 2>1, 6>3, 6>4, 6>5
	p.Feedback = 5

	p.Ops[0] = opr(1, 99, [4]int{70, 40, 30, 55}, [4]int{99, 95, 90, 0})
	p.Ops[1] = opr(1, 76, [4]int{55, 40, 30, 55}, [4]int{99, 88, 80, 0})
	p.Ops[1].VelocitySens = 3
	p.Ops[2] = opr(1, 90, [4]int{68, 40, 30, 55}, [4]int{99, 95, 90, 0})
	p.Ops[2].Detune = 5
	p.Ops[3] = opr(1, 88, [4]int{66, 40, 30, 55}, [4]int{99, 95, 90, 0})
	p.Ops[3].Detune = 9
	p.Ops[4] = opr(2, 70, [4]int{66, 40, 30, 55}, [4]int{99, 90, 85, 0})
	p.Ops[5] = opr(1, 72, [4]int{58, 40, 30, 55}, [4]int{99, 85, 80, 0})
	p.Ops[5].VelocitySens = 3

	p.LFOSpeed = 33
	p.LFODelay = 45
	p.LFOPitchDepth = 8
	p.PitchModSens = 3
	p.LFOWaveform = 0
	return p
}

// BellPad is a slow pad with a bell like shimmer on top
func BellPad() Patch {
	p := InitVoice()
	p.Name = "BELL PAD"
	p.Algorithm = 4 // algorithm 5: 6>5, 4>3, 2>1

	p.Ops[0] = opr(1, 95, [4]int{45, 30, 20, 40}, [4]int{99, 90, 85, 0})
	p.Ops[1] = opr(1, 68, [4]int{40, 30, 20, 40}, [4]int{99, 80, 70, 0})
	p.Ops[2] = opr(1, 90, [4]int{43, 30, 20, 40}, [4]int{99, 90, 85, 0})
	p.Ops[2].Detune = 10
	p.Ops[3] = opr(1, 60, [4]int{40, 30, 20, 40}, [4]int{99, 75, 65, 0})
	p.Ops[4] = opr(3, 72, [4]int{75, 35, 25, 40}, [4]int{99, 60, 0, 0})
	p.Ops[5] = opr(7, 60, [4]int{75, 40, 25, 40}, [4]int{99, 50, 0, 0})
	p.Ops[5].Fine = 50 // 7 * 1.5, an inharmonic partial for the bell

	p.LFOSpeed = 30
	p.LFOAmpDepth = 10
	p.LFOWaveform = 4
	p.Ops[0].AmpModSens = 1
	p.Ops[2].AmpModSens = 1
	return p
}
