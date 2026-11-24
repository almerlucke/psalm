// guitarset measures the isolated notes of GuitarSet (Xi, Bittner, Pauwels, Ye, Bello: "GuitarSet: A Dataset for Guitar
// Transcription", ISMIR 2018, CC BY 4.0, doi 10.5281/zenodo.3371780: an acoustic guitar played by six players) with the
// measures of the package reference, and writes them per pitch (synths/waveguide/reference/guitarset.csv): what a
// guitar of that kind does at each pitch, to fit a preset to the way reference.csv is for the nylon string guitar.
//
//	go run ./cmd/tools/guitarset -dir recordings/guitarset
//
// The dir holds the annotation of the dataset (annotation/*.jams) and its microphone audio (audio_mono-mic/*.wav). Only
// the solo takes are used. A note is measured when no other note starts within 250 ms before it and 100 ms after it
// (the early bands: the 2048 samples from 5 ms after its attack, as for the recording of the nylon guitar), and its fall
// and its brightness when it also has 1.5 s to ring (the fall of the level from 0.3-0.5 s to 1.1-1.3 s, and the energy above
// 2 kHz in 8192 samples from 0.25 s)
package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/almerlucke/psalm/blocks/fft"
	"github.com/almerlucke/psalm/synths/waveguide/reference"
)

const sr = 44100.0

type jams struct {
	Annotations []struct {
		Namespace string `json:"namespace"`
		Meta      struct {
			Source string `json:"data_source"`
		} `json:"annotation_metadata"`
		Data json.RawMessage `json:"data"`
	} `json:"annotations"`
}

type note struct {
	Time, Duration, Value float64
}

type measure struct {
	pitch    int
	level    float64
	early    [reference.NumBands]float64
	fall     float64 // dB/s, NaN when the note has no time to ring
	lateHigh float64 // dB, NaN as well
}

func wav(path string) []float64 {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	i := 12
	for i+8 < len(b) && string(b[i:i+4]) != "data" {
		i += 8 + int(binary.LittleEndian.Uint32(b[i+4:]))
	}
	d := b[i+8:]
	x := make([]float64, len(d)/2)
	for k := range x {
		x[k] = float64(int16(binary.LittleEndian.Uint16(d[2*k:]))) / 32768
	}
	return x
}

// attack is the first frame of 1 ms with 1% of the power of the loudest of the 60 ms from x (-1: the note is not in them)
func attack(x []float64) int {
	const frame = 44
	power := func(i int) float64 {
		e := 0.0
		for _, v := range x[i*frame : (i+1)*frame] {
			e += v * v
		}
		return e
	}
	top := 0.0
	for i := range 60 {
		top = max(top, power(i))
	}
	if top < 1e-6 || power(0) > 0.01*top {
		return -1
	}
	for i := range 100 {
		if power(i) > 0.01*top {
			return i * frame
		}
	}
	return -1
}

func db(e float64) float64 { return 10 * math.Log10(e+1e-12) }

func level(x []float64, from, to float64, at int) float64 {
	e := 0.0
	seg := x[at+int(from*sr) : at+int(to*sr)]
	for _, v := range seg {
		e += v * v
	}
	return db(e / float64(len(seg)))
}

// highShare is the energy above 2 kHz against all (70 Hz to 12 kHz) in 8192 samples from 0.25 s after the attack
func highShare(x []float64, at int) float64 {
	const n = 8192
	w := fft.HannWindow(n)
	seg := make([]float64, n)
	for i := range seg {
		seg[i] = x[at+int(0.25*sr)+i] * w[i]
	}
	re, im := make([]float64, n/2+1), make([]float64, n/2+1)
	fft.NewReal(n).Forward(seg, re, im)
	bin := func(hz float64) int { return int(hz * n / sr) }
	total, high := 0.0, 0.0
	for k := bin(70); k <= bin(12000); k++ {
		e := re[k]*re[k] + im[k]*im[k]
		total += e
		if k >= bin(2000) {
			high += e
		}
	}
	return db(high / total)
}

func median(v []float64) float64 {
	v = slices.DeleteFunc(slices.Clone(v), math.IsNaN)
	if len(v) == 0 {
		return math.NaN()
	}
	sort.Float64s(v)
	return v[len(v)/2]
}

func main() {
	dir := flag.String("dir", "recordings/guitarset", "the dataset: annotation/*.jams and audio_mono-mic/*_mic.wav")
	out := flag.String("out", "synths/waveguide/reference/guitarset.csv", "the measures per pitch")
	flag.Parse()
	files, _ := filepath.Glob(*dir + "/annotation/*_solo.jams")
	sort.Strings(files)
	var all []measure
	for _, f := range files {
		base := strings.TrimSuffix(filepath.Base(f), ".jams")
		raw, _ := os.ReadFile(f)
		var j jams
		if err := json.Unmarshal(raw, &j); err != nil {
			panic(err)
		}
		var notes []note
		for _, a := range j.Annotations {
			if a.Namespace == "note_midi" {
				var d []note
				json.Unmarshal(a.Data, &d)
				notes = append(notes, d...)
			}
		}
		sort.Slice(notes, func(a, b int) bool { return notes[a].Time < notes[b].Time })
		x := wav(*dir + "/audio_mono-mic/" + base + "_mic.wav")
		// the noise of the take: its quietest 100 ms
		noise := math.Inf(1)
		for i := 0; i+4410 < len(x); i += 4410 {
			noise = min(noise, level(x, 0, 0.1, i))
		}
		for k, n := range notes {
			if k > 0 && notes[k-1].Time > n.Time-0.25 || k+1 < len(notes) && notes[k+1].Time < n.Time+0.1 {
				continue
			}
			s0 := int((n.Time - 0.03) * sr)
			if s0 < 0 || s0+int(0.2*sr) > len(x) {
				continue
			}
			at := attack(x[s0 : s0+int(0.2*sr)])
			if at < 0 {
				continue
			}
			m := measure{pitch: int(math.Round(n.Value)), fall: math.NaN(), lateHigh: math.NaN()}
			m.early = reference.EarlyBandShares(x[s0:s0+int(0.2*sr)], sr)
			m.level = level(x, 0.03, 0.13, s0+at)
			free := 1e9
			if k+1 < len(notes) {
				free = notes[k+1].Time - n.Time
			}
			start := s0 + at
			if free >= 1.5 && start+int(1.5*sr) < len(x) {
				l1, l2 := level(x, 0.3, 0.5, start), level(x, 1.1, 1.3, start)
				if l2 > noise+8 { // the note over the noise at the end
					m.fall = (l1 - l2) / 0.8
					m.lateHigh = highShare(x, start)
				}
			}
			all = append(all, m)
		}
	}
	by := map[int][]measure{}
	for _, m := range all {
		by[m.pitch] = append(by[m.pitch], m)
	}
	var pitches []int
	for p, v := range by {
		if len(v) >= 5 {
			pitches = append(pitches, p)
		}
	}
	sort.Ints(pitches)
	var sb strings.Builder
	sb.WriteString("pitch,notes,fall_notes,drop_dbps,late_high_db")
	for _, e := range reference.BandEdges {
		fmt.Fprintf(&sb, ",e%d", int(e))
	}
	sb.WriteString(",early_high_db\n")
	for _, p := range pitches {
		v := by[p]
		var falls, lates, highs []float64
		for _, m := range v {
			falls, lates = append(falls, m.fall), append(lates, m.lateHigh)
			highs = append(highs, reference.EarlyHigh(m.early))
		}
		nf := 0
		for _, f := range falls {
			if !math.IsNaN(f) {
				nf++
			}
		}
		fmt.Fprintf(&sb, "%d,%d,%d,%.2f,%.1f", p, len(v), nf, median(falls), median(lates))
		for i := range reference.NumBands {
			var b []float64
			for _, m := range v {
				b = append(b, m.early[i])
			}
			fmt.Fprintf(&sb, ",%.1f", median(b))
		}
		fmt.Fprintf(&sb, ",%.1f\n", median(highs))
	}
	if err := os.WriteFile(*out, []byte(sb.String()), 0o644); err != nil {
		panic(err)
	}
	fmt.Println(len(all), "notes,", len(pitches), "pitches ->", *out)
}
