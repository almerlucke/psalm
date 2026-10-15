package levels

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/meter"
	"github.com/almerlucke/psalm/synths/dx"
	"github.com/almerlucke/psalm/synths/grains"
	"github.com/almerlucke/psalm/synths/modal"
	"github.com/almerlucke/psalm/synths/organ"
	"github.com/almerlucke/psalm/synths/pd"
	"github.com/almerlucke/psalm/synths/piano"
	"github.com/almerlucke/psalm/synths/pulsar"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/synths/supersaw"
	"github.com/almerlucke/psalm/synths/vocal"
	"github.com/almerlucke/psalm/synths/waveguide"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
	"github.com/almerlucke/psalm/synths/westcoast"
	"github.com/almerlucke/sndfile"
)

const (
	target  = -12.0 // LUFS
	ceiling = -1.0  // dBFS
)

type voice interface {
	Note(float64, float64, psalm.Message, psalm.CtrlTick, psalm.Config)
	Synthesize()
	Output(int) *psalm.Socket
	Config() psalm.Config
}

// fresh gives every measurement its own seeded environment, so random textures measure the same
func fresh() {
	psalm.Seed(1)
	psalm.SetCurrentEnv(psalm.NewEnv(psalm.CurrentStackConfig()))
}

// measure plays a 2 second note at amplitude 1 and returns the loudness and peak of 3 seconds
func measure(v voice, msg psalm.Message) (float64, float64) {
	v.Note(2, 1, msg, psalm.CtrlTick{}, v.Config())
	l := meter.NewLoudness(v.Config().SampleRate)
	peak := 0.0
	for range int(3 * v.Config().SampleRate / float64(v.Config().BufferSize)) {
		v.Synthesize()
		ch := v.Output(0).Channels
		for i := range ch[0] {
			a, b := float64(ch[0][i]), float64(ch[len(ch)-1][i])
			l.Add(a, b)
			peak = math.Max(peak, math.Max(math.Abs(a), math.Abs(b)))
		}
	}
	return l.Integrated(), 20 * math.Log10(peak)
}

func check(t *testing.T, name string, v voice, msg psalm.Message) {
	t.Helper()
	lufs, peak := measure(v, msg)
	atTarget := math.Abs(lufs-target) <= 1
	atCeiling := math.Abs(peak-ceiling) <= 1 && lufs < target+1
	if peak > ceiling+0.5 || (!atTarget && !atCeiling) {
		t.Errorf("%s: %.1f LUFS, peak %.1f dB: expected %v LUFS, or a %v dB peak", name, lufs, peak, target, ceiling)
	}
}

var note = psalm.Message{"pitch": 57.0, "velocity": 0.8}

func TestSubtractive(t *testing.T) {
	for n, f := range subtractive.Presets {
		fresh()
		check(t, n, subtractive.New(f()), note)
	}
}

func TestPulsar(t *testing.T) {
	low := map[string]float64{"rhythm": 6, "crackle": 50, "insects": 45, "raindrops": 12, "motor": 25,
		"grainCloud": 40, "shimmerDust": 60, "lowCloud": 25, "scatterBells": 20, "swarmCloud": 80,
		"stutterTick": 12, "gallop": 10, "clavePulse": 8, "machineGun": 18,
		"fireCrackle": 45, "crickets": 40, "drizzle": 15, "geigerTicks": 30}
	for n, f := range pulsar.Presets {
		fresh()
		msg := note
		if fq, ok := low[n]; ok {
			msg = psalm.Message{"frequency": fq} // textures for very low notes
		}
		check(t, n, pulsar.New(f()), msg)
	}
}

func TestWavetable(t *testing.T) {
	table := gen.Brightness(32)
	for n, f := range wavetable.Presets {
		fresh()
		check(t, n, wavetable.New(f(table)), note)
	}
}

func TestGrains(t *testing.T) {
	choir, err := sndfile.NewMipMapSoundFile[float32]("../../assets/sounds/various/mixkit-angelical-choir-654.wav", 4)
	if err != nil {
		t.Skip("choir recording not in assets")
	}
	for n, f := range grains.Presets {
		fresh()
		check(t, n, grains.New(f(choir, 57)), note)
	}
}

func TestDX(t *testing.T) {
	for n, f := range dx.Presets {
		fresh()
		check(t, n, dx.New(f(), 2), note)
	}
}

func TestPiano(t *testing.T) {
	for n, f := range piano.Presets {
		fresh()
		check(t, n, piano.NewPoly(f(), 4), note) // the whole instrument: voices, resonance, soundboard
	}
}

func TestOrgan(t *testing.T) {
	for n, f := range organ.Presets {
		fresh()
		check(t, n, organ.New(f()), note)
	}
}

func TestWaveguide(t *testing.T) {
	for n, f := range waveguide.Presets {
		fresh()
		check(t, n, waveguide.New(f()), note)
	}
}

func TestModal(t *testing.T) {
	for n, f := range modal.Presets {
		fresh()
		check(t, n, modal.New(f()), note)
	}
}

func TestVocal(t *testing.T) {
	for n, f := range vocal.Presets {
		fresh()
		check(t, n, vocal.New(f()), note)
	}
}

func TestPD(t *testing.T) {
	for n, f := range pd.Presets {
		fresh()
		check(t, n, pd.New(f()), note)
	}
}

func TestWestcoast(t *testing.T) {
	for n, f := range westcoast.Presets {
		fresh()
		check(t, n, westcoast.New(f()), note)
	}
}

func TestSupersaw(t *testing.T) {
	for n, f := range supersaw.Presets {
		fresh()
		check(t, n, supersaw.New(f()), note)
	}
}
