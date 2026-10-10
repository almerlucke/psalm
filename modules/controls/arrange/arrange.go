// Package arrange builds songs from sections: a Form lists the sections (a name, a number of bars
// and a dynamics level), parts are written per section and compiled into score sequencers over
// the whole song, and the Arrangement module follows the song position to run automation lanes
// and section and bar events
package arrange

import (
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/score"
)

// Section is a part of the song form
type Section struct {
	Name     string
	Bars     int
	Dynamics float64        // scales the amplitude of every part in the section, 0 is 1
	Params   map[string]any // parameters of the section: streams take them in their templates (template.NewParameter), SendParams sends them to modules
	Index    int            // position in the form, set by NewForm
	StartBar int            // first bar in the song, set by NewForm

	BeatsPerBar float64 // the bar of the form, set when a part is written for the section
}

// Form is the song form: its sections in order and the beats per bar
type Form struct {
	Sections    []Section
	BeatsPerBar float64
	Loop        bool // the song starts again after the last section

	transport *Transport
	partsMu   sync.Mutex
	parts     []*part // the parts and streams, in the order they were added
}

// NewForm creates a form of sections with 4 beats per bar that loops
func NewForm(sections ...Section) *Form {
	f := &Form{BeatsPerBar: 4, Loop: true}
	bar := 0
	for i, s := range sections {
		s.Index = i
		s.StartBar = bar
		if s.Dynamics == 0 {
			s.Dynamics = 1
		}
		bar += s.Bars
		f.Sections = append(f.Sections, s)
	}
	return f
}

// Bars returns the length of the song in bars
func (f *Form) Bars() int {
	n := 0
	for _, s := range f.Sections {
		n += s.Bars
	}
	return n
}

// Beats returns the length of the song in beats
func (f *Form) Beats() float64 {
	return float64(f.Bars()) * f.BeatsPerBar
}

// Section returns the section with a name
func (f *Form) Section(name string) (Section, bool) {
	for _, s := range f.Sections {
		if s.Name == name {
			return s, true
		}
	}
	return Section{}, false
}

// PositionAt returns the position at a tick (counted from the start of the song) for a PPQ, when
// the song plays as written: every section in order. The transport says where it is while it plays
func (f *Form) PositionAt(t int64, ppq int) Position {
	songTicks := int64(f.Beats() * float64(ppq))
	var pos Position
	if songTicks <= 0 {
		return pos
	}
	pos.Loop = int(t / songTicks)
	if !f.Loop && pos.Loop > 0 {
		t = songTicks - 1 // stays at the end
		pos.Loop = 0
		pos.Ended = true
	}
	t %= songTicks
	pos.Tick = t
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

// Schedule returns the start time of every section, at the current tempo
func (f *Form) Schedule() string {
	var b strings.Builder
	barSec := f.BeatsPerBar * 60.0 / psalm.BPM()
	for _, s := range f.Sections {
		fmt.Fprintf(&b, "%6.1fs  bar %3d  %-12s %d bars\n", float64(s.StartBar)*barSec, s.StartBar+1, s.Name, s.Bars)
	}
	end := "end"
	if f.Loop {
		end = "(loops)"
	}
	fmt.Fprintf(&b, "%6.1fs  %s\n", float64(f.Bars())*barSec, end)
	return b.String()
}

// Score writes the notes of a part for a section, nil when the part doesn't play in it
type Score func(s Section) []score.Note

// Compile builds the notes of a part over the whole song: the notes of every section, rests for
// the sections it doesn't play in, the amplitude scaled by the dynamics of each section. It panics
// when a section doesn't last exactly its bars, naming the part and the section
func (f *Form) Compile(name string, sc Score) []score.Note {
	notes, _ := f.compile(name, sc)
	return notes
}

// compile is Compile, also telling per section whether the part plays a note in it
func (f *Form) compile(name string, sc Score) ([]score.Note, []bool) {
	var notes []score.Note
	plays := make([]bool, len(f.Sections))
	for i, s := range f.Sections {
		expect := float64(s.Bars) * f.BeatsPerBar
		s.BeatsPerBar = f.BeatsPerBar
		ns := sc(s)
		if ns == nil {
			notes = append(notes, score.Rest(expect))
			continue
		}
		for _, n := range ns {
			plays[i] = plays[i] || !n.IsRest()
		}
		if got := score.Beats(ns); math.Abs(got-expect) > 1e-6 {
			panic(fmt.Sprintf("arrange: %s lasts %v beats in the %s, expected %v (%d bars)", name, got, s.Name, expect, s.Bars))
		}
		for _, n := range ns {
			if s.Dynamics != 1 && !n.IsRest() {
				if n.Amp == 0 {
					n.Amp = 1
				}
				n.Amp *= s.Dynamics
			}
			notes = append(notes, n)
		}
	}
	return notes, plays
}

// Part compiles a part, creates a sequencer for it that follows the transport of the form (in
// order, looping with the song, skipping and repeating sections as the transport does), connects it
// to the synth and adds it to the patch
func (f *Form) Part(p psalm.Patcher, synth psalm.Modular, name string, cfg score.Config, sc Score) *score.Sequencer {
	cfg.Loop = f.Loop
	cfg.Length = f.Beats()
	notes, plays := f.compile(name, sc)
	seq := score.NewSequencer(notes, cfg)
	tr, ppq := f.Transport(), seq.Config().PPQ
	f.addPart(&part{name: name, enable: seq.SetEnabled, enabled: seq.Enabled, active: func() bool {
		i := tr.playing.Load()
		return i >= 0 && int(i) < len(plays) && plays[i]
	}})
	seq.SetClock(func(tick int64) (int64, bool) {
		pos := tr.At(tick, ppq)
		return pos.Tick, !pos.Ended
	})
	seq.AddTo(p)
	synth.ValueIn(seq)
	return seq
}

// PerBar writes a section bar by bar. A bar without notes (nil) rests until the end of the bar:
// a whole bar of rest, nothing when a longer note before it already lasts through the bar
func PerBar(s Section, f func(bar int) []score.Note) []score.Note {
	var notes []score.Note
	beats := 0.0
	for bar := range s.Bars {
		ns := f(bar)
		if ns == nil {
			if s.BeatsPerBar <= 0 {
				panic("arrange: PerBar: a bar without notes needs the length of a bar, write the bars in a part of the form")
			}
			if rest := float64(bar+1)*s.BeatsPerBar - beats; rest > 1e-9 {
				ns = []score.Note{score.Rest(rest)}
			}
		}
		notes = append(notes, ns...)
		beats += score.Beats(ns)
	}
	return notes
}

// Only plays a score in one section only, a rest in the others: for a part written for a section
func Only(section string, sc Score) Score {
	return func(s Section) []score.Note {
		if s.Name != section {
			return nil
		}
		return sc(s)
	}
}

// Bars writes a part bar by bar in one section only (Only and PerBar in one): f gets the section
// and the bar (from 0), a bar without notes (nil) is a rest
func Bars(section string, f func(s Section, bar int) []score.Note) Score {
	return Only(section, func(s Section) []score.Note {
		return PerBar(s, func(bar int) []score.Note { return f(s, bar) })
	})
}

// In returns true when the section is one of the names, for parts that play in some sections
func In(s Section, names ...string) bool {
	for _, n := range names {
		if s.Name == n {
			return true
		}
	}
	return false
}
