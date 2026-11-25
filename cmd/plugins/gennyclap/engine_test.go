package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
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

func TestSwingDelaysOddSteps(t *testing.T) {
	s, e := swungTimes(0.25, 0.5, 0.4)
	if s <= 0.25 || e <= 0.5 {
		t.Fatalf("the note on step 1 was not delayed: %v %v", s, e)
	}
	s, e = swungTimes(0.5, 0.75, 0.4)
	if s != 0.5 || e != 0.75 {
		t.Fatalf("the note on step 2 moved: %v %v", s, e)
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

func TestBadSpecIsAnError(t *testing.T) {
	if _, err := renderSpec(`{"tracks": [{"source": {"kind": "nope"}}]}`, 1); err == nil {
		t.Fatal("a spec with an unknown kind rendered")
	}
}
