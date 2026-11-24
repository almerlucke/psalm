package reference

import (
	_ "embed"
	"math"
	"strconv"
	"strings"
	"sync"
)

// The measures of a steel string guitar: GuitarSet (Xi, Bittner, Pauwels, Ye, Bello, "GuitarSet: A Dataset for Guitar
// Transcription", ISMIR 2018, doi 10.5281/zenodo.3371780, Creative Commons Attribution 4.0), six players on an acoustic
// guitar, measured by cmd/tools/guitarset as the median of the isolated notes at each pitch (guitarset.csv). What is in
// it of the dataset are these numbers, not its recordings.

//go:embed guitarset.csv
var steelCSV string

// SteelPitch is the measures of the notes of one pitch
type SteelPitch struct {
	Pitch     int
	Notes     int               // the notes measured for the early bands
	FallNotes int               // the notes that had time to ring: for the fall and the brightness
	DropDBps  float64           // fall of the level, 0.3-0.5 s to 1.1-1.3 s, dB per second (NaN: none measured)
	LateHigh  float64           // energy above 2 kHz against all (70 Hz to 12 kHz), 8192 samples from 0.25 s, dB (NaN: none)
	Early     [NumBands]float64 // the octave bands of the start of the note, as Note.Early
	EarlyHigh float64           // EarlyHigh of the notes of the pitch
}

var (
	steelOnce    sync.Once
	steelPitches []SteelPitch
)

// SteelPitches are the pitches of GuitarSet with at least 5 isolated notes, in order
func SteelPitches() []SteelPitch {
	steelOnce.Do(func() {
		for i, line := range strings.Split(strings.TrimSpace(steelCSV), "\n") {
			if i == 0 {
				continue
			}
			f := strings.Split(line, ",")
			v := make([]float64, len(f))
			for j, s := range f {
				v[j], _ = strconv.ParseFloat(s, 64) // NaN stays NaN
			}
			p := SteelPitch{Pitch: int(v[0]), Notes: int(v[1]), FallNotes: int(v[2]), DropDBps: v[3], LateHigh: v[4], EarlyHigh: v[5+NumBands]}
			copy(p.Early[:], v[5:5+NumBands])
			steelPitches = append(steelPitches, p)
		}
	})
	return steelPitches
}

// steelMean is the mean of a measure over the pitches around one, a Gaussian of width semitones, each counting by the
// notes it has (weight returns it, 0 when it has none)
func steelMean(pitch, width float64, value func(SteelPitch) float64, weight func(SteelPitch) float64) float64 {
	sum, total := 0.0, 0.0
	for _, p := range SteelPitches() {
		v, w := value(p), weight(p)
		if math.IsNaN(v) || w == 0 {
			continue
		}
		d := (float64(p.Pitch) - pitch) / width
		g := math.Exp(-d*d/2) * math.Sqrt(w)
		sum += g * v
		total += g
	}
	return sum / math.Max(total, 1e-9)
}

// SteelEarlyHigh is EarlyHigh of the steel guitar at a pitch: the mean over the pitches around it, two semitones wide
func SteelEarlyHigh(pitch float64) float64 {
	return steelMean(pitch, 2, func(p SteelPitch) float64 { return p.EarlyHigh }, func(p SteelPitch) float64 { return float64(p.Notes) })
}

// SteelRate is the fall of the level (dB per second, as SteelPitch.DropDBps) of the steel guitar at a pitch: the notes that
// had time to ring are few, so it is the mean over the pitches around it, three semitones wide
func SteelRate(pitch float64) float64 {
	return steelMean(pitch, 3, func(p SteelPitch) float64 { return p.DropDBps }, func(p SteelPitch) float64 { return float64(p.FallNotes) })
}

// SteelLateHigh is the brightness of the ring of the steel guitar at a pitch (SteelPitch.LateHigh): the mean over the
// pitches around it, three semitones wide
func SteelLateHigh(pitch float64) float64 {
	return steelMean(pitch, 3, func(p SteelPitch) float64 { return p.LateHigh }, func(p SteelPitch) float64 { return float64(p.FallNotes) })
}
