package fdnreverb

import (
	"math"

	"github.com/almerlucke/psalm"
)

// Config for the FDN reverb
type Config struct {
	Size      float64 // 0-1, scales the delay line lengths (room size)
	Decay     float64 // reverb time (RT60) in seconds at low frequencies
	Damping   float64 // 0-1, how much faster high frequencies decay (0 is as long as the lows)
	PreDelay  float64 // milliseconds before the reverb starts
	Diffusion float64 // 0-1, input smearing, higher is a softer, denser attack
	ModDepth  float64 // delay modulation depth in milliseconds, removes metallic ringing
	ModRate   float64 // delay modulation rate in Hz
	Width     float64 // 0 is mono, 1 is full stereo
	Mix       float64 // 0 is dry only, 1 is wet only
}

func DefaultConfig() Config {
	return Config{
		Size:      0.6,
		Decay:     2.5,
		Damping:   0.5,
		PreDelay:  20.0,
		Diffusion: 0.7,
		ModDepth:  0.4,
		ModRate:   0.6,
		Width:     1.0,
		Mix:       0.3,
	}
}

const numLines = 8

// base delay line lengths in milliseconds, chosen to be mutually prime-ish so echoes don't line up
var baseLengths = [numLines]float64{29.7, 37.1, 41.1, 43.7, 53.3, 59.9, 67.7, 71.9}

// base input diffuser lengths in milliseconds
var diffuserLengths = [4]float64{3.2, 2.4, 8.6, 6.3}

// output sign patterns for left and right, orthogonal so both channels are decorrelated
var leftSigns = [numLines]float64{1, 1, 1, 1, -1, -1, -1, -1}
var rightSigns = [numLines]float64{1, -1, 1, -1, 1, -1, 1, -1}

// delayLine is a ring buffer with linearly or allpass interpolated reads
type delayLine struct {
	buf    []float64
	pos    int
	apPrev float64 // previous output of the allpass interpolator
}

func newDelayLine(maxSamples int) *delayLine {
	return &delayLine{buf: make([]float64, maxSamples+4)}
}

func (d *delayLine) read(delay float64) float64 {
	n := len(d.buf)
	rp := float64(d.pos) - delay
	for rp < 0 {
		rp += float64(n)
	}
	i := int(rp)
	frac := rp - float64(i)
	a := d.buf[i%n]
	b := d.buf[(i+1)%n]
	return a + (b-a)*frac
}

func (d *delayLine) at(delay int) float64 {
	n := len(d.buf)
	return d.buf[((d.pos-delay)%n+n)%n]
}

// readAllpass reads a fractional delay with first order allpass interpolation, which (unlike
// linear interpolation) doesn't attenuate high frequencies, so a modulated feedback loop doesn't
// lose energy on every pass. The fractional part is kept in 0.1-1.1 so the coefficient stays stable
func (d *delayLine) readAllpass(delay float64) float64 {
	i := int(math.Floor(delay - 0.1))
	frac := delay - float64(i)
	a := (1.0 - frac) / (1.0 + frac)

	y := a*(d.at(i)-d.apPrev) + d.at(i+1)
	d.apPrev = y
	return y
}

func (d *delayLine) write(x float64) {
	d.buf[d.pos] = x
	d.pos = (d.pos + 1) % len(d.buf)
}

// allpass is a Schroeder allpass diffuser
type allpass struct {
	line  *delayLine
	delay float64
	gain  float64
}

func (a *allpass) process(x float64) float64 {
	delayed := a.line.read(a.delay)
	v := x + a.gain*delayed
	a.line.write(v)
	return delayed - a.gain*v
}

// absorption is the per line damping filter H(z) = b / (1 - a z^-1), with an exact gain at DC
// (low frequency decay) and at nyquist (high frequency decay)
type absorption struct {
	a, b, y float64
}

func (f *absorption) set(gainLow float64, gainHigh float64) {
	f.a = (gainLow - gainHigh) / (gainLow + gainHigh)
	f.b = gainLow * (1.0 - f.a)
}

func (f *absorption) process(x float64) float64 {
	f.y = f.b*x + f.a*f.y
	return f.y
}

// FDNReverb is an 8 line feedback delay network reverb: modulated delay lines mixed through an
// orthogonal Hadamard matrix (lossless), with per line damping filters tuned so the tail decays in
// Decay seconds at low frequencies and faster at high frequencies. Stereo input, stereo output
type FDNReverb struct {
	*psalm.Module
	cfg       Config
	sr        float64
	preDelay  [2]*delayLine
	diffusers [2][4]*allpass
	lines     [numLines]*delayLine
	filters   [numLines]absorption
	lengths   [numLines]float64 // delay in samples
	modPhase  [numLines]float64
	modInc    [numLines]float64
	modDepth  float64
	feedback  [numLines]float64
	out       [numLines]float64
	preDelayN float64 // pre-delay in samples
	width     float64
}

func New(cfg Config) *FDNReverb {
	r := &FDNReverb{
		Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}),
		cfg:    cfg,
	}

	r.sr = r.Config().SampleRate
	msToSamps := r.sr * 0.001

	// allocate for the largest size, pre-delay up to 500 ms, modulation up to 5 ms
	maxModSamps := int(5.0 * msToSamps)
	for c := range 2 {
		r.preDelay[c] = newDelayLine(int(500.0 * msToSamps))
		for i := range r.diffusers[c] {
			// slightly different lengths per channel for a wider image
			length := diffuserLengths[i] * msToSamps * (1.0 + 0.07*float64(c))
			r.diffusers[c][i] = &allpass{line: newDelayLine(int(length) + 2), delay: length}
		}
	}
	for i := range numLines {
		r.lines[i] = newDelayLine(int(baseLengths[i]*1.5*msToSamps) + maxModSamps)
		r.modPhase[i] = float64(i) / numLines
	}

	r.update()
	r.SetSelf(r)

	return r
}

// update recalculates delay lengths, damping filters and modulation from the config
func (r *FDNReverb) update() {
	cfg := &r.cfg
	msToSamps := r.sr * 0.001

	size := math.Max(0.0, math.Min(cfg.Size, 1.0))
	decay := math.Max(0.05, cfg.Decay)
	damping := math.Max(0.0, math.Min(cfg.Damping, 1.0))
	decayHigh := decay * (1.0 - 0.9*damping)

	// modulated reads are centered on length + modDepth
	r.modDepth = math.Max(0.0, math.Min(cfg.ModDepth, 4.0)) * msToSamps

	for i := range numLines {
		// whole samples, so unmodulated reads are exact (interpolation would add damping every pass)
		r.lengths[i] = math.Round(baseLengths[i] * (0.5 + size) * msToSamps)

		// gain per pass so the level drops 60 dB after the decay time
		loop := r.lengths[i] + r.modDepth
		gainLow := math.Pow(10.0, -3.0*loop/(r.sr*decay))
		gainHigh := math.Pow(10.0, -3.0*loop/(r.sr*decayHigh))
		r.filters[i].set(gainLow, gainHigh)

		// every line wobbles at a slightly different rate
		r.modInc[i] = cfg.ModRate * (1.0 + 0.13*float64(i)) / r.sr
	}

	r.preDelayN = math.Max(0.0, math.Min(cfg.PreDelay, 490.0)) * msToSamps
	r.width = math.Max(0.0, math.Min(cfg.Width, 1.0))

	diffGain := 0.75 * math.Max(0.0, math.Min(cfg.Diffusion, 1.0))
	for c := range 2 {
		for _, ap := range r.diffusers[c] {
			ap.gain = diffGain
		}
	}
}

func (r *FDNReverb) Receive(value any, _ int) {
	if msg, ok := psalm.MsgVal(value); ok {
		r.ReceiveMessage(msg)
	}
}

func (r *FDNReverb) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("size", func(f float64) { r.cfg.Size = f })
	msg.IfFltVal("decay", func(f float64) { r.cfg.Decay = f })
	msg.IfFltVal("damping", func(f float64) { r.cfg.Damping = f })
	msg.IfFltVal("preDelay", func(f float64) { r.cfg.PreDelay = f })
	msg.IfFltVal("diffusion", func(f float64) { r.cfg.Diffusion = f })
	msg.IfFltVal("modDepth", func(f float64) { r.cfg.ModDepth = f })
	msg.IfFltVal("modRate", func(f float64) { r.cfg.ModRate = f })
	msg.IfFltVal("width", func(f float64) { r.cfg.Width = f })
	msg.IfFltVal("mix", func(f float64) { r.cfg.Mix = f })
	r.update()
}

// hadamard8 applies the 8x8 Hadamard matrix in place with a fast Walsh-Hadamard transform,
// scaled by 1/sqrt(8) so it is orthogonal (energy preserving)
func hadamard8(x *[numLines]float64) {
	for h := 1; h < numLines; h <<= 1 {
		for i := 0; i < numLines; i += h << 1 {
			for j := i; j < i+h; j++ {
				a, b := x[j], x[j+h]
				x[j], x[j+h] = a+b, a-b
			}
		}
	}
	const scale = 0.35355339059327373 // 1 / sqrt(8)
	for i := range x {
		x[i] *= scale
	}
}

// ProcessSample runs one stereo sample through the reverb and returns the wet (reverberated)
// signal only, useful to build other effects around the reverb
func (r *FDNReverb) ProcessSample(inL float64, inR float64) (float64, float64) {
	const outScale = 0.35355339059327373 // 1 / sqrt(8)

	var in [2]float64

	// pre-delay and input diffusion per channel
	for c, x := range [2]float64{inL, inR} {
		r.preDelay[c].write(x)
		if r.preDelayN > 0.0 {
			x = r.preDelay[c].read(r.preDelayN + 1.0)
		}
		for _, ap := range r.diffusers[c] {
			x = ap.process(x)
		}
		in[c] = x
	}

	// read the modulated delay lines through their damping filters
	for l := range numLines {
		mod := math.Sin(2.0*math.Pi*r.modPhase[l]) * r.modDepth
		r.modPhase[l] += r.modInc[l]
		if r.modPhase[l] >= 1.0 {
			r.modPhase[l] -= 1.0
		}
		r.out[l] = r.filters[l].process(r.lines[l].readAllpass(r.lengths[l] + mod + r.modDepth))
	}

	// stereo outputs from orthogonal sign patterns
	left, right := 0.0, 0.0
	for l := range numLines {
		left += r.out[l] * leftSigns[l]
		right += r.out[l] * rightSigns[l]
	}
	left *= outScale
	right *= outScale

	mid := (left + right) * 0.5
	side := (left - right) * 0.5 * r.width

	// mix the lines and feed them back with the input (left into even, right into odd lines)
	r.feedback = r.out
	hadamard8(&r.feedback)
	for l := range numLines {
		r.lines[l].write(r.feedback[l] + in[l%2]*0.5)
	}

	return mid + side, mid - side
}

func (r *FDNReverb) Synthesize() {
	ins := r.Input(0).Channels
	outs := r.Output(0).Channels
	wet, dry := r.cfg.Mix, 1.0-r.cfg.Mix

	for i := range r.Config().BufferSize {
		inL, inR := float64(ins[0][i]), float64(ins[1][i])
		left, right := r.ProcessSample(inL, inR)
		outs[0][i] = float32(dry*inL + wet*left)
		outs[1][i] = float32(dry*inR + wet*right)
	}
}
