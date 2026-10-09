package arrange

import (
	"math"

	"github.com/almerlucke/genny"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
)

// Stream is a part played by generators instead of written notes: a rhythm generator gives the
// steps, a template the message of every note. Generators in the template give every note new
// values (a slice of values splits a note into a chord, like templ) and template.NewParameter
// entries take the parameters of the section (Section.Params). A stream plays live, so a song that
// loops is never played the same way twice
type Stream struct {
	Rhythm  genny.Generator[float64] // length of every step in beats, a negative step is a rest
	Message template.Template        // the message of every step: for notes "pitch" or "frequency" and any setting of the synth
	Amp     any                      // amplitude of the notes: a number or a genny.Generator[float64] (0 is 1)
	Legato  any                      // part of its step a note sounds: a number or a genny.Generator[float64] (0 is 0.9)
	Output  Output                   // what is sent: notes for a poly synth (the default), messages or values for any module
}

// Output is what a stream sends every step
type Output int

const (
	// Notes sends a trigger per note (per message of the template) for a poly synth, with the
	// amplitude and the length of the note
	Notes Output = iota
	// Messages sends the messages of the template as they are, for any module that takes messages:
	// effects, filters, mixer strips, players, templ. Add "bang": true for modules that play on a
	// bang (a player, a templ). Amp and Legato are not used
	Messages
	// Values sends the "value" of the template as a plain number, for a value input of a module
	// (a filter cutoff, a mixer strip volume). Use Form.NewStream to connect it to an input
	Values
)

// StreamScore returns the stream of a part for a section, nil when the part doesn't play in it.
// It is asked again at the start of every section (also when the song loops): return the same
// stream to go on where it was, a new one to start fresh at the start of the section
type StreamScore func(s Section) *Stream

// Streamer plays the streams of a part through the song, a control module made by Form.Stream
type Streamer struct {
	*psalm.Module
	form    *Form
	score   StreamScore
	stream  *Stream
	section int
	loop    int
	next    float64 // tick of the next step
	params  []*template.Parameter
	gain    float64
}

// Stream adds a part played by streams: it creates a streamer, connects it to value input 0 of a
// module (the trigger input of a poly synth, the message input of most modules) and adds it to
// the patch
func (f *Form) Stream(p psalm.Patcher, target psalm.Modular, name string, sc StreamScore) *Streamer {
	s := f.NewStream(p, name, sc)
	target.ValueIn(s)
	return s
}

// NewStream creates a streamer added to the patch but not connected: connect it yourself, to
// another value input or to several modules: filter.ValueIn(streamer, 0, 1) drives input 1
func (f *Form) NewStream(p psalm.Patcher, name string, sc StreamScore) *Streamer {
	s := &Streamer{Module: psalm.NewControl(), form: f, score: sc, section: -1, loop: -1, gain: 1}
	s.SetSelf(s)
	s.SetID(name)
	s.AddTo(p)
	return s
}

// SetGain scales the amplitude of the notes that follow
func (s *Streamer) SetGain(gain float64) {
	s.gain = gain
}

// Restart starts the part again from the start of the song (psalm.Restarter): the stream of the
// first section is asked for again
func (s *Streamer) Restart() {
	s.form.Transport().restart()
	s.section, s.loop, s.stream, s.next = -1, -1, nil, 0
}

// number returns a number or the next value of a generator
func number(v any, otherwise float64) float64 {
	switch n := v.(type) {
	case nil:
		return otherwise
	case float64:
		if n == 0 {
			return otherwise
		}
		return n
	case int:
		if n == 0 {
			return otherwise
		}
		return float64(n)
	case genny.Generator[float64]:
		return n.Generate()
	}
	return otherwise
}

func (s *Streamer) CtrlTick(tick psalm.CtrlTick) {
	t := tick.TickTimestamp - 1
	ppq := s.Config().PPQ
	pos := s.form.Transport().At(t, ppq)
	if pos.Ended {
		s.stream = nil
		return
	}

	// a new section (or the same one in the next loop): ask for its stream
	if pos.Section.Index != s.section || pos.Loop != s.loop {
		s.section, s.loop = pos.Section.Index, pos.Loop
		stream := s.score(pos.Section)
		if stream != s.stream || stream == nil {
			s.next = float64(t) // a new stream starts with the section
		}
		s.stream = stream
		s.params = s.params[:0]
		for k, v := range pos.Section.Params {
			s.params = append(s.params, template.NewParameter(k, v))
		}
	}

	st := s.stream
	if st == nil || st.Rhythm == nil {
		return
	}

	// play every step that starts at this tick (zero length steps are skipped after a few)
	for guard := 0; float64(t) >= math.Round(s.next) && guard < 64; guard++ {
		if st.Rhythm.Done() && !st.Rhythm.Continuous() {
			s.stream = nil // the rhythm ended, silent until the next section
			return
		}
		step := st.Rhythm.Generate()
		if step == 0 {
			continue
		}
		s.next += math.Abs(step) * float64(ppq)
		if step > 0 {
			s.play(st, step, pos.Section, tick)
		}
	}
}

// play sends the notes, messages or values of a step
func (s *Streamer) play(st *Stream, step float64, sec Section, tick psalm.CtrlTick) {
	if st.Message == nil || (st.Message.Done() && !st.Message.Continuous()) {
		return
	}
	if len(s.params) > 0 {
		st.Message.SetParameters(s.params)
	}

	switch st.Output {
	case Messages:
		for _, m := range st.Message.Generate() {
			s.Send(psalm.Message(m), 0)
		}
		return
	case Values:
		for _, m := range st.Message.Generate() {
			if v, ok := m["value"]; ok {
				s.Send(v, 0)
			}
		}
		return
	}

	amp := number(st.Amp, 1) * s.gain * sec.Dynamics
	seconds := step * number(st.Legato, 0.9) * 60 / s.BPM()

	for _, m := range st.Message.Generate() {
		s.Send(psalm.Message{
			"command":   "trigger",
			"duration":  seconds,
			"amplitude": amp,
			"tick":      tick,
			"message":   psalm.Message(m),
		}, 0)
	}
}

// SendParams sends the parameters of every section (Section.Params) as a message to modules at the
// start of the section: templ modules take them as template parameters, synths and effects as
// settings (a granulator takes "mix" or "freeze"), so sections become scenes
func (a *Arrangement) SendParams(modules ...psalm.Modular) *Arrangement {
	return a.OnSection(func(pos Position) {
		if len(pos.Section.Params) == 0 {
			return
		}
		for _, m := range modules {
			m.Receive(psalm.Message(pos.Section.Params), 0)
		}
	})
}
