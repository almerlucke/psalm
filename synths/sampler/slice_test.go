package sampler

import (
	"math"
	"os"
	"testing"

	"github.com/almerlucke/psalm"
)

// hits is a sound of decaying sines starting at the given seconds, the pitch of each a different one
func hits(total float64, starts ...float64) Sound {
	buf := make([]float32, int(total*sr))
	for k, s := range starts {
		for i := int(s * sr); i < len(buf); i++ {
			t := float64(i)/sr - s
			buf[i] += float32(0.8 * math.Exp(-t*25) * math.Sin(2*math.Pi*(300+100*float64(k))*t))
		}
	}
	return NewSound(sr, buf)
}

func TestOnsetCutsFindTheHits(t *testing.T) {
	starts := []float64{0.1, 0.6, 1.1, 1.6}
	snd := hits(2.1, starts...)
	cuts := OnsetCuts(snd, 9, 0.08)
	if len(cuts) != len(starts)+2 {
		t.Fatalf("cuts %v, want 0, the four hits and the end", cuts)
	}
	if cuts[0] != 0 || cuts[len(cuts)-1] != snd.NumFrames() {
		t.Errorf("the first and last cuts are %d and %d", cuts[0], cuts[len(cuts)-1])
	}
	for i, s := range starts {
		if d := math.Abs(float64(cuts[i+1])/sr - s); d > 0.012 {
			t.Errorf("hit %d found at %.3f s, it is at %.3f", i, float64(cuts[i+1])/sr, s)
		}
	}
}

func TestEvenCuts(t *testing.T) {
	snd := NewSound(sr, sine(200, 2))
	cuts := EvenCuts(snd, 4)
	if len(cuts) != 5 || cuts[0] != 0 || cuts[4] != snd.NumFrames() {
		t.Fatalf("cuts %v", cuts)
	}
	for i := 1; i < 4; i++ {
		if d := math.Abs(float64(cuts[i]) - float64(snd.NumFrames())*float64(i)/4); d > 0.005*sr {
			t.Errorf("cut %d is %v frames off", i, d)
		}
	}
}

func TestSlicesPlayTheirPieces(t *testing.T) {
	snd := hits(2.1, 0.1, 0.6, 1.1, 1.6)
	cuts := OnsetCuts(snd, 9, 0.08)
	regions := Slices(snd, 36, cuts)
	if len(regions) != 5 || regions[0].LoKey != 36 || regions[4].HiKey != 40 {
		t.Fatalf("%d regions", len(regions))
	}
	v := voice(regions...)
	// the third piece starts with its hit (the second of the four), the length of the piece, and is silent after it, though the key is held
	x := play(v, psalm.Message{"pitch": 38.0}, buffers(1))
	want := int(cuts[3] - cuts[2])
	peak := 0.0
	for _, y := range x[:600] {
		peak = math.Max(peak, math.Abs(y))
	}
	if peak < 0.05 {
		t.Errorf("the piece does not begin with its hit: %g", peak)
	}
	tail := 0.0
	for _, y := range x[want+500 : want+3000] {
		tail = math.Max(tail, math.Abs(y))
	}
	if tail > 1e-6 {
		t.Errorf("sound after the end of the piece: %g", tail)
	}
	// the pitch is the one it was recorded at
	if f := frequency(x[600:3000]); math.Abs(f-400) > 25 {
		t.Errorf("%g Hz, the hit was 400", f)
	}
}

func TestChangeCutsFindTheChangesOfALegatoLine(t *testing.T) {
	// five notes of the same level, one into the next with no gap, half a second each
	var buf []float32
	for k := range 5 {
		freq := 300 * math.Pow(1.25, float64(k))
		for i := 0; i < int(0.5*sr); i++ {
			buf = append(buf, float32(0.5*math.Sin(2*math.Pi*freq*float64(i)/sr)+0.2*math.Sin(2*math.Pi*2*freq*float64(i)/sr)))
		}
	}
	snd := NewSound(sr, buf)
	if cuts := OnsetCuts(snd, 9, 0.1); len(cuts) != 2 {
		t.Errorf("the level does not move, the hits are %v", cuts)
	}
	cuts := ChangeCuts(snd, 1.5, 0.2)
	if len(cuts) != 6 {
		t.Fatalf("cuts %v, want the four changes between the five notes", cuts)
	}
	for i := 1; i < 5; i++ {
		if d := math.Abs(float64(cuts[i])/sr - 0.5*float64(i)); d > 0.06 {
			t.Errorf("change %d found at %.3f s, it is at %.1f", i, float64(cuts[i])/sr, 0.5*float64(i))
		}
	}
}

func TestSlicesDoNotClick(t *testing.T) {
	// a piece cut out of a loud steady tone ends in the middle of a wave: the end must fade, not jump
	snd := NewSound(sr, sine(330.3, 2))
	regions := Slices(snd, 36, []int64{0, 30001, 60007})
	v := voice(regions...)
	x := play(v, psalm.Message{"pitch": 36.0}, buffers(1))
	biggest := 0.0
	for i := 100; i < 40000; i++ {
		biggest = math.Max(biggest, math.Abs(x[i]-x[i-1]))
	}
	// the step of a sine of 330 Hz is 0.047; a jump from the wave to nothing would be as big as the wave
	if biggest > 0.06 {
		t.Errorf("a step of %g in the piece or at its end", biggest)
	}
}

func TestSlicesInOrderAreTheRecording(t *testing.T) {
	// played one right after the other, the pieces add up to the recording: no gaps, no dips where they meet
	snd := NewSound(sr, sine(330.3, 2))
	cuts := []int64{0, 30001, 60007}
	regions := Slices(snd, 36, cuts)
	a := play(voice(regions...), psalm.Message{"pitch": 36.0}, buffers(1.6))
	b := play(voice(regions...), psalm.Message{"pitch": 37.0}, buffers(1.6))
	orig := snd.Buffer(0, 0)
	worst := 0.0
	for i := 2000; i < 58000; i++ {
		got := a[i]
		if j := i - int(cuts[1]); j >= 0 {
			got += b[j]
		}
		worst = math.Max(worst, math.Abs(got-float64(orig[i])))
	}
	if worst > 0.01 {
		t.Errorf("the pieces in order differ from the recording by %g", worst)
	}
}

func TestParseBanks(t *testing.T) {
	load := func(path string) (Sound, error) {
		if path == "gone.wav" {
			return nil, os.ErrNotExist
		}
		return hits(2.1, 0.1, 0.6, 1.1, 1.6), nil
	}
	inst, report, err := ParseBanks(`
# a loop cut at its hits, a whole sound across keys, and a quiet even cut
36 loop.wav hits 9 0.08 vol=-6 name=loop
60 pad.wav whole 72 66
80 cut.wav even 4 pan=0.5
`, load)
	if err != nil {
		t.Fatal(err)
	}
	if lo, hi := inst.Keys(); lo != 36 || hi != 83 {
		t.Errorf("keys %d to %d", lo, hi)
	}
	if len(inst.Regions) != 5+1+4 || inst.Regions[0].Name != "loop" || inst.Regions[0].Volume != -6 || inst.Regions[5].HiKey != 72 || inst.Regions[5].Root != 66 || inst.Regions[6].Pan != 0.5 {
		t.Errorf("regions %+v", inst.Regions)
	}
	if len(report) != 3 || report[0] != "loop: 5 on keys 36-40" {
		t.Errorf("report %q", report)
	}
	for _, bad := range []string{"x loop.wav hits", "36 gone.wav hits", "36 loop.wav chop", "36 loop.wav hits 9 vol=loud", "36 loop.wav"} {
		if _, _, err := ParseBanks(bad, load); err == nil {
			t.Errorf("%q: no error", bad)
		}
	}
	// a voice takes another instrument by message
	v := voice(NewRegion(NewSound(sr, constant(0.5, 1))).Keys(0, 127, 60))
	v.ReceiveMessage(psalm.Message{"instrument": inst})
	if v.cfg.Instrument != inst {
		t.Errorf("the voice did not take the instrument")
	}
	// key 60 of the new one is the pad: its sound is a decaying sine, and the old one was a constant
	x := play(v, psalm.Message{"pitch": 60.0}, buffers(0.3))
	if math.Abs(x[len(x)-1]) > 0.45 {
		t.Errorf("still the constant sound of the old instrument: %g", x[len(x)-1])
	}
}
