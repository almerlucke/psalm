package panel

import (
	"math"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/almerlucke/psalm"
)

// recorder keeps the messages and values it receives
type recorder struct {
	*psalm.Module
	got []float64
}

func newRecorder() *recorder {
	r := &recorder{Module: psalm.NewModule(nil, []int{psalm.Stereo})}
	r.SetSelf(r)
	return r
}

func (r *recorder) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		if v, ok := msg.FltVal("cutoff"); ok {
			r.got = append(r.got, v)
		}
		return
	}
	r.got = append(r.got, psalm.FltVal[float64](value))
}

func (r *recorder) last() float64 { return r.got[len(r.got)-1] }

// ticker runs the control ticks of a panel the way a patch does
type ticker struct {
	pn     *Panel
	sample int64
}

func (t *ticker) tick(n int) {
	for range n {
		t.sample += 10
		t.pn.CtrlTick(psalm.CtrlTick{SampleTimestamp: t.sample, TickTimestamp: t.sample / 10})
	}
}

func newPanel() (*Panel, *ticker) {
	pn := New(psalm.New(nil), "Test")
	return pn, &ticker{pn: pn}
}

func TestDefaultAndSet(t *testing.T) {
	pn, tk := newPanel()
	rec := newRecorder()
	cutoff := pn.Fader("Cutoff", 200, 12000, Exp, Message(rec, "cutoff"), Default(1000))
	tk.tick(1)
	if len(rec.got) != 1 || rec.last() != 1000 {
		t.Fatalf("the default should be sent at the start, got %v", rec.got)
	}
	tk.tick(3)
	if len(rec.got) != 1 {
		t.Fatal("nothing should be sent while nothing changes")
	}
	cutoff.Set(50000)
	tk.tick(1)
	if rec.last() != 12000 || cutoff.Value() != 12000 {
		t.Fatalf("a value should be clamped to the range, got %v", rec.last())
	}
}

func TestStepAndCurves(t *testing.T) {
	pn, _ := newPanel()
	notes := pn.Fader("Voices", 1, 16, Linear, Func(func(float64) {}), Step(1))
	notes.Set(4.4)
	if notes.Value() != 4 {
		t.Fatalf("the step should round, got %v", notes.Value())
	}
	freq := pn.Knob("Freq", 100, 6400, Exp, Func(func(float64) {}))
	if math.Abs(freq.At(0.5)-800) > 1e-9 || math.Abs(freq.Position(1600)-2.0/3) > 1e-12 {
		t.Fatalf("an exponential control should move in octaves: at half %v", freq.At(0.5))
	}
	defer func() {
		if recover() == nil {
			t.Fatal("an exponential range through 0 should panic")
		}
	}()
	pn.Fader("Bad", 0, 1, Exp, nil)
}

func TestSmooth(t *testing.T) {
	pn, tk := newPanel()
	rec := newRecorder()
	level := pn.Fader("Level", -60, 0, Linear, Input(rec, 1), Default(-60), Smooth(0.05))
	tk.tick(1)
	level.Set(0)
	tk.tick(1)
	if v := rec.last(); v <= -60 || v >= 0 {
		t.Fatalf("a smoothed control should glide, not jump: %v", v)
	}
	tk.tick(2000)
	if rec.last() != 0 {
		t.Fatalf("the glide should arrive exactly, got %v", rec.last())
	}
	n := len(rec.got)
	tk.tick(10)
	if len(rec.got) != n {
		t.Fatal("nothing should be sent once the glide arrived")
	}
}

func TestButtonsAndSelects(t *testing.T) {
	pn, tk := newPanel()
	pressed := 0
	bang := pn.Button("Bang", func() { pressed++ })
	chosen := -1
	sel := pn.Select("Preset", []string{"a", "b", "c"}, func(i int) { chosen = i })
	tk.tick(1)
	if pressed != 0 || chosen != 0 {
		t.Fatalf("a select should start at its first option and a button not be pressed: %d %d", pressed, chosen)
	}
	bang.Press()
	bang.Press()
	sel.Set(2)
	tk.tick(1)
	if pressed != 2 || chosen != 2 {
		t.Fatalf("both presses should count and the select change: %d %d", pressed, chosen)
	}
}

func TestMeter(t *testing.T) {
	pn, tk := newPanel()
	src := newRecorder()
	m := pn.Meter("Source", src)
	for i := range src.Output(0).Channels[0] {
		src.Output(0).Channels[0][i] = 0.5
		src.Output(0).Channels[1][i] = 0.5
	}
	tk.tick(1)
	peak, rms := m.Levels()
	if math.Abs(peak-(-6.02)) > 0.01 || math.Abs(rms-(-6.02)) > 0.01 {
		t.Fatalf("a constant 0.5 is -6 dB, got peak %v rms %v", peak, rms)
	}
}

func TestSnapshot(t *testing.T) {
	pn, _ := newPanel()
	a := pn.Fader("A", 0, 10, Linear, Func(func(float64) {}), Default(3))
	pn.Toggle("B", Func(func(float64) {}))
	pn.Button("C", func() {})
	snap := pn.Snapshot()
	if len(snap) != 2 || snap["A"] != 3 || snap["B"] != 0 {
		t.Fatalf("unexpected snapshot %v", snap)
	}
	path := filepath.Join(t.TempDir(), "preset.json")
	if err := pn.Save(path); err != nil {
		t.Fatal(err)
	}
	a.Set(7)
	if err := pn.Load(path); err != nil {
		t.Fatal(err)
	}
	if a.Value() != 3 {
		t.Fatalf("loading should restore the value, got %v", a.Value())
	}
}

// Controls are changed from other goroutines while the audio thread ticks (run with -race)
func TestConcurrent(t *testing.T) {
	pn, tk := newPanel()
	rec := newRecorder()
	f := pn.Fader("F", 0, 1, Linear, Input(rec, 1), Smooth(0.01))
	b := pn.Button("B", func() {})
	var wg sync.WaitGroup
	for g := range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 1000 {
				f.Set(float64((i+g)%100) / 100)
				b.Press()
				_ = pn.Snapshot()
			}
		}()
	}
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				tk.tick(1)
			}
		}
	}()
	wg.Wait()
	close(done)
}

func TestDuplicateNamePanics(t *testing.T) {
	pn, _ := newPanel()
	pn.Toggle("X", nil)
	defer func() {
		if recover() == nil {
			t.Fatal("two controls with the same name should panic")
		}
	}()
	pn.Toggle("X", nil)
}

// Place puts a control at a value the page shows without sending it, and drops what was asked
// for but not yet sent (also with a glide)
func TestPlace(t *testing.T) {
	pn, tk := newPanel()
	rec := newRecorder()
	level := pn.Fader("Level", 0, 1, Linear, Input(rec, 1), Default(0.2), Smooth(0.05))
	tk.tick(1)
	n := len(rec.got)
	level.Set(0.9)
	level.Place(0.5)
	tk.tick(50)
	if level.Value() != 0.5 || len(rec.got) != n {
		t.Fatalf("placed at %v, sent %d more", level.Value(), len(rec.got)-n)
	}
	level.Set(0.6) // moves from where it was placed
	tk.tick(1)
	if len(rec.got) == n {
		t.Error("a control moved after being placed should send")
	}
}

// the input device of an engine that can choose: a select of the devices, the one in use shown, the choice passed to the engine (off the audio thread)
type fakeInputs struct {
	devices []string
	now     string
	chosen  chan string
}

func (f *fakeInputs) InputDevices() []string { return f.devices }
func (f *fakeInputs) InputDevice() string    { return f.now }
func (f *fakeInputs) SetInputDevice(name string) error {
	f.now = name
	f.chosen <- name
	return nil
}

func TestInputSelect(t *testing.T) {
	pn, tk := newPanel()
	f := &fakeInputs{devices: []string{"default", "headset", "built-in"}, now: "headset", chosen: make(chan string, 1)}
	sel := pn.InputSelect(f)
	if sel == nil || sel.Value() != 1 {
		t.Fatalf("the select shows the device in use: %v", sel)
	}
	tk.tick(1)
	select {
	case got := <-f.chosen:
		t.Fatalf("the device in use was chosen again: %q", got)
	case <-time.After(30 * time.Millisecond):
	}
	sel.Set(2)
	tk.tick(1)
	select {
	case got := <-f.chosen:
		if got != "built-in" {
			t.Errorf("chose %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("the engine was not asked")
	}
	if pn.InputSelect(struct{}{}) != nil || pn.InputSelect(&fakeInputs{devices: []string{"default"}}) != nil {
		t.Error("a select where there is no choice")
	}
}
