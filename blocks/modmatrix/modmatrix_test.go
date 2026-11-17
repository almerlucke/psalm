package modmatrix

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
)

var testDests = []Dest{{Key: "cutoff", Label: "cutoff", Span: 6}, {Key: "level", Label: "level", Span: 1}}

func TestMatrixSumsSlots(t *testing.T) {
	m := New(testDests)
	if m.Active() || m.Mod(0) != 0 {
		t.Fatal("an empty matrix does something")
	}
	m.Set(0, Slot{Src: LFO1, Dest: "cutoff", Amount: 0.5})
	m.Set(1, Slot{Src: Velocity, Dest: "cutoff", Amount: -1})
	m.Set(2, Slot{Src: Wheel, Dest: "nothing", Amount: 1}) // no such destination: empty
	if !m.Active() || !m.Uses(LFO1) || m.Uses(Wheel) || !m.Targets(0) || m.Targets(1) {
		t.Fatalf("in use: active %v lfo %v wheel %v", m.Active(), m.Uses(LFO1), m.Uses(Wheel))
	}
	m.Values[LFO1], m.Values[Velocity] = 1, 0.25
	m.Eval(44100)
	if got, want := m.Mod(0), (0.5*1-1*0.25)*6; math.Abs(got-want) > 1e-12 {
		t.Errorf("cutoff moved by %g, want %g", got, want)
	}
	if m.Mod(1) != 0 {
		t.Errorf("level moved by %g", m.Mod(1))
	}
	m.Set(0, Slot{})
	m.Set(1, Slot{})
	if m.Active() {
		t.Error("emptied slots still do something")
	}
}

func TestMatrixMessages(t *testing.T) {
	m := New(testDests)
	m.Receive(psalm.Message{"mod2Src": int(Wheel), "mod2Dest": 2, "mod2Amount": 0.75})
	if s := m.Slot(1); s.Src != Wheel || s.Dest != "level" || s.Amount != 0.75 {
		t.Errorf("slot %+v", s)
	}
	m.Receive(psalm.Message{"mod2Dest": 0})
	if s := m.Slot(1); s.Dest != "" || s.Src != Wheel {
		t.Errorf("slot %+v after none", s)
	}
	m.Receive(psalm.Message{"ext3": 0.4})
	if m.Ext(2) != 0.4 {
		t.Errorf("ext 3 at %g", m.Ext(2))
	}
}

// the cabled values glide to where they are moved to, and reach it
func TestMatrixExtGlides(t *testing.T) {
	m := New(testDests)
	m.Set(0, Slot{Src: Ext2, Dest: "level", Amount: 1})
	m.SetExt(1, 1)
	m.Eval(44100)
	if v := m.Mod(1); v <= 0 || v >= 0.5 {
		t.Errorf("first sample of a glide at %g", v)
	}
	for range 4410 {
		m.Eval(44100)
	}
	if v := m.Mod(1); v != 1 {
		t.Errorf("ended at %g, want 1", v)
	}
}

func TestMatrixDescribe(t *testing.T) {
	m := New(testDests)
	ps := m.Describe("Mod")
	if len(ps) != Slots*3+Exts {
		t.Fatalf("%d settings", len(ps))
	}
	for _, p := range ps {
		if p.Group != "Mod" || p.Min >= p.Max {
			t.Errorf("%s: %+v", p.Key, p)
		}
	}
	if d := ps[1]; len(d.Options) != len(testDests)+1 || d.Max != float64(len(testDests)) {
		t.Errorf("destinations %v, max %g", d.Options, d.Max)
	}
}

func TestMatrixReceiveDoesNotAllocate(t *testing.T) {
	m := New(testDests)
	msg := psalm.Message{"note": 60}
	if n := testing.AllocsPerRun(100, func() { m.Receive(msg) }); n != 0 {
		t.Fatalf("Receive allocates %v times", n)
	}
}
