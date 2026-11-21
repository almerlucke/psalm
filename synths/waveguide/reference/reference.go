// Package reference is what was measured of a real guitar, to hold the plucked string
// of the waveguide against: the notes of the recording (reference.csv, measured by measure.py), the
// two measures used on both the real notes and the model's. cmd/tools/guitarcompare fits the model to them,
// and writes what it found as the KeyTrack of the waveguide's ownGuitar preset.
//
// The recording is the nylon string guitar that is modelled, played for it: 26 single notes, each let ring out, one
// player, one microphone or pickup in one room (reference.csv). Its notes are a few cents flat of A = 440 Hz.
// The measures are for the decay of the strings and their brightness and spectrum, not for the attack
package reference

import (
	_ "embed"
	"math"
	"strconv"
	"strings"
	"sync"

	"github.com/almerlucke/psalm/blocks/fft"
)

//go:embed reference.csv
var csv string

// Note is a note of the recording
type Note struct {
	Onset    float64           // seconds into the recording
	Hz       float64           // the fundamental
	Pitch    int               // MIDI
	Cents    float64           // how far the fundamental is from that pitch
	LevelDB  float64           // 0.3-0.5 s after the onset
	DropDBps float64           // the fall of the level from 0.3-0.5 s to 2.0-2.2 s, dB per second
	HighDB   float64           // energy above 2 kHz against all (70 Hz and up, to 12 kHz), in dB
	Bands    [NumBands]float64 // the same for the octave bands from BandEdges up
}

// NumBands octave bands, from 80 Hz up: 80-160, 160-320 ... 5120-10240
const NumBands = 7

// BandEdges are the lower edges (Hz) of the octave bands
var BandEdges = [NumBands]float64{80, 160, 320, 640, 1280, 2560, 5120}

var (
	once  sync.Once
	notes []Note
)

// Notes returns the notes of the recording, in order
func Notes() []Note {
	once.Do(func() {
		for i, line := range strings.Split(strings.TrimSpace(csv), "\n") {
			if i == 0 {
				continue
			}
			f := strings.Split(line, ",")
			v := make([]float64, len(f))
			for j, s := range f {
				v[j], _ = strconv.ParseFloat(s, 64)
			}
			n := Note{Onset: v[0], Hz: v[1], Pitch: int(v[2]), Cents: v[3], LevelDB: v[4], DropDBps: v[5], HighDB: v[6]}
			copy(n.Bands[:], v[7:])
			notes = append(notes, n)
		}
	})
	return notes
}

// TargetRate is the fall of the level (dB per second) of the real guitar at a pitch: the mean of the
// notes around it, the nearer ones counting more (a Gaussian two semitones wide), which smooths the
// differences between strings and plucks
func TargetRate(pitch float64) float64 {
	sum, weight := 0.0, 0.0
	for _, n := range Notes() {
		d := (float64(n.Pitch) - pitch) / 2
		w := math.Exp(-d * d / 2)
		sum += w * n.DropDBps
		weight += w
	}
	return sum / math.Max(weight, 1e-9)
}

// TargetHigh is the brightness (HighShare, in dB) of the real guitar at a pitch: the mean of the notes
// around it, three semitones wide, wider than for the decay, since the brightness differs more from
// note to note
func TargetHigh(pitch float64) float64 {
	sum, weight := 0.0, 0.0
	for _, n := range Notes() {
		d := (float64(n.Pitch) - pitch) / 3
		w := math.Exp(-d * d / 2)
		sum += w * n.HighDB
		weight += w
	}
	return sum / math.Max(weight, 1e-9)
}

// TargetBands is the spectrum of the real guitar at a pitch: the level of each octave band against all
// (dB), the mean of the notes around it as for TargetHigh
func TargetBands(pitch float64) [NumBands]float64 {
	var sum [NumBands]float64
	weight := 0.0
	for _, n := range Notes() {
		d := (float64(n.Pitch) - pitch) / 3
		w := math.Exp(-d * d / 2)
		for i, b := range n.Bands {
			sum[i] += w * b
		}
		weight += w
	}
	for i := range sum {
		sum[i] /= math.Max(weight, 1e-9)
	}
	return sum
}

// DropRate is the fall of the level of a note, in dB per second, measured as it is for the real notes:
// from 0.3-0.5 s to 2.0-2.2 s after its start. x holds at least 2.2 s
func DropRate(x []float64, sr float64) float64 {
	level := func(from, to float64) float64 {
		e := 0.0
		for _, s := range x[int(from*sr):int(to*sr)] {
			e += s * s
		}
		return 10 * math.Log10(e/((to-from)*sr)+1e-12)
	}
	return (level(0.3, 0.5) - level(2.0, 2.2)) / 1.7
}

// spectrum is the energy per bin of the window HighShare and BandShares look at: 32768 samples from 0.25 s
// after the start of a note, Hann windowed. x holds that window
func spectrum(x []float64, sr float64) (energy []float64, bin func(hz float64) int) {
	const n = 32768
	start := int(0.25 * sr)
	seg := make([]float64, n)
	w := fft.HannWindow(n)
	for i := range seg {
		seg[i] = x[start+i] * w[i]
	}
	re, im := make([]float64, n/2+1), make([]float64, n/2+1)
	fft.NewReal(n).Forward(seg, re, im)
	energy = make([]float64, n/2+1)
	for k := range energy {
		energy[k] = re[k]*re[k] + im[k]*im[k]
	}
	return energy, func(hz float64) int { return int(hz * n / sr) }
}

// HighShare is the energy above 2 kHz against all (70 Hz to 12 kHz, the band of the recording) in dB,
// in the window of 32768 samples from 0.25 s after the start of a note. x holds that window
func HighShare(x []float64, sr float64) float64 {
	energy, bin := spectrum(x, sr)
	total, high := 0.0, 0.0
	for k := bin(70); k <= bin(12000); k++ {
		total += energy[k]
		if k >= bin(2000) {
			high += energy[k]
		}
	}
	return 10 * math.Log10(high/total+1e-12)
}

// BandShares is the level of each octave band (BandEdges) against all energy from 70 Hz to 12 kHz, in
// dB, in the same window
func BandShares(x []float64, sr float64) [NumBands]float64 {
	energy, bin := spectrum(x, sr)
	total := 0.0
	for k := bin(70); k <= bin(12000); k++ {
		total += energy[k]
	}
	var out [NumBands]float64
	for i, lo := range BandEdges {
		e := 0.0
		for k := bin(lo); k < bin(2*lo); k++ {
			e += energy[k]
		}
		out[i] = 10 * math.Log10(e/total+1e-12)
	}
	return out
}
