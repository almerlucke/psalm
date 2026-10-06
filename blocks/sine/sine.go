// Package sine is a fast sine for audio loops: a table of one cycle read with linear interpolation,
// accurate to about 3e-7 (-130 dB), several times faster than math.Sin. Use it where a sine is
// computed every sample (oscillators, operators, windows), keep math.Sin elsewhere
package sine

import "math"

// size of the table, a power of 2
const size = 4096

var table = func() []float64 {
	t := make([]float64, size+1)
	for i := range t {
		t[i] = math.Sin(2 * math.Pi * float64(i) / size)
	}
	return t
}()

// Cycle returns the sine of a phase in cycles (1 is a full turn): sin(2π phase)
func Cycle(phase float64) float64 {
	phase -= math.Floor(phase)
	pos := phase * size
	i := int(pos)
	a := table[i]
	return a + (table[i+1]-a)*(pos-float64(i))
}

// CosCycle returns the cosine of a phase in cycles
func CosCycle(phase float64) float64 {
	return Cycle(phase + 0.25)
}

// Sin returns the sine of x in radians
func Sin(x float64) float64 {
	return Cycle(x * (0.5 / math.Pi))
}

// Cos returns the cosine of x in radians
func Cos(x float64) float64 {
	return Cycle(x*(0.5/math.Pi) + 0.25)
}
