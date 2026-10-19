package catalog_test

import (
	"math"
	"strings"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/editor"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
	"github.com/almerlucke/psalm/synths/shaper"
)

// the shaper synth's chains are texts of its module: they start as the preset's, a chain typed
// changes the sound at once, one that can't be read is kept to show and the sound goes on, and a
// factory preset chosen puts its chains in the boxes
func TestShaperChains(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 240, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	p := psalm.New(nil)
	psalm.Seed(1)
	pt := patcher.New(p, panel.New(p, "A"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	p.Out(pt.Out())
	roll, _ := pt.Add("noteroll", "", 0, 0)
	syn, err := pt.Add("shaper", "glass", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []patcher.Cable{{From: roll, To: syn}, {From: syn, To: "out"}} {
		if _, err := pt.Connect(c); err != nil {
			t.Fatal(err)
		}
	}
	pt.SettingsOf(roll).Apply(map[string]float64{"length": 4, "n/0/start": 0, "n/0/length": 4, "n/0/key": 57, "n/0/velocity": 0.9})

	texts := pt.Texts(syn)
	want := shaper.Glass()
	if len(texts) != 4 || texts[0].Key != "chainA" || texts[0].Value != want.Chains[0] || texts[1].Value != want.Chains[1] || texts[3].Value != "" {
		t.Fatalf("texts %+v", texts)
	}

	var all []float64
	render := func(seconds float64) []float64 {
		all = all[:0]
		for range int(seconds * 44100 / 256) {
			p.ResetRun()
			p.Run()
			for _, s := range pt.Out().Output(0).Channels[0] {
				all = append(all, float64(s))
			}
		}
		return all
	}
	goertzel := func(x []float64, freq float64) float64 {
		w := 2 * math.Pi * freq / 44100
		var s1, s2 float64
		for _, v := range x {
			s := v + 2*math.Cos(w)*s1 - s2
			s2, s1 = s1, s
		}
		re, im := s1-s2*math.Cos(w), s2*math.Sin(w)
		return math.Hypot(re, im) / float64(len(x))
	}
	// a sine only, a note of 220 Hz held: no third harmonic. Then a saw: one
	if err := pt.SetText(syn, "chainA", "sin"); err != nil {
		t.Fatal(err)
	}
	pt.SettingsOf(syn).Apply(map[string]float64{"morphAmount": 0, "morph": 0, "level2": 0})
	render(0.4)
	x := render(0.4)
	ratio := func(x []float64) float64 { return goertzel(x, 3*220) / goertzel(x, 220) } // the envelope falls: compare to the fundamental
	sine := ratio(x)
	if err := pt.SetText(syn, "chainA", "bipolar"); err != nil {
		t.Fatal(err)
	}
	if e := pt.TextError(syn, "chainA"); e != "" {
		t.Fatalf("error %q", e)
	}
	render(0.2)
	saw := ratio(render(0.4))
	if !(saw > 20*sine && saw > 0.05) {
		t.Errorf("third harmonic %v for a sine, %v for a saw: typing the chain should change the sound", sine, saw)
	}

	// a chain that can't be read: kept as typed with its error, the sound is the saw still
	if err := pt.SetText(syn, "chainA", "wobble 3"); err != nil {
		t.Fatal(err)
	}
	if e := pt.TextError(syn, "chainA"); e == "" {
		t.Error("no error shown for a chain that can't be read")
	}
	if again := ratio(render(0.4)); math.Abs(again-saw) > saw*0.3 {
		t.Errorf("the saw went from %v to %v with a chain that can't be read", saw, again)
	}
	// chain A can't be left empty
	if _ = pt.SetText(syn, "chainA", "  "); pt.TextError(syn, "chainA") == "" {
		t.Error("an empty chain A is an error")
	}

	// a factory preset chosen: its chains are in the boxes, and what was typed is gone
	if err := pt.SetText(syn, "chainA", "tri"); err != nil {
		t.Fatal(err)
	}
	if err := pt.SettingsOf(syn).(*editor.Editor).Presets().Load("morphPad", false); err != nil {
		t.Fatal(err)
	}
	pt.Wait()
	got := pt.Texts(syn)
	mp := shaper.MorphPad()
	for i := range 4 {
		if got[i].Value != mp.Chains[i] {
			t.Errorf("chain %c after choosing morphPad: %q, want %q", 'A'+i, got[i].Value, mp.Chains[i])
		}
	}
	if e := pt.TextError(syn, "chainA"); e != "" {
		t.Errorf("the old error stays: %q", e)
	}
}

// a script plays the shaper synth and rewrites it: set() sends a chain (read by the script) to the
// voices, and the next notes sound with it
func TestShaperFromScript(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 240, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	p := psalm.New(nil)
	psalm.Seed(1)
	pt := patcher.New(p, panel.New(p, "A"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	p.Out(pt.Out())
	clock, _ := pt.Add("clock", "", 0, 0)
	sc, _ := pt.Add("script", "", 0, 0)
	syn, _ := pt.Add("shaper", "organ", 0, 0)
	for _, c := range []patcher.Cable{{From: clock, To: sc}, {From: sc, To: syn}, {From: syn, To: "out"}} {
		if _, err := pt.Connect(c); err != nil {
			t.Fatal(err)
		}
	}
	pt.SettingsOf(syn).Apply(map[string]float64{"level2": 0})
	if err := pt.SetText(sc, "code", `let bar = 0
function trigger(ev) {
  if (ev.step % 16 === 0) { bar++; set({chainA: bar > 2 ? "bipolar" : "sin", chainB: "", chainC: "", morph: 0, ampSustain: 1}) }
  if (ev.step % 16 === 1) note(57, 0.9, 12)
}`); err != nil {
		t.Fatal(err)
	}
	goertzel := func(x []float64, freq float64) float64 {
		w := 2 * math.Pi * freq / 44100
		var s1, s2 float64
		for _, v := range x {
			s := v + 2*math.Cos(w)*s1 - s2
			s2, s1 = s1, s
		}
		re, im := s1-s2*math.Cos(w), s2*math.Sin(w)
		return math.Hypot(re, im) / float64(len(x))
	}
	var out []float64
	bufs := int(44100 * 4 * 60 / 240 / 256) // a bar of sixteenths at 240 bpm: one second
	for b := 1; b <= 4; b++ {
		out = out[:0]
		for range bufs {
			p.ResetRun()
			p.Run()
			for _, v := range pt.Out().Output(0).Channels[0] {
				out = append(out, float64(v))
			}
		}
		out := out[len(out)/3:]
		ratio := goertzel(out, 3*220) / (goertzel(out, 220) + 1e-12)
		if saw := b >= 3; saw != (ratio > 0.1) {
			t.Errorf("bar %d: third harmonic %v of the fundamental, chain A should be %s", b, ratio, map[bool]string{true: "a saw", false: "a sine"}[saw])
		}
	}
	if st := pt.Status(sc); strings.HasPrefix(st, "error") {
		t.Error(st)
	}
}
