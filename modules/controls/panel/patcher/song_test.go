package patcher_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/script"
)

func TestParseSections(t *testing.T) {
	secs, err := patcher.ParseSections("Intro 4, Verse 8\nMiddle eight 2; Outro 4")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, s := range secs {
		got = append(got, s.Name+" "+string(rune('0'+s.Bars)))
	}
	if strings.Join(got, ",") != "Intro 4,Verse 8,Middle eight 2,Outro 4" {
		t.Errorf("sections %v", got)
	}
	for _, bad := range []string{"", "Verse", "Verse x", "A 2, A 4", "B 0"} {
		if _, err := patcher.ParseSections(bad); err == nil {
			t.Errorf("%q read", bad)
		}
	}
}

// heard keeps what reaches its inputs: the sample of a trigger, a value
type heard struct {
	*psalm.Module
	at     map[int][]int64
	values []float64
}

func (h *heard) Receive(v any, i int) {
	switch x := v.(type) {
	case patcher.Event:
		h.at[i] = append(h.at[i], x.Tick.SampleTimestamp)
	case float64:
		h.values = append(h.values, x)
	}
}

// songRig is a patch at 120 bpm (a bar of 4 beats is 2 s: 88200 samples) with a song, a synth that
// records and a module that hears triggers and values
type songRig struct {
	t      *testing.T
	p      *psalm.Psalm
	pt     *patcher.Patcher
	rec    *played
	hear   *heard
	song   string
	stepAt int64
}

func newSongRig(t *testing.T, sections string) *songRig {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	t.Cleanup(func() { psalm.PopConfig() })
	r := &songRig{t: t, p: psalm.New(nil)}
	r.rec = &played{Module: psalm.NewControl()}
	r.rec.SetSelf(r.rec)
	r.hear = &heard{Module: psalm.NewControl(), at: map[int][]int64{}}
	r.hear.SetSelf(r.hear)
	kinds := append(patcher.Generators(), script.Kind(script.NewRuntime()),
		patcher.Kind{Name: "rec", Ins: []patcher.Port{{Name: "notes", Type: patcher.Notes}},
			Make: func(string) (*patcher.Made, error) { return &patcher.Made{Module: r.rec}, nil }},
		patcher.Kind{Name: "hear", Ins: []patcher.Port{{Name: "bar", Type: patcher.Trigger}, {Name: "section", Type: patcher.Trigger, Index: 1}, {Name: "number", Type: patcher.Value, Index: 2}},
			Make: func(string) (*patcher.Made, error) { return &patcher.Made{Module: r.hear}, nil }})
	r.pt = patcher.New(r.p, panel.New(r.p, "Test"), "Patch", kinds)
	r.p.Out(r.pt.Out())
	r.song = r.add("song")
	if err := r.pt.SetText(r.song, "sections", sections); err != nil {
		t.Fatal(err)
	}
	if e := r.pt.TextError(r.song, "sections"); e != "" {
		t.Fatal(e)
	}
	return r
}

func (r *songRig) add(kind string) string {
	r.t.Helper()
	id, err := r.pt.Add(kind, "", 0, 0)
	if err != nil {
		r.t.Fatal(err)
	}
	return id
}

func (r *songRig) connect(c patcher.Cable) {
	r.t.Helper()
	if _, err := r.pt.Connect(c); err != nil {
		r.t.Fatal(err)
	}
}

func (r *songRig) run(seconds float64) {
	for range int(seconds * 44100 / 256) {
		r.p.ResetRun()
		r.p.Run()
	}
}

// bars of 88200 samples, from the first
func bars(at []int64) []int64 {
	out := make([]int64, len(at))
	for i, a := range at {
		out[i] = (a + 44100) / 88200
	}
	return out
}

// the song: a trigger every bar and at every section, the section's number; it loops
func TestSongModule(t *testing.T) {
	r := newSongRig(t, "A 1, B 2")
	hear := r.add("hear")
	for i := range 3 {
		r.connect(patcher.Cable{From: r.song, Out: i, To: hear, In: i})
	}
	r.run(7.9) // 4 bars: A, B, B, A
	if got := bars(r.hear.at[0]); !slices.Equal(got, []int64{0, 1, 2, 3}) {
		t.Errorf("bar triggers at bars %v", got)
	}
	if got := bars(r.hear.at[1]); !slices.Equal(got, []int64{0, 1, 3}) {
		t.Errorf("section triggers at bars %v, want 0 (A), 1 (B), 3 (A again)", got)
	}
	if !slices.Equal(r.hear.values, []float64{0, 1, 0}) {
		t.Errorf("section numbers %v", r.hear.values)
	}
	if r.hear.at[0][0] > 92 { // the first control tick (a tick: 91.875 samples)
		t.Errorf("the first bar at sample %d, want the first tick", r.hear.at[0][0])
	}

	// a restart: the song from its start
	r.run(1) // into B
	r.hear.at = map[int][]int64{}
	r.p.Restart()
	r.run(0.1)
	if len(r.hear.at[1]) != 1 || len(r.hear.values) == 0 || r.hear.values[len(r.hear.values)-1] != 0 {
		t.Errorf("after a restart: section triggers %v, numbers %v: want A again", r.hear.at[1], r.hear.values)
	}
}

// a sequence and a clock per section: values and a pattern for each, from the section's start
func TestSongSections(t *testing.T) {
	r := newSongRig(t, "A 1, B 1")
	clock, seq, notes, syn := r.add("clock"), r.add("sequence"), r.add("notes"), r.add("rec")
	r.pt.SettingsOf(clock).Apply(map[string]float64{"division": 0}) // quarters: 4 a bar
	if err := r.pt.SetText(clock, "pattern", "A: x . x x\nB: x"); err != nil || r.pt.TextError(clock, "pattern") != "" {
		t.Fatal(err, r.pt.TextError(clock, "pattern"))
	}
	if err := r.pt.SetText(seq, "values", "A: 60 61\nB: 70 71 72"); err != nil || r.pt.TextError(seq, "values") != "" {
		t.Fatal(err, r.pt.TextError(seq, "values"))
	}
	r.connect(patcher.Cable{From: clock, To: seq})
	r.connect(patcher.Cable{From: seq, Out: 0, To: notes, In: 1})
	r.connect(patcher.Cable{From: seq, Out: 1, To: notes})
	r.connect(patcher.Cable{From: notes, To: syn})
	r.run(5.9) // A, B, A
	want := []float64{60, 61, 60, 70, 71, 72, 70, 60, 61, 60}
	if !slices.Equal(r.rec.pitches, want) {
		t.Errorf("pitches %v, want %v (A: x . x x on 60 61, B: x on 70 71 72, each from its start)", r.rec.pitches, want)
	}
	if e := r.pt.TextError(seq, "values"); e != "" {
		t.Error(e)
	}
	_ = r.pt.SetText(seq, "values", "A: 60 H9")
	if e := r.pt.TextError(seq, "values"); !strings.HasPrefix(e, "A:") {
		t.Errorf("an error in a section's values: %q, want it to name the section", e)
	}
}

// a script follows the song: song.section in its trigger, section() when one starts
func TestSongScript(t *testing.T) {
	r := newSongRig(t, "Verse 1, Chorus 1")
	s, syn := r.add("script"), r.add("rec")
	if err := r.pt.SetText(s, "code", `
let started = []
function section(name, index) { started.push(name + index); log(started.join(" ")) }
function trigger(ev) { note(song.section === "Chorus" ? 72 : 60 + song.bar, 0.8, 1) }`); err != nil || r.pt.TextError(s, "code") != "" {
		t.Fatal(err, r.pt.TextError(s, "code"))
	}
	r.connect(patcher.Cable{From: r.song, To: s}) // a trigger every bar
	r.connect(patcher.Cable{From: s, To: syn, Type: patcher.Notes})
	r.run(5.9) // Verse, Chorus, Verse
	if !slices.Equal(r.rec.pitches, []float64{60, 72, 60}) {
		t.Errorf("pitches %v, want 60 72 60", r.rec.pitches)
	}
	lines := strings.Split(r.pt.Journal().Text(), "\n")
	if st := lines[len(lines)-1]; !strings.HasSuffix(st, "  Verse0 Chorus1 Verse0") {
		t.Errorf("sections called: %q", st)
	}
}

// the line for the top of the page: the time in the song and the length of a pass
func TestSongClock(t *testing.T) {
	r := newSongRig(t, "A 1, B 2")
	if got := r.pt.SongClock(); got != "" {
		t.Errorf("before it runs: %q", got)
	}
	r.run(3.5) // 3.5 s of the 6 s: the first bar of B
	if got, want := r.pt.SongClock(), "0:03 / 0:06"; got != want {
		t.Errorf("clock %q, want %q", got, want)
	}
}
