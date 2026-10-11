// Package phaser is a phaser: the sound through a chain of allpass filters whose frequency an LFO
// sweeps, mixed with itself, makes notches that move up and down: the swoosh of a 70s electric
// piano or strings
package phaser

import (
	"math"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/mix"
)

// Config describes a phaser
type Config struct {
	Stages   int     // allpass stages: 4 (the classic), 6, 8, 12 (more notches); 0 is 4
	Rate     float64 // Hz of the sweep
	Sync     float64 // beats of one sweep instead, following the tempo (used when above 0)
	Min, Max float64 // Hz, the range of the sweep
	Feedback float64 // -0.95 to 0.95: deeper, more resonant notches (negative moves them)
	Stereo   float64 // 0-1, how far apart the sweeps of left and right are (0.25 is a quarter)
	Mix      float64 // 0 dry, 0.5 the deepest notches, 1 only the shifted sound (a vibrato)
}

// DefaultConfig is a slow, warm phaser
func DefaultConfig() Config {
	return Config{Stages: 4, Rate: 0.4, Min: 300, Max: 2500, Feedback: 0.4, Stereo: 0.25, Mix: 0.5}
}

// Phaser is the module: input 0 is stereo audio. Value inputs: 1 rate, 2 feedback, 3 mix
type Phaser struct {
	*psalm.Module
	cfg   Config
	phase float64
	state [2][]float64 // per channel the allpass states (input and output of the last sample)
	last  [2]float64   // the output of the chain, for the feedback
	mix   mix.Smoother
	fb    mix.Smoother
}

// New creates a phaser
func New(cfg Config) *Phaser {
	if cfg.Stages <= 0 {
		cfg.Stages = 4
	}
	sr := psalm.SampleRate()
	p := &Phaser{
		Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}),
		cfg:    cfg,
		mix:    mix.NewSmoother(clamp(cfg.Mix, 0, 1), mix.SmoothMs, sr),
		fb:     mix.NewSmoother(clamp(cfg.Feedback, -0.95, 0.95), mix.SmoothMs, sr),
	}
	for c := range p.state {
		p.state[c] = make([]float64, 2*cfg.Stages)
	}
	p.SetSelf(p)
	return p
}

// Settings returns the settings
func (p *Phaser) Settings() Config { return p.cfg }

func (p *Phaser) SetRate(hz float64)    { p.cfg.Rate, p.cfg.Sync = hz, 0 }
func (p *Phaser) SetSync(beats float64) { p.cfg.Sync = beats }
func (p *Phaser) SetFeedback(f float64) { p.cfg.Feedback = f }
func (p *Phaser) SetMix(m float64)      { p.cfg.Mix = m }

func clamp(x, lo, hi float64) float64 { return math.Max(lo, math.Min(x, hi)) }

func (p *Phaser) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		p.ReceiveMessage(msg)
		return
	}
	v := psalm.FltVal[float64](value)
	switch index {
	case 1:
		p.SetRate(v)
	case 2:
		p.SetFeedback(v)
	case 3:
		p.SetMix(v)
	}
}

func (p *Phaser) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("rate", p.SetRate)
	msg.IfFltVal("sync", p.SetSync)
	msg.IfFltVal("feedback", p.SetFeedback)
	msg.IfFltVal("mix", p.SetMix)
	msg.IfFltVal("min", func(f float64) { p.cfg.Min = f })
	msg.IfFltVal("max", func(f float64) { p.cfg.Max = f })
	msg.IfFltVal("stereo", func(f float64) { p.cfg.Stereo = f })
}

// rate returns the sweeps per second
func (p *Phaser) rate() float64 {
	if p.cfg.Sync > 0 {
		return p.BPM() / 60 / p.cfg.Sync
	}
	return p.cfg.Rate
}

// coefficient returns the allpass coefficient of a frequency
func coefficient(fc, sr float64) float64 {
	t := math.Tan(math.Pi * clamp(fc, 10, sr*0.45) / sr)
	return (t - 1) / (t + 1)
}

func (p *Phaser) Synthesize() {
	var (
		cfg   = p.cfg
		sr    = p.Config().SampleRate
		in    = p.Input(0).Channels
		out   = p.Output(0).Channels
		inc   = p.rate() / sr
		lo    = math.Max(10, math.Min(cfg.Min, cfg.Max))
		hi    = math.Max(lo, math.Max(cfg.Min, cfg.Max))
		mixTo = clamp(cfg.Mix, 0, 1)
		fbTo  = clamp(cfg.Feedback, -0.95, 0.95)
	)
	for i := range out[0] {
		m := p.mix.Next(mixTo)
		fb := p.fb.Next(fbTo)
		for c := range out {
			ph := p.phase + float64(c)*cfg.Stereo
			sweep := 0.5 - 0.5*math.Cos(2*math.Pi*ph)       // 0-1, smooth at the turns
			a := coefficient(lo*math.Pow(hi/lo, sweep), sr) // exponential: even over the octaves
			x := float64(in[c%len(in)][i])
			v := x + fb*p.last[c]
			st := p.state[c]
			for s := 0; s < len(st); s += 2 { // first order allpass: y = a*x + x1 - a*y1
				y := a*v + st[s] - a*st[s+1]
				st[s], st[s+1] = v, y
				v = y
			}
			p.last[c] = v
			out[c][i] = float32(x*(1-m) + v*m) // linear: the notches come from cancelling
		}
		p.phase += inc
		p.phase -= math.Floor(p.phase)
	}
}

// Tail lets it rest while silent (with feedback it rings a moment)
func (p *Phaser) Tail() float64 { return 0.1 }
