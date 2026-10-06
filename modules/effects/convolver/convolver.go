package convolver

import (
	"math"
	"math/rand/v2"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/fft"
	"github.com/almerlucke/sndfile"
)

// Config for the convolver
type Config struct {
	PartitionSize int     // block size, power of 2, the latency of the wet signal in samples
	Mix           float64 // 0 is dry only, 1 is wet only
	Gain          float64 // wet gain
}

func DefaultConfig() Config {
	return Config{
		PartitionSize: 512,
		Mix:           1.0,
		Gain:          1.0,
	}
}

// channel holds the partitioned impulse response spectra and the frequency domain delay line of
// one channel. Only the bins 0..n/2 are stored, the spectra of real signals are conjugate symmetric
type channel struct {
	hRe, hIm     [][]float64 // impulse response partitions
	fdlRe, fdlIm [][]float64 // input spectra of the latest blocks
	fdlPos       int
	window       []float64 // the last two input blocks
	inBlock      []float64
	outBlock     []float64
}

// Convolver convolves its (stereo) input with an impulse response using uniformly partitioned
// overlap-save FFT convolution, so long impulse responses (like reverbs) only add one partition of
// latency. A mono impulse response is used for both channels, a stereo one maps left to left and
// right to right
type Convolver struct {
	*psalm.Module
	cfg      Config
	fft      *fft.FFT
	block    int
	pos      int
	channels []*channel
	re, im   []float64
	irLength int // samples
}

// New creates a convolver for the impulse response (one slice per channel, at the current sample rate)
func New(ir [][]float64, cfg Config) *Convolver {
	if cfg.PartitionSize == 0 {
		cfg.PartitionSize = DefaultConfig().PartitionSize
	}
	if len(ir) == 0 {
		ir = [][]float64{{1.0}}
	}

	b := cfg.PartitionSize
	n := 2 * b

	c := &Convolver{
		Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}),
		cfg:    cfg,
		fft:    fft.New(n),
		block:  b,
		re:     make([]float64, n),
		im:     make([]float64, n),
	}

	for ch := range psalm.Stereo {
		c.channels = append(c.channels, c.newChannel(ir[ch%len(ir)]))
		c.irLength = max(c.irLength, len(ir[ch%len(ir)]))
	}

	c.SetSelf(c)

	return c
}

func (c *Convolver) newChannel(ir []float64) *channel {
	var (
		b     = c.block
		n     = 2 * b
		bins  = n/2 + 1
		parts = max(1, (len(ir)+b-1)/b)
		ch    = &channel{
			window:   make([]float64, n),
			inBlock:  make([]float64, b),
			outBlock: make([]float64, b),
		}
	)

	for p := range parts {
		// zero padded partition, transformed once up front
		clear(c.re)
		clear(c.im)
		end := min(len(ir), (p+1)*b)
		if p*b < end {
			copy(c.re, ir[p*b:end])
		}
		c.fft.Forward(c.re, c.im)

		ch.hRe = append(ch.hRe, append([]float64(nil), c.re[:bins]...))
		ch.hIm = append(ch.hIm, append([]float64(nil), c.im[:bins]...))
		ch.fdlRe = append(ch.fdlRe, make([]float64, bins))
		ch.fdlIm = append(ch.fdlIm, make([]float64, bins))
	}

	return ch
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

// process convolves the latest input block of a channel, the result replaces its output block
func (c *Convolver) process(ch *channel) {
	var (
		b     = c.block
		n     = 2 * b
		bins  = n/2 + 1
		parts = len(ch.hRe)
	)

	// slide the input window and transform it
	copy(ch.window, ch.window[b:])
	copy(ch.window[b:], ch.inBlock)
	copy(c.re, ch.window)
	clear(c.im)
	c.fft.Forward(c.re, c.im)

	copy(ch.fdlRe[ch.fdlPos], c.re[:bins])
	copy(ch.fdlIm[ch.fdlPos], c.im[:bins])

	// multiply every delayed input spectrum with its impulse response partition and accumulate
	clear(c.re)
	clear(c.im)
	for p := range parts {
		idx := (ch.fdlPos - p + parts) % parts
		xr, xi := ch.fdlRe[idx], ch.fdlIm[idx]
		hr, hi := ch.hRe[p], ch.hIm[p]
		for k := range bins {
			c.re[k] += xr[k]*hr[k] - xi[k]*hi[k]
			c.im[k] += xr[k]*hi[k] + xi[k]*hr[k]
		}
	}
	ch.fdlPos = (ch.fdlPos + 1) % parts

	// restore the conjugate symmetric half and go back to the time domain
	for k := 1; k < n/2; k++ {
		c.re[n-k] = c.re[k]
		c.im[n-k] = -c.im[k]
	}
	c.fft.Inverse(c.re, c.im)

	// overlap-save: the second half is the valid (non circular) part
	copy(ch.outBlock, c.re[b:])
}

func (c *Convolver) Synthesize() {
	ins := c.Input(0).Channels
	outs := c.Output(0).Channels
	wet := c.cfg.Mix * c.cfg.Gain
	dry := 1.0 - c.cfg.Mix

	for i := range c.Config().BufferSize {
		for ci, ch := range c.channels {
			in := float64(ins[ci][i])
			ch.inBlock[c.pos] = in
			outs[ci][i] = float32(dry*in + wet*ch.outBlock[c.pos])
		}

		c.pos++
		if c.pos == c.block {
			c.pos = 0
			for _, ch := range c.channels {
				c.process(ch)
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
