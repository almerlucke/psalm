package psalm

import (
	"sync"
	"testing"
)

// source outputs a constant, sink sums its input; both count what they receive
type testModule struct {
	*Module
	level    float32
	got      []any
	ticks    int
	received []int
}

func newTest(ins, outs int, level float32) *testModule {
	m := &testModule{Module: NewModule(NumSockets(ins, 1), NumSockets(outs, 1)), level: level}
	m.SetSelf(m)
	return m
}

func (m *testModule) Synthesize() {
	for _, out := range m.outputs {
		for i := range out.Channels[0] {
			out.Channels[0][i] = m.level
		}
	}
	if len(m.inputs) > 0 && len(m.outputs) > 0 { // pass the input on, plus the level
		for i, x := range m.inputs[0].Channels[0] {
			m.outputs[0].Channels[0][i] += x
		}
	}
}

func (m *testModule) Receive(v any, i int) {
	m.got = append(m.got, v)
	m.received = append(m.received, i)
}
func (m *testModule) CtrlTick(CtrlTick) { m.ticks++ }

func first(m Modular) float32 { return m.Output(0).Channels[0][0] }

func run(p *Psalm) {
	p.ResetRun()
	p.Run()
}

func TestDisconnect(t *testing.T) {
	p := New(nil)
	a, b := newTest(0, 1, 0.25), newTest(0, 1, 0.5)
	sink := newTest(1, 1, 0)
	for _, m := range []Modular{a, b, sink} {
		m.AddTo(p)
	}
	sink.In(a, b, 0, 0) // both into input 0
	p.Out(sink)
	run(p)
	if first(sink) != 0.75 {
		t.Fatalf("the sink should sum both, got %v", first(sink))
	}
	if !a.Disconnect(0, sink, 0) || a.Disconnect(0, sink, 0) {
		t.Fatal("disconnect should tell whether there was a connection")
	}
	run(p)
	if first(sink) != 0.5 || len(a.Output(0).Connections) != 0 || len(sink.Input(0).Connections) != 1 {
		t.Fatalf("a disconnected source should be gone at both ends, got %v", first(sink))
	}
	if a.Disconnect(3, sink, 0) || a.Disconnect(0, sink, 7) {
		t.Fatal("sockets out of range: nothing to disconnect")
	}
}

func TestValueDisconnect(t *testing.T) {
	New(nil)
	from, to := newTest(0, 0, 0), newTest(0, 0, 0)
	to.ValueIn(from, 0, 1)
	from.Send(1.0, 0)
	if !from.ValueDisconnect(0, to, 1) || from.ValueDisconnect(0, to, 1) {
		t.Fatal("value disconnect should tell whether there was a connection")
	}
	from.Send(2.0, 0)
	if len(to.got) != 1 || len(to.valueInputs) != 0 {
		t.Fatalf("a value disconnected should not arrive: %v", to.got)
	}
}

func TestDisconnectAll(t *testing.T) {
	p := New(nil)
	src, mid, sink := newTest(0, 1, 0.25), newTest(1, 1, 0), newTest(1, 1, 0)
	ctl, listener := newTest(0, 0, 0), newTest(0, 0, 0)
	for _, m := range []Modular{src, mid, sink, ctl, listener} {
		m.AddTo(p)
	}
	mid.In(src)
	sink.In(mid)
	mid.ValueIn(ctl)      // values to it
	listener.ValueIn(mid) // values from it
	p.Out(sink)
	mid.DisconnectAll()
	run(p)
	ctl.Send(1.0, 0)
	mid.Send(1.0, 0)
	if first(sink) != 0 || len(src.Output(0).Connections) != 0 || len(mid.got) != 0 || len(listener.got) != 0 {
		t.Fatal("disconnect all should leave the module on its own, every other end too")
	}
}

func TestRemove(t *testing.T) {
	p := New(nil)
	src, sink := newTest(0, 1, 0.25), newTest(1, 1, 0)
	src.AddTo(p)
	sink.AddTo(p)
	sink.In(src)
	p.Out(sink)
	run(p)
	ticks := src.ticks
	if !p.Remove(src) || p.Remove(src) {
		t.Fatal("remove should tell whether the module was in the patch")
	}
	run(p)
	if src.ticks != ticks || first(sink) != 0 {
		t.Fatal("a removed module gets no ticks and no longer sounds")
	}

	// a patch within the patch keeps its insides: out and back in
	sub := NewPatch(nil, []int{1})
	inner := newTest(0, 1, 0.5)
	inner.AddTo(sub)
	sub.Out(inner)
	sub.AddTo(p)
	sink.In(sub)
	run(p)
	if first(sink) != 0.5 {
		t.Fatalf("the sub patch should sound, got %v", first(sink))
	}
	p.Remove(sub)
	run(p)
	if first(sink) != 0 {
		t.Fatal("a removed sub patch should be silent")
	}
	sub.AddTo(p)
	sink.In(sub)
	run(p)
	if first(sink) != 0.5 || inner.ticks == 0 {
		t.Fatal("a sub patch added again should sound again, its insides intact")
	}
}

func TestDo(t *testing.T) {
	e := &fakeEngine{}
	p := New(e)
	done := false
	p.Do(func() { done = true })
	if !done {
		t.Fatal("while stopped a change is made at once")
	}
	_ = p.AudioStart()
	var order []int
	p.Do(func() { order = append(order, 1) })
	p.Do(func() { order = append(order, 2) })
	if len(order) != 0 {
		t.Fatal("while playing a change waits for the audio thread")
	}
	run(p)
	if len(order) != 2 || order[0] != 1 {
		t.Fatalf("the changes should be made in order at the next buffer: %v", order)
	}
	p.Do(func() { order = append(order, 3) })
	_ = p.AudioStop()
	if len(order) != 3 {
		t.Fatal("stopping should make the changes still waiting")
	}

	// from other goroutines while the audio runs
	_ = p.AudioStart()
	src := newTest(0, 1, 0.25)
	sink := newTest(1, 1, 0)
	sink.AddTo(p)
	p.Out(sink)
	var wg sync.WaitGroup
	stop, stopped := make(chan struct{}), make(chan struct{})
	go func() { // the audio thread
		defer close(stopped)
		for {
			select {
			case <-stop:
				return
			default:
				run(p)
			}
		}
	}()
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.Do(func() { src.AddTo(p); sink.In(src) })
			p.Do(func() { p.Remove(src) })
		}()
	}
	wg.Wait()
	close(stop)
	<-stopped // an engine calls no more after it stops
	_ = p.AudioStop()
	if len(sink.Input(0).Connections) != 0 {
		t.Fatal("every change should have been made")
	}
}
