package vst3

import (
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/almerlucke/psalm"
)

// testPlugins compiles the test plugins (testdata/testplugin.cpp) into a bundle in a directory and returns it
func testPlugins(t *testing.T) string {
	t.Helper()
	cxx, err := exec.LookPath("c++")
	if err != nil {
		t.Skip("no C++ compiler")
	}
	dir := t.TempDir()
	so := filepath.Join(dir, "test.vst3", "Contents", "x86_64-linux")
	if err := os.MkdirAll(so, 0o755); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(cxx, "-std=c++17", "-shared", "-fPIC", "-I", "include", "-I", ".", "-o", filepath.Join(so, "test.so"), "testdata/testplugin.cpp", "funknown_cpp.cpp", "iids.cpp").CombinedOutput()
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	return dir
}

func find(t *testing.T, dir, name string) Descriptor {
	t.Helper()
	for _, d := range Scan(dir) {
		if d.Name == name {
			return d
		}
	}
	t.Fatalf("no plugin %s in %s", name, dir)
	return Descriptor{}
}

func block(p *Plugin) []float32 {
	p.ResetRun()
	p.Synthesize()
	return append([]float32(nil), p.Output(0).Channels[0]...)
}

func trigger(command map[string]any) psalm.Message {
	m := psalm.Message{"command": "trigger"}
	for k, v := range command {
		m[k] = v
	}
	return m
}

func noteOn(id string, pitch float64) psalm.Message {
	return trigger(map[string]any{"noteOn": id, "amplitude": 1.0, "message": psalm.Message{"pitch": pitch}, "tick": psalm.CtrlTick{}})
}

func peak(bs ...[]float32) (m float64) {
	for _, b := range bs {
		for _, s := range b {
			m = math.Max(m, math.Abs(float64(s)))
		}
	}
	return
}

func TestScan(t *testing.T) {
	dir := testPlugins(t)
	ds := Scan(dir)
	if len(ds) != 2 {
		t.Fatalf("%d plugins: %+v", len(ds), ds)
	}
	if d := find(t, dir, "Psalm VST3 Sine"); !d.Instrument() || d.Effect() || d.Vendor != "psalm" || len(d.CID) != 32 {
		t.Errorf("%+v", d)
	}
	if d := find(t, dir, "Psalm VST3 Gain"); d.Instrument() || !d.Effect() {
		t.Errorf("%+v", d)
	}
}

// the instrument: a note sounds at its pitch (a pitch between the keys by the tuning of the note, in cents), the parameter moves, the note off stops it
func TestInstrument(t *testing.T) {
	dir := testPlugins(t)
	p, err := New(find(t, dir, "Psalm VST3 Sine"))
	if err != nil {
		t.Fatal(err)
	}
	defer p.Free()
	cfg := p.Config()
	if keys := p.Keys(); len(keys) != 1 || keys[0] != "gain" {
		t.Fatalf("keys %v", keys)
	}
	if set := p.Describe(); len(set.Params) != 1 || set.Params[0].Value != 0.5 {
		t.Fatalf("describe %+v", set)
	}
	freq := func(pitch float64) float64 {
		p.AllNotesOff()
		block(p)
		p.ReceiveMessage(noteOn("a", pitch))
		crossings, samples := 0, 0
		var last float32
		for i := 0; i < 40; i++ {
			for _, s := range block(p) {
				if last < 0 && s >= 0 {
					crossings++
				}
				last = s
				samples++
			}
		}
		p.ReceiveMessage(trigger(map[string]any{"noteOff": "a", "tick": psalm.CtrlTick{}}))
		return float64(crossings) / (float64(samples) / cfg.SampleRate)
	}
	for _, pitch := range []float64{69, 57, 69.5, 66.02} {
		want := 440 * math.Pow(2, (pitch-69)/12)
		if got := freq(pitch); math.Abs(got-want)/want > 0.01 {
			t.Errorf("pitch %v: %.1f Hz, want %.1f", pitch, got, want)
		}
	}
	p.AllNotesOff()
	block(p)
	p.ReceiveMessage(noteOn("g", 60))
	loud := peak(block(p), block(p), block(p))
	p.ReceiveMessage(psalm.Message{"gain": 0.1})
	quiet := peak(block(p), block(p), block(p))
	if math.Abs(loud-0.5) > 0.02 || math.Abs(quiet-0.1) > 0.02 {
		t.Errorf("gain: %v then %v, want 0.5 then 0.1", loud, quiet)
	}
	p.ReceiveMessage(trigger(map[string]any{"noteOff": "g", "tick": psalm.CtrlTick{}}))
	block(p)
	if q := peak(block(p), block(p)); q != 0 {
		t.Errorf("the note off still sounds: %v", q)
	}
	// a note of a duration lets go by itself, and a note late in the block starts late
	p.ReceiveMessage(trigger(map[string]any{"duration": 0.01, "amplitude": 1.0, "message": psalm.Message{"pitch": 60.0}, "tick": psalm.CtrlTick{SampleOffset: 100}}))
	first := block(p)
	for _, s := range first[:100] {
		if s != 0 {
			t.Fatal("sounds before its offset")
		}
	}
	if peak(first) == 0 {
		t.Error("the note of a duration does not sound")
	}
	for range int(0.02*cfg.SampleRate)/len(first) + 2 {
		block(p)
	}
	if q := peak(block(p), block(p)); q != 0 {
		t.Errorf("the note of a duration does not end: %v", q)
	}
}

// the ports: an instrument with three outputs gives all three (the second at half level), an effect has a side chain, the tempo reaches it, it says its latency; its sound
// is its state: a state given back brings the gain back
func TestPortsAndState(t *testing.T) {
	dir := testPlugins(t)
	p, err := New(find(t, dir, "Psalm VST3 Sine"))
	if err != nil {
		t.Fatal(err)
	}
	defer p.Free()
	if p.NumOutputs() != 3 {
		t.Fatalf("%d outputs, want 3", p.NumOutputs())
	}
	p.ReceiveMessage(noteOn("a", 69))
	var peak0, peak1, peak2 float64
	for range 6 {
		p.ResetRun()
		p.Synthesize()
		for i := range p.Output(0).Channels[1] {
			peak0 = math.Max(peak0, math.Abs(float64(p.Output(0).Channels[1][i])))
			peak1 = math.Max(peak1, math.Abs(float64(p.Output(1).Channels[0][i])))
			peak2 = math.Max(peak2, math.Abs(float64(p.Output(2).Channels[0][i])))
		}
	}
	if peak0 < 0.4 || math.Abs(peak1-peak0/2) > 0.02 || peak2 != 0 {
		t.Errorf("the outputs peak at %v %v %v", peak0, peak1, peak2)
	}

	fx, err := New(find(t, dir, "Psalm VST3 Gain"))
	if err != nil {
		t.Fatal(err)
	}
	defer fx.Free()
	if fx.NumInputs() != 2 {
		t.Fatalf("an effect with a side chain has %d inputs, want 2", fx.NumInputs())
	}
	if fx.Latency() != 64 {
		t.Errorf("latency %d, want 64", fx.Latency())
	}
	for c := range fx.Input(0).Channels {
		for i := range fx.Input(0).Channels[c] {
			fx.Input(0).Channels[c][i] = 0.5
			fx.Input(1).Channels[c][i] = 0.4
		}
	}
	fx.ResetRun()
	fx.Synthesize()
	if got := fx.Output(0).Channels[0][10]; math.Abs(float64(got)-0.6) > 1e-6 { // the input, and a quarter of the side chain
		t.Errorf("side chain: %v, want 0.6", got)
	}
	fx.CtrlTick(psalm.CtrlTick{BPM: 120, PPQ: 240, TickTimestamp: 480})
	fx.ResetRun()
	fx.Synthesize()
	if got := fx.Output(0).Channels[0][10]; math.Abs(float64(got)-0.612) > 1e-5 { // and the tempo, a ten thousandth of it
		t.Errorf("tempo: %v, want 0.612", got)
	}

	// the state: kept, the gain changed, given back
	fx.ReceiveMessage(psalm.Message{"gain": 0.25}) // 0.5 as the effect has it
	fx.ResetRun()
	fx.Synthesize()
	st, err := fx.SaveState()
	if err != nil {
		t.Fatal(err)
	}
	fx.ReceiveMessage(psalm.Message{"gain": 1.0})
	fx.ResetRun()
	fx.Synthesize()
	if err := fx.LoadState(st); err != nil {
		t.Fatal(err)
	}
	fx.ResetRun()
	fx.Synthesize()
	if got := fx.Output(0).Channels[0][10]; math.Abs(float64(got)-0.5*0.5-0.1-0.012) > 1e-4 {
		t.Errorf("after the state is given back: %v, want 0.362", got)
	}
	if v := fx.Values()["gain"]; math.Abs(v-0.25) > 1e-6 {
		t.Errorf("the parameter after the state is given back: %v, want 0.25", v)
	}
}

// a plugin that renames its parameters tells the host (restart): the editor has the new labels
func TestRelabel(t *testing.T) {
	dir := testPlugins(t)
	p, err := New(find(t, dir, "Psalm VST3 Sine"))
	if err != nil {
		t.Fatal(err)
	}
	defer p.Free()
	labels := make(chan map[string]string, 4)
	p.Bind(func(map[string]float64) {}, func(l map[string]string) { labels <- l })
	p.ReceiveMessage(psalm.Message{"gain": 0.9})
	for i := 0; i < 400; i++ {
		block(p) // the control tick sets the value on the controller, which tells the host
		select {
		case l := <-labels:
			if l["gain"] != "Loudness" {
				t.Fatalf("labels %v", l)
			}
			return
		default:
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Error("the editor was not told the parameter is called Loudness now")
}

// a plugin that crashes (a write to address 0 in its process) does not take the program down: it is dead and silent, and another one goes on
func TestCrash(t *testing.T) {
	dir := testPlugins(t)
	bad, err := New(find(t, dir, "Psalm VST3 Sine"))
	if err != nil {
		t.Fatal(err)
	}
	defer bad.Free()
	good, err := New(find(t, dir, "Psalm VST3 Sine"))
	if err != nil {
		t.Fatal(err)
	}
	defer good.Free()
	good.ReceiveMessage(noteOn("a", 69))
	bad.ReceiveMessage(noteOn("a", 126))
	for range 3 {
		bad.ResetRun()
		bad.Synthesize() // the fault is in here
		good.ResetRun()
		good.Synthesize()
	}
	if sig := bad.Crashed(); sig != 11 {
		t.Fatalf("crashed with the signal %d, want 11", sig)
	}
	if peak(bad.Output(0).Channels[0]) != 0 {
		t.Fatal("a plugin that crashed makes sound")
	}
	if good.Crashed() != 0 || peak(good.Output(0).Channels[0]) < 0.3 {
		t.Errorf("the other plugin: crashed %d, peak %v", good.Crashed(), peak(good.Output(0).Channels[0]))
	}
	bad.ReceiveMessage(noteOn("b", 60))
	bad.ResetRun()
	bad.Synthesize()
}
