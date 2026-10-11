package tremolo

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

// ones is a source playing 1 on both sides
type ones struct{ *psalm.Module }

func (o *ones) Synthesize() {
	for _, ch := range o.Output(0).Channels {
		for i := range ch {
			ch[i] = 1
		}
	}
}

// play runs a tremolo on a steady sound in a patch for some seconds and returns left and right
func play(cfg Config, seconds float64) ([]float64, []float64, *psalm.Psalm) {
	p := psalm.New(nil)
	src := &ones{psalm.NewModule(nil, []int{psalm.Stereo})}
	src.SetSelf(src)
	src.AddTo(p)
	tr := New(cfg)
	tr.AddTo(p)
	tr.In(src)
	p.Out(tr)
	var l, r []float64
	for range int(seconds * p.Config().SampleRate / float64(p.Config().BufferSize)) {
		p.ResetRun()
		p.Run()
		for i := range p.Output(0).Channels[0] {
			l = append(l, float64(p.Output(0).Channels[0][i]))
			r = append(r, float64(p.Output(0).Channels[1][i]))
		}
	}
	return l, r, p
}

func TestTremoloDepth(t *testing.T) {
	l, _, _ := play(Config{Rate: 5, Depth: 1}, 1)
	lo, hi := 1.0, 0.0
	for _, x := range l {
		lo, hi = math.Min(lo, x), math.Max(hi, x)
	}
	if lo > 0.01 || hi < 0.99 {
		t.Fatalf("a full depth tremolo goes from silence to full: %.3f to %.3f", lo, hi)
	}
	l, _, _ = play(Config{Rate: 5, Depth: 0.5}, 1)
	lo = 1
	for _, x := range l {
		lo = math.Min(lo, x)
	}
	if math.Abs(lo-0.5) > 0.01 {
		t.Fatalf("half depth goes down to half: %.3f", lo)
	}
}

func TestSyncedOnTheBeat(t *testing.T) {
	// a cycle a beat: full on the beat, silent halfway
	l, _, p := play(Config{Sync: 1, Depth: 1}, 2)
	beat := 60 / psalm.BPM() * p.Config().SampleRate
	spt := p.SamplesPerTick()
	for k := 1.0; k < 3; k++ {
		on := int(k*beat + spt) // the beat (the clock's first tick comes a tick in)
		half := int((k+0.5)*beat + spt)
		if l[on] < 0.98 || l[half] > 0.02 {
			t.Fatalf("beat %v: full on the beat (%.3f) and silent halfway (%.3f)", k, l[on], l[half])
		}
	}
}

func TestStereoAndPan(t *testing.T) {
	l, r, _ := play(Config{Rate: 2, Depth: 1, Stereo: 0.5}, 1)
	for i := 0; i < len(l); i += 997 {
		if math.Abs(l[i]+r[i]-1) > 0.02 {
			t.Fatalf("half a cycle apart, left and right take turns: %.3f + %.3f", l[i], r[i])
		}
	}
	l, r, _ = play(Config{Mode: Pan, Rate: 1, Depth: 1}, 1)
	if r[0] < 1.4 || l[0] > 0.01 {
		t.Fatalf("full pan at the top of the cycle is all right (+1, as pan and the mixer): %.3f, %.3f", l[0], r[0])
	}
	mid := int(0.25 * psalm.SampleRate()) // a quarter cycle: the middle
	if math.Abs(l[mid]-1) > 0.02 || math.Abs(r[mid]-1) > 0.02 {
		t.Fatalf("in the middle both sides as they were: %.3f, %.3f", l[mid], r[mid])
	}
}
