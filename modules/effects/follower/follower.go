// Package follower follows the loudness of a sound. The Follower passes the sound on and sends its
// level as a value (on every control tick), mapped to a range, for any setting to follow: a filter
// opening as a part gets louder, a reverb send growing with it. The Wah is an auto-wah: a resonant
// filter swept by the level, every note quacks open as it is played
package follower

import (
	"math"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/filters/svf"
	"github.com/almerlucke/psalm/blocks/mix"
)

// envelope follows a level: up over attack, down over release
type envelope struct {
	value float64
}

func coef(ms, sr float64) float64 {
	if ms <= 0 {
		return 1
	}
	return 1 - math.Exp(-1/(ms*0.001*sr))
}

func (e *envelope) next(x, attack, release float64) float64 {
	x = math.Abs(x)
	if x > e.value {
		e.value += (x - e.value) * attack
	} else {
		e.value += (x - e.value) * release
	}
	return e.value
}

// Config describes a follower
type Config struct {
	Attack   float64 // ms
	Release  float64 // ms
	Gain     float64 // dB of sensitivity: the level times it, 1 is the top of the range
	Min, Max float64 // the range of the value sent (0 and 1 when both are 0)
}

// DefaultConfig follows a part's level, the value from 0 to 1
func DefaultConfig() Config { return Config{Attack: 5, Release: 150, Gain: 6, Min: 0, Max: 1} }

// Follower is the module: input 0 is stereo audio, output 0 the same sound; value output 0 is the
// level, mapped from Min to Max, sent on every control tick
type Follower struct {
	*psalm.Module
	cfg Config
	env envelope
}

// New creates a follower
func New(cfg Config) *Follower {
	f := &Follower{Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}), cfg: cfg}
	f.SetSelf(f)
	return f
}

// Settings returns the settings
func (f *Follower) Settings() Config { return f.cfg }

// Level returns the level followed now (0-1, before the gain)
func (f *Follower) Level() float64 { return f.env.value }

// Value returns the value it sends: the level times the gain, mapped to the range
func (f *Follower) Value() float64 {
	lo, hi := f.cfg.Min, f.cfg.Max
	if lo == 0 && hi == 0 {
		hi = 1
	}
	x := min(1, f.env.value*math.Pow(10, f.cfg.Gain/20))
	return lo + (hi-lo)*x
}

func (f *Follower) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		f.ReceiveMessage(msg)
	}
}

func (f *Follower) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("attack", func(v float64) { f.cfg.Attack = v })
	msg.IfFltVal("release", func(v float64) { f.cfg.Release = v })
	msg.IfFltVal("gain", func(v float64) { f.cfg.Gain = v })
	msg.IfFltVal("min", func(v float64) { f.cfg.Min = v })
	msg.IfFltVal("max", func(v float64) { f.cfg.Max = v })
}

// CtrlTick sends the value
func (f *Follower) CtrlTick(psalm.CtrlTick) { f.Send(f.Value(), 0) }

func (f *Follower) Synthesize() {
	sr := f.Config().SampleRate
	attack, release := coef(f.cfg.Attack, sr), coef(f.cfg.Release, sr)
	in, out := f.Input(0).Channels, f.Output(0).Channels
	for i := range out[0] {
		l, r := float64(in[0][i]), float64(in[len(in)-1][i])
		f.env.next(max(math.Abs(l), math.Abs(r)), attack, release)
		out[0][i], out[len(out)-1][i] = in[0][i], in[len(in)-1][i]
	}
}

// Kind is the filter of a wah
type Kind int

const (
	BandPass Kind = iota // the classic quack
	LowPass              // a darker sweep, the sound opening up
)

// WahConfig describes an auto-wah
type WahConfig struct {
	Kind     Kind
	Min, Max float64 // Hz, the sweep from quiet to loud
	Q        float64 // resonance (0 is 4)
	Gain     float64 // dB of sensitivity: how far a note sweeps it
	Attack   float64 // ms
	Release  float64 // ms
	Down     bool    // a louder note sweeps it down instead of up
	Mix      float64 // 0 dry, 1 wah
}

// DefaultWah is a funky auto-wah for a clavinet, a bass or keys
func DefaultWah() WahConfig {
	return WahConfig{Kind: BandPass, Min: 350, Max: 2200, Q: 4, Gain: 12, Attack: 4, Release: 120, Mix: 1}
}

// Wah is the module: input 0 is stereo audio. Value inputs: 1 gain (sensitivity), 2 mix
type Wah struct {
	*psalm.Module
	cfg     WahConfig
	env     envelope
	filters [2]*svf.SVF
	mix     mix.Smoother
	count   int
}

// NewWah creates an auto-wah
func NewWah(cfg WahConfig) *Wah {
	sr := psalm.SampleRate()
	if cfg.Q <= 0 {
		cfg.Q = 4
	}
	w := &Wah{
		Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}),
		cfg:    cfg,
		mix:    mix.NewSmoother(max(0, min(cfg.Mix, 1)), mix.SmoothMs, sr),
	}
	for c := range w.filters {
		w.filters[c] = svf.New(cfg.Min, 0, sr)
		w.filters[c].SetFrequencyQ(max(cfg.Min, 20), cfg.Q)
	}
	w.SetSelf(w)
	return w
}

// Settings returns the settings
func (w *Wah) Settings() WahConfig { return w.cfg }

// Frequency returns where the filter is now
func (w *Wah) Frequency() float64 { return w.freq() }

func (w *Wah) freq() float64 {
	lo, hi := max(w.cfg.Min, 20), max(w.cfg.Max, 20)
	x := min(1, w.env.value*math.Pow(10, w.cfg.Gain/20))
	if w.cfg.Down {
		x = 1 - x
	}
	return lo * math.Pow(hi/lo, x) // over the octaves
}

func (w *Wah) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		w.ReceiveMessage(msg)
		return
	}
	v := psalm.FltVal[float64](value)
	switch index {
	case 1:
		w.cfg.Gain = v
	case 2:
		w.cfg.Mix = v
	}
}

func (w *Wah) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("gain", func(v float64) { w.cfg.Gain = v })
	msg.IfFltVal("mix", func(v float64) { w.cfg.Mix = v })
	msg.IfFltVal("min", func(v float64) { w.cfg.Min = v })
	msg.IfFltVal("max", func(v float64) { w.cfg.Max = v })
	msg.IfFltVal("q", func(v float64) { w.cfg.Q = v })
	msg.IfFltVal("attack", func(v float64) { w.cfg.Attack = v })
	msg.IfFltVal("release", func(v float64) { w.cfg.Release = v })
	msg.IfIntVal("kind", func(k int) { w.cfg.Kind = Kind(max(0, min(k, int(LowPass)))) })
	msg.IfBoolVal("down", func(b bool) { w.cfg.Down = b })
}

func (w *Wah) Synthesize() {
	var (
		cfg     = w.cfg
		sr      = w.Config().SampleRate
		in      = w.Input(0).Channels
		out     = w.Output(0).Channels
		attack  = coef(cfg.Attack, sr)
		release = coef(cfg.Release, sr)
		mode    = svf.BandPass
		mixTo   = max(0, min(cfg.Mix, 1))
		q       = cfg.Q
	)
	if cfg.Kind == LowPass {
		mode = svf.LowPass
	}
	if q <= 0 {
		q = 4
	}
	for i := range out[0] {
		l, r := float64(in[0][i]), float64(in[len(in)-1][i])
		w.env.next(max(math.Abs(l), math.Abs(r)), attack, release)
		if w.count%16 == 0 { // the filter follows every 16 samples
			f := w.freq()
			for _, flt := range w.filters {
				flt.SetFrequencyQ(f, q)
			}
		}
		w.count++
		m := w.mix.Next(mixTo)
		for c := range out {
			x := float64(in[c%len(in)][i])
			y := w.filters[c].Process(x, mode)
			out[c][i] = float32(x*(1-m) + y*m)
		}
	}
}

// Tail lets it rest while silent (the resonance rings a moment)
func (w *Wah) Tail() float64 { return 0.1 }
