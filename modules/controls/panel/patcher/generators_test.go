package patcher_test

import (
	"math"
	"slices"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
)

// played keeps the notes a synth gets: their pitch and the sample they start on
type played struct {
	*psalm.Module
	pitches []float64
	at      []int64
	seconds []float64 // how long, a note of a duration
}

func (r *played) ReceiveMessage(m psalm.Message) {
	if m["command"] != "trigger" {
		return
	}
	tick := m.MustTickVal("tick")
	r.pitches = append(r.pitches, m.MustMsg("message").MustFltVal("pitch"))
	r.at = append(r.at, tick.SampleTimestamp)
	d, _ := m.FltVal("duration")
	r.seconds = append(r.seconds, d)
}

func (r *played) Receive(v any, _ int) {
	if m, ok := psalm.MsgVal(v); ok {
		r.ReceiveMessage(m)
	}
}

// a clock of sixteenths at 120 bpm through a sequence, a quantizer and notes into a synth that
// records: the notes in order, quantized, on their steps
func TestGenerators(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	p := psalm.New(nil)
	rec := &played{Module: psalm.NewControl()}
	rec.SetSelf(rec)
	kinds := append(patcher.Generators(), patcher.Kind{Name: "rec", Ins: []patcher.Port{{Name: "notes", Type: patcher.Notes}},
		Make: func(string) (*patcher.Made, error) { return &patcher.Made{Module: rec}, nil }})
	pt := patcher.New(p, panel.New(p, "Test"), "Patch", kinds)
	must := func(id string, err error) string {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	clock := must(pt.Add("clock", "", 0, 0))
	seq := must(pt.Add("sequence", "", 0, 0))
	q := must(pt.Add("quantize", "", 0, 0))
	note := must(pt.Add("notes", "", 0, 0))
	syn := must(pt.Add("rec", "", 0, 0))
	if err := pt.SetText(seq, "values", "C3 E3 G3 B3"); err != nil { // E and B leave C minor: Eb, Bb
		t.Fatal(err)
	}
	must(pt.Connect(patcher.Cable{From: clock, To: seq}))               // trigger
	must(pt.Connect(patcher.Cable{From: seq, Out: 0, To: q}))           // value
	must(pt.Connect(patcher.Cable{From: q, To: note, In: 1}))           // pitch
	must(pt.Connect(patcher.Cable{From: seq, Out: 1, To: note, In: 0})) // the trigger after the value
	must(pt.Connect(patcher.Cable{From: note, To: syn}))                // notes

	for range int(2 * 44100 / 256) { // two seconds: 16 sixteenths
		p.ResetRun()
		p.Run()
	}
	if len(rec.pitches) < 15 || len(rec.pitches) > 17 {
		t.Fatalf("%d notes in two seconds of sixteenths", len(rec.pitches))
	}
	if want := []float64{48, 51, 55, 58, 48, 51}; !slices.Equal(rec.pitches[:6], want) {
		t.Errorf("pitches %v, want %v (quantized to C minor)", rec.pitches[:6], want)
	}
	for i := 1; i < len(rec.at); i++ {
		if d := rec.at[i] - rec.at[i-1]; math.Abs(float64(d)-5512.5) > 1 {
			t.Errorf("notes %d and %d are %d samples apart, a sixteenth is 5512.5", i-1, i, d)
			break
		}
	}

	// a pattern with rests, a chord
	rec.pitches, rec.at = nil, nil
	_ = pt.SetText(clock, "pattern", "x . x x")
	_ = pt.SetText(seq, "values", "[C3 Eb3 G3] Bb3")
	for range int(1 * 44100 / 256) { // eight sixteenths: six hits, every other one a chord
		p.ResetRun()
		p.Run()
	}
	hits := map[int64]int{}
	for _, at := range rec.at {
		hits[at]++
	}
	if len(hits) < 5 || len(hits) > 7 {
		t.Errorf("%d hits in eight steps of x . x x", len(hits))
	}
	chords := 0
	for _, n := range hits {
		if n == 3 {
			chords++
		}
	}
	if chords < 2 {
		t.Errorf("no chords played: %v", rec.pitches)
	}
	if err := pt.SetText(clock, "pattern", "x ? x"); err != nil {
		t.Fatal(err)
	}
	if e := pt.TextError(clock, "pattern"); e == "" {
		t.Errorf("a bad pattern isn't reported")
	}
}

// the texts of the modules are kept with the patch
func TestGeneratorsKept(t *testing.T) {
	p := psalm.New(nil)
	dir := t.TempDir()
	pt := patcher.New(p, panel.New(p, "Test"), "Patch", patcher.Generators(), patcher.Keep(dir))
	clock, _ := pt.Add("clock", "", 0, 0)
	seq, _ := pt.Add("sequence", "", 0, 0)
	_ = pt.SetText(clock, "pattern", "x . . x")
	_ = pt.SetText(seq, "values", "60 [62 65] 67")
	if err := pt.Save("gen"); err != nil {
		t.Fatal(err)
	}
	pt.Clear()
	if err := pt.Load("gen"); err != nil {
		t.Fatal(err)
	}
	texts := map[string]string{}
	for _, n := range pt.Nodes() {
		for _, tx := range pt.Texts(n.ID) {
			texts[n.Kind+"/"+tx.Key] = tx.Value
		}
	}
	if texts["clock/pattern"] != "x . . x" || texts["sequence/values"] != "60 [62 65] 67" {
		t.Errorf("texts loaded %v", texts)
	}
}

// a clock's division changed while it plays: its triggers go on at once, at the new division
func TestClockDivisionChanged(t *testing.T) {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	p := psalm.New(nil)
	var at []int64
	hear := &heard{Module: psalm.NewControl(), at: map[int][]int64{}}
	hear.SetSelf(hear)
	kinds := append(patcher.Generators(), patcher.Kind{Name: "hear", Ins: []patcher.Port{{Name: "trigger", Type: patcher.Trigger}},
		Make: func(string) (*patcher.Made, error) { return &patcher.Made{Module: hear}, nil }})
	pt := patcher.New(p, panel.New(p, "Test"), "Patch", kinds)
	clock, _ := pt.Add("clock", "", 0, 0)
	h, _ := pt.Add("hear", "", 0, 0)
	if _, err := pt.Connect(patcher.Cable{From: clock, To: h}); err != nil {
		t.Fatal(err)
	}
	run := func(seconds float64) {
		for range int(seconds * 44100 / 256) {
			p.ResetRun()
			p.Run()
		}
	}
	run(4)                                                        // sixteenths
	pt.SettingsOf(clock).Apply(map[string]float64{"division": 1}) // eighths
	run(0.1)
	from := len(hear.at[0])
	run(1)
	at = hear.at[0][from:]
	if len(at) < 3 {
		t.Fatalf("%d triggers in a second of eighths after the division changed (at 120 bpm: 4)", len(at))
	}
	for i := 1; i < len(at); i++ {
		if d := at[i] - at[i-1]; d < 11024 || d > 11026 {
			t.Fatalf("triggers %d samples apart, an eighth is 11025", d)
		}
	}
}

func TestParsePattern(t *testing.T) {
	for _, c := range []struct {
		text  string
		steps float64
		hits  []patcher.Hit
	}{
		{"x . x x", 4, []patcher.Hit{{At: 0, Len: 1, Chance: 1}, {At: 2, Len: 1, Chance: 1}, {At: 3, Len: 1, Chance: 1}}},
		{"x3 .", 2, []patcher.Hit{{At: 0, Len: 1.0 / 3, Chance: 1}, {At: 1.0 / 3, Len: 1.0 / 3, Chance: 1}, {At: 2.0 / 3, Len: 1.0 / 3, Chance: 1}}},
		{"[x x x]", 1, []patcher.Hit{{At: 0, Len: 1.0 / 3, Chance: 1}, {At: 1.0 / 3, Len: 1.0 / 3, Chance: 1}, {At: 2.0 / 3, Len: 1.0 / 3, Chance: 1}}},
		{"[x x x]*2", 2, []patcher.Hit{{At: 0, Len: 2.0 / 3, Chance: 1}, {At: 2.0 / 3, Len: 2.0 / 3, Chance: 1}, {At: 4.0 / 3, Len: 2.0 / 3, Chance: 1}}},
		{"x*2 x", 3, []patcher.Hit{{At: 0, Len: 2, Chance: 1}, {At: 2, Len: 1, Chance: 1}}},
		{"x? x?25", 2, []patcher.Hit{{At: 0, Len: 1, Chance: 0.5}, {At: 1, Len: 1, Chance: 0.25}}},
		{"[x . x2]", 1, []patcher.Hit{{At: 0, Len: 1.0 / 3, Chance: 1}, {At: 2.0 / 3, Len: 1.0 / 6, Chance: 1}, {At: 5.0 / 6, Len: 1.0 / 6, Chance: 1}}},
		{"1010", 4, []patcher.Hit{{At: 0, Len: 1, Chance: 1}, {At: 2, Len: 1, Chance: 1}}},
		{"x | x\nx", 3, []patcher.Hit{{At: 0, Len: 1, Chance: 1}, {At: 1, Len: 1, Chance: 1}, {At: 2, Len: 1, Chance: 1}}},
	} {
		p, err := patcher.ParsePattern(c.text)
		if err != nil {
			t.Errorf("%q: %v", c.text, err)
			continue
		}
		near := func(a, b float64) bool { return math.Abs(a-b) < 1e-9 }
		ok := near(p.Steps, c.steps) && len(p.Hits) == len(c.hits)
		for i := 0; ok && i < len(c.hits); i++ {
			h, w := p.Hits[i], c.hits[i]
			ok = near(h.At, w.At) && near(h.Len, w.Len) && near(h.Chance, w.Chance)
		}
		if !ok {
			t.Errorf("%q: %v steps %v, want %v steps %v", c.text, p.Steps, p.Hits, c.steps, c.hits)
		}
	}
	for _, bad := range []string{"", "x ?", "[x x", "x]", "[]", "x0", "x*", "x*0", "x?200", "y"} {
		if _, err := patcher.ParsePattern(bad); err == nil {
			t.Errorf("%q read", bad)
		}
	}
}

// triggers keeps the triggers it gets: the sample they fall on, how long they last
type triggers struct {
	*psalm.Module
	at      []int64
	seconds []float64
	vel     []float64
}

func (tr *triggers) Receive(v any, _ int) {
	if ev, ok := v.(patcher.Event); ok {
		tr.at = append(tr.at, ev.Tick.SampleTimestamp)
		tr.seconds = append(tr.seconds, ev.Seconds)
		tr.vel = append(tr.vel, ev.Vel)
	}
}

// a clock of quarters at 120 bpm (a step: 22050 samples) playing a pattern: where its triggers fall
func clockPlays(t *testing.T, pattern string, swing, seconds float64) *triggers {
	t.Helper()
	return clockLanes(t, pattern, "", 0, swing, seconds)
}

// the same with drift (how far the timing wanders, in steps)
func clockDrift(t *testing.T, drift, seconds float64) *triggers {
	t.Helper()
	return clockRun(t, "x", "", 0, drift, 0.5, seconds)
}

// the same with mutations (how much, the seed, frozen from the start)
func clockMutate(t *testing.T, pattern string, mutate float64, seed int, freeze bool, seconds float64) *triggers {
	t.Helper()
	return clockRun2(t, pattern, "", 0, 0, 0.5, seconds, func(c *patcher.Clock) { c.Mutate, c.Seed, c.Freeze = mutate, seed, freeze })
}

// the same with lanes and a groove (an index of rhythm.Grooves)
func clockLanes(t *testing.T, pattern, lanes string, groove int, swing, seconds float64) *triggers {
	t.Helper()
	return clockRun(t, pattern, lanes, groove, 0, swing, seconds)
}

func clockRun(t *testing.T, pattern, lanes string, groove int, drift, swing, seconds float64) *triggers {
	t.Helper()
	return clockRun2(t, pattern, lanes, groove, drift, swing, seconds, nil)
}

func clockRun2(t *testing.T, pattern, lanes string, groove int, drift, swing, seconds float64, setup func(*patcher.Clock)) *triggers {
	t.Helper()
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	defer psalm.PopConfig()
	p := psalm.New(nil)
	tr := &triggers{Module: psalm.NewControl()}
	tr.SetSelf(tr)
	kinds := append(patcher.Generators(), patcher.Kind{Name: "triggers", Ins: []patcher.Port{{Name: "trigger", Type: patcher.Trigger}},
		Make: func(string) (*patcher.Made, error) { return &patcher.Made{Module: tr}, nil }})
	pt := patcher.New(p, panel.New(p, "Test"), "Patch", kinds)
	clock, _ := pt.Add("clock", "", 0, 0)
	h, _ := pt.Add("triggers", "", 0, 0)
	c := pt.Module(clock).(*patcher.Clock) // before it plays: its settings would come with its first tick
	c.Division, c.Swing, c.Groove, c.Drift = 0, swing, groove, drift
	if setup != nil {
		setup(c)
	}
	if err := pt.SetText(clock, "lanes", lanes); err != nil || pt.TextError(clock, "lanes") != "" {
		t.Fatal(err, pt.TextError(clock, "lanes"))
	}
	if err := pt.SetText(clock, "pattern", pattern); err != nil || pt.TextError(clock, "pattern") != "" {
		t.Fatal(err, pt.TextError(clock, "pattern"))
	}
	if _, err := pt.Connect(patcher.Cable{From: clock, To: h}); err != nil {
		t.Fatal(err)
	}
	for range int(seconds * 44100 / 256) {
		p.ResetRun()
		p.Run()
	}
	return tr
}

// rolls, groups, longer steps, chances, swing: the triggers where they fall (to a tick: 92 samples)
func TestClockPatterns(t *testing.T) {
	for _, c := range []struct {
		pattern string
		swing   float64
		at      []float64 // in samples
		seconds []float64 // how long the first ones last
	}{
		{"x3 .", 0.5, []float64{0, 7350, 14700, 44100, 51450}, []float64{0.5 / 3}},
		{"[x x x]*2 x", 0.5, []float64{0, 14700, 29400, 44100, 66150}, []float64{1.0 / 3, 1.0 / 3, 1.0 / 3, 0.5}},
		{"x*2 x", 0.5, []float64{0, 44100, 66150}, []float64{1, 0.5}},
		{"x?0 x", 0.5, []float64{22050, 66150, 110250}, nil},
		{"x2 x2", 0.75, []float64{0, 16537.5, 33075, 38587.5, 44100}, nil}, // swung: the first step 1.5 long, the second 0.5
	} {
		tr := clockPlays(t, c.pattern, c.swing, 2.6)
		ok := len(tr.at) >= len(c.at)
		for i := 0; ok && i < len(c.at); i++ {
			ok = math.Abs(float64(tr.at[i])-c.at[i]) <= 92
		}
		for i := 0; ok && i < len(c.seconds); i++ {
			ok = math.Abs(tr.seconds[i]-c.seconds[i]) < 1e-9
		}
		if !ok {
			t.Errorf("%q (swing %v): triggers at %v lasting %v, want at %v lasting %v", c.pattern, c.swing, tr.at, tr.seconds, c.at, c.seconds)
		}
	}
}

// a voice plays what the clock, the sequence and the notes play with the cables between them: the same notes on the same samples, per section of the song
func TestVoiceIsTheThreeInOne(t *testing.T) {
	run := func(voice bool) *played {
		r := newHarmonyRig(t, "A 1, B 1", "A: Dm7\nB: G7")
		pattern := "A: x . x x . x . .\nB: x x . x"
		values := "A: C3 E3 G3\nB: D3 F3"
		if voice {
			v := r.add("voice")
			for k, text := range map[string]string{"pattern": pattern, "values": values} {
				if err := r.pt.SetText(v, k, text); err != nil || r.pt.TextError(v, k) != "" {
					t.Fatal(k, err, r.pt.TextError(v, k))
				}
			}
			r.connect(patcher.Cable{From: v, To: r.add("rec")})
		} else {
			clock, seq, notes := r.add("clock"), r.add("sequence"), r.add("notes")
			_ = r.pt.SetText(clock, "pattern", pattern)
			_ = r.pt.SetText(seq, "values", values)
			r.connect(patcher.Cable{From: clock, To: seq})
			r.connect(patcher.Cable{From: clock, To: notes})
			r.connect(patcher.Cable{From: seq, To: notes, In: 1})
			r.connect(patcher.Cable{From: notes, To: r.add("rec")})
		}
		r.run(4.1)
		return r.rec
	}
	trio, one := run(false), run(true)
	if len(trio.pitches) < 8 {
		t.Fatalf("the three play %d notes", len(trio.pitches))
	}
	if !slices.Equal(trio.pitches, one.pitches) || !slices.Equal(trio.at, one.at) {
		t.Errorf("the voice plays %v on %v, the three %v on %v", one.pitches, one.at, trio.pitches, trio.at)
	}
}
