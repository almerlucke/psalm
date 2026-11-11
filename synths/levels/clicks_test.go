package levels

import (
	"fmt"
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/poly"
	"github.com/almerlucke/psalm/synths/dx"
	"github.com/almerlucke/psalm/synths/modal"
	"github.com/almerlucke/psalm/synths/organ"
	"github.com/almerlucke/psalm/synths/pulsar"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/synths/supersaw"
	"github.com/almerlucke/psalm/synths/vocal"
	"github.com/almerlucke/psalm/synths/waveguide"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
)

// instrument is a synth with its voices, played by the messages of a sequencer
type instrument interface {
	ReceiveMessage(psalm.Message)
	Synthesize()
	Output(int) *psalm.Socket
	Config() psalm.Config
}

// playNotes plays notes (time, length in seconds, pitch) for 2.4 seconds
func playNotes(v instrument, notes [][3]float64) []float64 {
	sr := v.Config().SampleRate
	bs := v.Config().BufferSize
	var x []float64
	next := 0
	for b := 0; b < int(2.4*sr)/bs; b++ {
		t := float64(b*bs) / sr
		for next < len(notes) && notes[next][0] <= t {
			n := notes[next]
			v.ReceiveMessage(psalm.Message{"command": "trigger", "tick": psalm.CtrlTick{}, "duration": n[1], "amplitude": 1.0,
				"message": psalm.Message{"pitch": n[2], "velocity": 0.8}})
			next++
		}
		v.Synthesize()
		for _, s := range v.Output(0).Channels[0] {
			x = append(x, float64(s))
		}
	}
	return x
}

// tolerated are the presets whose notes begin with a burst of their own (a rough voice, a tick)
var tolerated = map[string]bool{"vocal/giant": true, "modal/spaceTine": true, "pulsar/geigerTicks": true, "wavetable/hatMetal": true}

// the notes of the sweep: each starts while the one before still sounds
var stealNotes = [][3]float64{{0.1, 0.5, 57}, {0.35, 0.5, 62}, {0.6, 0.5, 52}, {0.85, 0.5, 64}}

// cuts returns the notes whose start jumps (the largest step of the first 5 ms) by far more than the note makes alone and the sound before it: the sound that was ringing
// was cut off
func cuts(x []float64, alone func(n [3]float64) []float64, sr float64) []string {
	var out []string
	for i, n := range stealNotes[1:] {
		a := alone(n)
		at := int(math.Ceil(n[0]*sr/512)) * 512
		jump, ref := 0.0, 0.0
		for j := at - int(0.02*sr); j < at; j++ {
			ref = max(ref, math.Abs(x[j]-x[j-1]))
		}
		for j := at; j < at+int(0.005*sr); j++ {
			jump = max(jump, math.Abs(x[j]-x[j-1]))
			ref = max(ref, math.Abs(a[j]-a[j-1]))
		}
		if jump > 0.3 && jump > 5*ref+0.05 {
			out = append(out, fmt.Sprintf("note %d at %.3f: step %.3f, alone %.3f", i+2, float64(at)/sr, jump, ref))
		}
	}
	return out
}

func sweep(t *testing.T, name string, mk func(mono bool) instrument) {
	t.Helper()
	alones := map[int][]float64{} // the notes alone, once for both modes
	alone := func(n [3]float64) []float64 {
		k := int(n[0] * 1000)
		if alones[k] == nil {
			fresh()
			alones[k] = playNotes(mk(false), [][3]float64{n})
		}
		return alones[k]
	}
	for _, mode := range []string{"steal", "mono"} {
		build := func() instrument {
			fresh()
			return mk(mode == "mono")
		}
		v := build()
		x := playNotes(v, stealNotes)
		bad := cuts(x, alone, v.Config().SampleRate)
		for _, b := range bad {
			if !tolerated[name] {
				t.Errorf("%s %s: the start of %s", name, mode, b)
			}
		}
	}
}

func one[T poly.Voice](p *poly.Poly[T], mono bool) instrument {
	if mono {
		p.SetMono(true, false)
	}
	return p
}

// TestClickSweep plays a note while another sounds, on every preset of the synths with voices, on one voice (a note takes the voice of the one before) and in mono, and
// looks at the start of the new note for a cut
func TestClickSweep(t *testing.T) {
	if testing.Short() {
		t.Skip("plays every preset of seven synths")
	}
	for n, f := range subtractive.Presets {
		sweep(t, "subtractive/"+n, func(m bool) instrument { return one(subtractive.NewPoly(f(), 1), m) })
	}
	for n, f := range dx.Presets {
		sweep(t, "dx/"+n, func(m bool) instrument { return one(dx.NewPoly(f(), 1, psalm.Stereo), m) })
	}
	for n, f := range modal.Presets {
		sweep(t, "modal/"+n, func(m bool) instrument { return one(modal.NewPoly(f(), 1), m) })
	}
	for n, f := range vocal.Presets {
		sweep(t, "vocal/"+n, func(m bool) instrument { return one(vocal.NewPoly(f(), 1), m) })
	}
	for n, f := range supersaw.Presets {
		sweep(t, "supersaw/"+n, func(m bool) instrument { return one(supersaw.NewPoly(f(), 1), m) })
	}
	for n, f := range pulsar.Presets {
		sweep(t, "pulsar/"+n, func(m bool) instrument { return one(pulsar.NewPoly(f(), 1), m) })
	}
	table := gen.Brightness(32)
	for n, f := range wavetable.Presets {
		sweep(t, "wavetable/"+n, func(m bool) instrument { return one(wavetable.NewPoly(f(table), 1), m) })
	}
	for n, f := range waveguide.Presets {
		cfg := f()
		waveguide.WarmBow(cfg)
		waveguide.WaitBow()
		sweep(t, "waveguide/"+n, func(m bool) instrument { return one(waveguide.NewPoly(f(), 1), m) })
	}
	_ = organ.Presets
}
