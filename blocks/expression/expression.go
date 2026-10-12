// Package expression keeps what a player's hands do to a sound while it plays: the pitch bend,
// the mod wheel and the aftertouch (pressing harder on a held key). A voice takes them from its
// messages ("pitchBend" -1 to 1, "bendRange" in semitones, "modWheel" 0 to 1, "pressure" 0 to 1),
// glides to them (a few milliseconds, so moving them never zips or clicks) and reads them every
// sample. The aftertouch adds to the wheel: whatever the wheel does, pressing harder does too.
package expression

import (
	"math"

	"github.com/almerlucke/psalm"
)

// DefaultRange is the semitones a full bend goes up or down when no range is set
const DefaultRange = 2.0

// glideSeconds is how long the bend and the wheel take to get where they are moved to (a third)
const glideSeconds = 0.004

// Expression is the bend and the wheel of a voice
type Expression struct {
	Range float64 // semitones of a full bend (0 is DefaultRange)

	bend, bendNow float64 // -1 to 1: where it is moved to, where it is
	mod, modNow   float64 // 0 to 1
	press, pressN float64 // the aftertouch, 0 to 1
	ratio         float64 // the frequency ratio of bendNow
	ratioOf       float64 // the semitones ratio was computed for
	vibPhase      float64 // the wheel's vibrato
	sr, c         float64 // the sample rate and the glide per sample at it
}

// Receive takes "pitchBend", "bendRange" and "modWheel" from a message
func (e *Expression) Receive(msg psalm.Message) {
	msg.IfFltVal("pitchBend", func(f float64) { e.bend = math.Max(-1, math.Min(f, 1)) })
	msg.IfFltVal("bendRange", func(f float64) { e.Range = math.Max(0, f) })
	msg.IfFltVal("modWheel", func(f float64) { e.mod = math.Max(0, math.Min(f, 1)) })
	msg.IfFltVal("pressure", func(f float64) { e.press = math.Max(0, math.Min(f, 1)) })
}

// Moved says whether the bend or the wheel are anywhere but at rest (or still getting there)
func (e *Expression) Moved() bool {
	return e.bend != 0 || e.bendNow != 0 || e.mod != 0 || e.modNow != 0 || e.press != 0 || e.pressN != 0
}

// coef returns the glide per sample
func coef(sr float64) float64 { return 1 - math.Exp(-1/(glideSeconds*sr)) }

// Tick moves the bend and the wheel a sample towards where they are moved to
func (e *Expression) Tick(sr float64) {
	if e.bendNow == e.bend && e.modNow == e.mod && e.pressN == e.press {
		return
	}
	if sr != e.sr {
		e.sr, e.c = sr, coef(sr)
	}
	if e.bendNow != e.bend {
		e.bendNow += (e.bend - e.bendNow) * e.c
		if math.Abs(e.bend-e.bendNow) < 1e-6 {
			e.bendNow = e.bend
		}
	}
	if e.modNow != e.mod {
		e.modNow += (e.mod - e.modNow) * e.c
		if math.Abs(e.mod-e.modNow) < 1e-6 {
			e.modNow = e.mod
		}
	}
	if e.pressN != e.press {
		e.pressN += (e.press - e.pressN) * e.c
		if math.Abs(e.press-e.pressN) < 1e-6 {
			e.pressN = e.press
		}
	}
}

// Settle puts the bend and the wheel where they are moved to at once (for a voice that reads them
// once a buffer, or a note that starts)
func (e *Expression) Settle() {
	e.bendNow, e.modNow, e.pressN = e.bend, e.mod, e.press
}

// Semitones returns the bend now in semitones
func (e *Expression) Semitones() float64 {
	r := e.Range
	if r == 0 {
		r = DefaultRange
	}
	return e.bendNow * r
}

// Ratio returns the frequency ratio of the bend now (1 at rest)
func (e *Expression) Ratio() float64 {
	s := e.Semitones()
	if s == 0 {
		return 1
	}
	if s != e.ratioOf || e.ratio == 0 {
		e.ratio, e.ratioOf = math.Exp2(s/12), s
	}
	return e.ratio
}

// Wheel returns the mod wheel now with the aftertouch added, 0 to 1
func (e *Expression) Wheel() float64 { return math.Min(1, e.modNow+e.pressN) }

// Pressure returns the aftertouch now, 0 to 1 (also in Wheel)
func (e *Expression) Pressure() float64 { return e.pressN }

// Vibrato returns the semitones of a vibrato the wheel brings in (none at rest, depth semitones
// up and down with the wheel all the way), and moves it a sample at rate Hz
func (e *Expression) Vibrato(sr, rate, depth float64) float64 {
	w := e.Wheel()
	if w == 0 {
		e.vibPhase = 0
		return 0
	}
	e.vibPhase += rate / sr
	if e.vibPhase >= 1 {
		e.vibPhase--
	}
	return w * depth * math.Sin(2*math.Pi*e.vibPhase)
}

// VibratoRatio returns the frequency ratio of the bend and the wheel's vibrato together, moving the
// vibrato a sample
func (e *Expression) VibratoRatio(sr, rate, depth float64) float64 {
	v := e.Vibrato(sr, rate, depth)
	if v == 0 {
		return e.Ratio()
	}
	return math.Exp2((e.Semitones() + v) / 12)
}

// Bend returns the bend now, -1 to 1 (for a voice that maps it itself, like a sampler's bend_up and
// bend_down)
func (e *Expression) Bend() float64 { return e.bendNow }
