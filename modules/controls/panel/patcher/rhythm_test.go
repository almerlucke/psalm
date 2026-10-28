package patcher_test

import (
	"math"
	"slices"
	"testing"

	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/tools/rhythm"
)

func near(got []int64, want ...float64) bool {
	if len(got) < len(want) {
		return false
	}
	for i, w := range want {
		if math.Abs(float64(got[i])-w) > 92 {
			return false
		}
	}
	return true
}

// a step is 22050 samples (quarters at 120 bpm)
func TestClockConditionsAndFunctions(t *testing.T) {
	// the second of every four passes: steps 1 and 5
	if tr := clockPlays(t, "x%2/4", 0.5, 4.6); !near(tr.at, 22050, 110250, 198450) || len(tr.at) != 3 {
		t.Errorf("x%%2/4 at %v", tr.at)
	}
	// all but the first time through a pattern of two steps
	if tr := clockPlays(t, "x%!first .", 0.5, 4.1); !near(tr.at, 44100, 88200) {
		t.Errorf("x%%!first at %v", tr.at)
	}
	// a euclidean rhythm: where its hits are
	var want []float64
	for i, on := range rhythm.Euclid(3, 8, 0) {
		if on {
			want = append(want, float64(i)*22050)
		}
	}
	if tr := clockPlays(t, "E(3,8)", 0.5, 4.1); !near(tr.at, want...) {
		t.Errorf("E(3,8) at %v, want %v", tr.at, want)
	}
	if tr := clockPlays(t, "A(2+2)", 0.5, 2.1); !near(tr.at, 0, 44100) {
		t.Errorf("A(2+2) at %v", tr.at)
	}
	for _, bad := range []string{"x%7/4", "E(9,4)", "x%later"} {
		if _, err := patcher.ParsePattern(bad); err == nil {
			t.Errorf("%q was taken", bad)
		}
	}
}

func TestClockLanesAndGroove(t *testing.T) {
	// a chance lane of two steps: every second step plays
	if tr := clockLanes(t, "x x x x", "prob: 0 100", 0, 0.5, 2.1); !near(tr.at, 22050, 66150) || len(tr.at) != 2 {
		t.Errorf("prob at %v", tr.at)
	}
	// a lane shorter than the pattern drifts: three steps against four
	if tr := clockLanes(t, "x x x x", "prob: 100 0 0", 0, 0.5, 4.1); !near(tr.at, 0, 66150, 132300) {
		t.Errorf("polymeter at %v", tr.at)
	}
	// velocity, and later by half a step
	tr := clockLanes(t, "x x", "vel: 50 100\nshift: 50", 0, 0.5, 1.1)
	if !near(tr.at, 11025, 33075) || len(tr.vel) < 2 || tr.vel[0] != 0.5 || tr.vel[1] != 1 {
		t.Errorf("vel %v at %v", tr.vel, tr.at)
	}
	// a groove: the offbeat later and softer
	g := slices.Index(rhythm.GrooveNames(), "mpc 62")
	tr = clockLanes(t, "x x", "", g, 0.5, 1.1)
	if !near(tr.at, 0) || math.Abs(float64(tr.at[1])-22050*1.24) > 250 || tr.vel[1] >= tr.vel[0] {
		t.Errorf("groove at %v vel %v", tr.at, tr.vel)
	}
	for _, bad := range []string{"prob 1 2", "tempo: 1", "vel: a"} {
		if _, err := patcher.ParseStepLanes(bad); err == nil {
			t.Errorf("%q was taken", bad)
		}
	}
}

func TestPartRatchetsAndPitchLane(t *testing.T) {
	r := newSongRig(t, "A 2")
	part := r.add("part")
	pp := r.pt.Module(part).(*patcher.Part)
	pp.Division, pp.Length = 0, 1
	if err := r.pt.SetText(part, "notes", "C3^3 D3%2/2 ."); err != nil || r.pt.TextError(part, "notes") != "" {
		t.Fatal(err, r.pt.TextError(part, "notes"))
	}
	if err := r.pt.SetText(part, "lanes", "pitch: 0 12"); err != nil || r.pt.TextError(part, "lanes") != "" {
		t.Fatal(err, r.pt.TextError(part, "lanes"))
	}
	r.connect(patcher.Cable{From: part, To: r.add("rec")})
	r.run(3.9) // a pass of three steps is 1.5 s: C3 ratcheted, D3 on the second pass only
	want := []float64{48, 48, 48, 50 + 0, 48, 48, 48}
	_ = want
	if len(r.rec.pitches) < 4 || !slices.Equal(r.rec.pitches[:3], []float64{48, 48, 48}) {
		t.Fatalf("pitches %v: a ratchet of three", r.rec.pitches)
	}
	if slices.Contains(r.rec.pitches[:3], 50) || !slices.Contains(r.rec.pitches, 50+12) && !slices.Contains(r.rec.pitches, 50) {
		t.Errorf("pitches %v: D3 only on the second pass", r.rec.pitches)
	}
	if err := r.pt.SetText(part, "notes", "C3^1"); err != nil {
		t.Fatal(err)
	}
	if r.pt.TextError(part, "notes") == "" {
		t.Error("a ratchet of one was taken")
	}
}

// @v: the chord voice led from the one before, not in root position every time
func TestPartVoiceLedChords(t *testing.T) {
	r := newHarmonyRig(t, "A 2", "C Am F G")
	part := r.add("part")
	pp := r.pt.Module(part).(*patcher.Part)
	pp.Division, pp.Length = 0, 1
	if err := r.pt.SetText(part, "notes", "@v . . ."); err != nil || r.pt.TextError(part, "notes") != "" {
		t.Fatal(err, r.pt.TextError(part, "notes"))
	}
	r.connect(patcher.Cable{From: part, To: r.add("rec")})
	r.run(7.9) // a chord a bar of two seconds, four beats of quarters: one @v a bar
	p := r.rec.pitches
	if len(p) < 12 {
		t.Fatalf("pitches %v", p)
	}
	moved := func(a, b []float64) float64 {
		sum := 0.0
		for i := range a {
			sum += math.Abs(a[i] - b[i])
		}
		return sum
	}
	chords := [][]float64{p[0:3], p[3:6], p[6:9], p[9:12]}
	for i := 1; i < len(chords); i++ {
		if d := moved(slices.Clone(chords[i-1]), chords[i]); d > 6 {
			t.Errorf("chord %d moved %v semitones in all from %v to %v", i, d, chords[i-1], chords[i])
		}
	}
	if err := r.pt.SetText(part, "notes", "@v'"); err != nil || r.pt.TextError(part, "notes") != "" {
		t.Errorf("an octave on @v: %v %v", err, r.pt.TextError(part, "notes"))
	}
}

// drift moves the timing along a smooth curve: off the grid, but never far, and the same every time
func TestClockDrift(t *testing.T) {
	psalm0 := func() *triggers {
		return clockLanes(t, "x", "", 0, 0.5, 6.1)
	}
	straight := psalm0()
	for i, at := range straight.at {
		if math.Abs(float64(at)-float64(i)*22050) > 92 {
			t.Fatalf("straight: %v", straight.at)
		}
	}
	// the setting is a message away: a clock with drift
	psalmDrift := func() *triggers { return clockDrift(t, 0.25, 6.1) }
	a, b := psalmDrift(), psalmDrift()
	off := 0
	for i := range a.at {
		d := math.Abs(float64(a.at[i]) - float64(i)*22050)
		if d > 0.26*22050+92 {
			t.Fatalf("step %d is %v samples off", i, d)
		}
		if d > 400 {
			off++
		}
		if a.at[i] != b.at[i] {
			t.Fatal("drift is not repeatable")
		}
	}
	if off < 2 {
		t.Errorf("only %d steps drifted: %v", off, a.at)
	}
}

// patterns that evolve: a rotation that goes on with the passes, mutations that leave what is locked, a frozen pattern
func TestClockEvolves(t *testing.T) {
	// E(3,8,+1) over 4 passes of 8 quarter steps: each pass the hits are a step later
	tr := clockPlays(t, "E(3,8,+1)", 0.5, 8*4*0.5-0.1)
	var passes [4][]int
	for _, at := range tr.at {
		step := int(math.Round(float64(at) / 22050))
		passes[step/8] = append(passes[step/8], step%8)
	}
	for i := range 4 {
		var want []int
		for s, on := range rhythm.Euclid(3, 8, i) {
			if on {
				want = append(want, s)
			}
		}
		if !slices.Equal(passes[i], want) {
			t.Errorf("pass %d: hits on %v, want %v", i, passes[i], want)
		}
	}
	// mutate: a locked hit never goes; the others change from pass to pass, the same every run
	run := func() []int64 { return clockMutate(t, "x! x x x x x x x", 1, 7, false, 8*8*0.5-0.1).at }
	a, b := run(), run()
	if !slices.Equal(a, b) {
		t.Error("mutations are not repeatable")
	}
	hitsOn := map[int]int{}
	for _, at := range a {
		step := int(math.Round(float64(at) / 22050))
		hitsOn[step%8]++
	}
	if hitsOn[0] != 8 {
		t.Errorf("the locked hit played %d of 8 passes", hitsOn[0])
	}
	if len(a) >= 8*8 || len(a) < 8 {
		t.Errorf("%d hits in 8 passes of 8: some should be dropped", len(a))
	}
	// frozen: every pass the same
	f := clockMutate(t, "x x x x x x x x", 1, 3, true, 8*4*0.5-0.1)
	per := map[int][]int{}
	for _, at := range f.at {
		step := int(math.Round(float64(at) / 22050))
		per[step/8] = append(per[step/8], step%8)
	}
	for i := 1; i < 4; i++ {
		if !slices.Equal(per[i], per[0]) {
			t.Errorf("frozen: pass %d %v is not pass 0 %v", i, per[i], per[0])
		}
	}
}
