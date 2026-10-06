package filters

import (
	"math"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/filters/biquad"
	"github.com/almerlucke/psalm/blocks/filters/formant"
	"github.com/almerlucke/psalm/blocks/filters/ladder"
	"github.com/almerlucke/psalm/blocks/filters/svf"
)

// Params holds the filter parameters, each filter type uses the ones it needs
type Params struct {
	Cutoff    float64 // Hz (not used by the formant filter)
	Resonance float64 // svf and ladder: 0-1, biquad and formant: Q
	Mode      int     // svf.Mode, ladder.Mode or biquad.Type
	GainDB    float64 // biquad peak and shelf gain
	Drive     float64 // ladder input saturation
	Vowel     float64 // formant vowel position 0-4 (a e i o u)
	Shift     float64 // formant frequency shift
}

// processor is one channel of a filter
type processor interface {
	apply(p *Params)
	process(x float64) float64
	clear()
}

// coefficient update interval in samples while parameters glide
const updateInterval = 16

// Filter is a stereo filter module wrapping one of the filter blocks. Cutoff and vowel glide
// smoothly over each buffer, so they can be modulated. Input 0 is audio, value inputs 1 (cutoff),
// 2 (resonance) and 3 (vowel) accept plain values, for example from an LFO
type Filter struct {
	*psalm.Module
	target Params
	cur    Params
	procs  []processor
}

func newFilter(params Params, newProc func(sr float64) processor) *Filter {
	f := &Filter{
		Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}),
		target: params,
		cur:    params,
	}

	sr := f.Config().SampleRate
	for range psalm.Stereo {
		proc := newProc(sr)
		proc.apply(&f.cur)
		f.procs = append(f.procs, proc)
	}

	f.SetSelf(f)

	return f
}

// NewSVF creates a state variable filter, resonance 0-1
func NewSVF(mode svf.Mode, cutoff float64, resonance float64) *Filter {
	return newFilter(Params{Cutoff: cutoff, Resonance: resonance, Mode: int(mode)}, func(sr float64) processor {
		return &svfProc{filter: svf.New(cutoff, resonance, sr)}
	})
}

// NewLadder creates a Moog style ladder filter, resonance 0-1 (self-oscillates near 1), drive 1 is mild
func NewLadder(mode ladder.Mode, cutoff float64, resonance float64, drive float64) *Filter {
	return newFilter(Params{Cutoff: cutoff, Resonance: resonance, Mode: int(mode), Drive: drive}, func(sr float64) processor {
		return &ladderProc{filter: ladder.New(cutoff, resonance, sr)}
	})
}

// NewBiquad creates a biquad (EQ) filter, q 0.707 is flat for low and high pass, gainDB is used by
// the peak and shelf types
func NewBiquad(typ biquad.Type, cutoff float64, q float64, gainDB float64) *Filter {
	return newFilter(Params{Cutoff: cutoff, Resonance: q, Mode: int(typ), GainDB: gainDB}, func(sr float64) processor {
		return &biquadProc{filter: biquad.New(typ, cutoff, q, gainDB, sr)}
	})
}

// NewFormant creates a vowel filter, vowel 0-4 (a e i o u, fractions morph), q around 5-20,
// shift scales the formant frequencies (1 is an adult male voice)
func NewFormant(vowel float64, q float64, shift float64) *Filter {
	if shift <= 0.0 {
		shift = 1.0
	}
	return newFilter(Params{Resonance: q, Vowel: vowel, Shift: shift}, func(sr float64) processor {
		return &formantProc{filter: formant.New(vowel, q, sr)}
	})
}

func (f *Filter) SetCutoff(cutoff float64) {
	f.target.Cutoff = math.Max(1.0, cutoff)
}

func (f *Filter) SetResonance(res float64) {
	f.target.Resonance = res
}

func (f *Filter) SetVowel(vowel float64) {
	f.target.Vowel = vowel
}

func (f *Filter) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		f.ReceiveMessage(msg)
		return
	}

	switch index {
	case 1:
		f.SetCutoff(psalm.FltVal[float64](value))
	case 2:
		f.SetResonance(psalm.FltVal[float64](value))
	case 3:
		f.SetVowel(psalm.FltVal[float64](value))
	}
}

func (f *Filter) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("cutoff", f.SetCutoff)
	msg.IfFltVal("resonance", f.SetResonance)
	msg.IfFltVal("q", f.SetResonance)
	msg.IfFltVal("vowel", f.SetVowel)
	msg.IfIntVal("mode", func(m int) { f.target.Mode = m })
	msg.IfFltVal("gainDB", func(g float64) { f.target.GainDB = g })
	msg.IfFltVal("drive", func(d float64) { f.target.Drive = d })
	msg.IfFltVal("shift", func(s float64) { f.target.Shift = s })
}

// Clear resets the filter state
func (f *Filter) Clear() {
	for _, proc := range f.procs {
		proc.clear()
	}
}

func (f *Filter) Synthesize() {
	var (
		ins     = f.Input(0).Channels
		outs    = f.Output(0).Channels
		bufSize = f.Config().BufferSize
		start   = f.cur
		target  = f.target
		glide   = start.Cutoff != target.Cutoff || start.Vowel != target.Vowel || start.Resonance != target.Resonance
	)

	// parameters without a glide take effect right away
	f.cur.Mode, f.cur.GainDB, f.cur.Drive, f.cur.Shift = target.Mode, target.GainDB, target.Drive, target.Shift

	for i := range bufSize {
		if i == 0 || (glide && i%updateInterval == 0) {
			t := float64(i+updateInterval) / float64(bufSize)
			if !glide || t > 1.0 {
				t = 1.0
			}

			// cutoff glides in log frequency, so sweeps sound even
			if start.Cutoff > 0.0 && target.Cutoff > 0.0 {
				f.cur.Cutoff = start.Cutoff * math.Pow(target.Cutoff/start.Cutoff, t)
			}
			f.cur.Resonance = start.Resonance + (target.Resonance-start.Resonance)*t
			f.cur.Vowel = start.Vowel + (target.Vowel-start.Vowel)*t

			for _, proc := range f.procs {
				proc.apply(&f.cur)
			}
		}

		for c, proc := range f.procs {
			outs[c][i] = float32(proc.process(float64(ins[c][i])))
		}
	}

	f.cur = target
}

type svfProc struct {
	filter *svf.SVF
	mode   svf.Mode
}

func (p *svfProc) apply(params *Params) {
	p.filter.Set(params.Cutoff, params.Resonance)
	p.mode = svf.Mode(params.Mode)
}

func (p *svfProc) process(x float64) float64 { return p.filter.Process(x, p.mode) }
func (p *svfProc) clear()                    { p.filter.Clear() }

type ladderProc struct {
	filter *ladder.Ladder
	mode   ladder.Mode
	drive  float64
}

func (p *ladderProc) apply(params *Params) {
	if params.Drive != p.drive {
		p.drive = params.Drive
		p.filter.SetDrive(params.Drive)
	}
	p.filter.Set(params.Cutoff, params.Resonance)
	p.mode = ladder.Mode(params.Mode)
}

func (p *ladderProc) process(x float64) float64 { return p.filter.Process(x, p.mode) }
func (p *ladderProc) clear()                    { p.filter.Clear() }

type biquadProc struct {
	filter *biquad.Biquad
}

func (p *biquadProc) apply(params *Params) {
	p.filter.Set(biquad.Type(params.Mode), params.Cutoff, params.Resonance, params.GainDB)
}

func (p *biquadProc) process(x float64) float64 { return p.filter.Process(x) }
func (p *biquadProc) clear()                    { p.filter.Clear() }

type formantProc struct {
	filter *formant.Formant
	shift  float64
}

func (p *formantProc) apply(params *Params) {
	if params.Shift != p.shift {
		p.shift = params.Shift
		p.filter.SetShift(params.Shift)
	}
	p.filter.Set(params.Vowel, params.Resonance)
}

func (p *formantProc) process(x float64) float64 { return p.filter.Process(x) }
func (p *formantProc) clear()                    { p.filter.Clear() }

// Tail lets the filter rest while silent: a resonance rings on, keeping its output (and the filter)
// going until it is silent
func (f *Filter) Tail() float64 {
	return 0.01
}
