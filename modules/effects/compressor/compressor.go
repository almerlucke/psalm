// Package compressor is a dynamics compressor: above a threshold it turns the level down by a
// ratio, with a soft knee, attack and release. A sidechain input lets another sound drive it, so a
// kick ducks the bass or a pad (the pumping of dance music), and a mix blends in the dry signal
// (parallel compression)
package compressor

import (
	"math"
	"sync/atomic"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/mix"
)

// Config describes a compressor
type Config struct {
	Threshold  float64 // dB, the level above which it compresses
	Ratio      float64 // 4 is 4:1 (4 dB more in gives 1 dB more out), at least 1; 20 and up is a limiter
	Knee       float64 // dB, the width of the soft knee around the threshold (0 is a hard knee)
	Attack     float64 // ms, how fast it turns down
	Release    float64 // ms, how fast it lets go
	Makeup     float64 // dB of gain after compressing
	AutoMakeup bool    // makeup gain of half the reduction at 0 dB in, instead of Makeup
	Mix        float64 // 0 dry, 1 compressed (parallel compression in between)
	SideCut    float64 // Hz, highpass on the detector (0 is off): the low end doesn't pump it
}

// DefaultConfig is a gentle compressor for a part or a bus
func DefaultConfig() Config {
	return Config{Threshold: -18, Ratio: 3, Knee: 6, Attack: 10, Release: 150, Mix: 1}
}

// Ducker is a sidechain setting: a kick on input 1 pulls the sound down by up to depth dB and lets
// it come back over release ms
func Ducker(depth, release float64) Config {
	return Config{Threshold: -30, Ratio: 1 + depth/6, Knee: 6, Attack: 2, Release: release, Mix: 1}
}

// Compressor is the module: input 0 is the stereo sound, input 1 an optional stereo sidechain (when
// something is connected it drives the compression instead of the sound itself). Value inputs:
// 1 threshold, 2 ratio, 3 mix
type Compressor struct {
	*psalm.Module
	cfg       Config
	gr        float64 // the gain reduction now, dB (0 or less)
	hpL, hpR  float64 // detector highpass states
	env       float64 // the detector's level: a peak that falls over envMs
	mix       mix.Smoother
	reduction atomic.Uint64 // the most gain reduction since it was read, dB (float64 bits, positive)
}

// New creates a compressor
func New(cfg Config) *Compressor {
	if cfg.Ratio < 1 {
		cfg.Ratio = 1
	}
	c := &Compressor{
		Module: psalm.NewModule([]int{psalm.Stereo, psalm.Stereo}, []int{psalm.Stereo}),
		cfg:    cfg,
		mix:    mix.NewSmoother(clamp(cfg.Mix, 0, 1), mix.SmoothMs, psalm.SampleRate()),
	}
	c.SetSelf(c)
	return c
}

// Settings returns the settings
func (c *Compressor) Settings() Config { return c.cfg }

func (c *Compressor) SetThreshold(db float64) { c.cfg.Threshold = db }
func (c *Compressor) SetRatio(r float64)      { c.cfg.Ratio = math.Max(1, r) }
func (c *Compressor) SetKnee(db float64)      { c.cfg.Knee = math.Max(0, db) }
func (c *Compressor) SetAttack(ms float64)    { c.cfg.Attack = math.Max(0, ms) }
func (c *Compressor) SetRelease(ms float64)   { c.cfg.Release = math.Max(0, ms) }
func (c *Compressor) SetMakeup(db float64)    { c.cfg.Makeup = db }
func (c *Compressor) SetMix(m float64)        { c.cfg.Mix = m }

// Reduction returns the most gain reduction (in dB, 0 or more) since it was asked last, for a
// meter; it can be asked from any goroutine
func (c *Compressor) Reduction() float64 {
	return math.Float64frombits(c.reduction.Swap(0))
}

func (c *Compressor) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		c.ReceiveMessage(msg)
		return
	}
	v := psalm.FltVal[float64](value)
	switch index {
	case 1:
		c.SetThreshold(v)
	case 2:
		c.SetRatio(v)
	case 3:
		c.SetMix(v)
	}
}

func (c *Compressor) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("threshold", c.SetThreshold)
	msg.IfFltVal("ratio", c.SetRatio)
	msg.IfFltVal("knee", c.SetKnee)
	msg.IfFltVal("attack", c.SetAttack)
	msg.IfFltVal("release", c.SetRelease)
	msg.IfFltVal("makeup", c.SetMakeup)
	msg.IfFltVal("mix", c.SetMix)
	msg.IfFltVal("sideCut", func(f float64) { c.cfg.SideCut = f })
}

func clamp(x, lo, hi float64) float64 { return math.Max(lo, math.Min(x, hi)) }

// envMs is how fast the detector's level falls: a fast attack doesn't follow the waves of a bass
const envMs = 10.0

// coef returns the one pole coefficient of a time in ms (0 is at once)
func coef(ms, sr float64) float64 {
	if ms <= 0 {
		return 1
	}
	return 1 - math.Exp(-1/(ms*0.001*sr))
}

// curve returns the level out (dB) for a level in (dB): the gain computer with a soft knee
func curve(in, threshold, ratio, knee float64) float64 {
	over := in - threshold
	switch {
	case 2*over < -knee:
		return in
	case knee > 0 && 2*math.Abs(over) <= knee:
		x := over + knee/2
		return in + (1/ratio-1)*x*x/(2*knee)
	default:
		return threshold + over/ratio
	}
}

// makeup returns the makeup gain in dB
func (c *Compressor) makeup() float64 {
	if c.cfg.AutoMakeup {
		return -curve(0, c.cfg.Threshold, c.cfg.Ratio, c.cfg.Knee) / 2
	}
	return c.cfg.Makeup
}

func (c *Compressor) Synthesize() {
	var (
		cfg     = c.cfg
		sr      = c.Config().SampleRate
		in      = c.Input(0).Channels
		side    = in
		out     = c.Output(0).Channels
		attack  = coef(cfg.Attack, sr)
		release = coef(cfg.Release, sr)
		hp      = 0.0
		makeup  = c.makeup()
		mixTo   = clamp(cfg.Mix, 0, 1)
		most    = 0.0
		decay   = 1 - coef(envMs, sr)
	)
	if c.Input(1).IsConnected() {
		side = c.Input(1).Channels
	}
	if cfg.SideCut > 0 {
		hp = 1 - math.Exp(-2*math.Pi*cfg.SideCut/sr)
	}
	for i := range out[0] {
		// the detector: the louder side of the sidechain (stereo linked), highpassed
		sl, sr2 := float64(side[0][i]), float64(side[len(side)-1][i])
		if hp > 0 {
			c.hpL += hp * (sl - c.hpL)
			c.hpR += hp * (sr2 - c.hpR)
			sl, sr2 = sl-c.hpL, sr2-c.hpR
		}
		c.env = math.Max(math.Max(math.Abs(sl), math.Abs(sr2)), c.env*decay) // the level, not every wave
		db := -120.0
		if c.env > 1e-6 {
			db = 20 * math.Log10(c.env)
		}
		target := curve(db, cfg.Threshold, math.Max(1, cfg.Ratio), cfg.Knee) - db // 0 or less
		if target < c.gr {
			c.gr += (target - c.gr) * attack
		} else {
			c.gr += (target - c.gr) * release
		}
		most = math.Max(most, -c.gr)
		g := math.Pow(10, (c.gr+makeup)/20)
		m := c.mix.Next(mixTo) // linear: the compressed and the dry sound are the same sound
		for ch := range out {
			x := float64(in[ch][i])
			out[ch][i] = float32(x*(1-m) + x*g*m)
		}
	}
	for { // the most reduction since it was read
		old := c.reduction.Load()
		if most <= math.Float64frombits(old) || c.reduction.CompareAndSwap(old, math.Float64bits(most)) {
			break
		}
	}
}

// Tail: silent in, silent out (the sidechain alone makes no sound)
func (c *Compressor) Tail() float64 { return 0 }

// TailInputs: only the sound counts, not the sidechain
func (c *Compressor) TailInputs() []int { return []int{0} }
