package patcher

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
)

// --- lanes: a value per section of the song ---

// Curve is how a lane moves through a section: from a value to another (or staying at one)
type Curve struct {
	From, To float64
	Shape    string // "lin", "exp" (from and to above 0: even steps per octave), "smooth" (eased)
}

// At is the value of the curve at a progress through the section, 0-1
func (c Curve) At(progress float64) float64 {
	x := max(0, min(progress, 1))
	switch c.Shape {
	case "exp":
		return c.From * math.Pow(c.To/c.From, x)
	case "smooth":
		x = 0.5 - 0.5*math.Cos(math.Pi*x)
	}
	return c.From + (c.To-c.From)*x
}

// ParseCurve reads a curve: a value ("0.5"), or a value to another and how ("0.2 → 1", "0.2 -> 1
// exp", "smooth")
func ParseCurve(text string) (Curve, error) {
	f := strings.Fields(strings.NewReplacer("→", " → ", "->", " → ").Replace(text))
	num := func(s string) (float64, error) {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, fmt.Errorf("%q: a number", s)
		}
		return v, nil
	}
	switch {
	case len(f) == 1:
		v, err := num(f[0])
		return Curve{From: v, To: v, Shape: "lin"}, err
	case len(f) == 3 || len(f) == 4:
		if f[1] != "→" {
			break
		}
		a, err := num(f[0])
		if err != nil {
			return Curve{}, err
		}
		b, err := num(f[2])
		if err != nil {
			return Curve{}, err
		}
		c := Curve{From: a, To: b, Shape: "lin"}
		if len(f) == 4 {
			c.Shape = f[3]
		}
		switch c.Shape {
		case "lin", "smooth":
		case "exp":
			if a <= 0 || b <= 0 {
				return Curve{}, fmt.Errorf("exp: from and to above 0")
			}
		default:
			return Curve{}, fmt.Errorf("%q: lin, exp or smooth", c.Shape)
		}
		return c, nil
	}
	return Curve{}, fmt.Errorf("%q: a value (0.5) or a value to another (0.2 → 1, and lin, exp or smooth)", strings.TrimSpace(text))
}

// Lane gives a value per section of the song: a curve each ("Intro: 0 → 1", "Verse: 1"), the
// lines without a section for the others, through the section as it plays. Outputs: mod (0-1: a mod
// cable moves a setting over its range) and the value as it is (to a value input). Without a song
// it stays at where its curve starts
type Lane struct {
	*psalm.Module
	song    *Song
	curves  atomic.Pointer[laneCurves]
	mod     atomicFloat
	sent    float64
	sentAt  int64
	hasSent bool
}

type laneCurves struct {
	each    map[string]Curve
	rest    Curve
	hasRest bool
}

// NewLane returns a lane from 0 to 1 through every section
func NewLane() *Lane {
	l := &Lane{Module: psalm.NewControl()}
	l.SetSelf(l)
	l.song = SongOf(l.Env())
	_ = l.SetCurves(DefaultLane)
	return l
}

// DefaultLane is a new lane's curves
const DefaultLane = "0 → 1"

// SetCurves sets the curves typed (from any goroutine)
func (l *Lane) SetCurves(text string) error {
	each, rest, hasRest, err := perSection(text, ParseCurve)
	if err == nil {
		l.curves.Store(&laneCurves{each: each, rest: rest, hasRest: hasRest})
	}
	return err
}

// Value is the lane's value now, 0-1 (mod)
func (l *Lane) Value() float64 { return max(0, min(l.mod.Load(), 1)) }

// CtrlTick moves the lane through the section of the song
func (l *Lane) CtrlTick(tick psalm.CtrlTick) {
	cs := l.curves.Load()
	v, ok := 0.0, false
	if pos, playing := l.song.At(tick, l.Config().PPQ); playing && !pos.Ended {
		if c, has := cs.each[pos.Section.Name]; has {
			v, ok = c.At(pos.Progress), true
		} else if cs.hasRest {
			v, ok = cs.rest.At(pos.Progress), true
		}
	} else if cs.hasRest {
		v, ok = cs.rest.At(0), true
	}
	if !ok {
		return // nothing for this section: it stays where it is
	}
	l.mod.Store(v)
	// the value out: when it changed, not every tick (a value input is not a stream)
	if !l.hasSent || v != l.sent && tick.TickTimestamp-l.sentAt >= 8 {
		l.hasSent, l.sent, l.sentAt = true, v, tick.TickTimestamp
		l.Send(v, 0)
	}
}

// LaneHelp is what to know to type a lane
const LaneHelp = "a curve per section, a line each: `Intro: 0 → 1`, `Verse: 1`, `Chorus: 0.2 → 0.8 smooth` (lin, exp, smooth)\n" +
	"the lines without a section are for the others; it moves through a section as it plays (a song module)\n" +
	"mod: 0 to 1, a mod cable to any setting moves it over its range (chosen below the patch); value: as it is\n" +
	"Ctrl+Enter (or Run) takes it"

// --- gates: parts of the song that play in some sections ---

// Gate lets sound, notes and triggers through in some sections of the song ("Verse, Chorus", or
// "not Intro"), and when it is on: a part of the song, switched on and off by its tile in the song's
// side panel. A note let go always goes through (no note hangs). Inputs: audio, notes, trigger;
// outputs the same
type Gate struct {
	*psalm.Module
	song     *Song
	name     atomic.Pointer[string]
	sections atomic.Pointer[gateSections]
	on       atomic.Bool
	onParam  *panel.Param // its switch (kept with the patch): the tile sets it
	openNow  atomic.Bool  // it lets through now
	gain     float32      // the audio's, fading to open or closed (on the audio thread)
	targets  []psalm.Receiver
}

type gateSections struct {
	not   bool
	names map[string]bool // none: every section
}

// ParseGateSections reads the sections a gate plays in: names by commas ("Verse, Chorus"), "not"
// first for all but those, "all" (or nothing) for every one
func ParseGateSections(text string) (gateSections, error) {
	t := strings.TrimSpace(text)
	gs := gateSections{}
	if strings.EqualFold(t, "all") || t == "" {
		return gs, nil
	}
	if low := strings.ToLower(t); strings.HasPrefix(low, "not ") {
		gs.not, t = true, strings.TrimSpace(t[4:])
	}
	gs.names = map[string]bool{}
	for _, n := range strings.FieldsFunc(t, func(r rune) bool { return r == ',' || r == '\n' || r == ';' }) {
		if n = strings.TrimSpace(n); n != "" {
			gs.names[n] = true
		}
	}
	if len(gs.names) == 0 {
		return gs, fmt.Errorf("the sections it plays in: Verse, Chorus (or not Intro, or all)")
	}
	return gs, nil
}

// plays tells whether the gate plays in a section
func (gs *gateSections) plays(section string) bool {
	if len(gs.names) == 0 {
		return true
	}
	return gs.names[section] != gs.not
}

// NewGate returns a gate of a name playing in every section
func NewGate(name string) *Gate {
	g := &Gate{Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo})}
	g.SetSelf(g)
	g.song = SongOf(g.Env())
	g.name.Store(&name)
	g.sections.Store(&gateSections{})
	g.on.Store(true)
	g.song.addGate(g)
	return g
}

// Name is the gate's name (its tile's)
func (g *Gate) Name() string { return *g.name.Load() }

// open tells whether it lets through at a tick
func (g *Gate) open(tick psalm.CtrlTick) bool {
	if !g.on.Load() {
		return false
	}
	if pos, ok := g.song.At(tick, g.Config().PPQ); ok && !pos.Ended {
		return g.sections.Load().plays(pos.Section.Name)
	}
	return true
}

func (g *Gate) CtrlTick(tick psalm.CtrlTick) { g.openNow.Store(g.open(tick)) }

// Receive lets a trigger or a value (input 0) through when open
func (g *Gate) Receive(v any, i int) {
	if m, ok := psalm.MsgVal(v); ok {
		g.ReceiveMessage(m)
		return
	}
	if g.openNow.Load() {
		g.Send(v, i)
	}
}

// ReceiveMessage lets notes through to the synths its notes cable goes to when open; a note let go
// always
func (g *Gate) ReceiveMessage(msg psalm.Message) {
	if cmd, _ := msg.StrVal("command"); cmd == "trigger" {
		if _, off := msg.StrVal("noteOff"); !off && !g.openNow.Load() {
			return
		}
	}
	for _, t := range g.targets {
		t.ReceiveMessage(msg)
	}
}

// SetTargets sets the synths it lets notes through to (on the audio thread)
func (g *Gate) SetTargets(targets []psalm.Receiver) { g.targets = targets }

// Synthesize lets the sound through, fading in and out (5 ms)
func (g *Gate) Synthesize() {
	out := g.Output(0).Channels
	in := g.Input(0)
	target := float32(0)
	if g.openNow.Load() {
		target = 1
	}
	step := float32(1 / (0.005 * g.Config().SampleRate))
	gain := g.gain
	for j := range out[0] {
		if gain < target {
			gain = min(target, gain+step)
		} else if gain > target {
			gain = max(target, gain-step)
		}
		for c := range out {
			x := float32(0)
			if in.IsConnected() {
				x = in.Channels[c%len(in.Channels)][j]
			}
			out[c][j] = x * gain
		}
	}
	g.gain = gain
}

// close takes the gate out of the song's parts
func (g *Gate) close() { g.song.removeGate(g) }

// the gates of a song: its parts, in the order they were made
type gateList struct {
	mu    sync.Mutex
	gates []*Gate
}

func (s *Song) addGate(g *Gate) {
	s.gates.mu.Lock()
	defer s.gates.mu.Unlock()
	s.gates.gates = append(s.gates.gates, g)
}

func (s *Song) removeGate(g *Gate) {
	s.gates.mu.Lock()
	defer s.gates.mu.Unlock()
	s.gates.gates = slices.DeleteFunc(s.gates.gates, func(x *Gate) bool { return x == g })
}

// Gates are the parts of the song: its gates
func (s *Song) Gates() []*Gate {
	s.gates.mu.Lock()
	defer s.gates.mu.Unlock()
	return slices.Clone(s.gates.gates)
}

// partCues shows the gates of a song as tiles: a tap switches one on or off, those letting through
// now light up
type partCues struct{ s *Song }

func (c partCues) CueNames() []string {
	var names []string
	for _, g := range c.s.Gates() {
		names = append(names, g.Name())
	}
	return names
}

func (c partCues) CueState() panel.CueState {
	st := panel.CueState{Playing: -1, Queued: -1}
	for _, g := range c.s.Gates() {
		st.Names = append(st.Names, g.Name())
		st.Enabled = append(st.Enabled, g.on.Load())
		st.Active = append(st.Active, g.openNow.Load())
	}
	return st
}

func (c partCues) QueueCue(int) {}

func (c partCues) EnableCue(i int, on bool) {
	if gs := c.s.Gates(); i >= 0 && i < len(gs) {
		gs[i].setOn(on)
	}
}

// setOn switches the gate on or off: by its setting (kept with the patch), or at once without one
func (g *Gate) setOn(on bool) {
	if g.onParam != nil {
		v := 0.0
		if on {
			v = 1
		}
		g.onParam.Set(v)
	}
	g.on.Store(on)
}

// GateHelp is what to know to type a gate's sections
const GateHelp = "the sections it plays in: `Verse, Chorus`, or `not Intro`, or `all`\n" +
	"what goes through it: sound (in, out), notes (a keyboard, notes, a script to a synth), triggers; a note let go always goes through\n" +
	"its name is its tile in the song's side panel: a tap switches the part off and on, it lights up when it plays\n" +
	"Ctrl+Enter (or Run) takes them"

// gateSettings are a gate's settings kept with a patch: on or off
type gateSettings struct{ g *Gate }

func (st gateSettings) Values() map[string]float64 {
	if st.g.on.Load() {
		return map[string]float64{"on": 1}
	}
	return map[string]float64{"on": 0}
}

func (st gateSettings) Apply(values map[string]float64) {
	if v, ok := values["on"]; ok {
		st.g.setOn(v >= 0.5)
	}
}

// --- the kinds ---

// laneKind is the kind of lanes
func laneKind() Kind {
	return Kind{Name: "lane", Group: "Song", Outs: []Port{{Name: "value", Type: Value}, {Name: "mod", Type: Mod}},
		Make: func(string) (*Made, error) {
			l := NewLane()
			return &Made{Module: l,
				Texts:   []Text{{Key: "curves", Label: "curves", Value: DefaultLane, Help: LaneHelp}},
				SetText: func(_, v string) error { return l.SetCurves(v) },
				Mod:     func(int) func() float64 { return l.Value }}, nil
		}}
}

// gateKind is the kind of gates: every one a part of the song
func gateKind() Kind {
	return Kind{Name: "gate", Group: "Song",
		Ins:  []Port{{Name: "in", Type: Audio}, {Name: "notes", Type: Notes}, {Name: "trigger", Type: Trigger}},
		Outs: []Port{{Name: "out", Type: Audio}, {Name: "notes", Type: Notes}, {Name: "trigger", Type: Trigger}},
		Make: func(string) (*Made, error) {
			g := NewGate("part")
			return &Made{Module: g, Notes: g.SetTargets, Close: g.close,
				Texts: []Text{{Key: "name", Label: "name (its tile)", Value: "part"}, {Key: "sections", Label: "sections", Value: "all", Help: GateHelp}},
				SetText: func(key, v string) error {
					if key == "name" {
						n := strings.TrimSpace(v)
						if n == "" {
							return fmt.Errorf("a name for its tile")
						}
						g.name.Store(&n)
						return nil
					}
					gs, err := ParseGateSections(v)
					if err == nil {
						g.sections.Store(&gs)
					}
					return err
				},
				Edit: func(page *panel.Panel, name, _ string) Settings { // its switch, which its tile sets too
					page.Section("Gate")
					g.onParam = page.Toggle(name+" on", panel.Func(func(v float64) { g.on.Store(v >= 0.5) }), panel.Label("On"), panel.Default(1))
					return gateSettings{g}
				},
			}, nil
		}}
}
