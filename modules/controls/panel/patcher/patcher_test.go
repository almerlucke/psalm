package patcher_test

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
)

// a patch made in code as the page makes it: a keyboard plays a synth through a chorus into the
// output, an LFO moves a mixer's level; kept, cleared and made again from the file
func TestPatch(t *testing.T) {
	p := psalm.New(nil)
	q := midi.NewQueue()
	ctl := panel.New(p, "Test")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: q}), patcher.Keep(t.TempDir()))
	p.Out(pt.Out()) // what pulls the patch
	run := func(buffers int) float64 {
		peak := 0.0
		for range buffers {
			p.ResetRun()
			p.Run()
			for _, ch := range pt.Out().Output(0).Channels {
				for _, x := range ch {
					peak = math.Max(peak, math.Abs(float64(x)))
				}
			}
		}
		return peak
	}
	must := func(id string, err error) string {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	kb := must(pt.Add("keyboard", "", 10, 10))
	synth := must(pt.Add("subtractive", "lead", 200, 10))
	ch := must(pt.Add("chorus", "", 400, 10))
	must(pt.Connect(patcher.Cable{From: kb, Out: 0, To: synth, In: 0}))
	must(pt.Connect(patcher.Cable{From: synth, Out: 0, To: ch, In: 0}))
	must(pt.Connect(patcher.Cable{From: ch, Out: 0, To: "out", In: 0}))
	if _, err := pt.Connect(patcher.Cable{From: kb, Out: 0, To: ch, In: 0}); err == nil {
		t.Errorf("notes into an audio input should be refused")
	}

	if peak := run(20); peak != 0 {
		t.Fatalf("sound before a key is played: %g", peak)
	}
	q.PushOffset(midi.NoteOnMsg(1, 60, 100), 0)
	if peak := run(40); peak < 0.01 {
		t.Fatalf("the key doesn't sound through the patch: peak %g", peak)
	}

	// an LFO on a mixer's first level, over all of its range
	mix := must(pt.Add("mixer", "", 600, 10))
	lfo := must(pt.Add("lfo", "", 600, 200))
	mod := must(pt.Connect(patcher.Cable{From: lfo, Out: 0, To: mix, In: 10, Key: "level1", Lo: 0, Hi: 1})) // after 8 channels and 2 returns
	pt.SettingsOf(lfo).Apply(map[string]float64{"rate": 10})
	levels := map[float64]bool{}
	for range 200 {
		run(1)
		levels[math.Round(pt.Module(mix).(*patcher.Mixer).Level[0]*10)/10] = true
	}
	if len(levels) < 8 {
		t.Errorf("the LFO moved the level through %d values", len(levels))
	}
	pt.Disconnect(mod)

	// the tempo, from its control (in the bar at the top)
	pt.Tempo().Set(97)
	run(2)
	if bpm := p.BPM(); bpm != 97 {
		t.Fatalf("the tempo set to 97 is %g", bpm)
	}

	// kept, cleared, made again: the modules, the cables, the settings, the tempo
	pt.SettingsOf(ch).Apply(map[string]float64{"mix": 0.9})
	if err := pt.Save("test"); err != nil {
		t.Fatal(err)
	}
	pt.Clear()
	if n := len(pt.Nodes()); n != 1 {
		t.Fatalf("cleared: %d modules (the output stays)", n)
	}
	pt.Tempo().Set(150)
	run(2)
	if err := pt.Load("test"); err != nil {
		t.Fatal(err)
	}
	run(2)
	if bpm := p.BPM(); bpm != 97 || pt.Tempo().Value() != 97 {
		t.Errorf("the tempo loaded: %g (its control %g), want 97", bpm, pt.Tempo().Value())
	}
	nodes := pt.Nodes()
	if len(nodes) != 6 || len(pt.Cables()) != 3 {
		t.Fatalf("loaded %d modules, %d cables", len(nodes), len(pt.Cables()))
	}
	for _, n := range nodes {
		if n.Kind == "chorus" {
			if v := pt.SettingsOf(n.ID).Values()["mix"]; math.Abs(v-0.9) > 1e-9 {
				t.Errorf("the chorus mix loaded %g, want 0.9", v)
			}
		}
	}
	q.PushOffset(midi.NoteOnMsg(1, 64, 100), 0)
	if peak := run(40); peak < 0.01 {
		t.Errorf("the patch loaded doesn't sound: %g", peak)
	}
	if err := pt.Remove("out"); err == nil {
		t.Errorf("the output can't go")
	}
}

// a mixer's send: a channel sent to a reverb whose output comes back on a return
func TestMixerSends(t *testing.T) {
	p := psalm.New(nil)
	q := midi.NewQueue()
	ctl := panel.New(p, "Test")
	pt := patcher.New(p, ctl, "Patch", catalog.Kinds(catalog.Config{Input: q}))
	p.Out(pt.Out())
	must := func(id string, err error) string {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	kb := must(pt.Add("keyboard", "", 0, 0))
	synth := must(pt.Add("subtractive", "lead", 0, 0))
	mix := must(pt.Add("mixer", "4 channels", 0, 0))
	rev := must(pt.Add("reverb", "", 0, 0))
	m := pt.Module(mix).(*patcher.Mixer)
	if m.Channels != 4 || m.NumInputs() != 6 || m.NumOutputs() != 3 {
		t.Fatalf("a mixer of 4 channels: %d channels, %d inputs, %d outputs", m.Channels, m.NumInputs(), m.NumOutputs())
	}
	must(pt.Connect(patcher.Cable{From: kb, To: synth}))
	must(pt.Connect(patcher.Cable{From: synth, To: mix, In: 0}))
	must(pt.Connect(patcher.Cable{From: mix, Out: 1, To: rev})) // send A to the reverb
	must(pt.Connect(patcher.Cable{From: rev, To: mix, In: 4}))  // back on return A
	must(pt.Connect(patcher.Cable{From: mix, Out: 0, To: "out"}))
	pt.SettingsOf(mix).Apply(map[string]float64{"sendA1": 1, "level1": 0}) // the channel only through the reverb
	q.PushOffset(midi.NoteOnMsg(1, 60, 100), 0)
	peakOf := func(ch psalm.Channels) float64 {
		peak := 0.0
		for _, c := range ch {
			for _, x := range c {
				peak = math.Max(peak, math.Abs(float64(x)))
			}
		}
		return peak
	}
	main, send := 0.0, 0.0
	for range 120 {
		p.ResetRun()
		p.Run()
		main = math.Max(main, peakOf(m.Output(0).Channels))
		send = math.Max(send, peakOf(m.Output(1).Channels))
	}
	if send != 0 {
		t.Errorf("a channel at level 0 sends (post fader): %g", send)
	}
	pt.SettingsOf(mix).Apply(map[string]float64{"level1": 1})
	for range 120 {
		p.ResetRun()
		p.Run()
		main = math.Max(main, peakOf(m.Output(0).Channels))
		send = math.Max(send, peakOf(m.Output(1).Channels))
	}
	if send < 0.01 || main < 0.01 {
		t.Errorf("the send %g, the main out (with the reverb's return) %g", send, main)
	}
}
