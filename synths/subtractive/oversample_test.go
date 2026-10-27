package subtractive

import (
	"math"
	"math/cmplx"
	"testing"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
)

func fftOver(x []complex128) []complex128 {
	n := len(x)
	if n == 1 {
		return x
	}
	even, odd := make([]complex128, n/2), make([]complex128, n/2)
	for i := range n / 2 {
		even[i], odd[i] = x[2*i], x[2*i+1]
	}
	even, odd = fftOver(even), fftOver(odd)
	out := make([]complex128, n)
	for k := range n / 2 {
		t := cmplx.Rect(1, -2*math.Pi*float64(k)/float64(n)) * odd[k]
		out[k], out[k+n/2] = even[k]+t, even[k]-t
	}
	return out
}

// aliasLevel is the energy that is not at a harmonic of f (the closest frequency to it within half a percent) as dB of the energy that is
func aliasLevel(sig []float64, f, sr float64) float64 {
	n := len(sig)
	x := make([]complex128, n)
	for i, v := range sig {
		a := 2 * math.Pi * float64(i) / float64(n)
		x[i] = complex(v*(0.35875-0.48829*math.Cos(a)+0.14128*math.Cos(2*a)-0.01168*math.Cos(3*a)), 0)
	}
	spectrum := fftOver(x)
	bin := sr / float64(n)
	best, first := 0.0, true
	for d := -0.004; d <= 0.004; d += 0.0002 {
		ff := f * (1 + d)
		var harmonic, alias float64
		for k := 1; k < n/2; k++ {
			p := real(spectrum[k] * cmplx.Conj(spectrum[k]))
			if h := float64(k) * bin / ff; math.Abs(h-math.Round(h)) < 3*bin/ff {
				harmonic += p
			} else {
				alias += p
			}
		}
		if db := 10 * math.Log10(alias/harmonic); first || db < best {
			best, first = db, false
		}
	}
	return best
}

// a hard drive on a saw makes harmonics over the Nyquist frequency that the filter (wide open) does not take away: played twice and four times a sample, they do not
// fold back
func TestOversamplingLowersTheAliasOfTheDrive(t *testing.T) {
	const pitch = 84.0
	f := 440 * math.Pow(2, (pitch-69)/12)

	var alias [3]float64
	for o := range 3 {
		psalm.Seed(1)
		cfg := DefaultConfig()
		cfg.Osc1Level, cfg.Osc2Level = 0.5, 0
		cfg.Drive = 3
		cfg.Cutoff, cfg.Resonance, cfg.KeyTrack, cfg.FilterEnvAmount = 18000, 0, 0, 0
		cfg.AmpEnv = *adsr.NewSetting(1, 1, 1, 100, 1, 100)
		cfg.FilterOversample = o
		v := New(cfg)
		v.Note(3, 0.8, psalm.Message{"pitch": pitch, "velocity": 0.8}, psalm.CtrlTick{}, v.Config())
		var sig []float64
		for b := 0; len(sig) < 1<<15+22050; b++ {
			v.Synthesize()
			for _, s := range v.Output(0).Channels[0] {
				sig = append(sig, float64(s))
			}
		}
		alias[o] = aliasLevel(sig[22050:22050+1<<15], f, 44100)
	}

	t.Logf("alias %.1f dB, %.1f dB, %.1f dB", alias[0], alias[1], alias[2])
	// (the floor is the oscillator, about -50 dB at C6)
	for o, want := range []float64{0, 10, 10} {
		if alias[o] > alias[0]-want {
			t.Errorf("oversample %d: %.1f dB of alias, want %.0f dB under the %.1f dB of once", o, alias[o], want, alias[0])
		}
	}
}
