// Package gate is a noise gate and downward expander: below a threshold it turns the level down,
// by a ratio (an expander) or all the way to the range (a gate), with attack, hold and release. A
// sidechain input lets another sound open it (a gated pad that plays along with the drums)
package gate

import (
	"math"

	"github.com/almerlucke/psalm"
)

// Config describes a gate or expander
type Config struct {
	Threshold float64 // dB, below it the level goes down
	Ratio     float64 // 2 is a 1:2 expander (1 dB under gives 2 dB under); 0 (or 100 and more) a gate
	Range     float64 // dB, the most it turns down (a positive number: 80 is nearly silence; 0 is 80)
	Attack    float64 // ms, how fast it opens
	Hold      float64 // ms, how long it stays open after the level drops
	Release   float64 // ms, how fast it closes
	SideCut   float64 // Hz, highpass on the detector (0 is off)
}

// DefaultConfig is a gate for drums and noisy recordings
func DefaultConfig() Config {
	return Config{Threshold: -45, Range: 80, Attack: 1, Hold: 30, Release: 120}
}

// Expander is a gentle 1:2 expander below a threshold, for a part's quiet tails and noise
func Expander(threshold float64) Config {
	return Config{Threshold: threshold, Ratio: 2, Range: 30, Attack: 2, Hold: 10, Release: 150}
}

// Gate is the module: input 0 is the stereo sound, input 1 an optional stereo sidechain (it opens the
// gate when connected). Value inputs: 1 threshold, 2 range
type Gate struct {
	*psalm.Module
	cfg  Config
	gr   float64 // the gain now, dB (0 open, -range closed)
	held int     // samples it stays open
	env  float64 // the level of the detector: a peak that falls over envMs
	hpL  float64
	hpR  float64
}

// New creates a gate
func New(cfg Config) *Gate {
	g := &Gate{Module: psalm.NewModule([]int{psalm.Stereo, psalm.Stereo}, []int{psalm.Stereo}), cfg: cfg}
	g.gr = -g.rangeDB()
	g.SetSelf(g)
	return g
}

// Settings returns the settings
func (g *Gate) Settings() Config { return g.cfg }

func (g *Gate) SetThreshold(db float64) { g.cfg.Threshold = db }
func (g *Gate) SetRange(db float64)     { g.cfg.Range = math.Abs(db) }
func (g *Gate) SetRatio(r float64)      { g.cfg.Ratio = r }

func (g *Gate) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		g.ReceiveMessage(msg)
		return
	}
	v := psalm.FltVal[float64](value)
	switch index {
	case 1:
		g.SetThreshold(v)
	case 2:
		g.SetRange(v)
	}
}

func (g *Gate) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("threshold", g.SetThreshold)
	msg.IfFltVal("range", g.SetRange)
	msg.IfFltVal("ratio", g.SetRatio)
	msg.IfFltVal("attack", func(ms float64) { g.cfg.Attack = ms })
	msg.IfFltVal("hold", func(ms float64) { g.cfg.Hold = ms })
	msg.IfFltVal("release", func(ms float64) { g.cfg.Release = ms })
	msg.IfFltVal("sideCut", func(f float64) { g.cfg.SideCut = f })
}

func (g *Gate) rangeDB() float64 {
	if g.cfg.Range <= 0 {
		return 80
	}
	return g.cfg.Range
}

// envMs is how fast the detector's level falls: slow enough not to follow the waves of a low note
const envMs = 10.0

func coef(ms, sr float64) float64 {
	if ms <= 0 {
		return 1
	}
	return 1 - math.Exp(-1/(ms*0.001*sr))
}

// target returns the gain (dB) for a level in dB
func (g *Gate) target(db float64) float64 {
	under := g.cfg.Threshold - db
	if under <= 0 {
		return 0
	}
	r := g.rangeDB()
	if g.cfg.Ratio <= 0 || g.cfg.Ratio >= 100 {
		return -r // a gate: closed
	}
	return -math.Min(under*(g.cfg.Ratio-1), r)
}

func (g *Gate) Synthesize() {
	var (
		sr      = g.Config().SampleRate
		in      = g.Input(0).Channels
		side    = in
		out     = g.Output(0).Channels
		attack  = coef(g.cfg.Attack, sr)
		release = coef(g.cfg.Release, sr)
		hold    = int(g.cfg.Hold * 0.001 * sr)
		hp      = 0.0
		decay   = 1 - coef(envMs, sr)
	)
	if g.Input(1).IsConnected() {
		side = g.Input(1).Channels
	}
	if g.cfg.SideCut > 0 {
		hp = 1 - math.Exp(-2*math.Pi*g.cfg.SideCut/sr)
	}
	for i := range out[0] {
		l, r := float64(side[0][i]), float64(side[len(side)-1][i])
		if hp > 0 {
			g.hpL += hp * (l - g.hpL)
			g.hpR += hp * (r - g.hpR)
			l, r = l-g.hpL, r-g.hpR
		}
		g.env = math.Max(math.Max(math.Abs(l), math.Abs(r)), g.env*decay) // the level, not every wave
		db := -120.0
		if g.env > 1e-6 {
			db = 20 * math.Log10(g.env)
		}
		t := g.target(db)
		switch {
		case t > g.gr: // opening
			g.gr += (t - g.gr) * attack
			if t == 0 {
				g.held = hold
			}
		case t == 0:
			g.held = hold
		case g.held > 0: // holding open
			g.held--
		default: // closing
			g.gr += (t - g.gr) * release
		}
		gain := float32(math.Pow(10, g.gr/20))
		for ch := range out {
			out[ch][i] = in[ch][i] * gain
		}
	}
}

// Tail: silent in, silent out
func (g *Gate) Tail() float64 { return 0 }

// TailInputs: only the sound counts, not the sidechain
func (g *Gate) TailInputs() []int { return []int{0} }
