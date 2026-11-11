package tune

import (
	"math"
	"testing"
)

// a voice: a glide from 120 to 150 Hz with harmonics that fall away, a pause, a second note at 130 Hz
func voice(sr float64) []float64 {
	x := make([]float64, int(3*sr))
	phase := 0.0
	for i := range x {
		t := float64(i) / sr
		var f float64
		switch {
		case t < 1.2:
			f = 120 + 30*t/1.2
		case t > 1.8:
			f = 130
		default:
			continue
		}
		phase += f / sr
		var s float64
		for h := 1; h <= 20; h++ {
			s += math.Sin(2*math.Pi*phase*float64(h)) / float64(h)
		}
		env := min(1, min(t, 1.2-t+0.1)*20)
		if t > 1.8 {
			env = min(1, (t-1.8)*20) * min(1, (3-t)*20)
		}
		x[i] = 0.3 * s * env
	}
	return x
}

func TestDetect(t *testing.T) {
	const sr = 44100
	x := voice(sr)
	tr := Detect(x, sr, DetectOptions{})
	for _, c := range []struct{ at, want float64 }{{0.5, 120 + 30*0.5/1.2}, {1.0, 120 + 30/1.2}, {2.4, 130}} {
		got := tr.At(c.at * sr)
		if math.Abs(got/c.want-1) > 0.02 {
			t.Errorf("pitch at %.1f s is %.1f Hz, want %.1f", c.at, got, c.want)
		}
	}
	if tr.At(1.5*sr) != 0 {
		t.Errorf("a pause has the pitch %.1f", tr.At(1.5*sr))
	}
}

func TestShift(t *testing.T) {
	const sr = 44100
	x := voice(sr)
	tr := Detect(x, sr, DetectOptions{})
	for _, semis := range []float64{3, -4, 0.5} {
		target := Track{Hop: tr.Hop, SampleRate: sr, F0: make([]float64, len(tr.F0))}
		for i, f := range tr.F0 {
			target.F0[i] = f * math.Pow(2, semis/12)
		}
		y := Shift(x, tr, target)
		if len(y) != len(x) {
			t.Fatalf("length %d, was %d", len(y), len(x))
		}
		for i, v := range y {
			if math.IsNaN(v) || math.Abs(v) > 2 {
				t.Fatalf("sample %d is %v", i, v)
			}
		}
		got := Detect(y, sr, DetectOptions{})
		for _, at := range []float64{0.5, 1.0, 2.4} {
			want := tr.At(at*sr) * math.Pow(2, semis/12)
			if g := got.At(at * sr); math.Abs(g/want-1) > 0.02 {
				t.Errorf("shifted by %v: the pitch at %.1f s is %.1f Hz, want %.1f", semis, at, g, want)
			}
		}
		// as loud as it was
		var a, b float64
		for i := range x {
			a += x[i] * x[i]
			b += y[i] * y[i]
		}
		if r := math.Sqrt(b / a); r < 0.8 || r > 1.25 {
			t.Errorf("shifted by %v: the level is %.2f of what it was", semis, r)
		}
	}
}

func TestTargetChant(t *testing.T) {
	const sr = 44100
	x := voice(sr)
	tr := Detect(x, sr, DetectOptions{})
	// the notes of a D major chord: D, F#, A
	o := Options{Mode: Chant, Allowed: func(float64) []int { return []int{2, 6, 9} }, Speed: 0, Keep: 0, Ramp: 1}
	tg := Target(tr, o)
	for _, at := range []float64{0.5, 1.0, 2.4} {
		m := Midi(tg.At(at * sr))
		pc := ((int(math.Round(m)) % 12) + 12) % 12
		if math.Abs(m-math.Round(m)) > 0.05 || (pc != 2 && pc != 6 && pc != 9) {
			t.Errorf("at %.1f s the voice is on %.2f, not a note of the chord", at, m)
		}
	}
	if tg.At(1.5*sr) != 0 {
		t.Error("a pause got a pitch")
	}
}

// what is not moved comes out as it went in
func TestShiftIdentity(t *testing.T) {
	const sr = 44100
	x := voice(sr)
	for i := range x { // a little noise, for the parts without a pitch
		x[i] += 0.01 * math.Sin(float64(i)*1.7+float64(i%13))
	}
	tr := Detect(x, sr, DetectOptions{})
	y := Shift(x, tr, tr)
	var worst, energy float64
	for i := 4000; i < len(x)-4000; i++ {
		worst = max(worst, math.Abs(y[i]-x[i]))
		energy = max(energy, math.Abs(x[i]))
	}
	if worst > 0.02*energy {
		t.Errorf("a voice that is not moved differs by %.4f (the loudest sample is %.2f)", worst, energy)
	}
}
