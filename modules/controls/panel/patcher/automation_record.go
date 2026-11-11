package patcher

import (
	"fmt"
	"math"
	"slices"
	"time"
)

// --- recording what is played into a lane ---

// laneRecorder writes a lane while a setting is moved by hand: the lane is armed, its cable stops moving the setting it goes to, and the moves of that setting (a knob
// on the page of its module) are written into the lane at the bar the song is in, thinned to the points that matter. A move begins when the setting changes and is over
// when it has been still for most of a second, the lane is the move over the bars it took (what was there is taken away) and as it was outside them
type laneRecorder struct {
	pt   *Patcher
	a    *Automation
	id   string
	lane int
	stop chan struct{}

	target string // the module the setting is of, its key, and the cable's part of its range
	key    string
	lo, hi float64

	base       float64 // the setting as last seen
	touching   bool
	orig       []Point // the lane before the move
	got        []Point // the move
	start      float64 // bars: where the move began
	lastChange float64 // bars: where the setting last changed
	lastX      float64
}

// recIdle is how long a setting has to be still for a move to be over, in seconds of the song
const recIdle = 0.7

// ArmLane arms a lane of an automation module to record (-1: none): with the song playing, the setting its mod cable moves is played by hand (its knob on the page
// of its module) and the move is written into the lane
func (pt *Patcher) ArmLane(id string, lane int) error {
	pt.mu.Lock()
	n := pt.node(id)
	var a *Automation
	if n != nil {
		a, _ = n.made.Module.(*Automation)
	}
	if a == nil {
		pt.mu.Unlock()
		return fmt.Errorf("no automation module %q", id)
	}
	old := pt.recs[id]
	delete(pt.recs, id)
	pt.mu.Unlock()
	if old != nil {
		close(old.stop)
	}
	if lane < 0 {
		a.SetRecord(-1)
		return nil
	}
	if lane >= len(a.Lanes()) {
		return fmt.Errorf("no lane %d", lane+1)
	}
	r := &laneRecorder{pt: pt, a: a, id: id, lane: lane, stop: make(chan struct{})}
	if err := r.attach(); err != nil {
		a.SetRecord(-1)
		return err
	}
	a.SetRecord(lane)
	pt.mu.Lock()
	if pt.recs == nil {
		pt.recs = map[string]*laneRecorder{}
	}
	pt.recs[id] = r
	pt.mu.Unlock()
	go r.run()
	return nil
}

// target finds the cable of the lane: the module its setting is of, its key and the range the cable moves it over
func (r *laneRecorder) cable() (*node, *Cable) {
	r.pt.mu.Lock()
	defer r.pt.mu.Unlock()
	for _, c := range r.pt.cables {
		if c.Type == Mod && c.From == r.id && c.Out == r.lane {
			return r.pt.node(c.To), c
		}
	}
	return nil, nil
}

// attach looks at where the lane goes: the setting played by hand is what it has now
func (r *laneRecorder) attach() error {
	tn, c := r.cable()
	if tn == nil || tn.settings == nil {
		return fmt.Errorf("cable lane %d to a setting first (a mod cable): its moves are what is recorded", r.lane+1)
	}
	if c.Hi == c.Lo {
		return fmt.Errorf("the cable of lane %d moves its setting over no range", r.lane+1)
	}
	r.target, r.key, r.lo, r.hi = tn.ID, c.Key, c.Lo, c.Hi
	r.base = tn.settings.Values()[c.Key]
	return nil
}

func (r *laneRecorder) run() {
	every := r.pt.recEvery
	if every <= 0 {
		every = 25 * time.Millisecond
	}
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-r.stop:
			return
		case <-t.C:
			if !r.step() {
				r.pt.mu.Lock()
				if r.pt.recs[r.id] == r {
					delete(r.pt.recs, r.id)
				}
				r.pt.mu.Unlock()
				r.a.SetRecord(-1)
				return
			}
		}
	}
}

// y is a value of the setting as a position on the lane: along the cable's part of the range of the setting
func (r *laneRecorder) y(tn *node, v float64) float64 {
	p, ok := tn.set.Find(r.key)
	if !ok {
		return 0
	}
	var x float64
	if p.Exp && p.Min > 0 && v > 0 {
		x = math.Log(v/p.Min) / math.Log(p.Max/p.Min)
	} else if p.Max != p.Min {
		x = (v - p.Min) / (p.Max - p.Min)
	}
	return math.Round(max(0, min((x-r.lo)/(r.hi-r.lo), 1))*10000) / 10000
}

// step looks at the setting once (every 25 ms; a test calls it as it runs the patch); false: the lane is not to be recorded any more (its cable or module is gone)
func (r *laneRecorder) step() bool {
	tn, c := r.cable()
	if tn == nil || tn.settings == nil || c.Key != r.key || c.Lo != r.lo || c.Hi != r.hi {
		return false
	}
	r.pt.mu.Lock()
	gone := r.pt.node(r.id) == nil
	r.pt.mu.Unlock()
	if gone {
		return false
	}
	a := r.a
	if !a.live.Load() {
		return true
	}
	x := a.x.Load()
	if r.touching && x < r.lastX-0.5 { // the song went round: the move ends at the end of what it had
		r.finish()
	}
	r.lastX = x
	v := tn.settings.Values()[r.key]
	if v != r.base {
		r.base = v
		y := r.y(tn, v)
		if !r.touching {
			r.touching, r.start = true, x
			r.orig = slices.Clone(a.Lanes()[r.lane].Points)
			r.got = nil
			a.writing.Store(true)
		}
		r.lastChange = x
		r.got = append(r.got, Point{X: x, Y: y})
		a.shown.Store(y)
		r.apply(false)
	} else if r.touching {
		rate := a.rate.Load()
		if rate <= 0 {
			rate = 0.5
		}
		if x-r.lastChange >= recIdle*rate {
			r.finish()
		}
	}
	return true
}

// apply writes the lane: what was there outside the move, and the move
func (r *laneRecorder) apply(final bool) {
	got := r.got
	if final {
		got = thin(got, 0.004)
	}
	end := r.lastChange
	var pts []Point
	for _, p := range r.orig {
		if p.X < r.start-1e-9 || p.X > end+1e-9 {
			pts = append(pts, p)
		}
	}
	pts = append(pts, got...)
	if last := got[len(got)-1]; end-last.X > 1e-9 { // the last value held to where it was last played
		pts = append(pts, Point{X: end, Y: last.Y})
	}
	slices.SortStableFunc(pts, func(a, b Point) int {
		switch {
		case a.X < b.X:
			return -1
		case a.X > b.X:
			return 1
		}
		return 0
	})
	lanes := r.a.Lanes()
	lanes = slices.Clone(lanes)
	lanes[r.lane].Points = pts
	_ = r.pt.SetLanes(r.id, lanes)
}

// finish ends a move: the lane is written thinned
func (r *laneRecorder) finish() {
	if !r.touching {
		return
	}
	r.touching = false
	r.a.writing.Store(false)
	if len(r.got) > 0 {
		r.apply(true)
	}
	r.got = nil
}

// thin takes away the points of a line that are within an error of the line the points before and after make (Douglas-Peucker on the values)
func thin(pts []Point, eps float64) []Point {
	if len(pts) < 3 {
		return pts
	}
	keep := make([]bool, len(pts))
	keep[0], keep[len(pts)-1] = true, true
	var rec func(a, b int)
	rec = func(a, b int) {
		worst, at := 0.0, -1
		for i := a + 1; i < b; i++ {
			t := 0.0
			if pts[b].X != pts[a].X {
				t = (pts[i].X - pts[a].X) / (pts[b].X - pts[a].X)
			}
			if d := math.Abs(pts[i].Y - (pts[a].Y + (pts[b].Y-pts[a].Y)*t)); d > worst {
				worst, at = d, i
			}
		}
		if at >= 0 && worst > eps {
			keep[at] = true
			rec(a, at)
			rec(at, b)
		}
	}
	rec(0, len(pts)-1)
	var out []Point
	for i, p := range pts {
		if keep[i] {
			out = append(out, p)
		}
	}
	return out
}

// StepRecording looks once at the setting of every lane being recorded (what a ticker does every 25 ms; for a test that runs a patch faster than it plays)
func (pt *Patcher) StepRecording() {
	pt.mu.Lock()
	recs := make([]*laneRecorder, 0, len(pt.recs))
	for _, r := range pt.recs {
		recs = append(recs, r)
	}
	pt.mu.Unlock()
	for _, r := range recs {
		if !r.step() {
			pt.mu.Lock()
			if pt.recs[r.id] == r {
				delete(pt.recs, r.id)
			}
			pt.mu.Unlock()
			r.a.SetRecord(-1)
		}
	}
}
