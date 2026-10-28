// Package audioin is the sound input of the computer (a microphone, a line in) as a module of a patch: a voice to sing over a piece, to go through effects
// and a mixer like any other sound. Without an input (a render, a test) it is silent.
package audioin

import (
	"math"
	"slices"
	"sync/atomic"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/audio"
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
	sel  audio.InputSelector      // when the engine can choose the device it listens to
	devs atomic.Pointer[[]string] // the devices as the settings offered them (the message of the choice is an index in them: asking the engine takes its time, and the module takes its messages on the audio thread)
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
	if sel, ok := src.(audio.InputSelector); ok {
		in.sel = sel
	}
	return in
}

// Devices are the input devices to choose from (none: the engine has no choice), "default" first
func (in *Input) Devices() []string {
	if in.sel == nil {
		return nil
	}
	return in.sel.InputDevices()
}

// Device is the device listened to ("" when the engine has no choice)
func (in *Input) Device() string {
	if in.sel == nil {
		return ""
	}
	return in.sel.InputDevice()
}

// SetDevice listens to another device: the engine opens its stream again, which waits for the audio callback to finish, so it is done on a goroutine of its own (the module takes its
// messages on the audio thread)
func (in *Input) SetDevice(name string) {
	if in.sel != nil && name != "" && name != in.sel.InputDevice() {
		go func() { _ = in.sel.SetInputDevice(name) }()
	}
}

// Describe describes the settings of the input: its gain and mono, and with an engine that has a choice the device to listen to
func (in *Input) Describe() params.Set {
	set := Describe(in.cfg)
	if devs := in.Devices(); len(devs) > 1 {
		in.devs.Store(&devs)
		set.Params = append([]params.Param{params.C("Input", "device", devs, max(0, slices.Index(devs, in.Device())))}, set.Params...)
	}
	return set
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
	msg.IfIntVal("device", func(i int) {
		if devs := in.devs.Load(); devs != nil && i >= 0 && i < len(*devs) {
			in.SetDevice((*devs)[i])
		}
	})
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
