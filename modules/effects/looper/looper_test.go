package looper

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/almerlucke/psalm"
)

// rig drives a looper as the engine does: a control tick for every tick that begins in a buffer, then the buffer
type rig struct {
	l       *Looper
	t       *testing.T
	samples int64   // the samples played
	acc     float64 // the samples into the tick
	tick    int64
	in      func(n int64) float32
}

func newRig(t *testing.T, bars int) *rig {
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	t.Cleanup(func() { psalm.PopConfig() })
	_ = psalm.New(nil)
	cfg := DefaultConfig()
	cfg.Bars = bars
	return &rig{l: New(cfg), t: t, in: func(int64) float32 { return 0 }}
}

// run plays n samples (whole buffers) and returns the output of the left channel
func (r *rig) run(n int) []float32 {
	bs := r.l.Config().BufferSize
	perTick := 44100.0 * 60 / (120 * 240)
	var out []float32
	for range n / bs {
		for i := range bs {
			r.acc++
			for r.acc >= perTick {
				r.acc -= perTick
				r.l.CtrlTick(psalm.CtrlTick{SampleTimestamp: r.samples, TickTimestamp: r.tick, SampleOffset: i})
				r.tick++
			}
			v := r.in(r.samples + int64(i))
			r.l.Input(0).Channels[0][i], r.l.Input(0).Channels[1][i] = v, -v
		}
		r.samples += int64(bs)
		r.l.Synthesize()
		out = append(out, r.l.Output(0).Channels[0]...)
	}
	return out
}

// a layer recorded in one pass plays in the next, in the place it was played; the sound that comes in goes on as it is
func TestLoopRecordsAndPlays(t *testing.T) {
	r := newRig(t, 1)
	r.in = func(n int64) float32 { return float32(math.Sin(float64(n)*0.01)) * 0.5 }
	r.l.ReceiveMessage(psalm.Message{"record": 1.0})
	first := r.run(300 * 256)
	if first[1000] != r.in(1000) { // the dry sound passes while recording
		t.Fatalf("dry %v, in %v", first[1000], r.in(1000))
	}
	r.l.ReceiveMessage(psalm.Message{"record": 0.0})
	r.run(60 * 256) // the pass ends: the layer is done
	if n, rec := r.l.Layers(); n != 1 || rec {
		t.Fatalf("%d layers, recording %v", n, rec)
	}
	// silence in: the pass comes back as it was recorded
	r.in = func(int64) float32 { return 0 }
	second := r.run(345 * 256)
	// the second pass began at sample 88200 (tick 960); the one recorded at sample 0
	var got, want float64
	base := int(r.samples) - len(second)
	for n := 0; n < 80000; n++ {
		at := 88200 + n - base
		if at < 0 || at >= len(second) {
			continue
		}
		got += math.Abs(float64(second[at]))
		want += math.Abs(float64(r.in2(int64(n))))
	}
	if want == 0 || math.Abs(got-want)/want > 0.001 {
		t.Errorf("the loop is not what was played: %v against %v", got, want)
	}
}

func (r *rig) in2(n int64) float32 { return float32(math.Sin(float64(n)*0.01)) * 0.5 }

// layers are taken away alone, by undo and by clear, and the page's view follows
func TestLoopLayers(t *testing.T) {
	r := newRig(t, 1)
	r.in = func(n int64) float32 { return 0.25 }
	record := func() {
		r.l.ReceiveMessage(psalm.Message{"record": 1.0})
		r.run(256)
		for r.l.recording == false { // wait for the pass to begin
			r.run(256)
		}
		r.l.ReceiveMessage(psalm.Message{"record": 0.0})
		for r.l.recording {
			r.run(256)
		}
	}
	for range 3 {
		record()
	}
	if n, _ := r.l.Layers(); n != 3 {
		t.Fatalf("%d layers", n)
	}
	st := r.l.State().(State)
	if len(st.Layers) != 3 || st.Bars != 1 {
		t.Fatalf("state %+v", st)
	}
	mid := st.Layers[1].ID
	r.l.Drop(mid)
	r.run(256)
	if st = r.l.State().(State); len(st.Layers) != 2 || st.Layers[0].ID == mid || st.Layers[1].ID == mid {
		t.Fatalf("after taking one away: %+v", st.Layers)
	}
	var released []string
	r.l.OnRelease(func(k string) { released = append(released, k) })
	r.l.ReceiveMessage(psalm.Message{"undo": 1.0})
	r.run(256)
	if n, _ := r.l.Layers(); n != 1 {
		t.Errorf("after undo %d layers", n)
	}
	r.l.ReceiveMessage(psalm.Message{"clear": 1.0})
	r.run(256)
	if n, _ := r.l.Layers(); n != 0 || len(released) != 2 || released[0] != "undo" || released[1] != "clear" {
		t.Errorf("after clear %d layers, released %v", n, released)
	}
	// the peaks of a layer are measured (0.25)
	r2 := newRig(t, 1)
	r2.in = func(int64) float32 { return 0.25 }
	r2.l.ReceiveMessage(psalm.Message{"record": 1.0})
	r2.run(256)
	r2.l.ReceiveMessage(psalm.Message{"record": 0.0})
	for r2.l.recording || len(*r2.l.snap.Load()) == 0 {
		r2.run(256)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if st := r2.l.State().(State); len(st.Layers) == 1 && len(st.Layers[0].Peaks) == Columns {
			if p := st.Layers[0].Peaks[Columns/2]; math.Abs(float64(p)-0.25) > 1e-6 {
				t.Errorf("peak %v", p)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("the peaks were not measured")
}

// export writes a file for every layer, or one with them together
func TestLoopExports(t *testing.T) {
	for _, one := range []bool{false, true} {
		r := newRig(t, 1)
		dir := t.TempDir()
		r.l.SetDir(dir)
		said := make(chan string, 64)
		r.l.OnEvent(func(s string) { said <- s })
		r.in = func(int64) float32 { return 0.1 }
		for range 2 {
			r.l.ReceiveMessage(psalm.Message{"record": 1.0})
			r.run(256)
			for !r.l.recording {
				r.run(256)
			}
			r.l.ReceiveMessage(psalm.Message{"record": 0.0})
			for r.l.recording {
				r.run(256)
			}
		}
		if one {
			r.l.ReceiveMessage(psalm.Message{"files": 1.0})
		}
		r.l.ReceiveMessage(psalm.Message{"export": 1.0})
		r.run(256)
		want := 2
		if one {
			want = 1
		}
		deadline := time.After(3 * time.Second)
		saved := 0
		for saved < want {
			select {
			case s := <-said:
				if strings.Contains(s, "saved ") {
					saved++
				}
			case <-deadline:
				t.Fatalf("one %v: %d files written, want %d", one, saved, want)
			}
		}
		all, _ := filepath.Glob(filepath.Join(dir, "*.wav"))
		layers, _ := filepath.Glob(filepath.Join(dir, "loop-*-layer*.wav"))
		if len(all) != want || (one && len(layers) != 0) || (!one && len(layers) != 2) {
			t.Errorf("one %v: files %v", one, all)
		}
		if fi, err := os.Stat(all[0]); err != nil || fi.Size() < 88200*2*4 {
			t.Errorf("file %v %v", fi, err)
		}
	}
}

func TestLoopDescribes(t *testing.T) {
	l := New(DefaultConfig())
	if n := len(Describe(l.Settings()).Params); n != 10 {
		t.Errorf("%d settings", n)
	}
	l.ReceiveMessage(psalm.Message{"bars": 4, "level": 0.5})
	if s := l.Settings(); s.Bars != 4 || s.Level != 0.5 {
		t.Errorf("settings %+v", s)
	}
}

// a layer can be played earlier than it was recorded, to make up for the sound card: what was late comes in time, and the end comes round to the beginning
func TestLoopPlaysEarly(t *testing.T) {
	r := newRig(t, 1)
	r.in = func(n int64) float32 { return float32(n%1000) / 1000 }
	r.l.ReceiveMessage(psalm.Message{"record": 1.0, "dry": 0.0, "early": 10.0}) // 441 samples
	r.run(300 * 256)
	r.l.ReceiveMessage(psalm.Message{"record": 0.0})
	r.run(60 * 256)
	r.in = func(int64) float32 { return 0 }
	second := r.run(345 * 256)
	base := int(r.samples) - len(second)
	for _, n := range []int{5000, 20000, 60000} {
		at := 88200 + n - base
		want := float32((n+441)%1000) / 1000
		if got := second[at]; math.Abs(float64(got-want)) > 1e-6 {
			t.Errorf("sample %d: %v, want %v", n, got, want)
		}
	}
}
