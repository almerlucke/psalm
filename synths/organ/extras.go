package organ

import "github.com/almerlucke/psalm/blocks/modmatrix"

// The destinations of the matrix (see Dests), by number
const (
	dDrive = iota
	dClick
	dVibrato
	dLevel
	dCutoff
	dResonance
	dPerc
	dLeakage
	dPitch
	dTone
	dLeslie
	dLeslieSpeed
)

// Dests are the settings the matrix of the organ moves: by their keys, as described (Describe), with
// how far a full source moves them (the setting itself, octaves for the cutoff, semitones for the
// pitch). The click is read as a key goes down or up, the others every sample.
var Dests = []modmatrix.Dest{
	dDrive:       {Key: "drive", Label: "drive", Span: 1},
	dClick:       {Key: "click", Label: "click", Span: 1},
	dVibrato:     {Key: "vibratoDepth", Label: "vibrato depth", Span: 1}, // 1 is V3 deep
	dLevel:       {Key: "level", Label: "level", Span: 1},
	dCutoff:      {Key: "cutoff", Label: "cutoff", Span: 6},
	dResonance:   {Key: "resonance", Label: "resonance", Span: 1},
	dPerc:        {Key: "percLevel", Label: "percussion level", Span: 1},
	dLeakage:     {Key: "leakage", Label: "leakage", Span: 1},
	dPitch:       {Key: "pitch", Label: "pitch", Span: 12},
	dTone:        {Key: "tone", Label: "tone", Span: 1},
	dLeslie:      {Key: "leslie", Label: "leslie", Span: 1},
	dLeslieSpeed: {Key: "leslieSpeed", Label: "leslie speed", Span: 1},
}

// The value inputs: 1 to 9 set the drawbars, 10 to 13 the values of the matrix (Ext 1 to 4) that a
// slot can move any setting by
const InExt1 = 10
