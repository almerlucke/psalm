package panel

import "github.com/almerlucke/psalm"

// Target is where a control sends its value, on the audio thread
type Target interface {
	Set(v float64)
}

type targetFunc func(v float64)

func (f targetFunc) Set(v float64) { f(v) }

// Message sends the value as a message with a key: "cutoff" of a filter, "volume" or "send.hall"
// of a mixer strip, "sustain" of a piano, any setting a module takes by name
func Message(m psalm.Modular, key string) Target {
	return targetFunc(func(v float64) { m.Receive(psalm.Message{key: v}, 0) })
}

// Input sends the value to a value input of a module by number (a filter's cutoff is input 1)
func Input(m psalm.Modular, index int) Target {
	return targetFunc(func(v float64) { m.Receive(v, index) })
}

// Func calls a function with the value: anything else, like the tempo (p.Env().SetBPM)
func Func(f func(v float64)) Target { return targetFunc(f) }

// Targets sends the value to several targets
func Targets(targets ...Target) Target {
	return targetFunc(func(v float64) {
		for _, t := range targets {
			t.Set(v)
		}
	})
}
