package shaper

import (
	"fmt"
	"math"
	"math/cmplx"
	"sort"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/shapechain"
)

// the presets played in the register they are for: basses, drones and impacts low
var lowNames = map[string]bool{"subSine": true, "kickSub": true, "reese": true, "acid": true, "wobble": true, "pulseBass": true, "fmBass": true,
	"growl": true, "subFold": true, "subDrone": true, "impact": true, "thunder": true, "lionBass": true, "throneDrone": true, "drone5": true, "darkPad": true}

// effects are allowed their edge
var effects = map[string]bool{"laser": true, "zap": true, "lightning": true}

func fftc(a []complex128) []complex128 {
	n := len(a)
	if n == 1 {
		return a
	}
	e, o := make([]complex128, n/2), make([]complex128, n/2)
	for i := range n / 2 {
		e[i], o[i] = a[2*i], a[2*i+1]
	}
	e, o = fftc(e), fftc(o)
	out := make([]complex128, n)
	for k := range n / 2 {
		t := cmplx.Exp(complex(0, -2*math.Pi*float64(k)/float64(n))) * o[k]
		out[k], out[k+n/2] = e[k]+t, e[k]-t
	}
	return out
}

// audit is what is measured of a preset: a note held for 8 seconds, then let go
type audit struct {
	sustained bool
	loud      float64   // dB: the 90th percentile of the rms of 100 ms for a sustained sound, else the peak
	peak      float64   // of the whole note
	high      float64   // dB: the share of the energy above 6 kHz in the loudest half second
	fp        []float64 // a fingerprint: the level over time and the spectrum in octave bands
	ends      bool      // it stopped sounding within 12 seconds of the key being let go
	finite    bool
}

func auditOf(name string) audit {
	cfg := Presets[name]()
	pitch := 60.0
	if lowNames[name] {
		pitch = 40
	}
	psalm.CurrentEnv().Seed(3) // a unison and the drift are random
	v := New(cfg)
	v.NoteOn(1, psalm.Message{"pitch": pitch}, psalm.CtrlTick{}, v.Config())
	x := run(v, 8)
	a := audit{sustained: cfg.AmpEnv.DecayLevel >= 0.3, finite: true, peak: peak(x)}
	for _, s := range x {
		if math.IsNaN(s) || math.IsInf(s, 0) {
			a.finite = false
		}
	}
	win := func(from, to float64) float64 {
		s := 0.0
		seg := x[int(from*sr()):int(to*sr())]
		for _, v := range seg {
			s += v * v
		}
		return 10 * math.Log10(s/float64(len(seg))+1e-12)
	}
	var rms []float64
	w := int(0.1 * sr())
	for i := 0; i+w <= len(x); i += w {
		s := 0.0
		for _, v := range x[i : i+w] {
			s += v * v
		}
		rms = append(rms, 10*math.Log10(s/float64(w)+1e-12))
	}
	sort.Float64s(rms)
	if a.sustained {
		a.loud = rms[len(rms)*9/10]
	} else {
		a.loud = 20 * math.Log10(a.peak)
	}
	// the loudest half second
	best, bi := -1e9, 0
	for i := 0; i+int(0.5*sr()) <= len(x); i += int(0.25 * sr()) {
		if l := win(float64(i)/sr(), float64(i)/sr()+0.5); l > best {
			best, bi = l, i
		}
	}
	c := make([]complex128, 16384)
	for i := range c {
		c[i] = complex(x[bi+i]*(0.5-0.5*math.Cos(2*math.Pi*float64(i)/16383)), 0)
	}
	F := fftc(c)
	a.fp = []float64{win(0, 0.05), win(0.05, 0.3), win(0.3, 1), win(1, 2), win(2, 4)}
	edges := []float64{0, 100, 200, 400, 800, 1600, 3200, 6400, 22050}
	var tot, hi float64
	bands := make([]float64, len(edges)-1)
	for k := 1; k < 8192; k++ {
		f := float64(k) * sr() / 16384
		e := cmplx.Abs(F[k]) * cmplx.Abs(F[k])
		tot += e
		if f > 6000 {
			hi += e
		}
		for b := range bands {
			if f >= edges[b] && f < edges[b+1] {
				bands[b] += e
			}
		}
	}
	a.high = 10 * math.Log10(hi/tot+1e-12)
	top := -1e9
	for _, e := range bands {
		top = math.Max(top, 10*math.Log10(e+1e-9))
	}
	for _, e := range bands { // relative to the loudest band, so the level doesn't count
		a.fp = append(a.fp, math.Max(10*math.Log10(e+1e-9)-top, -60))
	}
	v.NoteOff()
	run(v, 12)
	a.ends = !v.IsActive()
	return a
}

// every preset is checked the same way, and they are checked against each other: each must be
// clean (finite, not clipping), as loud as the others (a sustained sound at -15 dB rms give or take
// 3, a short one peaking around 0.5), not shrill (under -18 dB of its energy above 6 kHz, but for the
// effects), over when the key is let go, and not the same as another preset
func TestPresetsAudit(t *testing.T) {
	if len(Presets) < 40 {
		t.Fatalf("%d presets", len(Presets))
	}
	var names []string
	for n := range Presets {
		names = append(names, n)
	}
	sort.Strings(names)
	audits := map[string]audit{}
	for _, n := range names {
		a := auditOf(n)
		audits[n] = a
		t.Run(n, func(t *testing.T) {
			cfg := Presets[n]()
			for i, text := range cfg.Chains {
				if text == "" {
					continue
				}
				if _, err := shapechain.Parse(text); err != nil {
					t.Errorf("chain %c %q: %v", 'A'+i, text, err)
				}
			}
			if cfg.Chains[0] == "" {
				t.Error("no chain A")
			}
			if !a.finite {
				t.Fatal("not a number in the sound")
			}
			if a.peak < 0.05 || a.peak > 1 {
				t.Errorf("peak %v", a.peak)
			}
			if a.sustained {
				if a.loud < -18 || a.loud > -12 {
					t.Errorf("a sustained sound at %.1f dB rms, want about -15", a.loud)
				}
			} else if a.peak < 0.3 || a.peak > 0.8 {
				t.Errorf("a short sound peaking at %.2f, want about 0.5", a.peak)
			}
			if !effects[n] && a.high > -18 {
				t.Errorf("%.1f dB of the energy above 6 kHz: shrill", a.high)
			}
			if !a.ends {
				t.Error("still sounding 12 s after the key was let go")
			}
		})
	}
	// no two alike
	for i, a := range names {
		for _, b := range names[i+1:] {
			d := 0.0
			for k := range audits[a].fp {
				d += (audits[a].fp[k] - audits[b].fp[k]) * (audits[a].fp[k] - audits[b].fp[k])
			}
			if d = math.Sqrt(d); d < 8 {
				t.Errorf("%s and %s sound alike (a distance of %.1f)", a, b, d)
			}
		}
	}
	if testing.Verbose() {
		for _, n := range names {
			a := audits[n]
			fmt.Printf("%-13s %5.1f dB  peak %.2f  >6k %6.1f dB\n", n, a.loud, a.peak, a.high)
		}
	}
}
