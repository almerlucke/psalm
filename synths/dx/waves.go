package dx

import (
	"math"

	"github.com/almerlucke/psalm/blocks/sine"
)

// WaveNames are the waves of an operator, by number: the sine of the DX7 and the seven others of the
// TX81Z and the SY77 (the sine cut and folded, in pulses, at double speed, a square, a saw)
var WaveNames = []string{"sine", "half sine", "abs sine", "pulse sine", "even sine", "even abs", "square", "log saw"}

// wave returns the wave w (1-7, 0 is the sine) at a phase in cycles (any, it wraps)
func wave(w int, phase float64) float64 {
	x := phase - math.Floor(phase)
	switch w {
	case 1: // the positive half of the sine
		if x < 0.5 {
			return sine.Cycle(x)
		}
		return 0
	case 2: // both halves positive
		return math.Abs(sine.Cycle(x))
	case 3: // the first quarter of each half
		if x < 0.25 || (x >= 0.5 && x < 0.75) {
			return math.Abs(sine.Cycle(x))
		}
		return 0
	case 4: // a full sine in the first half (double speed), then silence
		if x < 0.5 {
			return sine.Cycle(2 * x)
		}
		return 0
	case 5: // the rectified sine at double speed in the first half
		if x < 0.5 {
			return math.Abs(sine.Cycle(2 * x))
		}
		return 0
	case 6: // a square
		if x < 0.5 {
			return 1
		}
		return -1
	case 7: // a falling saw bent like a logarithm: it drops fast and settles slowly
		y := 2*x - 1
		return -math.Copysign(1-math.Pow(1-math.Abs(y), 2.2), y)
	}
	return sine.Cycle(phase)
}
