// Package flanger is a stereo flanger: the input mixed with a copy of itself delayed by a few
// milliseconds that sweeps up and down, so the notches of the comb filter it makes sweep through
// the sound
package flanger

import (
	"math"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/delay"
	"github.com/almerlucke/psalm/blocks/sine"
)

// Shape is the shape of the sweep
type Shape int

const (
	Sine     Shape = iota // smooth
	Triangle              // the classic even sweep up and down
)

// Config describes a flanger
type Config struct {
	Rate     float64 // Hz, speed of the sweep
	Beats    float64 // length of a sweep in beats, follows the tempo; used instead of Rate when above 0
	Manual   float64 // milliseconds, the shortest delay (the highest notches), 0.1-10
	Depth    float64 // milliseconds the sweep reaches above Manual, 0-20
	Feedback float64 // -0.95 to 0.95: positive sharpens the notches into a metallic ring, negative hollows them out
	Stereo   float64 // 0-1: how far the sweep of the right side is out of phase with the left (1 is half a cycle)
	Shape    Shape
	Mix      float64 // 0 dry, 1 only the delayed copy; 0.5 makes the deepest notches
}

// DefaultConfig is a slow, classic stereo flanger
func DefaultConfig() Config {
	return Config{
		Rate:     0.2,
		Manual:   0.5,
		Depth:    4,
		Feedback: 0.5,
		Stereo:   0.5,
		Shape:    Triangle,
		Mix:      0.5,
	}
}

const (
	maxDelayMs = 21.0
	smoothMs   = 30.0 // settings glide this fast, so moving them doesn't click
)

// Flanger is the flanger module: stereo input 0. Value inputs: 1 rate, 2 depth, 3 feedback, 4 mix.
// Messages: "rate", "beats", "manual", "depth", "feedback", "stereo", "mix"
type Flanger struct {
	*psalm.Module
	cfg    Config
	lines  [2]*delay.Delay
	phase  float64
	manual float64 // smoothed settings
	depth  float64
	mix    float64
}

// New creates a flanger
func New(cfg Config) *Flanger {
	sr := psalm.SampleRate()
	n := int(math.Ceil(maxDelayMs*0.001*sr)) + 4
	f := &Flanger{
		Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}),
		cfg:    cfg,
		lines:  [2]*delay.Delay{delay.New(n), delay.New(n)},
		manual: cfg.Manual,
		depth:  cfg.Depth,
		mix:    cfg.Mix,
	}
	f.SetSelf(f)
	return f
}

// Settings returns the settings
func (f *Flanger) Settings() Config { return f.cfg }

func (f *Flanger) SetRate(rate float64)     { f.cfg.Rate = math.Max(0, rate); f.cfg.Beats = 0 }
func (f *Flanger) SetBeats(beats float64)   { f.cfg.Beats = math.Max(0, beats) }
func (f *Flanger) SetManual(ms float64)     { f.cfg.Manual = math.Max(0.1, math.Min(ms, 10)) }
func (f *Flanger) SetDepth(ms float64)      { f.cfg.Depth = math.Max(0, math.Min(ms, 20)) }
func (f *Flanger) SetFeedback(fb float64)   { f.cfg.Feedback = math.Max(-0.95, math.Min(fb, 0.95)) }
func (f *Flanger) SetStereo(stereo float64) { f.cfg.Stereo = math.Max(0, math.Min(stereo, 1)) }
func (f *Flanger) SetMix(mix float64)       { f.cfg.Mix = math.Max(0, math.Min(mix, 1)) }

func (f *Flanger) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		f.ReceiveMessage(msg)
		return
	}
	v := psalm.FltVal[float64](value)
	switch index {
	case 1:
		f.SetRate(v)
	case 2:
		f.SetDepth(v)
	case 3:
		f.SetFeedback(v)
	case 4:
		f.SetMix(v)
	}
}

func (f *Flanger) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("rate", f.SetRate)
	msg.IfFltVal("beats", f.SetBeats)
	msg.IfFltVal("manual", f.SetManual)
	msg.IfFltVal("depth", f.SetDepth)
	msg.IfFltVal("feedback", f.SetFeedback)
	msg.IfFltVal("stereo", f.SetStereo)
	msg.IfFltVal("mix", f.SetMix)
}

// sweep returns the sweep at a phase, 0-1
func (f *Flanger) sweep(phase float64) float64 {
	phase -= math.Floor(phase)
	if f.cfg.Shape == Triangle {
		return 1 - math.Abs(2*phase-1)
	}
	return 0.5 - 0.5*sine.CosCycle(phase)
}

func (f *Flanger) Synthesize() {
	var (
		cfg    = &f.cfg
		sr     = f.Config().SampleRate
		in     = f.Input(0).Channels
		out    = f.Output(0).Channels
		smooth = 1 - math.Exp(-1/(smoothMs*0.001*sr))
		msToS  = 0.001 * sr
		fb     = math.Max(-0.95, math.Min(cfg.Feedback, 0.95))
		rate   = cfg.Rate
		offset = 0.5 * math.Max(0, math.Min(cfg.Stereo, 1))
	)
	if cfg.Beats > 0 {
		rate = f.BPM() / 60 / cfg.Beats
	}

	for i := range out[0] {
		f.manual += (cfg.Manual - f.manual) * smooth
		f.depth += (cfg.Depth - f.depth) * smooth
		f.mix += (cfg.Mix - f.mix) * smooth
		ph := f.phase
		f.phase += rate / sr
		f.phase -= math.Floor(f.phase)

		for ch := range 2 {
			p := ph
			if ch == 1 {
				p += offset
			}
			d := (f.manual + f.depth*f.sweep(p)) * msToS
			x := float64(in[ch][i])
			delayed := f.lines[ch].ReadHermite(math.Max(1, d))
			f.lines[ch].Write(x + fb*delayed)
			out[ch][i] = float32(x*(1-f.mix) + delayed*f.mix)
		}
	}
}

// Tail lets the flanger rest while silent: its longest delay (feedback keeps the output and so
// the flanger going while it rings)
func (f *Flanger) Tail() float64 {
	return (f.cfg.Manual+f.cfg.Depth)/1000 + 0.02
}
