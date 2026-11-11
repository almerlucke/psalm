package mixer

import (
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/almerlucke/psalm"
)

// source plays a function of the sample index on its (mono or stereo) output
type source struct {
	*psalm.Module
	n int
	f func(n int, ch int) float64
}

func newSource(channels int, f func(n int, ch int) float64) *source {
	s := &source{Module: psalm.NewModule(nil, []int{channels}), f: f}
	s.SetSelf(s)
	return s
}

func (s *source) Synthesize() {
	out := s.Output(0).Channels
	for i := range out[0] {
		for ch := range out {
			out[ch][i] = float32(s.f(s.n, ch))
		}
		s.n++
	}
}

func constant(v float64) func(int, int) float64 { return func(int, int) float64 { return v } }

// gainer scales its input, an insert effect for tests
type gainer struct {
	*psalm.Module
	g float32
}

func newGainer(g float32) *gainer {
	m := &gainer{Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}), g: g}
	m.SetSelf(m)
	return m
}

func (m *gainer) Synthesize() {
	for ch, out := range m.Output(0).Channels {
		for i, x := range m.Input(0).Channels[ch] {
			out[i] = x * m.g
		}
	}
}

// render runs the patch and returns the master output, without the limiter latency
func render(p *psalm.Psalm, mix *Mixer, buffers int) ([]float64, []float64) {
	var l, r []float64
	for range buffers {
		p.ResetRun()
		p.Run()
		for i := range mix.Master().Output(0).Channels[0] {
			l = append(l, float64(mix.Master().Output(0).Channels[0][i]))
			r = append(r, float64(mix.Master().Output(0).Channels[1][i]))
		}
	}
	lat := mix.Master().Latency()
	return l[lat:], r[lat:]
}

func setup() (*psalm.Psalm, *Mixer) {
	p := psalm.New(nil)
	mix := New(p)
	p.Out(mix.Master())
	return p, mix
}

func last(x []float64) float64 { return x[len(x)-1] }

func TestUnity(t *testing.T) {
	p, mix := setup()
	mix.Channel("a", newSource(psalm.Mono, constant(0.25)).AddTo(p), StripConfig{})
	l, r := render(p, mix, 4)
	for i := range l {
		if math.Abs(l[i]-0.25) > 1e-6 || math.Abs(r[i]-0.25) > 1e-6 {
			t.Fatalf("a strip at 0 dB in the centre should pass a mono source on both sides, got %v %v at %d", l[i], r[i], i)
		}
	}
}

func TestVolumeAndPan(t *testing.T) {
	p, mix := setup()
	mix.Channel("a", newSource(psalm.Mono, constant(0.25)).AddTo(p), StripConfig{Volume: -6, Pan: -1})
	l, r := render(p, mix, 4)
	want := 0.25 * DB(-6) * math.Sqrt2 // equal power: hard left gets the power of both sides
	if math.Abs(last(l)-want) > 1e-6 || math.Abs(last(r)) > 1e-6 {
		t.Fatalf("expected %v on the left only, got %v %v", want, last(l), last(r))
	}
}

func TestSendsAndBuses(t *testing.T) {
	p, mix := setup()
	mix.Bus("fx", StripConfig{Volume: -6}, newGainer(1).AddTo(p))
	mix.Channel("a", newSource(psalm.Mono, constant(0.2)).AddTo(p), StripConfig{Volume: -120, Sends: map[string]float64{"fx": 0.5}})
	mix.Channel("b", newSource(psalm.Mono, constant(0.2)).AddTo(p), StripConfig{Volume: -120, PreFader: true, Sends: map[string]float64{"fx": 0.5}})
	l, _ := render(p, mix, 4)
	// a sends after its (silent) fader: nothing; b before it: 0.1, returned at -6 dB
	if want := 0.1 * DB(-6); math.Abs(last(l)-want) > 1e-6 {
		t.Fatalf("expected only the pre fader send to return, %v, got %v", want, last(l))
	}
}

func TestGroupWithInsert(t *testing.T) {
	p, mix := setup()
	mix.Group("g", StripConfig{}, newGainer(0.5).AddTo(p))
	mix.Channel("a", newSource(psalm.Mono, constant(0.1)).AddTo(p), StripConfig{Group: "g"})
	mix.Channel("b", newSource(psalm.Stereo, constant(0.3)).AddTo(p), StripConfig{Group: "g"})
	mix.Channel("c", newSource(psalm.Mono, constant(0.05)).AddTo(p), StripConfig{})
	l, _ := render(p, mix, 4)
	if want := (0.1+0.3)*0.5 + 0.05; math.Abs(last(l)-want) > 1e-6 {
		t.Fatalf("expected the group through its insert plus the direct channel, %v, got %v", want, last(l))
	}
}

func TestMuteAndSoloDontClick(t *testing.T) {
	p, mix := setup()
	mix.Bus("fx", StripConfig{}, newGainer(1).AddTo(p))
	a := mix.Channel("a", newSource(psalm.Mono, constant(0.2)).AddTo(p), StripConfig{Sends: map[string]float64{"fx": 0.5}})
	b := mix.Channel("b", newSource(psalm.Mono, constant(0.3)).AddTo(p), StripConfig{})
	render(p, mix, 2)

	b.SetSolo(true) // a falls silent, the fx return is solo safe but a no longer sends
	l, _ := render(p, mix, 4)
	if math.Abs(last(l)-0.3) > 1e-4 {
		t.Fatalf("expected only the soloed channel, got %v", last(l))
	}
	maxStep := 0.0
	for i := 1; i < len(l); i++ {
		maxStep = max(maxStep, math.Abs(l[i]-l[i-1]))
	}
	if maxStep > 0.01 {
		t.Fatalf("solo should fade, not step: %v", maxStep)
	}

	b.SetSolo(false)
	a.SetMute(true)
	l, _ = render(p, mix, 4)
	if math.Abs(last(l)-0.3) > 1e-4 {
		t.Fatalf("expected the muted channel gone, got %v", last(l))
	}
}

func TestFade(t *testing.T) {
	p, mix := setup()
	a := mix.Channel("a", newSource(psalm.Mono, constant(0.5)).AddTo(p), StripConfig{})
	render(p, mix, 1)
	a.FadeTo(-20, 1)
	l, _ := render(p, mix, int(1.2*psalm.SampleRate()/float64(psalm.BufferSize())))
	half := int(0.5 * psalm.SampleRate())
	if db := ToDB(l[half] / 0.5); math.Abs(db+10) > 0.5 { // even dB steps: halfway is -10 dB
		t.Fatalf("expected -10 dB halfway the fade, got %.1f", db)
	}
	if db := ToDB(last(l) / 0.5); math.Abs(db+20) > 0.01 {
		t.Fatalf("expected -20 dB after the fade, got %.2f", db)
	}
}

func TestLimiter(t *testing.T) {
	p, mix := setup()
	cfg := DefaultMasterConfig()
	cfg.Release = 50
	mix.Master().SetConfig(cfg)
	sr := psalm.SampleRate()
	// a quiet tone with loud bursts
	mix.Channel("a", newSource(psalm.Stereo, func(n, _ int) float64 {
		amp := 0.3
		if (n/4410)%4 == 1 {
			amp = 2.0
		}
		return amp * math.Sin(2*math.Pi*220*float64(n)/sr)
	}).AddTo(p), StripConfig{})
	l, _ := render(p, mix, 40)
	ceiling := DB(-1)
	for i, x := range l {
		if math.Abs(x) > ceiling+1e-6 {
			t.Fatalf("sample %d over the ceiling: %v", i, x)
		}
	}
	// the quiet part after a burst is untouched once the limiter let go
	quiet := l[int(0.39*sr):int(0.4*sr)]
	peak := 0.0
	for _, x := range quiet {
		peak = max(peak, math.Abs(x))
	}
	if math.Abs(peak-0.3) > 0.01 {
		t.Fatalf("the limiter should let go of quiet parts, peak %v", peak)
	}
	// inside a burst the tone is turned down, not clipped: clipping would add odd harmonics
	burst := l[int(0.13*sr):int(0.18*sr)]
	level := func(freq float64) float64 {
		re, im := 0.0, 0.0
		for i, x := range burst {
			re += x * math.Cos(2*math.Pi*freq*float64(i)/sr)
			im += x * math.Sin(2*math.Pi*freq*float64(i)/sr)
		}
		return math.Hypot(re, im)
	}
	if third := ToDB(level(660) / level(220)); third > -50 {
		t.Fatalf("the limited burst has a third harmonic at %.1f dB, it was clipped", third)
	}
	// the bursts (2.0) are turned down to the ceiling (0.89): about 7 dB
	if red := mix.Master().Reduction(); red < 5 || red > 8 {
		t.Fatalf("expected about 7 dB of gain reduction, got %.1f", red)
	}
}

func TestMetersAndMessages(t *testing.T) {
	p, mix := setup()
	a := mix.Channel("a", newSource(psalm.Mono, constant(0.5)).AddTo(p), StripConfig{Sends: map[string]float64{}})
	render(p, mix, 4)
	peak, rms := a.Levels()
	if math.Abs(peak-ToDB(0.5)) > 0.01 || math.Abs(rms-ToDB(0.5)) > 0.01 {
		t.Fatalf("meters %v %v", peak, rms)
	}
	report := mix.Meters()
	if !strings.Contains(report, "master") || !strings.Contains(report, "limiter") {
		t.Fatalf("unexpected report:\n%s", report)
	}

	a.ReceiveMessage(psalm.Message{"volume": -12.0, "pan": 0.5})
	a.Receive(1.0, 3) // mute
	if s := a.Settings(); s.Volume != -12 || s.Pan != 0.5 || !s.Mute {
		t.Fatalf("unexpected settings %+v", s)
	}
	mix.ResetMeters()
	if peak, _ := a.Levels(); peak != MinDB {
		t.Fatal("reset meters should read silence")
	}
}

func TestRoutingMistakesPanic(t *testing.T) {
	_, mix := setup()
	for _, cfg := range []StripConfig{{Group: "nope"}, {Sends: map[string]float64{"nope": 1}}} {
		func() {
			defer func() {
				if r := recover(); r == nil || !strings.Contains(r.(string), "nope") {
					t.Fatalf("expected a panic naming the missing group or bus, got %v", r)
				}
			}()
			mix.Channel("x", newSource(psalm.Mono, constant(0)), cfg)
		}()
	}
}

// late delays its input by some samples and reports it, like a vocoder
type late struct {
	*psalm.Module
	buf [][]float32
	pos int
}

func newLate(samples int) *late {
	l := &late{Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}), buf: [][]float32{make([]float32, samples), make([]float32, samples)}}
	l.SetSelf(l)
	return l
}

func (l *late) SignalLatency() int { return len(l.buf[0]) }

func (l *late) Synthesize() {
	in, out := l.Input(0).Channels, l.Output(0).Channels
	for i := range in[0] {
		for c := range 2 {
			out[c][i] = l.buf[c][l.pos]
			l.buf[c][l.pos] = in[c][i]
		}
		l.pos = (l.pos + 1) % len(l.buf[0])
	}
}

// click is a single click at sample 0
func click(n int, _ int) float64 {
	if n == 0 {
		return 0.25
	}
	return 0
}

// peaks returns the sample positions where the output is not silent
func peaks(x []float64) []int {
	var at []int
	for i, v := range x {
		if math.Abs(v) > 1e-6 {
			at = append(at, i)
		}
	}
	return at
}

func TestLatencyIsLinedUp(t *testing.T) {
	p, mix := setup()
	lateSrc := newLate(300)
	lateSrc.AddTo(p)
	lateSrc.In(newSource(psalm.Stereo, click).AddTo(p))
	mix.Channel("vocoder", lateSrc, StripConfig{})
	mix.Channel("piano", newSource(psalm.Stereo, click).AddTo(p), StripConfig{})
	l, _ := render(p, mix, 4)
	if at := peaks(l); len(at) != 1 || at[0] != 300 || math.Abs(l[300]-0.5) > 1e-6 {
		t.Fatalf("expected both clicks together at sample 300, got %v", at)
	}
	if mix.Strip("piano").Delay() != 300 || mix.Strip("vocoder").Delay() != 0 {
		t.Fatalf("delays %d %d", mix.Strip("piano").Delay(), mix.Strip("vocoder").Delay())
	}
}

func TestLatencyInGroups(t *testing.T) {
	p, mix := setup()
	mix.Group("voices", StripConfig{})
	lateSrc := newLate(200)
	lateSrc.AddTo(p)
	lateSrc.In(newSource(psalm.Stereo, click).AddTo(p))
	mix.Channel("vocoder", lateSrc, StripConfig{Group: "voices"})
	mix.Channel("choir", newSource(psalm.Stereo, click).AddTo(p), StripConfig{Group: "voices"})
	mix.Channel("bass", newSource(psalm.Stereo, click).AddTo(p), StripConfig{})
	l, _ := render(p, mix, 4)
	if at := peaks(l); len(at) != 1 || at[0] != 200 {
		t.Fatalf("expected all clicks together at sample 200, got %v", at)
	}
	if mix.Strip("choir").Delay() != 200 || mix.Strip("bass").Delay() != 200 || mix.Strip("voices").Delay() != 0 {
		t.Fatal("expected the choir lined up in the group and the bass with the group")
	}
}

// A late effect on a bus is a pre delay, the dry signal isn't held back for it
func TestBusLatencyIsNotCompensated(t *testing.T) {
	p, mix := setup()
	fx := newLate(500)
	fx.AddTo(p)
	mix.Bus("reverb", StripConfig{}, fx)
	mix.Channel("piano", newSource(psalm.Stereo, click).AddTo(p), StripConfig{Sends: map[string]float64{"reverb": 1}})
	l, _ := render(p, mix, 4)
	if at := peaks(l); len(at) != 2 || at[0] != 0 || at[1] != 500 {
		t.Fatalf("expected the dry click at 0 and the wet one at 500, got %v", at)
	}
}

func TestRemove(t *testing.T) {
	p, mix := setup()
	fx := newGainer(1).AddTo(p)
	mix.Bus("fx", StripConfig{}, fx)
	mix.Group("g", StripConfig{})
	a := newSource(psalm.Mono, constant(0.2)).AddTo(p)
	mix.Channel("a", a, StripConfig{Sends: map[string]float64{"fx": 0.5}})
	mix.Channel("b", newSource(psalm.Mono, constant(0.1)).AddTo(p), StripConfig{Group: "g"})
	l, _ := render(p, mix, 4)
	if want := 0.2 + 0.1 + 0.1; math.Abs(last(l)-want) > 1e-6 {
		t.Fatalf("a, its send and b through the group: %v, got %v", want, last(l))
	}

	if err := mix.Remove("nope", 0); err == nil {
		t.Fatal("an unknown strip is an error")
	}
	if err := mix.Remove("g", 0); err == nil || !strings.Contains(err.Error(), "b still plays into") {
		t.Fatalf("a group with strips in it can't go: %v", err)
	}

	seconds := 2 * float64(psalm.BufferSize()) / psalm.SampleRate()
	if err := mix.Remove("a", seconds); err != nil {
		t.Fatal(err)
	}
	l, _ = render(p, mix, 1)
	if !(last(l) < l[0]-0.05 && last(l) > 0.1) { // going down, not cut
		t.Fatalf("a should fade out (and its send with it), not stop: %v ... %v", l[0], last(l))
	}
	l, _ = render(p, mix, 3)
	if math.Abs(last(l)-0.1) > 1e-6 || mix.Strip("a") != nil || len(mix.Strips()) != 3 {
		t.Fatalf("after the fade only b is left: %v, strips %d", last(l), len(mix.Strips()))
	}
	if len(a.Output(0).Connections) != 0 {
		t.Fatal("the source of a removed channel should be left on its own")
	}

	// a bus goes with its effects, a group once it is empty
	if err := mix.Remove("fx", 0); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(p.Modules(), fx) || mix.Strip("fx") != nil {
		t.Fatal("a bus removed should take its effects with it")
	}
	if err := mix.Remove("b", 0); err != nil {
		t.Fatal(err)
	}
	if err := mix.Remove("g", 0); err != nil {
		t.Fatalf("an empty group can go: %v", err)
	}
	l, _ = render(p, mix, 2)
	if last(l) != 0 || len(mix.Strips()) != 0 {
		t.Fatalf("nothing left: %v, %d strips", last(l), len(mix.Strips()))
	}
}
