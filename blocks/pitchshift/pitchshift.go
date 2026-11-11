package pitchshift

import (
	"math"

	"github.com/almerlucke/psalm/blocks/sine"
)

// numHeads is the number of overlapping read heads. More evenly spaced heads don't help: their
// phases can then sum to zero for some frequencies (four heads at 0, 90, 180 and 270 degrees)
const numHeads = 2

// PitchShift is a delay line pitch shifter with two crossfading read heads (like a rotating tape
// head). Each head sweeps through a short window at the shifted speed and the heads crossfade with
// sin^2 windows that always sum to a constant, so the output stays smooth.
//
// Known limitation of this classic design: for input frequencies where the two heads (half a window
// apart) are out of phase, they partly cancel during the crossfade, and the output warbles with
// sidebands at the head rate (|ratio - 1| / window, about 20 Hz for an octave up with a 50 ms
// window) around the shifted pitch. It is fine for dense material like reverb tails (shimmer),
// for clean shifting of exposed sustained notes a phase vocoder would be better
type PitchShift struct {
	buf    []float64
	pos    int
	phase  float64
	inc    float64
	window float64 // window length in samples
	ratio  float64
}

// New creates a pitch shifter, window is the grain size in milliseconds (around 30-100, larger is
// smoother for low notes, smaller has less echo)
func New(semitones float64, windowMs float64, sr float64) *PitchShift {
	window := max(4.0, windowMs*0.001*sr)
	p := &PitchShift{
		buf:    make([]float64, int(window)+8),
		window: window,
	}
	p.SetSemitones(semitones)
	return p
}

func (p *PitchShift) SetSemitones(semitones float64) {
	p.SetRatio(math.Pow(2.0, semitones/12.0))
}

// SetRatio sets the pitch ratio directly, 2 is an octave up, 0.5 an octave down
func (p *PitchShift) SetRatio(ratio float64) {
	p.ratio = ratio
	// the delay changes by 1 - ratio samples per sample, so the heads read at the shifted speed
	p.inc = (1.0 - ratio) / p.window
}

// Latency returns the average delay of the shifter in samples
func (p *PitchShift) Latency() int {
	return int(p.window / 2.0)
}

func (p *PitchShift) Clear() {
	clear(p.buf)
	p.phase = 0.0
}

// read returns the buffer delayed by a fractional number of samples (linear interpolation)
func (p *PitchShift) read(delay float64) float64 {
	n := len(p.buf)
	rp := float64(p.pos) - delay
	for rp < 0 {
		rp += float64(n)
	}
	i := int(rp)
	frac := rp - float64(i)
	a := p.buf[i%n]
	b := p.buf[(i+1)%n]
	return a + (b-a)*frac
}

func (p *PitchShift) Process(x float64) float64 {
	p.buf[p.pos] = x
	p.pos = (p.pos + 1) % len(p.buf)

	out := 0.0
	for head := range numHeads {
		ph := p.phase + float64(head)/numHeads
		ph -= math.Floor(ph)

		w := sine.Cycle(0.5 * ph)
		out += p.read(1.0+ph*p.window) * w * w
	}

	// sin^2 windows of numHeads evenly spaced heads always sum to numHeads / 2
	out *= 2.0 / numHeads

	p.phase += p.inc
	p.phase -= math.Floor(p.phase)

	return out
}
