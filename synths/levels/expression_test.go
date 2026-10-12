package levels

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/synths/drumsynth"
	"github.com/almerlucke/psalm/synths/dx"
	"github.com/almerlucke/psalm/synths/fm"
	"github.com/almerlucke/psalm/synths/grains"
	"github.com/almerlucke/psalm/synths/modal"
	"github.com/almerlucke/psalm/synths/organ"
	"github.com/almerlucke/psalm/synths/pd"
	"github.com/almerlucke/psalm/synths/pulsar"
	"github.com/almerlucke/psalm/synths/sampler"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/synths/supersaw"
	"github.com/almerlucke/psalm/synths/vocal"
	"github.com/almerlucke/psalm/synths/waveguide"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
	"github.com/almerlucke/psalm/synths/westcoast"
	"github.com/almerlucke/sndfile"
)

// played is a voice (or an instrument) played by a note and its messages
type played interface {
	voice
	ReceiveMessage(psalm.Message)
}

// expressive is a synth to try the bend and the wheel on: how to make it, the note it plays
type expressive struct {
	name string
	make func() played
	note psalm.Message
}

func expressives(t *testing.T) []expressive {
	a4 := psalm.Message{"pitch": 69.0, "velocity": 0.8}
	sine := make([]float32, 88200)
	for i := range sine {
		sine[i] = float32(math.Sin(2 * math.Pi * 440 * float64(i) / 44100))
	}
	region := sampler.NewRegion(sampler.NewSound(44100, sine)).Keys(0, 127, 69)
	region.Loop, region.LoopStart, region.LoopEnd = sampler.LoopContinuous, 0, 88199
	region.CCGain = []sampler.CCGain{{CC: 1, DB: -12}} // the wheel (controller 1) turns it down
	inst := sampler.MustInstrument(region)
	list := []expressive{
		{"subtractive", func() played { return subtractive.New(subtractive.LeadPreset()) }, a4},
		{"fm", func() played { return fm.New(fm.OrganPreset()) }, a4},
		{"dx", func() played { return dx.New(dx.Presets["softBrass"](), 2) }, a4},
		{"wavetable", func() played { return wavetable.New(wavetable.LeadPreset(gen.Brightness(32))) }, a4},
		{"pulsar", func() played { return pulsar.New(pulsar.DefaultConfig()) }, a4},
		{"organ", func() played { return organ.New(organ.Jazz()) }, a4},
		{"waveguide bow", func() played { return waveguide.New(waveguide.Violin()) }, a4},
		{"waveguide pluck", func() played { return waveguide.New(waveguide.Harp()) }, a4},
		{"modal", func() played { return modal.New(modal.Vibraphone()) }, a4},
		{"vocal", func() played { return vocal.New(vocal.Solo()) }, a4},
		{"pd", func() played { return pd.New(pd.Brass()) }, a4},
		{"westcoast", func() played { return westcoast.New(westcoast.Drone()) }, a4},
		{"supersaw", func() played { return supersaw.New(supersaw.Lead()) }, a4},
		{"sampler", func() played { return sampler.New(sampler.DefaultConfig(inst)) }, a4},
		{"drumsynth", func() played {
			kit := drumsynth.Kit808()
			tom := kit.Sounds["tomLow"]
			tom.Decay = 5 // long enough to hear it bend
			kit.Sounds["tomLow"] = tom
			return drumsynth.New(drumsynth.Config{Kit: kit})
		}, psalm.Message{"sound": "tomLow"}},
	}
	// grains of a sine (a recording has a pitch of its own, the bend would be hard to measure)
	path := filepath.Join(t.TempDir(), "sine.wav")
	writeSine(t, path, sine)
	if snd, err := sndfile.NewMipMapSoundFile[float32](path, 4); err == nil {
		list = append(list, expressive{"grains", func() played { return grains.New(grains.ChoirPad(snd, 69)) }, a4})
	} else {
		t.Errorf("the sine for the grains: %v", err)
	}
	return list
}

// writeSine writes a mono 16 bit WAV at 44.1 kHz
func writeSine(t *testing.T, path string, samples []float32) {
	t.Helper()
	data := make([]byte, 44+2*len(samples))
	copy(data, "RIFF")
	binary.LittleEndian.PutUint32(data[4:], uint32(36+2*len(samples)))
	copy(data[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(data[16:], 16)
	binary.LittleEndian.PutUint16(data[20:], 1)
	binary.LittleEndian.PutUint16(data[22:], 1)
	binary.LittleEndian.PutUint32(data[24:], 44100)
	binary.LittleEndian.PutUint32(data[28:], 88200)
	binary.LittleEndian.PutUint16(data[32:], 2)
	binary.LittleEndian.PutUint16(data[34:], 16)
	copy(data[36:], "data")
	binary.LittleEndian.PutUint32(data[40:], uint32(2*len(samples)))
	for i, s := range samples {
		binary.LittleEndian.PutUint16(data[44+2*i:], uint16(int16(math.Round(float64(s)*30000))))
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// left renders a voice for seconds, the left channel
func left(v played, seconds float64) []float64 {
	var x []float64
	for range int(seconds * v.Config().SampleRate / float64(v.Config().BufferSize)) {
		v.Synthesize()
		for _, s := range v.Output(0).Channels[0] {
			x = append(x, float64(s))
		}
	}
	return x
}

// period estimates the period of x in samples (YIN), 0 when it has none
func period(x []float64) float64 {
	const maxLag = 1200
	w := min(len(x)-maxLag-2, 4096)
	if w < 512 {
		return 0
	}
	d := make([]float64, maxLag+2)
	for tau := 1; tau <= maxLag+1; tau++ {
		for i := range w {
			e := x[i] - x[i+tau]
			d[tau] += e * e
		}
	}
	cm := make([]float64, maxLag+2)
	sum := 0.0
	for tau := 1; tau <= maxLag+1; tau++ {
		sum += d[tau]
		cm[tau] = d[tau] * float64(tau) / sum
	}
	best := 2
	for tau := 2; tau <= maxLag; tau++ {
		if cm[tau] < 0.2 {
			for tau+1 <= maxLag && cm[tau+1] < cm[tau] {
				tau++
			}
			best = tau
			a, b, c := cm[tau-1], cm[tau], cm[tau+1]
			return float64(tau) + 0.5*(a-c)/(a-2*b+c)
		}
		if cm[tau] < cm[best] {
			best = tau
		}
	}
	return float64(best)
}

func TestPitchBend(t *testing.T) {
	for _, e := range expressives(t) {
		fresh()
		v := e.make()
		v.Note(10, 1, e.note, psalm.CtrlTick{}, v.Config())
		before := period(left(v, 0.5)[int(0.3*v.Config().SampleRate):])
		v.ReceiveMessage(psalm.Message{"pitchBend": 1.0})
		bent := period(left(v, 0.8)[int(0.4*v.Config().SampleRate):]) // grains take the bend as they start: let the old ones go
		if before == 0 || bent == 0 {
			t.Errorf("%s: no pitch to measure (%g, %g)", e.name, before, bent)
			continue
		}
		if cents := 1200 * math.Log2(before/bent); math.Abs(cents-200) > 25 {
			t.Errorf("%s: a full bend up moves the note %.0f cents, want 200", e.name, cents)
		}
	}
}

func TestModWheel(t *testing.T) {
	for _, e := range expressives(t) {
		// two voices alike (the same seed), one with the wheel up
		render := func(wheel float64) []float64 {
			fresh()
			v := e.make()
			v.ReceiveMessage(psalm.Message{"modWheel": wheel})
			v.Note(10, 1, e.note, psalm.CtrlTick{}, v.Config())
			return left(v, 0.8)
		}
		still, moved := render(0), render(1)
		var diff, all float64
		for i := range still {
			diff += (moved[i] - still[i]) * (moved[i] - still[i])
			all += still[i] * still[i]
		}
		if all == 0 {
			t.Errorf("%s: silent", e.name)
			continue
		}
		if r := math.Sqrt(diff / all); r < 0.1 {
			t.Errorf("%s: the wheel up changes the sound only %.1f%%", e.name, 100*r)
		}
		// and at rest it changes nothing: the same seed renders the same
		again := render(0)
		for i := range still {
			if still[i] != again[i] {
				t.Fatalf("%s: two voices alike render differently: the comparison means nothing", e.name)
			}
		}
	}
}
