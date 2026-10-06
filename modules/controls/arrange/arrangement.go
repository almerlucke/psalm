package arrange

import (
	"fmt"
	"math"
	"strings"

	"github.com/almerlucke/psalm"
)

// Position is a moment in the song
type Position struct {
	Beat     float64 // beats since the start of the song (of the current loop)
	Bar      int     // bar in the song, from 0
	Section  Section
	Progress float64 // 0-1 through the section
	Loop     int     // how many times the song looped
}

// SectionBar returns the bar in the current section, from 0
func (p Position) SectionBar() int {
	return p.Bar - p.Section.StartBar
}

// Curve gives a value for the progress (0-1) through a section
type Curve func(progress float64) float64

// Const is a curve that stays at a value
func Const(v float64) Curve {
	return func(float64) float64 { return v }
}

// Linear moves in a straight line from one value to another
func Linear(from, to float64) Curve {
	return func(p float64) float64 { return from + (to-from)*p }
}

// Exp moves exponentially from one value to another (both above 0), even steps per octave, for
// frequencies like a cutoff
func Exp(from, to float64) Curve {
	return func(p float64) float64 { return from * math.Pow(to/from, p) }
}

// Smooth eases in and out between two values
func Smooth(from, to float64) Curve {
	return func(p float64) float64 { return from + (to-from)*(0.5-0.5*math.Cos(math.Pi*p)) }
}

// Lane is an automation lane: a curve per section and a value for the sections without one
type Lane struct {
	Default  float64
	Sections map[string]Curve
}

// value returns the value of the lane at a position
func (l Lane) value(pos Position) float64 {
	if c, ok := l.Sections[pos.Section.Name]; ok {
		return c(pos.Progress)
	}
	return l.Default
}

// automation sends a lane to a module
type automation struct {
	lane  Lane
	send  func(v float64)
	last  float64
	valid bool
}

// Arrangement follows the song position: it runs the automation lanes, calls the section and bar
// events and prints the sections as they start. Add it to the patch
type Arrangement struct {
	*psalm.Module
	form        *Form
	automations []*automation
	onSection   []func(Position)
	onBar       []func(Position)
	print       bool
	tempo       *Lane
	pushed      bool    // a tempo config of the arrangement is on the config stack
	base        float64 // the tempo below it
	section     int
	bar         int
	every       int64 // ticks between automation updates
}

// NewArrangement creates an arrangement following a form
func NewArrangement(form *Form) *Arrangement {
	a := &Arrangement{
		Module:  psalm.NewControl(),
		form:    form,
		section: -1,
		bar:     -1,
		every:   max(1, int64(psalm.PPQ()/48)), // 48 updates per beat
	}
	a.SetSelf(a)
	return a
}

// PrintSections prints the name of every section as it starts
func (a *Arrangement) PrintSections(print bool) *Arrangement {
	a.print = print
	return a
}

// Automate sends a lane to a value input of a module (many modules take a setting on their value
// inputs, a filter its cutoff on input 1). Values are only sent when they change
func (a *Arrangement) Automate(lane Lane, to psalm.Modular, input int) *Arrangement {
	a.automations = append(a.automations, &automation{lane: lane, send: func(v float64) { to.Receive(v, input) }})
	return a
}

// AutomateMessage sends a lane as a message with a key, for settings without a value input
func (a *Arrangement) AutomateMessage(lane Lane, to psalm.Modular, key string) *Arrangement {
	a.automations = append(a.automations, &automation{lane: lane, send: func(v float64) {
		to.Receive(psalm.Message{key: v}, 0)
	}})
	return a
}

// AutomateFunc calls a function with the values of a lane
func (a *Arrangement) AutomateFunc(lane Lane, f func(v float64)) *Arrangement {
	a.automations = append(a.automations, &automation{lane: lane, send: f})
	return a
}

// Tempo automates the tempo in BPM: the arrangement pushes a config with the new tempo, psalm and
// the sequencers follow it (changes are updated 48 times per beat, so a ritardando is smooth). Where the
// lane is 0 (the default when a section has no curve) the tempo returns to the one the arrangement
// started with, so a lane only needs curves for the sections that change tempo:
//
//	Tempo(arrange.Lane{Sections: map[string]arrange.Curve{"Outro": arrange.Linear(120, 90)}})
//
// The song position is counted in ticks, so the parts stay in time with each other
func (a *Arrangement) Tempo(lane Lane) *Arrangement {
	a.tempo = &lane
	return a
}

// tempoResolution is the smallest tempo change (in BPM) the arrangement passes on
const tempoResolution = 0.01

// setTempo replaces the tempo config of the arrangement, 0 removes it. The arrangement pops its own
// config before pushing a new one, so the config stack doesn't grow
func (a *Arrangement) setTempo(bpm float64) {
	if a.pushed && math.Abs(bpm-psalm.BPM()) < tempoResolution {
		return
	}
	if a.pushed {
		psalm.PopConfig()
		a.pushed = false
	}
	if bpm <= 0 || math.Abs(bpm-psalm.BPM()) < tempoResolution {
		return
	}
	cfg := psalm.CurrentConfig()
	a.base = cfg.BPM
	cfg.BPM = bpm
	psalm.PushConfig(cfg)
	a.pushed = true
}

// Schedule returns the start time of every section, following the tempo lane
func (a *Arrangement) Schedule() string {
	if a.tempo == nil {
		return a.form.Schedule()
	}
	var b strings.Builder
	base := psalm.BPM() // the tempo of the sections without a curve
	if a.pushed {
		base = a.base
	}
	ppq := int64(psalm.PPQ())
	step := max(1, ppq/48)
	seconds := 0.0
	at := func(t int64) float64 {
		bpm := a.tempo.value(a.PositionAt(t))
		if bpm <= 0 {
			bpm = base
		}
		return bpm
	}
	songTicks := int64(a.form.Beats() * float64(ppq))
	starts := map[int]float64{}
	for t := int64(0); t < songTicks; t += step {
		pos := a.PositionAt(t)
		if _, ok := starts[pos.Section.Index]; !ok && pos.Bar == pos.Section.StartBar {
			starts[pos.Section.Index] = seconds
		}
		seconds += float64(step) / float64(ppq) * 60 / at(t)
	}
	for _, s := range a.form.Sections {
		fmt.Fprintf(&b, "%6.1fs  bar %3d  %-12s %d bars\n", starts[s.Index], s.StartBar+1, s.Name, s.Bars)
	}
	end := "end"
	if a.form.Loop {
		end = "(loops)"
	}
	fmt.Fprintf(&b, "%6.1fs  %s\n", seconds, end)
	return b.String()
}

// OnSection calls f at the start of every section
func (a *Arrangement) OnSection(f func(pos Position)) *Arrangement {
	a.onSection = append(a.onSection, f)
	return a
}

// OnBar calls f at the start of every bar
func (a *Arrangement) OnBar(f func(pos Position)) *Arrangement {
	a.onBar = append(a.onBar, f)
	return a
}

// PositionAt returns the position at a tick (counted from the start of the patch)
func (a *Arrangement) PositionAt(t int64) Position {
	f := a.form
	ppq := int64(psalm.PPQ())
	songTicks := int64(f.Beats() * float64(ppq))
	var pos Position
	if songTicks <= 0 {
		return pos
	}
	pos.Loop = int(t / songTicks)
	if !f.Loop && pos.Loop > 0 {
		t = songTicks - 1 // stays at the end
		pos.Loop = 0
	}
	t %= songTicks
	pos.Beat = float64(t) / float64(ppq)
	barBeats := f.BeatsPerBar
	pos.Bar = int(pos.Beat / barBeats)
	for _, s := range f.Sections {
		if pos.Bar < s.StartBar+s.Bars {
			pos.Section = s
			break
		}
	}
	start := float64(pos.Section.StartBar) * barBeats
	pos.Progress = (pos.Beat - start) / (float64(pos.Section.Bars) * barBeats)
	return pos
}

func (a *Arrangement) CtrlTick(tick psalm.CtrlTick) {
	t := tick.TickTimestamp - 1
	pos := a.PositionAt(t)

	if pos.Bar != a.bar {
		a.bar = pos.Bar
		if pos.Section.Index != a.section || pos.Bar == pos.Section.StartBar {
			a.section = pos.Section.Index
			if a.print {
				fmt.Printf("bar %3d  %s\n", pos.Bar+1, pos.Section.Name)
			}
			for _, f := range a.onSection {
				f(pos)
			}
		}
		for _, f := range a.onBar {
			f(pos)
		}
	}

	if t%a.every == 0 {
		if a.tempo != nil {
			a.setTempo(a.tempo.value(pos))
		}
		for _, au := range a.automations {
			v := au.lane.value(pos)
			if !au.valid || v != au.last {
				au.send(v)
				au.last, au.valid = v, true
			}
		}
	}
}
