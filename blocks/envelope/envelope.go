// Package envelope lets messages change an envelope while a synth plays, the same way on every
// synth: "<prefix>Attack", "<prefix>Decay" and "<prefix>Release" in milliseconds, and
// "<prefix>Sustain", the level held while the key is down (0-1). The amplitude envelope of a synth
// is "amp" ("ampAttack", "ampRelease", ...), its other envelopes have names of their own ("filter",
// "position", "formant"). A change takes effect from the next note.
package envelope

import (
	"math"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
)

// shortest is the shortest stage, in milliseconds (no clicks)
const shortest = 0.1

// Receive changes an envelope setting (durations in milliseconds) from a message
func Receive(msg psalm.Message, prefix string, s *adsr.Setting) {
	msg.IfFltVal(prefix+"Attack", func(f float64) { s.AttackDuration = math.Max(shortest, f) })
	msg.IfFltVal(prefix+"Decay", func(f float64) { s.DecayDuration = math.Max(shortest, f) })
	msg.IfFltVal(prefix+"Sustain", func(f float64) { s.DecayLevel = math.Max(0, math.Min(f, 1)) * level(s) })
	msg.IfFltVal(prefix+"Release", func(f float64) { s.ReleaseDuration = math.Max(shortest, f) })
}

// level is the peak of the attack (1 when unset)
func level(s *adsr.Setting) float64 {
	if s.AttackLevel <= 0 {
		return 1
	}
	return s.AttackLevel
}

// Seconds changes durations kept in seconds (a synth with a simpler envelope: an attack and a
// release, a decay) from the same messages in milliseconds; nil leaves one out
func Seconds(msg psalm.Message, prefix string, attack, decay, release *float64) {
	set := func(name string, p *float64) {
		if p != nil {
			msg.IfFltVal(prefix+name, func(f float64) { *p = math.Max(shortest, f) / 1000 })
		}
	}
	set("Attack", attack)
	set("Decay", decay)
	set("Release", release)
}
