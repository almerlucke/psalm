package grains

import (
	"math"
	"slices"
	"sort"
	"sync"

	"github.com/almerlucke/sndfile"
)

// ---- the filter of a grain

// the filters a grain can have (Config.Filter)
const (
	FilterOff = iota
	FilterLowpass
	FilterHighpass
	FilterBandpass
	numFilters
)

// FilterNames are the names of the filters, in the order of Config.Filter
var FilterNames = []string{"off", "lowpass", "highpass", "bandpass"}

// svf is a state variable filter (the trapezoidal one: stable when its cutoff changes), each grain has one
type svf struct {
	a1, a2, a3, k, ic1, ic2 float64
	mode                    int
}

func newSVF(mode int, fc, res, sr float64) svf {
	fc = max(20, min(fc, 0.45*sr))
	g := math.Tan(math.Pi * fc / sr)
	k := 2 - 1.95*max(0, min(res, 1))
	a1 := 1 / (1 + g*(g+k))
	return svf{a1: a1, a2: g * a1, a3: g * g * a1, k: k, mode: mode}
}

// run filters the samples in place
func (f *svf) run(buf []float64) {
	for i, x := range buf {
		v3 := x - f.ic2
		v1 := f.a1*f.ic1 + f.a2*v3
		v2 := f.ic2 + f.a2*f.ic1 + f.a3*v3
		f.ic1 = 2*v1 - f.ic1
		f.ic2 = 2*v2 - f.ic2
		switch f.mode {
		case FilterLowpass:
			buf[i] = v2
		case FilterHighpass:
			buf[i] = x - f.k*v1 - v2
		default:
			buf[i] = v1 * f.k * 0.5 // the band of a narrow filter is as loud as the sound there is
		}
	}
}

// ---- the modulation matrix: four slots, each a source that moves a destination

// the sources of a modulation slot
const (
	ModOff      = iota
	ModLFO      // a sine, Rate Hz
	ModRandom   // a new random value for every grain
	ModRise     // from 0 to 1 over the first 1/Rate seconds of the note, and then 1
	ModVelocity // the velocity of the note
	ModEnvelope // the note volume envelope
	numModSources
)

// the destinations of a modulation slot
const (
	ModPosition = iota // +-Amount of the source
	ModPitch           // +-12 semitones
	ModSize            // +-2 octaves of length
	ModPan             // +-half of the stereo field
	ModAmp             // the level of the grain, 1+Amount*value
	ModCutoff          // +-4 octaves
	ModDensity         // faster by 2 octaves
	numModDests
)

// ModSourceNames and ModDestNames name the choices of a slot
var (
	ModSourceNames = []string{"off", "lfo", "random", "rise", "velocity", "envelope"}
	ModDestNames   = []string{"position", "pitch", "size", "pan", "level", "cutoff", "density"}
)

const numMods = 4

// Mod is one slot of the modulation matrix: Source, times Amount (-1 to 1), goes to Dest. Rate is for the lfo (Hz) and the rise
type Mod struct {
	Source, Dest int
	Amount, Rate float64
}

// modValue is what the sources of a destination say now (the sum of the slots that go there), 0 when none does
func (v *Voice) modValue(dest int) float64 {
	sum := 0.0
	for _, m := range v.cfg.Mods {
		if m.Source == ModOff || m.Dest != dest || m.Amount == 0 {
			continue
		}
		var u float64
		switch m.Source {
		case ModLFO:
			u = math.Sin(2 * math.Pi * m.Rate * v.age)
		case ModRandom:
			u = v.Rand().Float64()*2 - 1
		case ModRise:
			u = min(1, v.age*m.Rate)
		case ModVelocity:
			u = v.velocity
		case ModEnvelope:
			u = v.curEnv
		}
		sum += m.Amount * u
	}
	return sum
}

// ---- feedback: grains of what the voice itself played a moment ago

const feedbackSeconds = 4

// fbGain is the level of a grain of the feedback: less than one so it dies away
const fbGain = 0.75

// write puts a sample in the ring (the ring of a voice's own output)
func (r *Recorder) write(x float64) {
	r.ring[r.head] = x
	r.head++
	if r.head == len(r.ring) {
		r.head = 0
	}
	if r.count < len(r.ring) {
		r.count++
	}
}

// ---- slices: markers in a recording

// the choices of Config.Slices: off, the onsets of the recording, or equal parts
var SliceChoices = []string{"off", "onsets", "4", "8", "16", "32"}

var sliceParts = []int{0, 0, 4, 8, 16, 32}

// sliceable is a source that has markers: the slices (the starts of them and the end, 0 to 1) for a choice of SliceChoices
type sliceable interface {
	sliceBounds(mode int) []float64
}

func (s *sampleSource) sliceBounds(mode int) []float64 {
	if mode <= 0 || mode >= len(sliceParts) {
		return nil
	}
	if mode == 1 {
		return append(onsetsOf(s.f), 1)
	}
	n := sliceParts[mode]
	b := make([]float64, n+1)
	for i := range b {
		b[i] = float64(i) / float64(n)
	}
	return b
}

var onsets sync.Map // sndfile.SoundFiler[float32] -> []float64

// onsetsOf finds where the sound of a recording starts anew: the places where the level rises fast (6 dB over the last 60 ms), at least
// 80 ms apart, the 64 strongest. The first slice starts at 0 whatever the recording does there
func onsetsOf(f sndfile.SoundFiler[float32]) []float64 {
	if x, ok := onsets.Load(f); ok {
		return slices.Clone(x.([]float64))
	}
	n := int(f.NumFrames())
	hop := max(1, int(0.01*f.SampleRate()))
	var level []float64
	for start := 0; start < n; start += hop {
		end := min(n, start+hop)
		sum := 0.0
		for ch := range f.NumChannels() {
			for _, x := range f.Buffer(ch, 0)[start:end] {
				sum += float64(x) * float64(x)
			}
		}
		level = append(level, 20*math.Log10(math.Sqrt(sum/float64((end-start)*f.NumChannels()))+1e-5))
	}
	type onset struct {
		at   int
		rise float64
	}
	var found []onset
	for i := 6; i < len(level)-1; i++ {
		mean := 0.0
		for _, l := range level[i-6 : i] {
			mean += l / 6
		}
		rise := level[i] - mean
		if rise < 6 || level[i] < level[i-1] || level[i] < level[i+1] {
			continue
		}
		if len(found) > 0 && i-found[len(found)-1].at < 8 {
			if rise > found[len(found)-1].rise {
				found[len(found)-1] = onset{i, rise}
			}
			continue
		}
		found = append(found, onset{i, rise})
	}
	if len(found) > 63 {
		sort.Slice(found, func(a, b int) bool { return found[a].rise > found[b].rise })
		found = found[:63]
		sort.Slice(found, func(a, b int) bool { return found[a].at < found[b].at })
	}
	starts := []float64{0}
	for _, o := range found {
		// the level rose within the block before: start there
		if t := float64(max(0, o.at-1)*hop) / float64(n); t > starts[len(starts)-1] {
			starts = append(starts, t)
		}
	}
	onsets.Store(f, starts)
	return slices.Clone(starts)
}

// sliceBounds are the markers of the source of the voice for its setting Slices (nil: the whole source)
func (v *Voice) sliceBounds() []float64 {
	if v.cfg.Slices <= 0 {
		return nil
	}
	if v.boundsFor != v.src || v.boundsMode != v.cfg.Slices {
		v.bounds = nil
		if s, ok := v.src.(sliceable); ok {
			v.bounds = s.sliceBounds(v.cfg.Slices)
		}
		v.boundsFor, v.boundsMode = v.src, v.cfg.Slices
	}
	return v.bounds
}
