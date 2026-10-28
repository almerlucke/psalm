package patcher_test

import (
	"math"
	"slices"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/script"
)

// chordsHeard keeps the chords a harmony sends, the sample of each, and the bass notes
type chordsHeard struct {
	*psalm.Module
	chords [][]float64
	at     []int64
	basses []float64
	change []int64
}

func (c *chordsHeard) Receive(v any, i int) {
	switch x := v.(type) {
	case []float64:
		c.chords = append(c.chords, x)
	case float64:
		c.basses = append(c.basses, x)
	case patcher.Event:
		c.change = append(c.change, x.Tick.SampleTimestamp)
	}
}

// harmonyRig is a patch at 120 bpm (a bar of 4 beats is 2 s) with a song, a harmony, a synth that records and a module that hears the chords
type harmonyRig struct {
	*songRig
	harmony string
	chords  *chordsHeard
}

func newHarmonyRig(t *testing.T, sections, chords string) *harmonyRig {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	t.Cleanup(func() { psalm.PopConfig() })
	r := &songRig{t: t, p: psalm.New(nil)}
	r.rec = &played{Module: psalm.NewControl()}
	r.rec.SetSelf(r.rec)
	hr := &harmonyRig{songRig: r, chords: &chordsHeard{Module: psalm.NewControl()}}
	hr.chords.SetSelf(hr.chords)
	kinds := append(patcher.Generators(), script.Kind(script.NewRuntime()),
		patcher.Kind{Name: "rec", Ins: []patcher.Port{{Name: "notes", Type: patcher.Notes}},
			Make: func(string) (*patcher.Made, error) { return &patcher.Made{Module: r.rec}, nil }},
		patcher.Kind{Name: "hear", Ins: []patcher.Port{{Name: "chord", Type: patcher.Value}, {Name: "bass", Type: patcher.Value, Index: 1}, {Name: "change", Type: patcher.Trigger, Index: 2}},
			Make: func(string) (*patcher.Made, error) { return &patcher.Made{Module: hr.chords}, nil }})
	r.pt = patcher.New(r.p, panel.New(r.p, "Test"), "Patch", kinds)
	r.p.Out(r.pt.Out())
	r.song = r.add("song")
	if err := r.pt.SetText(r.song, "sections", sections); err != nil || r.pt.TextError(r.song, "sections") != "" {
		t.Fatal(err, r.pt.TextError(r.song, "sections"))
	}
	hr.harmony = r.add("harmony")
	if err := r.pt.SetText(hr.harmony, "chords", chords); err != nil || r.pt.TextError(hr.harmony, "chords") != "" {
		t.Fatal(err, r.pt.TextError(hr.harmony, "chords"))
	}
	return hr
}

// the harmony sends the chord when it changes: per section, a number of bars each, looping, the chord as pitches from its lowest note, its bass, a trigger
func TestHarmonyChords(t *testing.T) {
	r := newHarmonyRig(t, "A 3, B 2", "A: Dm7 G7*2\nB: Cmaj7*0.5 Am7*1.5")
	hear := r.add("hear")
	for i := range 3 {
		r.connect(patcher.Cable{From: r.harmony, Out: i, To: hear, In: i})
	}
	r.run(9.9) // A (3 bars), B (2 bars)
	want := [][]float64{{50, 53, 57, 60}, {55, 59, 62, 65}, {48, 52, 55, 59}, {57, 60, 64, 67}}
	if len(r.chords.chords) != len(want) {
		t.Fatalf("%d chords sent: %v, want %v", len(r.chords.chords), r.chords.chords, want)
	}
	for i := range want {
		if !slices.Equal(r.chords.chords[i], want[i]) {
			t.Errorf("chord %d is %v, want %v", i, r.chords.chords[i], want[i])
		}
	}
	// the bass notes: the root at or above the voicing's lowest note an octave down
	if wantBass := []float64{38, 43, 36, 45}; !slices.Equal(r.chords.basses, wantBass) {
		t.Errorf("basses %v, want %v", r.chords.basses, wantBass)
	}
	// the changes: bar 0 (A), bar 1 (G7 for two bars), bar 3 (B: C for half a bar), bar 3.5 (A minor)
	at := func(bar float64) int64 { return int64(bar * 88200) }
	for i, bar := range []float64{0, 1, 3, 3.5} {
		if i >= len(r.chords.change) || math.Abs(float64(r.chords.change[i]-at(bar))) > 100 {
			t.Errorf("change %d at sample %v, want about %d (bar %v)", i, r.chords.change, at(bar), bar)
			break
		}
	}
}

// roman numerals are in the key and the scale of the settings, a - holds the chord, a . is no chord
func TestHarmonyNumerals(t *testing.T) {
	r := newHarmonyRig(t, "A 4", "i iv - .")
	hear := r.add("hear")
	r.connect(patcher.Cable{From: r.harmony, Out: 0, To: hear})
	r.pt.SettingsOf(r.harmony).Apply(map[string]float64{"key": 2, "scale": float64(slicesIndex(patcher.Scales, "minor"))}) // D minor
	r.run(7.9)
	got := r.chords.chords // (the first tick had the key of C: Cm, before the settings were in)
	if want := [][]float64{{50, 53, 57}, {55, 58, 62}}; len(got) < 2 || !slices.Equal(got[len(got)-2], want[0]) || !slices.Equal(got[len(got)-1], want[1]) {
		t.Errorf("chords %v, want %v (Dm, then Gm for two bars, then none)", got, want)
	}
	if err := r.pt.SetText(r.harmony, "chords", "Dm7 Hm9"); err != nil {
		t.Fatal(err)
	}
	if e := r.pt.TextError(r.harmony, "chords"); e == "" {
		t.Errorf("a bad chord isn't reported")
	}
}

func slicesIndex(list []string, name string) int { return slices.Index(list, name) }

// a part with chord notes follows the harmony: the root, the fifth, the bass, the whole chord, an octave up and down, a tone below the root
func TestPartChordNotes(t *testing.T) {
	r := newHarmonyRig(t, "A 2", "Dm7 G7")
	part := r.add("part")
	pp := r.pt.Module(part).(*patcher.Part)
	pp.Division, pp.Length = 0, 1 // quarters, legato
	if err := r.pt.SetText(part, "notes", "@1 @3 @b @c"); err != nil || r.pt.TextError(part, "notes") != "" {
		t.Fatal(err, r.pt.TextError(part, "notes"))
	}
	r.connect(patcher.Cable{From: part, To: r.add("rec")})
	r.run(3.9)
	want := []float64{50, 57, 38, 50, 53, 57, 60, 55, 62, 43, 55, 59, 62, 65}
	if !slices.Equal(r.rec.pitches, want) {
		t.Errorf("pitches %v, want %v (Dm7: root, fifth, bass, chord; G7 the same)", r.rec.pitches, want)
	}

	// octaves, the tones past the chord, the tone before the root, a chord with a note
	r.rec.pitches = nil
	r.p.Restart()
	if err := r.pt.SetText(part, "notes", "@1' @1, @5 @-1\nB: ."); err != nil || r.pt.TextError(part, "notes") != "" {
		t.Fatal(err, r.pt.TextError(part, "notes"))
	}
	r.run(1.9) // Dm7 for a bar
	if want := []float64{62, 38, 62, 48}; !slices.Equal(r.rec.pitches, want) {
		t.Errorf("pitches %v, want %v (root up an octave, root down, the fifth tone of a four-tone chord is the root an octave up, the tone before the root is the seventh below)", r.rec.pitches, want)
	}

	r.rec.pitches = nil
	r.p.Restart()
	_ = r.pt.SetText(part, "notes", "C3+@3 .")
	r.run(0.6)
	if want := []float64{48, 57}; !slices.Equal(r.rec.pitches, want) {
		t.Errorf("pitches %v, want %v (a note and a chord note together)", r.rec.pitches, want)
	}
}

// without a harmony the chord notes rest; a harmony with no chord in a section rests them too
func TestPartChordNotesWithoutChords(t *testing.T) {
	r := newSongRig(t, "A 1")
	part := r.add("part")
	pp := r.pt.Module(part).(*patcher.Part)
	pp.Division = 0
	if err := r.pt.SetText(part, "notes", "@1 C3 @3 D3"); err != nil || r.pt.TextError(part, "notes") != "" {
		t.Fatal(err, r.pt.TextError(part, "notes"))
	}
	r.connect(patcher.Cable{From: part, To: r.add("rec")})
	r.run(1.9)
	if want := []float64{48, 50}; !slices.Equal(r.rec.pitches, want) {
		t.Errorf("pitches %v, want %v", r.rec.pitches, want)
	}
	if err := r.pt.SetText(part, "notes", "@9x"); err != nil {
		t.Fatal(err)
	}
	if e := r.pt.TextError(part, "notes"); e == "" {
		t.Errorf("a bad chord note isn't reported")
	}
}

// a script asks the harmony for its chord
func TestScriptAsksTheHarmony(t *testing.T) {
	r := newHarmonyRig(t, "A 2", "Dm7 G7")
	s := r.add("script")
	if err := r.pt.SetText(s, "code", `
clock("1/4")
function trigger(ev) { const c = harmony(); note(c ? c.close(48)[0] : 0, 0.8, 1) }`); err != nil || r.pt.TextError(s, "code") != "" {
		t.Fatal(err, r.pt.TextError(s, "code"))
	}
	r.connect(patcher.Cable{From: s, To: r.add("rec"), Type: patcher.Notes})
	r.run(3.9)
	if len(r.rec.pitches) < 7 || r.rec.pitches[0] != 50 || r.rec.pitches[len(r.rec.pitches)-1] != 55 {
		t.Errorf("roots %v, want D (50) in the first bar and G (55) in the second", r.rec.pitches)
	}
}
