package psalm

import (
	"sync"
	"testing"
)

// fakeEngine counts starts and stops, and fails when it is started twice (as JACK does)
type fakeEngine struct {
	mu            sync.Mutex
	started       bool
	starts, stops int
	twice         bool
}

func (e *fakeEngine) GetSampleRate() float64                { return 44100 }
func (e *fakeEngine) GetBufferSize() int                    { return 64 }
func (e *fakeEngine) SetCallback(func(in, out [][]float32)) {}
func (e *fakeEngine) Init() error                           { return nil }
func (e *fakeEngine) Close() error                          { return nil }
func (e *fakeEngine) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.started {
		e.twice = true
	}
	e.started = true
	e.starts++
	return nil
}
func (e *fakeEngine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.started = false
	e.stops++
	return nil
}

func TestAudioStartStop(t *testing.T) {
	e := &fakeEngine{}
	p := New(e)
	if !p.Live() || p.Running() {
		t.Fatal("a patch with an engine is live and starts stopped")
	}
	var wg sync.WaitGroup
	for range 8 { // from several goroutines at once: a panel's page and the main goroutine
		wg.Add(2)
		go func() { defer wg.Done(); _ = p.AudioStart() }()
		go func() { defer wg.Done(); _ = p.AudioStart() }()
	}
	wg.Wait()
	if !p.Running() || e.starts != 1 || e.twice {
		t.Fatalf("starting a patch that plays should do nothing: %d starts", e.starts)
	}
	_ = p.AudioStop()
	_ = p.AudioStop()
	if p.Running() || e.stops != 1 {
		t.Fatalf("stopping a stopped patch should do nothing: %d stops", e.stops)
	}
	if New(nil).Live() {
		t.Fatal("a patch without an engine renders offline")
	}
}

// ticker keeps the ticks it gets, and counts its restarts
type ticker struct {
	*Module
	ticks    []int64
	restarts int
}

func (t *ticker) CtrlTick(tick CtrlTick) { t.ticks = append(t.ticks, tick.TickTimestamp) }
func (t *ticker) Restart()               { t.restarts++; t.ticks = t.ticks[:0] }

func TestRestart(t *testing.T) {
	p := New(nil)
	top := &ticker{Module: NewControl()}
	top.SetSelf(top)
	top.AddTo(p)
	sub := NewPatch(nil, nil)
	sub.AddTo(p)
	inner := &ticker{Module: NewControl()}
	inner.SetSelf(inner)
	inner.AddTo(sub)
	for range 10 {
		p.ResetRun()
		p.Run()
	}
	p.Restart()
	if top.restarts != 0 {
		t.Fatal("a restart should wait for the next buffer")
	}
	p.ResetRun()
	p.Run()
	if top.restarts != 1 || inner.restarts != 1 {
		t.Fatalf("every module keeping time should restart, in patches within too: %d %d", top.restarts, inner.restarts)
	}
	if len(top.ticks) == 0 || top.ticks[0] != 1 || len(top.ticks) != len(inner.ticks) {
		t.Fatalf("the clock should start again at the first tick: %v", top.ticks[:min(3, len(top.ticks))])
	}
}
