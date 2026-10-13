package vocoder

import (
	"math"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/fft"
)

// Config for the FFT vocoder
type Config struct {
	FFTSize       int     // frame size, power of 2 (latency is one frame)
	Overlap       int     // frames per FFT size, hop = FFTSize / Overlap
	NumBands      int     // number of log spaced analysis bands
	MinFreq       float64 // lowest band (Hz)
	MaxFreq       float64 // highest band (Hz)
	Attack        float64 // band envelope attack in milliseconds
	Release       float64 // band envelope release in milliseconds
	Flatten       float64 // 0-1, how much the carrier's own spectral shape is removed before imprinting
	Sibilance     float64 // amount of the modulator passed straight through above SibilanceFreq (while a carrier plays)
	SibilanceFreq float64 // Hz
	FormantShift  float64 // scales the modulator envelope in frequency, > 1 shifts formants up
	Gain          float64
}

func DefaultConfig() Config {
	return Config{
		FFTSize:       1024,
		Overlap:       4,
		NumBands:      32,
		MinFreq:       80.0,
		MaxFreq:       10000.0,
		Attack:        5.0,
		Release:       50.0,
		Flatten:       1.0,
		Sibilance:     0.25,
		SibilanceFreq: 5000.0,
		FormantShift:  1.0,
		Gain:          1.0,
	}
}

// bandPos is a band index plus fraction to interpolate between neighbouring bands
type bandPos struct {
	index int
	frac  float64
}

// Vocoder imprints the spectral envelope of the modulator (input 0, e.g. a voice) on the carrier
// (input 1, e.g. a synth) using windowed FFT's with overlap-add resynthesis
type Vocoder struct {
	*psalm.Module
	cfg Config

	fft      *fft.FFT
	window   []float64
	hop      int
	olaScale float64

	// input rings, frame buffers and overlap-add output
	modIn    []float64
	carIn    []float64
	inPos    int
	hopCount int
	modRe    []float64
	modIm    []float64
	carRe    []float64
	carIm    []float64
	outAccum []float64
	outQueue []float64
	outPos   int

	// bands
	binLo      []int
	binHi      []int
	modBands   []float64 // smoothed modulator band envelope
	carBands   []float64
	modBinPos  []bandPos // per bin, formant shifted
	carBinPos  []bandPos // per bin
	attackCoef float64
	relCoef    float64
}

func New(cfg Config) *Vocoder {
	def := DefaultConfig()
	if cfg.FFTSize == 0 {
		cfg.FFTSize = def.FFTSize
	}
	if cfg.Overlap == 0 {
		cfg.Overlap = def.Overlap
	}
	if cfg.NumBands < 2 {
		cfg.NumBands = def.NumBands
	}
	if cfg.MinFreq <= 0.0 {
		cfg.MinFreq = def.MinFreq
	}
	if cfg.MaxFreq <= cfg.MinFreq {
		cfg.MaxFreq = def.MaxFreq
	}
	if cfg.FormantShift <= 0.0 {
		cfg.FormantShift = 1.0
	}

	n := cfg.FFTSize

	v := &Vocoder{
		Module:   psalm.NewModule([]int{psalm.Mono, psalm.Mono}, []int{psalm.Stereo}),
		cfg:      cfg,
		fft:      fft.New(n),
		window:   fft.HannWindow(n),
		hop:      n / cfg.Overlap,
		modIn:    make([]float64, n),
		carIn:    make([]float64, n),
		modRe:    make([]float64, n),
		modIm:    make([]float64, n),
		carRe:    make([]float64, n),
		carIm:    make([]float64, n),
		outAccum: make([]float64, n),
	}

	v.outQueue = make([]float64, v.hop)

	// Hann analysis + synthesis windows overlap-add to a constant, normalize it to 1
	sum := 0.0
	for _, w := range v.window {
		sum += w * w
	}
	v.olaScale = float64(v.hop) / sum

	v.setupBands()
	v.updateEnvelopeCoefs()

	v.SetSelf(v)

	return v
}

func (v *Vocoder) sampleRate() float64 {
	return v.Config().SampleRate
}

func (v *Vocoder) setupBands() {
	var (
		cfg   = v.cfg
		n     = cfg.FFTSize
		nb    = cfg.NumBands
		binHz = v.sampleRate() / float64(n)
		ratio = cfg.MaxFreq / cfg.MinFreq
		half  = n / 2
	)

	v.binLo = make([]int, nb)
	v.binHi = make([]int, nb)
	v.modBands = make([]float64, nb)
	v.carBands = make([]float64, nb)

	for b := range nb {
		lo := cfg.MinFreq * math.Pow(ratio, float64(b)/float64(nb))
		hi := cfg.MinFreq * math.Pow(ratio, float64(b+1)/float64(nb))

		loBin := max(1, min(half, int(math.Floor(lo/binHz))))
		hiBin := max(1, min(half+1, int(math.Ceil(hi/binHz))))
		if hiBin <= loBin {
			hiBin = loBin + 1
		}

		v.binLo[b] = loBin
		v.binHi[b] = hiBin
	}

	v.carBinPos = v.binPositions(1.0)
	v.modBinPos = v.binPositions(cfg.FormantShift)
}

// binPositions maps every bin (up to nyquist) to a fractional band position, band centers are
// geometric so the position is linear in log frequency. shift scales the frequency looked up
func (v *Vocoder) binPositions(shift float64) []bandPos {
	var (
		cfg     = v.cfg
		n       = cfg.FFTSize
		nb      = cfg.NumBands
		binHz   = v.sampleRate() / float64(n)
		ratio   = cfg.MaxFreq / cfg.MinFreq
		center0 = cfg.MinFreq * math.Pow(ratio, 0.5/float64(nb))
		logSpan = math.Log(ratio) * float64(nb-1) / float64(nb) // log(lastCenter / firstCenter)
		pos     = make([]bandPos, n/2+1)
	)

	for k := range pos {
		freq := math.Max(float64(k), 0.5) * binHz / shift
		p := math.Log(freq/center0) / logSpan * float64(nb-1)
		p = math.Max(0.0, math.Min(float64(nb-1), p))

		idx := int(p)
		if idx >= nb-1 {
			idx = nb - 2
		}

		pos[k] = bandPos{index: idx, frac: p - float64(idx)}
	}

	return pos
}

func (v *Vocoder) updateEnvelopeCoefs() {
	hopSec := float64(v.hop) / v.sampleRate()

	coef := func(ms float64) float64 {
		if ms <= 0.0 {
			return 1.0
		}
		return 1.0 - math.Exp(-hopSec/(ms*0.001))
	}

	v.attackCoef = coef(v.cfg.Attack)
	v.relCoef = coef(v.cfg.Release)
}

// Settings returns the settings
func (v *Vocoder) Settings() Config { return v.cfg }

func (v *Vocoder) SetFormantShift(shift float64) {
	if shift <= 0.0 {
		return
	}
	v.cfg.FormantShift = shift
	v.modBinPos = v.binPositions(shift)
}

func (v *Vocoder) SetAttack(ms float64) {
	v.cfg.Attack = ms
	v.updateEnvelopeCoefs()
}

func (v *Vocoder) SetRelease(ms float64) {
	v.cfg.Release = ms
	v.updateEnvelopeCoefs()
}

func (v *Vocoder) Receive(value any, _ int) {
	if msg, ok := psalm.MsgVal(value); ok {
		v.ReceiveMessage(msg)
	}
}

func (v *Vocoder) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("flatten", func(f float64) { v.cfg.Flatten = f })
	msg.IfFltVal("sibilance", func(f float64) { v.cfg.Sibilance = f })
	msg.IfFltVal("sibilanceFreq", func(f float64) { v.cfg.SibilanceFreq = f })
	msg.IfFltVal("gain", func(f float64) { v.cfg.Gain = f })
	msg.IfFltVal("attack", v.SetAttack)
	msg.IfFltVal("release", v.SetRelease)
	msg.IfFltVal("formantShift", v.SetFormantShift)
}

// Latency returns the delay of the vocoder in samples
func (v *Vocoder) Latency() int {
	return v.cfg.FFTSize
}

func lerpBands(bands []float64, p bandPos) float64 {
	return bands[p.index] + (bands[p.index+1]-bands[p.index])*p.frac
}

func (v *Vocoder) bandEnergies(re []float64, im []float64, bands []float64) {
	for b := range bands {
		sum := 0.0
		for k := v.binLo[b]; k < v.binHi[b]; k++ {
			sum += re[k]*re[k] + im[k]*im[k]
		}
		bands[b] = math.Sqrt(sum / float64(v.binHi[b]-v.binLo[b]))
	}
}

func (v *Vocoder) processFrame() {
	var (
		cfg  = &v.cfg
		n    = cfg.FFTSize
		half = n / 2
	)

	// Window the latest frame of both inputs (oldest sample is at inPos)
	for j := range n {
		idx := (v.inPos + j) % n
		w := v.window[j]
		v.modRe[j] = v.modIn[idx] * w
		v.modIm[j] = 0.0
		v.carRe[j] = v.carIn[idx] * w
		v.carIm[j] = 0.0
	}

	v.fft.Forward(v.modRe, v.modIm)
	v.fft.Forward(v.carRe, v.carIm)

	// Band envelopes, the modulator envelope is smoothed over time
	modRaw := v.carBands // reuse as scratch before computing the carrier bands
	v.bandEnergies(v.modRe, v.modIm, modRaw)
	for b, e := range modRaw {
		coef := v.relCoef
		if e > v.modBands[b] {
			coef = v.attackCoef
		}
		v.modBands[b] += (e - v.modBands[b]) * coef
	}

	v.bandEnergies(v.carRe, v.carIm, v.carBands)

	carAvg := 0.0
	for _, e := range v.carBands {
		carAvg += e
	}
	carAvg /= float64(len(v.carBands))

	const eps = 1e-3

	binHz := v.sampleRate() / float64(n)
	sibBin := int(math.Ceil(cfg.SibilanceFreq / binHz))

	// the consonants pass only while there is a carrier to vocode: fully above -60 dB, fading
	// out below it, so a vocoder without a carrier is silent
	carPeak := 0.0
	for _, x := range v.carIn {
		carPeak = math.Max(carPeak, math.Abs(x))
	}
	sibilance := cfg.Sibilance * math.Min(1, carPeak/0.001)

	// Imprint: flatten the carrier by its own envelope and apply the modulator envelope, the result
	// goes into the carrier buffers
	for k := 0; k <= half; k++ {
		modEnv := lerpBands(v.modBands, v.modBinPos[k])
		carEnv := lerpBands(v.carBands, v.carBinPos[k])
		ref := (1.0-cfg.Flatten)*carAvg + cfg.Flatten*carEnv
		g := cfg.Gain * modEnv / (ref + eps)

		re := v.carRe[k] * g
		im := v.carIm[k] * g

		if sibilance > 0.0 && k >= sibBin {
			re += v.modRe[k] * sibilance * cfg.Gain
			im += v.modIm[k] * sibilance * cfg.Gain
		}

		v.carRe[k] = re
		v.carIm[k] = im

		// keep the spectrum conjugate symmetric so the output is real
		if k > 0 && k < half {
			v.carRe[n-k] = re
			v.carIm[n-k] = -im
		}
	}

	v.fft.Inverse(v.carRe, v.carIm)

	// Overlap-add with the synthesis window
	for j := range n {
		v.outAccum[j] += v.carRe[j] * v.window[j] * v.olaScale
	}

	copy(v.outQueue, v.outAccum[:v.hop])
	copy(v.outAccum, v.outAccum[v.hop:])
	clear(v.outAccum[n-v.hop:])
	v.outPos = 0
}

func (v *Vocoder) Synthesize() {
	var (
		mod  = v.Input(0).Channels[0]
		car  = v.Input(1).Channels[0]
		outs = v.Output(0).Channels
		n    = v.cfg.FFTSize
	)

	for i := range v.Config().BufferSize {
		v.modIn[v.inPos] = float64(mod[i])
		v.carIn[v.inPos] = float64(car[i])
		v.inPos = (v.inPos + 1) % n

		out := float32(v.outQueue[v.outPos])
		v.outPos++

		for _, ch := range outs {
			ch[i] = out
		}

		v.hopCount++
		if v.hopCount == v.hop {
			v.hopCount = 0
			v.processFrame()
		}
	}
}

// Tail lets the vocoder rest while both its inputs are silent: its frame and release
func (v *Vocoder) Tail() float64 {
	return float64(2*v.cfg.FFTSize)/v.sampleRate() + v.cfg.Release/1000
}

// TailInputs: the vocoder is silent when its carrier (input 1) is, whatever the voice does
func (v *Vocoder) TailInputs() []int {
	return []int{1}
}

// SignalLatency: the vocoder sounds a frame late, the mixer lines the other strips up with it
func (v *Vocoder) SignalLatency() int {
	return v.cfg.FFTSize
}
