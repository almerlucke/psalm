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
					peak = max(peak, math.Abs(float64(x)))
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
	mod := must(pt.Connect(patcher.Cable{From: lfo, Out: 0, To: mix, In: 12, Key: "level1", Lo: 0, Hi: 1})) // after 8 channels and 4 returns
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
	if m.Channels != 4 || m.NumInputs() != 8 || m.NumOutputs() != 5 {
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
				peak = max(peak, math.Abs(float64(x)))
			}
		}
		return peak
	}
	main, send := 0.0, 0.0
	for range 120 {
		p.ResetRun()
		p.Run()
		main = max(main, peakOf(m.Output(0).Channels))
		send = max(send, peakOf(m.Output(1).Channels))
	}
	if send != 0 {
		t.Errorf("a channel at level 0 sends (post fader): %g", send)
	}
	pt.SettingsOf(mix).Apply(map[string]float64{"level1": 1})
	for range 120 {
		p.ResetRun()
		p.Run()
		main = max(main, peakOf(m.Output(0).Channels))
		send = max(send, peakOf(m.Output(1).Channels))
	}
	if send < 0.01 || main < 0.01 {
		t.Errorf("the send %g, the main out (with the reverb's return) %g", send, main)
	}
}

// modules copied and pasted: their kind, preset, settings and texts, the cables between them (not
// those to other modules), every paste a little further; the output isn't copied
func TestCopyPaste(t *testing.T) {
	p := psalm.New(nil)
	pt := patcher.New(p, panel.New(p, "Test"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	must := func(id string, err error) string {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	clock := must(pt.Add("clock", "", 100, 100))
	seq := must(pt.Add("sequence", "", 300, 100))
	notes := must(pt.Add("notes", "", 500, 100))
	syn := must(pt.Add("subtractive", "pad", 700, 100))
	_ = pt.SetText(seq, "values", "C3 G3")
	pt.SettingsOf(seq).Apply(map[string]float64{"order": 2})
	pt.SettingsOf(syn).Apply(map[string]float64{"cutoff": 900})
	must(pt.Connect(patcher.Cable{From: clock, To: seq}))
	must(pt.Connect(patcher.Cable{From: seq, Out: 1, To: notes}))
	must(pt.Connect(patcher.Cable{From: notes, To: syn}))
	must(pt.Connect(patcher.Cable{From: syn, To: "out"}))

	if n := pt.Copy("out"); n != 0 {
		t.Errorf("the output copied: %d", n)
	}
	if n := pt.Copy(clock, seq, syn, "out"); n != 3 {
		t.Fatalf("copied %d, want 3", n)
	}
	at := map[string]patcher.NodeInfo{}
	byID := func() {
		for _, n := range pt.Nodes() {
			at[n.ID] = n
		}
	}
	ids, err := pt.Paste()
	if err != nil || len(ids) != 3 {
		t.Fatalf("pasted %v: %v", ids, err)
	}
	byID()
	var c2, s2, y2 string
	for _, id := range ids {
		switch at[id].Kind {
		case "clock":
			c2 = id
		case "sequence":
			s2 = id
		case "subtractive":
			y2 = id
		}
	}
	if at[c2].X != 130 || at[c2].Y != 130 || at[y2].Preset != "pad" {
		t.Errorf("pasted: the clock at %v,%v (want 130,130), the synth at preset %q", at[c2].X, at[c2].Y, at[y2].Preset)
	}
	if tx := pt.Texts(s2); len(tx) != 1 || tx[0].Value != "C3 G3" {
		t.Errorf("the sequence's values pasted: %v", tx)
	}
	if v := pt.SettingsOf(s2).Values()["order"]; v != 2 {
		t.Errorf("the sequence's order pasted: %v", v)
	}
	if v := pt.SettingsOf(y2).Values()["cutoff"]; math.Abs(v-900) > 1 {
		t.Errorf("the synth's cutoff pasted: %v", v)
	}
	var between, outside int
	for _, c := range pt.Cables() {
		switch {
		case c.From == c2 && c.To == s2:
			between++
		case c.From == s2 || c.To == y2 || c.From == y2:
			outside++
		}
	}
	if between != 1 || outside != 0 {
		t.Errorf("cables pasted: %d between the modules (want 1), %d to others (want 0)", between, outside)
	}

	// again: further; and after the modules copied are gone
	for _, id := range []string{clock, seq, syn} {
		if err := pt.Remove(id); err != nil {
			t.Fatal(err)
		}
	}
	ids, err = pt.Paste()
	if err != nil || len(ids) != 3 {
		t.Fatalf("pasted again %v: %v", ids, err)
	}
	byID()
	for _, id := range ids {
		if at[id].Kind == "clock" && (at[id].X != 160 || at[id].Y != 160) {
			t.Errorf("the second paste at %v,%v, want 160,160", at[id].X, at[id].Y)
		}
	}
}

// a module can be given a name; a patch kept has it
func TestModuleNameIsKept(t *testing.T) {
	p := psalm.New(nil)
	pt := patcher.New(p, panel.New(p, "Test"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}), patcher.Keep(t.TempDir()))
	id, _ := pt.Add("clock", "", 0, 0)
	if err := pt.SetName(id, "  kick "); err != nil {
		t.Fatal(err)
	}
	if err := pt.SetName("nope", "x"); err == nil {
		t.Error("a name for a module that is not there")
	}
	if err := pt.Save("named"); err != nil {
		t.Fatal(err)
	}
	pt.Clear()
	if err := pt.Load("named"); err != nil {
		t.Fatal(err)
	}
	for _, n := range pt.Nodes() {
		if n.Kind == "clock" {
			if n.Name != "kick" {
				t.Errorf("the name after loading: %q", n.Name)
			}
			return
		}
	}
	t.Error("the clock is gone")
}

// the chord model is made when the grid asks for chords, and its live switch follows
func TestChordsLiveFromTheGrid(t *testing.T) {
	p := psalm.New(nil)
	pt := patcher.New(p, panel.New(p, "Test"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}))
	if err := pt.Chords("live", "", true); err != nil {
		t.Fatal(err)
	}
	kinds := map[string]bool{}
	for _, n := range pt.Nodes() {
		kinds[n.Kind] = true
	}
	if !kinds["song"] || kinds["chordmodel"] || kinds["harmony"] {
		t.Errorf("a song with the model and the harmony inside it is made: %v", kinds)
	}
	if err := pt.Chords("nope", "", false); err == nil {
		t.Error("an action it does not know")
	}
	if _, err := pt.AddVoice("tuba", "", ""); err == nil {
		t.Error("a voice it does not know")
	}
	ids, err := pt.AddVoice("melody", "lead", "")
	if err != nil || len(ids) != 1 {
		t.Errorf("a melody: %v %v", ids, err)
	}
}

// a song has the harmony and the chord model inside it: its chords are the harmony of the patch, and a patch kept has them, the model and the key
func TestSongHasTheHarmonyInside(t *testing.T) {
	p := psalm.New(nil)
	pt := patcher.New(p, panel.New(p, "Test"), "Patch", catalog.Kinds(catalog.Config{Input: midi.NewQueue()}), patcher.Keep(t.TempDir()))
	song, _ := pt.Add("song", "", 0, 0)
	if patcher.HarmonyOf(p.Env()) != nil {
		t.Error("a song without chords claims the harmony of the patch")
	}
	if err := pt.SetText(song, "sections", "A 2, B 2"); err != nil {
		t.Fatal(err)
	}
	if err := pt.SetText(song, "chords", "A: Dm7 G7\nB: Cmaj7"); err != nil || pt.TextError(song, "chords") != "" {
		t.Fatal(err, pt.TextError(song, "chords"))
	}
	h := patcher.HarmonyOf(p.Env())
	if h == nil || len(h.Spans("A", 8, 4)) == 0 {
		t.Fatalf("the chords of the song are not the harmony of the patch: %v", h)
	}
	sf := pt.Module(song).(*patcher.SongForm)
	if sf.Model() == nil || sf.Harmony() != h {
		t.Fatal("the song has no model or harmony inside")
	}
	sf.Model().Preset("minor")
	pt.SettingsOf(song).Apply(map[string]float64{"key": 5, "low": 40})
	if err := pt.Save("with chords"); err != nil {
		t.Fatal(err)
	}
	pt.Clear()
	if err := pt.Load("with chords"); err != nil {
		t.Fatal(err)
	}
	var back string
	for _, n := range pt.Nodes() {
		if n.Kind == "song" {
			back = n.ID
		}
	}
	if back == "" {
		t.Fatal("no song after loading")
	}
	text := ""
	for _, x := range pt.Texts(back) {
		if x.Key == "chords" {
			text = x.Value
		}
	}
	if text != "A: Dm7 G7\nB: Cmaj7" {
		t.Errorf("the chords after loading: %q", text)
	}
	if got := pt.Module(back).(*patcher.SongForm).Model().State().Preset; got != "minor" {
		t.Errorf("the model after loading is %q", got)
	}
	if v := pt.SettingsOf(back).Values(); v["key"] != 5 || v["low"] != 40 {
		t.Errorf("the settings after loading: %v", v)
	}
}
