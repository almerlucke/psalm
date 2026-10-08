package arrange

import (
	"fmt"
	"math"
)

// Span measures a section in bars, for curves that change at a bar instead of at a part of the
// section. Bars count from 0 and can be fractions; form.Span(name) gives the span of a section:
//
//	day := form.Span("Day 1")                         // 12 bars
//	day.Between(6, 7, arrange.Exp(300, 18000))        // a filter opens over bar 6
//	form.Span("Coda").Last(4, arrange.Smooth(66, 52)) // a ritardando over the last 4 bars
type Span float64

// Span returns the span of a section of the form. It panics on a section the form doesn't have
func (f *Form) Span(name string) Span {
	s, ok := f.Section(name)
	if !ok {
		panic(fmt.Sprintf("arrange: Span of section %q, which the form doesn't have", name))
	}
	return Span(s.Bars)
}

// bar returns the bar at a progress through the section
func (s Span) bar(progress float64) float64 { return progress * float64(s) }

// Between runs a curve from one bar to another: before it the curve's first value, after it its
// last
func (s Span) Between(from, to float64, c Curve) Curve {
	return func(progress float64) float64 {
		if to <= from {
			if s.bar(progress) < from {
				return c(0)
			}
			return c(1)
		}
		return c(math.Max(0, math.Min((s.bar(progress)-from)/(to-from), 1)))
	}
}

// From runs a curve from a bar to the end of the section
func (s Span) From(bar float64, c Curve) Curve { return s.Between(bar, float64(s), c) }

// Last runs a curve over the last bars of the section
func (s Span) Last(bars float64, c Curve) Curve { return s.Between(float64(s)-bars, float64(s), c) }

// Steps is a value for every bar, from bar 0: the last value holds on to the end of the section
// (Steps(0, 0, 0.5, 1): two bars of 0, a bar of 0.5, then 1)
func (s Span) Steps(values ...float64) Curve {
	return func(progress float64) float64 {
		if len(values) == 0 {
			return 0
		}
		b := int(math.Floor(s.bar(progress)))
		return values[max(0, min(b, len(values)-1))]
	}
}

// On is 1 from one bar to another and 0 outside them: a switch (a pedal held down, an effect
// frozen) for part of a section
func (s Span) On(from, to float64) Curve {
	return func(progress float64) float64 {
		if b := s.bar(progress); b >= from && b < to {
			return 1
		}
		return 0
	}
}

// Silent is the level (dB) Fade is at outside its bars, silence for a mixer strip (mixer.MinDB)
const Silent = -120.0

// Fade is a level in dB from one bar to another, faded in and out at its edges (an even fade of
// the amplitude, over a bar or a quarter of the bars when they are fewer) and Silent outside them:
// for the volume of a mixer strip, a sound that comes in for part of a section. A bar beyond the
// section (from -1, to Span+1) leaves out the fade at that edge, for a sound that carries on into
// the section before or after it
func (s Span) Fade(db, from, to float64) Curve {
	edge := math.Min(1, (to-from)/4)
	return func(progress float64) float64 {
		b := s.bar(progress)
		if b < from || b > to || edge <= 0 {
			return Silent
		}
		g := math.Min(math.Min((b-from)/edge, (to-b)/edge), 1)
		if g <= 0 {
			return Silent
		}
		return math.Max(Silent, db+20*math.Log10(g))
	}
}

// pedalLift is the part of a bar the pedal is up at a bar line
const pedalLift = 0.05

// Pedal is a sustain pedal changed at every bar line from one bar to another, the way a pianist
// changes it with the harmony: up for a moment at the bar line to clear the chord before, down
// for the rest of the bar (1), up (0) outside the bars
func (s Span) Pedal(from, to float64) Curve {
	return func(progress float64) float64 {
		b := s.bar(progress)
		if b < from || b >= to || b-math.Floor(b) < pedalLift {
			return 0
		}
		return 1
	}
}
