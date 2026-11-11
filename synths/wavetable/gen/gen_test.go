package gen

import (
	"math"
	"path/filepath"
	"testing"

	"github.com/almerlucke/psalm/blocks/fft"
	"github.com/almerlucke/psalm/synths/wavetable"
)

// harmonics returns the amplitudes of the harmonics (index 0 is DC) of the frame at position
func harmonics(t *wavetable.Table, position float64) []float64 {
	re := make([]float64, FrameSize)
	im := make([]float64, FrameSize)
	for j := range FrameSize {
		re[j] = t.Sample(float64(j)/FrameSize, position, 1e-9) // a tiny increment reads all harmonics
	}
	fft.New(FrameSize).Forward(re, im)
	amps := make([]float64, FrameSize/2)
	for k := range amps {
		amps[k] = 2 * math.Hypot(re[k], im[k]) / FrameSize
	}
	return amps
}

// centroid in harmonics
func centroid(amps []float64) float64 {
	num, den := 0.0, 0.0
	for k := 1; k < len(amps); k++ {
		num += float64(k) * amps[k] * amps[k]
		den += amps[k] * amps[k]
	}
	return num / den
}

func evenToOdd(amps []float64) float64 {
	even, odd := 0.0, 0.0
	for k := 1; k < len(amps); k++ {
		if k%2 == 0 {
			even += amps[k] * amps[k]
		} else {
			odd += amps[k] * amps[k]
		}
	}
	return even / odd
}

// brighter checks that the centroid rises over the table
func brighter(t *testing.T, name string, table *wavetable.Table, factor float64) {
	t.Helper()
	lo, hi := centroid(harmonics(table, 0)), centroid(harmonics(table, 1))
	if hi < lo*factor {
		t.Errorf("%s: expected the table to get brighter, centroid %.1f to %.1f harmonics", name, lo, hi)
	}
}

func TestBrightness(t *testing.T) {
	table := Brightness(32)
	if c := centroid(harmonics(table, 0)); c > 1.2 {
		t.Fatalf("expected a sine at the start, centroid %v", c)
	}
	// the end is close to a sawtooth: harmonic k at about 1/k
	amps := harmonics(table, 1)
	for _, k := range []int{2, 10, 100} {
		if r := amps[k] * float64(k) / amps[1]; r < 0.85 || r > 1.01 {
			t.Errorf("harmonic %d at %.2f of a sawtooth", k, r)
		}
	}
}

func TestPWM(t *testing.T) {
	table := PWM(32, 0.05)
	if r := evenToOdd(harmonics(table, 0)); r > 1e-6 {
		t.Fatalf("a square has no even harmonics, got even/odd %v", r)
	}
	if r := evenToOdd(harmonics(table, 1)); r < 0.5 {
		t.Fatalf("a thin pulse has even harmonics, got even/odd %v", r)
	}
	brighter(t, "pwm", table, 2)
}

func TestSync(t *testing.T) {
	table := Sync(32, 8)
	amps := harmonics(table, 0) // a plain sawtooth: harmonic k at 1/k
	if r := amps[2] / amps[1]; math.Abs(r-0.5) > 0.01 {
		t.Fatalf("expected a sawtooth at ratio 1, harmonic 2/1 is %v", r)
	}
	brighter(t, "sync", table, 3)
}

func TestFM(t *testing.T) {
	table := FM(32, 3, 6)
	amps := harmonics(table, 0)
	if amps[2]/amps[1] > 1e-4 {
		t.Fatal("expected a sine at index 0")
	}
	brighter(t, "fm", table, 3)
}

func TestFold(t *testing.T) {
	table := Fold(32, 6, 0)
	if r := evenToOdd(harmonics(table, 1)); r > 1e-6 {
		t.Fatalf("a symmetric fold has no even harmonics, got even/odd %v", r)
	}
	brighter(t, "fold", table, 3)
	if r := evenToOdd(harmonics(Fold(8, 6, 0.3), 1)); r < 0.05 {
		t.Fatalf("a bias adds even harmonics, got even/odd %v", r)
	}
}

// The strongest harmonic is near the first formant: around harmonic 5-6 for "a" (730 Hz at
// 130 Hz), harmonic 2 for "i" (270 Hz)
func TestVowels(t *testing.T) {
	table := Vowels(64, 130)
	peak := func(position float64) int {
		amps := harmonics(table, position)
		best := 1
		for k := range amps {
			if amps[k] > amps[best] {
				best = k
			}
		}
		return best
	}
	if k := peak(0); k < 5 || k > 6 {
		t.Errorf("a: strongest harmonic %d", k)
	}
	if k := peak(0.5); k != 2 {
		t.Errorf("i: strongest harmonic %d", k)
	}
}

func TestDrawbars(t *testing.T) {
	table := Drawbars(16, 12)
	count := func(position float64) int {
		n := 0
		for _, a := range harmonics(table, position)[1:] {
			if a > 0.01 {
				n++
			}
		}
		return n
	}
	if n := count(0); n != 1 {
		t.Fatalf("expected only the fundamental at the start, %d harmonics", n)
	}
	if n := count(1); n != 12 {
		t.Fatalf("expected 12 harmonics at the end, %d", n)
	}
}

func TestRandom(t *testing.T) {
	a, b, c := Random(16, 4, 32, 1), Random(16, 4, 32, 1), Random(16, 4, 32, 2)
	same, diff := 0.0, 0.0
	for _, pos := range []float64{0, 0.3, 1} {
		ha, hb, hc := harmonics(a, pos), harmonics(b, pos), harmonics(c, pos)
		for k := range ha {
			same += math.Abs(ha[k] - hb[k])
			diff += math.Abs(ha[k] - hc[k])
		}
	}
	if same > 1e-9 || diff < 0.1 {
		t.Fatalf("expected the same table for the same seed and another for another seed: %v, %v", same, diff)
	}
}

func TestSaveAndLoad(t *testing.T) {
	table := Fold(8, 5, 0.2)
	path := filepath.Join(t.TempDir(), "fold.wav")
	if err := table.Save(path); err != nil {
		t.Fatal(err)
	}
	loaded, err := wavetable.Load(path, FrameSize)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.NumFrames() != table.NumFrames() {
		t.Fatalf("saved %d frames, loaded %d", table.NumFrames(), loaded.NumFrames())
	}
	for _, pos := range []float64{0, 0.5, 1} {
		for j := range FrameSize {
			phase := float64(j) / FrameSize
			if d := math.Abs(table.Sample(phase, pos, 1e-9) - loaded.Sample(phase, pos, 1e-9)); d > 1e-4 {
				t.Fatalf("position %v sample %d differs by %v", pos, j, d)
			}
		}
	}
}

// Drawn shapes with jumps, played high, stay alias free: the oversampled drawing leaves nothing to
// fold back
func TestDrawnTablesDontAlias(t *testing.T) {
	const sr = 44100.0
	for name, table := range map[string]*wavetable.Table{"pwm": PWM(8, 0.03), "sync": Sync(8, 8), "fold": Fold(8, 8, 0.2)} {
		for _, f0 := range []float64{110, 2900} {
			x := make([]float64, 1<<16)
			phase := 0.0
			for i := range x {
				x[i] = table.Sample(phase, 1, f0/sr)
				phase += f0 / sr
				phase -= math.Floor(phase)
			}
			const n = 16384
			f := fft.New(n)
			w := fft.HannWindow(n)
			harm, between := 0.0, 0.0
			for start := 0; start+n <= len(x); start += n / 2 {
				re := make([]float64, n)
				im := make([]float64, n)
				for i := range n {
					re[i] = x[start+i] * w[i]
				}
				f.Forward(re, im)
				for k := 1; k < n/2; k++ {
					fr := float64(k) * sr / n
					p := re[k]*re[k] + im[k]*im[k]
					if d := math.Abs(fr - f0*math.Round(fr/f0)); d < 8 {
						harm += p
					} else if d > 30 {
						between += p
					}
				}
			}
			if db := 10 * math.Log10(between/harm); db > -60 {
				t.Errorf("%s at %v Hz: aliasing at %.1f dB", name, f0, db)
			}
		}
	}
}

func strongest(amps []float64, from int) int {
	best := from
	for k := from; k < len(amps); k++ {
		if amps[k] > amps[best] {
			best = k
		}
	}
	return best
}

func TestFilter(t *testing.T) {
	table := Filter(64, 8)
	brighter(t, "filter", table, 10)
	// halfway the cutoff is at harmonic 128^0.5 = 11.3, the resonance peak makes it the strongest
	// overtone (the fundamental stays louder: the sawtooth falls by 1/k)
	if k := strongest(harmonics(table, 0.5), 2); k < 10 || k > 12 {
		t.Fatalf("expected the resonance peak near harmonic 11, strongest overtone is %d", k)
	}
}

func TestResonance(t *testing.T) {
	table := Resonance(64, 12)
	brighter(t, "resonance", table, 2)
	if k := strongest(harmonics(table, 1), 2); k < 10 || k > 14 {
		t.Fatalf("expected the resonance near harmonic 12, strongest above the fundamental is %d", k)
	}
}

func TestOddEven(t *testing.T) {
	table := OddEven(32)
	if r := evenToOdd(harmonics(table, 0)); r > 1e-6 {
		t.Fatalf("expected only odd harmonics at the start, even/odd %v", r)
	}
	amps := harmonics(table, 1)
	odd := 0.0
	for k := 3; k < len(amps); k += 2 {
		odd += amps[k] * amps[k]
	}
	if odd > 1e-9 {
		t.Fatalf("expected only the fundamental and even harmonics at the end, odd energy %v", odd)
	}
}

func TestComb(t *testing.T) {
	table := Comb(32, 0.5)
	amps := harmonics(table, 1) // half a cycle delay: the odd harmonics cancel
	odd, even := 0.0, 0.0
	for k := 1; k < len(amps); k++ {
		if k%2 == 1 {
			odd += amps[k] * amps[k]
		} else {
			even += amps[k] * amps[k]
		}
	}
	if odd > even*1e-6 {
		t.Fatalf("expected the odd harmonics to cancel: odd %v, even %v", odd, even)
	}
}

func TestSaturate(t *testing.T) {
	table := Saturate(32, 10)
	if r := harmonics(table, 0); r[3]/r[1] > 0.1 {
		t.Fatalf("expected about a sine at drive 1, harmonic 3/1 %v", r[3]/r[1])
	}
	if r := evenToOdd(harmonics(table, 1)); r > 1e-6 {
		t.Fatalf("a symmetric clipper adds no even harmonics, even/odd %v", r)
	}
	if r := harmonics(table, 1); r[3]/r[1] < 0.2 {
		t.Fatalf("expected a strong third harmonic at full drive, harmonic 3/1 %v", r[3]/r[1])
	}
}

func TestQuantize(t *testing.T) {
	table := Quantize(32, 3)
	if r := harmonics(table, 0); r[3]/r[1] > 0.01 {
		t.Fatalf("expected about a sine at 8 bits, harmonic 3/1 %v", r[3]/r[1])
	}
	brighter(t, "quantize", table, 2)
}

func TestSteps(t *testing.T) {
	table := Steps(32, 8)
	// 8 steps mirror the fundamental around harmonic 8: the strongest overtones are 7 and 9
	if k := strongest(harmonics(table, 1), 2); k != 7 && k != 9 {
		t.Fatalf("expected the mirror images at harmonics 7 and 9, strongest overtone %d", k)
	}
}

func TestCrossfade(t *testing.T) {
	table := Crossfade(9, Sine, Triangle, Saw, Square)
	if r := harmonics(table, 0); r[2]/r[1] > 1e-4 || r[3]/r[1] > 1e-4 {
		t.Fatal("expected a sine at the start")
	}
	end := harmonics(table, 1)
	if r := evenToOdd(end); r > 1e-6 || math.Abs(end[3]/end[1]-1.0/3) > 0.01 {
		t.Fatalf("expected a square at the end: even/odd %v, harmonic 3/1 %v", r, end[3]/end[1])
	}
}

func TestFromSound(t *testing.T) {
	// a sine with exactly 8 cycles per frame plays as the 8th harmonic
	sr := 44100.0
	f := 8 * sr / FrameSize
	samples := make([]float32, 3*44100)
	for i := range samples {
		samples[i] = float32(math.Sin(2 * math.Pi * f * float64(i) / sr))
	}
	table := FromSound(samples, 16)
	if table.NumFrames() != 16 {
		t.Fatalf("%d frames", table.NumFrames())
	}
	if k := strongest(harmonics(table, 0.5), 1); k != 8 {
		t.Fatalf("expected harmonic 8, strongest is %d", k)
	}

	// any other frequency doesn't fit the frame, the loop crossfade keeps the wrap smooth
	for i := range samples {
		samples[i] = float32(math.Sin(2 * math.Pi * 441.3 * float64(i) / sr))
	}
	table = FromSound(samples, 4)
	maxStep := 0.0
	prev := table.Sample(1-1.0/FrameSize, 0.5, 1e-9)
	for j := range FrameSize {
		x := table.Sample(float64(j)/FrameSize, 0.5, 1e-9)
		maxStep = max(maxStep, math.Abs(x-prev))
		prev = x
	}
	if smooth := 2 * math.Pi * 441.3 / sr; maxStep > 3*smooth {
		t.Fatalf("expected a smooth loop, largest step %v (a sine steps %v)", maxStep, smooth)
	}
}

// Silence at the ends and in pauses of a recording is skipped, no frame is silent
func TestFromSoundSkipsSilence(t *testing.T) {
	sr := 44100.0
	samples := make([]float32, 5*44100)
	for i := range samples {
		sec := float64(i) / sr
		if (sec > 1 && sec < 2) || (sec > 3 && sec < 4) { // sound, a pause, sound
			samples[i] = float32(0.5 * math.Sin(2*math.Pi*220*sec))
		}
	}
	table := FromSound(samples, 32)
	for f := range 32 {
		pos := float64(f) / 31
		sum := 0.0
		for j := range FrameSize {
			x := table.Sample(float64(j)/FrameSize, pos, 1e-9)
			sum += x * x
		}
		if r := math.Sqrt(sum / FrameSize); r < 0.1 {
			t.Fatalf("frame %d is (nearly) silent, rms %v", f, r)
		}
	}
}

func TestAdditive(t *testing.T) {
	want := []float64{1, 0, 0.5, 0, 0.25, 0.125}
	table := Additive(want, 4)
	if table.NumFrames() != 4 {
		t.Fatalf("%d frames", table.NumFrames())
	}
	for _, pos := range []float64{0, 0.5, 1} {
		amps := harmonics(table, pos)
		for k, w := range want {
			if got := amps[k+1] / amps[1]; math.Abs(got-w/want[0]) > 0.01 {
				t.Errorf("position %g harmonic %d: %g, want %g", pos, k+1, got, w)
			}
		}
		for k := len(want) + 1; k < len(amps); k++ {
			if amps[k] > 1e-3*amps[1] {
				t.Fatalf("harmonic %d sounds: %g", k, amps[k])
			}
		}
	}
	if Additive(nil, 0).NumFrames() != 1 {
		t.Error("an empty list is one frame")
	}
}

func TestAdditiveMorph(t *testing.T) {
	table := AdditiveMorph([]float64{1}, []float64{0, 1, 1}, 9)
	lo, mid, hi := harmonics(table, 0), harmonics(table, 0.5), harmonics(table, 1)
	if lo[2] > 1e-3 || hi[1] > 1e-3 {
		t.Errorf("ends: %g %g", lo[2], hi[1])
	}
	// halfway through harmonic 1 is at half and 2 and 3 at half of the end
	if r := mid[2] / mid[1]; math.Abs(r-1) > 0.05 {
		t.Errorf("middle: harmonics 1 and 2 %g", r)
	}
	if r := mid[1] / lo[1]; math.Abs(r-0.5) > 0.1 {
		t.Errorf("fundamental in the middle %g of the start", r)
	}
	for _, a := range [][]float64{lo, mid, hi} {
		for _, v := range a {
			if math.IsNaN(v) {
				t.Fatal("NaN")
			}
		}
	}
}
