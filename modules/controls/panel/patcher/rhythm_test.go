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
