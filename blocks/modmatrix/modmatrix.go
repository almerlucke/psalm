// Package modmatrix is a modulation matrix for a voice: a few slots, each taking a source (an
// envelope, an LFO, the velocity, the key, a random value per note, the mod wheel, or a value cabled in
// from outside) to a destination (a setting of the voice, by its key) over an amount. A voice
// fills in the values of its sources every sample, asks for what each destination is moved by, and
// applies that in its own way (octaves for a cutoff, semitones for a pitch, an added level):
//
//	m := modmatrix.New(dests)                      // dests: what the voice can move
//	m.Set(0, modmatrix.Slot{Src: modmatrix.Wheel, Dest: "cutoff", Amount: 0.5})
//	...
//	m.Values[modmatrix.LFO1] = lfo                 // every sample
//	m.Eval()
//	cutoff *= math.Exp2(m.Mod(dCutoff))            // the amount of the slots, times the span of the destination
//
// The settings of the slots are messages ("mod1Src", "mod1Dest", "mod1Amount") and are described
// for an editor (Describe), so they are saved in a preset and edited like any other setting.
package modmatrix

import (
	"fmt"
	"math"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/params"
)

// Source is where a modulation comes from
type Source int

const (
	None      Source = iota
	FilterEnv        // 0 to 1
	AmpEnv           // 0 to 1
	LFO1             // -1 to 1
	LFO2             // -1 to 1
	Velocity         // 0 to 1
	Key              // -1 to 1: the note, from five octaves below middle C to five above
	Random           // -1 to 1: a new value for every note
	Wheel            // 0 to 1: the mod wheel (and the aftertouch)
	Ext1             // 0 to 1: a value cabled in
	Ext2
	Ext3
	Ext4
	NumSources
)

// Exts is how many values can be cabled in from outside
const Exts = 4

// SourceNames are the names of the sources, by number
var SourceNames = []string{"none", "filter env", "amp env", "lfo 1", "lfo 2", "velocity", "key", "random", "wheel",
	"ext 1", "ext 2", "ext 3", "ext 4"}

// Slots is how many sources and destinations a matrix joins
const Slots = 8

// Slot joins a source to a destination (its key, "" for none)
type Slot struct {
	Src    Source
	Dest   string
	Amount float64 // -1 to 1, of the span of the destination
}

// Dest is a setting a voice lets the matrix move: its key (as in its params), the label shown for
// it, and its span: how far a full source (1) with a full amount (1) moves it, in the unit the voice
// applies (semitones, octaves, a level)
type Dest struct {
	Key   string
	Label string
	Span  float64
}

type slot struct {
	src    Source
	dest   int // -1 for none
	amount float64
}

// Matrix holds the slots of a voice
type Matrix struct {
	dests []Dest
	slots [Slots]slot
	used  [NumSources]bool
	out   []float64
	// Values are the sources, set by the voice before Eval (the Ext sources are taken from SetExt)
	Values [NumSources]float64
	ext    [Exts]float64 // where the cabled values are moved to
	extNow [Exts]float64 // where they are
	coef   float64       // the glide of the cabled values per sample
	sr     float64
	active bool
}

// New returns a matrix for the destinations a voice has, with every slot empty
func New(dests []Dest) *Matrix {
	m := &Matrix{dests: dests, out: make([]float64, len(dests))}
	for i := range m.slots {
		m.slots[i].dest = -1
	}
	return m
}

// Index returns the number of a destination by its key (-1 for none)
func (m *Matrix) Index(key string) int {
	for i, d := range m.dests {
		if d.Key == key {
			return i
		}
	}
	return -1
}

// Set fills a slot (a destination that doesn't exist leaves it empty)
func (m *Matrix) Set(i int, s Slot) {
	if i < 0 || i >= Slots {
		return
	}
	d := -1
	if s.Dest != "" {
		d = m.Index(s.Dest)
	}
	m.slots[i] = slot{src: s.Src, dest: d, amount: max(-1, min(s.Amount, 1))}
	m.refresh()
}

// Slot returns a slot
func (m *Matrix) Slot(i int) Slot {
	s := m.slots[i]
	out := Slot{Src: s.src, Amount: s.amount}
	if s.dest >= 0 {
		out.Dest = m.dests[s.dest].Key
	}
	return out
}

// refresh sees what is in use
func (m *Matrix) refresh() {
	m.used = [NumSources]bool{}
	m.active = false
	for _, s := range m.slots {
		if s.src != None && s.dest >= 0 && s.amount != 0 {
			m.used[s.src] = true
			m.active = true
		}
	}
}

// Active says whether any slot does anything
func (m *Matrix) Active() bool { return m.active }

// Uses says whether a source is joined to a destination
func (m *Matrix) Uses(src Source) bool { return m.used[src] }

// Targets says whether any slot moves a destination
func (m *Matrix) Targets(dest int) bool {
	for _, s := range m.slots {
		if s.dest == dest && s.src != None && s.amount != 0 {
			return true
		}
	}
	return false
}

// SetExt sets a value cabled in from outside (0 to 1), n from 0
func (m *Matrix) SetExt(n int, v float64) {
	if n >= 0 && n < Exts {
		m.ext[n] = max(0, min(v, 1))
	}
}

// Ext returns where a cabled value is moved to
func (m *Matrix) Ext(n int) float64 { return m.ext[n] }

// glide is how long a cabled value takes to get where it is moved to (a third of it)
const glide = 0.004

// Eval works out what each destination is moved by: call it every sample after setting Values
func (m *Matrix) Eval(sr float64) {
	if !m.active {
		return
	}
	if sr != m.sr {
		m.sr, m.coef = sr, 1-math.Exp(-1/(glide*sr))
	}
	for i := range m.ext {
		if m.extNow[i] != m.ext[i] {
			m.extNow[i] += (m.ext[i] - m.extNow[i]) * m.coef
			if math.Abs(m.ext[i]-m.extNow[i]) < 1e-6 {
				m.extNow[i] = m.ext[i]
			}
		}
		m.Values[Ext1+Source(i)] = m.extNow[i]
	}
	clear(m.out)
	for _, s := range m.slots {
		if s.src == None || s.dest < 0 || s.amount == 0 {
			continue
		}
		m.out[s.dest] += m.Values[s.src] * s.amount * m.dests[s.dest].Span
	}
}

// Mod returns what a destination is moved by, in its unit (0 when no slot moves it)
func (m *Matrix) Mod(dest int) float64 {
	if !m.active {
		return 0
	}
	return m.out[dest]
}

// Receive takes "mod<n>Src", "mod<n>Dest" (0 for none, then the destinations in order), "mod<n>Amount"
// and "ext<n>" (0 to 1) from a message, n from 1
func (m *Matrix) Receive(msg psalm.Message) {
	for i := range Slots {
		p := fmt.Sprintf("mod%d", i+1)
		s := m.Slot(i)
		set := false
		msg.IfIntVal(p+"Src", func(v int) { s.Src, set = Source(max(0, min(v, int(NumSources)-1))), true })
		msg.IfIntVal(p+"Dest", func(v int) {
			s.Dest, set = "", true
			if v > 0 && v <= len(m.dests) {
				s.Dest = m.dests[v-1].Key
			}
		})
		msg.IfFltVal(p+"Amount", func(v float64) { s.Amount, set = v, true })
		if set {
			m.Set(i, s)
		}
	}
	for i := range Exts {
		msg.IfFltVal(fmt.Sprintf("ext%d", i+1), func(v float64) { m.SetExt(i, v) })
	}
}

// Describe describes the slots, and the cabled values, as the settings of a voice (see params)
func (m *Matrix) Describe(group string) []params.Param {
	dests := []string{"none"}
	for _, d := range m.dests {
		dests = append(dests, d.Label)
	}
	var ps []params.Param
	for i := range Slots {
		s, p := m.Slot(i), fmt.Sprintf("mod%d", i+1)
		d := 0
		if k := m.Index(s.Dest); k >= 0 {
			d = k + 1
		}
		ps = append(ps,
			params.C(group, p+"Src", SourceNames, int(s.Src), params.Label(fmt.Sprintf("%d source", i+1))),
			params.C(group, p+"Dest", dests, d, params.Label(fmt.Sprintf("%d to", i+1))),
			params.K(group, p+"Amount", -1, 1, s.Amount, params.Label(fmt.Sprintf("%d amount", i+1))),
		)
	}
	for i := range Exts {
		ps = append(ps, params.K(group, fmt.Sprintf("ext%d", i+1), 0, 1, m.ext[i], params.Label(fmt.Sprintf("ext %d", i+1))))
	}
	return ps
}
