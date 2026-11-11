package convolver

import (
	"math"
	"math/rand/v2"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/fft"
	"github.com/almerlucke/psalm/blocks/mix"
	"github.com/almerlucke/sndfile"
)

// Config for the convolver
type Config struct {
	PartitionSize int     // block size, power of 2, the latency of the wet signal in samples
	Mix           float64 // 0 is dry only, 1 is wet only, an equal power crossfade that keeps the loudness
	Gain          float64 // wet gain
	MonoInput     bool    // both sides get the same input (a mono send to a stereo response): it is transformed once
}

func DefaultConfig() Config {
	return Config{
		PartitionSize: 512,
		Mix:           1.0,
		Gain:          1.0,
	}
}

// maxBlock is the largest block of a segment: larger ones convolve a long tail with fewer
// multiplications, but do more work at once
const maxBlock = 8192

// segmentGrowth is how much larger every next block size is: each size used a few times (three
// for 4) costs fewer transforms than doubling (every size costs about the same in transforms per
// sample, the multiplications are cheap)
const segmentGrowth = 4

// trimDB is where an impulse response is cut: the energy left after it is this far below all of it
const trimDB = -90.0

// segment is a part of the impulse response convolved with one block size (uniformly partitioned
// overlap-save). The first segment has the block size of the convolver, the next ones grow it (see
// segmentGrowth): a segment of block L can start at L - PartitionSize into the response, its
// output (L late) is then just in time. So the head of the response is convolved in small blocks (low latency) and the
// long tail in large ones (few multiplications)
type segment struct {
	block    int // L
	parts    int
	fft      *fft.Real
	pos      int
	channels []*segmentChannel
	accRe    []float64 // the bins being accumulated
	accIm    []float64
	time     []float64 // 2L samples
}

// segmentChannel holds the partitions of the response and the frequency domain delay line of one
// channel of a segment. Only the bins 0..L are stored, the spectra of real signals are conjugate
// symmetric
type segmentChannel struct {
	hRe, hIm     [][]float64 // impulse response partitions
	fdlRe, fdlIm [][]float64 // input spectra of the latest blocks
	fdlPos       int
	window       []float64 // the last two input blocks
	inBlock      []float64
	outBlock     []float64
}

// Convolver convolves its (stereo) input with an impulse response using non-uniformly partitioned
// overlap-save FFT convolution: the start of the response in blocks of PartitionSize (its only
// latency), the rest in blocks that double up to maxBlock, so even long responses (reverbs) at a
// low latency cost little. The inaudible end of the response (below trimDB) is left out. A mono
// impulse response is used for both channels, a stereo one maps left to left and right to right
type Convolver struct {
	*psalm.Module
	cfg      Config
	block    int
	segments []*segment
	irLength int          // samples, after trimming
	mix      mix.Smoother // mix and gain glide when they change
	gain     mix.Smoother
}

// New creates a convolver for the impulse response (one slice per channel, at the current sample rate)
func New(ir [][]float64, cfg Config) *Convolver {
	if cfg.PartitionSize == 0 {
		cfg.PartitionSize = DefaultConfig().PartitionSize
	}
	if len(ir) == 0 {
		ir = [][]float64{{1.0}}
	}

	c := &Convolver{
		Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}),
		cfg:    cfg,
		block:  cfg.PartitionSize,
	}

	length := trimmedLength(ir)
	chans := make([][]float64, psalm.Stereo)
	for ch := range chans {
		chans[ch] = ir[ch%len(ir)][:min(length, len(ir[ch%len(ir)]))]
	}
	c.irLength = length

	// the segments: each block size covers the response until the next, segmentGrowth times as
	// large, is in time (a block of L can start at L - PartitionSize)
	b := c.block
	for offset, block := 0, b; offset < length; block = min(block*segmentGrowth, max(b, maxBlock)) {
		end := length
		if block < max(b, maxBlock) {
			next := min(block*segmentGrowth, max(b, maxBlock))
			end = min(length, next-b) // where the next block size can start
		}
		parts := max(1, (end-offset+block-1)/block)
		c.segments = append(c.segments, newSegment(chans, offset, block, parts))
		offset += parts * block
	}

	sr := c.Config().SampleRate
	c.mix = mix.NewSmoother(max(0, min(cfg.Mix, 1)), mix.SmoothMs, sr)
	c.gain = mix.NewSmoother(cfg.Gain, mix.SmoothMs, sr)
	c.SetSelf(c)

	return c
}

// Settings returns the settings
func (c *Convolver) Settings() Config { return c.cfg }

// trimmedLength returns how much of a response is heard: up to where the energy left is trimDB
// below all of it (the loudest channel decides)
func trimmedLength(ir [][]float64) int {
	length := 1
	for _, ch := range ir {
		total := 0.0
		for _, x := range ch {
			total += x * x
		}
		left, end := 0.0, len(ch)
		limit := total * math.Pow(10, trimDB/10)
		for end > 0 && left+ch[end-1]*ch[end-1] <= limit {
			end--
			left += ch[end] * ch[end]
		}
		length = max(length, end)
	}
	return length
}

// newSegment transforms the partitions of the response from offset, parts blocks of a size
func newSegment(ir [][]float64, offset, block, parts int) *segment {
	n := 2 * block
	bins := block + 1
	s := &segment{
		block: block,
		parts: parts,
		fft:   fft.NewReal(n),
		accRe: make([]float64, bins),
		accIm: make([]float64, bins),
		time:  make([]float64, n),
	}
	padded := make([]float64, n)
	for _, h := range ir {
		ch := &segmentChannel{
			window:   make([]float64, n),
			inBlock:  make([]float64, block),
			outBlock: make([]float64, block),
		}
		for p := range parts {
			// zero padded partition, transformed once up front
			clear(padded)
			start := offset + p*block
			if start < len(h) {
				copy(padded, h[start:min(len(h), start+block)])
			}
			re, im := make([]float64, bins), make([]float64, bins)
			s.fft.Forward(padded, re, im)
			ch.hRe = append(ch.hRe, re)
			ch.hIm = append(ch.hIm, im)
			ch.fdlRe = append(ch.fdlRe, make([]float64, bins))
			ch.fdlIm = append(ch.fdlIm, make([]float64, bins))
		}
		s.channels = append(s.channels, ch)
	}
	return s
}

// Latency returns the delay of the wet signal in samples
func (c *Convolver) Latency() int {
	return c.block
}

func (c *Convolver) Receive(value any, _ int) {
	if msg, ok := psalm.MsgVal(value); ok {
		c.ReceiveMessage(msg)
	}
}

func (c *Convolver) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("mix", func(f float64) { c.cfg.Mix = f })
	msg.IfFltVal("gain", func(f float64) { c.cfg.Gain = f })
}

// process convolves the latest input block of a channel of a segment, the result replaces its
// output block
func (s *segment) process(ch, shared *segmentChannel) {
	var (
		b     = s.block
		bins  = b + 1
		parts = s.parts
	)

	// slide the input window and transform it into the delay line (or take the transform of the
	// first channel, when every channel has the same input)
	if shared != nil {
		at := (shared.fdlPos - 1 + parts) % parts // the first channel has just moved on
		copy(ch.fdlRe[ch.fdlPos], shared.fdlRe[at])
		copy(ch.fdlIm[ch.fdlPos], shared.fdlIm[at])
	} else {
		copy(ch.window, ch.window[b:])
		copy(ch.window[b:], ch.inBlock)
		s.fft.Forward(ch.window, ch.fdlRe[ch.fdlPos], ch.fdlIm[ch.fdlPos])
	}

	// multiply every delayed input spectrum with its impulse response partition and accumulate
	accRe, accIm := s.accRe[:bins], s.accIm[:bins]
	clear(accRe)
	clear(accIm)
	for p := range parts {
		idx := (ch.fdlPos - p + parts) % parts
		xr, xi := ch.fdlRe[idx][:bins], ch.fdlIm[idx][:bins]
		hr, hi := ch.hRe[p][:bins], ch.hIm[p][:bins]
		for k := range bins {
			accRe[k] += xr[k]*hr[k] - xi[k]*hi[k]
			accIm[k] += xr[k]*hi[k] + xi[k]*hr[k]
		}
	}
	ch.fdlPos = (ch.fdlPos + 1) % parts

	// overlap-save: back to time, the second half is the valid (non circular) part
	s.fft.Inverse(accRe, accIm, s.time)
	copy(ch.outBlock, s.time[b:])
}

func (c *Convolver) Synthesize() {
	ins := c.Input(0).Channels
	outs := c.Output(0).Channels
	mixTo := max(0, min(c.cfg.Mix, 1))

	for i := range c.Config().BufferSize {
		dry, wet := mix.EqualPower(c.mix.Next(mixTo))
		wet *= c.gain.Next(c.cfg.Gain)
		for ci := range outs {
			in := float64(ins[ci][i])
			y := 0.0
			for _, s := range c.segments {
				ch := s.channels[ci]
				ch.inBlock[s.pos] = in
				y += ch.outBlock[s.pos]
			}
			outs[ci][i] = float32(dry*in + wet*y)
		}

		for _, s := range c.segments {
			s.pos++
			if s.pos == s.block {
				s.pos = 0
				for ci, ch := range s.channels {
					var shared *segmentChannel
					if c.cfg.MonoInput && ci > 0 {
						shared = s.channels[0]
					}
					s.process(ch, shared)
				}
			}
		}
	}
}

// IRFromSoundFile returns the channels of a sound file as impulse response, normalized to unit
// energy (see NormalizeIR), so a gain of 1 roughly keeps the input level whatever the recording.
// The sound file is not resampled, so it should have the same sample rate as the patch
func IRFromSoundFile(sf sndfile.SoundFiler[float32]) [][]float64 {
	ir := make([][]float64, sf.NumChannels())
	for ch := range ir {
		buf := sf.Buffer(ch, 0)
		ir[ch] = make([]float64, len(buf))
		for i, v := range buf {
			ir[ch][i] = float64(v)
		}
	}
	return NormalizeIR(ir)
}

// NormalizeIR scales an impulse response to unit energy (the mean of its channels, so the stereo
// balance stays): convolving keeps about the loudness of the input, as with SyntheticReverbIR.
// Recorded impulse responses differ a lot in level, normalized a gain means the same for all
func NormalizeIR(ir [][]float64) [][]float64 {
	energy := 0.0
	for _, ch := range ir {
		for _, x := range ch {
			energy += x * x
		}
	}
	if len(ir) == 0 || energy <= 0 {
		return ir
	}
	scale := 1 / math.Sqrt(energy/float64(len(ir)))
	for _, ch := range ir {
		for i := range ch {
			ch[i] *= scale
		}
	}
	return ir
}

// SyntheticReverbIR generates a stereo reverb impulse response from exponentially decaying noise
// (-60 dB after the given number of seconds). Damping (0-1) makes the tail darker over time. Every
// channel is normalized to unit energy, so the reverb roughly keeps the input level
func SyntheticReverbIR(seconds float64, damping float64, sampleRate float64, seed uint64) [][]float64 {
	numFrames := int(seconds * sampleRate)
	ir := make([][]float64, psalm.Stereo)

	for ch := range ir {
		rng := rand.New(rand.NewPCG(seed, uint64(ch)+1))
		buf := make([]float64, numFrames)
		lp := 0.0
		energy := 0.0

		for i := range buf {
			t := float64(i) / float64(numFrames)
			noise := rng.Float64()*2.0 - 1.0

			// one pole lowpass that closes as the tail decays
			coef := 1.0 - damping*t*0.95
			lp += (noise - lp) * coef

			buf[i] = lp * math.Exp(-6.9*t)
			energy += buf[i] * buf[i]
		}

		if energy > 0.0 {
			scale := 1.0 / math.Sqrt(energy)
			for i := range buf {
				buf[i] *= scale
			}
		}

		ir[ch] = buf
	}

	return ir
}

// Tail lets the convolver rest while silent: sound can come out up to the length of the impulse
// response (which can hold silent gaps before late reflections) after the input stopped
func (c *Convolver) Tail() float64 {
	return float64(c.irLength+c.block) / c.Config().SampleRate
}

// SignalLatency: a fully wet convolver (a cabinet, a creative convolution) is late as a whole, the
// mixer lines the other strips up with it. Mixed with its dry signal the latency is a pre delay
// of the reverb, nothing to line up
func (c *Convolver) SignalLatency() int {
	if c.cfg.Mix >= 1 {
		return c.block
	}
	return 0
}
