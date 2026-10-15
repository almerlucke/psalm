package piano

import (
	"math"
	"math/cmplx"
	"testing"

	"github.com/almerlucke/psalm"
)

// radix2 is a plain radix 2 transform (the length is a power of two)
func radix2(x []complex128) {
	n := len(x)
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			x[i], x[j] = x[j], x[i]
		}
	}
	for size := 2; size <= n; size <<= 1 {
		w := cmplx.Exp(complex(0, -2*math.Pi/float64(size)))
		for start := 0; start < n; start += size {
			f := complex(1, 0)
			for k := range size / 2 {
				a, b := x[start+k], x[start+k+size/2]*f
				x[start+k], x[start+k+size/2] = a+b, a-b
				f *= w
			}
		}
	}
}

// spectrum returns the magnitudes of the Hann windowed segment (zero padded to size), bin width sr/size
func spectrum(x []float64, size int) []float64 {
	buf := make([]complex128, size)
	for i, v := range x[:min(len(x), size)] {
		buf[i] = complex(v*(0.5-0.5*math.Cos(2*math.Pi*float64(i)/float64(len(x)))), 0)
	}
	radix2(buf)
	out := make([]float64, size/2)
	for i := range out {
		out[i] = cmplx.Abs(buf[i])
	}
	return out
}

// centroid returns the spectral centroid (Hz) of the first seconds of a signal
func centroid(x []float64, seconds float64) float64 {
	n := int(seconds * sr)
	s := spectrum(x[:n], 32768)
	num, den := 0.0, 0.0
	for i, m := range s {
		f := float64(i) * sr / 32768
		num += f * m * m
		den += m * m
	}
	return num / den
}

// A harder strike is brighter: the centroid of the attack rises with the velocity, in the bass, the
// middle and the treble
func TestVelocityCentroid(t *testing.T) {
	for _, p := range []float64{36, 60, 84} {
		prev := 0.0
		for _, vel := range []float64{0.1, 0.3, 0.5, 0.7, 0.9, 1} {
			x, _ := play(grand(), psalm.Message{"pitch": p, "velocity": vel}, 1, 0.4)
			c := centroid(x, 0.3)
			if c <= prev {
				t.Errorf("key %v: the centroid should rise with the velocity: %.0f Hz at %v after %.0f Hz", p, c, vel, prev)
			}
			prev = c
		}
	}
}

// The partials of mid and bass notes lie at f0 n sqrt(1 + B n²) / sqrt(1 + B)
func TestPartialsInharmonic(t *testing.T) {
	for _, p := range []float64{36, 45, 57, 69} {
		x, _ := play(single(), psalm.Message{"pitch": p, "velocity": 0.6}, 3, 1.6)
		seg := x[int(0.3*sr):int(1.5*sr)]
		cfg := single()
		f0 := 440 * math.Exp2((stretched(p, &cfg)-69)/12)
		b := inharmonicityAt(p, cfg.Wound)
		f1 := peakFrequency(seg, f0, 0.01)
		for n := 2.0; n <= 8; n++ {
			want := n * f1 * math.Sqrt((1+b*n*n)/(1+b))
			got := peakFrequency(seg, want, 0.006)
			if c := 1200 * math.Log2(got/want); math.Abs(c) > 3.5 {
				t.Errorf("key %v partial %v is %.1f cents from %.1f Hz", p, n, c, want)
			}
		}
	}
}

// A treble note with three strings falls fast and then lingers (the strings move together, then
// against each other)
func TestTrebleDoubleDecay(t *testing.T) {
	x := playPiano(grand(), []psalm.Message{{"pitch": 76.0}}, 6, 5, nil)
	early := (rmsDB(x, 0.1, 0.4) - rmsDB(x, 0.7, 1.0)) / 0.6
	late := (rmsDB(x, 1.5, 2.0) - rmsDB(x, 3.5, 4.0)) / 2
	if early < 2*late || late < 0 {
		t.Fatalf("a treble note should fall fast first (%.1f dB/s) and then linger (%.1f dB/s)", early, late)
	}
}

// The soft pedal (una corda) moves the hammer: the note is quieter, darker, and the string it
// missed is nearly silent, so the three strings of a treble key sound as two
func TestUnaCorda(t *testing.T) {
	for _, p := range []float64{48, 72} {
		normal, _ := play(grand(), psalm.Message{"pitch": p}, 2, 1)
		soft, _ := play(grand(), psalm.Message{"pitch": p, "soft": 1.0}, 2, 1)
		if c, s := centroid(normal, 0.5), centroid(soft, 0.5); s > c*0.97 {
			t.Errorf("key %v: una corda should be darker: %.0f Hz against %.0f Hz", p, s, c)
		}
		if d := rmsDB(normal, 0.05, 0.5) - rmsDB(soft, 0.05, 0.5); d < 1.5 || d > 9 {
			t.Errorf("key %v: una corda should be a few dB quieter, %.1f dB", p, d)
		}
		if d := rmsDB(difference(normal, soft), 0.05, 0.5) - rmsDB(normal, 0.05, 0.5); d < -3 {
			t.Errorf("key %v: una corda should change the sound, %.1f dB", p, d)
		}
	}
}

// The bass strings are stretched by their own motion and ring longitudinally: the phantom partials
// are at the sums of the frequencies of the partials
func TestLongitudinalSums(t *testing.T) {
	cfg := single()
	cfg.Bark = 1
	none := cfg
	none.Bark = 0
	m := []psalm.Message{{"pitch": 36.0, "velocity": 1.0}}
	a := seeded(cfg, m, 2, 1.2, nil)
	b := seeded(none, m, 2, 1.2, nil)
	d := difference(a, b)[int(0.1*sr):int(1.1*sr)]
	plain := b[int(0.1*sr):int(1.1*sr)]
	f0 := 440 * math.Exp2((stretched(36, &cfg)-69)/12)
	bb := inharmonicityAt(36, cfg.Wound)
	pn := func(n float64) float64 { return f0 * n * math.Sqrt((1+bb*n*n)/(1+bb)) }
	level := func(x []float64, f float64) float64 {
		s := spectrum(x, 65536)
		i := int(math.Round(f * 65536 / sr))
		best := 0.0
		for k := i - 2; k <= i+2; k++ {
			best = math.Max(best, s[k])
		}
		return best
	}
	// the sums of partial pairs the string has (its low partials are loud)
	for _, pair := range [][2]float64{{3, 4}, {2, 6}, {4, 5}, {3, 7}} {
		f := pn(pair[0]) + pn(pair[1])
		peak := level(d, f)
		off := level(d, f+0.5*f0) // between the sums
		if 20*math.Log10(peak/off) < 6 {
			t.Errorf("no longitudinal energy at the sum of partials %v (%.0f Hz): %.1f dB over its surroundings", pair, f, 20*math.Log10(peak/off))
		}
	}
	// and it is energy of its own, not the string
	if r := rmsDB(d, 0, 0.5) - rmsDB(plain, 0, 0.5); r < -30 {
		t.Errorf("the bark should be heard at the top velocity, %.1f dB", r)
	}
}

// The dampers landing on the string when the key is released are heard; with the sustain pedal
// down they are not on the string, only the (much softer) key comes up
func TestDamperThump(t *testing.T) {
	quiet := grand()
	quiet.Mechanics = 0
	thump := func(setup psalm.Message) float64 {
		a := seeded(grand(), []psalm.Message{{"pitch": 55.0}}, 0.5, 0.8, setup)
		b := seeded(quiet, []psalm.Message{{"pitch": 55.0}}, 0.5, 0.8, setup)
		n := difference(a, b)
		return rmsDB(n, 0.5, 0.6)
	}
	up, down := thump(nil), thump(psalm.Message{"sustain": 1.0})
	// (the pedal itself is heard going down before the note: look at the release only)
	if up < down+10 {
		t.Errorf("the dampers should thump on the release (%.1f dB) and not with the pedal down (%.1f dB)", up, down)
	}
}

// A key struck hard lands on the keybed: a low thud that a soft strike doesn't make
func TestKeyBed(t *testing.T) {
	none := grand()
	none.KeyBed = 0
	thud := func(vel float64) float64 {
		a := seeded(grand(), []psalm.Message{{"pitch": 60.0, "velocity": vel}}, 2, 0.3, nil)
		b := seeded(none, []psalm.Message{{"pitch": 60.0, "velocity": vel}}, 2, 0.3, nil)
		return rmsDB(difference(a, b), 0.0, 0.1) - rmsDB(b, 0, 0.1)
	}
	soft, hard := thud(0.3), thud(1)
	t.Logf("key bed against the note: %.1f dB soft, %.1f dB hard", soft, hard)
	if hard < -45 || hard > -22 || hard < soft+10 {
		t.Errorf("the key bed should thud when struck hard (%.1f dB against the note) and hardly when soft (%.1f dB)", hard, soft)
	}
}

// No NaN, no explosion on any key at any velocity
func TestNoNaN(t *testing.T) {
	for key := 21; key <= 108; key++ {
		for _, vel := range []float64{0, 0.2, 0.5, 0.8, 1} {
			v := New(GrandPreset())
			v.Note(0.1, 1, psalm.Message{"pitch": float64(key), "velocity": vel}, psalm.CtrlTick{}, v.Config())
			for range 8 {
				v.Synthesize()
				for _, ch := range v.Output(0).Channels {
					for _, x := range ch {
						if math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) || math.Abs(float64(x)) > 50 {
							t.Fatalf("key %d velocity %v: %v", key, vel, x)
						}
					}
				}
			}
		}
	}
}

// seeded plays notes on a piano with the random streams at the same start, so renders that differ in
// one setting can be subtracted without their noise
func seeded(cfg Config, notes []psalm.Message, duration, seconds float64, setup psalm.Message) []float64 {
	psalm.Seed(3)
	psalm.SetCurrentEnv(psalm.NewEnv(psalm.CurrentStackConfig()))
	return playPiano(cfg, notes, duration, seconds, setup)
}
