package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/almerlucke/psalm/modules/controls/panel/patcher/script"
)

// playFor makes an instance that plays the default spec for a seed, without the goroutine that renders
func playFor(t *testing.T, seed uint64) (*instance, *lineData) {
	t.Helper()
	l, err := renderSpec(defaultSpec, seed)
	if err != nil {
		t.Fatal(err)
	}
	in := newInstance()
	in.store.Store(&store{lines: map[int64]*lineData{0: l}, length: l.length, overrun: l.overrun})
	return in, l
}

// every note that sounds is let go by the event with its id, later, and the events come in order of time
func TestCollectNotesAndTheirEnds(t *testing.T) {
	in, l := playFor(t, 7)
	on := map[int]goEvent{}
	var prev float64
	for beat := 0.0; beat < 2*l.length; beat += 0.05 {
		events := slices.Clone(in.collect(beat, beat+0.05))
		for _, e := range events {
			if e.beat < prev {
				t.Fatalf("events out of order: %v after %v", e.beat, prev)
			}
			prev = e.beat
			if e.on {
				on[e.id] = e
			} else if s, ok := on[e.id]; !ok || e.beat < s.beat {
				t.Fatalf("the end of note %d is without its start (or before it)", e.id)
			} else {
				delete(on, e.id)
			}
		}
	}
	if len(on) > 10 {
		t.Fatalf("%d notes never ended", len(on))
	}
}

func TestTransposeAndGain(t *testing.T) {
	in, l := playFor(t, 7)
	plain := slices.Clone(in.collect(0, l.length))
	in.set(pTranspose, 12)
	in.set(pGain, 0.5)
	moved := slices.Clone(in.collect(0, l.length))
	if len(plain) == 0 || len(plain) != len(moved) {
		t.Fatalf("%d events, then %d", len(plain), len(moved))
	}
	for i := range plain {
		if moved[i].key != plain[i].key+12 && moved[i].key != 127 {
			t.Fatalf("event %d: key %d, was %d", i, moved[i].key, plain[i].key)
		}
		if plain[i].on && moved[i].velocity > plain[i].velocity*0.5+1e-9 && moved[i].velocity > 0.01 {
			t.Fatalf("event %d: velocity %v, was %v", i, moved[i].velocity, plain[i].velocity)
		}
	}
}

// the same seed gives the same notes, another seed other ones
func TestSeed(t *testing.T) {
	a, la := playFor(t, 7)
	b, _ := playFor(t, 7)
	c, _ := playFor(t, 8)
	ea, eb, ec := slices.Clone(a.collect(0, la.length)), slices.Clone(b.collect(0, la.length)), slices.Clone(c.collect(0, la.length))
	if !slices.Equal(ea, eb) {
		t.Fatal("the same seed gave other notes")
	}
	if slices.Equal(ea, ec) {
		t.Fatal("another seed gave the same notes")
	}
}

// a pitch between two keys is the nearest key and how far from it (a tuning of the note), unless microtones are off
func TestMicrotones(t *testing.T) {
	l := newLine([]ev{{start: 0, end: 1, pitch: 60.4, vel: 0.8}, {start: 1, end: 2, pitch: 60.6, vel: 0.8}, {start: 2, end: 3, pitch: 62, vel: 0.8}}, 4, 0.25)
	in := newInstance()
	in.store.Store(&store{lines: map[int64]*lineData{0: l}, length: l.length})

	var ons []goEvent
	for _, e := range in.collect(0, 4) {
		if e.on {
			ons = append(ons, e)
		}
	}
	if len(ons) != 3 {
		t.Fatalf("%d note ons", len(ons))
	}
	for i, want := range []struct {
		key  int
		tune float64
	}{{60, 0.4}, {61, -0.4}, {62, 0}} {
		if ons[i].key != want.key || math.Abs(ons[i].tune-want.tune) > 1e-9 {
			t.Errorf("note %d: key %d, tune %v, want %d, %v", i, ons[i].key, ons[i].tune, want.key, want.tune)
		}
	}

	in.set(pMicro, 0)
	for _, e := range in.collect(0, 4) {
		if e.tune != 0 {
			t.Fatalf("microtones off, but a note is tuned by %v", e.tune)
		}
	}

	// the end of a note is the key it started on, whatever the tuning
	in.set(pMicro, 1)
	keys := map[int]int{}
	for _, e := range in.collect(0, 4) {
		if e.on {
			keys[e.id] = e.key
		} else if keys[e.id] != e.key {
			t.Fatalf("note %d ends on key %d, started on %d", e.id, e.key, keys[e.id])
		}
	}
}

// a script with another tuning reaches the plugin with its fractional pitches
func TestScriptTuning(t *testing.T) {
	rt := script.NewRuntime()
	piece, err := rt.Render(`function render() { const t = genny.tuning({edo: 31, root: 60}); return [0, 1, 2, 3].map(d => ({pitch: t.pitch(d), start: d, duration: 1})) }`, 1)
	if err != nil {
		t.Fatal(err)
	}
	l := buildPiece(piece)
	in := newInstance()
	in.store.Store(&store{lines: map[int64]*lineData{0: l}, length: l.length})
	var tunes []float64
	for _, e := range in.collect(0, l.length) {
		if e.on {
			tunes = append(tunes, e.tune)
		}
	}
	// 31 steps of 38.7 cents: 0, 0.387, 0.774 (the key above, -0.226), 1.161 (the key above, +0.161)
	want := []float64{0, 0.387, -0.226, 0.161}
	if len(tunes) != 4 {
		t.Fatalf("%d notes", len(tunes))
	}
	for i := range want {
		if math.Abs(tunes[i]-want[i]) > 0.001 {
			t.Errorf("note %d: tune %.3f, want %.3f", i, tunes[i], want[i])
		}
	}
}

func TestSwingDelaysOddSteps(t *testing.T) {
	s, e := swungTimes(0.25, 0.5, 0.4, 0.25)
	if s <= 0.25 || e <= 0.5 {
		t.Fatalf("the note on step 1 was not delayed: %v %v", s, e)
	}
	s, e = swungTimes(0.5, 0.75, 0.4, 0.25)
	if s != 0.5 || e != 0.75 {
		t.Fatalf("the note on step 2 moved: %v %v", s, e)
	}
	// another step: triplets of 1/3 beat
	if s, _ := swungTimes(1.0/3, 2.0/3, 0.5, 1.0/3); s <= 1.0/3 {
		t.Fatalf("the swing does not follow the step of the piece: %v", s)
	}
}

func TestStateRoundTrip(t *testing.T) {
	a := newInstance()
	a.set(pSeed, 42)
	a.set(pTranspose, -5)
	a.set(pSwing, 0.25)
	a.text = defaultSpec
	b := newInstance()
	b.loadState(a.saveState())
	for id := range numParams {
		if a.get(id) != b.get(id) {
			t.Fatalf("parameter %q: %v, was %v", params[id].name, b.get(id), a.get(id))
		}
	}
	if b.text != defaultSpec || !b.fromSave {
		t.Fatal("the spec of the project was not kept")
	}
}

// wait polls until cond holds
func wait(t *testing.T, what string, cond func() bool) {
	t.Helper()
	for range 300 {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timeout: %s", what)
}

// the goroutine reads the spec file, reads it again when it is edited, and with "new each pass" renders each cycle
// with its own seed
func TestRunReloadsAndRenews(t *testing.T) {
	path := filepath.Join(t.TempDir(), "genny.json")
	t.Setenv("GENNY_SPEC", path)
	in := newInstance()
	in.stop = make(chan struct{})
	go in.run()
	defer close(in.stop)

	wait(t, "the first render", func() bool { return in.store.Load() != nil })
	if _, err := os.Stat(path); err != nil {
		t.Fatal("the default spec was not written:", err)
	}
	first := in.store.Load()

	// an edited file is read again
	edited := `{"steps_per_beat": 4, "tracks": [{"name": "t", "source": {"kind": "notes", "params": {"notes": [{"pitch": 60, "start": 0, "duration": 1, "velocity": 0.5}]}}}]}`
	time.Sleep(20 * time.Millisecond)
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(2 * time.Second)
	_ = os.Chtimes(path, future, future)
	wait(t, "the edited spec", func() bool {
		st := in.store.Load()
		return st != first && len(st.lines[0].events) == 1
	})

	// a spec with an error keeps what played
	good := in.store.Load()
	if err := os.WriteFile(path, []byte(`{"tracks": [{"source": {"kind": "nope"}}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	future = future.Add(2 * time.Second)
	_ = os.Chtimes(path, future, future)
	time.Sleep(600 * time.Millisecond)
	if in.store.Load() != good {
		t.Fatal("a spec with an error replaced the one that played")
	}
}

// with "new each pass" the cycles ahead of the transport are rendered, each with its own seed
func TestRenewRendersTheCyclesAhead(t *testing.T) {
	t.Setenv("GENNY_SPEC", filepath.Join(t.TempDir(), "genny.json"))
	in := newInstance()
	in.set(pRegen, 1)
	in.stop = make(chan struct{})
	go in.run()
	defer close(in.stop)

	in.want.Store(3)
	wait(t, "the cycles 3 to 5", func() bool {
		st := in.store.Load()
		return st != nil && st.renew && st.lines[3] != nil && st.lines[4] != nil && st.lines[5] != nil
	})
	st := in.store.Load()
	if slices.Equal(st.lines[3].events, st.lines[4].events) {
		t.Fatal("two cycles have the same notes")
	}
	in.want.Store(9)
	wait(t, "the cycle 9", func() bool {
		st := in.store.Load()
		return st.lines[9] != nil && st.lines[3] == nil
	})
}

// a file that ends in .js is a script: it is written when missing, rendered by the script runtime, and read again
// when it is edited
func TestScriptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "genny.js")
	t.Setenv("GENNY_SPEC", path)
	in := newInstance()
	in.stop = make(chan struct{})
	go in.run()
	defer close(in.stop)

	wait(t, "the default script", func() bool { return in.store.Load() != nil })
	st := in.store.Load()
	if data, err := os.ReadFile(path); err != nil || string(data) != defaultScript {
		t.Fatal("the default script was not written")
	}
	l := st.lines[0]
	channels := map[int]bool{}
	for _, e := range l.events {
		channels[e.channel] = true
	}
	if len(l.events) < 10 || !channels[0] || !channels[1] || l.length != 16 {
		t.Fatalf("%d events, channels %v, %v beats", len(l.events), channels, l.length)
	}

	edited := `function render(p) { return {stepsPerBeat: 3, length: 6, notes: [{pitch: 60, start: 1, duration: 1}]} }`
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(2 * time.Second)
	_ = os.Chtimes(path, future, future)
	wait(t, "the edited script", func() bool {
		s := in.store.Load()
		return s != st && len(s.lines[0].events) == 1
	})
	got := in.store.Load().lines[0]
	if math.Abs(got.length-2) > 1e-9 || math.Abs(got.step-1.0/3) > 1e-9 {
		t.Fatalf("length %v beats, step %v", got.length, got.step)
	}
}

func TestStateKeepsScript(t *testing.T) {
	a := newInstance()
	a.text, a.isScript = defaultScript, true
	b := newInstance()
	b.loadState(a.saveState())
	if !b.isScript || b.text != defaultScript {
		t.Fatal("the script of the project was not kept")
	}
}

// errors, what a script logs and what a render made go to the host and to the file next to the spec
func TestLogsErrorsAndNews(t *testing.T) {
	path := filepath.Join(t.TempDir(), "genny.js")
	t.Setenv("GENNY_SPEC", path)
	var mu sync.Mutex
	var said []string
	logHook = func(handle, severity int, text string) {
		mu.Lock()
		said = append(said, fmt.Sprintf("%d %s", severity, text))
		mu.Unlock()
	}
	defer func() { logHook = nil }()

	if err := os.WriteFile(path, []byte(`function render() { log("hello", 1); throw new Error("boom") }`), 0o644); err != nil {
		t.Fatal(err)
	}
	in := newInstance()
	in.stop, in.done = make(chan struct{}), make(chan struct{})
	go in.run()
	defer func() { close(in.stop); <-in.done }()

	readLog := func() string { data, _ := os.ReadFile(path + ".log"); return string(data) }
	wait(t, "the error in the log file", func() bool { return strings.Contains(readLog(), "boom") })
	// a script that fails is not tried again and again: it said what it had to say once
	time.Sleep(300 * time.Millisecond)
	got := readLog()
	if n := strings.Count(got, "boom"); n != 1 {
		t.Fatalf("the error was reported %d times:\n%s", n, got)
	}
	if !strings.Contains(got, "info: script: hello 1") || !strings.Contains(got, "error: ") || !strings.Contains(got, "nothing plays") {
		t.Fatalf("log file:\n%s", got)
	}
	mu.Lock()
	if len(said) < 2 || !strings.HasPrefix(said[0], "1 genny: script: hello 1") || !strings.HasPrefix(said[1], "3 genny: ") {
		t.Fatalf("host log: %q", said)
	}
	mu.Unlock()

	// a repaired script says what it made
	if err := os.WriteFile(path, []byte(`function render() { return [{pitch: 60, start: 0}, {pitch: 62, start: 1}] }`), 0o644); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(2 * time.Second)
	_ = os.Chtimes(path, future, future)
	wait(t, "the news in the log file", func() bool { return strings.Contains(readLog(), "2 notes in 4 beats") })
}

// a genny.js that appears next to genny.json is played from then on (the folder of the config can be a sandbox's)
func TestScriptAppearsNextToSpec(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("GENNY_SPEC", "")
	in := newInstance()
	in.stop, in.done = make(chan struct{}), make(chan struct{})
	go in.run()
	defer func() { close(in.stop); <-in.done }()

	spec := filepath.Join(dir, "psalm", "genny.json")
	wait(t, "the default spec", func() bool { return in.store.Load() != nil && fileExists(spec) })
	first := in.store.Load()
	if len(first.lines[0].events) < 100 {
		t.Fatalf("%d notes of the default spec", len(first.lines[0].events))
	}
	js := `function render() { return [{pitch: 61.5, start: 0, duration: 2}] }`
	if err := os.WriteFile(filepath.Join(dir, "psalm", "genny.js"), []byte(js), 0o644); err != nil {
		t.Fatal(err)
	}
	wait(t, "the script", func() bool {
		st := in.store.Load()
		return st != first && len(st.lines[0].events) == 1 && st.lines[0].events[0].pitch == 61.5
	})
	if data, _ := os.ReadFile(filepath.Join(dir, "psalm", "genny.js.log")); !strings.Contains(string(data), "1 notes in 4 beats") {
		t.Fatalf("log of the script:\n%s", data)
	}
}

// a project keeps its copy of the file it was saved with, but plays another file when that one is there
func TestProjectPlaysAnotherFile(t *testing.T) {
	dir := t.TempDir()
	old, other := filepath.Join(dir, "genny.json"), filepath.Join(dir, "genny.js")
	t.Setenv("GENNY_SPEC", old)
	if err := os.WriteFile(old, []byte(defaultSpec), 0o644); err != nil {
		t.Fatal(err)
	}
	a := newInstance()
	a.loadFile()
	saved := a.saveState()

	// the same file: the copy of the project is kept
	b := newInstance()
	b.loadState(saved)
	b.text = "kept"
	b.loadFile()
	if b.text != "kept" {
		t.Fatalf("the copy of the project was replaced: %q", b.text[:min(len(b.text), 20)])
	}

	// another file: it is played
	t.Setenv("GENNY_SPEC", other)
	if err := os.WriteFile(other, []byte("function render() { return [] }"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := newInstance()
	c.loadState(saved)
	c.loadFile()
	if !c.isScript || !strings.HasPrefix(c.text, "function render") {
		t.Fatalf("the other file was not played: %q", c.text[:min(len(c.text), 20)])
	}
}

func TestBadSpecIsAnError(t *testing.T) {
	if _, err := renderSpec(`{"tracks": [{"source": {"kind": "nope"}}]}`, 1); err == nil {
		t.Fatal("a spec with an unknown kind rendered")
	}
}
