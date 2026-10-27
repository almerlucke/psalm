package ladder

import "math"

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
// feedback of 0-4.2, the filter self-oscillates at 1. Drive saturates the input of the ladder
type Ladder struct {
	sr       float64
	fc       float64
	res      float64
	drive    float64
	g        float64 // G = g / (1 + g)
	k        float64 // feedback
	s        [4]float64
	compGain float64
}

func New(fc float64, res float64, sr float64) *Ladder {
	l := &Ladder{sr: sr, fc: fc, res: res, drive: 1.0}
	l.update()
	return l
}

func (l *Ladder) Set(fc float64, res float64) {
	l.fc = fc
	l.res = res
	l.update()
}

// SetDrive sets the input saturation, 1 is mild, higher values distort more
func (l *Ladder) SetDrive(drive float64) {
	l.drive = math.Max(0.1, drive)
	l.update()
}

func (l *Ladder) update() {
	fc := math.Max(1.0, math.Min(l.fc, l.sr*0.45))
	g := math.Tan(math.Pi * fc / l.sr)

	l.g = g / (1.0 + g)
	// the feedback of 4 is where the ladder would oscillate, but the saturation takes a little of the loop's gain away: the last part of the range goes over 4, so that
	// a resonance of 1 sings on by itself (a sine at the cutoff) and 0.95 does not yet
	res := math.Max(0.0, math.Min(l.res, 1.0))
	l.k = 4.0*res + 0.2*math.Pow(res, 6)

	// the ladder loses passband level as the feedback rises, partly compensate it
	l.compGain = 1.0 + 0.5*l.k
}

func (l *Ladder) Clear() {
	l.s = [4]float64{}
}

// Process filters a single sample and returns the output for the given mode
func (l *Ladder) Process(x float64, mode Mode) float64 {
	G := l.g
	oneMinusG := 1.0 - G

	// every stage is y = G*x + (1-G)*s, solve the feedback loop for the last stage output
	sigma := G*G*G*oneMinusG*l.s[0] + G*G*oneMinusG*l.s[1] + G*oneMinusG*l.s[2] + oneMinusG*l.s[3]
	G4 := G * G * G * G

	x *= l.compGain
	y4 := (G4*x + sigma) / (1.0 + l.k*G4)

	// saturating the ladder input keeps high resonance musical and bounded, dividing by the drive
	// keeps unity gain for small signals
	u := math.Tanh((x-l.k*y4)*l.drive) / l.drive

	var y [5]float64
	y[0] = u
	for i := range 4 {
		v := (y[i] - l.s[i]) * G
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
