package main

import (
	"math/rand"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/effects/freeverb"
	"github.com/almerlucke/psalm/modules/granular"
	"github.com/almerlucke/psalm/modules/granular/parabolic"
	"github.com/almerlucke/sndfile"
)

type Param struct {
	sf        sndfile.SoundFiler[float32]
	offset    float64
	loopRange float64
	onset     int64
	duration  float64
	amplitude float64
	speed     float64
}

func (p *Param) Onset() int64 {
	return p.onset
}

func (p *Param) Duration() float64 {
	return p.duration
}

func (p *Param) Amplitude() float64 {
	return p.amplitude
}

type Source struct {
	sf          sndfile.SoundFiler[float32]
	inc         float64
	phase       float64
	depth       int
	startOffset float64
	endOffset   float64
	loopRange   float64
	amp         float32
}

func NewSource() *Source {
	return &Source{}
}

func (s *Source) Synthesize(outBuffers [][]float32, bufSize int) {
	numFramesMinus1 := float64(s.sf.NumFrames() - 1)

	for i := range bufSize {
		out := s.sf.Lookup(s.phase*numFramesMinus1, 0, s.depth, false) * s.amp
		for _, outBuf := range outBuffers {
			outBuf[i] = out
		}

		s.phase += s.inc

		np := s.phase - s.startOffset

		for np >= s.loopRange {
			np -= s.loopRange
		}
		for np < 0.0 {
			np += s.loopRange
		}

		s.phase = s.startOffset + np
	}
}

func (s *Source) Activate(sampsToGo int64, p *Param, cfg psalm.Config) {
	s.sf = p.sf
	s.inc = (p.speed * s.sf.SampleRate() / cfg.SampleRate) / float64(s.sf.NumFrames())
	s.depth = sndfile.SpeedToMipMapDepth(p.speed)
	if s.depth >= s.sf.Depth() {
		s.depth = s.sf.Depth() - 1
	}

	zc := s.sf.ZeroCrossings(0)
	zcLow := p.offset - p.loopRange*0.5
	zcHigh := p.offset + p.loopRange*0.5

	if zcLow < 0.0 {
		zcLow = 0.0
	}

	if zcHigh > 1.0 {
		zcHigh = 1.0
	}

	cr, ok := zc.NearestPos(zcLow, sndfile.DirectionAny)
	if ok {
		zcLow = cr.Position
	}
	cr, ok = zc.NearestPos(zcHigh, sndfile.DirectionAny)
	if ok {
		zcHigh = cr.Position
	}

	s.startOffset = zcLow
	s.endOffset = zcHigh
	s.loopRange = s.endOffset - s.startOffset

	if s.inc > 0.0 {
		s.phase = zcLow
	} else {
		s.phase = zcHigh
	}

	s.amp = float32(p.amplitude)
}

func (s *Source) New(cfg any) *Source {
	return NewSource()
}

type ParamGen struct {
	granular.ParameterGenerator[*Param]
	sf        sndfile.SoundFiler[float32]
	parameter Param // single parameter can be passed for each NextOnset call, prevent from allocating new params
}

func NewParamGen(sf sndfile.SoundFiler[float32]) *ParamGen {
	g := &ParamGen{}
	g.sf = sf
	return g
}

func (pgen *ParamGen) NextOnset(_ int64, config psalm.Config) *Param {
	p := &pgen.parameter

	p.sf = pgen.sf
	p.duration = rand.Float64()*5.2 + 1.02
	p.amplitude = rand.Float64()*0.4 + 0.4
	p.speed = rand.Float64()*0.4 + 0.8
	p.onset = int64((rand.Float64()*3.005 + 1.0001) * config.SampleRate)
	p.offset = rand.Float64()*0.6 + 0.2
	p.loopRange = rand.Float64()*0.1 + 0.1

	return p
}

func (pgen *ParamGen) Type() granular.ParameterGeneratorType {
	return granular.Onset
}

func main() {
	p := psalm.New()

	sf := sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/John_1-1.wav", 4)

	gl := granular.New[*Param, *Source, *parabolic.Envelope[*Param]](
		1,
		&Source{},
		&parabolic.Envelope[*Param]{},
		40,
		NewParamGen(sf),
	).AddTo(p)

	p.Out(freeverb.New(0.9, 0.5, 0.95, 0.3, 0.6).AddTo(p).In(
		gl,
	))

	_ = p.Listen()
}
