package main

import (
	"math/rand"

	"github.com/almerlucke/genny"
	"github.com/almerlucke/genny/bucket"
	"github.com/almerlucke/genny/float/rwalk"
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/effects/freeverb"
	"github.com/almerlucke/psalm/modules/granular"
	"github.com/almerlucke/psalm/modules/granular/envelopes/parabolic"
	"github.com/almerlucke/psalm/modules/granular/sources/loop"
	"github.com/almerlucke/sndfile"
)

type ParamGen struct {
	granular.ParameterGenerator[*loop.Param]
	sb        sndfile.SoundBank[float32]
	panGen    genny.Generator[float64]
	offsetGen genny.Generator[float64]
	sndGen    genny.Generator[string]
	//speedGen      genny.Generator[float64]
	speedWidthGen genny.Generator[float64]
	parameter     loop.Param // single parameter can be passed for each NextOnset call, prevent from allocating new params
}

func NewParamGen(sb sndfile.SoundBank[float32]) *ParamGen {
	g := &ParamGen{}
	g.sb = sb
	g.panGen = rwalk.New(0.5, 0.0, 1.0, 0.001, rwalk.Bounce)
	// g.offsetGen = float.FromFrame(shape.New(float.ToFrame(phasor.New(0.01, 4000.0, 0.0)), linear.New(0.95, 0.05)), 0)
	g.sndGen = bucket.NewLoop("sf1", "sf2", "sf3")
	//g.speedGen = rwalk.New(1.0, 0.6, 1.4, 0.001, rwalk.Jump)
	g.speedWidthGen = rwalk.New(0.01, 0.01, 0.05, 0.00001, rwalk.Jump)
	g.offsetGen = rwalk.New(0.5, 0.0, 1.0, 0.001, rwalk.Jump)
	return g
}

func (pgen *ParamGen) NextOnset(_ int64, config psalm.Config) *loop.Param {
	p := &pgen.parameter

	p.Sound = pgen.sb[pgen.sndGen.Generate()]
	p.GrainDuration = rand.Float64()*0.7 + 0.02
	p.GrainAmplitude = rand.Float64()*0.3 + 0.05
	p.Speed = rand.Float64()*pgen.speedWidthGen.Generate() + 1.0
	if rand.Float64() > 0.5 {
		p.Speed = -p.Speed
	}
	p.GrainOnset = int64((rand.Float64()*0.0005 + 0.00001) * config.SampleRate)
	p.Offset = pgen.offsetGen.Generate()
	p.LoopRange = rand.Float64()*0.25 + 0.025
	p.Crossfade = p.LoopRange * 0.15
	p.Pan = pgen.panGen.Generate()

	return p
}

func (pgen *ParamGen) Type() granular.ParameterGeneratorType {
	return granular.Onset
}

func main() {
	p := psalm.New()

	sf1 := sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/mixkit-angelical-choir-654.wav", 4)
	sf2 := sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/mixkit-cinematic-angelical-choir-transition-663.wav", 4)
	sf3 := sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/mixkit-movie-trailer-synth-impact-649.wav", 4)
	sb := sndfile.SoundBank[float32]{
		"sf1": sf1,
		"sf2": sf2,
		"sf3": sf3,
	}

	gl := granular.New[*loop.Param, *loop.Source, *parabolic.Envelope[*loop.Param]](
		2,
		&loop.Source{},
		&parabolic.Envelope[*loop.Param]{},
		140,
		NewParamGen(sb),
	).AddTo(p)

	p.Out(freeverb.New(0.8, 0.4, 0.95, 0.4, 0.5).AddTo(p).In(
		gl,
	))

	p.Out(gl)

	_ = p.Listen()
}
