// Package meter measures audio: peak, RMS and loudness in LUFS after EBU R128 (ITU-R BS.1770):
// momentary (400 ms), short-term (3 s) and integrated (the whole programme, gated) loudness.
// The Meter module passes its stereo input through unchanged, put it anywhere in a patch
package meter

import (
	"fmt"
	"math"

	"github.com/almerlucke/psalm"
)

// Silence is the level reported for no signal, in dB and LUFS
const Silence = -120.0

// biquad is a second order filter (direct form 1)
type biquad struct {
	b0, b1, b2, a1, a2 float64
	x1, x2, y1, y2     float64
}

func (f *biquad) process(x float64) float64 {
	y := f.b0*x + f.b1*f.x1 + f.b2*f.x2 - f.a1*f.y1 - f.a2*f.y2
	f.x2, f.x1 = f.x1, x
	f.y2, f.y1 = f.y1, y
	return y
}

// kWeighting returns the two stages of the K weighting filter of BS.1770 for a sample rate: a high
// shelf of about +4 dB above 1.5 kHz (the head) and a high pass at 38 Hz
func kWeighting(sr float64) (biquad, biquad) {
	// stage 1, the shelf
	db, f0, q := 3.999843853973347, 1681.974450955533, 0.7071752369554196
	k := math.Tan(math.Pi * f0 / sr)
	vh := math.Pow(10, db/20)
	vb := math.Pow(vh, 0.4996667741545416)
	a0 := 1 + k/q + k*k
	shelf := biquad{
		b0: (vh + vb*k/q + k*k) / a0,
		b1: 2 * (k*k - vh) / a0,
		b2: (vh - vb*k/q + k*k) / a0,
		a1: 2 * (k*k - 1) / a0,
		a2: (1 - k/q + k*k) / a0,
	}
	// stage 2, the high pass
	f0, q = 38.13547087602444, 0.5003270373238773
	k = math.Tan(math.Pi * f0 / sr)
	a0 = 1 + k/q + k*k
	hp := biquad{
		b0: 1, b1: -2, b2: 1,
		a1: 2 * (k*k - 1) / a0,
		a2: (1 - k/q + k*k) / a0,
	}
	return shelf, hp
}

// Loudness measures loudness after EBU R128: feed it stereo samples with Add
type Loudness struct {
	shelf, hp [2]biquad
	block     int       // samples per 100 ms
	n         int       // samples in the current 100 ms
	sum       float64   // weighted energy of the current 100 ms
	recent    []float64 // energies of the last 30 blocks of 100 ms (3 s)
	gated     []float64 // energies of every 400 ms window (in steps of 100 ms), for integrated loudness
}

// NewLoudness creates a loudness meter for a sample rate
func NewLoudness(sr float64) *Loudness {
	shelf, hp := kWeighting(sr)
	return &Loudness{
		shelf: [2]biquad{shelf, shelf},
		hp:    [2]biquad{hp, hp},
		block: int(math.Round(sr / 10)),
	}
}

// Add adds a stereo sample
func (l *Loudness) Add(left, right float64) {
	a := l.hp[0].process(l.shelf[0].process(left))
	b := l.hp[1].process(l.shelf[1].process(right))
	l.sum += a*a + b*b
	l.n++
	if l.n < l.block {
		return
	}
	l.recent = append(l.recent, l.sum/float64(l.n))
	if len(l.recent) > 30 {
		l.recent = l.recent[1:]
	}
	if len(l.recent) >= 4 {
		l.gated = append(l.gated, mean(l.recent[len(l.recent)-4:]))
	}
	l.n, l.sum = 0, 0
}

// Reset starts measuring again
func (l *Loudness) Reset() {
	l.recent, l.gated = l.recent[:0], l.gated[:0]
	l.n, l.sum = 0, 0
}

func mean(x []float64) float64 {
	s := 0.0
	for _, v := range x {
		s += v
	}
	return s / float64(len(x))
}

// lufs converts a mean weighted energy to LUFS
func lufs(energy float64) float64 {
	if energy <= 0 {
		return Silence
	}
	return math.Max(Silence, -0.691+10*math.Log10(energy))
}

// Momentary returns the loudness of the last 400 ms in LUFS
func (l *Loudness) Momentary() float64 {
	if len(l.recent) < 4 {
		return Silence
	}
	return lufs(mean(l.recent[len(l.recent)-4:]))
}

// ShortTerm returns the loudness of the last 3 s in LUFS
func (l *Loudness) ShortTerm() float64 {
	if len(l.recent) == 0 {
		return Silence
	}
	return lufs(mean(l.recent))
}

// Integrated returns the loudness of everything since the start (or a reset) in LUFS, gated like
// EBU R128: windows below -70 LUFS are left out (silence), then windows more than 10 LU below the
// loudness of the rest (quiet passages)
func (l *Loudness) Integrated() float64 {
	var abs []float64
	for _, e := range l.gated {
		if lufs(e) > -70 {
			abs = append(abs, e)
		}
	}
	if len(abs) == 0 {
		return Silence
	}
	threshold := lufs(mean(abs)) - 10
	var rel []float64
	for _, e := range abs {
		if lufs(e) > threshold {
			rel = append(rel, e)
		}
	}
	return lufs(mean(rel))
}

// Meter is a module measuring its stereo input, which it passes on unchanged: peak, RMS and
// loudness. Read it between runs, Reset starts again
type Meter struct {
	*psalm.Module
	loud  *Loudness
	peak  float64
	sum   float64
	count int
}

// New creates a meter
func New() *Meter {
	m := &Meter{Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo})}
	m.loud = NewLoudness(m.Config().SampleRate)
	m.SetSelf(m)
	return m
}

func (m *Meter) Synthesize() {
	in := m.Input(0).Channels
	m.Output(0).Channels.Copy(in)
	for i := range in[0] {
		l, r := float64(in[0][i]), float64(in[1][i])
		m.peak = math.Max(m.peak, math.Max(math.Abs(l), math.Abs(r)))
		m.sum += (l*l + r*r) / 2
		m.count++
		m.loud.Add(l, r)
	}
}

// ToDB returns a gain in dB
func ToDB(g float64) float64 {
	if g <= 0 {
		return Silence
	}
	return math.Max(Silence, 20*math.Log10(g))
}

// Peak returns the highest sample in dB since the start or a reset
func (m *Meter) Peak() float64 { return ToDB(m.peak) }

// RMS returns the RMS level in dB since the start or a reset
func (m *Meter) RMS() float64 {
	if m.count == 0 {
		return Silence
	}
	return ToDB(math.Sqrt(m.sum / float64(m.count)))
}

// Loudness returns the loudness meter, for momentary, short-term and integrated loudness
func (m *Meter) Loudness() *Loudness { return m.loud }

// Reset starts measuring again
func (m *Meter) Reset() {
	m.peak, m.sum, m.count = 0, 0, 0
	m.loud.Reset()
}

// String reports peak, RMS and loudness
func (m *Meter) String() string {
	return fmt.Sprintf("peak %6.1f dB  rms %6.1f dB  integrated %6.1f LUFS  short-term %6.1f LUFS",
		m.Peak(), m.RMS(), m.loud.Integrated(), m.loud.ShortTerm())
}
