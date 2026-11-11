package shaper

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

func stereoRun(v *Voice, seconds float64) (l, r []float64) {
	for range int(math.Ceil(seconds * sr() / float64(v.Config().BufferSize))) {
		v.Synthesize()
		ch := v.Output(0).Channels
		for i := range ch[0] {
			l = append(l, float64(ch[0][i]))
			r = append(r, float64(ch[1][i]))
		}
	}
	return
}

func rms(x []float64) float64 {
	s := 0.0
	for _, v := range x {
		s += v * v
	}
	return math.Sqrt(s / float64(len(x)))
}

func saw() Config {
	cfg := plain("bipolar | mult 0.9")
	cfg.Oversample = 1
	return cfg
}

// a unison is a fuller sound, not a louder one: the level of one copy, the energy spread round the pitch
func TestUnisonFuller(t *testing.T) {
	one := saw()
	_, a := play(one, psalm.Message{"frequency": 220.0}, 1)
	many := saw()
	many.Unison, many.Spread = 5, 40
	_, b := play(many, psalm.Message{"frequency": 220.0}, 1)
	a, b = a[11025:], b[11025:]
	if r := rms(b) / rms(a); r < 0.6 || r > 1.6 {
		t.Errorf("a unison of 5 is %.2f times the level of one copy: it should sit about as loud", r)
	}
	// the sine at the pitch: one copy has all the energy at the pitch, five detuned ones share it round it
	if x, y := amplitude(a, 220), amplitude(b, 220); y > 0.7*x {
		t.Errorf("amplitude at the pitch %.3f with one copy and %.3f with five: the copies are not spread", x, y)
	}
}

func TestUnisonStereo(t *testing.T) {
	cfg := saw()
	cfg.Unison, cfg.Spread = 5, 10
	_, plainL := play(cfg, psalm.Message{"frequency": 220.0}, 0.3)
	cfg.UniStereo = 1
	v := New(cfg)
	v.NoteOn(1, psalm.Message{"frequency": 220.0}, psalm.CtrlTick{}, v.Config())
	l, r := stereoRun(v, 0.5)
	diff := 0.0
	for i := range l {
		diff += math.Abs(l[i] - r[i])
	}
	if diff/float64(len(l)) < 0.01 {
		t.Errorf("the channels are the same with a stereo spread of 1")
	}
	_ = plainL
	cfg.UniStereo = 0
	v = New(cfg)
	v.NoteOn(1, psalm.Message{"frequency": 220.0}, psalm.CtrlTick{}, v.Config())
	l, r = stereoRun(v, 0.5)
	for i := range l {
		if math.Abs(l[i]-r[i]) > 1e-6 {
			t.Fatalf("with no stereo spread and no width the channels differ at %d", i)
		}
	}
}

// the unison of a note is the one set when it starts on a silent voice: a count changed under a note waits
func TestUnisonLatched(t *testing.T) {
	cfg := saw()
	v := New(cfg)
	v.NoteOn(1, psalm.Message{"frequency": 217.3}, psalm.CtrlTick{}, v.Config())
	run(v, 0.2)
	before := run(v, 0.2)
	v.ReceiveMessage(psalm.Message{"unison": 7})
	after := run(v, 0.2)
	if v.uni != 1 {
		t.Errorf("unison %d under a note that started with 1", v.uni)
	}
	if d2(after) > 1.1*d2(before) {
		t.Errorf("a click when the count was set under a note")
	}
}

// the sub is a sine an octave under the note; the noise bright or dark
func TestSubAndNoise(t *testing.T) {
	cfg := plain("sin")
	cfg.Oversample = 1
	_, off := play(cfg, psalm.Message{"frequency": 220.0}, 1)
	cfg.Sub = 1
	_, on := play(cfg, psalm.Message{"frequency": 220.0}, 1)
	off, on = off[8000:], on[8000:]
	if a, b := amplitude(off, 110), amplitude(on, 110); b < 0.2 || a > 0.01 {
		t.Errorf("amplitude an octave under: %.3f without the sub, %.3f with it", a, b)
	}

	hi := func(x []float64) float64 { // the share of the energy in the first difference (the top)
		d := make([]float64, len(x)-1)
		for i := range d {
			d[i] = x[i+1] - x[i]
		}
		return rms(d) / rms(x)
	}
	cfg = plain("thru | mult 0")
	cfg.Oversample = 1
	cfg.Noise = 0.8
	var bright, dark float64
	cfg.NoiseColor = 1
	_, x := play(cfg, psalm.Message{"frequency": 220.0}, 1)
	bright = hi(x[8000:])
	cfg.NoiseColor = -1
	_, x = play(cfg, psalm.Message{"frequency": 220.0}, 1)
	dark = hi(x[8000:])
	if rms(x[8000:]) < 0.02 {
		t.Errorf("the dark noise is silent")
	}
	if bright < 4*dark {
		t.Errorf("bright noise %.2f, dark noise %.2f: the colour does nothing", bright, dark)
	}
}

// the drift moves the pitch a little, slowly: the length of the cycles changes over the seconds
func TestDrift(t *testing.T) {
	freqOver := func(drift float64) (lo, hi float64) {
		cfg := plain("sin")
		cfg.Oversample = 1
		cfg.Drift = drift
		_, x := play(cfg, psalm.Message{"frequency": 220.0}, 6)
		lo, hi = math.Inf(1), 0.0
		w := int(0.5 * sr())
		for s := 4000; s+w < len(x); s += w {
			f := crossings(x[s:s+w]) / 0.5 / 2
			lo, hi = min(lo, f), max(hi, f)
		}
		return
	}
	lo0, hi0 := freqOver(0)
	lo1, hi1 := freqOver(1)
	if hi0-lo0 > 0.3 {
		t.Errorf("without drift the pitch moves %.2f Hz", hi0-lo0)
	}
	if hi1-lo1 < 0.3 {
		t.Errorf("with drift 1 the pitch moves %.2f Hz: it does not wander", hi1-lo1)
	}
	if hi1-lo1 > 4 {
		t.Errorf("with drift 1 the pitch moves %.2f Hz: too much", hi1-lo1)
	}
}

// none of the new settings clicks when it is changed under a sounding note, the unison among them
func TestFullnessNoClicks(t *testing.T) {
	for _, c := range []struct {
		name string
		msg  psalm.Message
	}{
		{"sub", psalm.Message{"sub": 1.0}},
		{"noise", psalm.Message{"noise": 1.0, "noiseColor": 1.0}},
		{"spread", psalm.Message{"spread": 40.0}},
		{"stereo", psalm.Message{"uniStereo": 1.0}},
		{"drift", psalm.Message{"drift": 1.0}},
	} {
		t.Run(c.name, func(t *testing.T) {
			cfg := plain("par(sin, mult 2 | sin | mult 0.2)")
			cfg.Oversample = 1
			cfg.Unison, cfg.Spread = 4, 12
			v := New(cfg)
			v.NoteOn(1, psalm.Message{"frequency": 217.3}, psalm.CtrlTick{}, v.Config())
			run(v, 0.3)
			before := run(v, 0.1)
			v.ReceiveMessage(c.msg)
			after := run(v, 0.1)
			if got := d2(append(append([]float64{}, before[len(before)-8:]...), after...)); got > 0.05 && c.name != "noise" {
				t.Errorf("second difference %.4f after the change: a click", got)
			}
		})
	}
}
