package dx

import "github.com/almerlucke/psalm/blocks/modmatrix"

// The destinations of the matrix (see Dests), by number
const (
	dModDepth = iota
	dFeedback
	dOp1Gain // to dOp1Gain+5: what an operator is heard or modulates with
	_
	_
	_
	_
	_
	dPitch
	dLevel
	dCutoff
	dResonance
	dLFO2Rate
)

// Dests are the settings the matrix of a voice moves, by their keys, with how far a full source
// moves them (a gain is the level of a free operator, or added to the amplitude of the others)
var Dests = []modmatrix.Dest{
	dModDepth:    {Key: "modDepth", Label: "mod depth", Span: 2},
	dFeedback:    {Key: "freeFeedback", Label: "feedback", Span: 1},
	dOp1Gain:     {Key: "op1Gain", Label: "op 1 gain", Span: 1},
	dOp1Gain + 1: {Key: "op2Gain", Label: "op 2 gain", Span: 1},
	dOp1Gain + 2: {Key: "op3Gain", Label: "op 3 gain", Span: 1},
	dOp1Gain + 3: {Key: "op4Gain", Label: "op 4 gain", Span: 1},
	dOp1Gain + 4: {Key: "op5Gain", Label: "op 5 gain", Span: 1},
	dOp1Gain + 5: {Key: "op6Gain", Label: "op 6 gain", Span: 1},
	dPitch:       {Key: "pitch", Label: "pitch", Span: 24},
	dLevel:       {Key: "level", Label: "level", Span: 1},
	dCutoff:      {Key: "cutoff", Label: "cutoff (filter)", Span: 6},
	dResonance:   {Key: "resonance", Label: "resonance (filter)", Span: 1},
	dLFO2Rate:    {Key: "lfo2Rate", Label: "lfo 2 rate", Span: 3},
}

// InExt1 is the first of the value inputs 5 to 8 that carry the matrix's sources Ext 1 to 4
const InExt1 = 5
