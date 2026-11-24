package main

import (
	"fmt"
	"go/format"
	"math"
	"os"
	"slices"
	"strings"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/synths/waveguide"
	"github.com/almerlucke/psalm/synths/waveguide/reference"
)

// The fit of the start of the notes: the real notes start brighter than the string pulled aside does. The Edge of
// the pluck is fitted to the share of the energy above 1280 Hz in the first 50 ms (the pluck is loud: over the noise
// at every pitch); the damping and the decay as for -resolved, with that edge

const edgeSteps = 10

// renderEdge plays one note of the model without its body, with a decay, a damping and an edge of the pluck
func renderEdge(pitch, decay, damping, edge float64) []float64 {
	cfg := noBody(waveguide.Guitar())
	cfg.Edge = edge
	return play(cfg, psalm.Message{"pitch": pitch, "decay": decay, "damping": damping})
}

func earlyHigh(pitch, decay, damping, edge float64) float64 {
	return reference.EarlyHigh(reference.EarlyBandShares(renderEdge(pitch, decay, damping, edge), sampleRate))
}

func highEdge(pitch, decay, damping, edge float64) float64 {
	return reference.HighShare(renderEdge(pitch, decay, damping, edge), sampleRate)
}

// late640 is the octave band 640-1280 Hz against all, from 0.25 s on
func late640(pitch, decay, damping, edge float64) float64 {
	return reference.BandShares(renderEdge(pitch, decay, damping, edge), sampleRate)[3]
}

func rateEdge(pitch, decay, damping, edge float64) float64 {
	return min(reference.DropRate(renderEdge(pitch, decay, damping, edge), sampleRate), 300)
}

// bisect finds the x in [lo, hi] where f crosses the target (f rises with x when up is set)
func bisect(lo, hi, target float64, up bool, f func(float64) float64) float64 {
	for range edgeSteps {
		mid := (lo + hi) / 2
		if (f(mid) > target) == up {
			hi = mid
		} else {
			lo = mid
		}
	}
	return (lo + hi) / 2
}

// fitDecayEdge is the decay at which the model falls as the real guitar does at a pitch
func fitDecayEdge(pitch, damping, edge float64) float64 {
	return math.Exp(bisect(math.Log(0.3), math.Log(120), reference.TargetRate(pitch), false, func(l float64) float64 {
		return rateEdge(pitch, math.Exp(l), damping, edge)
	}))
}

// fitDampingEdge is the damping (from -1/9) at which the model is as bright above 2 kHz as the target, for an edge
func fitDampingEdge(pitch, target, edge float64) float64 {
	decay, damping := 7.0, presetDamp
	for range fitRounds {
		damping = bisect(-1.0/9, maxDamp, target, false, func(d float64) float64 { return highEdge(pitch, decay, d, edge) })
		decay = fitDecayEdge(pitch, damping, edge)
	}
	return damping
}

// fitEarly writes the track of the pluck. The edge is fitted to the start of the notes at every pitch (over the noise
// there, smoothed). The damping is fitted only at the pitches of the notes of the recording that are over its noise,
// to their brightness above 2 kHz for that edge, and goes straight from one of them to the next (the brightness
// follows the damping steeply below 0.1, and the target is only known at these pitches: a smoothing would miss them
// all); below the lowest it comes from the table of the ownGuitar preset, above the highest it holds. The decay is
// fitted to the fall of the level at every pitch
func fitEarly(out, name string) {
	base := waveguide.OwnGuitar().Track
	var edges []float64
	for p := fitFrom; p <= fitTo; p++ {
		pf := float64(p)
		tHigh := reference.TargetEarlyHigh(pf)
		edges = append(edges, bisect(0, 1, tHigh, true, func(e float64) float64 { return earlyHigh(pf, base.DecayAt(pf), base.DampingAt(pf), e) }))
	}
	edges = smooth(edges, smoothWidth)

	// the anchors: the pitches of the notes over the noise, the highest damping the fit goes to is 0.45
	type anchor struct{ pitch, damping float64 }
	var anchors []anchor
	last := -1
	for _, n := range reference.Notes() {
		if n.Margin < reference.ResolvedMargin || n.Pitch == last {
			continue
		}
		last = n.Pitch
		sum, k := 0.0, 0.0
		for _, m := range reference.Notes() {
			if m.Pitch == n.Pitch && m.Margin >= reference.ResolvedMargin {
				sum += m.HighDB
				k++
			}
		}
		pf := float64(n.Pitch)
		d := min(fitDampingEdge(pf, sum/k, edges[n.Pitch-fitFrom]), 0.45)
		fmt.Printf("anchor %d: %.1f dB above 2 kHz, damping %+.3f\n", n.Pitch, sum/k, d)
		anchors = append(anchors, anchor{pf, d})
	}
	slices.SortFunc(anchors, func(a, b anchor) int { return int(a.pitch - b.pitch) })
	// from the table where the recording tells nothing, up to the lowest anchor (one before it, so the table is what it was)
	from := anchors[0].pitch - 4
	anchors = slices.Insert(anchors, 0, anchor{from, base.DampingAt(from)})
	damping := func(p float64) float64 {
		if p <= from {
			return base.DampingAt(p)
		}
		for i := 1; i < len(anchors); i++ {
			if p <= anchors[i].pitch {
				f := (p - anchors[i-1].pitch) / (anchors[i].pitch - anchors[i-1].pitch)
				return anchors[i-1].damping*(1-f) + anchors[i].damping*f
			}
		}
		return anchors[len(anchors)-1].damping
	}
	// below the notes that are over the noise the string keeps its damping, and the sharp pluck must not make its ring
	// brighter than the real one: the loop filter is gentle at a low pitch, a bright start stays bright for a while. The
	// edge goes no further than the band 640-1280 Hz of the ring allows
	for p := fitFrom; float64(p) <= from; p++ {
		pf := float64(p)
		i := p - fitFrom
		target := reference.TargetBands(pf)[3]
		limit := bisect(0, 1, target, true, func(e float64) float64 { return late640(pf, base.DecayAt(pf), base.DampingAt(pf), e) })
		edges[i] = min(edges[i], limit)
	}
	edges = smooth(edges, smoothWidth)
	dampings := make([]float64, len(edges))
	for i := range dampings {
		dampings[i] = damping(float64(fitFrom + i))
	}
	var decays, damps, edgeTab strings.Builder
	for i := range dampings {
		p := float64(fitFrom + i)
		dm, decay := dampings[i], base.DecayAt(p)
		if p > from {
			// a pitch that the recording has only under its noise: its notes are "at most this bright", and when the
			// anchors around it have the string much brighter (6 dB) than that it is brought down to it
			bound, resolved := math.Inf(1), false
			for _, n := range reference.Notes() {
				if float64(n.Pitch) == p {
					bound = min(bound, n.HighDB)
					resolved = resolved || n.Margin >= reference.ResolvedMargin
				}
			}
			if !resolved && highEdge(p, base.DecayAt(p), dm, edges[i]) > bound+6 {
				dm = max(dm, min(fitDampingEdge(p, bound, edges[i]), 0.7))
			}
			decay = fitDecayEdge(p, dm, edges[i])
		}
		fmt.Printf("%d: edge %.2f damping %+.3f decay %.2f\n", fitFrom+i, edges[i], dm, decay)
		fmt.Fprintf(&decays, "\t\t%.3f, // %d: %.1f Hz, to fall %.1f dB/s\n", decay, fitFrom+i, hz(p), reference.TargetRate(p))
		fmt.Fprintf(&damps, "\t\t%.3f, // %d\n", dm, fitFrom+i)
		fmt.Fprintf(&edgeTab, "\t\t%.3f, // %d: to be %.0f dB above 1280 Hz at the start\n", edges[i], fitFrom+i, reference.TargetEarlyHigh(p))
	}
	src := "// Code generated by cmd/tools/guitarcompare -fit -early; DO NOT EDIT.\n\npackage waveguide\n\n" +
		fmt.Sprintf("// %s is how the strings of the guitar of the reference are plucked and die, at each pitch (see cmd/tools/guitarcompare: -fit -early -var %s -out synths/waveguide/keytrack_own.go).\n// The edge of the pluck is fitted to the start of the notes, the damping (where the recording tells it: the notes over its noise) to the brightness of the ring, the decay to the fall of the level\n", name, name) +
		fmt.Sprintf("var %s = &KeyTrack{\n\tFrom: %d, Ref: %d,\n\tDecay: []float64{\n%s\t},\n\tDamping: []float64{\n%s\t},\n\tEdge: []float64{\n%s\t},\n}\n", name, fitFrom, fitRef, decays.String(), damps.String(), edgeTab.String())
	b, err := format.Source([]byte(src))
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(out, b, 0o644); err != nil {
		panic(err)
	}
	fmt.Println("wrote", out)
}
