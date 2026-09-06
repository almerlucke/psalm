package granular

import (
	"math"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/tools/factory"
)

type ParameterGeneratorType int

const (
	Onset ParameterGeneratorType = iota
	PerFrame
)

type Parameter interface {
	Onset() int64      // Onset in samples
	Duration() float64 // Duration in seconds
	Amplitude() float64
}

type ParameterGenerator interface {
	psalm.Receiver
	Next(int64, psalm.Config) []Parameter
	Type() ParameterGeneratorType
}

type Envelope interface {
	Synthesize([][]float32, int)
	Activate(float64, int64, Parameter, psalm.Config)
}

type Source interface {
	Synthesize([][]float32, int)
	Activate(int64, Parameter, psalm.Config)
}

type grain struct {
	envelope  Envelope
	source    Source
	sampsToGo int64
}

func (g *grain) activate(p Parameter, config psalm.Config) {
	g.sampsToGo = int64(math.Round(p.Duration() * config.SampleRate))
	g.source.Activate(g.sampsToGo, p, config)
	g.envelope.Activate(p.Amplitude(), g.sampsToGo, p, config)
}

func (g *grain) synthesize(sourceBufs [][]float32, bufSize int) int {
	sampsToSynthesize := min(g.sampsToGo, int64(bufSize))

	g.sampsToGo -= sampsToSynthesize

	n := int(sampsToSynthesize)

	g.source.Synthesize(sourceBufs, n)
	g.envelope.Synthesize(sourceBufs, n)

	return n
}

type Granulator struct {
	*psalm.Module
	freeGrains    []*grain
	activeGrains  []*grain
	paramGen      ParameterGenerator
	nextParameter Parameter
	interOnset    int64
	timestamp     int64
	sourceBufs    [][]float32
}

func New(numChannels int, sf factory.Factory[Source], ef factory.Factory[Envelope], grainPoolSize int, paramGen ParameterGenerator) *Granulator {
	config := psalm.CurrentConfig()

	grains := make([]grain, grainPoolSize)
	freeGrains := make([]*grain, grainPoolSize)
	for i := range grains {
		grains[i].source = sf.New(config)
		grains[i].envelope = ef.New(config)
		freeGrains[i] = &grains[i]
	}

	gl := &Granulator{
		Module:       psalm.NewModule(nil, []int{numChannels}),
		freeGrains:   freeGrains,
		activeGrains: make([]*grain, 0, grainPoolSize),
		paramGen:     paramGen,
		sourceBufs:   make([][]float32, numChannels), // synthesize buffer for grain source
	}

	for i := range numChannels {
		gl.sourceBufs[i] = make([]float32, config.BufferSize)
	}

	if paramGen.Type() == Onset {
		gl.nextParameter = paramGen.Next(0, config)[0]
		gl.interOnset = gl.nextParameter.Onset()
	}

	gl.SetSelf(gl)

	return gl
}

func (gl *Granulator) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		gl.ReceiveMessage(msg)
		return
	}

	gl.paramGen.Receive(value, index)
}

func (gl *Granulator) ReceiveMessage(msg psalm.Message) {
	gl.paramGen.ReceiveMessage(msg)
}

func (gl *Granulator) synthesizeActiveGrains(bufSize int) {
	channels := gl.Output(0).Channels

	for i := 0; i < len(gl.activeGrains); {
		g := gl.activeGrains[i]
		n := g.synthesize(gl.sourceBufs, bufSize)

		for c, ch := range channels {
			dst := ch[:n]
			src := gl.sourceBufs[c][:n]
			for j := range dst {
				dst[j] += src[j]
			}
		}

		if g.sampsToGo == 0 {
			// Swap with last active grain and return to free pool
			lastIdx := len(gl.activeGrains) - 1
			gl.activeGrains[i] = gl.activeGrains[lastIdx]
			gl.activeGrains = gl.activeGrains[:lastIdx]
			gl.freeGrains = append(gl.freeGrains, g)
		} else {
			i++
		}
	}
}

func (gl *Granulator) onsetSynthesize() {
	var (
		outIndex     int
		cfg          = gl.Config()
		sampsLeft    = int64(cfg.BufferSize)
		timestamp    = gl.timestamp
		endTimestamp = timestamp + sampsLeft
	)

	// Step through inter onsets in current cycle
	for {
		// If interOnset is greater then sampsLeft, decrement interOnset and break
		if gl.interOnset > sampsLeft {
			gl.interOnset -= sampsLeft
			break
		}

		// Skip interonset samples
		sampsLeft -= gl.interOnset
		outIndex += int(gl.interOnset)
		timestamp += gl.interOnset

		gl.interOnset = 0

		channels := gl.Output(0).Channels

		// While interOnset == 0 generate new grains and synthesize for remaining samples in this frame
		for gl.interOnset == 0 {
			if len(gl.freeGrains) > 0 {
				g := gl.freeGrains[len(gl.freeGrains)-1]
				g.activate(gl.nextParameter, cfg)

				// Synthesize remaining samples in this frame
				n := g.synthesize(gl.sourceBufs, int(sampsLeft))

				for c, ch := range channels {
					dst := ch[outIndex : outIndex+n]
					src := gl.sourceBufs[c][:n]
					for i := range dst {
						dst[i] += src[i]
					}
				}

				// If grain is not done keep it for next frame in active grains
				if g.sampsToGo > 0 {
					gl.freeGrains = gl.freeGrains[:len(gl.freeGrains)-1]
					gl.activeGrains = append(gl.activeGrains, g)
				}
			}

			gl.nextParameter = gl.paramGen.Next(timestamp, cfg)[0]
			gl.interOnset = gl.nextParameter.Onset()
		}
	}

	// Update timestamp
	gl.timestamp = endTimestamp
}

func (gl *Granulator) perFrameSynthesize() {
	var (
		cfg        = gl.Config()
		bufferSize = int64(cfg.BufferSize)
		timestamp  = gl.timestamp
	)

	channels := gl.Output(0).Channels

	// Get parameters for this frame
	params := gl.paramGen.Next(timestamp, cfg)

	// For each parameter activate a grain and run it for the remaining samples in this frame
	for _, param := range params {
		if len(gl.freeGrains) > 0 {
			g := gl.freeGrains[len(gl.freeGrains)-1]
			g.activate(gl.nextParameter, cfg)

			// Bookkeeping
			interFrameOnset := param.Onset()
			outIndex := int(interFrameOnset)
			sampsToSynthesize := bufferSize - interFrameOnset

			// Synthesize remaining samples in this frame
			n := g.synthesize(gl.sourceBufs, int(sampsToSynthesize))

			// Copy grain output to output buffers
			for c, ch := range channels {
				dst := ch[outIndex : outIndex+n]
				src := gl.sourceBufs[c][:n]
				for i := range dst {
					dst[i] += src[i]
				}
			}

			// If grain is not done keep it for next frame in active grains
			if g.sampsToGo > 0 {
				gl.freeGrains = gl.freeGrains[:len(gl.freeGrains)-1]
				gl.activeGrains = append(gl.activeGrains, g)
			}
		}
	}

	// Update timestamp
	gl.timestamp = timestamp + bufferSize
}

func (gl *Granulator) Synthesize() {
	// Zero buffers for new frame
	gl.Output(0).Clear()

	// First run all currently active grains for full buffer size
	gl.synthesizeActiveGrains(gl.Config().BufferSize)

	// Synthesize based on parameter generator type
	switch gl.paramGen.Type() {
	case Onset:
		gl.onsetSynthesize()
	case PerFrame:
		gl.perFrameSynthesize()
	}
}
