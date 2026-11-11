// Package rotary is a rotary speaker, the sound of an organ through a Leslie: a horn for the highs
// and a drum for the lows turn, each sound coming closer and going away (a Doppler vibrato and a
// rising and falling level), picked up by two microphones on both sides (stereo). Slow (chorale)
// and fast (tremolo) speeds, and the horn and the heavier drum speed up and slow down at their own
// pace when switched
package rotary

import (
	"math"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/delay"
	"github.com/almerlucke/psalm/blocks/filters/biquad"
	"github.com/almerlucke/psalm/blocks/mix"
)

// Config describes a rotary speaker
type Config struct {
	Fast      bool    // tremolo speed; slow (chorale) when false
	Crossover float64 // Hz between the drum and the horn (0 is 800)
	Drive     float64 // dB into the tube amplifier before it (0 is clean)
	Horn      float64 // 0-1, the level of the horn against the drum (0 is 0.5: as loud)
	Spread    float64 // 0-1, the stereo width of the microphones (0 is 1)
	Mix       float64 // 0 dry, 1 only the speaker
}

// DefaultConfig is a rotary speaker turning slowly, a little driven
func DefaultConfig() Config {
	return Config{Crossover: 800, Drive: 3, Horn: 0.5, Spread: 1, Mix: 1}
}

// rotor is the horn or the drum
type rotor struct {
	slow, fast float64 // turns per second
	up, down   float64 // seconds to reach the new speed (speeding up, slowing down)
	doppler    float64 // ms the sound moves to and fro
	am         float64 // 0-1, how much the level rises and falls
	rate       float64 // the speed now
	phase      float64 // 0-1
	line       *delay.Delay
}

// Rotary is the module: input 0 is stereo audio (played mono into the speaker, as an organ is).
// Value inputs: 1 fast (above 0.5), 2 mix
type Rotary struct {
	*psalm.Module
	cfg        Config
	horn, drum rotor
	lo, hi     *biquad.Biquad
	lo2, hi2   *biquad.Biquad // the crossover is fourth order: two of each
	mix        mix.Smoother
}

// New creates a rotary speaker
func New(cfg Config) *Rotary {
	sr := psalm.SampleRate()
	if cfg.Crossover <= 0 {
		cfg.Crossover = 800
	}
	length := int(0.01*sr) + 4
	r := &Rotary{
		Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}),
		cfg:    cfg,
		// the speeds of a Leslie 122: the horn faster than the drum, the drum slower to change
		horn: rotor{slow: 0.83, fast: 6.7, up: 0.7, down: 0.9, doppler: 0.3, am: 0.45, line: delay.New(length)},
		drum: rotor{slow: 0.67, fast: 5.8, up: 4.5, down: 5.5, doppler: 0.15, am: 0.2, line: delay.New(length)},
		lo:   biquad.New(biquad.LowPass, cfg.Crossover, 0.707, 0, sr),
		lo2:  biquad.New(biquad.LowPass, cfg.Crossover, 0.707, 0, sr),
		hi:   biquad.New(biquad.HighPass, cfg.Crossover, 0.707, 0, sr),
		hi2:  biquad.New(biquad.HighPass, cfg.Crossover, 0.707, 0, sr),
		mix:  mix.NewSmoother(clamp(cfg.Mix, 0, 1), mix.SmoothMs, sr),
	}
	r.horn.rate, r.drum.rate = r.horn.target(cfg.Fast), r.drum.target(cfg.Fast)
	r.drum.phase = 0.37 // the rotors don't turn in step
	r.SetSelf(r)
	return r
}

func (ro *rotor) target(fast bool) float64 {
	if fast {
		return ro.fast
	}
	return ro.slow
}

// Settings returns the settings
func (r *Rotary) Settings() Config { return r.cfg }

// SetFast switches between the fast (tremolo) and the slow (chorale) speed: the rotors get there at
// their own pace
func (r *Rotary) SetFast(fast bool) { r.cfg.Fast = fast }

// Fast tells whether it is switched to fast
func (r *Rotary) Fast() bool { return r.cfg.Fast }

// Speeds returns the turns per second of the horn and the drum now
func (r *Rotary) Speeds() (horn, drum float64) { return r.horn.rate, r.drum.rate }

func (r *Rotary) SetMix(m float64) { r.cfg.Mix = m }

func clamp(x, lo, hi float64) float64 { return max(lo, min(x, hi)) }

func (r *Rotary) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		r.ReceiveMessage(msg)
		return
	}
	v := psalm.FltVal[float64](value)
	switch index {
	case 1:
		r.SetFast(v > 0.5)
	case 2:
		r.SetMix(v)
	}
}

func (r *Rotary) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("fast", func(v float64) { r.SetFast(v > 0.5) }) // a number or true/false
	msg.IfFltVal("mix", r.SetMix)
	msg.IfFltVal("drive", func(db float64) { r.cfg.Drive = db })
	msg.IfFltVal("horn", func(h float64) { r.cfg.Horn = h })
	msg.IfFltVal("spread", func(s float64) { r.cfg.Spread = s })
	msg.IfFltVal("crossover", r.SetCrossover)
}

// SetCrossover moves the split between the drum and the horn (Hz, 150 to 4000)
func (r *Rotary) SetCrossover(hz float64) {
	r.cfg.Crossover = max(150, min(hz, 4000))
	for _, f := range []*biquad.Biquad{r.lo, r.lo2, r.hi, r.hi2} {
		f.SetFrequency(r.cfg.Crossover)
	}
}

// turn moves a rotor a sample on: its speed towards the target, its phase by its speed
func (ro *rotor) turn(fast bool, sr float64) {
	target := ro.target(fast)
	t := ro.down
	if target > ro.rate {
		t = ro.up
	}
	ro.rate += (target - ro.rate) * (1 - math.Exp(-1/(t/3*sr))) // about there in t seconds
	ro.phase += ro.rate / sr
	ro.phase -= math.Floor(ro.phase)
}

// hear returns what a microphone at an angle (0-1 of a turn) hears of a rotor: the sound delayed
// as it comes and goes, louder when it faces the microphone
func (ro *rotor) hear(angle, sr float64) float64 {
	a := 2 * math.Pi * (ro.phase + angle)
	d := 1 + ro.doppler*0.001*sr*(1+math.Sin(a))
	level := 1 - ro.am*(0.5-0.5*math.Cos(a))
	return ro.line.ReadHermite(d) * level
}

func (r *Rotary) Synthesize() {
	var (
		cfg    = r.cfg
		sr     = r.Config().SampleRate
		in     = r.Input(0).Channels
		out    = r.Output(0).Channels
		drive  = math.Pow(10, cfg.Drive/20)
		mixTo  = clamp(cfg.Mix, 0, 1)
		horn   = cfg.Horn
		spread = cfg.Spread
	)
	if horn <= 0 {
		horn = 0.5
	}
	if spread <= 0 {
		spread = 1
	}
	hornGain, drumGain := 2*clamp(horn, 0, 1), 2*(1-clamp(horn, 0, 1))
	for i := range out[0] {
		l, rr := float64(in[0][i]), float64(in[len(in)-1][i])
		x := (l + rr) / 2
		if cfg.Drive > 0 { // the tube amplifier
			x = math.Tanh(x*drive) / math.Tanh(drive)
		}
		r.drum.line.Write(r.lo2.Process(r.lo.Process(x)))
		r.horn.line.Write(r.hi2.Process(r.hi.Process(x)))
		r.horn.turn(cfg.Fast, sr)
		r.drum.turn(cfg.Fast, sr)
		// two microphones a quarter turn apart, the spread brings them together
		angle := 0.25 * spread
		left := hornGain*r.horn.hear(0, sr) + drumGain*r.drum.hear(0, sr)
		right := hornGain*r.horn.hear(angle, sr) + drumGain*r.drum.hear(angle, sr)
		m := r.mix.Next(mixTo)
		out[0][i] = float32(l*(1-m) + left*m)
		out[len(out)-1][i] = float32(rr*(1-m) + right*m)
	}
}

// Tail lets it rest while silent
func (r *Rotary) Tail() float64 { return 0.02 }
