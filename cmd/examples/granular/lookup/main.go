package main

import (
	"math"
	"math/rand"

	"github.com/almerlucke/genny"
	"github.com/almerlucke/genny/float/rwalk"
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
	crossfade float64
	onset     int64
	duration  float64
	amplitude float64
	speed     float64
	pan       float64
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
	crossfade   float64
	amp         float32
	pan         [2]float32
}

func NewSource() *Source {
	return &Source{}
}

func (s *Source) Synthesize(outBuffers [][]float32, bufSize int) {
	numFramesMinus1 := float64(s.sf.NumFrames() - 1)
	effectiveLoopRange := s.loopRange - s.crossfade

	for i := range bufSize {
		var out float32

		if s.crossfade > 0.0 {
			forwardDist := s.endOffset - s.crossfade
			backwardDist := s.startOffset + s.crossfade
			if s.inc > 0.0 && s.phase >= forwardDist {
				x := float32((s.endOffset - s.phase) / s.crossfade)
				if x < 0 {
					x = 0
				} else if x > 1 {
					x = 1
				}
				phase2 := s.startOffset + (s.phase - forwardDist)
				out1 := s.sf.Lookup(s.phase*numFramesMinus1, 0, s.depth, false)
				out2 := s.sf.Lookup(phase2*numFramesMinus1, 0, s.depth, false)
				out = out1*x + out2*(1.0-x)
			} else if s.inc < 0.0 && s.phase <= backwardDist {
				x := float32((s.phase - s.startOffset) / s.crossfade)
				if x < 0 {
					x = 0
				} else if x > 1 {
					x = 1
				}
				phase2 := s.endOffset - (backwardDist - s.phase)
				out1 := s.sf.Lookup(s.phase*numFramesMinus1, 0, s.depth, false)
				out2 := s.sf.Lookup(phase2*numFramesMinus1, 0, s.depth, false)
				out = out1*x + out2*(1.0-x)
			} else {
				out = s.sf.Lookup(s.phase*numFramesMinus1, 0, s.depth, false)
			}
		} else {
			out = s.sf.Lookup(s.phase*numFramesMinus1, 0, s.depth, false)
		}

		for j, outBuf := range outBuffers {
			outBuf[i] = s.pan[j] * out
		}

		s.phase += s.inc

		if effectiveLoopRange > 0.0 {
			for s.phase >= s.endOffset {
				s.phase -= effectiveLoopRange
			}
			for s.phase < s.startOffset {
				s.phase += effectiveLoopRange
			}
		}
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

	maxCrossfade := s.loopRange * 0.5
	s.crossfade = p.crossfade
	if s.crossfade <= 0.0 {
		s.crossfade = s.loopRange * 0.15
	}
	if s.crossfade > maxCrossfade {
		s.crossfade = maxCrossfade
	}

	if s.inc > 0.0 {
		s.phase = zcLow
	} else {
		s.phase = zcHigh
	}

	s.pan[0] = float32(math.Cos(p.pan*math.Pi/2.0) * p.amplitude)
	s.pan[1] = float32(math.Sin(p.pan*math.Pi/2.0) * p.amplitude)
}

func (s *Source) New(cfg any) *Source {
	return NewSource()
}

type ParamGen struct {
	granular.ParameterGenerator[*Param]
	sf        sndfile.SoundFiler[float32]
	panGen    genny.Generator[float64]
	offsetGen genny.Generator[float64]
	parameter Param // single parameter can be passed for each NextOnset call, prevent from allocating new params
}

func NewParamGen(sf sndfile.SoundFiler[float32]) *ParamGen {
	g := &ParamGen{}
	g.sf = sf
	g.panGen = rwalk.New(0.5, 0.0, 1.0, 0.0375, rwalk.Bounce)
	g.offsetGen = rwalk.New(0.05, 0.05, 0.95, 0.075, rwalk.Bounce)
	return g
}

func (pgen *ParamGen) NextOnset(_ int64, config psalm.Config) *Param {
	p := &pgen.parameter

	p.sf = pgen.sf
	p.duration = rand.Float64()*3.2 + 0.32
	p.amplitude = rand.Float64()*0.4 + 0.1
	p.speed = rand.Float64()*0.3 + 0.85
	p.onset = int64((rand.Float64()*0.05 + 0.0001) * config.SampleRate)
	p.offset = pgen.offsetGen.Generate()
	p.loopRange = rand.Float64()*0.025 + 0.025
	p.crossfade = p.loopRange * 0.15
	p.pan = pgen.panGen.Generate()

	return p
}

func (pgen *ParamGen) Type() granular.ParameterGeneratorType {
	return granular.Onset
}

func main() {
	p := psalm.New()

	sf := sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/John_1-1.wav", 4)

	gl := granular.New[*Param, *Source, *parabolic.Envelope[*Param]](
		2,
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
