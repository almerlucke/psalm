// Package transient is a transient shaper: it finds the start of every hit (where a fast follower
// of the level runs ahead of a slow one) and its decay (where it falls behind), and turns them up
// or down: a snappier or a softer drum, more or less room in its ring, whatever its level
package transient

import (
	"math"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/mix"
)

// Config describes a transient shaper
type Config struct {
	Attack  float64 // dB the start of a hit is turned up (negative: down), -24 to 24
	Sustain float64 // dB the decay is turned up (negative: down), -24 to 24
	Output  float64 // dB after it
	Mix     float64 // 0 dry, 1 shaped
}

// DefaultConfig makes drums snappier
func DefaultConfig() Config { return Config{Attack: 6, Sustain: -3, Mix: 1} }

// Shaper is the module: input 0 is stereo audio. Value inputs: 1 attack, 2 sustain
type Shaper struct {
	*psalm.Module
	cfg        Config
	fast, slow float64 // the followers of the level
	gain       float64 // dB now
	mix        mix.Smoother
}

// New creates a transient shaper
func New(cfg Config) *Shaper {
	s := &Shaper{
		Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}),
		cfg:    cfg,
		mix:    mix.NewSmoother(clamp(cfg.Mix, 0, 1), mix.SmoothMs, psalm.SampleRate()),
	}
	s.SetSelf(s)
	return s
}

// Settings returns the settings
func (s *Shaper) Settings() Config { return s.cfg }

func (s *Shaper) SetAttack(db float64)  { s.cfg.Attack = clamp(db, -24, 24) }
func (s *Shaper) SetSustain(db float64) { s.cfg.Sustain = clamp(db, -24, 24) }

func clamp(x, lo, hi float64) float64 { return math.Max(lo, math.Min(x, hi)) }

func (s *Shaper) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		s.ReceiveMessage(msg)
		return
	}
	v := psalm.FltVal[float64](value)
	switch index {
	case 1:
		s.SetAttack(v)
	case 2:
		s.SetSustain(v)
	}
}

func (s *Shaper) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("attack", s.SetAttack)
	msg.IfFltVal("sustain", s.SetSustain)
	msg.IfFltVal("output", func(v float64) { s.cfg.Output = v })
	msg.IfFltVal("mix", func(v float64) { s.cfg.Mix = v })
}

func coef(ms, sr float64) float64 { return 1 - math.Exp(-1/(ms*0.001*sr)) }

// follow moves a follower towards a level, up over one time and down over another
func follow(v *float64, x, up, down float64) {
	if x > *v {
		*v += (x - *v) * up
	} else {
		*v += (x - *v) * down
	}
}

func (s *Shaper) Synthesize() {
	var (
		cfg    = s.cfg
		sr     = s.Config().SampleRate
		in     = s.Input(0).Channels
		out    = s.Output(0).Channels
		fastUp = coef(0.5, sr)
		fastDn = coef(30, sr)
		slowUp = coef(25, sr)
		slowDn = coef(600, sr)
		smooth = coef(2, sr)
		output = math.Pow(10, cfg.Output/20)
		mixTo  = clamp(cfg.Mix, 0, 1)
	)
	for i := range out[0] {
		l, r := float64(in[0][i]), float64(in[len(in)-1][i])
		x := math.Max(math.Abs(l), math.Abs(r))
		follow(&s.fast, x, fastUp, fastDn)
		follow(&s.slow, x, slowUp, slowDn)
		target := 0.0
		if s.fast > 1e-6 && s.slow > 1e-6 {
			// how far ahead (a start: 12 dB is all the way) or behind (a decay: 4 dB, a decay lags
			// less) the fast follower is
			d := 20 * math.Log10(s.fast/s.slow)
			target = clamp(d/12, 0, 1)*cfg.Attack + clamp(-d/4, 0, 1)*cfg.Sustain
		}
		s.gain += (target - s.gain) * smooth
		g := math.Pow(10, s.gain/20) * output
		m := s.mix.Next(mixTo)
		for c := range out {
			v := float64(in[c%len(in)][i])
			out[c][i] = float32(v*(1-m) + v*g*m)
		}
	}
}

// Tail lets it rest while silent
func (s *Shaper) Tail() float64 { return 0 }
