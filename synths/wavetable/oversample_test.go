package wavetable

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

// renderWarp plays a warped wave at a pitch and returns a stretch of it, with its level
func renderWarp(cfg Config, pitch float64) ([]float64, float64) {
	psalm.Seed(1)
	v := New(cfg)
	v.Note(3, 0.8, psalm.Message{"pitch": pitch, "velocity": 0.8}, psalm.CtrlTick{}, v.Config())
	skip := int(0.3 * v.Config().SampleRate / float64(v.Config().BufferSize))
	var sig []float64
	for b := 0; len(sig) < 1<<15; b++ {
		v.Synthesize()
		if b >= skip {
			for _, s := range v.Output(0).Channels[0] {
				sig = append(sig, float64(s))
			}
		}
	}
	sig = sig[:1<<15]
	e := 0.0
	for _, s := range sig {
		e += s * s
	}
	return sig, 10 * math.Log10(e/float64(len(sig)))
}

// a warp bends the phase of the wave and makes corners (the mirror, the bend) and steps (the sync) that the table cannot band limit: played twice and four times a
// sample the alias is lower and the level stays
func TestOversamplingLowersTheAliasOfAWarp(t *testing.T) {
	const pitch = 84.0 // C6
	f := 440 * math.Pow(2, (pitch-69)/12)
	for _, mode := range []WarpMode{WarpMirror, WarpBendUp, WarpSync} {
		var alias, level [3]float64
		for o := range 3 {
			cfg := DefaultConfig()
			cfg.Filter = FilterOff
			cfg.WarpMode, cfg.Warp = mode, 0.7
			cfg.Oversample = o
			var sig []float64
			sig, level[o] = renderWarp(cfg, pitch)
			alias[o] = aliasLevel(sig, f, 44100)
		}
		t.Logf("warp %d: alias %.1f dB, %.1f dB, %.1f dB; level %.1f dB, %.1f dB, %.1f dB", mode, alias[0], alias[1], alias[2], level[0], level[1], level[2])
		for o := 1; o < 3; o++ {
			if alias[o] > alias[0]-1.5*float64(o) {
				t.Errorf("warp %d, oversample %d: the alias is %.1f dB, want %.1f dB under the %.1f dB of none", mode, o, alias[o], 1.5*float64(o), alias[0])
			}
			if math.Abs(level[o]-level[0]) > 1 {
				t.Errorf("warp %d, oversample %d: the level is %.1f dB against %.1f dB for none", mode, o, level[o], level[0])
			}
		}
	}
}

func TestOversampleMessage(t *testing.T) {
	v := New(DefaultConfig())
	if v.overL != nil {
		t.Fatal("a voice plays the oscillators once by default")
	}
	v.ReceiveMessage(psalm.Message{"oversample": 2})
	if v.overL == nil || v.overL.Factor != 4 {
		t.Errorf("oversample 2 should play four times a sample")
	}
	v.ReceiveMessage(psalm.Message{"oversample": 0})
	if v.overL != nil {
		t.Errorf("oversample 0 should play once")
	}
}
