// Package eq is a channel EQ: a low cut, a low shelf, two peaking bands, a high shelf and a high
// cut, the tone of a part in a mix (the low end out of a pad, presence on a voice, air on the top)
package eq

import (
	"math"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/filters/biquad"
)

// Band is a band of the EQ: its frequency, its gain (dB) and how wide it is (Q, peaks only)
type Band struct {
	Freq float64
	Gain float64
	Q    float64 // 0 is 1 (a peak), the shelves keep 0.707
}

// Config describes an EQ
type Config struct {
	LowCut  float64 // Hz, a 12 dB/octave highpass (0 is off)
	Low     Band    // a low shelf
	Mid1    Band    // a peak
	Mid2    Band    // a peak
	High    Band    // a high shelf
	HighCut float64 // Hz, a 12 dB/octave lowpass (0 is off)
	Output  float64 // dB after it
}

// DefaultConfig is a flat EQ: the bands at 100 Hz, 500 Hz, 2.5 kHz and 8 kHz, at 0 dB
func DefaultConfig() Config {
	return Config{
		Low:  Band{Freq: 100},
		Mid1: Band{Freq: 500, Q: 1},
		Mid2: Band{Freq: 2500, Q: 1},
		High: Band{Freq: 8000},
	}
}

// the bands in order
const (
	lowCut = iota
	low
	mid1
	mid2
	high
	highCut
	bands
)

// EQ is the module: input 0 is stereo audio
type EQ struct {
	*psalm.Module
	cfg     Config
	filters [2][bands]*biquad.Biquad
	now     [bands]Band // where the bands are (they glide to the settings)
	on      [bands]bool
}

// New creates an EQ
func New(cfg Config) *EQ {
	e := &EQ{Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}), cfg: cfg}
	sr := psalm.SampleRate()
	for c := range e.filters {
		for b := range bands {
			e.filters[c][b] = biquad.New(biquad.Peak, 1000, 1, 0, sr)
		}
	}
	e.now = e.targets()
	e.update()
	e.SetSelf(e)
	return e
}

// Settings returns the settings
func (e *EQ) Settings() Config { return e.cfg }

// SetConfig changes the settings (the bands glide there)
func (e *EQ) SetConfig(cfg Config) { e.cfg = cfg }

// targets returns the bands as set
func (e *EQ) targets() [bands]Band {
	c := e.cfg
	return [bands]Band{{Freq: c.LowCut}, c.Low, c.Mid1, c.Mid2, c.High, {Freq: c.HighCut}}
}

// update sets the filters to where the bands are now
func (e *EQ) update() {
	sr := e.Config().SampleRate
	for b, band := range e.now {
		var typ biquad.Type
		q := 0.707
		switch b {
		case lowCut:
			typ = biquad.HighPass
			e.on[b] = band.Freq > 0
		case highCut:
			typ = biquad.LowPass
			e.on[b] = band.Freq > 0 && band.Freq < sr*0.49
		case low:
			typ = biquad.LowShelf
			e.on[b] = band.Freq > 0 && math.Abs(band.Gain) > 0.01
		case high:
			typ = biquad.HighShelf
			e.on[b] = band.Freq > 0 && math.Abs(band.Gain) > 0.01
		default:
			typ = biquad.Peak
			q = band.Q
			if q <= 0 {
				q = 1
			}
			e.on[b] = band.Freq > 0 && math.Abs(band.Gain) > 0.01
		}
		if !e.on[b] {
			continue
		}
		for c := range e.filters {
			e.filters[c][b].Set(typ, band.Freq, q, band.Gain)
		}
	}
}

func (e *EQ) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		e.ReceiveMessage(msg)
	}
}

// ReceiveMessage takes "lowCut", "highCut", "output", and per band ("low", "mid1", "mid2", "high")
// its "...Freq", "...Gain" and "...Q": "lowGain", "mid1Freq", ...
func (e *EQ) ReceiveMessage(msg psalm.Message) {
	c := &e.cfg
	msg.IfFltVal("lowCut", func(v float64) { c.LowCut = v })
	msg.IfFltVal("highCut", func(v float64) { c.HighCut = v })
	msg.IfFltVal("output", func(v float64) { c.Output = v })
	for name, band := range map[string]*Band{"low": &c.Low, "mid1": &c.Mid1, "mid2": &c.Mid2, "high": &c.High} {
		msg.IfFltVal(name+"Freq", func(v float64) { band.Freq = v })
		msg.IfFltVal(name+"Gain", func(v float64) { band.Gain = v })
		msg.IfFltVal(name+"Q", func(v float64) { band.Q = v })
	}
}

// glideTo moves a value a part of the way, done when close
func glideTo(now, target float64, exp bool) float64 {
	if exp && now > 0 && target > 0 { // frequencies glide over the octaves
		next := now * math.Pow(target/now, 0.3)
		if math.Abs(math.Log(target/next)) < 0.001 {
			return target
		}
		return next
	}
	next := now + (target-now)*0.3
	if math.Abs(target-next) < 0.01 {
		return target
	}
	return next
}

func (e *EQ) Synthesize() {
	// the bands glide to their settings, a part every buffer: no steps, no clicks
	if t := e.targets(); t != e.now {
		for b := range e.now {
			e.now[b].Freq = glideTo(e.now[b].Freq, t[b].Freq, true)
			e.now[b].Gain = glideTo(e.now[b].Gain, t[b].Gain, false)
			e.now[b].Q = glideTo(e.now[b].Q, t[b].Q, false)
		}
		e.update()
	}
	in, out := e.Input(0).Channels, e.Output(0).Channels
	gain := math.Pow(10, e.cfg.Output/20)
	for c := range out {
		src := in[c%len(in)]
		for i, x := range src {
			v := float64(x)
			for b := range bands {
				if e.on[b] {
					v = e.filters[c][b].Process(v)
				}
			}
			out[c][i] = float32(v * gain)
		}
	}
}

// Tail lets it rest while silent
func (e *EQ) Tail() float64 { return 0.05 }
