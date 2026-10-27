package catalog_test

import (
	"fmt"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/editor"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
	"github.com/almerlucke/psalm/synths/waveguide"
)

const sr = 44100

// rendered plays a note on a module for some seconds, a buffer at a time through run (which
// synthesizes the patch the module is in), and returns what it made
func rendered(m psalm.Modular, run func(), pitch, seconds float64) []float64 {
	m.ReceiveMessage(psalm.Message{"command": "trigger", "noteOn": "k", "tick": psalm.CtrlTick{}, "amplitude": 0.8,
		"message": psalm.Message{"pitch": pitch, "velocity": 0.8}})
	var x []float64
	for range int(seconds * sr / 256) {
		run()
		for _, s := range m.Output(0).Channels[0] {
			x = append(x, float64(s))
		}
	}
	return x
}

func rmsWin(x []float64, a, b float64) float64 {
	seg := x[int(a*sr):int(b*sr)]
	var e float64
	for _, v := range seg {
		e += v * v
	}
	return math.Sqrt(e / float64(len(seg)))
}

func dB(a, b float64) float64 {
	if a < 1e-6 && b < 1e-6 {
		return 0
	}
	return 20 * math.Log10((a+1e-6)/(b+1e-6))
}

// period is the lag of the strongest repeat in the body of a sound (0: no clear pitch, or silence)
func period(x []float64) int {
	seg := x[int(0.3*sr):int(0.6*sr)]
	var e float64
	for _, v := range seg {
		e += v * v
	}
	if math.Sqrt(e/float64(len(seg))) < 0.01 {
		return 0
	}
	best, lag := 0.0, 0
	for l := 40; l < 800; l++ {
		var c float64
		for i := 0; i+l < len(seg); i++ {
			c += seg[i] * seg[i+l]
		}
		if c > best {
			best, lag = c, l
		}
	}
	if best < 0.6*e {
		return 0
	}
	return lag
}

// the windows compared: the attack, the body, the tail
var windows = [][2]float64{{0, 0.15}, {0.15, 0.6}, {0.8, 1.4}}

// sample picks the presets of a kind that are tried without AUDIT: the first, the last, a few from
// between, and (for the waveguide) one of every model
func sample(name string, presets []string) []string {
	if os.Getenv("AUDIT") != "" || len(presets) <= 8 {
		return presets
	}
	pick := map[string]bool{presets[0]: true, presets[len(presets)-1]: true}
	for _, i := range []int{1, len(presets) / 4, len(presets) / 2, len(presets) * 3 / 4} {
		pick[presets[i]] = true
	}
	if name == "waveguide" {
		models := map[waveguide.Model]bool{}
		for _, n := range presets {
			if m := waveguide.Presets[n]().Model; !models[m] {
				models[m] = true
				pick[n] = true
			}
		}
		pick["violin"], pick["altoFlute"], pick["guitar"] = true, true, true
	}
	var out []string
	for _, n := range presets {
		if pick[n] {
			out = append(out, n)
		}
	}
	return out
}

// TestFactoryPresetsAreThePresetsMade (not for the plugins of the clap kind: their presets are the plugins): choosing a factory preset on the page of a synth (the editor's
// own load, as the side panel of the patcher does) gives the sound of the synth made at that preset,
// whatever preset the page started on. AUDIT=1 tries every preset of every synth
func TestFactoryPresetsAreThePresetsMade(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: sr, BufferSize: 256})
	defer psalm.PopConfig()
	kinds := catalog.Kinds(catalog.Config{Input: midi.NewQueue(), Drums: "../../../../../assets/sounds/drums"})
	for _, k := range kinds {
		if k.Group != "Synths" || len(k.Presets) < 2 || k.Name == "dx" || k.Name == "clap" || k.Name == "vst3" || k.Name == "drummachine" || k.Name == "drumsynth" {
			continue
		}
		t.Run(k.Name, func(t *testing.T) {
			p := psalm.New(nil)
			psalm.Seed(1)
			pt := patcher.New(p, panel.New(p, "Test"), "Patch", kinds)
			p.Out(pt.Out())
			first := k.Presets[0]
			id, err := pt.Add(k.Name, first, 0, 0)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := pt.Connect(patcher.Cable{From: id, To: "out"}); err != nil {
				t.Fatal(err)
			}
			ed := pt.SettingsOf(id).(*editor.Editor)
			syn := pt.Module(id)
			run := func() { p.ResetRun(); p.Run() }
			var bad []string
			tried := sample(k.Name, k.Presets)
			for _, pr := range tried {
				if err := ed.Presets().Load(first, false); err != nil {
					t.Fatal(err)
				}
				pt.Wait()
				syn.ReceiveMessage(psalm.Message{"command": "trigger", "noteOff": "k", "tick": psalm.CtrlTick{}})
				for range 8 {
					run()
				}
				p.Env().Seed(7) // the voices made now draw the numbers the ones made for the reference do
				if err := ed.Presets().Load(pr, false); err != nil {
					t.Fatal(err)
				}
				pt.Wait()
				for range 4 { // the page's controls tick (they must not send the preset over the new synth)
					run()
				}
				got := rendered(syn, run, 57, 1.5)
				psalm.Seed(7)
				ref := psalm.New(nil)
				var want *patcher.Made
				ref.Within(func() { want, err = k.Make(pr) })
				if err != nil {
					t.Fatal(err)
				}
				want0 := rendered(want.Module, want.Module.Synthesize, 57, 1.5)
				var worst float64
				for _, w := range windows {
					worst = math.Max(worst, math.Abs(dB(rmsWin(got, w[0], w[1]), rmsWin(want0, w[0], w[1]))))
				}
				// every synth is held to the preset made: the same level in the windows, the same pitch
				tol, slack := 1.0, 0.03
				if worst > tol {
					bad = append(bad, fmt.Sprintf("%s(%.1fdB)", pr, worst))
				}
				if lg, lw := period(got), period(want0); lg != 0 && lw != 0 && math.Abs(float64(lg-lw)) > slack*float64(lw) {
					bad = append(bad, fmt.Sprintf("%s(period %d against %d)", pr, lg, lw))
				}
				if ed.Base() != pr {
					t.Errorf("%s: the editor's base is %q", pr, ed.Base())
				}
			}
			t.Logf("%d presets tried (from %s): %d differ: %s", len(tried), first, len(bad), strings.Join(bad, " "))
			if len(bad) > 0 {
				t.Errorf("%s: loaded from %s, these sound other than made at the preset: %s", k.Name, first, strings.Join(bad, " "))
			}
		})
	}
}

// a module made at a preset and then given another from its editor is kept as that preset (with the
// settings moved since), and a patch kept and loaded again sounds as it did
func TestFactoryPresetKept(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: sr, BufferSize: 256})
	defer psalm.PopConfig()
	kinds := catalog.Kinds(catalog.Config{Input: midi.NewQueue()})
	dir := t.TempDir()
	build := func() (*psalm.Psalm, *patcher.Patcher) {
		p := psalm.New(nil)
		pt := patcher.New(p, panel.New(p, "Test"), "Patch", kinds, patcher.Keep(dir))
		p.Out(pt.Out())
		return p, pt
	}
	p, pt := build()
	id, err := pt.Add("waveguide", "guitar", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = pt.Connect(patcher.Cable{From: id, To: "out"})
	ed := pt.SettingsOf(id).(*editor.Editor)
	if err := ed.Presets().Load("violin", false); err != nil {
		t.Fatal(err)
	}
	pt.Wait()
	ed.Apply(map[string]float64{"bowForce": 0.3})
	for range 3 {
		p.ResetRun()
		p.Run()
	}
	if got := pt.Nodes(); got[len(got)-1].Preset != "violin" {
		t.Fatalf("the module is kept as %q, want violin", got[len(got)-1].Preset)
	}
	if err := pt.Save("kept"); err != nil {
		t.Fatal(err)
	}
	p2, pt2 := build()
	if err := pt2.Load("kept"); err != nil {
		t.Fatal(err)
	}
	var id2 string
	for _, n := range pt2.Nodes() {
		if n.Kind == "waveguide" {
			id2 = n.ID
		}
	}
	if id2 == "" || pt2.SettingsOf(id2).(*editor.Editor).Values()["bowForce"] != 0.3 {
		t.Fatalf("loaded: %q", id2)
	}
	for range 3 {
		p2.ResetRun()
		p2.Run()
	}
	x := rendered(pt2.Module(id2), func() { p2.ResetRun(); p2.Run() }, 57, 1.0)
	if rmsWin(x, 0.3, 0.6) < 0.01 {
		t.Error("the violin loaded again is silent half a second in")
	}
}
