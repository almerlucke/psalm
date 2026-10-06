package render

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/effects/granulator"
	"github.com/almerlucke/psalm/modules/effects/pingpong"
)

// clicks plays a click at the given sample positions
type clicks struct {
	*psalm.Module
	at []int
	n  int
}

func newClicks(at ...int) *clicks {
	c := &clicks{Module: psalm.NewModule(nil, []int{psalm.Stereo}), at: at}
	c.SetSelf(c)
	return c
}

func (c *clicks) Synthesize() {
	out := c.Output(0).Channels
	out.Clear()
	for i := range out[0] {
		for _, a := range c.at {
			if c.n == a {
				out[0][i], out[1][i] = 1, 1
			}
		}
		c.n++
	}
}

// counter passes its input and counts how often it works
type counter struct {
	*psalm.Module
	runs int
}

func (c *counter) Synthesize() {
	c.runs++
	c.Output(0).Channels.Copy(c.Input(0).Channels)
}

func (c *counter) Tail() float64 { return 0.1 }

// render runs a patch and returns the left output
func renderLeft(p *psalm.Psalm, seconds float64) []float64 {
	var out []float64
	Run(p, Options{Seconds: seconds}, func(ch psalm.Channels) {
		for _, x := range ch[0] {
			out = append(out, float64(x))
		}
	})
	return out
}

// A module with a tail rests while its input and output are silent, and wakes up when sound comes
func TestSilentModulesRest(t *testing.T) {
	p := psalm.New(nil)
	src := newClicks(0, 2*44100).AddTo(p)
	c := &counter{Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo})}
	c.SetSelf(c)
	c.AddTo(p)
	c.In(src)
	p.Out(c)
	out := renderLeft(p, 3)

	buffers := int(3 * 44100 / 1024)
	if c.runs > buffers/3 {
		t.Fatalf("expected the module to rest while silent, it worked %d of %d buffers", c.runs, buffers)
	}
	if out[2*44100] != 1 {
		t.Fatal("the module should wake up for the second click")
	}
}

// A delay keeps working through the silence between its echoes
func TestDelayKeepsItsEchoes(t *testing.T) {
	p := psalm.New(nil)
	src := newClicks(0).AddTo(p)
	pp := pingpong.New(pingpong.Config{TimeMs: 900, Feedback: 0.5, Mix: 1, Stereo: 1})
	pp.AddTo(p)
	pp.In(src)
	p.Out(pp)
	out := renderLeft(p, 2.5)
	for _, at := range []float64{0.9, 1.8} {
		peak := 0.0
		for i := int(at*44100) - 5; i < int(at*44100)+5; i++ {
			peak = math.Max(peak, math.Abs(out[i]))
		}
		if peak < 0.2 {
			t.Fatalf("expected an echo at %v s, peak %v", at, peak)
		}
	}
}

// A frozen granulator plays its buffer with a silent input
func TestFrozenGranulatorKeepsPlaying(t *testing.T) {
	p := psalm.New(nil)
	src := newClicks(0, 700, 1400, 2100, 2800, 3500, 4200, 4900, 5600, 6300, 7000, 7700, 8400, 9100, 9800, 10500, 11200, 11900, 12600).AddTo(p) // clicks through the first 0.3 s
	cfg := granulator.DefaultConfig()
	cfg.Mix = 1
	cfg.Delay = 50
	cfg.Spray = 20
	g := granulator.New(cfg)
	g.AddTo(p)
	g.In(src)
	p.Out(g)
	Run(p, Options{Seconds: 0.35}, nil) // record the clicks
	g.SetFreeze(true)
	out := renderLeft(p, 3)
	late := 0.0
	for _, x := range out[2*44100:] {
		late = math.Max(late, math.Abs(x))
	}
	if late < 1e-3 {
		t.Fatalf("a frozen granulator should keep playing, peak %v", late)
	}
}
