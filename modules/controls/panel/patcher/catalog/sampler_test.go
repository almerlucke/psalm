package catalog_test

import (
	"math"
	"os"
	"strings"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
)

const tabla = "../../../../../assets/sounds/various/tabla_loop_4bars_90bpm-86005.wav"

// the sampler's sounds are a text in its side panel: a loop cut at its hits is a kit on the keys from the first, a text that can't be
// read (a file that is not there, a mode it doesn't know) leaves the sounds as they were, and the notes of a roll play the pieces
func TestSamplerSounds(t *testing.T) {
	if _, err := os.Stat(tabla); err != nil {
		t.Skip("no tabla loop in assets")
	}
	psalm.PushConfig(psalm.Config{BPM: 240, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	p := psalm.New(nil)
	pt := patcher.New(p, panel.New(p, "A"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	p.Out(pt.Out())
	roll, _ := pt.Add("noteroll", "", 0, 0)
	syn, err := pt.Add("sampler", "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []patcher.Cable{{From: roll, To: syn}, {From: syn, To: "out"}} {
		if _, err := pt.Connect(c); err != nil {
			t.Fatal(err)
		}
	}
	pt.SettingsOf(roll).Apply(map[string]float64{"length": 4, "n/0/start": 0, "n/0/length": 1, "n/0/key": 38, "n/0/velocity": 0.9})
	if texts := pt.Texts(syn); len(texts) != 1 || texts[0].Key != "sounds" {
		t.Fatalf("texts %+v", texts)
	}
	if err := pt.SetText(syn, "sounds", "36 "+tabla+" hits 9 0.08"); err != nil {
		t.Fatal(err)
	}
	if st := pt.Status(syn); !strings.Contains(st, "on keys 36-79") {
		t.Errorf("status %q", st)
	}
	_ = pt.SetText(syn, "sounds", "36 nothere.wav hits")
	if pt.TextError(syn, "sounds") == "" {
		t.Errorf("a missing file is no error")
	}
	if st := pt.Status(syn); !strings.HasPrefix(st, "error") {
		t.Errorf("status %q after a missing file", st)
	}
	var peak float64
	for range 260 {
		p.ResetRun()
		p.Run()
		for _, s := range pt.Out().Output(0).Channels[0] {
			peak = max(peak, math.Abs(float64(s)))
		}
	}
	if peak < 0.05 {
		t.Errorf("the roll plays a piece of the loop: peak %g", peak)
	}
}

// the grains synth's sounds are a text too: a name, a file and the pitch it is at; the first plays, a script picks another with sound
func TestGrainsSounds(t *testing.T) {
	const chant = "../../../../../assets/sounds/chants/Johannes.Hymnus.wav"
	if _, err := os.Stat(chant); err != nil {
		t.Skip("no chant in assets")
	}
	psalm.PushConfig(psalm.Config{BPM: 240, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	p := psalm.New(nil)
	pt := patcher.New(p, panel.New(p, "A"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	p.Out(pt.Out())
	roll, _ := pt.Add("noteroll", "", 0, 0)
	syn, err := pt.Add("grains", "choirPad", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []patcher.Cable{{From: roll, To: syn}, {From: syn, To: "out"}} {
		if _, err := pt.Connect(c); err != nil {
			t.Fatal(err)
		}
	}
	pt.SettingsOf(roll).Apply(map[string]float64{"length": 4, "n/0/start": 0, "n/0/length": 2, "n/0/key": 62, "n/0/velocity": 0.9})
	_ = pt.SetText(syn, "sounds", "choir "+chant+" 62\nbad")
	if pt.TextError(syn, "sounds") == "" {
		t.Errorf("a line without a file is no error")
	}
	if err := pt.SetText(syn, "sounds", "choir "+chant+" 62"); err != nil || pt.TextError(syn, "sounds") != "" {
		t.Fatalf("%v %s", err, pt.TextError(syn, "sounds"))
	}
	var peak float64
	for range 400 {
		p.ResetRun()
		p.Run()
		for _, s := range pt.Out().Output(0).Channels[0] {
			peak = max(peak, math.Abs(float64(s)))
		}
	}
	if peak < 0.05 {
		t.Errorf("grains of the chant: peak %g", peak)
	}
}

// the vocoder has a voice and a carrier to be connected to
func TestVocoderTakesTwoSources(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	p := psalm.New(nil)
	pt := patcher.New(p, panel.New(p, "A"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	p.Out(pt.Out())
	voc, err := pt.Add("vocoder", "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := pt.Add("sampler", "", 0, 0)
	b, _ := pt.Add("shaper", "stringPad", 0, 0)
	for _, c := range []patcher.Cable{{From: a, To: voc, In: 0}, {From: b, To: voc, In: 1}, {From: voc, To: "out"}} {
		if _, err := pt.Connect(c); err != nil {
			t.Fatal(err)
		}
	}
}

// the grains synth plays sounds that are made: single cycle waves, and the sound that a grain buffer hears
func TestGrainsOscAndLive(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 240, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	p := psalm.New(nil)
	pt := patcher.New(p, panel.New(p, "A"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	p.Out(pt.Out())
	roll, _ := pt.Add("noteroll", "", 0, 0)
	pt.SettingsOf(roll).Apply(map[string]float64{"length": 4, "n/0/start": 0, "n/0/length": 3, "n/0/key": 60, "n/0/velocity": 0.9})
	syn, err := pt.Add("grains", "glassKeys", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []patcher.Cable{{From: roll, To: syn}, {From: syn, To: "out"}} {
		if _, err := pt.Connect(c); err != nil {
			t.Fatal(err)
		}
	}
	peak := func(blocks int) float64 {
		var pk float64
		for range blocks {
			p.ResetRun()
			p.Run()
			for _, s := range pt.Out().Output(0).Channels[0] {
				pk = max(pk, math.Abs(float64(s)))
			}
		}
		return pk
	}

	// waves
	_ = pt.SetText(syn, "sounds", "tone osc sine nonsense")
	if pt.TextError(syn, "sounds") == "" {
		t.Errorf("a wave that is not there is an error")
	}
	if err := pt.SetText(syn, "sounds", "tone osc saw square"); err != nil || pt.TextError(syn, "sounds") != "" {
		t.Fatalf("%v %s", err, pt.TextError(syn, "sounds"))
	}
	if pk := peak(300); pk < 0.02 {
		t.Errorf("grains of a wave: peak %g", pk)
	}

	// a buffer that is not there (yet: a patch loads its modules in order, the grains may come first): no error, silent until it is
	if err := pt.SetText(syn, "sounds", "me live nobody"); err != nil || pt.TextError(syn, "sounds") != "" {
		t.Fatalf("a buffer that comes later is no error: %v %s", err, pt.TextError(syn, "sounds"))
	}

	// a buffer, fed by a synth that plays all the time, and a second one that listens to it
	src, _ := pt.Add("subtractive", "", 0, 0)
	buf, err := pt.Add("grainbuffer", "", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	roll2, _ := pt.Add("noteroll", "", 0, 0)
	pt.SettingsOf(roll2).Apply(map[string]float64{"length": 4, "n/0/start": 0, "n/0/length": 4, "n/0/key": 57, "n/0/velocity": 0.9})
	for _, c := range []patcher.Cable{{From: roll2, To: src}, {From: src, To: buf}, {From: buf, To: "out"}} {
		if _, err := pt.Connect(c); err != nil {
			t.Fatal(err)
		}
	}
	if err := pt.SetText(syn, "sounds", "me live"); err != nil || pt.TextError(syn, "sounds") != "" {
		t.Fatalf("%v %s", err, pt.TextError(syn, "sounds"))
	}
	if err := pt.SetText(buf, "name", "two words"); err == nil && pt.TextError(buf, "name") == "" {
		t.Errorf("a name is one word")
	}
	if pk := peak(600); pk < 0.01 {
		t.Errorf("grains of the sound that came in: peak %g", pk)
	}
}
