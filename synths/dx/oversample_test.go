package dx

import (
	"math"
	"math/cmplx"
	"testing"

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
		w := 0.35875 - 0.48829*math.Cos(2*math.Pi*float64(i)/float64(n)) + 0.14128*math.Cos(4*math.Pi*float64(i)/float64(n)) - 0.01168*math.Cos(6*math.Pi*float64(i)/float64(n))
		x[i] = complex(v*w, 0)
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

// playNote renders a patch at a pitch for the stretch the alias is measured over
func playNote(p Patch, pitch float64) []float64 {
	psalm.Seed(1)
	v := New(p, 2)
	v.Note(3, 0.8, psalm.Message{"pitch": pitch, "velocity": 0.8}, psalm.CtrlTick{}, v.Config())
	cfg := v.Config()
	skip := int(0.3 * cfg.SampleRate / float64(cfg.BufferSize))
	var sig []float64

	for b := 0; len(sig) < 1<<15; b++ {
		v.Synthesize()
		if b >= skip {
			for _, s := range v.Output(0).Channels[0] {
				sig = append(sig, float64(s))
			}
		}
	}

	return sig[:1<<15]
}

// the operators played twice and four times a sample leave the sound as it is (the same level, within a dB) and keep the FM that goes over the Nyquist frequency from
// folding back: a hard patch at a high note (C5) has less alias at 2 and less again at 4
func TestOversamplingKeepsTheSoundAndLowersTheAlias(t *testing.T) {
	const pitch = 72.0 // C5
	f := 440 * math.Pow(2, (pitch-69)/12)

	var alias, level [3]float64

	for o := range 3 {
		p := Presets["syncLead"]()
		p.Oversample = o
		sig := playNote(p, pitch)
		alias[o] = aliasLevel(sig, f, 44100)

		for _, s := range sig {
			level[o] += s * s
		}

		level[o] = 10 * math.Log10(level[o]/float64(len(sig)))
	}

	t.Logf("alias %v dB, level %v dB", alias, level)

	for o := 1; o < 3; o++ {
		if math.Abs(level[o]-level[0]) > 1 {
			t.Errorf("oversample %d: the level is %.1f dB against %.1f dB for none", o, level[o], level[0])
		}

		if alias[o] > alias[o-1]-2 {
			t.Errorf("oversample %d: the alias is %.1f dB, want 2 dB below the %.1f dB of the step before", o, alias[o], alias[o-1])
		}
	}
}

// the setting is there, and a patch that has it plays the operators that many times
func TestOversampleSetting(t *testing.T) {
	p := InitVoice()

	if p.OversampleFactor() != 1 {
		t.Errorf("a patch plays its operators %d times by default, want once", p.OversampleFactor())
	}

	found := false

	for _, s := range Settings {
		if s.Name == "oversample" {
			found = true
			*s.get(&p) = 2
		}
	}

	if !found || p.OversampleFactor() != 4 {
		t.Errorf("the setting oversample: found %v, factor %d", found, p.OversampleFactor())
	}
}

// the hard presets are played oversampled, the rest and the noises once
func TestHardPresetsAreOversampled(t *testing.T) {
	for name, over := range oversampled {
		if got := Presets[name]().Oversample; got != over {
			t.Errorf("%s is played at oversample %d, want %d", name, got, over)
		}
	}

	if got := Presets["bell"]().Oversample; got != 0 {
		t.Errorf("bell is played at oversample %d, want none", got)
	}
}
