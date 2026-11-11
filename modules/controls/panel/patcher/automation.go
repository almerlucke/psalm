package patcher

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/almerlucke/psalm"
)

// --- automation: lanes drawn over the bars of the song ---

// Point is a point of a lane: a position in bars from the start of the song, a value 0-1, and how the lane goes on from it to the next point
type Point struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Shape string  `json:"shape,omitempty"` // "" or "lin": a straight line; "smooth": eased; "step": stays until the next point; "exp": even steps per octave
}

// LaneData is a lane: a name and its points, from the first bar on
type LaneData struct {
	Name   string  `json:"name"`
	Points []Point `json:"points"`
}

// At is the value of a lane at a position in bars: the first value before the first point, the last after the last, 0 for a lane without points
func (l LaneData) At(x float64) float64 {
	p := l.Points
	if len(p) == 0 {
		return 0
	}
	if x <= p[0].X {
		return p[0].Y
	}
	last := p[len(p)-1]
	if x >= last.X {
		return last.Y
	}
	i := sort.Search(len(p), func(i int) bool { return p[i].X > x }) // the first point after x
	a, b := p[i-1], p[i]
	t := (x - a.X) / (b.X - a.X)
	switch a.Shape {
	case "step":
		return a.Y
	case "smooth":
		t = 0.5 - 0.5*math.Cos(math.Pi*t)
	case "exp":
		if a.Y > 0 && b.Y > 0 {
			return a.Y * math.Pow(b.Y/a.Y, t)
		}
	}
	return a.Y + (b.Y-a.Y)*t
}

// laneShapes are the shapes a point can have
var laneShapes = []string{"lin", "smooth", "step", "exp"}

// ParseLanes reads lanes typed, a line each: a name, a colon, then the points by commas, each a position in bars and a value 0-1 and perhaps how it goes
// on to the next ("Cutoff: 0 0.2, 6 0.8 smooth, 12 0.5 step"); a lane without points is a line of its name; blank lines and lines after # are left out
func ParseLanes(text string) ([]LaneData, error) {
	var out []LaneData
	for n, line := range strings.Split(text, "\n") {
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		name, rest, _ := strings.Cut(line, ":")
		lane := LaneData{Name: strings.TrimSpace(name), Points: []Point{}}
		for _, part := range strings.Split(rest, ",") {
			f := strings.Fields(part)
			if len(f) == 0 {
				continue
			}
			if len(f) < 2 || len(f) > 3 {
				return nil, fmt.Errorf("line %d: %q: a position, a value and perhaps a shape (6 0.8 smooth)", n+1, strings.TrimSpace(part))
			}
			x, errX := strconv.ParseFloat(f[0], 64)
			y, errY := strconv.ParseFloat(f[1], 64)
			if errX != nil || errY != nil || math.IsNaN(x+y) || math.IsInf(x+y, 0) {
				return nil, fmt.Errorf("line %d: %q: a position in bars and a value", n+1, strings.TrimSpace(part))
			}
			if x < 0 {
				return nil, fmt.Errorf("line %d: %q: a position of 0 (the start) or more", n+1, strings.TrimSpace(part))
			}
			pt := Point{X: x, Y: max(0, min(y, 1))}
			if len(f) == 3 {
				if !slicesContains(laneShapes, f[2]) {
					return nil, fmt.Errorf("line %d: %q: lin, smooth, step or exp", n+1, f[2])
				}
				if f[2] != "lin" {
					pt.Shape = f[2]
				}
			}
			lane.Points = append(lane.Points, pt)
		}
		sort.SliceStable(lane.Points, func(i, j int) bool { return lane.Points[i].X < lane.Points[j].X })
		out = append(out, lane)
	}
	return out, nil
}

func slicesContains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// FormatLanes writes lanes as ParseLanes reads them
func FormatLanes(lanes []LaneData) string {
	num := func(v float64, places int) string {
		return strconv.FormatFloat(math.Round(v*math.Pow10(places))/math.Pow10(places), 'f', -1, 64)
	}
	var b strings.Builder
	for _, l := range lanes {
		b.WriteString(strings.Join(strings.Fields(strings.NewReplacer(":", " ", ",", " ", "#", " ").Replace(l.Name)), " "))
		b.WriteString(":")
		for i, p := range l.Points {
			if i > 0 {
				b.WriteString(",")
			}
			b.WriteString(" " + num(p.X, 3) + " " + num(p.Y, 4))
			if p.Shape != "" && p.Shape != "lin" {
				b.WriteString(" " + p.Shape)
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

// AutomationLanes is how many lanes an automation module has, by its preset
var AutomationLanes = map[string]int{"2 lanes": 2, "4 lanes": 4, "8 lanes": 8, "16 lanes": 16}

// AutomationPresets are the presets of automation modules
var AutomationPresets = []string{"4 lanes", "2 lanes", "8 lanes", "16 lanes"}

// Automation moves lanes drawn over the bars of the song: every lane is a line of points, each with a shape to the next, and a mod output of its own (0-1: a
// mod cable moves a setting over its range). They are drawn and edited on the page, in the lanes at the bottom of the patch, or typed. Without a song the lanes
// go round over a number of bars
type Automation struct {
	*psalm.Module
	song  *Song
	count int
	lanes atomic.Pointer[[]LaneData]
	bars  atomic.Uint64 // the bars of a round without a song (float bits)
	vals  []atomicFloat
	x     atomicFloat // where it is, in bars from the start of the song
	rate  atomicFloat // bars a second
	live  atomic.Bool // the song is playing (it has not ended)

	// recording: the lane armed (-1 none) keeps its value for the cable while the setting it moves is played by hand (the recorder writes what is played into it)
	rec     atomic.Int32
	frozen  atomicFloat // what the armed lane's cable has
	shown   atomicFloat // what the page shows of the armed lane: what is being played
	writing atomic.Bool // a move is being written
}

// NewAutomation returns an automation module with a number of lanes, each without points
func NewAutomation(count int) *Automation {
	a := &Automation{Module: psalm.NewControl(), count: count, vals: make([]atomicFloat, count)}
	a.SetSelf(a)
	a.rec.Store(-1)
	a.song = SongOf(a.Env())
	a.bars.Store(math.Float64bits(16))
	empty := make([]LaneData, count)
	for i := range empty {
		empty[i] = LaneData{Name: fmt.Sprintf("lane %d", i+1), Points: []Point{}}
	}
	a.lanes.Store(&empty)
	return a
}

// SetLanes sets the lanes typed (from any goroutine); a lane more than the module has is left out, one fewer stays empty
func (a *Automation) SetLanes(text string) error {
	lanes, err := ParseLanes(text)
	if err != nil {
		return err
	}
	for len(lanes) < a.count {
		lanes = append(lanes, LaneData{Name: fmt.Sprintf("lane %d", len(lanes)+1), Points: []Point{}})
	}
	lanes = lanes[:a.count]
	a.lanes.Store(&lanes)
	return nil
}

// SetBars sets the bars of a round without a song, typed
func (a *Automation) SetBars(text string) error {
	v, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil || v < 1 || v > 4096 {
		return fmt.Errorf("%q: a number of bars, 1 to 4096", strings.TrimSpace(text))
	}
	a.bars.Store(math.Float64bits(v))
	return nil
}

// Lanes are its lanes now
func (a *Automation) Lanes() []LaneData { return *a.lanes.Load() }

// Value is what a lane is at, 0-1 (what the cable moves its setting to): an armed lane stays where it was when armed, the setting is played by hand
func (a *Automation) Value(lane int) float64 {
	if lane < 0 || lane >= a.count {
		return 0
	}
	if int(a.rec.Load()) == lane {
		return max(0, min(a.frozen.Load(), 1))
	}
	return max(0, min(a.vals[lane].Load(), 1))
}

// SetRecord arms a lane to record (-1: none): its cable stops moving its setting
func (a *Automation) SetRecord(lane int) {
	if lane < 0 || lane >= a.count {
		a.rec.Store(-1)
		a.writing.Store(false)
		return
	}
	a.frozen.Store(a.vals[lane].Load())
	a.shown.Store(a.vals[lane].Load())
	a.writing.Store(false)
	a.rec.Store(int32(lane))
}

// Armed is the lane armed to record, -1 for none
func (a *Automation) Armed() int { return int(a.rec.Load()) }

// CtrlTick moves the lanes with the song
func (a *Automation) CtrlTick(tick psalm.CtrlTick) {
	ppq := float64(a.Config().PPQ)
	beatsPerBar := 4.0
	var x float64
	if tr := a.song.Transport(); tr != nil {
		if bpb := tr.Form().BeatsPerBar; bpb > 0 {
			beatsPerBar = bpb
		}
		pos, _ := a.song.At(tick, a.Config().PPQ)
		x = float64(pos.Section.StartBar) + max(0, min(pos.Progress, 1))*float64(pos.Section.Bars)
		a.live.Store(!pos.Ended)
		if pos.Ended {
			x = float64(tr.Form().Bars())
		}
	} else {
		bars := math.Float64frombits(a.bars.Load())
		x = math.Mod(float64(tick.TickTimestamp)/ppq/beatsPerBar, bars)
		a.live.Store(true)
	}
	a.x.Store(x)
	a.rate.Store(a.BPM() / 60 / beatsPerBar)
	for i, l := range *a.lanes.Load() {
		a.vals[i].Store(l.At(x))
	}
}

// LanesView is what the page draws of the lanes: the lanes over the bars of the song, and where it is
type LanesView struct {
	Lanes    []LaneData    `json:"lanes"`
	Length   float64       `json:"length"` // bars, the song's or the round's
	Sections []SectionView `json:"sections,omitempty"`
	Targets  []string      `json:"targets"` // what the mod cable of every lane moves ("" none): the module and its setting
	Head     *LaneHead     `json:"-"`       // where it is: it changes all the time, so the page gets it apart from the rest
}

// LaneHead is where the lanes are now
type LaneHead struct {
	X      float64   `json:"x"`      // bars from the start
	Rate   float64   `json:"rate"`   // bars a second
	Live   bool      `json:"live"`   // the song is playing
	Values []float64 `json:"values"` // of every lane
	Armed  int       `json:"armed"`  // the lane armed to record, -1 for none
	// Writing is true while a move is being written into the armed lane
	Writing bool `json:"writing"`
}

// SectionView is a section of the song on the ruler of the lanes
type SectionView struct {
	Name  string `json:"name"`
	Start int    `json:"start"`
	Bars  int    `json:"bars"`
}

// View is the lanes as the page draws them (from any goroutine)
func (a *Automation) View() *LanesView {
	v := &LanesView{Lanes: a.Lanes(), Length: math.Float64frombits(a.bars.Load()), Targets: make([]string, a.count)}
	h := &LaneHead{X: math.Round(a.x.Load()*1000) / 1000, Rate: a.rate.Load(), Live: a.live.Load(), Values: make([]float64, a.count), Armed: a.Armed(), Writing: a.writing.Load()}
	for i := range h.Values {
		h.Values[i] = math.Round(a.Value(i)*1000) / 1000
	}
	if h.Armed >= 0 { // what is played, as it is played
		h.Values[h.Armed] = math.Round(max(0, min(a.shown.Load(), 1))*1000) / 1000
	}
	v.Head = h
	if tr := a.song.Transport(); tr != nil {
		v.Length = float64(tr.Form().Bars())
		for _, s := range tr.Form().Sections {
			v.Sections = append(v.Sections, SectionView{Name: s.Name, Start: s.StartBar, Bars: s.Bars})
		}
	}
	return v
}

// AutomationHelp is what to know to type the lanes
const AutomationHelp = "a line each lane: its name, a colon, then points by commas, each a position in bars from the start (0) and a value 0 to 1 and perhaps how it goes on to the next point: `Cutoff: 0 0.2, 6 0.8 smooth, 12 0.5 step` (lin, smooth, step, exp)\n" +
	"the lanes follow the song (a song module): a position is in the bars of its form; without a song they go round over the bars below\n" +
	"drawn on the page too: the Lanes button in the bar above the patch (click adds a point, drag moves it, double-click changes its shape, Delete takes it away)\n" +
	"every lane has a mod output (0 to 1): a mod cable to any setting moves it over its range\n" +
	"Ctrl+Enter (or Run) takes them"

// automationKind is the kind of automation modules: a mod output for every lane
func automationKind() Kind {
	return Kind{Name: "automation", Group: "Song", Presets: AutomationPresets, Outs: []Port{{Name: "lane 1", Type: Mod}},
		Make: func(preset string) (*Made, error) {
			n, ok := AutomationLanes[preset]
			if !ok {
				n = 4
			}
			a := NewAutomation(n)
			outs := make([]Port, n)
			for i := range outs {
				outs[i] = Port{Name: fmt.Sprintf("lane %d", i+1), Type: Mod, Index: i}
			}
			return &Made{Module: a, Outs: outs,
				Texts: []Text{
					{Key: "lanes", Label: "lanes", Value: FormatLanes(a.Lanes()), Lines: 8, Help: AutomationHelp},
					{Key: "bars", Label: "bars (without a song)", Value: "16"},
				},
				SetText: func(key, v string) error {
					if key == "bars" {
						return a.SetBars(v)
					}
					return a.SetLanes(v)
				},
				Mod:   func(out int) func() float64 { return func() float64 { return a.Value(out) } },
				Lanes: func() *LanesView { return a.View() },
			}, nil
		}}
}
