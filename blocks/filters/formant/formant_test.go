package formant

import (
	"math"
	"testing"
)

const sr = 44100.0

func gainDB(f *Formant, freq float64) float64 {
	f.Clear()
	in, out := 0.0, 0.0
	for i := range int(sr) {
		x := math.Sin(2 * math.Pi * freq * float64(i) / sr)
		y := f.Process(x)
		if i > int(sr)/2 {
			in += x * x
			out += y * y
		}
	}
	return 10 * math.Log10(out/in)
}

func TestVowelFormants(t *testing.T) {
	// "a": formants at 730 and 1090 Hz, a valley in between and far above
	a := New(0, 10, sr)
	f1 := gainDB(a, 730)
	if math.Abs(f1) > 1.0 {
		t.Fatalf("expected 0 dB at the first formant, got %v", f1)
	}
	if valley := gainDB(a, 1600); f1-valley < 10 {
		t.Fatalf("expected a valley between formants, F1 %v valley %v", f1, valley)
	}

	// each vowel emphasizes its own formants: 1090 Hz is a formant of "a" and far from those of
	// "i" (270, 2290, 3010), while 270 Hz is a formant of "i" and far from those of "a"
	i := New(2, 10, sr)
	if gainDB(a, 1090)-gainDB(i, 1090) < 10 {
		t.Fatal("expected vowel a to emphasize 1090 Hz much more than vowel i")
	}
	if gainDB(i, 270)-gainDB(a, 270) < 10 {
		t.Fatal("expected vowel i to emphasize 270 Hz much more than vowel a")
	}
}

func TestMorphIsBetweenVowels(t *testing.T) {
	// halfway between a (F1 730) and e (F1 530) the first formant is near their geometric mean
	f := New(0.5, 10, sr)
	mid := math.Sqrt(730 * 530)
	if gainDB(f, mid) < gainDB(f, 730)+3 {
		t.Fatal("expected the morphed first formant near the geometric mean")
	}
}
