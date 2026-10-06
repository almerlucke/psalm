// Package pingpong is a stereo delay whose echoes bounce between left and right. The echo time is
// set in beats and follows the tempo (psalm.BPM), also when it changes while playing
package pingpong

import (
	"math"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/delay"
)

// Config describes a ping pong delay
type Config struct {
	Time       float64 // echo time in beats, follows the tempo (0.75 is a dotted 8th)
	TimeMs     float64 // echo time in milliseconds instead, used when above 0 (doesn't follow the tempo)
	Feedback   float64 // 0-1, every echo is this much quieter than the one before (at most 0.98)
	Mix        float64 // 0 dry, 1 wet
	Stereo     float64 // 0-1: 0 bounces every echo to the other side (ping pong), 1 is a plain stereo delay
	HighCut    float64 // Hz, lowpass on every repeat, the echoes get darker (0 is off)
	LowCut     float64 // Hz, highpass on every repeat, the echoes get thinner (0 is off)
	MaxSeconds float64 // longest echo time (the buffer), longer times are clamped
}

// DefaultConfig is a dotted 8th ping pong with repeats that slowly darken
func DefaultConfig() Config {
	return Config{
		Time:       0.75,
		Feedback:   0.35,
		Mix:        0.25,
		HighCut:    6000,
		LowCut:     120,
		MaxSeconds: 4,
	}
}

// maxFeedback keeps the echoes decaying
const maxFeedback = 0.98

// glideMs is how fast the echo time follows a change (of the time or the tempo), like tape
const glideMs = 60.0

// PingPong is the delay module: input 0 is stereo audio. Value inputs: 0 time (beats), 1 feedback,
// 2 mix
type PingPong struct {
	*psalm.Module
	cfg     Config
	left    *delay.Delay
	right   *delay.Delay
	samples float64 // current echo time in samples, glides to the target
	lpL     float64 // filter states
	lpR     float64
	hpL     float64
	hpR     float64
}

// New creates a ping pong delay
func New(cfg Config) *PingPong {
	if cfg.MaxSeconds <= 0 {
		cfg.MaxSeconds = 4
	}
	sr := psalm.SampleRate()
	length := int(math.Ceil(cfg.MaxSeconds*sr)) + 2
	pp := &PingPong{
		Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}),
		cfg:    cfg,
		left:   delay.New(length),
		right:  delay.New(length),
	}
	pp.samples = pp.target()
	pp.SetSelf(pp)
	return pp
}

// Settings returns the settings
func (pp *PingPong) Settings() Config {
	return pp.cfg
}

// target returns the echo time in samples for the time setting and the current tempo
func (pp *PingPong) target() float64 {
	sr := pp.Config().SampleRate
	var seconds float64
	if pp.cfg.TimeMs > 0 {
		seconds = pp.cfg.TimeMs * 0.001
	} else {
		seconds = pp.cfg.Time * 60.0 / pp.BPM()
	}
	return math.Max(1, math.Min(seconds*sr, float64(pp.left.Length-2)))
}

// SetTime sets the echo time in beats (and leaves milliseconds)
func (pp *PingPong) SetTime(beats float64) {
	pp.cfg.Time = beats
	pp.cfg.TimeMs = 0
}

// SetTimeMs sets the echo time in milliseconds, which doesn't follow the tempo
func (pp *PingPong) SetTimeMs(ms float64) {
	pp.cfg.TimeMs = ms
}

func (pp *PingPong) SetFeedback(fb float64) { pp.cfg.Feedback = fb }
func (pp *PingPong) SetMix(mix float64)     { pp.cfg.Mix = mix }
func (pp *PingPong) SetStereo(s float64)    { pp.cfg.Stereo = s }

func (pp *PingPong) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		pp.ReceiveMessage(msg)
		return
	}
	v := psalm.FltVal[float64](value)
	switch index {
	case 0:
		pp.SetTime(v)
	case 1:
		pp.SetFeedback(v)
	case 2:
		pp.SetMix(v)
	}
}

func (pp *PingPong) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("time", pp.SetTime)
	msg.IfFltVal("timeMs", pp.SetTimeMs)
	msg.IfFltVal("feedback", pp.SetFeedback)
	msg.IfFltVal("mix", pp.SetMix)
	msg.IfFltVal("stereo", pp.SetStereo)
	msg.IfFltVal("highCut", func(f float64) { pp.cfg.HighCut = f })
	msg.IfFltVal("lowCut", func(f float64) { pp.cfg.LowCut = f })
}

// onePole returns the coefficient of a one pole lowpass at fc
func onePole(fc float64, sr float64) float64 {
	return 1.0 - math.Exp(-2.0*math.Pi*fc/sr)
}

func (pp *PingPong) Synthesize() {
	var (
		cfg      = &pp.cfg
		sr       = pp.Config().SampleRate
		inLeft   = pp.Input(0).Channels[0]
		inRight  = pp.Input(0).Channels[1]
		outLeft  = pp.Output(0).Channels[0]
		outRight = pp.Output(0).Channels[1]
		target   = pp.target()
		glide    = 1.0 - math.Exp(-1.0/(glideMs*0.001*sr))
		fb       = math.Max(0, math.Min(cfg.Feedback, maxFeedback))
		w        = 1 - math.Max(0, math.Min(cfg.Stereo, 1)) // how much the echoes bounce
		dry      = 1.0 - cfg.Mix
		wet      = cfg.Mix
		lpCoef   = 1.0
		hpCoef   = 0.0
	)
	if cfg.HighCut > 0 {
		lpCoef = onePole(math.Min(cfg.HighCut, sr*0.45), sr)
	}
	if cfg.LowCut > 0 {
		hpCoef = onePole(cfg.LowCut, sr)
	}

	// the tone of a repeat: a one pole lowpass and highpass
	tone := func(x float64, lp *float64, hp *float64) float64 {
		*lp += lpCoef * (x - *lp)
		x = *lp
		if hpCoef > 0 {
			*hp += hpCoef * (x - *hp)
			x -= *hp
		}
		return x
	}

	for i := range len(outLeft) {
		pp.samples += (target - pp.samples) * glide

		l := pp.left.ReadLinear(pp.samples)
		r := pp.right.ReadLinear(pp.samples)

		il := float64(inLeft[i])
		ir := float64(inRight[i])

		// the input: in a ping pong everything enters on the left and bounces from there, in a plain
		// stereo delay every side keeps its own input
		inL := (1-w/2)*il + (w/2)*ir
		inR := (1 - w) * ir

		// every repeat crosses to the other side (ping pong) or stays (stereo), losing fb
		writeL := inL + fb*(w*r+(1-w)*l)
		writeR := inR + fb*(w*l+(1-w)*r)
		pp.left.Write(tone(writeL, &pp.lpL, &pp.hpL))
		pp.right.Write(tone(writeR, &pp.lpR, &pp.hpR))

		outLeft[i] = float32(il*dry + l*wet)
		outRight[i] = float32(ir*dry + r*wet)
	}
}

// Tail lets the delay rest while silent: the echoes are spaced by the echo time, silent in
// between
func (pp *PingPong) Tail() float64 {
	return pp.target()/pp.Config().SampleRate + 0.05
}
