// Package gen generates wavetables: every generator sweeps one synthesis parameter over the frames
// of a table, so moving the position plays that sweep. All tables have frames of 2048 samples
package gen

import (
	"math"
	"math/rand/v2"

	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/sndfile"
)

// FrameSize is the size of the frames of the generated tables
const FrameSize = 2048

// fromLevels creates a table from the levels of the harmonics of every frame with Schroeder
// phases: the harmonics start at phases spread so their peaks don't pile up. It sounds the same as
// all harmonics starting together, but the wave has a lower peak for its loudness, so the table
// (normalized to its peak) plays louder
func fromLevels(numFrames int, numHarmonics int, f func(position float64, levels []float64)) *wavetable.Table {
	levels := make([]float64, FrameSize/2)
	return wavetable.FromSpectrum(numFrames, FrameSize, func(position float64, sin []float64, cos []float64) {
		clear(levels)
		f(position, levels)
		for k := 1; k <= len(levels); k++ {
			phase := math.Pi * float64(k*(k-1)) / float64(numHarmonics)
			sin[k-1] = levels[k-1] * math.Cos(phase)
			cos[k-1] = levels[k-1] * math.Sin(phase)
		}
	})
}

// Additive builds a table from the amplitudes of its partials: harmonics[0] is the fundamental,
// harmonics[1] the second harmonic, and so on (up to 1023, the rest is ignored; negative amplitudes
// count as 0). Every frame has the same spectrum, so the position changes nothing: a single cycle
// of your own (numFrames below 1 is 1). The table is normalized to its peak, so it is the ratios
// of the amplitudes that count
func Additive(harmonics []float64, numFrames int) *wavetable.Table {
	return AdditiveMorph(harmonics, harmonics, numFrames)
}

// AdditiveMorph builds a table that morphs from the partials of from to the partials of to: the
// amplitude of every harmonic crossfades linearly over the frames (a list shorter than the other
// counts as 0 for the harmonics it lacks). See Additive
func AdditiveMorph(from, to []float64, numFrames int) *wavetable.Table {
	numFrames = max(1, numFrames)
	n := min(max(len(from), len(to)), FrameSize/2-1)
	if n == 0 { // nothing to sound: a sine
		from, to, n = []float64{1}, []float64{1}, 1
	}
	at := func(h []float64, k int) float64 {
		if k < len(h) {
			return max(0, h[k])
		}
		return 0
	}
	return fromLevels(numFrames, n, func(position float64, amps []float64) {
		for k := range n {
			a, b := at(from, k), at(to, k)
			amps[k] = a + (b-a)*position
		}
	})
}

// Brightness morphs from a sine to a bright sawtooth: the harmonics of a sawtooth fade in from
// low to high as the position rises, like opening a lowpass filter
func Brightness(numFrames int) *wavetable.Table {
	return wavetable.FromHarmonics(numFrames, FrameSize, func(position float64, amps []float64) {
		cutoff := 0.5 + position*position*float64(len(amps)) // in harmonics
		for k := 1; k <= len(amps); k++ {
			amps[k-1] = math.Exp(-float64(k-1)/cutoff) / float64(k)
		}
	})
}

// PWM narrows a pulse from a square (width 0.5) to minWidth (for example 0.02), the classic pulse
// width sweep from hollow to thin and nasal
func PWM(numFrames int, minWidth float64) *wavetable.Table {
	minWidth = max(0.001, min(minWidth, 0.5))
	return wavetable.FromFunc(numFrames, FrameSize, func(position float64, phase float64) float64 {
		if phase < 0.5+(minWidth-0.5)*position {
			return 1
		}
		return -1
	})
}

// Sync is a hard synced sawtooth: a sawtooth restarted at every cycle, whose own frequency rises
// from the cycle's (ratio 1) to maxRatio times as the position rises, the tearing sound of a sync
// sweep
func Sync(numFrames int, maxRatio float64) *wavetable.Table {
	maxRatio = max(1, maxRatio)
	return wavetable.FromFunc(numFrames, FrameSize, func(position float64, phase float64) float64 {
		ratio := 1 + position*(maxRatio-1)
		x := phase * ratio
		return 2*(x-math.Floor(x)) - 1
	})
}

// FM is two operator FM: a sine carrier modulated by a sine at ratio (a whole number, others are
// rounded so the cycle closes) times its frequency, the modulation index rising from 0 to maxIndex.
// Ratio 1 brightens towards a sawtooth like tone, 2 gives odd and even mixes, higher ratios bells
// and metal
func FM(numFrames int, ratio float64, maxIndex float64) *wavetable.Table {
	ratio = max(1, math.Round(ratio))
	return wavetable.FromFunc(numFrames, FrameSize, func(position float64, phase float64) float64 {
		index := position * maxIndex
		return math.Sin(2*math.Pi*phase + index*math.Sin(2*math.Pi*ratio*phase))
	})
}

// Fold drives a sine into a wavefolder, the gain rising from 1 to maxGain: every time the wave
// passes the edge it folds back, adding harmonics in waves (west coast synthesis). Bias (-1 to 1)
// offsets the sine before folding, which adds even harmonics
func Fold(numFrames int, maxGain float64, bias float64) *wavetable.Table {
	maxGain = max(1, maxGain)
	return wavetable.FromFunc(numFrames, FrameSize, func(position float64, phase float64) float64 {
		gain := 1 + position*(maxGain-1)
		return fold(gain * (math.Sin(2*math.Pi*phase) + bias))
	})
}

// fold reflects x back into -1 to 1 like a triangle
func fold(x float64) float64 {
	m := math.Mod(x+1, 4)
	if m < 0 {
		m += 4
	}
	return 1 - math.Abs(m-2)
}

// vowel holds the first three formants (Hz) and their levels
type vowel struct {
	freqs  [3]float64
	levels [3]float64
}

// vowels a, e, i, o, u: average formant frequencies of adult male speakers from the classic
// measurements of Peterson and Barney (1952)
var vowels = []vowel{
	{[3]float64{730, 1090, 2440}, [3]float64{1, 0.5, 0.25}},
	{[3]float64{530, 1840, 2480}, [3]float64{1, 0.45, 0.3}},
	{[3]float64{270, 2290, 3010}, [3]float64{1, 0.3, 0.25}},
	{[3]float64{570, 840, 2410}, [3]float64{1, 0.6, 0.15}},
	{[3]float64{300, 870, 2240}, [3]float64{1, 0.3, 0.1}},
}

// formantBandwidths in Hz
var formantBandwidths = [3]float64{80, 100, 120}

// Vowels morphs through the vowels a, e, i, o and u: a voice like source (a sawtooth spectrum)
// shaped by the formants of each vowel. The formants are placed for a note of pitch Hz (for
// example 130, a low male voice); the table moves them with the note, so it sounds most natural
// near that pitch
func Vowels(numFrames int, pitch float64) *wavetable.Table {
	return fromLevels(numFrames, int(6000/pitch), func(position float64, amps []float64) {
		// the vowel at this position, formants interpolated between neighbours
		x := position * float64(len(vowels)-1)
		i := min(int(x), len(vowels)-2)
		frac := x - float64(i)
		var v vowel
		for f := range 3 {
			a, b := vowels[i], vowels[i+1]
			v.freqs[f] = a.freqs[f] * math.Pow(b.freqs[f]/a.freqs[f], frac)
			v.levels[f] = a.levels[f] + (b.levels[f]-a.levels[f])*frac
		}

		for k := 1; k <= len(amps); k++ {
			freq := float64(k) * pitch
			if freq > 6000 {
				break
			}
			env := 0.02 // a little of every harmonic, a voice is never fully silent between formants
			for f := range 3 {
				d := (freq - v.freqs[f]) / (formantBandwidths[f] / 2)
				env += v.levels[f] / (1 + d*d)
			}
			amps[k-1] = env / math.Sqrt(float64(k))
		}
	})
}

// Drawbars adds the harmonics of an organ one by one: at position 0 only the fundamental sounds,
// harmonic by harmonic the others fade in up to numHarmonics. The harmonics are equally loud, so it
// ends bright and buzzy
func Drawbars(numFrames int, numHarmonics int) *wavetable.Table {
	numHarmonics = max(1, min(numHarmonics, FrameSize/2-1))
	return fromLevels(numFrames, numHarmonics, func(position float64, amps []float64) {
		open := 1 + position*float64(numHarmonics-1) // number of harmonics sounding, fractional fades in
		for k := 1; k <= numHarmonics; k++ {
			amps[k-1] = max(0, min(1, open-float64(k)+1)) / math.Sqrt(float64(k))
		}
	})
}

// Random morphs smoothly between keyframes of random spectra: every keyframe picks random levels
// for the first numHarmonics harmonics (some left out, the levels falling with the harmonic number),
// the frames in between crossfade the levels. Every seed gives another table, the same seed the
// same table
func Random(numFrames int, keyframes int, numHarmonics int, seed uint64) *wavetable.Table {
	keyframes = max(2, keyframes)
	numHarmonics = max(1, min(numHarmonics, FrameSize/2-1))
	r := rand.New(rand.NewPCG(seed, 0x9e3779b97f4a7c15))

	keys := make([][]float64, keyframes)
	for i := range keys {
		keys[i] = make([]float64, numHarmonics)
		keys[i][0] = 0.5 + 0.5*r.Float64() // always a fundamental
		for k := 2; k <= numHarmonics; k++ {
			if r.Float64() < 0.35 {
				continue // left out, gives every keyframe its own gaps
			}
			keys[i][k-1] = r.Float64() / math.Pow(float64(k), 0.8)
		}
	}

	return fromLevels(numFrames, numHarmonics, func(position float64, amps []float64) {
		x := position * float64(keyframes-1)
		i := min(int(x), keyframes-2)
		frac := x - float64(i)
		frac = 0.5 - 0.5*math.Cos(math.Pi*frac) // ease in and out of every keyframe
		for k := range numHarmonics {
			amps[k] = keys[i][k] + (keys[i+1][k]-keys[i][k])*frac
		}
	})
}

// saw is a sawtooth cycle rising from -1 to 1
func saw(phase float64) float64 {
	return 2*(phase-math.Floor(phase)) - 1
}

// Filter sweeps a resonant lowpass over a sawtooth: the cutoff rises from the fundamental to the
// 128th harmonic, q (0.7 is smooth, 4 sharp, 10 whistling) sets the resonance peak at the cutoff
func Filter(numFrames int, q float64) *wavetable.Table {
	q = max(0.5, q)
	return wavetable.FromHarmonics(numFrames, FrameSize, func(position float64, amps []float64) {
		cutoff := math.Pow(128, position) // in harmonics
		for k := 1; k <= len(amps); k++ {
			r := float64(k) / cutoff
			// a two pole lowpass
			gain := 1 / math.Sqrt((1-r*r)*(1-r*r)+r*r/(q*q))
			amps[k-1] = gain / float64(k)
		}
	})
}

// Resonance is the resonant waveform of phase distortion synths (Casio CZ): a cosine at ratio
// times the cycle under a falling sawtooth window, the ratio rising from 1 to maxRatio. It sounds
// like a resonant filter sweeping up, without a filter
func Resonance(numFrames int, maxRatio float64) *wavetable.Table {
	maxRatio = max(1, maxRatio)
	return wavetable.FromFunc(numFrames, FrameSize, func(position float64, phase float64) float64 {
		ratio := 1 + position*(maxRatio-1)
		return (1 - phase) * math.Cos(2*math.Pi*ratio*phase)
	})
}

// OddEven moves the balance between the odd and even harmonics of a sawtooth: at 0 only odd
// harmonics (a square like, hollow tone), at 0.5 all of them (a sawtooth), at 1 the fundamental
// with only even harmonics (an open, octave like tone)
func OddEven(numFrames int) *wavetable.Table {
	return wavetable.FromHarmonics(numFrames, FrameSize, func(position float64, amps []float64) {
		odd := min(1, 2-2*position)
		even := min(1, 2*position)
		amps[0] = 1
		for k := 2; k <= len(amps); k++ {
			w := odd
			if k%2 == 0 {
				w = even
			}
			amps[k-1] = w / float64(k)
		}
	})
}

// Comb mixes a sawtooth with a copy of itself delayed from 0 to maxDelay (0-0.5) of the cycle, a
// comb filter that cuts notches through the harmonics like a flanger. At a delay of half the cycle
// the odd harmonics cancel and the tone jumps an octave
func Comb(numFrames int, maxDelay float64) *wavetable.Table {
	maxDelay = max(0, min(maxDelay, 0.5))
	return wavetable.FromFunc(numFrames, FrameSize, func(position float64, phase float64) float64 {
		d := position * maxDelay
		return 0.5 * (saw(phase) + saw(phase-d))
	})
}

// Saturate drives a sine into a soft clipper (tanh), the drive rising from 1 to maxDrive: from a
// sine through a warm, rounded tone towards a square
func Saturate(numFrames int, maxDrive float64) *wavetable.Table {
	maxDrive = max(1, maxDrive)
	return wavetable.FromFunc(numFrames, FrameSize, func(position float64, phase float64) float64 {
		drive := 1 + position*(maxDrive-1)
		return math.Tanh(drive*math.Sin(2*math.Pi*phase)) / math.Tanh(drive)
	})
}

// Quantize reduces the bit depth of a sine: the number of levels falls from 256 (8 bits) to
// minLevels (2 is a square), the gritty sound of early digital audio
func Quantize(numFrames int, minLevels int) *wavetable.Table {
	minLevels = max(2, minLevels)
	return wavetable.FromFunc(numFrames, FrameSize, func(position float64, phase float64) float64 {
		levels := math.Round(256 * math.Pow(float64(minLevels)/256, position))
		x := (math.Sin(2*math.Pi*phase) + 1) / 2 * (levels - 1)
		return math.Round(x)/(levels-1)*2 - 1
	})
}

// Steps reduces the sample rate of a sine: the cycle is held in steps, from 256 steps down to
// minSteps (for example 4). The steps mirror the harmonics around multiples of the step count,
// the metallic stair case sound of the first wavetable synths
func Steps(numFrames int, minSteps int) *wavetable.Table {
	minSteps = max(2, minSteps)
	return wavetable.FromFunc(numFrames, FrameSize, func(position float64, phase float64) float64 {
		steps := math.Round(256 * math.Pow(float64(minSteps)/256, position))
		return math.Sin(2 * math.Pi * math.Floor(phase*steps) / steps)
	})
}

// Shapes for Crossfade, single cycles from -1 to 1
var (
	Sine     = func(phase float64) float64 { return math.Sin(2 * math.Pi * phase) }
	Triangle = func(phase float64) float64 { return 1 - 4*math.Abs(math.Mod(phase+0.25, 1)-0.5) }
	Saw      = func(phase float64) float64 { return saw(phase + 0.5) }
	Square   = func(phase float64) float64 {
		if phase < 0.5 {
			return 1
		}
		return -1
	}
)

// Crossfade morphs through the shapes in turn (functions of the phase 0-1 of a cycle, like Sine,
// Triangle, Saw and Square or your own), crossfading between neighbours
func Crossfade(numFrames int, shapes ...func(phase float64) float64) *wavetable.Table {
	if len(shapes) == 0 {
		shapes = append(shapes, Sine)
	}
	if len(shapes) == 1 {
		shapes = append(shapes, shapes[0])
	}
	return wavetable.FromFunc(numFrames, FrameSize, func(position float64, phase float64) float64 {
		x := position * float64(len(shapes)-1)
		i := min(int(x), len(shapes)-2)
		frac := x - float64(i)
		return shapes[i](phase)*(1-frac) + shapes[i+1](phase)*frac
	})
}

// loopFade is the number of samples at the end of a sliced frame faded into its start
const loopFade = 128

// silenceLevel is the loudness (relative to the loudest part) below which FromSound skips a part
// of the recording
const silenceLevel = 0.1

// FromSound slices a recording into numFrames frames taken evenly through the parts that sound
// (silence, at the ends or in pauses, is skipped), so moving the position plays through the
// recording at the pitch of the note (the fixed frame import of wavetable synths). Every slice is
// looped with a short crossfade, so it doesn't click where the cycle wraps. A slice is not a single
// cycle of the recording, expect a buzzy, vocal or noisy colour more than the original sound
func FromSound(samples []float32, numFrames int) *wavetable.Table {
	numFrames = max(1, numFrames)
	need := FrameSize + loopFade
	if len(samples) < need {
		padded := make([]float32, need)
		copy(padded, samples)
		samples = padded
	}

	// candidate slices, keep the ones that sound
	numCandidates := max(numFrames*8, 64)
	last := len(samples) - need
	type candidate struct {
		offset int
		rms    float64
	}
	candidates := make([]candidate, 0, numCandidates)
	loudest := 0.0
	for c := range numCandidates {
		offset := 0
		if numCandidates > 1 {
			offset = c * last / (numCandidates - 1)
		}
		sum := 0.0
		for _, x := range samples[offset : offset+need] {
			sum += float64(x) * float64(x)
		}
		r := math.Sqrt(sum / float64(need))
		candidates = append(candidates, candidate{offset, r})
		loudest = max(loudest, r)
	}
	var sounding []int
	for _, c := range candidates {
		if c.rms >= silenceLevel*loudest {
			sounding = append(sounding, c.offset)
		}
	}
	if len(sounding) == 0 { // all silent
		sounding = []int{0}
	}

	frames := make([]float32, numFrames*FrameSize)
	for i := range numFrames {
		offset := sounding[0]
		if numFrames > 1 {
			offset = sounding[i*(len(sounding)-1)/(numFrames-1)]
		}
		seg := samples[offset : offset+need]
		frame := frames[i*FrameSize : (i+1)*FrameSize]
		copy(frame, seg)
		// the start crossfades from what follows the end, so the end runs smoothly into the start
		for j := range loopFade {
			w := float32(j) / loopFade
			frame[j] = seg[FrameSize+j]*(1-w) + seg[j]*w
		}
	}
	t, err := wavetable.NewTable(frames, FrameSize)
	if err != nil {
		panic(err)
	}
	return t
}

// LoadSound slices a sound file into a table with FromSound (stereo files are mixed to mono)
func LoadSound(path string, numFrames int) (*wavetable.Table, error) {
	sf, err := sndfile.NewSoundFile[float32](path)
	if err != nil {
		return nil, err
	}
	mono := make([]float32, sf.NumFrames())
	for c := range sf.NumChannels() {
		for i, x := range sf.Buffer(c, 0) {
			mono[i] += x / float32(sf.NumChannels())
		}
	}
	return FromSound(mono, numFrames), nil
}
