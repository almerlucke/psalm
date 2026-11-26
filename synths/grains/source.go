package grains

import (
	"math"

	"github.com/almerlucke/sndfile"
)

// Source is what the grains read: a recording, a set of single cycle waves, or the sound that is coming in now. A grain asks the
// source where it starts (Locate) and then reads its samples in blocks (Read), so a source costs one call per grain and
// buffer, not one per sample.
//
// A source speaks in frames of its own, played at its own Rate: a grain that reads at speed 1 hears the source at its pitch.
type Source interface {
	// Rate is the number of frames per second at which the source sounds at its own pitch (the sample rate of a recording)
	Rate() float64
	// Span is the number of frames the position of a note scans over (and wraps at)
	Span() float64
	// Depth is the number of mipmap levels (1 for a source that has none)
	Depth() int
	// Loudness is the level (RMS) of what sounds in the source, for playing every source at the same level; 0 when it is not
	// known (a live input), the source is then played as it is
	Loudness() float64
	// Locate says where a grain starts. position is 0 to 1 (where the note looks), scanned is the number of frames the note
	// has scanned on from there, spray is a random shift in frames, lead is how many frames the grain will move ahead of a
	// source that is being written as it is read (a live input) and must stay behind. The grain reads between lo and hi
	// and wraps there
	Locate(position, scanned, spray, lead float64) (start, lo, hi float64)
	// Read fills dst with the source from pos on, moving inc frames for every sample (negative is backwards) with the
	// mipmap level depth, wrapping between lo and hi, and says where it got to. It mixes the channels of the source to one
	Read(dst []float64, pos, inc float64, depth int, lo, hi float64) float64
	// Ready is whether there is something to read
	Ready() bool
}

// wrap puts x in [0, n)
func wrap(x, n float64) float64 { return x - math.Floor(x/n)*n }

// sampleSource is a recording: a sound file, read with its mipmaps
type sampleSource struct {
	f sndfile.SoundFiler[float32]
}

// NewSampleSource makes a recording a source
func NewSampleSource(f sndfile.SoundFiler[float32]) Source {
	if f == nil {
		return nil
	}
	return &sampleSource{f: f}
}

func (s *sampleSource) Rate() float64     { return s.f.SampleRate() }
func (s *sampleSource) Span() float64     { return float64(s.f.NumFrames()) }
func (s *sampleSource) Depth() int        { return s.f.Depth() }
func (s *sampleSource) Loudness() float64 { return sampleLoudness(s.f) }
func (s *sampleSource) Ready() bool       { return s.f.NumFrames() >= 2 }

func (s *sampleSource) Locate(position, scanned, spray, _ float64) (float64, float64, float64) {
	n := s.Span()
	start := max(0.0, min(position, 1.0))*(n-1) + scanned + spray
	return wrap(start, n), 0, n
}

func (s *sampleSource) Read(dst []float64, pos, inc float64, depth int, lo, hi float64) float64 {
	n := hi - lo
	for i := range dst {
		values := s.f.LookupAll(pos, depth, true)
		x := 0.0
		for _, c := range values {
			x += float64(c)
		}
		dst[i] = x / float64(len(values))
		pos += inc
		pos -= math.Floor(pos/n) * n
	}
	return pos
}

// Waves are the names of the waves of an oscillator source, in the order of the position
var Waves = []string{"sine", "triangle", "saw", "square"}

const (
	oscLen    = 2048 // frames of a cycle
	oscLevels = 16   // levels of the mipmap: fewer harmonics for faster reading
	oscTop    = oscLen / 2
)

// oscSource is a set of single cycle waves. A grain is a short burst of one of them (the position picks the wave), at the
// pitch of the note: the grains of a wavetable, of a pulsar or of an FOF voice. Every wave is made of its harmonics, with
// fewer in the levels that are read fast, so a high note does not alias
type oscSource struct {
	tables [][][]float64 // [wave][level][oscLen+1]
	rms    float64
}

// NewOscillator makes a source of single cycle waves from names (see Waves; none: all of them): a grain of it sounds at
// the pitch of the note, the root note of the config is the one that plays it at 261.6 Hz (middle C: leave it at 60)
func NewOscillator(names ...string) Source {
	if len(names) == 0 {
		names = Waves
	}
	o := &oscSource{}
	sum := 0.0
	for _, name := range names {
		var levels [][]float64
		for d := range oscLevels {
			// the harmonics that stay under half the sample rate when the cycle is read d+1 frames per sample
			h := max(1, oscTop/(d+1))
			levels = append(levels, cycle(name, h))
		}
		o.tables = append(o.tables, levels)
		r := 0.0
		for _, x := range levels[0][:oscLen] {
			r += x * x
		}
		sum += math.Sqrt(r / oscLen)
	}
	o.rms = sum / float64(len(names))
	return o
}

// cycle is one cycle of a wave made of its harmonics up to h
func cycle(name string, h int) []float64 {
	t := make([]float64, oscLen+1)
	for k := 1; k <= h; k++ {
		var amp float64
		sign := 1.0
		switch name {
		case "triangle":
			if k%2 == 0 {
				continue
			}
			amp = 1 / float64(k*k)
			if (k/2)%2 == 1 {
				sign = -1
			}
			amp *= 8 / (math.Pi * math.Pi)
		case "saw":
			amp = 2 / (math.Pi * float64(k))
			if k%2 == 0 {
				sign = -1
			}
		case "square":
			if k%2 == 0 {
				continue
			}
			amp = 4 / (math.Pi * float64(k))
		default: // sine
			if k > 1 {
				continue
			}
			amp = 1
		}
		for i := range oscLen {
			t[i] += sign * amp * math.Sin(2*math.Pi*float64(k*i)/oscLen)
		}
	}
	t[oscLen] = t[0]
	return t
}

func (o *oscSource) Rate() float64     { return oscLen * 261.6255653005986 }
func (o *oscSource) Span() float64     { return oscLen }
func (o *oscSource) Depth() int        { return oscLevels }
func (o *oscSource) Loudness() float64 { return o.rms }
func (o *oscSource) Ready() bool       { return true }

func (o *oscSource) Locate(position, scanned, spray, _ float64) (float64, float64, float64) {
	w := int(math.Round(max(0.0, min(position, 1.0)) * float64(len(o.tables)-1)))
	base := float64(w) * oscLen
	return base + wrap(scanned+spray, oscLen), base, base + oscLen
}

func (o *oscSource) Read(dst []float64, pos, inc float64, depth int, lo, hi float64) float64 {
	w := int(lo / oscLen)
	t := o.tables[w][min(max(depth, 0), oscLevels-1)]
	local := pos - lo
	for i := range dst {
		k := int(local)
		f := local - float64(k)
		dst[i] = t[k] + (t[k+1]-t[k])*f
		local += inc
		local -= math.Floor(local/oscLen) * oscLen
	}
	return lo + local
}
