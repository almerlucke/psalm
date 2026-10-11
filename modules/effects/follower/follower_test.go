package follower

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

// feed runs a sine of an amplitude for some seconds into a module with a stereo input 0
func feed(m psalm.Modular, amp, seconds float64) {
	bs, sr := m.Config().BufferSize, m.Config().SampleRate
	n := 0
	for range int(seconds * sr / float64(bs)) {
		for i := range bs {
			x := float32(amp * math.Sin(2*math.Pi*220*float64(n)/sr))
			m.Input(0).Channels[0][i], m.Input(0).Channels[1][i] = x, x
			n++
		}
		m.Synthesize()
	}
}

type values struct {
	*psalm.Module
	got []float64
}

func (v *values) Receive(x any, _ int) { v.got = append(v.got, x.(float64)) }

func TestFollower(t *testing.T) {
	psalm.New(nil)
	f := New(Config{Attack: 5, Release: 100, Gain: 0, Min: 200, Max: 2000})
	rec := &values{Module: psalm.NewControl()}
	rec.SetSelf(rec)
	rec.ValueIn(f)
	feed(f, 0.5, 0.5)
	if l := f.Level(); math.Abs(l-0.5) > 0.07 { // just under the peaks: it falls a little between them
		t.Fatalf("it follows the level: %.3f", l)
	}
	f.CtrlTick(psalm.CtrlTick{})
	if v := rec.got[len(rec.got)-1]; math.Abs(v-1100) > 130 {
		t.Fatalf("half the level is half the range: %.0f", v)
	}
	feed(f, 0, 0.05)
	if l := f.Level(); l < 0.2 || l > 0.45 {
		t.Fatalf("it lets go over the release: %.3f after 50 ms", l)
	}
	if f.Output(0).Channels[0][10] != f.Input(0).Channels[0][10] {
		t.Fatal("the sound passes through")
	}
}

func TestWah(t *testing.T) {
	psalm.New(nil)
	w := NewWah(WahConfig{Kind: BandPass, Min: 300, Max: 3000, Gain: 9, Attack: 2, Release: 50, Mix: 1})
	feed(w, 0.01, 0.3)
	quiet := w.Frequency()
	feed(w, 0.5, 0.3)
	loud := w.Frequency()
	if quiet > 400 || loud < 2900 {
		t.Fatalf("quiet it stays low, loud it sweeps up: %.0f Hz, %.0f Hz", quiet, loud)
	}
	down := NewWah(WahConfig{Kind: LowPass, Min: 300, Max: 3000, Gain: 6, Down: true, Mix: 1})
	feed(down, 0.5, 0.3)
	if f := down.Frequency(); f > 400 {
		t.Fatalf("down: loud sweeps it down: %.0f Hz", f)
	}
}
