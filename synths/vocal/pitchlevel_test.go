package vocal

import (
	"math"
	"sort"
	"testing"

	"github.com/almerlucke/psalm"
)

// plPeak is the peak of a single note of a preset over 150 buffers
func plPeak(cfg Config, pitch, vel float64) float64 {
	prFresh()
	v := New(cfg)
	v.Note(0.6, 1, psalm.Message{"pitch": pitch, "velocity": vel}, psalm.CtrlTick{}, v.Config())
	p := 0.0
	for range 150 {
		v.Synthesize()
		for _, ch := range v.Output(0).Channels {
			for _, s := range ch {
				x := float64(s)
				if math.IsNaN(x) || math.IsInf(x, 0) {
					return math.NaN()
				}
				p = max(p, math.Abs(x))
			}
		}
	}
	return p
}

// TestLevelAcrossPitch: the level of every preset stays within +9.5 and -12.5 dB of its level at pitch 57 from
// the lowest to the highest pitch, and never gets near clipping
func TestLevelAcrossPitch(t *testing.T) {
	names := make([]string, 0, len(Presets))
	for n := range Presets {
		names = append(names, n)
	}
	sort.Strings(names)
	worst := map[float64]float64{}
	for _, name := range names {
		for _, vel := range []float64{0.3, 1} {
			ref := plPeak(Presets[name](), 57, vel)
			if ref < 1e-3 || math.IsNaN(ref) {
				t.Errorf("%s: level %g at pitch 57", name, ref)
				continue
			}
			for _, pitch := range []float64{36, 48, 57, 72, 84, 93, 96} {
				p := plPeak(Presets[name](), pitch, vel)
				if math.IsNaN(p) {
					t.Errorf("%s: NaN at pitch %g", name, pitch)
					continue
				}
				worst[pitch] = max(worst[pitch], p)
				db := 20 * math.Log10(p/ref)
				if p > 1.5 || db > 9.5 || db < -12.5 {
					t.Errorf("%s vel %g pitch %g: peak %.3f, %+.1f dB against pitch 57 (%.3f)", name, vel, pitch, p, db, ref)
				}
			}
		}
	}
	t.Logf("max peak per pitch: %v", worst)
}

// TestExtremesStayBounded: a big choir, the widest vibrato, a diphthong and a sung word at the highest
// pitches never come near clipping
func TestExtremesStayBounded(t *testing.T) {
	for _, pitch := range []float64{36, 72, 90, 96, 100} {
		cfgs := map[string]Config{}
		c := Choir()
		c.Singers, c.VibratoDepth, c.VibratoDelay = 16, 1, 0
		cfgs["bigChoir"] = c
		c = Solo()
		c.VibratoDepth, c.VibratoRate, c.VibratoDelay, c.NaturalVibrato = 1, 8, 0, 1
		cfgs["vibrato"] = c
		c = Presets["diphthongAi"]()
		cfgs["diphthong"] = c
		c = Presets["spokenWords"]()
		cfgs["word"] = c
		c = Solo()
		c.Nasal, c.Subglottal, c.Whisper = 1, 1, 0.3
		cfgs["nose"] = c
		for name, cfg := range cfgs {
			if p := plPeak(cfg, pitch, 1); math.IsNaN(p) || p > 1.5 {
				t.Errorf("%s at pitch %g: peak %g", name, pitch, p)
			}
		}
	}
}
