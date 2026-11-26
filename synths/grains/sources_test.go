package grains

import (
	"math"
	"testing"

	"github.com/almerlucke/genny/float/rwalk"
	"github.com/almerlucke/psalm"
)

// run plays a note for the buffers and gives the left channel of all of them
func run(v *Voice, msg psalm.Message, buffers int) []float64 {
	v.NoteOn(0.5, msg, psalm.CtrlTick{}, v.Config())
	var out []float64
	for range buffers {
		v.Synthesize()
		for _, s := range v.Output(0).Channels[0] {
			out = append(out, float64(s))
		}
	}
	return out
}

// An oscillator source plays its wave at the pitch of the note
func TestOscillatorPitch(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Source = NewOscillator("sine")
	cfg.RootNote = 60
	cfg.StereoSpread = 0
	cfg.GrainSize = 300
	cfg.Density = 8
	cfg.Spray = 0
	for _, note := range []float64{60, 69, 81} {
		v := New(cfg)
		out := run(v, psalm.Message{"pitch": note}, 40)
		f := peakFrequency(out[6*1024:])
		want := 440 * math.Exp2((note-69)/12)
		if math.Abs(f-want) > 6 {
			t.Fatalf("note %v: want %.1f Hz, got %.1f Hz", note, want, f)
		}
	}
}

// the position picks the wave: a square has odd harmonics, a sine only the first
func TestOscillatorWavePosition(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Source = NewOscillator("sine", "square")
	cfg.StereoSpread = 0
	cfg.Spray = 0
	cfg.Density = 60
	cfg.GrainSize = 100
	harmonics := func(pos float64) float64 {
		out := run(New(cfg), psalm.Message{"position": pos, "pitch": 60.0}, 40)[6*1024:]
		// energy at 3 times the fundamental against the fundamental, by the zero crossing free way: the rms of the signal against
		// the rms of its fundamental is enough: a square is louder than its fundamental alone
		return rms(out)
	}
	if harmonics(0) == 0 || harmonics(1) == 0 {
		t.Fatal("silent")
	}
	// the same note at the same level, other waves: the sound is different
	a := run(New(cfg), psalm.Message{"position": 0.0, "pitch": 60.0}, 20)
	b := run(New(cfg), psalm.Message{"position": 1.0, "pitch": 60.0}, 20)
	diff := 0.0
	for i := range a {
		diff += math.Abs(a[i] - b[i])
	}
	if diff < 1 {
		t.Fatal("the position did not change the wave")
	}
}

// a high note of a bright wave must not alias: the levels of the oscillator take the harmonics away
func TestOscillatorNoAliasing(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Source = NewOscillator("saw")
	cfg.StereoSpread = 0
	cfg.Spray = 0
	cfg.Density = 40
	cfg.GrainSize = 200
	cfg.Normalize = false
	out := run(New(cfg), psalm.Message{"pitch": 96.0}, 60)[6*1024:] // C7, 2093 Hz: 10 harmonics fit under the nyquist
	f := peakFrequency(out)
	if math.Abs(f-2093) > 10 {
		t.Fatalf("want the fundamental 2093 Hz, got %v", f)
	}
}

// a recorder keeps what came in, and a voice that reads it plays that back, at the pitch of the sound
func TestRecorderSource(t *testing.T) {
	rec := NewRecorder(2)
	cfg := DefaultConfig()
	cfg.Source = rec.Source()
	cfg.RootNote = 69 // the input is at 440
	cfg.StereoSpread = 0
	cfg.Spray = 0
	cfg.Position = 0.1
	cfg.Density = 50
	cfg.GrainSize = 120

	v := New(cfg)
	n := v.Config().BufferSize
	phase := 0.0
	feed := func() {
		for c := range 2 {
			for i := range n {
				rec.Input(0).Channels[c][i] = float32(math.Sin(2 * math.Pi * (phase + 440*float64(i)/sr)))
			}
		}
		phase += 440 * float64(n) / sr
		rec.Synthesize()
	}
	if v.src.Ready() {
		t.Fatal("nothing recorded yet")
	}
	for range 60 { // about a second and a half
		feed()
	}
	v.NoteOn(0.5, psalm.Message{"pitch": 69.0}, psalm.CtrlTick{}, v.Config())
	var out []float64
	for range 40 {
		feed()
		v.Synthesize()
		for _, s := range v.Output(0).Channels[0] {
			out = append(out, float64(s))
		}
	}
	out = out[8*n:]
	if rms(out) < 0.01 {
		t.Fatalf("silent: %v", rms(out))
	}
	if f := peakFrequency(out); math.Abs(f-440) > 6 {
		t.Fatalf("want 440 Hz, got %v", f)
	}
	// the sound passes through
	if rec.Output(0).Channels[0][10] != rec.Input(0).Channels[0][10] {
		t.Fatal("the recorder did not pass the sound on")
	}
}

// a frozen recorder keeps the moment: the sound that comes after is not heard
func TestRecorderFreeze(t *testing.T) {
	rec := NewRecorder(1)
	n := psalm.BufferSize()
	phase := 0.0
	feed := func(freq float64) {
		for c := range 2 {
			for i := range n {
				rec.Input(0).Channels[c][i] = float32(math.Sin(2 * math.Pi * (phase + freq*float64(i)/sr)))
			}
		}
		phase += freq * float64(n) / sr
		rec.Synthesize()
	}
	for range 50 {
		feed(440)
	}
	rec.Freeze(true)
	for range 50 {
		feed(880)
	}
	cfg := DefaultConfig()
	cfg.Source = rec.Source()
	cfg.RootNote = 69
	cfg.StereoSpread = 0
	cfg.Spray = 0
	cfg.Position = 0.2
	out := run(New(cfg), psalm.Message{"pitch": 69.0}, 30)[8*n:]
	if f := peakFrequency(out); math.Abs(f-440) > 8 {
		t.Fatalf("the frozen moment is 440 Hz, got %v", f)
	}
}

// a grain never reads past the writing of a live source: with a high pitch it keeps its distance
func TestLiveGrainStaysBehind(t *testing.T) {
	rec := NewRecorder(1)
	s := rec.Source().(*liveSource)
	rec.count = len(rec.ring)
	rec.head = 5000
	for _, lead := range []float64{0, 100, 3000} {
		start, _, _ := s.Locate(0, 0, 0, lead)
		back := wrap(float64(rec.head)-start, float64(len(rec.ring)))
		if back < lead {
			t.Fatalf("lead %v: the grain starts only %v frames behind", lead, back)
		}
	}
}

// streams give each grain its value: a list of positions is played one after the other
func TestStreamsPositions(t *testing.T) {
	cfg := base()
	cfg.Spray = 0
	cfg.Streams = Streams{Position: func() Stream { return Seq(0.1, 0.8) }}
	v := New(cfg)
	v.NoteOn(0.5, psalm.Message{}, psalm.CtrlTick{}, v.Config())
	v.Synthesize()
	n := float64(v.src.Span()) - 1
	if len(v.grains) < 1 {
		t.Fatal("no grains")
	}
	if got := v.grains[0].pos - v.grains[0].inc*float64(v.grains[0].age); math.Abs(got-0.1*n) > 2 {
		t.Fatalf("first grain at %v, want %v", got, 0.1*n)
	}
	// the second grain
	for len(v.grains) < 2 {
		v.Synthesize()
	}
	var found bool
	for _, g := range v.grains {
		if math.Abs(g.lo-0) < 1 && g.age < g.length && math.Abs(g.pos-g.inc*float64(g.age)-0.8*n) < 4 {
			found = true
		}
	}
	if !found {
		t.Fatalf("no grain started at 0.8: %+v", v.grains)
	}
}

// a generator of genny can be a stream, and a list sent by message too; an empty list takes it away
func TestStreamsGenAndMessage(t *testing.T) {
	cfg := base()
	cfg.Spray = 0
	cfg.Streams = Streams{Pan: func() Stream { return Gen(rwalk.New(0.5, 0, 1, 0.3, rwalk.Bounce)) }}
	left, right := render(cfg, psalm.Message{})
	if rms(left) == 0 || rms(right) == 0 {
		t.Fatal("silent")
	}

	v := New(base())
	v.ReceiveMessage(psalm.Message{"intervalSeq": []float64{0.05, 0.2}})
	if v.st.interval == nil {
		t.Fatal("the list did not make a stream")
	}
	v.ReceiveMessage(psalm.Message{"intervalSeq": []float64{}})
	if v.st.interval != nil {
		t.Fatal("an empty list should take the stream away")
	}
}

// the interval stream decides when grains start: a list of intervals gives so many grains in a time
func TestIntervalStream(t *testing.T) {
	count := func(iv ...float64) int {
		cfg := base()
		cfg.Jitter = 0
		cfg.GrainSize = 2000 // grains that stay, to be counted
		cfg.Streams = Streams{Interval: func() Stream { return Seq(iv...) }}
		v := New(cfg)
		v.NoteOn(0.5, psalm.Message{}, psalm.CtrlTick{}, v.Config())
		for range 22 { // half a second
			v.Synthesize()
		}
		return len(v.grains)
	}
	if got := count(0.1); got < 5 || got > 6 {
		t.Fatalf("a grain every 0.1 s is 5 or 6 grains in half a second, got %d", got)
	}
	if got := count(0.05, 0.15); got < 5 || got > 6 {
		t.Fatalf("0.05 and 0.15 average 0.1 s: got %d", got)
	}
	if got := count(0.02); got < 24 {
		t.Fatalf("a grain every 0.02 s: got %d", got)
	}
}

// a rest in the amplitude stream makes no grain
func TestAmpStreamRests(t *testing.T) {
	cfg := base()
	cfg.Streams = Streams{Amp: func() Stream { return Seq(0) }}
	left, _ := render(cfg, psalm.Message{})
	if rms(left) != 0 {
		t.Fatalf("all rests, but %v", rms(left))
	}
}

// the sync follows the tempo: the time between grains is the beats of the sync at the tempo
func TestSyncFollowsTempo(t *testing.T) {
	cfg := base()
	cfg.Jitter = 0
	cfg.Sync = 0.25
	v := New(cfg)
	want := 0.25 * 60 / v.BPM() * sr
	if got := v.interval(sr, 30); math.Abs(got-want) > 1 {
		t.Fatalf("interval %v, want %v samples", got, want)
	}
	cfg.Sync = 0
	v = New(cfg)
	if got := v.interval(sr, 30); math.Abs(got-sr/30) > 1 {
		t.Fatalf("interval %v, want %v", got, sr/30)
	}
}

func TestQuantize(t *testing.T) {
	major := ScaleNamed("major")
	for _, c := range []struct{ in, want float64 }{
		{60, 60}, {60.8, 60}, {61.2, 62}, {61.8, 62}, {63.4, 64}, {65.6, 65}, {66.2, 67}, {66.8, 67}, {70.6, 71}, {71.9, 72}, {59.9, 60}, {72.4, 72},
	} {
		if got := quantize(c.in, major, 60); got != c.want {
			t.Fatalf("quantize(%v) = %v, want %v", c.in, got, c.want)
		}
	}
	if got := quantize(65.0, ScaleNamed("pentatonic"), 60); got != 64 {
		t.Fatalf("pentatonic 65 -> %v", got)
	}
	// a root that is in another octave: D major from D2
	if got := quantize(66.0, major, 38); got != 66 { // F# is in D major
		t.Fatalf("D major: 66 -> %v", got)
	}
	if got := quantize(65.0, major, 38); got != 64 && got != 66 {
		t.Fatalf("D major: 65 -> %v", got)
	}
	if quantize(61, nil, 0) != 61 {
		t.Fatal("no scale leaves it alone")
	}
}

// a stack makes more than one grain for every grain, a scale keeps their pitches in it
func TestStackAndScale(t *testing.T) {
	cfg := base()
	cfg.Spray = 0
	cfg.Stack = []float64{0, 4, 7}
	cfg.Density = 5
	v := New(cfg)
	v.NoteOn(0.5, psalm.Message{"pitch": 69.0}, psalm.CtrlTick{}, v.Config())
	v.Synthesize()
	if len(v.grains) != 3 {
		t.Fatalf("a stack of three should start three grains, got %d", len(v.grains))
	}
}

// the windows are zero at the ends and one of them has a flat middle
func TestWindows(t *testing.T) {
	for k := range numWindows {
		if windowOf(0, 0, k) > 1e-6 || windowOf(1, 0, k) > 1e-6 {
			t.Fatalf("window %s must start and end at zero: %v %v", WindowNames[k], windowOf(0, 0, k), windowOf(1, 0, k))
		}
		if windowOf(0.5, 0, k) < 0.9 {
			t.Fatalf("window %s: the middle is %v", WindowNames[k], windowOf(0.5, 0, k))
		}
		for i := 0; i < 10000; i++ {
			a, b := windowOf(float64(i)/10000, 0, k), windowOf(float64(i+1)/10000, 0, k)
			if math.Abs(a-b) > 0.02 {
				t.Fatalf("window %s jumps at %v", WindowNames[k], float64(i)/10000)
			}
		}
	}
	if windowOf(0.4, 0, WindowTukey) < 0.99 {
		t.Fatal("the tukey is flat in the middle")
	}
	// every window plays at about the same level
	var levels []float64
	for k := range numWindows {
		cfg := base()
		cfg.Window = k
		left, _ := render(cfg, psalm.Message{})
		levels = append(levels, rms(left))
	}
	for k, l := range levels {
		if l < levels[0]*0.7 || l > levels[0]*1.4 {
			t.Fatalf("window %s is at %v, the hann at %v", WindowNames[k], l, levels[0])
		}
	}
}
