package patcher_test

import (
	"fmt"
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/editor"
	"github.com/almerlucke/psalm/modules/controls/panel/paneltest"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
	"github.com/almerlucke/psalm/modules/poly"
	"github.com/almerlucke/psalm/synths/subtractive"
)

// presets changed while a sequence plays: once the notes stop, every voice ends (none held on, no
// NaN), whatever the order of the presets and when they come
func TestPresetsWhilePlaying(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	for seed := range uint64(6) {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			rng := rand.New(rand.NewPCG(seed, 7))
			p := psalm.New(nil)
			kinds := catalog.Kinds(catalog.Config{Input: midi.NewQueue()})
			pt := patcher.New(p, panel.New(p, "Test"), "Patch", kinds)
			p.Out(pt.Out())
			var kind patcher.Kind
			for _, k := range kinds {
				if k.Name == "subtractive" {
					kind = k
				}
			}
			made, _ := kind.Make(kind.Presets[0])
			must := func(id string, err error) string {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
				return id
			}
			clock := must(pt.Add("clock", "", 0, 0))
			seq := must(pt.Add("sequence", "", 0, 0))
			notes := must(pt.Add("notes", "", 0, 0))
			syn := must(pt.Add("subtractive", kind.Presets[rng.IntN(len(kind.Presets))], 0, 0))
			pt.SettingsOf(clock).Apply(map[string]float64{"division": float64(rng.IntN(6))})
			pt.SettingsOf(notes).Apply(map[string]float64{"length": []float64{0.3, 0.9, 2, 4}[rng.IntN(4)]})
			trig := must(pt.Connect(patcher.Cable{From: clock, To: seq}))
			must(pt.Connect(patcher.Cable{From: seq, Out: 0, To: notes, In: 1}))
			must(pt.Connect(patcher.Cable{From: seq, Out: 1, To: notes, In: 0}))
			must(pt.Connect(patcher.Cable{From: notes, To: syn}))
			lfo := must(pt.Add("lfo", "", 0, 0))
			mix := must(pt.Add("mixer", "4 channels", 0, 0))
			rev := must(pt.Add("reverb", "", 0, 0))
			must(pt.Connect(patcher.Cable{From: lfo, To: syn, In: 1, Key: "cutoff", Lo: 0.45, Hi: 0.8}))
			must(pt.Connect(patcher.Cable{From: syn, To: mix, In: 0}))
			must(pt.Connect(patcher.Cable{From: mix, Out: 1, To: rev}))
			must(pt.Connect(patcher.Cable{From: rev, To: mix, In: 4}))
			must(pt.Connect(patcher.Cable{From: mix, Out: 0, To: "out"}))
			pt.SettingsOf(mix).Apply(map[string]float64{"sendA1": 0.4})
			ed := pt.SettingsOf(syn).(*editor.Editor)
			voices := pt.Module(syn).(*poly.Poly[*subtractive.Voice])
			active := func() (n int) {
				voices.CallVoices(func(v *subtractive.Voice) {
					if v.IsActive() {
						n++
					}
				})
				return
			}
			nan := false
			run := func(seconds float64) (peak float64) {
				for range max(1, int(seconds*44100/256)) {
					p.ResetRun()
					p.Run()
					for _, ch := range pt.Out().Output(0).Channels {
						for _, s := range ch {
							if math.IsNaN(float64(s)) || math.IsInf(float64(s), 0) {
								nan = true
							}
							peak = math.Max(peak, math.Abs(float64(s)))
						}
					}
				}
				return
			}
			var loud float64
			for range 60 {
				ed.Load(made.Describe(kind.Presets[rng.IntN(len(kind.Presets))]))
				loud = math.Max(loud, run(rng.Float64()*0.3))
			}
			if loud < 0.01 {
				t.Fatalf("it did not play: peak %.4f", loud)
			}
			pt.Disconnect(trig)
			run(30) // the longest release, the reverb's tail
			if n, peak := active(), run(1); n > 0 || peak > 1e-3 || nan {
				t.Errorf("20 s after the last note: %d voices on, peak %.4f, NaN %v", n, peak, nan)
			}
		})
	}
}

// keys held while the preset changes: let go, every voice ends
func TestPresetsWhileHeld(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	p := psalm.New(nil)
	keys := midi.NewQueue()
	kinds := catalog.Kinds(catalog.Config{Input: keys})
	pt := patcher.New(p, panel.New(p, "Test"), "Patch", kinds)
	p.Out(pt.Out())
	var kind patcher.Kind
	for _, k := range kinds {
		if k.Name == "subtractive" {
			kind = k
		}
	}
	made, _ := kind.Make(kind.Presets[0])
	kb, _ := pt.Add("keyboard", "", 0, 0)
	syn, _ := pt.Add("subtractive", kind.Presets[0], 0, 0)
	_, _ = pt.Connect(patcher.Cable{From: kb, To: syn})
	_, _ = pt.Connect(patcher.Cable{From: syn, To: "out"})
	ed := pt.SettingsOf(syn).(*editor.Editor)
	run := func(seconds float64) (peak float64) {
		for range max(1, int(seconds*44100/256)) {
			p.ResetRun()
			p.Run()
			for _, ch := range pt.Out().Output(0).Channels {
				for _, s := range ch {
					peak = math.Max(peak, math.Abs(float64(s)))
				}
			}
		}
		return
	}
	note := 48
	var loud float64
	for _, wait := range []float64{0.2, 0.02, 0.005} {
		for _, preset := range kind.Presets {
			keys.Push(midi.NoteOnMsg(1, note, 100))
			loud = math.Max(loud, run(wait))
			ed.Load(made.Describe(preset))
			loud = math.Max(loud, run(wait))
			keys.Push(midi.NoteOffMsg(1, note, 0))
			run(wait)
			note = 48 + (note-47)%24
		}
	}
	if loud < 0.01 {
		t.Fatalf("it did not play: peak %.4f", loud)
	}
	run(20)
	if peak := run(1); peak > 1e-4 {
		t.Errorf("20 s after letting go it still sounds: peak %.4f", peak)
	}
}

// presets loaded from another goroutine (as the page does) while the audio runs
func TestPresetsWhilePlayingLive(t *testing.T) {
	p := psalm.New(paneltest.NewEngine(t))
	kinds := catalog.Kinds(catalog.Config{Input: midi.NewQueue()})
	pt := patcher.New(p, panel.New(p, "Test"), "Patch", kinds)
	p.Out(pt.Out())
	var kind patcher.Kind
	for _, k := range kinds {
		if k.Name == "subtractive" {
			kind = k
		}
	}
	made, _ := kind.Make(kind.Presets[0])
	if err := p.AudioStart(); err != nil {
		t.Fatal(err)
	}
	clock, _ := pt.Add("clock", "", 0, 0)
	seq, _ := pt.Add("sequence", "", 0, 0)
	notes, _ := pt.Add("notes", "", 0, 0)
	syn, _ := pt.Add("subtractive", kind.Presets[0], 0, 0)
	pt.SettingsOf(clock).Apply(map[string]float64{"division": 5})
	trig, _ := pt.Connect(patcher.Cable{From: clock, To: seq})
	_, _ = pt.Connect(patcher.Cable{From: seq, Out: 0, To: notes, In: 1})
	_, _ = pt.Connect(patcher.Cable{From: seq, Out: 1, To: notes, In: 0})
	_, _ = pt.Connect(patcher.Cable{From: notes, To: syn})
	_, _ = pt.Connect(patcher.Cable{From: syn, To: "out"})
	ed := pt.SettingsOf(syn).(*editor.Editor)
	voices := pt.Module(syn).(*poly.Poly[*subtractive.Voice])

	rng := rand.New(rand.NewPCG(1, 2))
	for range 300 {
		ed.Load(made.Describe(kind.Presets[rng.IntN(len(kind.Presets))]))
		time.Sleep(time.Duration(rng.IntN(3000)) * time.Microsecond)
	}
	pt.Disconnect(trig)
	time.Sleep(50 * time.Millisecond)
	if err := p.AudioStop(); err != nil {
		t.Fatal(err)
	}
	for range int(20 * 44100 / 256) { // 20 s after the last note
		p.ResetRun()
		p.Run()
	}
	n := 0
	voices.CallVoices(func(v *subtractive.Voice) {
		if v.IsActive() {
			n++
		}
	})
	if n > 0 {
		t.Errorf("20 s after the last note %d voices still sound", n)
	}
}

// longNotes is a sequence playing notes of 100 to 400 steps (12.5 to 50 s) into a subtractive synth
// at a preset: the trigger cable, the synth's editor, its voices
type longNotes struct {
	p      *psalm.Psalm
	pt     *patcher.Patcher
	trig   string
	ed     *editor.Editor
	voices *poly.Poly[*subtractive.Voice]
	made   *patcher.Made
}

func newLongNotes(t *testing.T, p *psalm.Psalm, preset string) *longNotes {
	t.Helper()
	kinds := catalog.Kinds(catalog.Config{Input: midi.NewQueue()})
	pt := patcher.New(p, panel.New(p, "Test"), "Patch", kinds)
	p.Out(pt.Out())
	l := &longNotes{p: p, pt: pt}
	for _, k := range kinds {
		if k.Name == "subtractive" {
			l.made, _ = k.Make(preset)
		}
	}
	must := func(id string, err error) string {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	clock := must(pt.Add("clock", "", 0, 0))
	seq := must(pt.Add("sequence", "", 0, 0))
	lengths := must(pt.Add("sequence", "", 0, 0))
	notes := must(pt.Add("notes", "", 0, 0))
	syn := must(pt.Add("subtractive", preset, 0, 0))
	if err := pt.SetText(lengths, "values", "100 200 300 400"); err != nil {
		t.Fatal(err)
	}
	l.trig = must(pt.Connect(patcher.Cable{From: clock, To: seq}))
	must(pt.Connect(patcher.Cable{From: clock, To: lengths}))
	must(pt.Connect(patcher.Cable{From: lengths, Out: 0, To: notes, In: 3}))
	must(pt.Connect(patcher.Cable{From: seq, Out: 0, To: notes, In: 1}))
	must(pt.Connect(patcher.Cable{From: seq, Out: 1, To: notes, In: 0}))
	must(pt.Connect(patcher.Cable{From: notes, To: syn}))
	must(pt.Connect(patcher.Cable{From: syn, To: "out"}))
	l.ed = pt.SettingsOf(syn).(*editor.Editor)
	l.voices = pt.Module(syn).(*poly.Poly[*subtractive.Voice])
	return l
}

// run runs it offline for some seconds: the loudest sample
func (l *longNotes) run(seconds float64) (peak float64) {
	for range int(seconds * 44100 / 256) {
		l.p.ResetRun()
		l.p.Run()
		for _, s := range l.pt.Out().Output(0).Channels[0] {
			peak = math.Max(peak, math.Abs(float64(s)))
		}
	}
	return
}

func (l *longNotes) active() (n int) {
	l.voices.CallVoices(func(v *subtractive.Voice) {
		if v.IsActive() {
			n++
		}
	})
	return
}

// notes held long under a pad, then the preset a pluck: they fall silent, as a pluck's would (the
// sustain they are held at follows the preset)
func TestLongNotesFollowThePreset(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	l := newLongNotes(t, psalm.New(nil), "pluck")
	l.ed.Load(l.made.Describe("pad"))
	if peak := l.run(3); peak < 0.05 {
		t.Fatalf("the pad plays at %.4f", peak)
	}
	l.ed.Load(l.made.Describe("pluck"))
	l.pt.Disconnect(l.trig)
	l.run(2.5) // the pad's decay to its sustain, then the pluck's sustain of 0
	if n, peak := l.active(), l.run(0.5); peak > 1e-3 {
		t.Errorf("the notes of the pad still sound as a pluck: peak %.4f (%d voices)", peak, n)
	}
}

// a restart lets go of the notes playing: they ring out their release, not their length
func TestRestartLetsGo(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	p := psalm.New(nil)
	l := newLongNotes(t, p, "lead")
	l.run(2)
	l.pt.Disconnect(l.trig)
	if l.active() == 0 {
		t.Fatal("no notes playing")
	}
	p.Restart()
	l.run(0.5) // the lead's release: 250 ms
	if n := l.active(); n > 0 {
		t.Errorf("after a restart %d notes go on", n)
	}
}

// stopping and starting again lets go of the notes playing
func TestStopLetsGo(t *testing.T) {
	p := psalm.New(paneltest.NewEngine(t))
	l := newLongNotes(t, p, "lead")
	if err := p.AudioStart(); err != nil {
		t.Fatal(err)
	}
	onAudio := func(f func() int64) int64 { // read between two buffers
		c := make(chan int64)
		p.Do(func() { c <- f() })
		return <-c
	}
	paneltest.Eventually(t, "notes playing", func() bool { return onAudio(func() int64 { return int64(l.active()) }) > 4 })
	l.pt.Disconnect(l.trig)
	if err := p.AudioStop(); err != nil {
		t.Fatal(err)
	}
	from := p.Ticks()
	if err := p.AudioStart(); err != nil {
		t.Fatal(err)
	}
	// a second of sound (the lead's release is 250 ms, its notes 12.5 s and more), the clock read
	// between two buffers
	second := int64(p.Config().PPQ) * int64(p.BPM()) / 60
	paneltest.Eventually(t, "a second of sound", func() bool { return onAudio(p.Ticks)-from >= second })
	if err := p.AudioStop(); err != nil {
		t.Fatal(err)
	}
	if n := l.active(); n > 0 {
		t.Errorf("after stopping and starting %d notes go on", n)
	}
}
