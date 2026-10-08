package harmony

import (
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/almerlucke/psalm/modules/controls/score"
)

func TestParseChord(t *testing.T) {
	for symbol, want := range map[string][]int{
		"C":      {0, 4, 7},
		"Dm":     {2, 5, 9},
		"Bb":     {10, 2, 5},
		"A7":     {9, 1, 4, 7},
		"F#m7b5": {6, 9, 0, 4},
		"Ebmaj7": {3, 7, 10, 2},
		"Gsus4":  {7, 0, 2},
		"Bdim7":  {11, 2, 5, 8},
		"Cadd9":  {0, 4, 7, 2},
		"G/B":    {7, 11, 2},
		"C/E":    {0, 4, 7},
		"Am/G":   {9, 0, 4, 7},
	} {
		c, err := ParseChord(symbol)
		if err != nil {
			t.Fatalf("%s: %v", symbol, err)
		}
		if got := c.PitchClasses(); !slices.Equal(got, want) {
			t.Errorf("%s: pitch classes %v, want %v", symbol, got, want)
		}
	}
	for _, bad := range []string{"H", "Cxyz", "C/Q", ""} {
		if _, err := ParseChord(bad); err == nil {
			t.Errorf("%q: expected an error", bad)
		}
	}
}

func TestVoicings(t *testing.T) {
	g := MustChord("G/B")
	if got := g.Pitches(55); !slices.Equal(got, []float64{47, 55, 59, 62}) {
		t.Fatalf("G/B from G3: %v", got)
	}
	dm := MustChord("Dm")
	if got := dm.Close(60); !slices.Equal(got, []float64{62, 65, 69}) {
		t.Fatalf("Dm close from C4: %v", got)
	}
	if got := Inversion([]float64{62, 65, 69}, 1); !slices.Equal(got, []float64{65, 69, 74}) {
		t.Fatalf("first inversion: %v", got)
	}
	if got := Inversion([]float64{62, 65, 69}, -1); !slices.Equal(got, []float64{57, 62, 65}) {
		t.Fatalf("down inversion: %v", got)
	}
	if got := Drop2([]float64{60, 64, 67, 71}); !slices.Equal(got, []float64{55, 60, 64, 71}) {
		t.Fatalf("drop 2: %v", got)
	}
	if got := MustChord("Bb").BassPitch(36); got != 46 {
		t.Fatalf("bass %v", got)
	}
}

func TestVoiceLead(t *testing.T) {
	prev := []float64{62, 65, 69, 74} // Dm
	next := VoiceLead(prev, MustChord("Bb"))
	// the smoothest Bb: D F Bb D (only A moves, up a semitone)
	if !slices.Equal(next, []float64{62, 65, 70, 74}) {
		t.Fatalf("Dm to Bb: %v", next)
	}
	// every chord tone is used when there are enough voices
	g := VoiceLead([]float64{60, 64, 67}, MustChord("G7"))
	pcs := map[int]bool{}
	for _, p := range g {
		pcs[pitchClass(p)] = true
	}
	if len(pcs) != 3 {
		t.Fatalf("three voices should take three different tones of G7, got %v", g)
	}
	for i := range g {
		if d := g[i] - []float64{60, 64, 67}[i]; d > 2 || d < -2 {
			t.Fatalf("voices should move at most a step: %v", g)
		}
	}
}

func TestScales(t *testing.T) {
	dm := MustKey("D minor")
	if got := dm.Pitch(1, 4); got != 62 {
		t.Fatalf("degree 1 of D minor in octave 4: %v", got)
	}
	if got := dm.Pitch(8, 4); got != 74 {
		t.Fatalf("degree 8: %v", got)
	}
	if got := dm.Pitch(0, 4); got != 60 {
		t.Fatalf("degree 0 is the 7th below: %v", got)
	}
	if !dm.Contains(70) || dm.Contains(71) {
		t.Fatal("D minor has Bb, not B")
	}
	if got := dm.Snap(66); got != 65 { // F# snaps down to F (tie between F and G goes down)
		t.Fatalf("snap F#: %v", got)
	}
	if got := dm.Step(62, 2); got != 65 {
		t.Fatalf("two steps up from D: %v", got)
	}
	if got := dm.Step(62, -1); got != 60 {
		t.Fatalf("a step down from D: %v", got)
	}
	if got := len(MustScale("C", "minor pentatonic").Pitches(60, 72)); got != 6 {
		t.Fatalf("pentatonic pitches in an octave: %d", got)
	}
	if _, err := NewScale("D", "nope"); err == nil {
		t.Fatal("expected an error for an unknown mode")
	}
}

func TestDiatonicChords(t *testing.T) {
	c := MustKey("C major")
	var names []string
	for d := 1; d <= 7; d++ {
		names = append(names, c.Triad(d).Name)
	}
	if got := strings.Join(names, " "); got != "C Dm Em F G Am Bdim" {
		t.Fatalf("C major triads: %s", got)
	}
	if got := c.Seventh(5).Name; got != "G7" {
		t.Fatalf("V7 in C: %s", got)
	}
	if got := c.Seventh(7).Name; got != "Bm7b5" {
		t.Fatalf("vii7 in C: %s", got)
	}
	// D minor is spelled with flats
	if got := MustKey("D minor").Triad(6).Name; got != "Bb" {
		t.Fatalf("VI in D minor: %s", got)
	}
}

func TestRoman(t *testing.T) {
	dm := MustKey("D minor")
	var names []string
	for _, c := range dm.MustProgression("i VI iv V V7 VII ii° #vii°7 III+ bII") {
		names = append(names, c.Name)
	}
	if got := strings.Join(names, " "); got != "Dm Bb Gm A A7 C Edim C#dim7 Faug Eb" {
		t.Fatalf("roman numerals in D minor: %s", got)
	}
	c := MustKey("C major")
	if got := c.MustProgression("I vi IV V ii7 Vsus4")[4].Name; got != "Dm7" {
		t.Fatalf("ii7: %s", got)
	}
	if _, err := c.Roman("X"); err == nil {
		t.Fatal("expected an error")
	}
}

func TestArp(t *testing.T) {
	pitches := func(ns []score.Note) []float64 {
		var ps []float64
		for _, n := range ns {
			ps = append(ps, n.Pitches[0])
		}
		return ps
	}
	am := []float64{57, 60, 64}
	if got := pitches(Arp{Pattern: Up, Octaves: 2}.Notes(am)); !slices.Equal(got, []float64{57, 60, 64, 69, 72, 76}) {
		t.Fatalf("up two octaves: %v", got)
	}
	if got := pitches(Arp{Pattern: UpDown}.Notes(am)); !slices.Equal(got, []float64{57, 60, 64, 60}) {
		t.Fatalf("up down: %v", got)
	}
	if got := pitches(Arp{Pattern: OutsideIn, Octaves: 2}.Notes(am)); !slices.Equal(got, []float64{57, 76, 60, 72, 64, 69}) {
		t.Fatalf("outside in: %v", got)
	}
	ns := Arp{Pattern: Down, Steps: 8, Beats: 0.25, Velocities: []float64{1, 0.5}}.Notes(am)
	if len(ns) != 8 || ns[3].Pitches[0] != 64 || ns[0].Beats != 0.25 || ns[1].Velocity != 0.5 {
		t.Fatalf("down 8 steps: %+v", ns)
	}
	a, b := Arp{Pattern: RandomWalk, Steps: 16, Seed: 3}.Notes(am), Arp{Pattern: RandomWalk, Steps: 16, Seed: 3}.Notes(am)
	if !slices.Equal(pitches(a), pitches(b)) {
		t.Fatal("the same seed should give the same arpeggio")
	}
}

// The clashes of the showcase, found by hand while writing it, are found by Check
func TestCheck(t *testing.T) {
	progression := Timeline(4, MustProgression("Dm Bb Gm A F C Gm A")...)
	theme := score.MustParse(`A4:2 F4:1 E4 | D4:3 F4:1 | G4:2 Bb4:1 A4 | E4:2 C#4:1 E4 |
		A4:2 C5:1 A4 | G4:2 E4:1 G4 | Bb4:1.5 A4:.5 G4:1 D5 | C#5:2 A4 |
		D5:2 C5:1 A4 | Bb4:2 A4:1 F4 | G4:1 A4 Bb4 D5 | C#5:3 E5:1 |
		F5:2 E5:1 C5 | E5:2 D5:1 C5 | Bb4:2 A4:1 G4 | A4:4`)
	if issues := Check(theme, progression); len(issues) > 0 {
		t.Fatalf("the theme fits its chords, got:\n%s", Report("theme", issues))
	}

	// the theme's 4th bar (over A) put over the coda's C chord: C# against C
	homecoming := Timeline(4, MustProgression("Dm Bb F C")...)
	wrong := score.Join(score.MustParse("A4:2 F4:1 E4 | D4:3 F4:1 | G4:2 Bb4:1 A4 | E4:2 C#4:1 E4"))
	issues := Check(wrong, homecoming)
	if len(issues) != 1 || issues[0].Pitch != 61 || issues[0].Chord.Name != "C" || issues[0].Beat != 14 {
		t.Fatalf("expected the C# against C at beat 14, got:\n%s", Report("coda", issues))
	}

	// choir chords held 7 beats, into the next chord: A over Bb
	held := []score.Note{{Pitches: []float64{57, 62, 65}, Beats: 4, Hold: 7}, score.Rest(4)}
	issues = Check(held, progression)
	if len(issues) != 1 || issues[0].Pitch != 57 || issues[0].Chord.Name != "Bb" || !strings.Contains(issues[0].Reason, "held") {
		t.Fatalf("expected the held A against Bb, got:\n%s", Report("choir", issues))
	}
	// held only slightly into the next chord (a release) is fine
	held[0].Hold = 4.2
	if issues := Check(held, progression); len(issues) > 0 {
		t.Fatalf("a short release into the next chord is fine, got:\n%s", Report("choir", issues))
	}

	// colour notes are fine: a 7th over Dm, a major 7th over Bb, a 9th over Gm
	if issues := Check(score.MustParse("[D4 F4 A4 C5]:4 [F4 A4 Bb4 D5]:4 [G4 Bb4 D5 A5]:4 r:20"), progression); len(issues) > 0 {
		t.Fatalf("7ths and 9ths are colour, not clashes, got:\n%s", Report("colour", issues))
	}
	// an avoid note held for a whole bar: Eb (a semitone above D) over Dm
	if issues := Check(score.MustParse("Eb5:4 r:28"), progression); len(issues) != 1 {
		t.Fatalf("expected Eb over Dm to clash, got:\n%s", Report("avoid", issues))
	}
}

func TestTunings(t *testing.T) {
	equal := EqualTemperament(440)
	if equal(69) != 440 || math.Abs(equal(60)-261.6256) > 1e-3 {
		t.Fatalf("equal temperament: A4 %v, C4 %v", equal(69), equal(60))
	}
	just := MustKey("C major").Just(440)
	c := equal(60)
	for _, tc := range []struct {
		pitch, ratio float64
	}{{60, 1}, {64, 5.0 / 4}, {67, 3.0 / 2}, {69, 5.0 / 3}, {72, 2}, {55, 3.0 / 4}, {48, 0.5}, {76, 5.0 / 2}} {
		if got := just(tc.pitch); math.Abs(got-c*tc.ratio) > 1e-9 {
			t.Fatalf("just C major: pitch %v is %v Hz, expected %v", tc.pitch, got, c*tc.ratio)
		}
	}
	// the triads on I, IV, V, vi and iii are pure: a 3/2 fifth over every root
	for _, root := range []float64{60, 65, 67, 69, 64} {
		if r := just(root+7) / just(root); math.Abs(r-1.5) > 1e-9 {
			t.Fatalf("the fifth on %v should be pure, %v", root, r)
		}
	}
	if math.Abs(just(60.25)/just(60)-math.Exp2(0.25/12)) > 1e-12 {
		t.Fatal("a fraction should stay a fraction above the note")
	}
	if eb := MustKey("Eb major").Just(440); math.Abs(eb(63)-equal(63)) > 1e-9 || math.Abs(eb(67)/eb(63)-1.25) > 1e-12 {
		t.Fatal("the tonic should sit where equal temperament has it, the third a pure 5/4 above")
	}
}

func TestTriad(t *testing.T) {
	for _, tc := range []struct {
		symbol             string
		lowest             float64
		root, third, fifth float64
	}{
		{"Ebmaj7", 34, 39, 43, 46}, // the root at or above the lowest
		{"Am", 33, 33, 36, 40},
		{"C/E", 36, 36, 40, 43}, // root position, not the bass
		{"G5", 40, 43, 50, 50},  // a power chord repeats its fifth
	} {
		r, th, f := MustChord(tc.symbol).Triad(tc.lowest)
		if r != tc.root || th != tc.third || f != tc.fifth {
			t.Fatalf("%s from %v: %v %v %v", tc.symbol, tc.lowest, r, th, f)
		}
	}
}
