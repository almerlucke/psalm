package ladder

import (
	"math"

	"github.com/almerlucke/psalm/blocks/oversample"
	"github.com/almerlucke/psalm/blocks/tanh"
)

// Mode selects the output of the ladder, the band and high pass modes are mixed from the stage
// outputs (as in multimode ladder synths)
type Mode int

const (
	LowPass24 Mode = iota
	LowPass12
	BandPass24
	BandPass12
	HighPass24
	HighPass12
)

// Ladder is a Moog style 4 pole (24 dB/oct) transistor ladder filter, built from four zero-delay
// feedback (TPT) one pole stages with a resolved global feedback loop. Resonance 0-1 maps to a
// feedback of 0-4.2, the filter self-oscillates at 1. Drive saturates the input of the ladder, and every stage after the first clips softly (stageClip)
type Ladder struct {
	sr       float64
	fc       float64
	res      float64
	drive    float64
	g        float64 // G = g / (1 + g)
	k        float64 // feedback
	s        [4]float64
	compGain float64
	stageAt  float64 // the level the clip of the stages carries up to at a drive of 1 (StageLevel)
	stage    float64 // and now
	factor   int     // the ladder runs this many times a sample (SetOversample), 1 for once
	over     *oversample.Oversampler
	mode     Mode // the mode of the sample in the oversampled run
	run      func(float64) float64
}

func New(fc float64, res float64, sr float64) *Ladder {
	l := &Ladder{sr: sr, fc: fc, res: res, drive: 1.0, stageAt: stageLevel, factor: 1}
	l.update()
	return l
}

// SetOversample runs the ladder two or four times a sample (factor 2 or 4; 1 or less for once), with the input brought up and the output down through a lowpass
// (blocks/oversample): the saturation of the drive and of the stages makes harmonics, and the ones over the Nyquist frequency fold back without it. The
// lowpass delays the sound by about a millisecond, so this is for a ladder that is driven hard
func (l *Ladder) SetOversample(factor int) {
	factor = max(1, factor)
	if factor == l.factor {
		return
	}
	l.factor = factor
	l.over = nil
	if factor >= 2 {
		l.over = oversample.New(factor, oversample.TapsPerPhase)
		l.run = func(x float64) float64 { return l.process(x, l.mode) }
	}
	l.Clear()
	l.update()
}

func (l *Ladder) Set(fc float64, res float64) {
	l.fc = fc
	l.res = res
	l.update()
}

// SetStageLevel sets the level the clip of the stages carries up to at a drive of 1 (4 by default, where only a hot signal feels it; about 1 clips a signal of normal
// level, which is the sound of an overdriven ladder)
func (l *Ladder) SetStageLevel(level float64) {
	l.stageAt = max(0.1, level)
	l.update()
}

// SetDrive sets the input saturation, 1 is mild, higher values distort more
func (l *Ladder) SetDrive(drive float64) {
	l.drive = max(0.1, drive)
	l.update()
}

func (l *Ladder) update() {
	sr := l.sr * float64(l.factor)
	fc := max(1.0, min(l.fc, l.sr*0.45))
	g := math.Tan(math.Pi * fc / sr)

	l.g = g / (1.0 + g)
	// the feedback of 4 is where the ladder would oscillate, but the saturation takes a little of the loop's gain away: the last part of the range goes over 4, so that
	// a resonance of 1 sings on by itself (a sine at the cutoff) and 0.95 does not yet
	res := max(0.0, min(l.res, 1.0))
	l.k = 4.0*res + 0.2*math.Pow(res, 6)

	// the ladder loses passband level as the feedback rises, partly compensate it
	l.compGain = 1.0 + 0.5*l.k
	l.stage = l.stageAt / math.Sqrt(l.drive)
}

func (l *Ladder) Clear() {
	l.s = [4]float64{}
	if l.over != nil {
		l.over.Reset()
	}
}

// stageLevel is the level the clip of the stages carries up to at a drive of 1: a signal that is hot (a loud one into a high resonance, a drive) reaches it, a
// quiet one does not feel it. More drive brings it down
const stageLevel = 4.0

// stageClip is a soft clip with a gain of 1 around zero (the shape of tanh, a rational approximation of it) that carries up to the level: a signal of a tenth of it keeps
// its level within a percent, and one the size of the level loses about a dB
func stageClip(x float64, level float64) float64 {
	x /= level

	if x > 3 {
		return level
	} else if x < -3 {
		return -level
	}

	return level * x * (27 + x*x) / (27 + 9*x*x)
}

// Process filters a single sample and returns the output for the given mode
func (l *Ladder) Process(x float64, mode Mode) float64 {
	if l.over != nil {
		l.mode = mode
		return l.over.Process(x, l.run)
	}

	return l.process(x, mode)
}

func (l *Ladder) process(x float64, mode Mode) float64 {
	G := l.g
	oneMinusG := 1.0 - G

	// every stage is y = G*x + (1-G)*s, solve the feedback loop for the last stage output
	sigma := G*G*G*oneMinusG*l.s[0] + G*G*oneMinusG*l.s[1] + G*oneMinusG*l.s[2] + oneMinusG*l.s[3]
	G4 := G * G * G * G

	x *= l.compGain
	y4 := (G4*x + sigma) / (1.0 + l.k*G4)

	// saturating the ladder input keeps high resonance musical and bounded, dividing by the drive
	// keeps unity gain for small signals
	u := tanh.Tanh((x-l.k*y4)*l.drive) / l.drive

	// every stage after the first gets what the one before it gives through a soft clip, as the transistor pairs of a real ladder do: a loud signal and a high
	// resonance compress and grow harmonics where a linear ladder would only get louder
	var y [5]float64
	y[0] = u
	for i := range 4 {
		in := y[i]
		if i > 0 {
			in = stageClip(in, l.stage)
		}

		v := (in - l.s[i]) * G
		y[i+1] = v + l.s[i]
		l.s[i] = y[i+1] + v
	}

	switch mode {
	case LowPass12:
		return y[2]
	case BandPass24:
		return 4.0*y[2] - 8.0*y[3] + 4.0*y[4]
	case BandPass12:
		return 2.0*y[1] - 2.0*y[2]
	case HighPass24:
		return y[0] - 4.0*y[1] + 6.0*y[2] - 4.0*y[3] + y[4]
	case HighPass12:
		return y[0] - 2.0*y[1] + y[2]
	default:
		return y[4]
	}
}
