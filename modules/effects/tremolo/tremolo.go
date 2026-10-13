// Package tremolo is a tremolo and an auto-pan: an LFO moves the level up and down (tremolo), or the
// sound from side to side (pan). Synced to the tempo its phase follows the clock of the patch, so
// it pulses on the beats
package tremolo

import (
	"math"

	"github.com/almerlucke/psalm"
)

// Mode is what the LFO moves
type Mode int

const (
	Tremolo Mode = iota // the level
	Pan                 // the place, left and right
)

// Shape is the shape of the LFO
type Shape int

const (
	Sine     Shape = iota // smooth
	Triangle              // even
	Square                // on and off (with rounded edges, no clicks): a choppy gate
)

// Config describes a tremolo or auto-pan
type Config struct {
	Mode   Mode
	Shape  Shape
	Rate   float64 // Hz
	Sync   float64 // beats of one cycle instead (0.25 is 16ths), on the beats of the clock (used when above 0)
	Depth  float64 // 0-1: how deep the level goes (1 to silence), or how far to the sides
	Stereo float64 // 0-1 (tremolo), how far the left and right cycles are apart (0.5: they take turns)
}

// DefaultConfig is a tremolo in 8ths
func DefaultConfig() Config {
	return Config{Mode: Tremolo, Shape: Sine, Sync: 0.5, Depth: 0.5}
}

// Trem is the module: input 0 is stereo audio. Value inputs: 1 rate (Hz), 2 depth
type Trem struct {
	*psalm.Module
	cfg      Config
	phase    float64 // of the cycle, 0-1, at the start of the next buffer
	anchor   int64   // the tick of the first control tick of this buffer
	anchored bool    // a control tick came in this buffer
	anchorAt int     // its sample in the buffer
}

// New creates a tremolo or auto-pan
func New(cfg Config) *Trem {
	t := &Trem{Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}), cfg: cfg}
	t.SetSelf(t)
	return t
}

// Settings returns the settings
func (t *Trem) Settings() Config { return t.cfg }

func (t *Trem) SetRate(hz float64)    { t.cfg.Rate, t.cfg.Sync = hz, 0 }
func (t *Trem) SetSync(beats float64) { t.cfg.Sync = beats }
func (t *Trem) SetDepth(d float64)    { t.cfg.Depth = clamp(d, 0, 1) }

func clamp(x, lo, hi float64) float64 { return math.Max(lo, math.Min(x, hi)) }

func (t *Trem) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		t.ReceiveMessage(msg)
		return
	}
	v := psalm.FltVal[float64](value)
	switch index {
	case 1:
		t.SetRate(v)
	case 2:
		t.SetDepth(v)
	}
}

func (t *Trem) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("rate", t.SetRate)
	msg.IfFltVal("sync", t.SetSync)
	msg.IfFltVal("depth", t.SetDepth)
	msg.IfFltVal("stereo", func(s float64) { t.cfg.Stereo = s })
	msg.IfIntVal("shape", func(s int) { t.cfg.Shape = Shape(s) })
	msg.IfIntVal("mode", func(m int) { t.cfg.Mode = Mode(m) })
}

// CtrlTick keeps the first control tick of a buffer: synced, the phase comes from the clock
func (t *Trem) CtrlTick(tick psalm.CtrlTick) {
	if !t.anchored {
		t.anchor, t.anchorAt, t.anchored = tick.TickTimestamp, tick.SampleOffset, true
	}
}

// lfo returns the LFO at a phase, 0-1 (1 the top)
func lfo(shape Shape, phase float64) float64 {
	switch shape {
	case Triangle:
		return 1 - math.Abs(2*phase-1)
	case Square:
		return 0.5 + 0.5*math.Tanh(8*math.Cos(2*math.Pi*phase))/math.Tanh(8)
	default:
		return 0.5 + 0.5*math.Cos(2*math.Pi*phase)
	}
}

func (t *Trem) Synthesize() {
	var (
		cfg = t.cfg
		sr  = t.Config().SampleRate
		in  = t.Input(0).Channels
		out = t.Output(0).Channels
		inc = cfg.Rate / sr
	)
	if cfg.Sync > 0 {
		inc = t.BPM() / 60 / cfg.Sync / sr
		if t.anchored { // the phase at the start of the buffer, from the clock
			beat := float64(t.anchor) / float64(t.Config().PPQ)
			t.phase = beat/cfg.Sync - float64(t.anchorAt)*inc
			t.phase -= math.Floor(t.phase)
		}
	}
	t.anchored = false
	depth := clamp(cfg.Depth, 0, 1)
	for i := range out[0] {
		l, r := float64(in[0][i]), float64(in[len(in)-1][i])
		switch cfg.Mode {
		case Pan:
			// the place, -depth to depth, an equal power balance (1 in the middle)
			p := depth * (2*lfo(cfg.Shape, t.phase) - 1)
			a := (p + 1) * math.Pi / 4
			l *= math.Sqrt2 * math.Cos(a)
			r *= math.Sqrt2 * math.Sin(a)
		default:
			ph2 := t.phase + clamp(cfg.Stereo, 0, 1)
			l *= 1 - depth*(1-lfo(cfg.Shape, t.phase))
			r *= 1 - depth*(1-lfo(cfg.Shape, ph2-math.Floor(ph2)))
		}
		out[0][i], out[len(out)-1][i] = float32(l), float32(r)
		t.phase += inc
		t.phase -= math.Floor(t.phase)
	}
}

// Tail lets it rest while silent
func (t *Trem) Tail() float64 { return 0 }

// Restart starts the cycle again (psalm.Restarter)
func (t *Trem) Restart() { t.phase = 0 }
