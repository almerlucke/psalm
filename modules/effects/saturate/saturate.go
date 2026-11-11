// Package saturate is saturation and distortion: the sound is driven into a curve that rounds its
// peaks off, from the warmth of tape and tubes to the grit of a fuzz and the hollow buzz of wave
// folding. It runs four times oversampled, so the harmonics it makes don't fold back as aliasing
package saturate

import (
	"math"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/mix"
	"github.com/almerlucke/psalm/blocks/oversample"
)

// Shape is the curve the sound is driven into
type Shape int

const (
	Soft Shape = iota // a smooth curve (tanh): tape, warm, odd harmonics
	Tube              // a lopsided curve: tubes, even harmonics too
	Hard              // clipped flat: fuzz, harsh
	Fold              // folded back over itself: a buzzy, hollow sound that keeps changing with the drive
)

// Config describes a saturation
type Config struct {
	Shape  Shape
	Drive  float64 // dB the sound is driven into the curve (0 just touches it, 24 is heavy)
	Bias   float64 // -1 to 1, shifts the curve (Tube uses 0.25 when 0): more even harmonics
	Tone   float64 // Hz, a lowpass after the curve, takes the fizz off (0 is off)
	Output float64 // dB after it
	Mix    float64 // 0 dry, 1 saturated
}

// DefaultConfig is a warm saturation
func DefaultConfig() Config {
	return Config{Shape: Soft, Drive: 9, Tone: 9000, Output: -4, Mix: 1}
}

// Saturate is the module: input 0 is stereo audio. Value inputs: 1 drive, 2 mix
type Saturate struct {
	*psalm.Module
	cfg   Config
	os    [2]*oversample.Oversampler
	lp    [2]float64 // tone filter states
	dc    [2]float64 // DC blocker states (a lopsided curve makes DC)
	dcIn  [2]float64
	dry   [2][]float64 // the dry sound delayed by the latency, to mix
	dpos  int
	drive mix.Smoother
	mix   mix.Smoother
}

// New creates a saturation
func New(cfg Config) *Saturate {
	sr := psalm.SampleRate()
	s := &Saturate{
		Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}),
		cfg:    cfg,
		drive:  mix.NewSmoother(cfg.Drive, mix.SmoothMs, sr),
		mix:    mix.NewSmoother(clamp(cfg.Mix, 0, 1), mix.SmoothMs, sr),
	}
	for c := range s.os {
		s.os[c] = oversample.New(4, oversample.TapsPerPhase)
		s.dry[c] = make([]float64, s.os[c].Latency()+1)
	}
	s.SetSelf(s)
	return s
}

// Settings returns the settings
func (s *Saturate) Settings() Config { return s.cfg }

func (s *Saturate) SetDrive(db float64)  { s.cfg.Drive = db }
func (s *Saturate) SetMix(m float64)     { s.cfg.Mix = m }
func (s *Saturate) SetShape(sh Shape)    { s.cfg.Shape = sh }
func (s *Saturate) SetTone(hz float64)   { s.cfg.Tone = hz }
func (s *Saturate) SetOutput(db float64) { s.cfg.Output = db }
func (s *Saturate) SetBias(b float64)    { s.cfg.Bias = clamp(b, -1, 1) }

// SignalLatency is the delay of the oversampling, a few samples (the dry sound in the mix is delayed
// as much, so they line up)
func (s *Saturate) SignalLatency() int { return s.os[0].Latency() }

// Tail lets it rest while silent (the filters ring for a moment)
func (s *Saturate) Tail() float64 { return 0.01 }

func clamp(x, lo, hi float64) float64 { return max(lo, min(x, hi)) }
func dbToGain(db float64) float64     { return math.Pow(10, db/20) }

func (s *Saturate) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		s.ReceiveMessage(msg)
		return
	}
	v := psalm.FltVal[float64](value)
	switch index {
	case 1:
		s.SetDrive(v)
	case 2:
		s.SetMix(v)
	}
}

func (s *Saturate) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("drive", s.SetDrive)
	msg.IfFltVal("mix", s.SetMix)
	msg.IfFltVal("tone", s.SetTone)
	msg.IfFltVal("output", s.SetOutput)
	msg.IfFltVal("bias", s.SetBias)
	msg.IfIntVal("shape", func(v int) { s.SetShape(Shape(v)) })
}

// curve is the shape at x (the input times the drive)
func curve(shape Shape, x, bias float64) float64 {
	switch shape {
	case Tube:
		if bias == 0 {
			bias = 0.25
		}
		return math.Tanh(x+bias) - math.Tanh(bias)
	case Hard:
		return clamp(x+bias, -1, 1)
	case Fold:
		// folds back at ±1, a triangle over the input
		v := x + bias
		v = math.Mod(v+1, 4)
		if v < 0 {
			v += 4
		}
		if v > 2 {
			v = 4 - v
		}
		return v - 1
	default:
		return math.Tanh(x + bias)
	}
}

func (s *Saturate) Synthesize() {
	var (
		cfg    = s.cfg
		sr     = s.Config().SampleRate
		in     = s.Input(0).Channels
		out    = s.Output(0).Channels
		output = dbToGain(cfg.Output)
		mixTo  = clamp(cfg.Mix, 0, 1)
		lp     = 1.0
		dcR    = 1 - 2*math.Pi*10/sr // a DC blocker at 10 Hz
	)
	if cfg.Tone > 0 {
		lp = 1 - math.Exp(-2*math.Pi*min(cfg.Tone, sr*0.45)/sr)
	}
	for i := range out[0] {
		drive := dbToGain(s.drive.Next(cfg.Drive))
		m := s.mix.Next(mixTo)
		shape := func(x float64) float64 { return curve(cfg.Shape, x*drive, cfg.Bias) }
		for c := range out {
			x := float64(in[c%len(in)][i])
			y := s.os[c].Process(x, shape)
			// the DC a lopsided curve makes, gone
			dc := y - s.dcIn[c] + dcR*s.dc[c]
			s.dcIn[c], s.dc[c] = y, dc
			s.lp[c] += lp * (dc - s.lp[c])
			s.dry[c][s.dpos] = x
			dry := s.dry[c][(s.dpos+1)%len(s.dry[c])] // latency samples ago
			out[c][i] = float32(dry*(1-m) + s.lp[c]*output*m)
		}
		s.dpos = (s.dpos + 1) % len(s.dry[0])
	}
}
