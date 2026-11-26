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
