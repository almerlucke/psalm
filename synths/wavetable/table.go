package wavetable

import (
	"fmt"
	"math"

	"github.com/almerlucke/psalm/blocks/fft"
	"github.com/almerlucke/sndfile"
	"github.com/almerlucke/sndfile/writer"
)

// Table is a wavetable: a series of single cycle frames to morph through. Every frame is stored as
// a set of band limited levels (mipmaps), each with half the harmonics of the one before, so high
// notes play a level without harmonics above the Nyquist frequency and never alias
type Table struct {
	frameSize int
	frames    [][]level // frames[frame][level]
	gains     []float64 // per frame, brings every frame to the average loudness of the table
}

// maxFrameGain limits how much a quiet frame is brought up
const maxFrameGain = 4.0

// level is one band limited version of a frame
type level struct {
	data []float32 // one cycle plus a guard sample for interpolation
	size float64
	mask int
}

// minLevelSize is the smallest level. Levels hold 32 samples per harmonic (up to the frame size),
// so the images of linear interpolation stay far below hearing (about -70 dB)
const minLevelSize = 64

// NewTable creates a table from samples holding consecutive frames of frameSize (a power of 2) samples.
// Every frame gets its DC removed, the table is normalized to a peak of 1 (the loudness differences
// between frames are kept)
func NewTable(samples []float32, frameSize int) (*Table, error) {
	if frameSize < minLevelSize || frameSize&(frameSize-1) != 0 {
		return nil, fmt.Errorf("wavetable: frame size %d is not a power of 2 of at least %d", frameSize, minLevelSize)
	}
	numFrames := len(samples) / frameSize
	if numFrames == 0 {
		return nil, fmt.Errorf("wavetable: %d samples is less than one frame of %d", len(samples), frameSize)
	}

	t := &Table{frameSize: frameSize, frames: make([][]level, numFrames)}

	// the spectrum of every frame
	f := fft.New(frameSize)
	spectra := make([][2][]float64, numFrames)
	re := make([]float64, frameSize)
	im := make([]float64, frameSize)
	for i := range numFrames {
		for j := range frameSize {
			re[j] = float64(samples[i*frameSize+j])
			im[j] = 0
		}
		f.Forward(re, im)
		spectra[i] = [2][]float64{append([]float64(nil), re...), append([]float64(nil), im...)}
	}

	// band limited levels: level l keeps the harmonics up to frameSize/2 >> l, rebuilt with an
	// inverse FFT at a size of 32 samples per harmonic
	peak := 0.0
	ffts := map[int]*fft.FFT{}
	for i, spec := range spectra {
		for harmonics := frameSize / 2; harmonics >= 1; harmonics /= 2 {
			size := min(frameSize, max(minLevelSize, 32*harmonics))
			lf, ok := ffts[size]
			if !ok {
				lf = fft.New(size)
				ffts[size] = lf
			}
			lre := make([]float64, size)
			lim := make([]float64, size)
			scale := float64(size) / float64(frameSize)
			for k := 1; k <= harmonics && k < size/2; k++ { // no DC, no Nyquist
				lre[k], lim[k] = spec[0][k]*scale, spec[1][k]*scale
				lre[size-k], lim[size-k] = spec[0][frameSize-k]*scale, spec[1][frameSize-k]*scale
			}
			lf.Inverse(lre, lim)

			data := make([]float32, size+1)
			for j := range size {
				data[j] = float32(lre[j])
				peak = math.Max(peak, math.Abs(lre[j]))
			}
			data[size] = data[0]
			t.frames[i] = append(t.frames[i], level{data: data, size: float64(size), mask: size - 1})
		}
	}

	if peak > 0 {
		gain := float32(1.0 / peak)
		for _, levels := range t.frames {
			for _, l := range levels {
				for j := range l.data {
					l.data[j] *= gain
				}
			}
		}
	}

	// loudness of every frame (all harmonics), the gains bring them to the average
	rmss := make([]float64, numFrames)
	mean := 0.0
	for i, levels := range t.frames {
		sum := 0.0
		for _, x := range levels[0].data[:frameSize] {
			sum += float64(x) * float64(x)
		}
		rmss[i] = math.Sqrt(sum / float64(frameSize))
		mean += rmss[i] / float64(numFrames)
	}
	t.gains = make([]float64, numFrames)
	for i, r := range rmss {
		t.gains[i] = 1
		if r > 1e-6 {
			t.gains[i] = math.Min(maxFrameGain, mean/r)
		}
	}

	return t, nil
}

// Load loads a table from a sound file (the first channel) of consecutive frames of frameSize
// samples, like the 2048 sample tables of most wavetable synths
func Load(path string, frameSize int) (*Table, error) {
	sf, err := sndfile.NewSoundFile[float32](path)
	if err != nil {
		return nil, err
	}
	return NewTable(sf.Buffer(0, 0), frameSize)
}

// MustLoad loads a table and panics on an error
func MustLoad(path string, frameSize int) *Table {
	t, err := Load(path, frameSize)
	if err != nil {
		panic(err)
	}
	return t
}

// oversample is how many times finer FromFunc draws a cycle before keeping the harmonics that fit
// in a frame. Shapes with jumps (pulses, sync) have harmonics far above the frame, drawn at the
// frame size those would fold back onto the harmonics; drawn 16 times finer what folds back is
// more than 80 dB down
const oversample = 16

// FromFunc creates a table of numFrames frames, f returns the value of a frame at a phase (0-1) of
// its cycle, for a frame position (0-1) through the table. Any shape works, jumps included: every
// cycle is drawn oversampled and reduced to the harmonics that fit in a frame
func FromFunc(numFrames int, frameSize int, f func(position float64, phase float64) float64) *Table {
	n := frameSize * oversample
	big := fft.New(n)
	re := make([]float64, n)
	im := make([]float64, n)
	return FromSpectrum(numFrames, frameSize, func(position float64, sin []float64, cos []float64) {
		for j := range n {
			re[j] = f(position, float64(j)/float64(n))
			im[j] = 0
		}
		big.Forward(re, im)
		for k := 1; k <= len(sin); k++ {
			// x(t) = sum a cos(2 pi k t) + b sin(2 pi k t), with X[k] = n/2 (a - i b)
			cos[k-1] = 2 * re[k] / float64(n)
			sin[k-1] = -2 * im[k] / float64(n)
		}
	})
}

// FromHarmonics creates a table from the amplitudes of the harmonics of every frame: f fills amps
// (index 0 is the fundamental, up to frameSize/2 harmonics, all zero when called) for a frame
// position (0-1). The harmonics are sines starting at phase 0, the table is exact, without aliasing
func FromHarmonics(numFrames int, frameSize int, f func(position float64, amps []float64)) *Table {
	return FromSpectrum(numFrames, frameSize, func(position float64, sin []float64, _ []float64) {
		f(position, sin)
	})
}

// FromSpectrum creates a table from the sine and cosine amplitudes of the harmonics of every
// frame, f fills them (index 0 is the fundamental, all zero when called) for a frame position (0-1)
func FromSpectrum(numFrames int, frameSize int, f func(position float64, sin []float64, cos []float64)) *Table {
	half := frameSize / 2
	sin := make([]float64, half)
	cos := make([]float64, half)
	frame := fft.New(frameSize)
	re := make([]float64, frameSize)
	im := make([]float64, frameSize)
	samples := make([]float32, numFrames*frameSize)

	for i := range numFrames {
		position := 0.0
		if numFrames > 1 {
			position = float64(i) / float64(numFrames-1)
		}
		clear(sin)
		clear(cos)
		f(position, sin, cos)

		clear(re)
		clear(im)
		for k := 1; k < half; k++ { // the Nyquist harmonic can't hold a sine, it is left out
			re[k] = cos[k-1] * float64(frameSize) / 2
			im[k] = -sin[k-1] * float64(frameSize) / 2
			re[frameSize-k] = re[k]
			im[frameSize-k] = -im[k]
		}
		frame.Inverse(re, im)
		for j := range frameSize {
			samples[i*frameSize+j] = float32(re[j])
		}
	}

	t, err := NewTable(samples, frameSize)
	if err != nil {
		panic(err)
	}
	return t
}

// Save writes the table as a 32 bit float WAV file of consecutive frames (all harmonics, normalized
// as in the table), the format most wavetable synths load
func (t *Table) Save(path string) error {
	w, err := writer.New(path, writer.WAV, 1, 44100, writer.NewNoConverter(t.frameSize))
	if err != nil {
		return err
	}
	for _, levels := range t.frames {
		if err := w.Write(levels[0].data[:t.frameSize], false); err != nil {
			_ = w.Close()
			return err
		}
	}
	return w.Close()
}

// Basic is a table morphing from a sine through a triangle and a sawtooth to a square, so the synth
// works without table files
func Basic() *Table {
	shapes := []func(p float64) float64{
		func(p float64) float64 { return math.Sin(2 * math.Pi * p) },
		func(p float64) float64 { return 1 - 4*math.Abs(math.Mod(p+0.25, 1)-0.5) },
		func(p float64) float64 { return 2*math.Mod(p+0.5, 1) - 1 },
		func(p float64) float64 {
			if p < 0.5 {
				return 1
			}
			return -1
		},
	}
	return FromFunc(4, 2048, func(position float64, phase float64) float64 {
		return shapes[int(math.Round(position*3))](phase)
	})
}

// NumFrames returns the number of frames
func (t *Table) NumFrames() int {
	return len(t.frames)
}

// FrameSize returns the number of samples per frame
func (t *Table) FrameSize() int {
	return t.frameSize
}

// levelFor returns the level to play at a phase increment (cycles per sample) and how much of the
// next level (half the harmonics) to mix in. Both levels keep their highest harmonic below 0.48
// times the sample rate, so nothing aliases: as the pitch rises through an octave the level fades
// into the next, smoothly giving up the harmonics that would soon cross the Nyquist frequency
func (t *Table) levelFor(inc float64) (int, float64) {
	harmonics := float64(t.frameSize / 2)
	maxHarmonic := 0.48 / math.Max(inc, 1e-9)
	x := math.Log2(harmonics / maxHarmonic) // level x holds exactly the harmonics allowed
	l := int(math.Floor(x)) + 1             // level l and l+1 are both safe
	if l < 0 {
		return 0, 0
	}
	n := len(t.frames[0]) - 1
	if l >= n {
		return n, 0
	}
	return l, x - math.Floor(x)
}

// lookup reads a level of a frame at a phase (0-1) with linear interpolation
func (l *level) lookup(phase float64) float64 {
	pos := phase * l.size
	i := int(pos)
	frac := pos - float64(i)
	i &= l.mask
	a := float64(l.data[i])
	return a + (float64(l.data[i+1])-a)*frac
}

// cursor is a position in the table: the frames to morph between and the levels to read, computed
// once per sample and shared by all the oscillators reading the table at about the same pitch
type cursor struct {
	f0, f1 []level
	ffrac  float64
	lv     int
	lfrac  float64
	gain   float64 // loudness compensation of the frames
}

func (t *Table) cursor(position float64, inc float64) cursor {
	fp := math.Max(0, math.Min(position, 1)) * float64(len(t.frames)-1)
	f0 := int(fp)
	f1 := min(f0+1, len(t.frames)-1)
	c := cursor{f0: t.frames[f0], f1: t.frames[f1], ffrac: fp - float64(f0)}
	c.gain = t.gains[f0] + (t.gains[f1]-t.gains[f0])*c.ffrac
	c.lv, c.lfrac = t.levelFor(inc)
	if c.lv+1 >= len(c.f0) {
		c.lfrac = 0
	}
	return c
}

func (c *cursor) readLevel(l int, phase float64) float64 {
	a := c.f0[l].lookup(phase)
	if c.ffrac == 0 {
		return a
	}
	return a + (c.f1[l].lookup(phase)-a)*c.ffrac
}

func (c *cursor) read(phase float64) float64 {
	x := c.readLevel(c.lv, phase)
	if c.lfrac > 0 {
		x += (c.readLevel(c.lv+1, phase) - x) * c.lfrac
	}
	return x
}

// Sample reads the table at a phase (0-1), a frame position (0-1, morphing between frames) and a
// phase increment (cycles per sample, picks the band limited level)
func (t *Table) Sample(phase float64, position float64, inc float64) float64 {
	c := t.cursor(position, inc)
	return c.read(phase)
}
