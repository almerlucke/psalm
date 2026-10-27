package clap

import (
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/almerlucke/psalm"
)

// testPlugins compiles the test plugins (testdata/testplugin.c) into a directory and returns it
func testPlugins(t *testing.T) string {
	t.Helper()
	cc, err := exec.LookPath("cc")
	if err != nil {
		t.Skip("no C compiler")
	}
	dir := t.TempDir()
	out, err := exec.Command(cc, "-shared", "-fPIC", "-O1", "-I", "include", "-o", filepath.Join(dir, "test.clap"), "testdata/testplugin.c", "-lm").CombinedOutput()
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	return dir
}

func find(t *testing.T, dir, id string) Descriptor {
	t.Helper()
	for _, d := range Scan(dir) {
		if d.ID == id {
			return d
		}
	}
	t.Fatalf("no plugin %s in %s", id, dir)
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

func TestScan(t *testing.T) {
	dir := testPlugins(t)
	ds := Scan(dir)
	if len(ds) != 2 {
		t.Fatalf("%d plugins: %+v", len(ds), ds)
	}
	if d := find(t, dir, "psalm.test.sine"); !d.Instrument() || d.Effect() || d.Name != "Psalm Test Sine" {
		t.Errorf("%+v", d)
	}
	if d := find(t, dir, "psalm.test.gain"); d.Instrument() || !d.Effect() {
		t.Errorf("%+v", d)
	}
}

// the instrument: a note sounds at its pitch (a pitch between the keys by the tuning of the note), the parameter moves, the note off stops it
func TestInstrument(t *testing.T) {
	dir := testPlugins(t)
	p, err := New(find(t, dir, "psalm.test.sine"))
	if err != nil {
		t.Fatal(err)
	}
	defer p.Free()
	cfg := p.Config()
	if keys := p.Keys(); len(keys) != 1 || keys[0] != "output gain" {
		t.Fatalf("keys %v", keys)
	}
	if set := p.Describe(); len(set.Params) != 1 || set.Params[0].Max != 1 || set.Params[0].Value != 0.5 {
		t.Fatalf("describe %+v", set)
	}
	freq := func(pitch float64) float64 {
		p.AllNotesOff()
		block(p)
		p.ReceiveMessage(trigger(map[string]any{"noteOn": "a", "amplitude": 0.8, "message": psalm.Message{"pitch": pitch}, "tick": psalm.CtrlTick{}}))
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
	peak := func(bs ...[]float32) (m float64) {
		for _, b := range bs {
			for _, s := range b {
				m = math.Max(m, math.Abs(float64(s)))
			}
		}
		return
	}
	p.AllNotesOff()
	block(p)
	p.ReceiveMessage(trigger(map[string]any{"noteOn": "g", "amplitude": 1.0, "message": psalm.Message{"pitch": 60.0}, "tick": psalm.CtrlTick{}}))
	loud := peak(block(p), block(p), block(p))
	p.ReceiveMessage(psalm.Message{"output gain": 0.1})
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

// the effect: the audio of its input through the plugin
func TestEffect(t *testing.T) {
	dir := testPlugins(t)
	p, err := New(find(t, dir, "psalm.test.gain"))
	if err != nil {
		t.Fatal(err)
	}
	defer p.Free()
	in := p.Input(0).Channels
	for c := range in {
		for i := range in[c] {
			in[c][i] = 0.5
		}
	}
	p.ResetRun()
	p.Synthesize()
	if got := p.Output(0).Channels[1][10]; got != 0.5 {
		t.Errorf("gain 1: %v", got)
	}
	p.ReceiveMessage(psalm.Message{"output gain": 0.5})
	p.Synthesize()
	if got := p.Output(0).Channels[0][10]; got != 0.25 {
		t.Errorf("gain 0.5: %v", got)
	}
	_ = os.Stdout
}
