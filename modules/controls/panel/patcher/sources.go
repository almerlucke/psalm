package patcher

import (
	"math"
	"math/rand/v2"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/params"
)

// LFO is a modulation source: a wave at a rate (or a number of beats of the patch's tempo), its
// value 0-1. It takes "rate" (Hz), "beats" (a cycle in beats, 0 follows the rate), "wave" (sine,
// triangle, saw up, saw down, square) and "phase" (0-1, where a cycle starts)
type LFO struct {
	*psalm.Module
	Rate, Beats, Phase float64
	Wave               int
	pos                float64 // where in the cycle, 0-1 (moved once a buffer)
	buffer             int64
	value              atomicFloat
}

// LFOWaves are the names of the LFO's waves
var LFOWaves = []string{"sine", "triangle", "saw up", "saw down", "square"}

// NewLFO returns an LFO at a rate
func NewLFO(rate float64) *LFO {
	l := &LFO{Module: psalm.NewControl(), Rate: rate, buffer: -1}
	l.SetSelf(l)
	return l
}

func (l *LFO) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("rate", func(f float64) { l.Rate = max(0, f) })
	msg.IfFltVal("beats", func(f float64) { l.Beats = max(0, f) })
	msg.IfFltVal("phase", func(f float64) { l.Phase = f - math.Floor(f) })
	msg.IfIntVal("wave", func(w int) { l.Wave = max(0, min(w, len(LFOWaves)-1)) })
}

func (l *LFO) Receive(v any, _ int) {
	if m, ok := psalm.MsgVal(v); ok {
		l.ReceiveMessage(m)
	}
}

// CtrlTick moves the wave a buffer on, at the start of every buffer
func (l *LFO) CtrlTick(tick psalm.CtrlTick) {
	cfg := l.Config()
	buffer := tick.SampleTimestamp / int64(max(1, cfg.BufferSize))
	if buffer == l.buffer {
		return
	}
	l.buffer = buffer
	rate := l.Rate
	if l.Beats > 0 {
		rate = l.BPM() / 60 / l.Beats
	}
	l.pos += rate * float64(cfg.BufferSize) / cfg.SampleRate
	l.pos -= math.Floor(l.pos)
	p := l.pos + l.Phase
	p -= math.Floor(p)
	var v float64
	switch l.Wave {
	case 0:
		v = 0.5 - 0.5*math.Cos(2*math.Pi*p)
	case 1:
		v = 1 - math.Abs(2*p-1)
	case 2:
		v = p
	case 3:
		v = 1 - p
	default:
		if p < 0.5 {
			v = 1
		}
	}
	l.value.Store(v)
}

// Value is where the wave is, 0-1
func (l *LFO) Value() float64 { return l.value.Load() }

// Describe describes an LFO's settings
func (l *LFO) Describe() params.Set {
	return params.Set{Name: "lfo", Params: []params.Param{
		params.K("LFO", "rate", 0.01, 40, max(0.01, l.Rate), params.Exp(), params.Unit("Hz")),
		params.K("LFO", "beats", 0, 32, l.Beats, params.Label("beats (0: the rate)")),
		params.C("LFO", "wave", LFOWaves, l.Wave),
		params.K("LFO", "phase", 0, 1, l.Phase),
	}}
}

// Random is a modulation source: a new random value every so often ("rate" Hz, or "beats"), gliding
// to it ("smooth", 0 jumps, 1 glides all the way), 0-1
type Random struct {
	*psalm.Module
	Rate, Beats, Smooth float64
	since               float64 // seconds since the last new value
	from, to, at        float64
	buffer              int64
	rng                 *rand.Rand
	value               atomicFloat
}

// NewRandom returns a random source at a rate
func NewRandom(rate float64) *Random {
	r := &Random{Module: psalm.NewControl(), Rate: rate, buffer: -1, rng: rand.New(rand.NewPCG(rand.Uint64(), 7))}
	r.to = r.rng.Float64()
	r.SetSelf(r)
	return r
}

func (r *Random) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("rate", func(f float64) { r.Rate = max(0, f) })
	msg.IfFltVal("beats", func(f float64) { r.Beats = max(0, f) })
	msg.IfFltVal("smooth", func(f float64) { r.Smooth = max(0, min(f, 1)) })
}

func (r *Random) Receive(v any, _ int) {
	if m, ok := psalm.MsgVal(v); ok {
		r.ReceiveMessage(m)
	}
}

func (r *Random) CtrlTick(tick psalm.CtrlTick) {
	cfg := r.Config()
	buffer := tick.SampleTimestamp / int64(max(1, cfg.BufferSize))
	if buffer == r.buffer {
		return
	}
	r.buffer = buffer
	period := 0.0
	if r.Beats > 0 {
		period = r.Beats * 60 / r.BPM()
	} else if r.Rate > 0 {
		period = 1 / r.Rate
	}
	r.since += float64(cfg.BufferSize) / cfg.SampleRate
	if period > 0 && r.since >= period {
		r.since -= period * math.Floor(r.since/period)
		r.from, r.to = r.at, r.rng.Float64()
	}
	glide := 1.0
	if period > 0 && r.Smooth > 0 {
		glide = min(1, r.since/(period*r.Smooth))
	}
	r.at = r.from + (r.to-r.from)*glide
	r.value.Store(r.at)
}

// Value is the random value now, 0-1
func (r *Random) Value() float64 { return r.value.Load() }

// Describe describes the random source's settings
func (r *Random) Describe() params.Set {
	return params.Set{Name: "random", Params: []params.Param{
		params.K("Random", "rate", 0.01, 40, max(0.01, r.Rate), params.Exp(), params.Unit("Hz")),
		params.K("Random", "beats", 0, 32, r.Beats, params.Label("beats (0: the rate)")),
		params.K("Random", "smooth", 0, 1, r.Smooth),
	}}
}

// sends is the number of sends and returns of a mixer, named A to D
const sends = 4

// Mixer is a mixer of channels for a patch: every channel (a stereo input) at a level, panned,
// muted or not, and sent to four buses, A to D (after its level: a reverb, a delay, a granulator,
// a shimmer); the buses are outputs of their own, to patch into effects whose outputs come back on
// the returns. Its inputs are the channels, then returns A to D; its outputs main, then sends A to D.
// Settings: "level<n>", "pan<n>" (-1 to 1), "mute<n>", "sendA<n>" to "sendD<n>" for every channel from 1,
// "returnA" to "returnD" and "level" (the main out)
type Mixer struct {
	*psalm.Module
	Channels   int
	Level, Pan []float64
	Sends      [sends][]float64 // the amount of every channel sent to each bus
	Mute       []bool
	Return     [sends]float64
	Master     float64
}

// NewMixer returns a mixer of channels, every one at level 1 in the middle, the sends off, the
// returns and the main out at 1
func NewMixer(channels int) *Mixer {
	channels = max(1, channels)
	ins := make([]int, channels+sends)
	for i := range ins {
		ins[i] = psalm.Stereo
	}
	outs := make([]int, 1+sends)
	for i := range outs {
		outs[i] = psalm.Stereo
	}
	m := &Mixer{Module: psalm.NewModule(ins, outs), Channels: channels,
		Level: make([]float64, channels), Pan: make([]float64, channels), Mute: make([]bool, channels), Master: 1}
	for i := range m.Sends {
		m.Sends[i] = make([]float64, channels)
		m.Return[i] = 1
	}
	for i := range m.Level {
		m.Level[i] = 1
	}
	m.SetSelf(m)
	return m
}

// busName is the letter of a send or a return
func busName(i int) string { return string(rune('A' + i)) }

func (m *Mixer) ReceiveMessage(msg psalm.Message) {
	clamp := func(v, lo, hi float64) float64 { return max(lo, min(v, hi)) }
	for i := range m.Channels {
		n := strconv.Itoa(i + 1)
		msg.IfFltVal("level"+n, func(f float64) { m.Level[i] = clamp(f, 0, 2) })
		msg.IfFltVal("pan"+n, func(f float64) { m.Pan[i] = clamp(f, -1, 1) })
		for b := range sends {
			msg.IfFltVal("send"+busName(b)+n, func(f float64) { m.Sends[b][i] = clamp(f, 0, 1) })
		}
		msg.IfBoolVal("mute"+n, func(b bool) { m.Mute[i] = b })
	}
	for b := range sends {
		msg.IfFltVal("return"+busName(b), func(f float64) { m.Return[b] = clamp(f, 0, 2) })
	}
	msg.IfFltVal("level", func(f float64) { m.Master = clamp(f, 0, 2) })
}

func (m *Mixer) Receive(v any, _ int) {
	if msg, ok := psalm.MsgVal(v); ok {
		m.ReceiveMessage(msg)
	}
}

func (m *Mixer) Synthesize() {
	main := m.Output(0).Channels
	main.Clear()
	var bus [sends]psalm.Channels
	for b := range bus {
		bus[b] = m.Output(1 + b).Channels
		bus[b].Clear()
	}
	for i := range m.Channels {
		in := m.Input(i)
		if !in.IsConnected() || m.Mute[i] || m.Level[i] == 0 {
			continue
		}
		// a constant power pan: the middle is as loud as either side
		angle := (m.Pan[i] + 1) * math.Pi / 4
		gains := [2]float32{float32(m.Level[i] * math.Cos(angle) * math.Sqrt2), float32(m.Level[i] * math.Sin(angle) * math.Sqrt2)}
		var amount [sends]float32
		for b := range amount {
			amount[b] = float32(m.Sends[b][i])
		}
		for c := range 2 {
			src := in.Channels[c%len(in.Channels)]
			g := gains[c]
			for j, x := range src {
				v := x * g
				main[c][j] += v
				for b, s := range amount {
					if s > 0 {
						bus[b][c][j] += v * s
					}
				}
			}
		}
	}
	for r, level := range m.Return {
		in := m.Input(m.Channels + r)
		if !in.IsConnected() || level == 0 {
			continue
		}
		for c := range 2 {
			src := in.Channels[c%len(in.Channels)]
			for j, x := range src {
				main[c][j] += x * float32(level)
			}
		}
	}
	if m.Master != 1 {
		for c := range main {
			for j := range main[c] {
				main[c][j] *= float32(m.Master)
			}
		}
	}
}

// Describe describes the mixer's settings: a group for every channel, the returns and the main out
func (m *Mixer) Describe() params.Set {
	s := params.Set{Name: "mixer"}
	for i := range m.Channels {
		n := strconv.Itoa(i + 1)
		g := "Channel " + n
		s.Params = append(s.Params,
			params.K(g, "level"+n, 0, 2, m.Level[i], params.Label("level")),
			params.K(g, "pan"+n, -1, 1, m.Pan[i], params.Label("pan")))
		for b := range sends {
			s.Params = append(s.Params, params.K(g, "send"+busName(b)+n, 0, 1, m.Sends[b][i], params.Label("send "+busName(b))))
		}
		s.Params = append(s.Params, params.S(g, "mute"+n, m.Mute[i], params.Label("mute")))
	}
	for b := range sends {
		s.Params = append(s.Params, params.K("Main", "return"+busName(b), 0, 2, m.Return[b], params.Label("return "+busName(b))))
	}
	s.Params = append(s.Params, params.K("Main", "level", 0, 2, m.Master, params.Label("level")))
	return s
}

// mixerPorts are the ports of a mixer of channels
func mixerPorts(channels int) (ins, outs []Port) {
	for i := range channels {
		ins = append(ins, Port{Name: strconv.Itoa(i + 1), Type: Audio, Index: i})
	}
	outs = []Port{{Name: "main", Type: Audio, Index: 0}}
	for b := range sends {
		ins = append(ins, Port{Name: "return " + busName(b), Type: Audio, Index: channels + b})
		outs = append(outs, Port{Name: "send " + busName(b), Type: Audio, Index: 1 + b})
	}
	return ins, outs
}

// Sources are the kinds of module the patcher has of its own: an LFO, a random source, a mixer
func Sources() []Kind {
	return []Kind{
		{Name: "lfo", Group: "Modulation", Outs: []Port{{Name: "out", Type: Mod}}, Make: func(string) (*Made, error) {
			l := NewLFO(1)
			return &Made{Module: l, Describe: func(string) params.Set { return l.Describe() },
				Mod: func(int) func() float64 { return l.Value }}, nil
		}},
		{Name: "random", Group: "Modulation", Outs: []Port{{Name: "out", Type: Mod}}, Make: func(string) (*Made, error) {
			r := NewRandom(2)
			return &Made{Module: r, Describe: func(string) params.Set { return r.Describe() },
				Mod: func(int) func() float64 { return r.Value }}, nil
		}},
		{Name: "mixer", Group: "Mix", Presets: []string{"8 channels", "4 channels", "16 channels"},
			Make: func(preset string) (*Made, error) {
				channels, _ := strconv.Atoi(strings.Fields(preset + " 8")[0])
				m := NewMixer(channels)
				ins, outs := mixerPorts(m.Channels)
				return &Made{Module: m, Describe: func(string) params.Set { return m.Describe() }, Ins: ins, Outs: outs}, nil
			}},
	}
}

// atomicFloat is a float64 the audio thread writes and anything reads
type atomicFloat struct{ bits atomic.Uint64 }

func (a *atomicFloat) Store(v float64) { a.bits.Store(math.Float64bits(v)) }
func (a *atomicFloat) Load() float64   { return math.Float64frombits(a.bits.Load()) }
