// Package tune corrects the pitch of a voice: it follows the pitch of the voice (Detect), and moves the voiced parts of it to other pitches without changing how long
// it lasts or how it sounds (Shift: pitch synchronous overlap-add, PSOLA). The consonants and the breaths, which have no pitch, are left as they were.
package tune

import (
	"math"
	"slices"
)

// Track is the pitch of a voice: a value every Hop samples, in Hz, 0 where there is none (a consonant, a breath, a pause)
type Track struct {
	Hop        int
	SampleRate float64
	F0         []float64
}

// At returns the pitch at a sample, interpolated between the frames (0 where one of the two has none)
func (t Track) At(sample float64) float64 {
	p := sample/float64(t.Hop) - 0.5
	i := int(math.Floor(p))
	f := p - float64(i)
	get := func(i int) float64 {
		if i < 0 || i >= len(t.F0) {
			return 0
		}
		return t.F0[i]
	}
	a, b := get(i), get(i+1)
	switch {
	case a == 0 && b == 0:
		return 0
	case a == 0:
		return b
	case b == 0:
		return a
	}
	return a * math.Pow(b/a, f)
}

// DetectOptions are the limits of a search for the pitch of a voice; the zero value is for a speaking voice
type DetectOptions struct {
	Min, Max  float64 // the lowest and highest pitch to look for, in Hz (60 and 450)
	Threshold float64 // how sure the period has to be, 0 to 1: lower is stricter (0.25)
	Quiet     float64 // frames this much quieter than the loud parts of the voice have no pitch (0.015 of the loud part)
}

func (o DetectOptions) withDefaults() DetectOptions {
	if o.Min == 0 {
		o.Min = 60
	}
	if o.Max == 0 {
		o.Max = 450
	}
	if o.Threshold == 0 {
		o.Threshold = 0.25
	}
	if o.Quiet == 0 {
		o.Quiet = 0.015
	}
	return o
}

const (
	decimate = 4   // the search is made on the voice at a quarter of the sample rate: the pitch of a voice is low
	hopSize  = 256 // samples between two frames of the track, at the full rate
)

// Detect follows the pitch of a voice (mono): the YIN method (a difference function of the signal against itself, normalized, the first dip under a threshold), made on the
// voice at a quarter of the sample rate, the dip placed between two samples, and the track made smooth where it jumps an octave for one frame
func Detect(x []float64, sampleRate float64, opts DetectOptions) Track {
	opts = opts.withDefaults()
	sr := sampleRate / decimate
	y := make([]float64, len(x)/decimate)
	for i := range y {
		var s float64
		for j := range decimate {
			s += x[i*decimate+j]
		}
		y[i] = s / decimate
	}
	tauMin := int(sr / opts.Max)
	tauMax := int(sr/opts.Min) + 1
	w := tauMax + tauMax/2 // the window the difference is summed over
	hop := hopSize / decimate
	frames := len(x) / hopSize
	t := Track{Hop: hopSize, SampleRate: sampleRate, F0: make([]float64, frames)}
	rms := make([]float64, frames)
	d := make([]float64, tauMax+1)
	cm := make([]float64, tauMax+1)
	for f := range frames {
		start := f*hop + hop/2 - w/2 // the window is centred on the frame
		if start < 0 || start+w+tauMax >= len(y) {
			continue
		}
		var e float64
		for j := range w {
			e += y[start+j] * y[start+j]
		}
		rms[f] = math.Sqrt(e / float64(w))
		for tau := 1; tau <= tauMax; tau++ {
			var s float64
			for j := range w {
				v := y[start+j] - y[start+j+tau]
				s += v * v
			}
			d[tau] = s
		}
		cm[0] = 1
		var run float64
		for tau := 1; tau <= tauMax; tau++ {
			run += d[tau]
			if run == 0 {
				cm[tau] = 1
			} else {
				cm[tau] = d[tau] * float64(tau) / run
			}
		}
		best := 0
		for tau := tauMin; tau < tauMax; tau++ {
			if cm[tau] < opts.Threshold {
				for tau+1 < tauMax && cm[tau+1] < cm[tau] {
					tau++
				}
				best = tau
				break
			}
		}
		if best == 0 {
			continue
		}
		// the dip between two samples: a parabola through it and its neighbours
		p := float64(best)
		if best > 1 && best < tauMax {
			a, b, c := cm[best-1], cm[best], cm[best+1]
			if den := a - 2*b + c; den > 1e-12 {
				p += 0.5 * (a - c) / den
			}
		}
		t.F0[f] = sr / p
	}
	// the frames too quiet to have a pitch (the room, the breath)
	loud := slices.Clone(rms)
	slices.Sort(loud)
	ref := loud[len(loud)*9/10]
	for f := range t.F0 {
		if rms[f] < ref*opts.Quiet {
			t.F0[f] = 0
		}
	}
	// one frame that is an octave (or more) from both its neighbours is a mistake of the search: it takes their pitch
	for f := 1; f+1 < len(t.F0); f++ {
		a, b, c := t.F0[f-1], t.F0[f], t.F0[f+1]
		if a > 0 && b > 0 && c > 0 && math.Abs(math.Log2(a/c)) < 0.15 && math.Abs(math.Log2(b/a)) > 0.4 {
			t.F0[f] = math.Sqrt(a * c)
		}
	}
	// a gap of up to about 80 ms inside a word (a weak frame, the closing of a consonant, between two voiced stretches at about the same pitch) is bridged: the word is one run, and it is
	// moved as one
	for f := 1; f < len(t.F0); f++ {
		if t.F0[f] != 0 || t.F0[f-1] == 0 {
			continue
		}
		g := f
		for g < len(t.F0) && t.F0[g] == 0 {
			g++
		}
		if g < len(t.F0) && g-f <= 14 && math.Abs(math.Log2(t.F0[g]/t.F0[f-1])) < 0.35 {
			for i := f; i < g; i++ {
				t.F0[i] = t.F0[f-1] * math.Pow(t.F0[g]/t.F0[f-1], float64(i-f+1)/float64(g-f+1))
			}
		}
		f = g
	}
	// a stretch of one or two frames is not a voice
	for f := 0; f < len(t.F0); {
		if t.F0[f] == 0 {
			f++
			continue
		}
		g := f
		for g < len(t.F0) && t.F0[g] > 0 {
			g++
		}
		if g-f < 3 {
			for i := f; i < g; i++ {
				t.F0[i] = 0
			}
		}
		f = g
	}
	return t
}

// Midi is the pitch of a frequency as a MIDI note (69 is A at 440 Hz), fractions between the notes
func Midi(hz float64) float64 { return 69 + 12*math.Log2(hz/440) }

// Hz is the frequency of a MIDI note
func Hz(midi float64) float64 { return 440 * math.Pow(2, (midi-69)/12) }
