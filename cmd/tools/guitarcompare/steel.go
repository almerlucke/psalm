package main

import (
	"fmt"
	"go/format"
	"math"
	"os"
	"strings"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/fft"
	"github.com/almerlucke/psalm/synths/waveguide"
	"github.com/almerlucke/psalm/synths/waveguide/reference"
)

// The fit of the steel string guitar: the edge of the pluck to the start of the notes of GuitarSet (the share of the
// energy above 1280 Hz in the first 50 ms), the damping to the brightness of the ring (the energy above 2 kHz from 0.25 s),
// the decay to the fall of the level (0.3-0.5 s to 1.1-1.3 s). The three depend on each other, so it goes round them

const (
	steelRounds = 3
	steelDamp   = 0.45 // the highest damping: the recordings of the low notes are mostly the body (the model has none here), so the ring there is not fitted further
	steelDecay  = 40.0 // the longest decay: beyond it the fall is the damping's
)

// steelRender plays a note of the steel string guitar of the preset, without the track and the body, for 3 seconds
// steelEdgeScale is how much of the fitted edge the steel guitar is played with, by ear: the fitted one is harsh in a strum.
// The model has no body to fill the low end that the recordings have, and the low strings were the rasp: 0.28 of the fit
// up to pitch 55 (X4 of the listening), 0.7 from 58 up, in a line between
func steelEdgeScale(pitch float64) float64 {
	return 0.28 + (0.7-0.28)*max(0, min(1, (pitch-55)/3))
}

func steelRender(pitch, decay, damping, edge float64) []float64 {
	cfg := noBody(waveguide.SteelGuitar())
	cfg.Track, cfg.Touch, cfg.OpenStrings, cfg.Edge = nil, 0, 0, edge
	return play(cfg, psalm.Message{"pitch": pitch, "decay": decay, "damping": damping})
}

func levelS(x []float64, from, to float64) float64 {
	e := 0.0
	seg := x[int(from*sampleRate):int(to*sampleRate)]
	for _, s := range seg {
		e += s * s
	}
	return 10 * math.Log10(e/float64(len(seg))+1e-12)
}

// fallS is the fall of the level, 0.3-0.5 s to 1.1-1.3 s (dB per second), as cmd/tools/guitarset measures it
func fallS(x []float64) float64 {
	return min((levelS(x, 0.3, 0.5)-levelS(x, 1.1, 1.3))/0.8, 300)
}

// lateHighS is the energy above 2 kHz against all (70 Hz to 12 kHz), 8192 samples from 0.25 s, dB
func lateHighS(x []float64) float64 {
	const n = 8192
	w := fft.HannWindow(n)
	seg := make([]float64, n)
	for i := range seg {
		seg[i] = x[int(0.25*sampleRate)+i] * w[i]
	}
	re, im := make([]float64, n/2+1), make([]float64, n/2+1)
	fft.NewReal(n).Forward(seg, re, im)
	bin := func(hz float64) int { return int(hz * n / sampleRate) }
	total, high := 0.0, 0.0
	for k := bin(70); k <= bin(12000); k++ {
		e := re[k]*re[k] + im[k]*im[k]
		total += e
		if k >= bin(2000) {
			high += e
		}
	}
	return 10 * math.Log10(high/total+1e-12)
}

func earlyS(x []float64) float64 {
	return reference.EarlyHigh(reference.EarlyBandShares(x, sampleRate))
}

func fitSteel(out, name string) {
	n := fitTo - fitFrom + 1
	decays, dampings, edges := make([]float64, n), make([]float64, n), make([]float64, n)
	for i := range n {
		pf := float64(fitFrom + i)
		tEarly, tFall, tLate := reference.SteelEarlyHigh(pf), reference.SteelRate(pf), reference.SteelLateHigh(pf)
		decay, damping, edge := 4.2, 0.18, 0.4
		for range steelRounds {
			edge = bisect(0, 1, tEarly, true, func(e float64) float64 { return earlyS(steelRender(pf, decay, damping, e)) })
			damping = bisect(-1.0/9, steelDamp, tLate, false, func(d float64) float64 { return lateHighS(steelRender(pf, decay, d, edge)) })
			decay = math.Exp(bisect(math.Log(0.3), math.Log(steelDecay), tFall, false, func(l float64) float64 { return fallS(steelRender(pf, math.Exp(l), damping, edge)) }))
		}
		decays[i], dampings[i], edges[i] = decay, damping, edge
	}
	// a table that jumps from one semitone to the next would be heard
	edges, dampings = smooth(edges, 1), smooth(dampings, 1)
	for i := range edges {
		edges[i] *= steelEdgeScale(float64(fitFrom + i))
	}
	logs := make([]float64, n)
	for i := range decays {
		logs[i] = math.Log(decays[i])
	}
	for i, l := range smooth(logs, 1) {
		decays[i] = math.Exp(l)
	}
	var dec, dam, edg strings.Builder
	var errEarly, errLate, errFall float64
	for i := range n {
		pf := float64(fitFrom + i)
		x := steelRender(pf, decays[i], dampings[i], edges[i])
		e, l, f := earlyS(x), lateHighS(x), fallS(x)
		tE, tL, tF := reference.SteelEarlyHigh(pf), reference.SteelLateHigh(pf), reference.SteelRate(pf)
		errEarly += math.Abs(e-tE) / float64(n)
		errLate += math.Abs(l-tL) / float64(n)
		errFall += math.Abs(f-tF) / float64(n)
		fmt.Printf("%d: edge %.2f damping %+.3f decay %6.2f   early %6.1f (%6.1f)  late %6.1f (%6.1f)  fall %5.1f (%5.1f)\n", fitFrom+i, edges[i], dampings[i], decays[i], e, tE, l, tL, f, tF)
		fmt.Fprintf(&dec, "\t\t%.3f, // %d: %.1f Hz, to fall %.1f dB/s\n", decays[i], fitFrom+i, hz(pf), tF)
		fmt.Fprintf(&dam, "\t\t%.3f, // %d: %.0f dB above 2 kHz in the ring\n", dampings[i], fitFrom+i, tL)
		fmt.Fprintf(&edg, "\t\t%.3f, // %d: %.0f dB above 1280 Hz at the start\n", edges[i], fitFrom+i, tE)
	}
	fmt.Printf("mean error: early %.1f dB, late %.1f dB, fall %.1f dB/s\n", errEarly, errLate, errFall)
	src := "// Code generated by cmd/tools/guitarcompare -fit -steel; DO NOT EDIT.\n\npackage waveguide\n\n" +
		fmt.Sprintf("// %s is how the strings of the steel string guitar of GuitarSet (see package reference) are plucked and die, at each pitch: the edge of the pluck to the start of its notes, the damping to the brightness of the ring, the decay to the fall of the level\n", name) +
		fmt.Sprintf("var %s = &KeyTrack{\n\tFrom: %d, Ref: %d,\n\tDecay: []float64{\n%s\t},\n\tDamping: []float64{\n%s\t},\n\tEdge: []float64{\n%s\t},\n}\n", name, fitFrom, fitRef, dec.String(), dam.String(), edg.String())
	b, err := format.Source([]byte(src))
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(out, b, 0o644); err != nil {
		panic(err)
	}
	fmt.Println("wrote", out)
}
