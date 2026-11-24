// guitarcompare holds the plucked string of the waveguide against a real nylon guitar (the notes of
// synths/waveguide/reference) and prints how far apart they are, per pitch:
//
//	go run ./cmd/tools/guitarcompare            the real guitar and the model's: the Guitar preset, the way the
//	                                            guitarbody example chose the decay before, and the ownGuitar preset
//	go run ./cmd/tools/guitarcompare -chain     the whole chain, the string through the measured body (an impulse
//	                                            response, -ir) and the tone after it (cmd/internal/guitartone), against
//	                                            the spectrum of the real guitar, with the EQ that would close the gap
//	go run ./cmd/tools/guitarcompare -fit       find the decay and the damping per pitch that fall as fast and are as
//	                                            bright as the real guitar, and write them as the KeyTrack of
//	                                            ownGuitar, synths/waveguide/keytrack_own.go
//
// The model is without its body (BodyMix 0: the body is a convolution, see cmd/examples/guitarbody),
// plucked at a velocity of 0.65. What is compared is the fall of the level (dB per second) and the
// brightness (energy above 2 kHz), the first for the decay, the second for the damping of the string: the
// loss of the high partials. The two depend on each other, so the fit goes round them a few times
package main

import (
	"flag"
	"fmt"
	"go/format"
	"math"
	"os"
	"slices"
	"strings"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/cmd/internal/guitartone"
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/psalm/modules/effects/eq"
	"github.com/almerlucke/psalm/synths/waveguide"
	"github.com/almerlucke/psalm/synths/waveguide/reference"
	"github.com/almerlucke/sndfile"
)

const (
	sampleRate  = 44100.0
	velocity    = 0.65
	presetDamp  = 0.35
	fitFrom     = 36
	fitTo       = 84
	fitRef      = 57 // the pitch the settings of the preset hold at (A3, 220 Hz)
	fitRounds   = 3
	maxDamp     = 0.7 // the fit does not go further: the string is a thud beyond it
	smoothWidth = 2   // semitones each way
	bisectSteps = 16
)

// render plays one note of the model (MIDI pitch, with a decay and a damping) for 3 seconds
func render(pitch, decay, damping float64) []float64 {
	return play(noBody(waveguide.Guitar()), psalm.Message{"pitch": pitch, "decay": decay, "damping": damping, "velocity": velocity})
}

// noBody is the string without the resonators of the preset: the body is a convolution
func noBody(cfg waveguide.Config) waveguide.Config {
	cfg.BodyMix = 0
	return cfg
}

// play plays one note for 3 seconds
func play(cfg waveguide.Config, msg psalm.Message) []float64 {
	msg["velocity"] = velocity
	v := waveguide.New(cfg)
	v.NoteOn(1, msg, psalm.CtrlTick{}, v.Config())
	var x []float64
	for len(x) < int(3*sampleRate) {
		v.Synthesize()
		for _, s := range v.Output(0).Channels[0] {
			x = append(x, float64(s))
		}
	}
	return x
}

func rate(pitch, decay, damping float64) float64 {
	return min(reference.DropRate(render(pitch, decay, damping), sampleRate), 300)
}

func high(pitch, decay, damping float64) float64 {
	return reference.HighShare(render(pitch, decay, damping), sampleRate)
}

func hz(pitch float64) float64 { return 440 * math.Exp2((pitch-69)/12) }

// fitDecay finds the decay at which the model falls as fast as the real guitar at a pitch, for a damping:
// the fall grows as the decay gets shorter, so it is a bisection (in the logarithm of the decay)
func fitDecay(pitch, damping float64) float64 {
	target := reference.TargetRate(pitch)
	lo, hi := math.Log(0.3), math.Log(120.0)
	for range bisectSteps {
		mid := (lo + hi) / 2
		if rate(pitch, math.Exp(mid), damping) > target {
			lo = mid // falls too fast: a longer decay
		} else {
			hi = mid
		}
	}
	return math.Exp((lo + hi) / 2)
}

// fitDamping finds the damping at which the model is as bright as the real guitar at a pitch, between 0
// and maxDamp (a brightness it cannot reach is the nearest it can): the brightness grows as the damping
// gets less. The decay changes the brightness too, and the damping the fall: they are fitted in turn
func fitDamping(pitch float64) float64 {
	return fitDampingTo(pitch, reference.TargetHigh(pitch), 0)
}

// fitDampingTo is fitDamping to a brightness, for a damping from lowest (below 0 the loop filter takes less of
// the high partials than it does at 0, down to -1/9, where it takes none: a string that keeps its top)
func fitDampingTo(pitch, target, lowest float64) float64 {
	decay, damping := 7.0, presetDamp
	for range fitRounds {
		lo, hi := lowest, maxDamp
		for range bisectSteps {
			mid := (lo + hi) / 2
			if high(pitch, decay, mid) > target {
				lo = mid // too bright: more damping
			} else {
				hi = mid
			}
		}
		damping = (lo + hi) / 2
		decay = fitDecay(pitch, damping)
	}
	return damping
}

// fitResolved writes a track that keeps the strings of the ownGuitar preset where the recording tells nothing of
// their brightness (it is the hiss there), and from the first pitch it does tell fits the damping and then the decay
// to the notes that are over the noise (with the corrected fall of the recording)
func fitResolved(out, name string) {
	base := waveguide.OwnGuitar().Track
	var dampings []float64
	first := -1
	last := 0.0
	for p := fitFrom; p <= fitTo; p++ {
		target, weight := reference.TargetHighResolved(float64(p))
		if weight < 0.25 {
			if first < 0 {
				dampings = append(dampings, base.DampingAt(float64(p)))
				continue
			}
			// above the highest note of the recording the string is as bright as at it
			target = last
		} else if first < 0 {
			first = p - fitFrom
		}
		last = target
		dampings = append(dampings, fitDampingTo(float64(p), target, -1.0/9))
	}
	if first < 0 {
		panic("no pitch with notes over the noise")
	}
	smoothed := smooth(dampings, smoothWidth)
	var decays, damps strings.Builder
	for i := range dampings {
		p := float64(fitFrom + i)
		dm, decay := dampings[i], base.DecayAt(p)
		note := "kept"
		if i >= first {
			dm = smoothed[i]
			decay = fitDecay(p, dm)
			note = "fitted"
		}
		target, _ := reference.TargetHighResolved(p)
		fmt.Fprintf(&decays, "\t\t%.3f, // %d: %.1f Hz, to fall %.1f dB/s (%s)\n", decay, fitFrom+i, hz(p), reference.TargetRate(p), note)
		if note == "kept" {
			fmt.Fprintf(&damps, "\t\t%.3f, // %d: kept\n", dm, fitFrom+i)
		} else {
			fmt.Fprintf(&damps, "\t\t%.3f, // %d: fitted, to be %.0f dB above 2 kHz\n", dm, fitFrom+i, target)
		}
	}
	src := "// Code generated by cmd/tools/guitarcompare -fit -resolved; DO NOT EDIT.\n\npackage waveguide\n\n" +
		fmt.Sprintf("// %s is how the strings of the guitar of the reference die, at each pitch (see cmd/tools/guitarcompare: -fit -resolved -var %s -out synths/waveguide/keytrack_own.go).\n// Where the recording tells the brightness of the guitar (the notes over its noise) the damping is fitted to it, and may go below 0;\n// below that the table is the one it was made from\n", name, name) +
		fmt.Sprintf("var %s = &KeyTrack{\n\tFrom: %d, Ref: %d,\n\tDecay: []float64{\n%s\t},\n\tDamping: []float64{\n%s\t},\n}\n", name, fitFrom, fitRef, decays.String(), damps.String())
	b, err := format.Source([]byte(src))
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(out, b, 0o644); err != nil {
		panic(err)
	}
	fmt.Println("wrote", out)
}

// smooth is the mean of the values within width of each (the fit is noisy where the damping hardly
// changes the brightness, and a table that jumps from one semitone to the next would be heard)
func smooth(x []float64, width int) []float64 {
	y := make([]float64, len(x))
	for i := range x {
		n := 0
		for j := max(0, i-width); j <= min(len(x)-1, i+width); j++ {
			y[i] += x[j]
			n++
		}
		y[i] /= float64(n)
	}
	return y
}

func main() {
	doFit := flag.Bool("fit", false, "find the decay and the damping per pitch and write the table")
	onlyResolved := flag.Bool("resolved", false, "with -fit: fit the brightness only where the notes of the recording are over its noise (see reference.Note.Margin), with a damping below 0 allowed; the other pitches keep the table the ownGuitar preset has")
	early := flag.Bool("early", false, "with -fit: fit the start of the notes too: the edge of the pluck to the first 50 ms, the damping to the band 640-1280 Hz of the ring, the decay to the fall; writes the Edge table as well")
	out := flag.String("out", "synths/waveguide/keytrack_own.go", "with -fit: the file to write")
	name := flag.String("var", "ownTrack", "with -fit: the name of the KeyTrack in the file")
	steel := flag.Bool("steel", false, "with -fit: fit the steel string guitar to the measures of GuitarSet (reference.SteelEarlyHigh and the others): the edge, the damping and the decay per pitch; -out synths/waveguide/keytrack_steel.go -var steelTrack are the defaults then")
	chain := flag.Bool("chain", false, "measure the whole chain: the string, the measured body and the tone")
	tune := flag.Bool("tune", false, "with -chain: find the EQ that brings the string through the body closest to the real guitar")
	irFile := flag.String("ir", "assets/IR44100/Guitar Body HAW 56.wav", "with -chain: the impulse response of the body")
	flag.Parse()
	psalm.PushConfig(psalm.Config{BPM: 100, PPQ: 240, SampleRate: sampleRate, BufferSize: 1024})

	if *chain {
		compareChain(*irFile, *tune)
		return
	}

	if *doFit && *steel {
		o, n := *out, *name
		if o == "synths/waveguide/keytrack_own.go" {
			o = "synths/waveguide/keytrack_steel.go"
		}
		if n == "ownTrack" {
			n = "steelTrack"
		}
		fitSteel(o, n)
		return
	}

	if *doFit && *early {
		fitEarly(*out, *name)
		return
	}

	if *doFit && *onlyResolved {
		fitResolved(*out, *name)
		return
	}

	if *doFit {
		var dampings []float64
		for p := fitFrom; p <= fitTo; p++ {
			dampings = append(dampings, fitDamping(float64(p)))
		}
		dampings = smooth(dampings, smoothWidth)
		var decays, damps strings.Builder
		for i, dm := range dampings {
			p := float64(fitFrom + i)
			fmt.Fprintf(&decays, "\t\t%.3f, // %d: %.1f Hz, to fall %.1f dB/s\n", fitDecay(p, dm), fitFrom+i, hz(p), reference.TargetRate(p))
			fmt.Fprintf(&damps, "\t\t%.3f, // %d: to be %.0f dB above 2 kHz\n", dm, fitFrom+i, reference.TargetHigh(p))
		}
		src := "// Code generated by cmd/tools/guitarcompare -fit; DO NOT EDIT.\n\npackage waveguide\n\n" +
			fmt.Sprintf("// %s is how the strings of the guitar of the reference die, at each pitch (see cmd/tools/guitarcompare)\n", *name) +
			fmt.Sprintf("var %s = &KeyTrack{\n\tFrom: %d, Ref: %d,\n\tDecay: []float64{\n%s\t},\n\tDamping: []float64{\n%s\t},\n}\n", *name, fitFrom, fitRef, decays.String(), damps.String())
		b, err := format.Source([]byte(src))
		if err != nil {
			panic(err)
		}
		if err := os.WriteFile(*out, b, 0o644); err != nil {
			panic(err)
		}
		fmt.Println("wrote", *out)
		return
	}

	// the real notes per pitch
	byPitch := map[int][]reference.Note{}
	for _, n := range reference.Notes() {
		byPitch[n.Pitch] = append(byPitch[n.Pitch], n)
	}
	var pitches []int
	for p := range byPitch {
		pitches = append(pitches, p)
	}
	slices.Sort(pitches)

	ways := []struct {
		name string
		play func(pitch float64) []float64
	}{
		{"preset", func(p float64) []float64 { return render(p, 3, presetDamp) }},
		{"before", func(p float64) []float64 { return render(p, 7.5*math.Sqrt(max(1, hz(p)/220)), presetDamp) }},
		{"ownGuitar", func(p float64) []float64 { return play(noBody(waveguide.OwnGuitar()), psalm.Message{"pitch": p}) }},
	}
	// the ways: the Guitar preset (decay 3), the decay of the guitarbody example before the fit (7.5, longer up),
	// and the ownGuitar preset
	type res struct{ rate, high float64 }
	got := make([][]res, len(pitches))
	realRate, realHigh := make([]float64, len(pitches)), make([]float64, len(pitches))
	// the brightness is the guitar's only where it is over the hiss of the recording (reference.Note.Margin)
	resolved := make([]bool, len(pitches))
	for i, p := range pitches {
		margin := 0.0
		for _, n := range byPitch[p] {
			realRate[i] += n.DropDBps / float64(len(byPitch[p]))
			realHigh[i] += n.HighDB / float64(len(byPitch[p]))
			margin += n.Margin / float64(len(byPitch[p]))
		}
		resolved[i] = margin >= reference.ResolvedMargin
		for _, w := range ways {
			x := w.play(float64(p))
			got[i] = append(got[i], res{min(reference.DropRate(x, sampleRate), 300), reference.HighShare(x, sampleRate)})
		}
	}

	table := func(title string, real []float64, pick func(res) float64, unit string, resolved []bool) {
		fmt.Printf("%s\n\n%-6s %5s %7s", title, "pitch", "notes", "real")
		for _, w := range ways {
			fmt.Printf(" %11s", w.name)
		}
		fmt.Println()
		errSum := make([]float64, len(ways))
		count := 0
		for i := range pitches {
			if resolved == nil || resolved[i] {
				count++
			}
		}
		for i, p := range pitches {
			mark := " "
			if resolved != nil && !resolved[i] {
				mark = "<" // at most: the notes are not over the noise
			}
			fmt.Printf("%-6d %5d %s%6.1f", p, len(byPitch[p]), mark, real[i])
			for j := range ways {
				v := pick(got[i][j])
				if resolved == nil || resolved[i] {
					errSum[j] += math.Abs(v-real[i]) / float64(count)
				}
				fmt.Printf(" %11.1f", v)
			}
			fmt.Println()
		}
		fmt.Printf("%-6s %5s %7s", "mean", "", "error")
		for _, e := range errSum {
			fmt.Printf(" %11.1f", e)
		}
		fmt.Printf("   (%s", unit)
		if resolved != nil {
			fmt.Printf(", over the %d pitches whose notes are over the noise; < is at most", count)
		}
		fmt.Printf(")\n\n")
	}
	table("the fall of the level (0.3-0.5 s to 2.0-2.2 s)", realRate, func(r res) float64 { return r.rate }, "dB/s", nil)
	table("the brightness: the energy above 2 kHz against all", realHigh, func(r res) float64 { return r.high }, "dB", resolved)
}

// through runs a note through a module with a stereo input and returns its left output
func through(m psalm.Modular, x []float64) []float64 {
	bs := psalm.BufferSize()
	var y []float64
	for start := 0; start < len(x); start += bs {
		for i := range bs {
			v := 0.0
			if start+i < len(x) {
				v = x[start+i]
			}
			m.Input(0).Channels[0][i], m.Input(0).Channels[1][i] = float32(v), float32(v)
		}
		m.Synthesize()
		for _, v := range m.Output(0).Channels[0] {
			y = append(y, float64(v))
		}
	}
	return y
}

// compareChain plays the notes of the real guitar through the model, with and without the body and the
// tone, and prints how each octave band differs from the real guitar's, in dB (the model less the real, as
// the mean over the pitches). The last line is the correction in dB at the centre of each band that would
// close the gap, to make an EQ of
func compareChain(irFile string, tune bool) {
	psalm.New(nil)
	ir := convolver.IRFromSoundFile(sndfile.MustSoundFile[float32](irFile))
	var pitches []int
	seen := map[int]bool{}
	for _, n := range reference.Notes() {
		if !seen[n.Pitch] {
			seen[n.Pitch] = true
			pitches = append(pitches, n.Pitch)
		}
	}
	slices.Sort(pitches)

	// the string and the string through the body, once per pitch
	strings1, ownBody, bodies := map[int][]float64{}, map[int][]float64{}, map[int][]float64{}
	blends := map[float64]map[int][]float64{0.3: {}, 0.5: {}, 0.7: {}}
	for _, p := range pitches {
		x := play(noBody(waveguide.OwnGuitar()), psalm.Message{"pitch": float64(p)})
		strings1[p] = x
		ownBody[p] = play(waveguide.OwnGuitar(), psalm.Message{"pitch": float64(p)})
		bodies[p] = through(convolver.New(ir, convolver.Config{PartitionSize: 128, Mix: 1, Gain: guitartone.ConvolverGain}), x)
		for mix := range blends {
			blends[mix][p] = through(convolver.New(ir, convolver.Config{PartitionSize: 128, Mix: mix, Gain: guitartone.ConvolverGain}), x)
		}
	}
	// score is how far a set of notes (per pitch) is from the real guitar: the model less the real per band
	// (mean over the pitches that have it), and the mean error over all of them. Only the bands that hold
	// the note count: from the octave of its fundamental up
	score := func(notes map[int][]float64) (diff [reference.NumBands]float64, count [reference.NumBands]int, mean float64) {
		n := 0
		for _, p := range pitches {
			got, want := reference.BandShares(notes[p], sampleRate), reference.TargetBands(float64(p))
			for b, lo := range reference.BandEdges {
				if 2*lo < hz(float64(p)) {
					continue
				}
				d := got[b] - want[b]
				diff[b] += d
				count[b]++
				mean += math.Abs(d)
				n++
			}
		}
		for b := range diff {
			if count[b] > 0 {
				diff[b] /= float64(count[b])
			}
		}
		return diff, count, mean / float64(max(1, n))
	}
	toned := func(cfg eq.Config) map[int][]float64 {
		out := map[int][]float64{}
		for _, p := range pitches {
			out[p] = through(eq.New(cfg), bodies[p])
		}
		return out
	}

	if tune {
		cfg := tuneTone(toned, score)
		fmt.Printf("\nthe EQ found (for cmd/internal/guitartone):\n\tc.Low = %#v\n\tc.Mid1 = %#v\n\tc.Mid2 = %#v\n\tc.High = %#v\n\n", cfg.Low, cfg.Mid1, cfg.Mid2, cfg.High)
		guitartoneOverride = &cfg
	}
	tone := guitartone.Tone()
	if guitartoneOverride != nil {
		tone = *guitartoneOverride
	}

	// the blends go through the tone too: the tone is for the body, the string alone is not toned
	toneOf := func(notes map[int][]float64) map[int][]float64 {
		out := map[int][]float64{}
		for _, p := range pitches {
			out[p] = through(eq.New(tone), notes[p])
		}
		return out
	}
	names := []string{"string alone", "preset's own body", "+ IR", "+ IR + tone", "IR 30% + tone", "IR 50% + tone", "IR 70% + tone"}
	sets := []map[int][]float64{strings1, ownBody, bodies, toned(tone), toneOf(blends[0.3]), toneOf(blends[0.5]), toneOf(blends[0.7])}
	diff := make([][reference.NumBands]float64, len(names))
	count := make([][reference.NumBands]int, len(names))
	abs := make([]float64, len(names))
	for i, notes := range sets {
		diff[i], count[i], abs[i] = score(notes)
	}
	fmt.Printf("the model less the real guitar, per octave band, in dB (mean over the pitches that have the band, %d in all)\n\n%-18s", len(pitches), "band (Hz)")
	for _, lo := range reference.BandEdges {
		fmt.Printf(" %5.0f", lo)
	}
	fmt.Printf("   mean error\n")
	for i, name := range names {
		fmt.Printf("%-18s", name)
		for _, d := range diff[i] {
			fmt.Printf(" %+5.1f", d)
		}
		fmt.Printf("   %5.1f dB\n", abs[i])
	}
	last := 3
	fmt.Printf("\nthe gain that would close the gap after the tone, at the centre of each band (Hz):\n")
	for b, lo := range reference.BandEdges {
		fmt.Printf("  %5.0f Hz %+5.1f dB   (%d pitches)\n", lo*math.Sqrt2, -diff[last][b], count[last][b])
	}
}

// the EQ the tuning found, to show in the table instead of the one of guitartone
var guitartoneOverride *eq.Config

// tuneTone searches the EQ: the gains of a low shelf and two peaks and the gain and the corner of a high
// shelf, a step at a time, as long as the error to the real guitar gets less
func tuneTone(toned func(eq.Config) map[int][]float64, score func(map[int][]float64) ([reference.NumBands]float64, [reference.NumBands]int, float64)) eq.Config {
	cfg := eq.DefaultConfig()
	cfg.Low = eq.Band{Freq: 120}
	cfg.Mid1 = eq.Band{Freq: 226, Q: 1.2}
	cfg.Mid2 = eq.Band{Freq: 453, Q: 1}
	cfg.High = eq.Band{Freq: 1500}
	err := func(c eq.Config) float64 { _, _, m := score(toned(c)); return m }
	best := err(cfg)
	fmt.Printf("tuning the EQ: flat %.2f dB\n", best)
	moves := []func(c *eq.Config, up bool){
		func(c *eq.Config, up bool) { c.Low.Gain += step(up, 1) },
		func(c *eq.Config, up bool) { c.Mid1.Gain += step(up, 1) },
		func(c *eq.Config, up bool) { c.Mid2.Gain += step(up, 1) },
		func(c *eq.Config, up bool) { c.High.Gain += step(up, 1) },
		func(c *eq.Config, up bool) { c.High.Freq *= map[bool]float64{true: 1.12, false: 1 / 1.12}[up] },
	}
	for round := 0; round < 12; round++ {
		improved := false
		for _, mv := range moves {
			for _, up := range []bool{true, false} {
				for {
					c := cfg
					mv(&c, up)
					if c.Low.Gain < -8 || c.Low.Gain > 8 || c.Mid1.Gain < -8 || c.Mid1.Gain > 8 || c.Mid2.Gain < -8 || c.Mid2.Gain > 8 ||
						c.High.Gain < -8 || c.High.Gain > 12 || c.High.Freq < 800 || c.High.Freq > 5000 {
						break
					}
					if e := err(c); e < best-0.01 {
						cfg, best, improved = c, e, true
					} else {
						break
					}
				}
			}
		}
		if !improved {
			break
		}
	}
	fmt.Printf("tuned: %.2f dB\n", best)
	return cfg
}

func step(up bool, by float64) float64 {
	if up {
		return by
	}
	return -by
}
