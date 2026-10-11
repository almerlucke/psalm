// Package widener is a stereo widener: the sound split into what both sides share (the middle)
// and what differs (the sides); more of the sides is wider, less is narrower, none is mono. The low
// end can be kept in the middle, where a bass belongs (and where it stays solid on any speaker)
package widener

import (
	"math"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/filters/biquad"
	"github.com/almerlucke/psalm/blocks/mix"
)

// Config describes a widener
type Config struct {
	Width     float64 // 0 mono, 1 as it is, 2 twice as wide (at most 3)
	MonoBelow float64 // Hz, below it the sound is kept in the middle (0 is off)
}

// DefaultConfig widens a little and keeps the bass in the middle
func DefaultConfig() Config { return Config{Width: 1.4, MonoBelow: 120} }

// Widener is the module: input 0 is stereo audio. Value inputs: 1 width
type Widener struct {
	*psalm.Module
	cfg   Config
	hp    [2]*biquad.Biquad // a fourth order highpass on the sides
	width mix.Smoother
}

// New creates a widener
func New(cfg Config) *Widener {
	sr := psalm.SampleRate()
	w := &Widener{
		Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}),
		cfg:    cfg,
		width:  mix.NewSmoother(clamp(cfg.Width, 0, 3), mix.SmoothMs, sr),
	}
	for i := range w.hp {
		w.hp[i] = biquad.New(biquad.HighPass, math.Max(cfg.MonoBelow, 10), 0.707, 0, sr)
	}
	w.SetSelf(w)
	return w
}

// Settings returns the settings
func (w *Widener) Settings() Config { return w.cfg }

func (w *Widener) SetWidth(width float64) { w.cfg.Width = width }

// SetMonoBelow sets the frequency below which the sound is kept in the middle (0 is off)
func (w *Widener) SetMonoBelow(hz float64) {
	w.cfg.MonoBelow = hz
	for _, f := range w.hp {
		f.SetFrequency(math.Max(hz, 10))
	}
}

func clamp(x, lo, hi float64) float64 { return math.Max(lo, math.Min(x, hi)) }

func (w *Widener) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		w.ReceiveMessage(msg)
		return
	}
	if index == 1 {
		w.SetWidth(psalm.FltVal[float64](value))
	}
}

func (w *Widener) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("width", w.SetWidth)
	msg.IfFltVal("monoBelow", w.SetMonoBelow)
}

func (w *Widener) Synthesize() {
	in, out := w.Input(0).Channels, w.Output(0).Channels
	widthTo := clamp(w.cfg.Width, 0, 3)
	mono := w.cfg.MonoBelow > 0
	for i := range out[0] {
		l, r := float64(in[0][i]), float64(in[len(in)-1][i])
		mid, side := (l+r)/2, (l-r)/2
		if mono {
			side = w.hp[1].Process(w.hp[0].Process(side))
		}
		side *= w.width.Next(widthTo)
		out[0][i] = float32(mid + side)
		out[len(out)-1][i] = float32(mid - side)
	}
}

// Tail lets it rest while silent
func (w *Widener) Tail() float64 { return 0.02 }
