// Package arrange builds songs from sections: a Form lists the sections (a name, a number of bars
// and a dynamics level), parts are written per section and compiled into score sequencers over
// the whole song, and the Arrangement module follows the song position to run automation lanes
// and section and bar events
package arrange

import (
	"fmt"
	"math"
	"strings"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/score"
)

// Section is a part of the song form
type Section struct {
	Name     string
	Bars     int
	Dynamics float64 // scales the amplitude of every part in the section, 0 is 1
	Index    int     // position in the form, set by NewForm
	StartBar int     // first bar in the song, set by NewForm
}

// Form is the song form: its sections in order and the beats per bar
type Form struct {
	Sections    []Section
	BeatsPerBar float64
	Loop        bool // the song starts again after the last section
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
	var notes []score.Note
	for _, s := range f.Sections {
		expect := float64(s.Bars) * f.BeatsPerBar
		ns := sc(s)
		if ns == nil {
			notes = append(notes, score.Rest(expect))
			continue
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
	return notes
}

// Part compiles a part, creates a sequencer for it that loops with the song, connects it to the
// synth and adds it to the patch
func (f *Form) Part(p psalm.Patcher, synth psalm.Modular, name string, cfg score.Config, sc Score) *score.Sequencer {
	cfg.Loop = f.Loop
	cfg.Length = f.Beats()
	seq := score.NewSequencer(f.Compile(name, sc), cfg)
	seq.AddTo(p)
	synth.ValueIn(seq)
	return seq
}

// PerBar writes a section bar by bar
func PerBar(s Section, f func(bar int) []score.Note) []score.Note {
	var notes []score.Note
	for bar := range s.Bars {
		notes = append(notes, f(bar)...)
	}
	return notes
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
