package clap

import (
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/params"
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
	if set := p.Describe(); len(set.Params) != 2 || set.Params[0].Key != "preset" || set.Params[1].Max != 1 || set.Params[1].Value != 0.5 {
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

// presets: two inside the plugin and the files of a directory, listed for the plugin they are for, loaded by number
func TestPresets(t *testing.T) {
	presetDir := t.TempDir()
	for name, gain := range map[string]string{"bell": "0.3", "organ": "0.9"} {
		if err := os.WriteFile(filepath.Join(presetDir, name+".tpreset"), []byte(gain), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(presetDir, "readme.txt"), []byte("not a preset"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PSALM_TEST_PRESETS", presetDir)
	dir := testPlugins(t)
	p, err := New(find(t, dir, "psalm.test.sine"))
	if err != nil {
		t.Fatal(err)
	}
	defer p.Free()
	var names []string
	for _, pr := range p.Presets() {
		names = append(names, pr.label())
	}
	slices.Sort(names)
	folder := filepath.Base(presetDir) // a file is filed under the folder it is in
	if want := []string{folder + " / bell", folder + " / organ", "soft / Quiet", "strong / Loud"}; !slices.Equal(names, want) {
		t.Fatalf("presets %v, want %v", names, want)
	}
	if fx, err := New(find(t, dir, "psalm.test.gain")); err != nil || len(fx.Presets()) != 0 {
		t.Errorf("the effect has no presets of its own: %v %v", err, fx)
	}
	set := p.Describe()
	if len(set.Params) != 2 || set.Params[0].Key != "preset" || set.Params[0].Kind != params.Choice || len(set.Params[0].Options) != 5 {
		t.Fatalf("settings %+v", set.Params)
	}
	gain := func() float64 { // what the sound of a note is: its amplitude is the gain
		p.AllNotesOff()
		block(p)
		p.ReceiveMessage(trigger(map[string]any{"noteOn": "g", "amplitude": 1.0, "message": psalm.Message{"pitch": 60.0}, "tick": psalm.CtrlTick{}}))
		var peak float64
		for range 8 {
			for _, s := range block(p) {
				peak = math.Max(peak, math.Abs(float64(s)))
			}
		}
		return peak
	}
	index := func(label string) int {
		for i, pr := range p.Presets() {
			if pr.label() == label {
				return i
			}
		}
		t.Fatalf("no preset %q", label)
		return -1
	}
	for label, want := range map[string]float64{"soft / Quiet": 0.1, "strong / Loud": 1.0, folder + " / bell": 0.3, folder + " / organ": 0.9} {
		if err := p.LoadPreset(index(label)); err != nil {
			t.Fatal(err)
		}
		if got := gain(); math.Abs(got-want) > 0.03 {
			t.Errorf("%s: the note peaks at %v, want %v", label, got, want)
		}
	}
	// by a message, off the audio thread
	p.ReceiveMessage(psalm.Message{"preset": float64(index("soft / Quiet") + 1)})
	deadline := time.Now().Add(2 * time.Second)
	for p.Loaded() != index("soft / Quiet") && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if got := gain(); math.Abs(got-0.1) > 0.03 {
		t.Errorf("by message: %v", got)
	}
	if p.LoadPreset(99) == nil {
		t.Error("a preset that is not there")
	}
}

// the ports: an instrument with three outputs gives all three (the second at half level), an effect has a side chain, the tempo reaches it, it says its latency
func TestPorts(t *testing.T) {
	dir := testPlugins(t)
	p, err := New(find(t, dir, "psalm.test.sine"))
	if err != nil {
		t.Fatal(err)
	}
	defer p.Free()
	if p.NumOutputs() != 3 {
		t.Fatalf("%d outputs, want 3", p.NumOutputs())
	}
	p.ReceiveMessage(trigger(map[string]any{"noteOn": "a", "amplitude": 1.0, "message": psalm.Message{"pitch": 69.0}, "tick": psalm.CtrlTick{}}))
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
	if _, outs := p.PortNames(); len(outs) != 3 || outs[0] != "main" {
		t.Errorf("port names %v", outs)
	}

	fx, err := New(find(t, dir, "psalm.test.gain"))
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
}

// a plugin that crashes (a write to address 0 in its process) does not take the program down: it is dead and silent, and another one goes on
func TestCrash(t *testing.T) {
	dir := testPlugins(t)
	bad, err := New(find(t, dir, "psalm.test.sine"))
	if err != nil {
		t.Fatal(err)
	}
	defer bad.Free()
	good, err := New(find(t, dir, "psalm.test.sine"))
	if err != nil {
		t.Fatal(err)
	}
	defer good.Free()
	good.ReceiveMessage(trigger(map[string]any{"noteOn": "a", "amplitude": 1.0, "message": psalm.Message{"pitch": 69.0}, "tick": psalm.CtrlTick{}}))
	bad.ReceiveMessage(trigger(map[string]any{"noteOn": "a", "amplitude": 1.0, "message": psalm.Message{"pitch": 126.0}, "tick": psalm.CtrlTick{}}))
	for range 3 {
		bad.ResetRun()
		bad.Synthesize() // the fault is in here
		good.ResetRun()
		good.Synthesize()
	}
	if sig := bad.Crashed(); sig != 11 {
		t.Fatalf("crashed with the signal %d, want 11", sig)
	}
	for _, s := range bad.Output(0).Channels[0] {
		if s != 0 {
			t.Fatal("a plugin that crashed makes sound")
		}
	}
	peak := 0.0
	for _, s := range good.Output(0).Channels[0] {
		peak = math.Max(peak, math.Abs(float64(s)))
	}
	if good.Crashed() != 0 || peak < 0.3 {
		t.Errorf("the other plugin: crashed %d, peak %v", good.Crashed(), peak)
	}
	// it answers nothing now, and nothing else breaks
	if bad.Describe().Name == "" || len(bad.Values()) == 0 {
		t.Error("a plugin that crashed cannot be described")
	}
	bad.ReceiveMessage(trigger(map[string]any{"noteOn": "b", "amplitude": 1.0, "message": psalm.Message{"pitch": 60.0}, "tick": psalm.CtrlTick{}}))
	bad.ResetRun()
	bad.Synthesize()
}
