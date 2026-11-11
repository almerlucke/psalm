// Package tape is a tape echo: the echoes of a tape loop, every repeat a little darker and warmer
// (filtered and saturated on the way round) and wavering in pitch with the wow and flutter of the
// tape. The echo time is set in beats and follows the tempo. Feedback can go above 1: the echoes
// build up into the singing self oscillation of a dub delay, the saturation keeps it in bounds
package tape

import (
	"math"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/delay"
	"github.com/almerlucke/psalm/blocks/mix"
)

// Config describes a tape echo
type Config struct {
	Time       float64 // echo time in beats, follows the tempo
	TimeMs     float64 // echo time in milliseconds instead, used when above 0
	Feedback   float64 // 0-1.2: how much of every echo comes round again (above 1 it builds up)
	Wow        float64 // 0-1, the slow wavering of the tape (0.3 is a used machine)
	Flutter    float64 // 0-1, the fast trembling of the tape
	Drive      float64 // dB of saturation on the way round (0 is a gentle 3)
	HighCut    float64 // Hz, the repeats get darker (0 is off)
	LowCut     float64 // Hz, the repeats get thinner (0 is off)
	Mix        float64 // 0 dry, 1 wet (an equal power crossfade)
	MaxSeconds float64 // the longest echo time, the length of the loop (0 is 4)
}

// DefaultConfig is a warm echo in dotted 8ths
func DefaultConfig() Config {
	return Config{Time: 0.75, Feedback: 0.45, Wow: 0.3, Flutter: 0.2, HighCut: 4500, LowCut: 150, Mix: 0.25, MaxSeconds: 4}
}

// Tape is the module: input 0 is stereo audio. Value inputs: 1 time (beats), 2 feedback, 3 mix
type Tape struct {
	*psalm.Module
	cfg      Config
	lines    [2]*delay.Delay
	samples  float64 // the echo time now, gliding to the setting (as a tape speeds up and down)
	wow      float64 // phases
	flutter  float64
	lp, hp   [2]float64
	mix, fb  mix.Smoother
	maxDelay float64
}

// New creates a tape echo
func New(cfg Config) *Tape {
	if cfg.MaxSeconds <= 0 {
		cfg.MaxSeconds = 4
	}
	sr := psalm.SampleRate()
	length := int(math.Ceil(cfg.MaxSeconds*sr)) + int(0.02*sr)
	t := &Tape{
		Module:   psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}),
		cfg:      cfg,
		mix:      mix.NewSmoother(clamp(cfg.Mix, 0, 1), mix.SmoothMs, sr),
		fb:       mix.NewSmoother(clamp(cfg.Feedback, 0, maxFeedback), mix.SmoothMs, sr),
		maxDelay: float64(length - 4),
	}
	for c := range t.lines {
		t.lines[c] = delay.New(length)
	}
	t.samples = t.target()
	t.SetSelf(t)
	return t
}

// maxFeedback is as far as the echoes may build up
const maxFeedback = 1.2

// Settings returns the settings
func (t *Tape) Settings() Config { return t.cfg }

func (t *Tape) SetTime(beats float64) { t.cfg.Time, t.cfg.TimeMs = beats, 0 }
func (t *Tape) SetTimeMs(ms float64)  { t.cfg.TimeMs = ms }
func (t *Tape) SetFeedback(f float64) { t.cfg.Feedback = f }
func (t *Tape) SetMix(m float64)      { t.cfg.Mix = m }

func clamp(x, lo, hi float64) float64 { return max(lo, min(x, hi)) }

func (t *Tape) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		t.ReceiveMessage(msg)
		return
	}
	v := psalm.FltVal[float64](value)
	switch index {
	case 1:
		t.SetTime(v)
	case 2:
		t.SetFeedback(v)
	case 3:
		t.SetMix(v)
	}
}

func (t *Tape) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("time", t.SetTime)
	msg.IfFltVal("timeMs", t.SetTimeMs)
	msg.IfFltVal("feedback", t.SetFeedback)
	msg.IfFltVal("mix", t.SetMix)
	msg.IfFltVal("wow", func(v float64) { t.cfg.Wow = v })
	msg.IfFltVal("flutter", func(v float64) { t.cfg.Flutter = v })
	msg.IfFltVal("drive", func(v float64) { t.cfg.Drive = v })
	msg.IfFltVal("highCut", func(v float64) { t.cfg.HighCut = v })
	msg.IfFltVal("lowCut", func(v float64) { t.cfg.LowCut = v })
}

// target returns the echo time in samples
func (t *Tape) target() float64 {
	sr := t.Config().SampleRate
	seconds := t.cfg.Time * 60 / t.BPM()
	if t.cfg.TimeMs > 0 {
		seconds = t.cfg.TimeMs * 0.001
	}
	return clamp(seconds*sr, 2, t.maxDelay-0.01*sr)
}

// EchoSamples returns the echo time now, in samples (it wavers with the wow and flutter)
func (t *Tape) EchoSamples() float64 { return t.samples }

func (t *Tape) Synthesize() {
	var (
		cfg    = t.cfg
		sr     = t.Config().SampleRate
		in     = t.Input(0).Channels
		out    = t.Output(0).Channels
		target = t.target()
		glide  = 1 - math.Exp(-1/(0.15*sr)) // the tape motor finds a new speed in a moment
		drive  = math.Pow(10, cfg.Drive/20)
		lp     = 1.0
		hp     = 0.0
		fbTo   = clamp(cfg.Feedback, 0, maxFeedback)
		mixTo  = clamp(cfg.Mix, 0, 1)
		// the wavering: wow about half a second round, flutter about 7 times a second
		wowDepth     = clamp(cfg.Wow, 0, 1) * 0.004 * sr // up to 4 ms
		flutterDepth = clamp(cfg.Flutter, 0, 1) * 0.0004 * sr
		wowInc       = 0.55 / sr
		flutterInc   = 7.1 / sr
	)
	if cfg.Drive == 0 {
		drive = math.Pow(10, 3.0/20)
	}
	if cfg.HighCut > 0 {
		lp = 1 - math.Exp(-2*math.Pi*min(cfg.HighCut, sr*0.45)/sr)
	}
	if cfg.LowCut > 0 {
		hp = 1 - math.Exp(-2*math.Pi*cfg.LowCut/sr)
	}
	for i := range out[0] {
		t.samples += (target - t.samples) * glide
		t.wow += wowInc
		t.flutter += flutterInc
		t.wow -= math.Floor(t.wow)
		t.flutter -= math.Floor(t.flutter)
		waver := wowDepth*(0.5+0.5*math.Sin(2*math.Pi*t.wow)) + flutterDepth*(0.5+0.5*math.Sin(2*math.Pi*t.flutter))
		fb := t.fb.Next(fbTo)
		dry, wet := mix.EqualPower(t.mix.Next(mixTo))
		for c := range out {
			x := float64(in[c%len(in)][i])
			// the two heads a few samples apart, the stereo of a real machine
			echo := t.lines[c].ReadHermite(t.samples + waver + float64(c)*0.3)
			// the way round: darker, thinner, saturated
			t.lp[c] += lp * (echo - t.lp[c])
			e := t.lp[c]
			if hp > 0 {
				t.hp[c] += hp * (e - t.hp[c])
				e -= t.hp[c]
			}
			t.lines[c].Write(math.Tanh((x+fb*e)*drive) / drive)
			out[c][i] = float32(x*dry + echo*wet)
		}
	}
}

// Tail lets it rest while silent: the echoes are spaced by the echo time
func (t *Tape) Tail() float64 {
	return t.target()/t.Config().SampleRate + 0.1
}
