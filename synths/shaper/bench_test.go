package shaper

import (
	"testing"

	"github.com/almerlucke/psalm"
)

// the cost of a voice: a second of a held note of each preset's sound
func BenchmarkVoice(b *testing.B) {
	for _, name := range []string{"glass", "foldPad", "syncLead", "morphPad", "metal", "organ", "vowel", "swarm", "supersawLead", "warmPad", "airPad", "stringPad", "reese", "evolve"} {
		b.Run(name, func(b *testing.B) {
			cfg := Presets[name]()
			cfg.AmpEnv = env(1, 100, 1, 50) // held at full, so every buffer costs the same
			v := New(cfg)
			v.NoteOn(1, psalm.Message{"pitch": 60.0}, psalm.CtrlTick{}, v.Config())
			n := v.Config().BufferSize
			b.ResetTimer()
			for range b.N {
				v.Synthesize()
			}
			b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*n), "ns/sample")
		})
	}
}
