package modal

import "github.com/almerlucke/psalm/blocks/modmatrix"

// The destinations of the matrix (see Dests), by number
const (
	dPitch = iota
	dDecay
	dHardness
	dPosition
	dTremoloDepth
	dBeat
	dNoise
	dLevel
	dCutoff
	dResonance
	dInharmonicity
	dHighDamping
)

// Dests are the settings the matrix of a voice moves: by their keys, as described (Describe), with
// how far a full source moves them (semitones, octaves, or the setting itself). The pitch is moved
// every 8 samples; decay, hardness, position, beat, noise, inharmonicity and high damping are set as
// the note is struck, from the sources known then (velocity, key, random, wheel, ext); the rest
// every sample.
var Dests = []modmatrix.Dest{
	dPitch:         {Key: "pitch", Label: "pitch", Span: 12},
	dDecay:         {Key: "decay", Label: "decay", Span: 2},
	dHardness:      {Key: "hardness", Label: "hardness", Span: 1},
	dPosition:      {Key: "position", Label: "position", Span: 0.5},
	dTremoloDepth:  {Key: "tremoloDepth", Label: "tremolo depth", Span: 1},
	dBeat:          {Key: "beat", Label: "beat", Span: 0.02},
	dNoise:         {Key: "noise", Label: "noise", Span: 1},
	dLevel:         {Key: "level", Label: "level", Span: 1},
	dCutoff:        {Key: "cutoff", Label: "cutoff", Span: 6},
	dResonance:     {Key: "resonance", Label: "resonance", Span: 1},
	dInharmonicity: {Key: "inharmonicity", Label: "inharmonicity", Span: 1},
	dHighDamping:   {Key: "highDamping", Label: "high damping", Span: 1},
}

// The value inputs: 1 sets the hardness, 2 the tremolo depth, 3 to 6 the values of the matrix (Ext
// 1 to 4) that a slot can move any setting by
const InExt1 = 3
