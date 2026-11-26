package grains

import "math"

// Windows are the shapes a grain can have, from the one that is smooth everywhere to the one that has a flat middle
const (
	WindowHann     = iota // sin squared: smooth, the grains blend into a cloud
	WindowGauss           // a bell: smooth, with more of the middle than the hann
	WindowTukey           // flat in the middle, with a slope at both ends: grains that are heard as notes
	WindowTriangle        // straight up, straight down: a sharper edge
	numWindows
)

// WindowNames are the names of the windows, in the order of the setting
var WindowNames = []string{"hann", "gauss", "tukey", "triangle"}

// windowTables hold the windows apart from the hann, read with linear interpolation (like the smooth table)
var windowTables = func() [][]float64 {
	const n = 1024
	tabs := make([][]float64, numWindows)
	for k := WindowGauss; k < numWindows; k++ {
		t := make([]float64, n+2)
		for i := range n + 1 {
			t[i] = windowShape(float64(i)/n, k)
		}
		tabs[k] = t
	}
	return tabs
}()

// windowShape is a window at t (0 to 1), from zero to zero
func windowShape(t float64, kind int) float64 {
	switch kind {
	case WindowGauss:
		g := func(x float64) float64 { d := (x - 0.5) / 0.17; return math.Exp(-0.5 * d * d) }
		g0 := g(0)
		return max(0, (g(t)-g0)/(1-g0))
	case WindowTukey:
		const edge = 0.25 // the slope at either end
		switch {
		case t < edge:
			s := math.Sin(math.Pi / 2 * t / edge)
			return s * s
		case t > 1-edge:
			s := math.Sin(math.Pi / 2 * (1 - t) / edge)
			return s * s
		}
		return 1
	case WindowTriangle:
		return 1 - math.Abs(2*t-1)
	}
	s := math.Sin(math.Pi * t)
	return s * s
}

// baseWindow is the window of a kind at t (0 to 1), by table
func baseWindow(t float64, kind int) float64 {
	if kind <= WindowHann || kind >= numWindows {
		return smoothWindow(t)
	}
	tab := windowTables[kind]
	pos := max(0, min(t, 1)) * float64(len(tab)-2)
	i := int(pos)
	a := tab[i]
	return a + (tab[i+1]-a)*(pos-float64(i))
}

// smoothTable holds the sin^2 window, read with linear interpolation (an error below 1e-6): every
// sample of every grain reads the window, a sine there costs more than all the rest of a grain
var smoothTable = func() []float64 {
	const n = 1024
	t := make([]float64, n+2)
	for i := range n + 1 {
		s := math.Sin(math.Pi * float64(i) / n)
		t[i] = s * s
	}
	return t
}()

func smoothWindow(t float64) float64 {
	pos := max(0, min(t, 1)) * float64(len(smoothTable)-2)
	i := int(pos)
	a := smoothTable[i]
	return a + (smoothTable[i+1]-a)*(pos-float64(i))
}

// window returns the grain envelope of the hann at t (0-1) morphing to a percussive shape: see windowOf
func window(t float64, shape float64) float64 { return windowOf(t, shape, WindowHann) }

// windowOf returns the grain envelope at t (0-1): the window of a kind, morphing to a percussive
// shape with a fast attack and an exponential decay as shape goes from 0 to 1
func windowOf(t float64, shape float64, kind int) float64 {
	smooth := baseWindow(t, kind)

	if shape <= 0.0 {
		return smooth
	}

	const attack = 0.03
	var perc float64
	if t < attack {
		perc = t / attack
	} else {
		// starts at 1 where the attack ends and reaches 0 at the end of the grain
		perc = math.Exp(-6.0*(t-attack)/(1.0-attack)) * (1.0 - t) / (1.0 - attack)
	}

	return smooth + (perc-smooth)*shape
}

// windowCompensation scales a grain window to the energy of the smooth hann, so percussive grains
// (which decay fast and carry less energy) and the other windows aren't louder or quieter
func windowCompensation(shape float64, kind int) float64 {
	if shape <= 0 && kind == WindowHann {
		return 1
	}
	const steps = 256
	smooth, sh := 0.0, 0.0
	for i := range steps {
		t := (float64(i) + 0.5) / steps
		ws, w := window(t, 0), windowOf(t, shape, kind)
		smooth += ws * ws
		sh += w * w
	}
	return math.Sqrt(smooth / sh)
}
