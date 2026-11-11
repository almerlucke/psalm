// Package paramstest checks that a synth or an effect describes itself right (params): every
// setting it describes is a message it takes, the value lands where the description says, and the
// envelopes take their stages.
package paramstest

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/params"
)

// Check sends every setting of the description to the module (send) another value in its range,
// and describes the module again (describe, from its settings as they are now): the setting has to
// have moved there. It checks the envelopes' stages the same way
func Check(t *testing.T, describe func() params.Set, send func(psalm.Message)) {
	t.Helper()
	set := describe()
	if set.Name == "" {
		t.Errorf("the description has no name")
	}
	seen := map[string]bool{}
	for _, p := range set.Params {
		if seen[p.Key] {
			t.Errorf("%s: %q twice", set.Name, p.Key)
		}
		seen[p.Key] = true
		if p.Group == "" {
			t.Errorf("%s %s: no group", set.Name, p.Key)
		}
		if p.Min >= p.Max {
			t.Errorf("%s %s: range %g to %g", set.Name, p.Key, p.Min, p.Max)
		}
		if p.Exp && p.Min <= 0 {
			t.Errorf("%s %s: exponential from %g", set.Name, p.Key, p.Min)
		}
		if p.Kind == params.Choice && len(p.Options) != int(p.Max)+1 {
			t.Errorf("%s %s: %d options for a range to %g", set.Name, p.Key, len(p.Options), p.Max)
		}
		if p.Value < p.Min-1e-9 || p.Value > p.Max+1e-9 {
			t.Errorf("%s %s: at %g, outside %g to %g", set.Name, p.Key, p.Value, p.Min, p.Max)
		}
		// another value: a third of the way from the end it isn't near
		want := p.Min + (p.Max-p.Min)*0.7
		if p.Value > p.Min+(p.Max-p.Min)*0.5 {
			want = p.Min + (p.Max-p.Min)*0.3
		}
		if p.Exp {
			want = p.Min * math.Pow(p.Max/p.Min, 0.7)
			if p.Value > math.Sqrt(p.Min*p.Max) {
				want = p.Min * math.Pow(p.Max/p.Min, 0.3)
			}
		}
		if p.Kind != params.Knob {
			want = p.Max
			if p.Value == p.Max {
				want = p.Min
			}
		}
		want = p.Clamp(want)
		send(psalm.Message{p.Key: want})
		got, ok := describe().Find(p.Key)
		if !ok {
			t.Errorf("%s %s: gone from the description", set.Name, p.Key)
			continue
		}
		if math.Abs(got.Value-want) > 1e-6*max(1, math.Abs(want)) {
			t.Errorf("%s %s: sent %g, it is at %g", set.Name, p.Key, want, got.Value)
		}
	}
	for i, e := range set.Envelopes {
		if e.Prefix == "" || e.Stages == "" {
			t.Errorf("%s envelope %d: no prefix or stages", set.Name, i)
			continue
		}
		stage := map[byte]struct {
			key  string
			v    float64
			read func(params.Envelope) float64
		}{
			'A': {"Attack", 123, func(e params.Envelope) float64 { return e.Attack }},
			'D': {"Decay", 456, func(e params.Envelope) float64 { return e.Decay }},
			'S': {"Sustain", 0.37, func(e params.Envelope) float64 { return e.Sustain }},
			'R': {"Release", 789, func(e params.Envelope) float64 { return e.Release }},
		}
		for j := range len(e.Stages) {
			st, ok := stage[e.Stages[j]]
			if !ok {
				t.Errorf("%s envelope %s: stage %q", set.Name, e.Prefix, e.Stages[j])
				continue
			}
			send(psalm.Message{e.Prefix + st.key: st.v})
			now := describe().Envelopes[i]
			if math.Abs(st.read(now)-st.v) > 1e-6*st.v {
				t.Errorf("%s %s%s: sent %g, it is at %g", set.Name, e.Prefix, st.key, st.v, st.read(now))
			}
		}
	}
}
