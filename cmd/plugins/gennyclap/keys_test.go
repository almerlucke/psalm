package main

import (
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/almerlucke/psalm/modules/controls/panel/patcher/script"
)

const keyScript = `function render(p) { return [{pitch: p.key, start: 0, duration: 2}, {pitch: p.key + 7, start: 2, duration: 2}] }`

// an instance with the pieces of the keys made, the key trigger on
func keyed(t *testing.T, text string, isScript bool, mode float64) *instance {
	t.Helper()
	in := newInstance()
	in.set(pKeys, mode)
	var rt *script.Runtime
	if err := in.renderKeys(&rt, text, isScript, 7); err != nil {
		t.Fatal(err)
	}
	return in
}

// the events as "on 62" / "off 62" / "end 3" with their beats
func summary(evs []goEvent) []string {
	var out []string
	for _, e := range evs {
		kind := "off"
		switch {
		case e.kill:
			out = append(out, "end")
			continue
		case e.on:
			kind = "on"
		}
		out = append(out, kind+" "+itoaF(e.key)+"@"+itoaF(int(math.Round(e.beat*100))))
	}
	return out
}

func itoaF(n int) string { return strconv.Itoa(n) }

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestKeyPlaysOnce(t *testing.T) {
	in := keyed(t, keyScript, true, 1)
	in.keyAt(62, 0.8, true, 10)
	if got, want := summary(in.collect(10, 11)), []string{"on 62@1000", "off 62@1050", "on 69@1050"}; !equal(got, want) {
		t.Fatalf("%v, want %v", got, want)
	}
	// the key is let go: a piece that plays once goes on
	in.keyAt(62, 0, false, 10.6)
	if got, want := summary(in.collect(11, 12)), []string{"off 69@1100"}; !equal(got, want) {
		t.Fatalf("%v, want %v", got, want)
	}
	// and is over: nothing more, however far the clock goes
	if got := in.collect(12, 40); len(got) != 0 {
		t.Fatalf("%v", summary(got))
	}
	if in.voices[62].on {
		t.Fatal("the voice is still there")
	}
}

func TestKeySnap(t *testing.T) {
	in := keyed(t, keyScript, true, 1)
	in.set(pSnap, 2) // a beat
	in.keyAt(60, 0.8, true, 10.3)
	if got := in.collect(10, 11); len(got) != 0 {
		t.Fatalf("started before the grid: %v", summary(got))
	}
	if got := summary(in.collect(11, 12)); len(got) != 3 || got[0] != "on 60@1100" {
		t.Fatalf("%v", got)
	}
	in.set(pSnap, 3) // a bar
	in.keyAt(61, 0.8, true, 12.5)
	got := summary(in.collect(12, 17))
	if !slices.Contains(got, "on 61@1600") || slices.Contains(got, "on 61@1250") {
		t.Fatalf("%v", got)
	}
}

func TestKeyLoopsWhileHeld(t *testing.T) {
	in := keyed(t, keyScript, true, 2)
	in.keyAt(60, 0.8, true, 10)
	var ons int
	for b := 10.0; b < 14.5; b += 0.5 { // the piece is 4 beats: the second time through starts at 14
		for _, e := range in.collect(b, b+0.5) {
			if e.on && e.key == 60 {
				ons++
			}
		}
	}
	if ons != 2 {
		t.Fatalf("%d starts of the first note in 4.5 beats, want 2 (once at 10, once at 14)", ons)
	}
	// let go in the middle of a note: the notes are let go with the key
	in.keyAt(60, 0, false, 14.2)
	got := in.collect(14.5, 15)
	if len(got) != 1 || !got[0].kill {
		t.Fatalf("%v", summary(got))
	}
	if in.voices[60].on || len(in.collect(15, 30)) != 0 {
		t.Fatal("the voice goes on after its key")
	}
}

func TestKeyAgainReplacesTheVoice(t *testing.T) {
	in := keyed(t, keyScript, true, 2)
	in.keyAt(60, 0.8, true, 10)
	first := in.voices[60].salt
	in.collect(10, 10.2)
	in.keyAt(60, 0.8, true, 10.3)
	got := in.collect(10.2, 10.4)
	if len(got) < 2 || !got[0].kill || got[0].voice != first || !got[1].on || got[1].voice == first {
		t.Fatalf("%v", summary(got))
	}
	if got[0].beat != 10.3 {
		t.Fatalf("the old voice stops at %v", got[0].beat)
	}
}

func TestKeyTriggerTurnedOff(t *testing.T) {
	in := keyed(t, keyScript, true, 2)
	in.keyAt(60, 0.8, true, 10)
	in.keyAt(64, 0.8, true, 10)
	in.collect(10, 10.1)
	in.set(pKeys, 0)
	got := in.collect(10.1, 10.2)
	if len(got) != 2 || !got[0].kill || !got[1].kill {
		t.Fatalf("%v", summary(got))
	}
	if len(in.collect(10.2, 12)) != 0 {
		t.Fatal("voices go on")
	}
}

func TestKeyVelocityAndReset(t *testing.T) {
	in := keyed(t, keyScript, true, 1)
	in.keyAt(60, 0.5, true, 10)
	if got := in.collect(10, 10.1); len(got) != 1 || math.Abs(got[0].velocity-0.5*0.8) > 1e-9 {
		t.Fatalf("%+v", got)
	}
	in.resetVoices()
	if got := in.collect(10.1, 12); len(got) != 0 {
		t.Fatalf("%v", summary(got))
	}
}

// a spec has no keys: the one piece is moved by the distance from middle C
func TestKeySpec(t *testing.T) {
	in := keyed(t, defaultSpec, false, 1)
	in.keyAt(60, 1, true, 0)
	in.keyAt(67, 1, true, 0)
	got := in.collect(0, 4)
	var low, high []int
	for _, e := range got {
		if e.on && e.voice == in.voices[60].salt {
			low = append(low, e.key)
		}
		if e.on && e.voice == in.voices[67].salt {
			high = append(high, e.key)
		}
	}
	if len(low) == 0 || len(low) != len(high) {
		t.Fatalf("%v %v", low, high)
	}
	slices.Sort(low)
	slices.Sort(high)
	for i := range low {
		if high[i] != low[i]+7 {
			t.Fatalf("%v %v", low, high)
		}
	}
}

// a script that fails for some keys leaves them silent; for all of them it is an error
func TestKeyFailures(t *testing.T) {
	picky := `function render(p) { if (p.key > 72) throw new Error("too high"); return [{pitch: p.key, start: 0, duration: 1}] }`
	in := keyed(t, picky, true, 1)
	ks := in.keys.Load()
	if ks.stores[60] == nil || ks.stores[73] != nil {
		t.Fatalf("%v %v", ks.stores[60] != nil, ks.stores[73] != nil)
	}
	in.keyAt(73, 1, true, 0)
	if got := in.collect(0, 2); len(got) != 0 {
		t.Fatalf("a key without notes plays: %v", summary(got))
	}
	in2 := newInstance()
	in2.set(pKeys, 1)
	var rt *script.Runtime
	if err := in2.renderKeys(&rt, `function render(p) { throw new Error("boom") }`, true, 7); err == nil {
		t.Fatal("no error when no key works")
	}
}

// the goroutine of the instance makes the pieces of the keys when the trigger is on, and again when the file changes
func TestRunMakesKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "genny.js")
	t.Setenv("GENNY_SPEC", path)
	if err := os.WriteFile(path, []byte(keyScript), 0o644); err != nil {
		t.Fatal(err)
	}
	in := newInstance()
	in.set(pKeys, 1)
	in.stop, in.done = make(chan struct{}), make(chan struct{})
	go in.run()
	defer func() { close(in.stop); <-in.done }()
	wait(t, "the keys", func() bool { ks := in.keys.Load(); return ks != nil && ks.stores[100] != nil })
	if st := in.store.Load(); st != nil {
		t.Fatal("the piece of the transport is made with the key trigger on")
	}
	changed := `function render(p) { return [{pitch: p.key + 12, start: 0, duration: 1}] }`
	if err := os.WriteFile(path, []byte(changed), 0o644); err != nil {
		t.Fatal(err)
	}
	future := timeNow().Add(2e9)
	_ = os.Chtimes(path, future, future)
	wait(t, "the changed script", func() bool {
		ks := in.keys.Load()
		return ks != nil && ks.stores[60] != nil && ks.stores[60].lines[0].events[0].pitch == 72
	})
}

func timeNow() time.Time { return time.Now() }

// the host shows the name of a value, and takes the name typed
func TestParamText(t *testing.T) {
	for _, c := range []struct {
		id   int
		v    float64
		want string
	}{
		{pKeys, 0, "off"}, {pKeys, 1, "once"}, {pKeys, 2, "loop while held"}, {pSnap, 2, "1 beat"}, {pLatch, 1, "on"},
		{pSeed, 7, "7"}, {pTranspose, -3, "-3"}, {pGain, 1.5, "1.50"}, {pKeys, 9, "9"},
	} {
		if got := paramText(c.id, c.v); got != c.want {
			t.Errorf("%s %v: %q, want %q", params[c.id].name, c.v, got, c.want)
		}
	}
	if v, ok := paramValue(pKeys, "Loop while held"); !ok || v != 2 {
		t.Errorf("%v %v", v, ok)
	}
	if v, ok := paramValue(pSnap, "2"); !ok || v != 2 {
		t.Errorf("%v %v", v, ok)
	}
	if _, ok := paramValue(pKeys, "sideways"); ok {
		t.Error("nonsense is a value")
	}
}

const pressScript = `function render(p) { return {length: 16, notes: [
	{pitch: p.key + p.held.length, start: 0, duration: 2, velocity: p.velocity},
	{pitch: p.key + 12, start: 8, duration: 2, velocity: p.velocity}]} }`

// ons are the note ons among the events
func ons(evs []goEvent) []goEvent {
	var out []goEvent
	for _, e := range evs {
		if e.on {
			out = append(out, e)
		}
	}
	return out
}

// an instance that makes the piece of a key when it is pressed
func pressing(t *testing.T) (*instance, **script.Runtime) {
	t.Helper()
	in := newInstance()
	in.set(pKeys, 1)
	in.set(pSnap, 2) // a beat
	in.set(pRender, 1)
	in.pressMode.Store(true)
	var rt *script.Runtime
	return in, &rt
}

// the piece is made for the press (its velocity, the keys held) while the snap waits for the grid
func TestPressMade(t *testing.T) {
	in, rt := pressing(t)
	in.input(60, 0.9, true)
	in.keyAt(60, 0.9, true, 10.3) // starts at 11
	if got := in.collect(10, 11); len(got) != 0 {
		t.Fatalf("%v", summary(got))
	}
	in.servePresses(rt, pressScript, 7, false, 0)
	got := ons(in.collect(11, 12))
	if len(got) != 1 || !got[0].on || got[0].key != 61 || math.Abs(got[0].velocity-0.9) > 1e-9 || got[0].beat != 11 {
		t.Fatalf("%+v", got)
	}
	// the velocity of the key is the script's business, the notes are not scaled by it again
	if in.voices[60].velocity != 1 {
		t.Fatalf("scaled twice: %v", in.voices[60].velocity)
	}
}

// the keys of a chord are held when the piece is made
func TestPressChord(t *testing.T) {
	in, rt := pressing(t)
	for _, k := range []int{60, 64, 67} {
		in.input(k, 0.8, true)
		in.keyAt(k, 0.8, true, 10.1)
	}
	in.servePresses(rt, pressScript, 7, false, 0)
	got := ons(in.collect(11, 12))
	var pitches []int
	for _, e := range got {
		pitches = append(pitches, e.key)
	}
	slices.Sort(pitches)
	if !slices.Equal(pitches, []int{63, 67, 70}) { // each key plus the 3 held
		t.Fatalf("%v", pitches)
	}
}

// the press waits a moment for the rest of a chord
func TestPressSettles(t *testing.T) {
	in, rt := pressing(t)
	in.input(60, 0.8, true)
	in.keyAt(60, 0.8, true, 10.1)
	in.servePresses(rt, pressScript, 7, false, time.Hour)
	if in.presses[60].Load() != nil {
		t.Fatal("made before it settled")
	}
	in.servePresses(rt, pressScript, 7, false, 0)
	if in.presses[60].Load() == nil {
		t.Fatal("not made")
	}
}

// a piece that comes late plays what is still to come
func TestPressLate(t *testing.T) {
	in, rt := pressing(t)
	in.input(60, 0.8, true)
	in.keyAt(60, 0.8, true, 10.1) // starts at 11
	if got := in.collect(11, 12.5); len(got) != 0 {
		t.Fatalf("%v", summary(got))
	}
	in.servePresses(rt, pressScript, 7, false, 0)
	got := ons(in.collect(12.5, 14)) // the first note was due at 11, the second at 11 + 2 beats
	if len(got) != 1 || got[0].key != 72 || got[0].beat != 13 {
		t.Fatalf("%+v", got)
	}
}

// a key let go before its piece is made, and a script that fails, leave nothing playing
func TestPressFailsAndEnds(t *testing.T) {
	in, rt := pressing(t)
	in.set(pKeys, 2) // loop
	in.keyAt(60, 0.8, true, 10.1)
	in.keyAt(60, 0, false, 11.5) // let go after it started, before it is made
	if got := in.collect(11, 12); len(got) != 0 {
		t.Fatalf("%v", summary(got))
	}
	in.servePresses(rt, pressScript, 7, false, 0)
	if got := in.collect(12, 20); len(got) != 0 || in.voices[60].on {
		t.Fatalf("a voice of a key that was let go plays: %v", summary(got))
	}
	in.keyAt(62, 0.8, true, 20.1)
	in.servePresses(rt, `function render(p) { throw new Error("boom") }`, 7, false, 0)
	if got := in.collect(21, 24); len(got) != 0 || in.voices[62].on {
		t.Fatalf("%v", summary(got))
	}
	if r := in.presses[62].Load(); r == nil || r.err == nil {
		t.Fatal("the failure is not kept")
	}
}

// with new each pass every press has its own seed; the pieces of two presses differ
func TestPressSeeds(t *testing.T) {
	in, rt := pressing(t)
	random := `function render(p) { return [{pitch: 40 + Math.floor(genny.sequence([0]).take(1)[0] + p.seed % 50), start: 0, duration: 1}] }`
	in.keyAt(60, 1, true, 10.1)
	in.keyAt(61, 1, true, 10.1)
	in.servePresses(rt, random, 7, true, 0)
	a, b := in.presses[60].Load(), in.presses[61].Load()
	if a == nil || b == nil || a.err != nil || b.err != nil {
		t.Fatalf("%+v %+v", a, b)
	}
	if a.st.lines[0].events[0].pitch == b.st.lines[0].events[0].pitch {
		t.Fatal("the presses have the same seed")
	}
	in.keyAt(62, 1, true, 10.1)
	in.keyAt(63, 1, true, 10.1)
	in.servePresses(rt, random, 7, false, 0)
	if in.presses[62].Load().st.lines[0].events[0].pitch != in.presses[63].Load().st.lines[0].events[0].pitch {
		t.Fatal("the presses have another seed without new each pass")
	}
}

// the goroutine of the instance decides: on press needs a snap (and a script); without them the pieces are made beforehand
func TestRunPressMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "genny.js")
	t.Setenv("GENNY_SPEC", path)
	if err := os.WriteFile(path, []byte(pressScript), 0o644); err != nil {
		t.Fatal(err)
	}
	in := newInstance()
	in.set(pKeys, 1)
	in.set(pRender, 1) // no snap yet
	in.stop, in.done = make(chan struct{}), make(chan struct{})
	go in.run()
	defer func() { close(in.stop); <-in.done }()
	wait(t, "the keys made beforehand", func() bool { ks := in.keys.Load(); return ks != nil && ks.stores[60] != nil })
	if in.pressMode.Load() {
		t.Fatal("on press without a snap")
	}
	in.set(pSnap, 2)
	wait(t, "on press", func() bool { return in.pressMode.Load() })
	in.input(60, 0.8, true)
	in.keyAt(60, 0.8, true, 10.2)
	wait(t, "the piece of the press", func() bool { r := in.presses[60].Load(); return r != nil && r.st != nil })
	in.set(pSnap, 0)
	wait(t, "ahead again", func() bool { return !in.pressMode.Load() })
}
