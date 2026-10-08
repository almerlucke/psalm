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
	Ended    bool    // the song doesn't loop and is over
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
	Sections Sections
}

// Sections is a lane by itself: a curve per section, 0 in the sections without one; Else sets
// another value for those. Where a lane is taken, both work:
//
//	Automate(arrange.Sections{"Coda": arrange.Exp(12000, 400)}.Else(12000), filter, 1)
//	AutomateMessage(arrange.Sections{"Chorus": arrange.Const(1)}, piano, "sustain")
type Sections map[string]Curve

// Else returns the lane with a value for the sections without a curve
func (s Sections) Else(v float64) Lane { return Lane{Default: v, Sections: s} }

// Laner is what the arrangement automates: a Lane, or Sections (0 elsewhere)
type Laner interface{ lane() Lane }

func (l Lane) lane() Lane     { return l }
func (s Sections) lane() Lane { return Lane{Sections: s} }

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
	pushed      bool    // the tempo lane changed the tempo
	base        float64 // the tempo before it did
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
func (a *Arrangement) Automate(lane Laner, to psalm.Modular, input int) *Arrangement {
	a.automations = append(a.automations, &automation{lane: lane.lane(), send: func(v float64) { to.Receive(v, input) }})
	return a
}

// AutomateMessage sends a lane as a message with a key, for settings without a value input, or
// to name the setting: "cutoff" of a filter, "volume" of a mixer strip
func (a *Arrangement) AutomateMessage(lane Laner, to psalm.Modular, key string) *Arrangement {
	a.automations = append(a.automations, &automation{lane: lane.lane(), send: func(v float64) {
		to.Receive(psalm.Message{key: v}, 0)
	}})
	return a
}

// AutomateFunc calls a function with the values of a lane
func (a *Arrangement) AutomateFunc(lane Laner, f func(v float64)) *Arrangement {
	a.automations = append(a.automations, &automation{lane: lane.lane(), send: f})
	return a
}

// Tempo automates the tempo in BPM: the arrangement sets the tempo of its patch, the patch and its
// sequencers follow it (changes are updated 48 times per beat, so a ritardando is smooth). Other
// patches keep their own tempo. Where the lane is 0 (the default when a section has no curve) the
// tempo returns to the one the arrangement started with, so a lane only needs curves for the
// sections that change tempo:
//
//	Tempo(arrange.Sections{"Outro": arrange.Linear(120, 90)})
func (a *Arrangement) Tempo(lane Laner) *Arrangement {
	l := lane.lane()
	a.tempo = &l
	return a
}

// tempoResolution is the smallest tempo change (in BPM) the arrangement passes on
const tempoResolution = 0.01

// setTempo sets the tempo of the patch, 0 returns it to the tempo before the lane changed it
func (a *Arrangement) setTempo(bpm float64) {
	env := a.Env()
	if bpm <= 0 {
		if a.pushed {
			env.SetBPM(a.base)
			a.pushed = false
		}
		return
	}
	if !a.pushed {
		if math.Abs(bpm-env.BPM()) < tempoResolution {
			return
		}
		a.base = env.BPM()
		a.pushed = true
	}
	if math.Abs(bpm-env.BPM()) >= tempoResolution {
		env.SetBPM(bpm)
	}
}

// timing returns the start time of every section and the length of one pass of the song in
// seconds, following the tempo lane
func (a *Arrangement) timing() (map[int]float64, float64) {
	base := a.Env().BPM() // the tempo of the sections without a curve
	if a.pushed {
		base = a.base
	}
	ppq := int64(a.Config().PPQ)
	step := max(1, ppq/48)
	seconds := 0.0
	starts := map[int]float64{}
	songTicks := int64(a.form.Beats() * float64(ppq))
	for t := int64(0); t < songTicks; t += step {
		pos := a.PositionAt(t)
		if _, ok := starts[pos.Section.Index]; !ok && pos.Bar == pos.Section.StartBar {
			starts[pos.Section.Index] = seconds
		}
		bpm := base
		if a.tempo != nil {
			if v := a.tempo.value(pos); v > 0 {
				bpm = v
			}
		}
		seconds += float64(step) / float64(ppq) * 60 / bpm
	}
	return starts, seconds
}

// Duration returns the length of one pass of the song in seconds, following the tempo lane
func (a *Arrangement) Duration() float64 {
	_, d := a.timing()
	return d
}

// Schedule returns the start time of every section, following the tempo lane
func (a *Arrangement) Schedule() string {
	starts, seconds := a.timing()
	var b strings.Builder
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

// Play makes a module sound only in some sections: it is unmuted at the start of those sections
// and muted at the start of the others. Modules that mute with a fade (mixer strips, anything with
// SetMute) use it; other modules are muted with SetMuted, a hard cut while the module keeps
// running (its generators go on). It panics on a section name the form doesn't have
func (a *Arrangement) Play(m psalm.Modular, sections ...string) *Arrangement {
	for _, name := range sections {
		if _, ok := a.form.Section(name); !ok {
			panic(fmt.Sprintf("arrange: Play %s in section %q, which the form doesn't have", m.ID(), name))
		}
	}
	return a.OnSection(func(pos Position) {
		mute := !In(pos.Section, sections...)
		if f, ok := m.(interface{ SetMute(bool) }); ok {
			f.SetMute(mute)
		} else {
			m.SetMuted(mute)
		}
	})
}

// OnBar calls f at the start of every bar
func (a *Arrangement) OnBar(f func(pos Position)) *Arrangement {
	a.onBar = append(a.onBar, f)
	return a
}

// At calls f at the start of a bar of a section (from 0), every time the song gets there: to
// start a recording, change a sound, anything that happens once at a moment. It panics on a
// section the form doesn't have or a bar it doesn't reach
func (a *Arrangement) At(section string, bar int, f func()) *Arrangement {
	s, ok := a.form.Section(section)
	if !ok || bar < 0 || bar >= s.Bars {
		panic(fmt.Sprintf("arrange: At bar %d of section %q, which the form doesn't have", bar, section))
	}
	return a.OnBar(func(pos Position) {
		if pos.Section.Name == section && pos.SectionBar() == bar {
			f()
		}
	})
}

// PositionAt returns the position at a tick (counted from the start of the patch)
func (a *Arrangement) PositionAt(t int64) Position {
	return a.form.PositionAt(t, a.Config().PPQ)
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
