// Package gain is a volume control in dB: a trim at the end of a chain that has no mixer, or a
// level to automate. Changes glide, so automating the volume doesn't click
package gain

import (
	"math"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/mix"
)

// Gain is the gain module: stereo input 0. Value input 1 is the volume (dB), message "volume"
type Gain struct {
	*psalm.Module
	volume float64 // dB
	gain   mix.Smoother
}

// New creates a gain of volume dB (0 is unity, -120 or less is silence)
func New(volume float64) *Gain {
	g := &Gain{
		Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}),
		volume: volume,
	}
	g.gain = mix.NewSmoother(g.target(), mix.SmoothMs, psalm.SampleRate())
	g.SetSelf(g)
	return g
}

// Volume returns the volume in dB
func (g *Gain) Volume() float64 { return g.volume }

// SetVolume sets the volume in dB
func (g *Gain) SetVolume(db float64) { g.volume = db }

func (g *Gain) target() float64 {
	if g.volume <= -120 {
		return 0
	}
	return math.Pow(10, g.volume/20)
}

func (g *Gain) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		g.ReceiveMessage(msg)
		return
	}
	if index == 1 {
		g.SetVolume(psalm.FltVal[float64](value))
	}
}

func (g *Gain) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("volume", g.SetVolume)
}

func (g *Gain) Synthesize() {
	ins := g.Input(0).Channels
	outs := g.Output(0).Channels
	target := g.target()
	for i := range g.Config().BufferSize {
		v := g.gain.Next(target)
		outs[0][i] = ins[0][i] * float32(v)
		outs[1][i] = ins[1][i] * float32(v)
	}
}

// Tail lets the gain rest while silent: it has no memory
func (g *Gain) Tail() float64 { return 0 }
