package vco

import (
	"math"
	"math/rand/v2"
)

type LFOWaveform int

const (
	LFOSine LFOWaveform = iota
	LFOTriangle
	LFOSaw  // rising
	LFORamp // falling
	LFOSquare
	LFOSampleHold // new random value every cycle
)

// LFOConfig describes a per-voice LFO and how much it modulates each destination,
// all amounts are zero (off) by default
type LFOConfig struct {
	Waveform LFOWaveform
	Rate     float64 // Hz
	Delay    float64 // fade-in time in milliseconds after note on (classic delayed vibrato)
	KeySync  bool    // restart the LFO cycle on every note

	// Destinations
	Pitch      float64 // semitones
	Cutoff     float64 // octaves
	PulseWidth float64 // added to pulse width (0-1 range)
	Amp        float64 // tremolo depth 0-1
}

func (c *LFOConfig) active() bool {
	return c.Rate > 0.0 && (c.Pitch != 0.0 || c.Cutoff != 0.0 || c.PulseWidth != 0.0 || c.Amp != 0.0)
}

// lfo is a simple bipolar (-1 to 1) low frequency oscillator with delayed fade-in
type lfo struct {
	phase   float64
	sh      float64
	fade    float64
	fadeInc float64
}

func (l *lfo) trigger(cfg *LFOConfig, sr float64) {
	if cfg.KeySync {
		l.phase = 0.0
		l.sh = rand.Float64()*2.0 - 1.0
	}

	if cfg.Delay > 0.0 {
		l.fade = 0.0
		l.fadeInc = 1.0 / (cfg.Delay * 0.001 * sr)
	} else {
		l.fade = 1.0
		l.fadeInc = 0.0
	}
}

func (l *lfo) tick(cfg *LFOConfig, sr float64) float64 {
	var v float64

	switch cfg.Waveform {
	case LFOTriangle:
		v = 1.0 - 4.0*math.Abs(l.phase-0.5)
	case LFOSaw:
		v = 2.0*l.phase - 1.0
	case LFORamp:
		v = 1.0 - 2.0*l.phase
	case LFOSquare:
		if l.phase < 0.5 {
			v = 1.0
		} else {
			v = -1.0
		}
	case LFOSampleHold:
		v = l.sh
	default:
		v = math.Sin(2.0 * math.Pi * l.phase)
	}

	l.phase += cfg.Rate / sr
	if l.phase >= 1.0 {
		l.phase -= math.Floor(l.phase)
		l.sh = rand.Float64()*2.0 - 1.0
	}

	if l.fade < 1.0 {
		l.fade = math.Min(1.0, l.fade+l.fadeInc)
	}

	return v * l.fade
}

type NoiseType int

const (
	WhiteNoise NoiseType = iota
	PinkNoise
)

// noise generates white or pink noise (Paul Kellet's economy pink filter)
type noise struct {
	b0, b1, b2 float64
}

func (n *noise) tick(nt NoiseType) float64 {
	white := rand.Float64()*2.0 - 1.0
	if nt != PinkNoise {
		return white
	}

	n.b0 = 0.99765*n.b0 + white*0.0990460
	n.b1 = 0.96300*n.b1 + white*0.2965164
	n.b2 = 0.57000*n.b2 + white*1.0526913

	return (n.b0 + n.b1 + n.b2 + white*0.1848) * 0.25
}
