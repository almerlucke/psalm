// Package crusher is a bit crusher: fewer bits (the steps of an old sampler, a grainy noise) and a
// lower sample rate (held samples, metallic aliasing): lo-fi on purpose
package crusher

import (
	"math"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/mix"
)

// Config describes a bit crusher
type Config struct {
	Bits   float64 // 1 to 24, the steps of the level (fractions in between: 4.5 is between 4 and 5 bits); 0 is off
	Rate   float64 // Hz, the sample rate it holds the sound at (0 is off)
	Output float64 // dB after it
	Mix    float64 // 0 dry, 1 crushed
}

// DefaultConfig is an old sampler: 8 bits at 11 kHz
func DefaultConfig() Config {
	return Config{Bits: 8, Rate: 11025, Mix: 1}
}

// Crusher is the module: input 0 is stereo audio. Value inputs: 1 bits, 2 rate, 3 mix
type Crusher struct {
	*psalm.Module
	cfg   Config
	phase float64    // of the held rate, 0-1
	held  [2]float64 // the samples held
	mix   mix.Smoother
}

// New creates a bit crusher
func New(cfg Config) *Crusher {
	c := &Crusher{
		Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}),
		cfg:    cfg,
		phase:  1, // the first sample is taken at once
		mix:    mix.NewSmoother(clamp(cfg.Mix, 0, 1), mix.SmoothMs, psalm.SampleRate()),
	}
	c.SetSelf(c)
	return c
}

// Settings returns the settings
func (c *Crusher) Settings() Config { return c.cfg }

func (c *Crusher) SetBits(b float64)    { c.cfg.Bits = b }
func (c *Crusher) SetRate(hz float64)   { c.cfg.Rate = hz }
func (c *Crusher) SetMix(m float64)     { c.cfg.Mix = m }
func (c *Crusher) SetOutput(db float64) { c.cfg.Output = db }

func clamp(x, lo, hi float64) float64 { return math.Max(lo, math.Min(x, hi)) }

func (c *Crusher) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		c.ReceiveMessage(msg)
		return
	}
	v := psalm.FltVal[float64](value)
	switch index {
	case 1:
		c.SetBits(v)
	case 2:
		c.SetRate(v)
	case 3:
		c.SetMix(v)
	}
}

func (c *Crusher) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("bits", c.SetBits)
	msg.IfFltVal("rate", c.SetRate)
	msg.IfFltVal("mix", c.SetMix)
	msg.IfFltVal("output", c.SetOutput)
}

// quantize rounds x to the steps of a number of bits (a fraction moves between the two)
func quantize(x, bits float64) float64 {
	if bits <= 0 || bits >= 24 {
		return x
	}
	bits = math.Max(1, bits)
	steps := math.Pow(2, bits-1)
	return clamp(math.Round(x*steps)/steps, -1, 1)
}

func (c *Crusher) Synthesize() {
	var (
		cfg    = c.cfg
		sr     = c.Config().SampleRate
		in     = c.Input(0).Channels
		out    = c.Output(0).Channels
		output = math.Pow(10, cfg.Output/20)
		mixTo  = clamp(cfg.Mix, 0, 1)
		inc    = 1.0
	)
	if cfg.Rate > 0 && cfg.Rate < sr {
		inc = cfg.Rate / sr
	}
	for i := range out[0] {
		c.phase += inc
		take := c.phase >= 1
		if take {
			c.phase -= math.Floor(c.phase)
		}
		m := c.mix.Next(mixTo)
		for ch := range out {
			x := float64(in[ch%len(in)][i])
			if take {
				c.held[ch] = quantize(x, cfg.Bits)
			}
			out[ch][i] = float32(x*(1-m) + c.held[ch]*output*m)
		}
	}
}

// Tail lets it rest while silent
func (c *Crusher) Tail() float64 { return 0 }
