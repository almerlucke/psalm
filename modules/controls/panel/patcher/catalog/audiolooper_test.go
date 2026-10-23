package catalog_test

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
	"github.com/almerlucke/psalm/modules/effects/looper"
)

// a synth recorded by the audio looper in a patch: once the synth is cut off, the layer plays on by itself
func TestAudioLooperPlaysBack(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	p := psalm.New(nil)
	pt := patcher.New(p, panel.New(p, "A"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	p.Out(pt.Out())
	roll, _ := pt.Add("noteroll", "", 0, 0)
	syn, err := pt.Add("subtractive", "lead", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	loop, _ := pt.Add("audiolooper", "", 0, 0)
	cables := map[string]string{}
	for name, c := range map[string]patcher.Cable{"roll": {From: roll, To: syn}, "syn": {From: syn, To: loop}, "out": {From: loop, To: "out"}} {
		id, err := pt.Connect(c)
		if err != nil {
			t.Fatal(err)
		}
		cables[name] = id
	}
	pt.SettingsOf(roll).Apply(map[string]float64{"length": 4, "n/0/start": 0, "n/0/length": 3, "n/0/key": 60, "n/0/velocity": 0.9})
	set := pt.SettingsOf(loop)
	set.Apply(map[string]float64{"bars": 1, "dry": 0, "record": 1})
	energy := func(seconds float64) float64 {
		var e float64
		for range int(seconds * 44100 / 256) {
			p.ResetRun()
			p.Run()
			for _, s := range pt.Out().Output(0).Channels[0] {
				e += math.Abs(float64(s))
			}
		}
		return e
	}
	energy(1) // recording, the first half of the pass
	set.Apply(map[string]float64{"record": 0})
	if e := energy(1.3); e != 0 { // dry is off: the pass is not heard while it is recorded
		t.Logf("energy while recording %g", e)
	}
	if n, rec := pt.Module(loop).(*looper.Looper).Layers(); n != 1 || rec {
		t.Fatalf("%d layers, recording %v", n, rec)
	}
	pt.Disconnect(cables["syn"])
	if e := energy(2); e < 1 {
		t.Errorf("the layer is not heard: energy %g", e)
	}
}
