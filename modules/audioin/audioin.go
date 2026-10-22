// Package audioin is the sound input of the computer (a microphone, a line in) as a module of a patch: a voice to sing over a piece, to go through effects
// and a mixer like any other sound. Without an input (a render, a test) it is silent.
package audioin

import (
	"math"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/mix"
	"github.com/almerlucke/psalm/params"
)

// Config describes an input
type Config struct {
	Gain float64 // dB, from -60 to 24
	Mono bool    // the left side on both sides (a microphone is on one channel of most interfaces)
}

// DefaultConfig is a microphone: mono, as it comes
func DefaultConfig() Config { return Config{Gain: 0, Mono: true} }

// Describe describes the settings of an input set as cfg (see params)
func Describe(cfg Config) params.Set {
	return params.Set{Name: "input", Params: []params.Param{
		params.K("Input", "gain", -60, 24, cfg.Gain, params.Unit("dB")),
		params.S("Input", "mono", cfg.Mono),
	}}
}

// Input is the module: no inputs, a stereo output. src is where the sound comes from, the input module of the patch of the
// audio engine (Psalm.PatchInput(0)); nil is silence
type Input struct {
	*psalm.Module
	cfg  Config
	src  psalm.Modular
	gain mix.Smoother
}

// New creates an input that reads src
func New(src psalm.Modular, cfg Config) *Input {
	in := &Input{
		Module: psalm.NewModule(nil, []int{psalm.Stereo}),
		cfg:    cfg,
		src:    src,
		gain:   mix.NewSmoother(db(cfg.Gain), mix.SmoothMs, psalm.SampleRate()),
	}
	in.SetSelf(in)
	return in
}

func db(x float64) float64 { return math.Pow(10, math.Max(-60, math.Min(x, 24))/20) }

// Settings returns the settings
func (in *Input) Settings() Config { return in.cfg }

func (in *Input) SetGain(dB float64) { in.cfg.Gain = dB }
func (in *Input) SetMono(on bool)    { in.cfg.Mono = on }

func (in *Input) Receive(value any, _ int) {
	if msg, ok := psalm.MsgVal(value); ok {
		in.ReceiveMessage(msg)
	}
}

func (in *Input) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("gain", in.SetGain)
	msg.IfFltVal("mono", func(v float64) { in.SetMono(v >= 0.5) })
}

func (in *Input) Synthesize() {
	out := in.Output(0).Channels
	if in.src == nil {
		in.Output(0).Clear()
		return
	}
	src := in.src.Output(0).Channels
	to := db(in.cfg.Gain)
	for i := range out[0] {
		g := float32(in.gain.Next(to))
		l, r := src[0][i], src[len(src)-1][i]
		if in.cfg.Mono {
			r = l
		}
		out[0][i], out[1][i] = l*g, r*g
	}
}
