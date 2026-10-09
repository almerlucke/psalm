// Package stepseq is a step sequencer playing a grid of a panel: rows of steps, every step with
// a velocity, a chance, a roll, a nudge, a pitch and a condition, in patterns. Rows play sounds of
// a drum synth or notes of any synth; each row can have its own length (rows loop apart), a level,
// mute, solo and a choke group. Swing, humanizing and a fill switch act on the whole sequencer.
//
//	ctl := panel.New(p, "Drums")
//	grid := stepseq.NewGrid(ctl, "Beat", []string{"kick", "snare", "hat", "open"}, 16, 4)
//	seq := stepseq.New(p, grid, stepseq.Config{StepsPerBeat: 4, Swing: 0.55}).
//		Row("kick", stepseq.Sound(kit, "kick")).
//		Row("hat", stepseq.Sound(kit, "hat"), stepseq.Choke(1)).
//		Row("open", stepseq.Sound(kit, "open"), stepseq.Choke(1))
//	seq.Controls(ctl)
//
// The grid is edited on the panel's page (or in code with grid.Set), the sequencer reads it every
// step: a change is heard the next time the step comes round.
package stepseq

import (
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
	"sync/atomic"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
)

// the fields of a step, in the grid
const (
	Velocity   = iota // 0 is off
	Chance            // the chance the step plays (0-1)
	Roll              // hits the step is split into (1 is a single hit)
	RollChance        // the chance the roll happens (else a single hit)
	RollRamp          // the velocities of a roll: up (1) from soft to the step's velocity, down (-1), even (0)
	Nudge             // when the step plays, in steps: early (-0.5) to late (0.5)
	Pitch             // semitones (the speed of a sample, the pitch of a note)
	Condition         // when the step plays: see Conditions
)

// Conditions are when a step plays: always; A:B on the Ath time of every B times the row comes
// round; with the fill switch on, or off; the first time the row plays (since the pattern started),
// or not
var Conditions = []string{"always", "1:2", "2:2", "1:3", "2:3", "3:3", "1:4", "2:4", "3:4", "4:4", "fill", "not fill", "first", "not first"}

// conditionAB holds A and B of the A:B conditions, by index
var conditionAB = map[int][2]int{1: {1, 2}, 2: {2, 2}, 3: {1, 3}, 4: {2, 3}, 5: {3, 3}, 6: {1, 4}, 7: {2, 4}, 8: {3, 4}, 9: {4, 4}}

// Fields are the fields of the steps of a sequencer's grid
func Fields() []panel.Field {
	return []panel.Field{
		{Name: "velocity", Min: 0, Max: 1, Default: 0},
		{Name: "chance", Min: 0, Max: 1, Default: 1},
		{Name: "roll", Min: 1, Max: 8, Default: 1, Step: 1},
		{Name: "roll chance", Min: 0, Max: 1, Default: 1},
		{Name: "roll ramp", Min: -1, Max: 1, Default: 0},
		{Name: "nudge", Min: -0.5, Max: 0.5, Default: 0, Unit: "step"},
		{Name: "pitch", Min: -12, Max: 12, Default: 0, Step: 1, Unit: "st"},
		{Name: "condition", Min: 0, Max: float64(len(Conditions) - 1), Default: 0, Step: 1, Options: Conditions},
	}
}

// NewGrid adds a grid for a sequencer to a panel: rows (one per sound), steps (the longest row) and
// patterns
func NewGrid(ctl *panel.Panel, name string, rows []string, steps, patterns int) *panel.Grid {
	return ctl.Grid(name, rows, steps, patterns, Fields()...)
}

// Config sets how a sequencer plays
type Config struct {
	StepsPerBeat     float64 // 4 is 16ths (the default), 2 8ths, 3 8th triplets
	Swing            float64 // where every second step lands: 0.5 (or 0) straight, 0.58 light, 0.66 a triplet feel
	Humanize         float64 // ± milliseconds of random timing per hit
	HumanizeVelocity float64 // ± random variation of the velocity (0.1 is ±10%)
	Seed             uint64  // the same seed plays the same chances, 0 is the patch's random stream
}

// Target is what a row plays
type Target interface {
	// play plays a hit at an amplitude, moved by semitones, at a tick (beat is the length of a beat
	// in seconds)
	play(amp, pitch, beat float64, tick psalm.CtrlTick)
	// choke stops the last hit (a choke group)
	choke(tick psalm.CtrlTick)
}

// sound is a sound of a drum synth (a poly synth of sample players: synths/drums)
type sound struct {
	synth psalm.Modular
	name  string
	n     int
	last  string
}

// Sound plays a sound of a drum synth (drums.NewDrums) by its name in the sound bank; pitch
// changes its speed. A choked sound fades out quickly
func Sound(drums psalm.Modular, name string) Target { return &sound{synth: drums, name: name} }

func (s *sound) play(amp, pitch, _ float64, tick psalm.CtrlTick) {
	s.n++
	s.last = fmt.Sprintf("%s#%d", s.name, s.n)
	s.synth.Receive(psalm.Message{
		"command":   "trigger",
		"noteOn":    s.last,
		"tick":      tick,
		"amplitude": amp,
		"message":   psalm.Message{"sound": s.name, "speed": math.Exp2(pitch / 12)},
	}, 0)
}

func (s *sound) choke(tick psalm.CtrlTick) {
	if s.last != "" {
		s.synth.Receive(psalm.Message{"command": "trigger", "noteOff": s.last, "tick": tick}, 0)
		s.last = ""
	}
}

// note is a note of a synth
type note struct {
	synth psalm.Modular
	pitch float64
	beats float64
}

// Note plays a note of a synth (any poly synth, the piano): a pitch (MIDI), moved by the step's
// pitch, lasting a number of beats. It can't be choked
func Note(synth psalm.Modular, pitch, beats float64) Target {
	return &note{synth: synth, pitch: pitch, beats: beats}
}

func (n *note) play(amp, pitch, beat float64, tick psalm.CtrlTick) {
	n.synth.Receive(psalm.Message{
		"command":   "trigger",
		"duration":  n.beats * beat,
		"tick":      tick,
		"amplitude": amp,
		"message":   psalm.Message{"pitch": n.pitch + pitch, "velocity": amp},
	}, 0)
}

func (n *note) choke(psalm.CtrlTick) {}

// RowOption sets something of a row
type RowOption func(r *row)

// Choke puts a row in a choke group: a hit of a row stops the last hit of the other rows of the
// group (an open hihat cut by the closed one). 0 is no group
func Choke(group int) RowOption { return func(r *row) { r.choke = group } }

// Length sets how many steps a row plays before it comes round: rows of other lengths than the
// grid loop apart (polymeter)
func Length(steps int) RowOption { return func(r *row) { r.length.Store(int32(steps)) } }

// row is a row of the grid and what it plays
type row struct {
	index  int
	target Target
	choke  int
	level  atomic.Uint64 // gain, float64 bits
	mute   atomic.Bool
	solo   atomic.Bool
	length atomic.Int32
}

// hit is a hit waiting to be played
type hit struct {
	tick      float64 // in ticks of the patch
	row       *row
	amp       float64
	pitch     float64
	condition int // fill and not fill are checked when the hit plays: a switch right before it counts
}

// Sequencer plays a grid. Its settings are safe to change from any goroutine (the panel's
// controls change them on the audio thread)
type Sequencer struct {
	*psalm.Module
	grid *panel.Grid
	cfg  Config
	rows []*row // by grid row, nil for rows that play nothing

	swing, humanize, humanizeVel atomic.Uint64 // float64 bits
	fill                         atomic.Bool

	rng       *rand.Rand
	scheduled int64 // the next step to schedule
	step      int64 // the step playing
	start     int64 // the step the pattern scheduled started at
	pattern   int   // the pattern scheduled
	pending   []hit

	// what plays, for the page: it changes when the step a new pattern starts at comes, not when
	// the step is scheduled (a step ahead)
	headStart, switchAt int64
	headPattern         int
}

// New creates a sequencer playing a grid and adds it to the patch. Give its rows something to play
// with Row
func New(p psalm.Patcher, grid *panel.Grid, cfg Config) *Sequencer {
	if cfg.StepsPerBeat <= 0 {
		cfg.StepsPerBeat = 4
	}
	s := &Sequencer{Module: psalm.NewControl(), grid: grid, cfg: cfg, rows: make([]*row, len(grid.Rows)), step: -1}
	s.SetSelf(s)
	s.SetSwing(cfg.Swing)
	s.SetHumanize(cfg.Humanize, cfg.HumanizeVelocity)
	if cfg.Seed != 0 {
		s.rng = rand.New(rand.NewPCG(cfg.Seed, 0x5eed))
	} else {
		s.rng = s.Env().NewRand()
	}
	s.pattern = grid.Pattern()
	s.headPattern, s.switchAt = s.pattern, -1
	grid.SetPlaying(s.pattern)
	s.AddTo(p)
	return s
}

// Row sets what a row of the grid plays. It panics on a row the grid doesn't have
func (s *Sequencer) Row(name string, target Target, opts ...RowOption) *Sequencer {
	i := s.grid.Row(name)
	if i < 0 {
		panic(fmt.Sprintf("stepseq: the grid %q has no row %q", s.grid.Name, name))
	}
	r := &row{index: i, target: target}
	r.level.Store(math.Float64bits(1))
	r.length.Store(int32(s.grid.Steps))
	for _, o := range opts {
		o(r)
	}
	s.grid.SetLength(i, int(r.length.Load()))
	s.rows[i] = r
	return s
}

func (s *Sequencer) rowByName(name string) *row {
	if i := s.grid.Row(name); i >= 0 {
		return s.rows[i]
	}
	return nil
}

// SetSwing sets where every second step lands: 0.5 straight to 0.75
func (s *Sequencer) SetSwing(swing float64) {
	if swing <= 0 {
		swing = 0.5
	}
	s.swing.Store(math.Float64bits(math.Max(0.5, math.Min(swing, 0.75))))
}

// SetHumanize sets the random timing (± milliseconds) and velocity (± part) of every hit
func (s *Sequencer) SetHumanize(ms, velocity float64) {
	s.humanize.Store(math.Float64bits(math.Max(0, ms)))
	s.humanizeVel.Store(math.Float64bits(math.Max(0, math.Min(velocity, 1))))
}

// SetFill switches the fill on or off: steps with the condition "fill" play only with it on,
// "not fill" only with it off
func (s *Sequencer) SetFill(on bool) { s.fill.Store(on) }

// SetLevel sets the level of a row in dB
func (s *Sequencer) SetLevel(name string, db float64) {
	if r := s.rowByName(name); r != nil {
		r.level.Store(math.Float64bits(math.Pow(10, db/20)))
	}
}

// SetMute mutes a row
func (s *Sequencer) SetMute(name string, mute bool) {
	if r := s.rowByName(name); r != nil {
		r.mute.Store(mute)
	}
}

// SetSolo solos a row: when any row is soloed only the soloed rows play
func (s *Sequencer) SetSolo(name string, solo bool) {
	if r := s.rowByName(name); r != nil {
		r.solo.Store(solo)
	}
}

// SetLength sets how many steps a row plays before it comes round
func (s *Sequencer) SetLength(name string, steps int) {
	if r := s.rowByName(name); r != nil {
		steps = max(1, min(steps, s.grid.Steps))
		r.length.Store(int32(steps))
		s.grid.SetLength(r.index, steps)
	}
}

// Controls adds controls for the sequencer to a panel: swing, humanize and fill, and a card for every
// row that plays: its level, length, mute and solo (named "kick level", "kick length", "kick mute"
// and "kick solo")
func (s *Sequencer) Controls(ctl *panel.Panel) {
	ctl.Section(s.grid.Name + " feel")
	f := func(v *atomic.Uint64) float64 { return math.Float64frombits(v.Load()) }
	ctl.Fader("Swing", 0.5, 0.75, panel.Linear, panel.Func(s.SetSwing), panel.Default(f(&s.swing)))
	ctl.Knob("Humanize", 0, 30, panel.Linear, panel.Func(func(v float64) { s.SetHumanize(v, f(&s.humanizeVel)) }),
		panel.Default(f(&s.humanize)), panel.Unit("ms"))
	ctl.Knob("Humanize velocity", 0, 0.5, panel.Linear, panel.Func(func(v float64) { s.SetHumanize(f(&s.humanize), v) }),
		panel.Default(f(&s.humanizeVel)))
	ctl.Toggle("Fill", panel.Func(func(v float64) { s.SetFill(v >= 0.5) }))

	// a card per row: its level, its length, mute and solo
	for _, r := range s.rows {
		if r == nil {
			continue
		}
		name := s.grid.Rows[r.index]
		ctl.Section(name)
		ctl.Knob(name+" level", -40, 6, panel.Linear, panel.Func(func(v float64) { s.SetLevel(name, v) }),
			panel.Unit("dB"), panel.Default(0), panel.Label("level"))
		ctl.Knob(name+" length", 1, float64(s.grid.Steps), panel.Linear, panel.Func(func(v float64) { s.SetLength(name, int(v)) }),
			panel.Step(1), panel.Default(float64(r.length.Load())), panel.Label("length"))
		ctl.Toggle(name+" mute", panel.Func(func(v float64) { s.SetMute(name, v >= 0.5) }), panel.Label("M"))
		ctl.Toggle(name+" solo", panel.Func(func(v float64) { s.SetSolo(name, v >= 0.5) }), panel.Label("S"))
	}
}

// Restart plays the grid from its first step again, with the pattern asked for (psalm.Restarter):
// the conditions count from the start again
func (s *Sequencer) Restart() {
	s.scheduled, s.step, s.start, s.pending = 0, -1, 0, s.pending[:0]
	s.pattern = s.grid.Pattern()
	s.headStart, s.headPattern, s.switchAt = 0, s.pattern, -1
	s.grid.SetPlaying(s.pattern)
}

// ticksPerStep is the length of a step in ticks of the patch
func (s *Sequencer) ticksPerStep() float64 { return float64(s.Config().PPQ) / s.cfg.StepsPerBeat }

// CtrlTick plays the hits due, and works out the hits of the step after the one starting (so a
// step nudged early is there in time)
func (s *Sequencer) CtrlTick(tick psalm.CtrlTick) {
	t := float64(tick.TickTimestamp - 1)
	tps := s.ticksPerStep()
	k := int64(math.Floor(t / tps))
	for s.scheduled <= k+1 {
		s.schedule(s.scheduled)
		s.scheduled++
	}
	if k != s.step {
		s.step = k
		if k == s.switchAt {
			s.headStart, s.headPattern, s.switchAt = k, s.pattern, -1
		}
		s.grid.SetPlaying(s.headPattern)
		for _, r := range s.rows {
			if r != nil {
				s.grid.SetPlayhead(r.index, int((k-s.headStart)%int64(r.length.Load())))
			}
		}
	}
	s.play(t, tick)
}

// play plays the hits due by now, at their sample within the tick
func (s *Sequencer) play(t float64, tick psalm.CtrlTick) {
	cfg := s.Config()
	samplesPerTick := s.Env().TickDuration() * cfg.SampleRate
	beat := s.Env().TickDuration() * float64(cfg.PPQ)
	n := 0
	solo := false
	for _, r := range s.rows {
		if r != nil && r.solo.Load() {
			solo = true
		}
	}
	for _, h := range s.pending {
		if h.tick >= t+1 {
			break
		}
		n++
		// mute, solo and the fill count when the hit plays: a switch right before it counts
		if h.row.mute.Load() || (solo && !h.row.solo.Load()) || !s.fillPasses(h.condition) {
			continue
		}
		at := tick
		if late := h.tick - t; late > 0 {
			at.SampleOffset = min(tick.SampleOffset+int(late*samplesPerTick), cfg.BufferSize-1)
		}
		if h.row.choke != 0 {
			for _, other := range s.rows {
				if other != nil && other != h.row && other.choke == h.row.choke {
					other.target.choke(at)
				}
			}
		}
		h.row.target.play(h.amp, h.pitch, beat, at)
	}
	s.pending = s.pending[n:]
}

// schedule works out the hits of a step: the pattern (it switches when it comes round), the
// condition, the chance, the roll, and the timing (swing, nudge, humanizing)
func (s *Sequencer) schedule(k int64) {
	g := s.grid
	if (k-s.start)%int64(g.Steps) == 0 {
		if p := g.Pattern(); p != s.pattern {
			s.pattern = p
			s.start = k // the rows start again, the conditions count from here
			s.switchAt = k
		}
	}
	tps := s.ticksPerStep()
	base := float64(k) * tps
	if k%2 == 1 {
		base += (math.Float64frombits(s.swing.Load()) - 0.5) * 2 * tps
	}
	humanize := math.Float64frombits(s.humanize.Load()) / 1000 / s.Env().TickDuration() // in ticks
	humanize = math.Min(humanize, 0.25*tps)
	humanizeVel := math.Float64frombits(s.humanizeVel.Load())

	for _, r := range s.rows {
		if r == nil {
			continue
		}
		length := int64(r.length.Load())
		step := int((k - s.start) % length)
		loop := int((k - s.start) / length)
		value := func(f int) float64 { return g.Value(s.pattern, r.index, step, f) }
		vel := value(Velocity)
		condition := int(value(Condition))
		if vel <= 0 || !s.passes(condition, loop) || s.rng.Float64() >= value(Chance) {
			continue
		}
		hits := 1
		if roll := int(value(Roll)); roll > 1 && s.rng.Float64() < value(RollChance) {
			hits = roll
		}
		ramp := value(RollRamp)
		at := base + value(Nudge)*tps
		if humanize > 0 {
			at += humanize * (2*s.rng.Float64() - 1)
		}
		level := math.Float64frombits(r.level.Load())
		for i := range hits {
			v := vel * rampFactor(i, hits, ramp)
			if humanizeVel > 0 {
				v *= 1 + humanizeVel*(2*s.rng.Float64()-1)
			}
			s.pending = append(s.pending, hit{
				tick:      at + float64(i)*tps/float64(hits),
				row:       r,
				amp:       math.Max(0, math.Min(v, 1)) * level,
				pitch:     value(Pitch),
				condition: condition,
			})
		}
	}
	sort.SliceStable(s.pending, func(i, j int) bool { return s.pending[i].tick < s.pending[j].tick })
}

// passes tells whether a step's condition lets it play the loopth time its row comes round (the
// fill is checked when it plays, see fillPasses)
func (s *Sequencer) passes(condition, loop int) bool {
	if ab, ok := conditionAB[condition]; ok {
		return loop%ab[1] == ab[0]-1
	}
	switch Conditions[max(0, min(condition, len(Conditions)-1))] {
	case "first":
		return loop == 0
	case "not first":
		return loop > 0
	}
	return true
}

// fillPasses tells whether the fill lets a hit of a condition play now
func (s *Sequencer) fillPasses(condition int) bool {
	switch Conditions[max(0, min(condition, len(Conditions)-1))] {
	case "fill":
		return s.fill.Load()
	case "not fill":
		return !s.fill.Load()
	}
	return true
}

// rampFactor is the velocity of hit i of a roll of n: ramp 1 rises from nothing to the full
// velocity, -1 falls to nothing, 0 keeps it
func rampFactor(i, n int, ramp float64) float64 {
	if n <= 1 || ramp == 0 {
		return 1
	}
	x := float64(i) / float64(n-1) // 0 at the first hit, 1 at the last
	if ramp > 0 {
		return 1 - ramp*(1-x)
	}
	return 1 + ramp*x
}
