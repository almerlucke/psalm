package poly

import (
	"sync"
	"sync/atomic"

	"github.com/almerlucke/psalm"
)

// Adopter is a synth that can take over what another one made like it has made: a Poly takes its
// voices (AdoptVoices), the organ and the piano, which are one module each, their whole inner
// state. Adopt returns false (and changes nothing) when other isn't made like it; it is called on
// the audio thread, between two buffers, and doesn't allocate
type Adopter interface {
	psalm.Modular
	Adopt(other psalm.Modular) bool
}

// Swap changes a synth in a patch into the synth made at a preset: the new one is made off the
// audio thread (a goroutine of its own, so a page that chooses a preset doesn't wait for 12 strings
// to be made), then its voices go into the live synth between two buffers. What a preset is made
// of that its description doesn't say (a model, a mode list, a word) comes with it, which moving the
// settings of the old preset to the new one's can't do. Choosing again before the last one is in
// leaves the last one out
type Swap struct {
	do     func(func())
	target Adopter
	build  func(name string) psalm.Modular
	seq    atomic.Int64
	wg     sync.WaitGroup
}

// NewSwap makes the swap of target: build makes a synth at a preset (the same kind as target, on
// any goroutine), do runs a change on the audio thread (psalm.Psalm.Do; nil runs it at once)
func NewSwap(do func(func()), target psalm.Modular, build func(name string) psalm.Modular) *Swap {
	a, _ := target.(Adopter)
	return &Swap{do: do, target: a, build: build}
}

// Can tells whether the target can adopt another (a synth that can't: a Swap of it does nothing)
func (s *Swap) Can() bool { return s.target != nil }

// Load makes the synth at a preset and swaps it in; it returns at once (Wait for it to be in)
func (s *Swap) Load(name string) {
	if s.target == nil {
		return
	}
	id := s.seq.Add(1)
	s.wg.Add(1)
	go func() {
		m := s.build(name)
		in := func() {
			if m != nil && s.seq.Load() == id {
				s.target.Adopt(m)
			}
			s.wg.Done()
		}
		if m == nil || s.seq.Load() != id {
			s.wg.Done()
		} else if s.do == nil {
			in()
		} else {
			s.do(in)
		}
	}()
}

// Wait waits for the swaps asked for to be in (or left out)
func (s *Swap) Wait() { s.wg.Wait() }
