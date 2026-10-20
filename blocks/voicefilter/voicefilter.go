// Package voicefilter is the filter a synth voice puts after its oscillators, the same on every
// synth: a state variable or a ladder filter on each channel with an envelope of its own, key
// tracking and a drive. A voice makes one with the settings of its config, feeds it messages
// (Receive), tells it when a note starts and ends, and runs every sample through it:
//
//	f := voicefilter.New(cfg.Filter, sr, channels)
//	f.Trigger(durationMs, mode)               // a note starts
//	f.Tick(freq, extraOctaves)                // every sample: the envelope and the cutoff
//	out = f.Process(channel, in)              // every sample for every channel
//
// The settings are messages: "filterType", "cutoff", "resonance", "filterEnvAmount", "keyTrack",
// "filterDrive" and the envelope "filter" (blocks/envelope).
package voicefilter

import (
	"math"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/envelope"
	"github.com/almerlucke/psalm/blocks/filters/ladder"
	"github.com/almerlucke/psalm/blocks/filters/svf"
	"github.com/almerlucke/psalm/params"
)

// Type is what the filter does
type Type int

const (
	Off Type = iota
	LowPass
	BandPass
	HighPass
	Notch
	Ladder24 // a Moog style ladder, 24 dB/oct
	Ladder12
)

// TypeNames are the names of the types, by number
var TypeNames = []string{"off", "low pass", "band pass", "high pass", "notch", "ladder 24", "ladder 12"}

// Config is the settings of a filter
type Config struct {
	Type      Type
	Cutoff    float64 // Hz
	Resonance float64 // 0-1
	EnvAmount float64 // octaves the envelope opens the cutoff at its peak
	KeyTrack  float64 // 0-1: the cutoff follows the note (relative to middle C)
	Drive     float64 // ladder input saturation (0 is 1)
	Env       adsr.Setting
}

// Default is a low pass at 2 kHz with a gentle envelope
func Default() Config {
	return Config{Type: LowPass, Cutoff: 2000, Resonance: 0.2, EnvAmount: 1, KeyTrack: 0.5,
		Env: *adsr.NewSetting(1.0, 5.0, 0.4, 300.0, 1.0, 300.0)}
}

const middleC = 261.6255653005986

// update is the interval in samples at which the coefficients are recalculated
const update = 4

// Filter is the filter of a voice
type Filter struct {
	cfg     Config
	svf     []*svf.SVF
	ladder  []*ladder.Ladder
	env     *adsr.ADSR
	cutoff  float64 // smoothed
	res     float64
	count   int
	drive   float64
	glideC  float64
	sr      float64
	lastEnv float64
}

// normalize gives a config without a filter an open one, with a plain envelope
func normalize(cfg Config) Config {
	if cfg.Cutoff <= 0 {
		cfg.Cutoff = 18000
	}
	if cfg.Env == (adsr.Setting{}) {
		cfg.Env = Default().Env
	}
	return cfg
}

// New returns a filter for a number of channels
func New(cfg Config, sr float64, channels int) *Filter {
	cfg = normalize(cfg)
	f := &Filter{cfg: cfg, sr: sr, cutoff: cfg.Cutoff, res: cfg.Resonance, glideC: 1 - math.Exp(-1/(0.005*sr/update))}
	f.env = adsr.New(&f.cfg.Env, adsr.NoteOff, sr)
	for range channels {
		f.svf = append(f.svf, svf.New(cfg.Cutoff, cfg.Resonance, sr))
		f.ladder = append(f.ladder, ladder.New(cfg.Cutoff, cfg.Resonance, sr))
	}
	return f
}

// SetConfig changes all settings at once, the filter and its envelope keep sounding
func (f *Filter) SetConfig(cfg Config) {
	f.cfg = normalize(cfg)
}

// Config returns the settings as they are now
func (f *Filter) Config() Config { return f.cfg }

// On says whether the filter does anything
func (f *Filter) On() bool { return f.cfg.Type != Off }

// Receive takes the settings from a message
func (f *Filter) Receive(msg psalm.Message) {
	envelope.Receive(msg, "filter", &f.cfg.Env)
	msg.IfIntVal("filterType", func(t int) { f.cfg.Type = Type(max(0, min(t, int(Ladder12)))) })
	msg.IfFltVal("cutoff", func(v float64) { f.cfg.Cutoff = math.Max(20, v) })
	msg.IfFltVal("resonance", func(v float64) { f.cfg.Resonance = math.Max(0, math.Min(v, 1)) })
	msg.IfFltVal("filterEnvAmount", func(v float64) { f.cfg.EnvAmount = v })
	msg.IfFltVal("keyTrack", func(v float64) { f.cfg.KeyTrack = v })
	msg.IfFltVal("filterDrive", func(v float64) { f.cfg.Drive = v })
}

// Trigger starts the envelope for a note (a duration in milliseconds, with the release mode)
func (f *Filter) Trigger(durationMs float64, mode adsr.ReleaseMode) {
	f.env.TriggerFull(durationMs, 1.0, &f.cfg.Env, mode)
}

// Release lets go of the envelope
func (f *Filter) Release() { f.env.Release() }

// Clear empties the filter and the envelope (a new voice)
func (f *Filter) Clear() {
	f.env.Clear()
	for _, s := range f.svf {
		s.Clear()
	}
	for _, l := range f.ladder {
		l.Clear()
	}
	f.cutoff, f.res = f.cfg.Cutoff, f.cfg.Resonance
	f.count = 0
}

// Snap empties the memory of the filter and moves the cutoff and the resonance to their settings at once, the envelope
// goes on: a filter that comes in under a sound starts where it is going, and not from a cutoff it had a while ago
func (f *Filter) Snap() {
	for _, s := range f.svf {
		s.Clear()
	}
	for _, l := range f.ladder {
		l.Clear()
	}
	f.cutoff, f.res = f.cfg.Cutoff, f.cfg.Resonance
	f.count = 0
}

// Env returns the value of the envelope at the last Tick, 0-1
func (f *Filter) Env() float64 { return f.lastEnv }

// Tick moves the envelope a sample on and, every few samples, sets the cutoff: freq is the note's
// frequency (for the key tracking), extra the octaves modulation adds (an LFO, the wheel, a matrix);
// resMod is added to the resonance
func (f *Filter) Tick(freq, extra, resMod float64) {
	f.lastEnv = f.env.Generate()
	if f.cfg.Type == Off {
		return
	}
	if f.count == 0 {
		f.cutoff += (f.cfg.Cutoff - f.cutoff) * f.glideC
		f.res += (f.cfg.Resonance - f.res) * f.glideC
		cut := f.cutoff * math.Pow(freq/middleC, f.cfg.KeyTrack) * math.Exp2(f.cfg.EnvAmount*f.lastEnv+extra)
		cut = math.Max(20, math.Min(cut, 0.45*f.sr))
		res := math.Max(0, math.Min(f.res+resMod, 1))
		switch f.cfg.Type {
		case Ladder24, Ladder12:
			drive := f.cfg.Drive
			if drive <= 0 {
				drive = 1
			}
			for _, l := range f.ladder {
				if drive != f.drive {
					l.SetDrive(drive)
				}
				l.Set(cut, res)
			}
			f.drive = drive
		default:
			for _, s := range f.svf {
				s.Set(cut, res)
			}
		}
	}
	f.count = (f.count + 1) % update
}

// Process runs a sample of a channel through the filter
func (f *Filter) Process(channel int, s float64) float64 {
	switch f.cfg.Type {
	case LowPass:
		return f.svf[channel].Process(s, svf.LowPass)
	case BandPass:
		return f.svf[channel].Process(s, svf.BandPass)
	case HighPass:
		return f.svf[channel].Process(s, svf.HighPass)
	case Notch:
		return f.svf[channel].Process(s, svf.Notch)
	case Ladder24:
		return f.ladder[channel].Process(s, ladder.LowPass24)
	case Ladder12:
		return f.ladder[channel].Process(s, ladder.LowPass12)
	}
	return s
}

// Describe describes the settings and the envelope of a filter as they are now
func (f *Filter) Describe(group string) ([]params.Param, params.Envelope) {
	return Describe(f.cfg, group)
}

// Describe describes the settings and the envelope of a config for an editor (see params)
func Describe(c Config, group string) ([]params.Param, params.Envelope) {
	if c.Cutoff <= 0 {
		c.Cutoff = 18000
	}
	if c.Env == (adsr.Setting{}) {
		c.Env = Default().Env
	}
	return []params.Param{
		params.C(group, "filterType", TypeNames, int(c.Type), params.Label("type")),
		params.K(group, "cutoff", 20, 18000, math.Max(20, c.Cutoff), params.Exp(), params.Unit("Hz")),
		params.K(group, "resonance", 0, 1, c.Resonance),
		params.K(group, "filterEnvAmount", -6, 6, c.EnvAmount, params.Unit("oct"), params.Label("env amount")),
		params.K(group, "keyTrack", 0, 1, c.KeyTrack, params.Label("key track")),
		params.K(group, "filterDrive", 1, 8, math.Max(1, c.Drive), params.Exp(), params.Label("drive")),
	}, params.E(group, "filter", "filter", c.Env.AttackDuration, c.Env.DecayDuration, c.Env.DecayLevel, c.Env.ReleaseDuration)
}
