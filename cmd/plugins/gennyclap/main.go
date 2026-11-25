// Command gennyclap is the spike of a CLAP plugin made of genny: a note effect that plays what the generators of
// genny make, in time with the transport of the host. Build it with build.sh, copy genny.clap to ~/.clap
package main

/*
#cgo CFLAGS: -I${SRCDIR}/../../../local/clap/include -I${SRCDIR}
#include "shim.h"
*/
import "C"

import (
	"math"
	"sync/atomic"
	"unsafe"

	"github.com/almerlucke/genny"
	"github.com/almerlucke/genny/euclid"
	"github.com/almerlucke/genny/groove"
	"github.com/almerlucke/genny/mutate"
	"github.com/almerlucke/genny/note"
	"github.com/almerlucke/genny/rest"
	"github.com/almerlucke/genny/scale"
)

const (
	stepsPerBeat = 4
	barSteps     = 32 // two bars of four beats
)

// the notes of the loop, rendered when the plugin is activated, read by the audio thread (never changed)
var line atomic.Pointer[[]note.Event]

//export genny_prepare
func genny_prepare() {
	genny.Seed(7)
	key := scale.New(57, scale.Minor)
	phrase := mutate.New([]int{7, 9, 11, 9, 8, 7, 4, 7}, mutate.Config[int]{SwapProb: 0.6})
	lead := note.Voice{
		Pitch:    rest.NewGated[float64](scale.NewDegrees(key, phrase), euclid.New(16, 7, 0)),
		Velocity: groove.NewAccent(4, 4, 4, 4),
		Offset:   groove.NewSwing(0.15),
	}
	events := lead.Render(barSteps)
	line.Store(&events)
}

// genny_collect writes the note on and off events that fall in the beats [from, to) of the transport
//
//export genny_collect
func genny_collect(from, to C.double, out *C.genny_event, capacity C.int) C.int {
	p := line.Load()
	if p == nil {
		return 0
	}
	buf := unsafe.Slice(out, int(capacity))
	n := 0
	put := func(beat float64, on bool, e note.Event) {
		if n >= len(buf) {
			return
		}
		buf[n] = C.genny_event{beat: C.double(beat), velocity: C.double(e.Velocity), key: C.int(math.Round(e.Pitch))}
		if on {
			buf[n].on = 1
		}
		n++
	}
	loop := float64(barSteps) / stepsPerBeat
	f, t := float64(from), float64(to)
	first := int(math.Floor(f/loop)) - 1 // a note that started in the cycle before can end in this block
	last := int(math.Floor(t / loop))
	for cycle := max(first, 0); cycle <= last; cycle++ {
		base := float64(cycle) * loop
		for _, e := range *p {
			start := base + e.Start/stepsPerBeat
			end := start + e.Duration/stepsPerBeat
			if start >= f && start < t {
				put(start, true, e)
			}
			if end >= f && end < t {
				put(end, false, e)
			}
		}
	}
	return C.int(n)
}

func main() {}
