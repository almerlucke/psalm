package drumsynth

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/poly"
)

var _ poly.Voice = (*Voice)(nil)

func sr() float64 { return psalm.CurrentEnv().Config().SampleRate }

func run(v *Voice, seconds float64) []float64 {
	var x []float64
	for range int(math.Ceil(seconds * sr() / float64(v.Config().BufferSize))) {
		v.Synthesize()
		for _, s := range v.Output(0).Channels[0] {
			x = append(x, float64(s))
		}
	}
	return x
}

func hit(kit Kit, msg psalm.Message, seconds float64) (*Voice, []float64) {
	v := New(Config{Kit: kit})
	v.NoteOn(1, msg, psalm.CtrlTick{}, v.Config())
	return v, run(v, seconds)
}

func peak(x []float64) float64 {
	p := 0.0
	for _, s := range x {
		p = max(p, math.Abs(s))
	}
	return p
}

// frequency counts zero crossings
func frequency(x []float64) float64 {
	n := 0
	for i := 1; i < len(x); i++ {
		if (x[i-1] < 0) != (x[i] < 0) {
			n++
		}
	}
	return float64(n) / 2 / (float64(len(x)) / sr())
}

func TestEverySoundPlaysAndEnds(t *testing.T) {
	for kn, f := range Kits {
		for name, snd := range f().Sounds {
			v, x := hit(f(), psalm.Message{"sound": name}, snd.Decay+0.3)
			if p := 20 * math.Log10(peak(x)); p < -20 || p > -1 || math.IsNaN(p) { // noisy sounds vary hit to hit
				t.Errorf("%s %s peaks at %.1f dB", kn, name, p)
			}
			if v.IsActive() {
				t.Errorf("%s %s plays on after its decay", kn, name)
			}
		}
	}
}

func TestKickPitch(t *testing.T) {
	kit := Kit808()
	_, x := hit(kit, psalm.Message{"sound": "kick"}, 0.6)
	if f := frequency(x[int(0.15*sr()):int(0.5*sr())]); math.Abs(f-49) > 2 {
		t.Errorf("the kick settles at %.1f Hz, want 49", f)
	}
	// the sweep: it starts an octave up
	if f := frequency(x[:int(0.03*sr())]); f < 60 {
		t.Errorf("the kick starts at %.1f Hz, want it falling from 98", f)
	}
	// a step's pitch (speed) moves it
	_, x = hit(kit, psalm.Message{"sound": "kick", "speed": 2.0}, 0.6)
	if f := frequency(x[int(0.15*sr()):int(0.5*sr())]); math.Abs(f-98) > 3 {
		t.Errorf("the kick at speed 2: %.1f Hz, want 98", f)
	}
}

func TestHatIsHigh(t *testing.T) {
	_, x := hit(Kit808(), psalm.Message{"sound": "hat"}, 0.1)
	// four one poles split at 2 kHz: almost all of a hat is above it
	low, all := 0.0, 0.0
	var lp [4]float64
	c := math.Exp(-2 * math.Pi * 2000 / sr())
	for _, s := range x {
		in := s
		for k := range lp {
			lp[k] = (1-c)*in + c*lp[k]
			in = lp[k]
		}
		low += in * in
		all += s * s
	}
	if low > all*0.02 {
		t.Errorf("%.1f%% of a hat is below 2 kHz", 100*low/all)
	}
}

func TestKeysAndUnknownSounds(t *testing.T) {
	kit := Kit909()
	_, byKey := hit(kit, psalm.Message{"pitch": 38.0}, 0.3)
	_, byName := hit(kit, psalm.Message{"sound": "snare"}, 0.3)
	if math.Abs(20*math.Log10(peak(byKey)/peak(byName))) > 2 {
		t.Errorf("key 38 is not the snare: %g against %g", peak(byKey), peak(byName))
	}
	v, x := hit(kit, psalm.Message{"sound": "gong"}, 0.1)
	if peak(x) != 0 || v.IsActive() {
		t.Errorf("an unknown sound plays")
	}
	_, x = hit(kit, psalm.Message{"pitch": 100.0}, 0.1)
	if peak(x) != 0 {
		t.Errorf("a key without a sound plays")
	}
}

func TestChoke(t *testing.T) {
	// the closed hat stops the open one; a note off stops a sound
	p := NewPoly(DefaultConfig(), 4)
	trigger := func(id, sound string) {
		p.ReceiveMessage(psalm.Message{"command": "trigger", "tick": psalm.CtrlTick{}, "noteOn": id, "amplitude": 1.0,
			"message": psalm.Message{"sound": sound}})
	}
	trigger("o", "open")
	p.Synthesize()
	trigger("h", "hat")
	var open *Voice
	p.CallVoices(func(v *Voice) {
		if v.snd.Decay == Kit808().Sounds["open"].Decay {
			open = v
		}
	})
	for range int(0.02 * sr() / float64(p.Config().BufferSize)) {
		p.Synthesize()
	}
	p.Synthesize()
	if open == nil || open.IsActive() {
		t.Errorf("the open hat plays on after the closed one")
	}
	v := New(DefaultConfig())
	v.NoteOn(1, psalm.Message{"sound": "cymbal"}, psalm.CtrlTick{}, v.Config())
	run(v, 0.1)
	v.NoteOff()
	run(v, 0.02)
	if v.IsActive() {
		t.Errorf("a note off doesn't choke the cymbal")
	}
}

func TestHitStartsOnItsSample(t *testing.T) {
	v := New(DefaultConfig())
	v.NoteOn(1, psalm.Message{"sound": "snare"}, psalm.CtrlTick{SampleOffset: 64}, v.Config())
	v.Synthesize()
	out := v.Output(0).Channels[0]
	if peak(f64(out[:64])) != 0 || peak(f64(out[64:80])) == 0 {
		t.Errorf("the hit doesn't start on sample 64")
	}
}

func f64(x []float32) []float64 {
	y := make([]float64, len(x))
	for i, s := range x {
		y[i] = float64(s)
	}
	return y
}

func TestEveryKitHasTheGeneralMIDIDrums(t *testing.T) {
	for kn, f := range Kits {
		kit := f()
		for key, name := range GeneralMIDI {
			if _, ok := kit.Sounds[name]; !ok {
				t.Errorf("%s has no %s (key %d)", kn, name, key)
			}
		}
	}
}
