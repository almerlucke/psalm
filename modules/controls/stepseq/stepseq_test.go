package stepseq

import (
	"math"
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
)

// recorder is a target keeping what it plays: the time (in ticks, with the sample offset) and the
// amplitude of every hit, and the chokes
type recorder struct {
	times, amps, pitches []float64
	chokes               int
	samplesPerTick       float64
	tick                 *int64
}

func (r *recorder) play(amp, pitch, _ float64, tick psalm.CtrlTick) {
	r.times = append(r.times, float64(*r.tick)+float64(tick.SampleOffset)/r.samplesPerTick)
	r.amps = append(r.amps, amp)
	r.pitches = append(r.pitches, pitch)
}

func (r *recorder) choke(psalm.CtrlTick) { r.chokes++ }

type rig struct {
	p    *psalm.Psalm
	ctl  *panel.Panel
	grid *panel.Grid
	seq  *Sequencer
	tick int64
	tps  float64 // ticks per step
	recs map[string]*recorder
}

func newRig(t *testing.T, cfg Config, rows ...string) *rig {
	t.Helper()
	p := psalm.New(nil)
	ctl := panel.New(p, "Test")
	r := &rig{p: p, ctl: ctl, grid: NewGrid(ctl, "Beat", rows, 16, 2), recs: map[string]*recorder{}}
	cfg.Seed = 1
	r.seq = New(p, r.grid, cfg)
	r.tps = float64(psalm.PPQ()) / r.seq.cfg.StepsPerBeat
	spt := r.seq.Env().TickDuration() * r.seq.Config().SampleRate
	for _, name := range rows {
		rec := &recorder{samplesPerTick: spt, tick: &r.tick}
		r.recs[name] = rec
		r.seq.Row(name, rec)
	}
	return r
}

// run plays a number of steps
func (r *rig) run(steps float64) {
	end := r.tick + int64(steps*r.tps)
	for ; r.tick < end; r.tick++ {
		r.seq.CtrlTick(psalm.CtrlTick{TickTimestamp: r.tick + 1})
	}
}

func (r *rig) on(row string, steps ...int) {
	for _, s := range steps {
		r.grid.Set(r.grid.Pattern(), r.grid.Row(row), s, Velocity, 0.8)
	}
}

func (r *rig) set(row string, step, field int, v float64) {
	r.grid.Set(r.grid.Pattern(), r.grid.Row(row), step, field, v)
}

func near(a, b float64) bool { return math.Abs(a-b) < 0.01 }

func TestStepsAndLevel(t *testing.T) {
	r := newRig(t, Config{}, "kick")
	r.on("kick", 0, 4, 8, 12)
	r.seq.SetLevel("kick", -6.0206)
	r.run(16)
	rec := r.recs["kick"]
	if len(rec.times) != 4 {
		t.Fatalf("expected 4 hits, got %d", len(rec.times))
	}
	for i, at := range rec.times {
		if !near(at, float64(i*4)*r.tps) || !near(rec.amps[i], 0.4) {
			t.Fatalf("hit %d at %v (%v), expected %v at half of 0.8", i, at, rec.amps[i], float64(i*4)*r.tps)
		}
	}
}

func TestChance(t *testing.T) {
	r := newRig(t, Config{}, "hat")
	for s := range 16 {
		r.on("hat", s)
		r.set("hat", s, Chance, 0.5)
	}
	r.run(16 * 20)
	if n := len(r.recs["hat"].times); n < 120 || n > 200 {
		t.Fatalf("half of 320 steps should play, %d did", n)
	}
	r.set("hat", 0, Chance, 0)
}

func TestRoll(t *testing.T) {
	r := newRig(t, Config{}, "snare")
	r.on("snare", 2)
	r.set("snare", 2, Roll, 4)
	r.set("snare", 2, RollRamp, 1)
	r.run(16)
	rec := r.recs["snare"]
	if len(rec.times) != 4 {
		t.Fatalf("a roll of 4 should play 4 hits, got %d", len(rec.times))
	}
	for i := range 4 {
		if !near(rec.times[i], (2+float64(i)/4)*r.tps) {
			t.Fatalf("roll hit %d at %v", i, rec.times[i])
		}
	}
	if rec.amps[0] != 0 || !near(rec.amps[3], 0.8) || rec.amps[1] >= rec.amps[2] {
		t.Fatalf("a ramp up should rise to the velocity: %v", rec.amps)
	}

	r.set("snare", 2, RollChance, 0)
	n := len(rec.times)
	r.run(16)
	if len(rec.times)-n != 1 {
		t.Fatal("a roll that doesn't happen is a single hit")
	}
}

func TestNudgeAndSwing(t *testing.T) {
	r := newRig(t, Config{Swing: 0.66}, "a", "b")
	r.on("a", 4)
	r.set("a", 4, Nudge, -0.25)
	r.on("b", 3)
	r.run(16)
	if at := r.recs["a"].times[0]; !near(at, 3.75*r.tps) {
		t.Fatalf("a step nudged early should play early, at %v not %v", at, 3.75*r.tps)
	}
	if at := r.recs["b"].times[0]; !near(at, (3+0.32)*r.tps) {
		t.Fatalf("an odd step should swing late, at %v not %v", at, 3.32*r.tps)
	}
}

func TestConditions(t *testing.T) {
	r := newRig(t, Config{}, "a", "b", "c")
	for name, cond := range map[string]string{"a": "1:2", "b": "fill", "c": "first"} {
		r.on(name, 0)
		for i, c := range Conditions {
			if c == cond {
				r.set(name, 0, Condition, float64(i))
			}
		}
	}
	r.run(16 * 4)
	if n := len(r.recs["a"].times); n != 2 {
		t.Fatalf("1:2 should play on loops 1 and 3 of 4, %d hits", n)
	}
	if n := len(r.recs["b"].times); n != 0 {
		t.Fatal("fill should not play without the fill")
	}
	if n := len(r.recs["c"].times); n != 1 {
		t.Fatalf("first should play once, %d hits", n)
	}
	r.seq.SetFill(true)
	r.run(16)
	if n := len(r.recs["b"].times); n != 1 {
		t.Fatal("fill should play with the fill on")
	}
}

func TestPolymeter(t *testing.T) {
	r := newRig(t, Config{}, "a")
	r.on("a", 0)
	r.seq.SetLength("a", 3)
	r.run(12)
	rec := r.recs["a"]
	if len(rec.times) != 4 || !near(rec.times[1], 3*r.tps) {
		t.Fatalf("a row of 3 steps should come round every 3 steps: %v", rec.times)
	}
	if r.grid.Lengths()[0] != 3 {
		t.Fatal("the grid should know the row's length")
	}
}

func TestPatternSwitch(t *testing.T) {
	r := newRig(t, Config{}, "a")
	r.grid.Set(0, 0, 0, Velocity, 0.5)
	r.grid.Set(1, 0, 4, Velocity, 0.9)
	r.run(6)
	r.grid.SetPattern(1)
	r.run(10) // the rest of pattern A: nothing more
	rec := r.recs["a"]
	if len(rec.times) != 1 || r.grid.Playing() != 0 {
		t.Fatalf("the pattern should switch when it comes round: %v, playing %d", rec.times, r.grid.Playing())
	}
	r.run(16)
	if len(rec.times) != 2 || !near(rec.times[1], 20*r.tps) || r.grid.Playing() != 1 {
		t.Fatalf("pattern B should play from step 16: %v", rec.times)
	}
}

func TestMuteSoloChoke(t *testing.T) {
	p := psalm.New(nil)
	ctl := panel.New(p, "Test")
	grid := NewGrid(ctl, "Beat", []string{"closed", "open", "kick"}, 16, 1)
	seq := New(p, grid, Config{Seed: 1})
	var tick int64
	recs := map[string]*recorder{}
	for _, name := range grid.Rows {
		recs[name] = &recorder{samplesPerTick: 100, tick: &tick}
	}
	seq.Row("closed", recs["closed"], Choke(1)).Row("open", recs["open"], Choke(1)).Row("kick", recs["kick"])
	for _, name := range grid.Rows {
		grid.Set(0, grid.Row(name), 0, Velocity, 1)
	}
	seq.SetMute("kick", true)
	steps := func(n int) {
		end := tick + int64(float64(n)*float64(psalm.PPQ())/4)
		for ; tick < end; tick++ {
			seq.CtrlTick(psalm.CtrlTick{TickTimestamp: tick + 1})
		}
	}
	steps(16)
	if len(recs["kick"].times) != 0 || recs["open"].chokes != 1 || recs["closed"].chokes != 1 {
		t.Fatalf("a muted row is silent, rows of a choke group choke each other: kick %d, chokes %d %d",
			len(recs["kick"].times), recs["open"].chokes, recs["closed"].chokes)
	}
	seq.SetMute("kick", false)
	seq.SetSolo("kick", true)
	steps(16)
	if len(recs["kick"].times) != 1 || len(recs["open"].times) != 1 {
		t.Fatal("a soloed row plays alone")
	}
	if heads := grid.Playheads(); heads[2] != 15 {
		t.Fatalf("the playheads should follow, got %v", heads)
	}
}

func TestControls(t *testing.T) {
	r := newRig(t, Config{Swing: 0.6}, "kick", "snare")
	r.seq.Controls(r.ctl)
	for _, name := range []string{"Swing", "Humanize", "Fill", "kick level", "kick length", "snare mute", "snare solo"} {
		if r.ctl.Param(name) == nil {
			t.Fatalf("the controls should have %q", name)
		}
	}
	if r.ctl.Param("Swing").Value() != 0.6 {
		t.Fatal("the swing control should start at the swing of the config")
	}
}
