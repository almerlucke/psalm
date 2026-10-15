package organ

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

func leslieConfig(bars string, speed float64) Config {
	cfg := clean(bars)
	cfg.Tonewheel = 0
	cfg.Leslie, cfg.LeslieSpeed = 1, speed
	return cfg
}

// envelopeRate is how often the level of x swings per second
func envelopeRate(x []float64, f float64) float64 {
	win := int(math.Ceil(0.01*f) / f * sr()) // whole cycles of the tone

	var env, ts []float64
	for i := 0; i+win < len(x); i += win / 2 {
		env = append(env, rms(x[i:i+win]))
		ts = append(ts, float64(i+win/2)/sr())
	}
	return swingRate(env, ts, 5)
}

func TestLeslieDopplerAndAmplitudeModulation(t *testing.T) {
	for _, c := range []struct {
		name  string
		pitch float64 // a horn note (above the crossover) or a drum note
		speed float64
		want  float64
		fm    bool
		secs  float64
	}{
		{"horn slow", 96, 0.5, 0.8, true, 8},
		{"horn fast", 96, 1, 6.7, true, 3},
		{"drum slow", 45, 0.5, 0.7, false, 8},
		{"drum fast", 45, 1, 6.0, false, 3},
	} {
		o := New(leslieConfig("008000000", c.speed))
		down(o, "a", c.pitch)
		x := render(o, c.secs)[int(0.5*sr()):]
		if r := envelopeRate(x, gearHz(int(c.pitch))); math.Abs(r-c.want) > 0.12*c.want {
			t.Errorf("%s: the level swings at %.2f Hz, want %.1f", c.name, r, c.want)
		}
		if c.fm {
			// the Doppler effect: the pitch swings at the same rate, by about the speed of the horn
			if r := modRate(x); math.Abs(r-c.want) > 0.12*c.want {
				t.Errorf("%s: the pitch swings at %.2f Hz, want %.1f", c.name, r, c.want)
			}
		}
	}
	// the horn's Doppler shows as a pitch deviation of percents when fast, tenths when slow
	dev := func(speed float64) float64 {
		o := New(leslieConfig("008000000", speed))
		down(o, "a", 96)
		x := render(o, 3)[int(0.5*sr()):]
		return spread(x)
	}
	if slow, fast := dev(0.5), dev(1); fast < 0.02 || fast > 0.12 || slow > fast/4 || slow < 0.001 {
		t.Errorf("pitch swing %.4f slow, %.4f fast", slow, fast)
	}
}

func TestLeslieInertia(t *testing.T) {
	cfg := leslieConfig("008000000", 0)
	o := New(cfg)
	if o.les.horn.rate != 0 || o.les.drum.rate != 0 {
		t.Fatalf("stopped rotors turn: %g %g", o.les.horn.rate, o.les.drum.rate)
	}
	o.ReceiveMessage(psalm.Message{"leslieSpeed": 1.0})
	var hornAt, drumAt float64
	step := float64(o.Config().BufferSize) / sr()
	for i := 0; i < int(12/step); i++ {
		o.Synthesize()
		now := float64(i+1) * step
		if hornAt == 0 && o.les.horn.rate > 0.95*6.7 {
			hornAt = now
		}
		if drumAt == 0 && o.les.drum.rate > 0.95*6.0 {
			drumAt = now
		}
	}
	if hornAt < 0.8 || hornAt > 2 {
		t.Errorf("the horn takes %.2f s to speed up", hornAt)
	}
	if drumAt < 2 || drumAt > 4 || drumAt < hornAt*1.5 {
		t.Errorf("the drum takes %.2f s (the horn %.2f s)", drumAt, hornAt)
	}
	// slowing down: the horn is at slow speed only after a while, and not at once
	o.ReceiveMessage(psalm.Message{"leslieSpeed": 0.5})
	o.Synthesize()
	if o.les.horn.rate < 6 {
		t.Errorf("the horn dropped to %g at once", o.les.horn.rate)
	}
	for range int(3 / step) {
		o.Synthesize()
	}
	if r := o.les.horn.rate; math.Abs(r-0.8) > 0.15 {
		t.Errorf("after 3 s the horn turns at %.2f, want 0.8", r)
	}
	// in the middle of a note the swing of the horn's pitch speeds up with the rotor
	o = New(leslieConfig("008000000", 0.5))
	down(o, "a", 96)
	slow := modRate(render(o, 9)[int(sr()):])
	o.ReceiveMessage(psalm.Message{"leslieSpeed": 1.0})
	render(o, 0.5)
	if r := o.les.horn.rate; r < 1.5 || r > 6 {
		t.Errorf("half a second into speeding up the horn turns at %.2f", r)
	}
	render(o, 2.5)
	fast := modRate(render(o, 3))
	if math.Abs(slow-0.8) > 0.15 || math.Abs(fast-6.7) > 0.7 {
		t.Errorf("swing %.2f Hz slow, %.2f Hz fast", slow, fast)
	}
}

func TestLeslieStereoAndOff(t *testing.T) {
	cfg := leslieConfig("888888888", 1)
	cfg.Leslie = 0
	o := New(cfg)
	down(o, "a", 60)
	render(o, 0.2)
	ch := o.Output(0).Channels
	for i := range ch[0] {
		if ch[0][i] != ch[1][i] {
			t.Fatal("with the Leslie off the channels differ")
		}
	}
	cfg.Leslie = 1
	o = New(cfg)
	down(o, "a", 60)
	render(o, 1)
	diff := 0.0
	for i := range ch[0] {
		diff += math.Abs(float64(o.Output(0).Channels[0][i] - o.Output(0).Channels[1][i]))
	}
	if diff == 0 {
		t.Error("the microphones hear the same")
	}
	// the microphones close together hear almost the same
	cfg.LesliePosition = 0
	o = New(cfg)
	down(o, "a", 60)
	render(o, 1)
	same := 0.0
	for i := range ch[0] {
		same += math.Abs(float64(o.Output(0).Channels[0][i] - o.Output(0).Channels[1][i]))
	}
	if same > diff/10 {
		t.Errorf("microphones together: %g apart, %g at a quarter turn", same, diff)
	}
	// a mono organ plays through it too
	cfg.NumChannels = 1
	m := New(cfg)
	down(m, "a", 60)
	if p := peak(render(m, 0.3)); p == 0 || math.IsNaN(p) {
		t.Errorf("mono leslie peaks at %g", p)
	}
}

func TestLeslieCrossover(t *testing.T) {
	// below the crossover the drum, above the horn: a low note has no Doppler, a high one does
	o := New(leslieConfig("008000000", 1))
	down(o, "a", 45)
	if s := spread(render(o, 2)[int(0.5*sr()):]); s > 0.015 {
		t.Errorf("a bass note has a pitch swing of %.4f", s)
	}
}
