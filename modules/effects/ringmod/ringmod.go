// Package ringmod is a ring modulator and a frequency shifter. Ring modulation multiplies the sound
// with a sine: every frequency becomes its sum and its difference with the carrier, a metallic, bell
// like sound. Frequency shifting moves every frequency by the same number of Hz (up or down): the
// harmonics of a note stop lining up, from a slight beating to a strange, inharmonic sound
package ringmod

import (
	"math"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/mix"
)

// Mode is ring modulation or a frequency shift
type Mode int

const (
	Ring  Mode = iota // the sum and the difference with the carrier
	Shift             // every frequency moved by Freq Hz (negative is down)
)

// Config describes a ring modulator or frequency shifter
type Config struct {
	Mode   Mode
	Freq   float64 // Hz of the carrier, or of the shift
	Stereo float64 // Hz the right side differs (a shift up on one side, down on the other is wide)
	Mix    float64 // 0 dry, 1 only the modulated sound
}

// DefaultConfig is a ring modulator at 300 Hz
func DefaultConfig() Config { return Config{Mode: Ring, Freq: 300, Mix: 0.5} }

// hilbert is an IIR Hilbert transform: two allpass chains whose outputs are 90 degrees apart over
// the audible band (Olli Niemitalo's coefficients)
type hilbert struct {
	a, b [4]allpass
	prev float64 // the first chain's output delayed a sample
}

type allpass struct {
	c              float64
	x1, x2, y1, y2 float64
}

func (ap *allpass) process(x float64) float64 {
	y := ap.c*(x+ap.y2) - ap.x2
	ap.x2, ap.x1 = ap.x1, x
	ap.y2, ap.y1 = ap.y1, y
	return y
}

func newHilbert() *hilbert {
	h := &hilbert{}
	for i, c := range []float64{0.6923878, 0.9360654322959, 0.9882295226860, 0.9987488452737} {
		h.a[i].c = c * c
	}
	for i, c := range []float64{0.4021921162426, 0.8561710882420, 0.9722909545651, 0.9952884791278} {
		h.b[i].c = c * c
	}
	return h
}

// process returns the sound and its 90 degree shifted version (re, im)
func (h *hilbert) process(x float64) (float64, float64) {
	re, im := x, x
	for i := range h.a {
		re = h.a[i].process(re)
		im = h.b[i].process(im)
	}
	re, h.prev = h.prev, re
	return re, im
}

// RingMod is the module: input 0 is stereo audio. Value inputs: 1 frequency, 2 mix
type RingMod struct {
	*psalm.Module
	cfg   Config
	phase [2]float64
	h     [2]*hilbert
	mix   mix.Smoother
	freq  [2]mix.Smoother
}

// New creates a ring modulator or frequency shifter
func New(cfg Config) *RingMod {
	sr := psalm.SampleRate()
	r := &RingMod{
		Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}),
		cfg:    cfg,
		h:      [2]*hilbert{newHilbert(), newHilbert()},
		mix:    mix.NewSmoother(clamp(cfg.Mix, 0, 1), mix.SmoothMs, sr),
	}
	for c := range r.freq {
		r.freq[c] = mix.NewSmoother(cfg.Freq+float64(c)*cfg.Stereo, mix.SmoothMs, sr)
	}
	r.SetSelf(r)
	return r
}

// Settings returns the settings
func (r *RingMod) Settings() Config { return r.cfg }

func (r *RingMod) SetFreq(hz float64) { r.cfg.Freq = hz }
func (r *RingMod) SetMix(m float64)   { r.cfg.Mix = m }

func clamp(x, lo, hi float64) float64 { return max(lo, min(x, hi)) }

func (r *RingMod) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		r.ReceiveMessage(msg)
		return
	}
	v := psalm.FltVal[float64](value)
	switch index {
	case 1:
		r.SetFreq(v)
	case 2:
		r.SetMix(v)
	}
}

func (r *RingMod) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("freq", r.SetFreq)
	msg.IfFltVal("mix", r.SetMix)
	msg.IfFltVal("stereo", func(v float64) { r.cfg.Stereo = v })
	msg.IfIntVal("mode", func(m int) { r.cfg.Mode = Mode(m) })
}

func (r *RingMod) Synthesize() {
	var (
		cfg   = r.cfg
		sr    = r.Config().SampleRate
		in    = r.Input(0).Channels
		out   = r.Output(0).Channels
		mixTo = clamp(cfg.Mix, 0, 1)
	)
	for i := range out[0] {
		m := r.mix.Next(mixTo) // linear: the dry and the modulated sound are related
		for c := range out {
			x := float64(in[c%len(in)][i])
			f := r.freq[c].Next(cfg.Freq + float64(c)*cfg.Stereo)
			a := 2 * math.Pi * r.phase[c]
			var y float64
			if cfg.Mode == Shift {
				re, im := r.h[c].process(x)
				y = re*math.Cos(a) + im*math.Sin(a) // only the side shifted by the frequency (up when positive)
			} else {
				y = x * math.Sin(a)
			}
			r.phase[c] += f / sr
			r.phase[c] -= math.Floor(r.phase[c])
			out[c][i] = float32(x*(1-m) + y*m)
		}
	}
}

// Tail lets it rest while silent
func (r *RingMod) Tail() float64 { return 0.01 }
