// Package tanh is a fast tanh for audio loops: a rational function (the continued fraction of
// tanh cut after the x^7 term), within 1e-4 of math.Tanh everywhere (-80 dB of the full level, and
// far less for a signal that is not driven hard) and about four times faster. It is odd, rises all
// the way and has a gain of exactly 1 around zero, as a saturator must. Use it where a tanh is
// computed every sample (saturation, feedback limits), keep math.Tanh elsewhere
package tanh

// limit is where the rational function reaches 1; beyond it the output is 1
const limit = 4.97

// Tanh returns the hyperbolic tangent of x
func Tanh(x float64) float64 {
	if x >= limit {
		return 1
	}
	if x <= -limit {
		return -1
	}
	x2 := x * x
	return x * (135135 + x2*(17325+x2*(378+x2))) / (135135 + x2*(62370+x2*(3150+28*x2)))
}
