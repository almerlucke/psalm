package loop

import (
	"math"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/sndfile"
)

type Param struct {
	Sound          sndfile.SoundFiler[float32]
	Offset         float64
	LoopRange      float64
	Crossfade      float64
	GrainOnset     int64
	GrainDuration  float64
	GrainAmplitude float64
	Speed          float64
	Pan            float64
}

func (p *Param) Onset() int64 {
	return p.GrainOnset
}

func (p *Param) Duration() float64 {
	return p.GrainDuration
}

func (p *Param) Amplitude() float64 {
	return p.GrainAmplitude
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
	s.sf = p.Sound
	s.inc = (p.Speed * s.sf.SampleRate() / cfg.SampleRate) / float64(s.sf.NumFrames())
	s.depth = sndfile.SpeedToMipMapDepth(p.Speed)
	if s.depth >= s.sf.Depth() {
		s.depth = s.sf.Depth() - 1
	}

	zc := s.sf.ZeroCrossings(0)
	zcLow := p.Offset - p.LoopRange*0.5
	zcHigh := p.Offset + p.LoopRange*0.5

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
	s.crossfade = p.Crossfade
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

	s.pan[0] = float32(math.Cos(p.Pan*math.Pi/2.0) * p.GrainAmplitude)
	s.pan[1] = float32(math.Sin(p.Pan*math.Pi/2.0) * p.GrainAmplitude)
}

func (s *Source) New(cfg any) *Source {
	return NewSource()
}
