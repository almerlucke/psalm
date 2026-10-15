package pd

import (
	"math"
	"sort"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/fft"
	"github.com/almerlucke/psalm/blocks/modmatrix"
)

// play renders a note: held for hold buffers, released, then tail more buffers; both channels as one
func playNote(cfg Config, pitch, velocity float64, hold, tail int, extra psalm.Message) []float64 {
	psalm.Seed(1)
	psalm.SetCurrentEnv(psalm.NewEnv(psalm.CurrentStackConfig()))
	v := New(cfg)
	msg := psalm.Message{"pitch": pitch}
	for k, x := range extra {
		msg[k] = x
	}
	v.NoteOn(velocity, msg, psalm.CtrlTick{}, v.Config())
	var out []float64
	for b := range hold + tail {
		if b == hold {
			v.NoteOff()
		}
		v.Synthesize()
		ch := v.Output(0).Channels
		for i := range ch[0] {
			out = append(out, (float64(ch[0][i])+float64(ch[len(ch)-1][i]))/2)
		}
	}
	return out
}

func rmsOf(x []float64) float64 {
	s := 0.0
	for _, v := range x {
		s += v * v
	}
	return math.Sqrt(s / float64(max(1, len(x))))
}

// newPresets are the names of the presets of presets_more.go
var newPresets = []string{
	"dcoBass", "subBass", "squareBass", "acidBass", "slapBass", "pluckBass", "growlBass",
	"resoLead", "squareLead", "pulseLead", "fluteLead", "whistleLead",
	"horns", "brassStab", "ensemble", "cello",
	"vibes", "harpsichord", "xylo", "toyPiano", "pipeOrgan",
	"glockenspiel", "chime", "musicBox",
	"darkPad", "warmPad", "airPad", "ahChoir", "ohChoir",
	"laserDown", "siren", "sonarPing", "riseSweep",
	"dcoKick", "dcoTom", "ringHat", "clack",
}

func TestNewPresetsRegistered(t *testing.T) {
	for _, n := range newPresets {
		if _, ok := Presets[n]; !ok {
			t.Errorf("%s is not registered", n)
		}
	}
}

func TestNewPresetsRender(t *testing.T) {
	for _, n := range newPresets {
		f := Presets[n]
		for _, pitch := range []float64{36, 57, 81} {
			for _, vel := range []float64{0.3, 1} {
				x := playNote(f(), pitch, vel, 30, 25, nil)
				peak := 0.0
				for i, v := range x {
					if math.IsNaN(v) || math.IsInf(v, 0) {
						t.Fatalf("%s pitch %g velocity %g: not finite at %d", n, pitch, vel, i)
					}
					if i < 30*1024 {
						peak = math.Max(peak, math.Abs(v))
					}
				}
				if peak < 1e-3 {
					t.Errorf("%s pitch %g velocity %g: silent (peak %g)", n, pitch, vel, peak)
				}
				if peak > 2 {
					t.Errorf("%s pitch %g velocity %g: peak %g", n, pitch, vel, peak)
				}
			}
		}
	}
}

// fingerprint is the spectral shape of a note: 24 log-spaced band levels (dB, relative to the
// loudest), the attack time (seconds to 90% of the peak of the envelope) and the centroid (Hz)
type fingerprint struct {
	bands    [24]float64
	attack   float64
	centroid float64
}

func fingerprintOf(x []float64) fingerprint {
	var fp fingerprint
	const blk = 44
	peak := 0.0
	env := make([]float64, len(x)/blk)
	for i := range env {
		env[i] = rmsOf(x[i*blk : (i+1)*blk])
		peak = math.Max(peak, env[i])
	}
	for i, e := range env {
		if e >= 0.9*peak {
			fp.attack = float64(i) * blk / 44100
			break
		}
	}
	const n = 8192
	f := fft.New(n)
	w := fft.HannWindow(n)
	re, im := make([]float64, n), make([]float64, n)
	power := make([]float64, n/2)
	for start := 0; start+n <= len(x); start += n / 2 {
		for i := range n {
			re[i], im[i] = x[start+i]*w[i], 0
		}
		f.Forward(re, im)
		for k := range power {
			power[k] += re[k]*re[k] + im[k]*im[k]
		}
	}
	lo, hi := 50.0, 16000.0
	var sum, wsum float64
	var band [24]float64
	for k := 1; k < len(power); k++ {
		fr := float64(k) * 44100 / n
		sum += power[k]
		wsum += power[k] * fr
		if fr < lo || fr >= hi {
			continue
		}
		b := int(24 * math.Log(fr/lo) / math.Log(hi/lo))
		band[min(b, 23)] += power[k]
	}
	fp.centroid = wsum / math.Max(sum, 1e-30)
	top := 0.0
	for _, v := range band {
		top = math.Max(top, v)
	}
	for i, v := range band {
		fp.bands[i] = 10 * math.Log10(math.Max(v/top, 1e-9))
	}
	return fp
}

// distance is how far apart two sounds are: the mean difference of the bands in dB, plus the
// difference of the attack (in octaves, from 5 ms) and of the centroid (in octaves), weighted
func distance(a, b fingerprint) float64 {
	d := 0.0
	for i := range a.bands {
		d += math.Abs(a.bands[i]-b.bands[i]) / 24
	}
	d += 2 * math.Abs(math.Log2((a.attack+0.005)/(b.attack+0.005)))
	d += 4 * math.Abs(math.Log2(a.centroid/b.centroid))
	return d
}

// minDistance is the least two presets may be apart, see distance
const minDistance = 4.0

func TestPresetsDistinct(t *testing.T) {
	names := make([]string, 0, len(Presets))
	for n := range Presets {
		names = append(names, n)
	}
	sort.Strings(names)
	buffers := int(1.5 * 44100 / float64(psalm.CurrentStackConfig().BufferSize))
	fps := map[string]fingerprint{}
	for _, n := range names {
		fps[n] = fingerprintOf(playNote(Presets[n](), 57, 0.8, buffers, 0, nil))
	}
	isNew := map[string]bool{}
	for _, n := range newPresets {
		isNew[n] = true
	}
	for i, a := range names {
		for _, b := range names[i+1:] {
			if !isNew[a] && !isNew[b] {
				continue
			}
			if d := distance(fps[a], fps[b]); d < minDistance {
				t.Errorf("%s and %s are too alike: distance %.2f (< %g)", a, b, d, minDistance)
			}
		}
	}
}

func TestNewPresetsMatrixResponds(t *testing.T) {
	used := 0
	for _, n := range newPresets {
		cfg := Presets[n]()
		for i, sl := range cfg.Mods {
			if sl.Dest == "" {
				continue
			}
			used++
			extra := psalm.Message{}
			if sl.Src == modmatrix.Wheel { // the wheel also distorts and opens the filter by itself
				extra["modWheel"] = 0.8
			}
			base := playNote(cfg, 57, 0.8, 110, 0, extra)
			without := cfg
			without.Mods[i].Amount = 0
			x := playNote(without, 57, 0.8, 110, 0, extra)
			// the largest change in a window of 20 buffers, to the level of the window
			best := 0.0
			for w := 0; w+20 <= 110; w += 10 {
				var diff []float64
				for j := w * 1024; j < (w+20)*1024; j++ {
					diff = append(diff, x[j]-base[j])
				}
				if r := rmsOf(base[w*1024 : (w+20)*1024]); r > 1e-3 {
					best = math.Max(best, rmsOf(diff)/r)
				}
			}
			if best < 0.01 {
				t.Errorf("%s: slot %d (%v to %s) changes the sound by only %.3f", n, i+1, sl.Src, sl.Dest, best)
			}
		}
	}
	if used < 12 {
		t.Errorf("only %d slots in use", used)
	}
}
