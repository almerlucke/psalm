// Package chorus is a stereo chorus: a few copies of the input, each delayed by a slowly moving
// delay, blended with the input so it sounds like several players
package chorus

import (
	"math"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/delay"
	"github.com/almerlucke/psalm/blocks/mix"
	"github.com/almerlucke/psalm/blocks/sine"
)

// Config describes a chorus
type Config struct {
	Rate     float64 // Hz, speed of the delay modulation
	Depth    float64 // milliseconds the delay moves either way (0-10)
	Delay    float64 // milliseconds, the center delay (5-30: shorter is closer to a flanger, longer a doubler)
	Voices   int     // delayed copies per side, 1-4 (0 is 3)
	Spread   float64 // 0-1, stereo width: how far the modulation of the right side is out of phase with the left
	Feedback float64 // 0-0.9, for a more metallic, resonant chorus
	Tone     float64 // Hz, lowpass on the delayed copies (0 is off)
	Mix      float64 // 0 dry, 1 wet, an equal power crossfade: halfway keeps the loudness
}

// DefaultConfig is a lush stereo chorus
func DefaultConfig() Config {
	return Config{
		Rate:   0.6,
		Depth:  3,
		Delay:  15,
		Voices: 3,
		Spread: 1,
		Tone:   9000,
		Mix:    0.4,
	}
}

const (
	maxVoices  = 4
	maxDelayMs = 50.0
	smoothMs   = 50.0 // settings glide this fast, so moving them doesn't click
)

// rates of the voices relative to Rate, slightly apart so they never move together
var voiceRates = [maxVoices]float64{1, 0.83, 1.19, 0.71}

// Chorus is the chorus module: stereo input 0. Value inputs: 1 rate, 2 depth, 3 mix. Messages:
// "rate", "depth", "delay", "spread", "feedback", "tone", "mix"
type Chorus struct {
	*psalm.Module
	cfg   Config
	lines [2]*delay.Delay
	phase [maxVoices]float64
	delay float64 // smoothed settings
	depth float64
	mix   float64
	tone  [2]float64 // lowpass states
}

// New creates a chorus
func New(cfg Config) *Chorus {
	sr := psalm.SampleRate()
	n := int(math.Ceil(maxDelayMs*0.001*sr)) + 4
	c := &Chorus{
		Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}),
		cfg:    cfg,
		lines:  [2]*delay.Delay{delay.New(n), delay.New(n)},
		delay:  cfg.Delay,
		depth:  cfg.Depth,
		mix:    cfg.Mix,
	}
	for v := range c.phase {
		c.phase[v] = float64(v) / maxVoices // spread the voices over the cycle
	}
	c.SetSelf(c)
	return c
}

// Settings returns the settings
func (c *Chorus) Settings() Config { return c.cfg }

func (c *Chorus) SetRate(rate float64)     { c.cfg.Rate = max(0, rate) }
func (c *Chorus) SetDepth(ms float64)      { c.cfg.Depth = max(0, min(ms, 10)) }
func (c *Chorus) SetDelay(ms float64)      { c.cfg.Delay = max(1, min(ms, 35)) }
func (c *Chorus) SetSpread(spread float64) { c.cfg.Spread = max(0, min(spread, 1)) }
func (c *Chorus) SetFeedback(fb float64)   { c.cfg.Feedback = max(0, min(fb, 0.9)) }
func (c *Chorus) SetTone(hz float64)       { c.cfg.Tone = max(0, hz) }
func (c *Chorus) SetMix(mix float64)       { c.cfg.Mix = max(0, min(mix, 1)) }

func (c *Chorus) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		c.ReceiveMessage(msg)
		return
	}
	v := psalm.FltVal[float64](value)
	switch index {
	case 1:
		c.SetRate(v)
	case 2:
		c.SetDepth(v)
	case 3:
		c.SetMix(v)
	}
}

func (c *Chorus) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("rate", c.SetRate)
	msg.IfFltVal("depth", c.SetDepth)
	msg.IfFltVal("delay", c.SetDelay)
	msg.IfFltVal("spread", c.SetSpread)
	msg.IfFltVal("feedback", c.SetFeedback)
	msg.IfFltVal("tone", c.SetTone)
	msg.IfFltVal("mix", c.SetMix)
	msg.IfIntVal("voices", func(v int) { c.cfg.Voices = max(1, min(v, maxVoices)) })
}

func (c *Chorus) Synthesize() {
	var (
		cfg    = &c.cfg
		sr     = c.Config().SampleRate
		in     = c.Input(0).Channels
		out    = c.Output(0).Channels
		voices = max(1, min(cfg.Voices, maxVoices))
		smooth = 1 - math.Exp(-1/(smoothMs*0.001*sr))
		msToS  = 0.001 * sr
		fb     = max(0, min(cfg.Feedback, 0.9))
		offset = 0.25 * max(0, min(cfg.Spread, 1)) // quarter cycle at full spread
		lp     = 1.0
	)
	if cfg.Voices == 0 {
		voices = 3
	}
	if cfg.Tone > 0 {
		lp = 1 - math.Exp(-2*math.Pi*min(cfg.Tone, sr*0.45)/sr)
	}
	outGain := math.Sqrt(float64(voices))

	for i := range out[0] {
		c.delay += (cfg.Delay - c.delay) * smooth
		c.depth += (cfg.Depth - c.depth) * smooth
		c.mix += (cfg.Mix - c.mix) * smooth
		// the voices hardly correlate with the input, so their powers add: an equal power
		// crossfade keeps the loudness at every mix
		dryGain, wetGain := mix.EqualPower(c.mix)
		wetGain *= outGain

		var wet [2]float64
		for v := range voices {
			ph := c.phase[v]
			c.phase[v] += cfg.Rate * voiceRates[v] / sr
			c.phase[v] -= math.Floor(c.phase[v])
			for ch := range 2 {
				p := ph
				if ch == 1 {
					p += offset
				}
				d := (c.delay + c.depth*sine.Cycle(p)) * msToS
				wet[ch] += c.lines[ch].ReadHermite(max(1, d))
			}
		}

		for ch := range 2 {
			x := float64(in[ch][i])
			// the average of the voices feeds back (a loop gain below the feedback amount, also for
			// low notes where the voices add up in phase), the output gets the voices at the level
			// of unrelated signals, which keeps the loudness of the input
			avg := wet[ch] / float64(voices)
			c.tone[ch] += lp * (avg - c.tone[ch])
			c.lines[ch].Write(x + fb*c.tone[ch])
			out[ch][i] = float32(x*dryGain + c.tone[ch]*wetGain)
		}
	}
}

// Tail lets the chorus rest while silent: its longest delay
func (c *Chorus) Tail() float64 {
	return (c.cfg.Delay+c.cfg.Depth)/1000 + 0.02
}
