package main

import (
	"errors"
	"math"
	"slices"
	"strconv"

	"github.com/almerlucke/psalm/modules/controls/panel/patcher/script"
)

// The key trigger: each key that is played starts the piece, made for that key (a script has p.key, a spec is moved by the distance from
// middle C), from the moment it is played or from the next point of the grid (snap). The piece of every key is made beforehand, so the
// audio thread only reads it. The notes go on while the key is held (key trigger "loop") or play to the end ("once"); with "loop" the
// notes are let go with the key

// keySet is the piece of each key, never changed once published
type keySet struct {
	stores [128]*store
	spec   bool // one piece for all keys, moved by the distance from middle C
}

// voice is a piece that plays for a key
type voice struct {
	st         *store
	start, end float64 // beats of the clock of the host; end is infinite until the key is let go
	salt       int     // its number: the notes it made are known by it
	loop       bool
	velocity   float64
	shift      float64
	on         bool
}

// snapBeats are the grids a key can be started on: not at all, a sixteenth, a beat, a bar of four
func snapBeats(v int) float64 { return [...]float64{0, 0.25, 1, 4}[min(max(v, 0), 3)] }

// keyAt takes a key played or let go, at a position in beats (the audio thread, without locks or allocation)
func (in *instance) keyAt(key int, velocity float64, on bool, beat float64) {
	mode := int(in.get(pKeys))
	if mode == 0 || key < 0 || key > 127 {
		return
	}
	v := &in.voices[key]
	if !on {
		if v.on && v.loop && math.IsInf(v.end, 1) {
			v.end = beat
			if v.end <= v.start {
				v.on = false // let go before it began
			}
		}
		return
	}
	if v.on {
		in.kill(v.salt, beat) // the key again: the old voice stops
		v.on = false
	}
	ks := in.keys.Load()
	if ks == nil || ks.stores[key] == nil {
		return // not made (yet)
	}
	start := beat
	if snap := snapBeats(int(in.get(pSnap))); snap > 0 {
		start = math.Ceil(beat/snap-1e-9) * snap
	}
	in.salt++
	shift := 0.0
	if ks.spec {
		shift = float64(key - 60)
	}
	*v = voice{st: ks.stores[key], start: start, end: math.Inf(1), salt: in.salt, loop: mode == 2, velocity: min(max(velocity, 0.01), 1), shift: shift, on: true}
}

func (in *instance) kill(salt int, beat float64) {
	if in.npending < len(in.pending) {
		in.pending[in.npending] = goEvent{beat: beat, kill: true, voice: salt}
		in.npending++
	}
}

func (in *instance) flushPending(put func(goEvent)) {
	for i := 0; i < in.npending; i++ {
		put(in.pending[i])
	}
	in.npending = 0
}

// collectVoices puts the notes of the voices in the beats [from, to)
func (in *instance) collectVoices(from, to float64, put func(goEvent)) {
	in.flushPending(put)
	for k := range in.voices {
		v := &in.voices[k]
		if !v.on || to <= v.start {
			continue
		}
		hi := to - v.start
		ending := v.end < to
		if ending {
			hi = min(hi, v.end-v.start)
		}
		in.emit(v.st, v.start, from-v.start, hi, v.salt, !v.loop, v.shift, v.velocity, put)
		switch {
		case ending:
			put(goEvent{beat: max(v.end, from), kill: true, voice: v.salt})
			v.on = false
		case !v.loop && to-v.start > v.st.length+v.st.overrun: // played to the end
			v.on = false
		}
	}
}

// endVoices lets go of the voices that sound when the key trigger is turned off
func (in *instance) endVoices(from float64, put func(goEvent)) {
	in.npending = 0
	for k := range in.voices {
		if v := &in.voices[k]; v.on {
			put(goEvent{beat: from, kill: true, voice: v.salt})
			v.on = false
		}
	}
}

// resetVoices forgets the voices (the transport started, stopped or jumped: the host lets go of the notes)
func (in *instance) resetVoices() {
	for k := range in.voices {
		in.voices[k].on = false
	}
	in.npending = 0
}

// keyOrder is the keys in the order they are made: from middle C outwards
var keyOrder = func() []int {
	var k []int
	for i := 0; i < 128; i++ {
		k = append(k, i)
	}
	slices.SortStableFunc(k, func(a, b int) int {
		da, db := math.Abs(float64(a)-60.5), math.Abs(float64(b)-60.5)
		switch {
		case da < db:
			return -1
		case da > db:
			return 1
		}
		return 0
	})
	return k
}()

var errStopped = errors.New("stopped")

// renderKeys makes the piece of every key and publishes them as they come (the middle first), so a piece that takes a while is
// playable at once for the keys that are used most. A key whose script fails stays silent; it is an error when none works
func (in *instance) renderKeys(rt **script.Runtime, text string, isScript bool, seed uint64) error {
	ks := &keySet{}
	if !isScript {
		l, err := renderSpec(text, seed)
		if err != nil {
			return err
		}
		st := &store{lines: map[int64]*lineData{0: l}, length: l.length, overrun: l.overrun, epoch: 1}
		for i := range ks.stores {
			ks.stores[i] = st
		}
		ks.spec = true
		in.keys.Store(ks)
		return nil
	}
	if *rt == nil {
		*rt = script.NewRuntime()
	}
	made, failed := 0, 0
	var first error
	for n, key := range keyOrder {
		select {
		case <-in.stop:
			return errStopped
		default:
		}
		(*rt).Flats = in.get(pNames) > 0.5
		renderMu.Lock()
		piece, err := (*rt).RenderKey(text, seed, nil, key)
		renderMu.Unlock()
		if key == 60 {
			for _, line := range piece.Log {
				in.say(sevInfo, "script: %s", line)
			}
		}
		if err != nil {
			if first == nil {
				first = errWithKey(key, err)
			}
			failed++
			continue
		}
		l := buildPiece(piece)
		ks.stores[key] = &store{lines: map[int64]*lineData{0: l}, length: l.length, overrun: l.overrun, epoch: 1}
		made++
		if n%16 == 15 { // what is made so far
			cp := *ks
			in.keys.Store(&cp)
		}
	}
	if made == 0 {
		return first
	}
	in.keys.Store(ks) // all of them (the last keys may have failed, which skips the batches above)
	if failed > 0 {
		in.say(sevWarn, "%d keys have no notes: %v", failed, first)
	}
	in.say(sevInfo, "%d keys made", made)
	return nil
}

type keyError struct {
	key int
	err error
}

func (e keyError) Error() string { return "key " + strconv.Itoa(e.key) + ": " + e.err.Error() }
func (e keyError) Unwrap() error { return e.err }

func errWithKey(key int, err error) error { return keyError{key, err} }
