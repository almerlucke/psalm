package dx

import (
	"math"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm/blocks/voicefilter"
)

// Filter is the settings of the filter of a patch, the way the DX7's are: 0 to 99 (the type 0-6, see
// voicefilter.TypeNames). A synth the DX7 isn't: it makes the sound darker, and its envelope
// moves it
type Filter struct {
	Type      int // 0 off, 1 low pass, 2 band pass, 3 high pass, 4 notch, 5 ladder 24, 6 ladder 12
	Cutoff    int // 20 Hz to 18 kHz
	Resonance int
	EnvAmount int // 0 to 6 octaves
	KeyTrack  int
	Attack    int // 1 ms to 5 s
	Decay     int
	Sustain   int
	Release   int
}

// DefaultFilter is what the settings are before they are set: off, a low pass at 3 kHz when it is on
func DefaultFilter() Filter {
	return Filter{Cutoff: 75, Resonance: 15, EnvAmount: 30, KeyTrack: 50, Attack: 0, Decay: 55, Sustain: 40, Release: 50}
}

// filter returns the settings of the filter of a patch as they are
func (p *Patch) filter() Filter {
	if p.FilterSet {
		return p.Filter
	}
	return DefaultFilter()
}

// ms makes 0 to 99 a time from 1 ms to 5 s
func ms(v int) float64 { return math.Pow(5000, float64(v)/99) }

// Config is the filter as voicefilter plays it
func (f Filter) Config() voicefilter.Config {
	return voicefilter.Config{
		Type:      voicefilter.Type(max(0, min(f.Type, int(voicefilter.Ladder12)))),
		Cutoff:    20 * math.Pow(900, float64(f.Cutoff)/99),
		Resonance: float64(f.Resonance) / 99,
		EnvAmount: float64(f.EnvAmount) / 99 * 6,
		KeyTrack:  float64(f.KeyTrack) / 99,
		Env:       *adsr.NewSetting(1.0, ms(f.Attack), float64(f.Sustain)/99, ms(f.Decay), 1.0, ms(f.Release)),
	}
}
