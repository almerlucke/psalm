package panel

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/keyboard"
)

// MIDI moves the controls from the control changes (CC) of a keyboard or controller: a control
// bound to a controller follows it. Bind controllers in code (Param.BindCC) or learn them: arm a
// control (Param.Learn, or MIDI learn on the page) and move a knob or fader on the controller.
//
// A fader or knob follows the controller over its whole travel (by its curve), a toggle is on from
// 64 up (a latching button, or a fader; a momentary button holds it on while it is pressed), a
// button is pressed when the controller goes up past 64 and a select picks an option over the
// controller's travel.
//
// Buttons and toggles (and the tiles of a cue list) can be bound to notes too, for the pads of a
// controller that send notes: a note on presses a button and switches a toggle over.

// pack packs a binding for an atomic, 0 is none
func pack(b Binding) int32 {
	note := 0
	if b.Note {
		note = 1
	}
	return int32(note<<16|b.Channel<<8|b.Controller) + 1
}

func unpack(code int32) Binding {
	code--
	return Binding{Channel: int(code>>8) & 0xff, Controller: int(code & 0xff), Note: code>>16 != 0}
}

// MIDI connects a keyboard (from the keyboard package): its control changes move the controls bound
// to them, its notes press the buttons bound to them, on the audio thread
func (pn *Panel) MIDI(kb *keyboard.Keyboard) *Panel {
	pn.midiFrom(func(f func(midi.Event, psalm.CtrlTick)) { kb.OnEvent(f) }, true)
	return pn
}

// midiFrom takes the events of a MIDI source (subscribe gives them to a function): the control
// changes, and the notes when notes is true
func (pn *Panel) midiFrom(subscribe func(func(midi.Event, psalm.CtrlTick)), notes bool) {
	subscribe(func(e midi.Event, _ psalm.CtrlTick) {
		switch e.Message.Kind() {
		case midi.ControlChange:
			pn.ControlChange(e.Message.Channel(), e.Message.Controller(), e.Message.Value())
		case midi.NoteOn:
			if notes {
				pn.NoteOn(e.Message.Channel(), e.Message.Note(), e.Message.Velocity())
			}
		}
	})
}

// MIDIControllers connects a keyboard for its control changes only: its keys play notes (a synth),
// they don't press buttons and MIDI learn doesn't take them. MIDI takes the notes too, for pads
func (pn *Panel) MIDIControllers(kb *keyboard.Keyboard) *Panel {
	pn.midiFrom(func(f func(midi.Event, psalm.CtrlTick)) { kb.OnEvent(f) }, false)
	return pn
}

// ControlChange takes a control change (channel 1-16, controller and value 0-127) from any MIDI
// source: a control learning takes the controller, the controls bound to it follow. Call it from
// one goroutine (the audio thread, as MIDI does)
func (pn *Panel) ControlChange(channel, controller, value int) {
	pn.receive(Binding{Channel: channel, Controller: controller}, value)
}

// NoteOn takes a note on (channel 1-16, note and velocity 0-127, 0 is a note off) from any MIDI
// source: a button or toggle learning takes the note, the buttons bound to it are pressed and the
// toggles switched over. Call it from the goroutine that calls ControlChange
func (pn *Panel) NoteOn(channel, note, velocity int) {
	if velocity > 0 {
		pn.receive(Binding{Channel: channel, Controller: note, Note: true}, velocity)
	}
}

// receive moves the controls bound to a controller or note
func (pn *Panel) receive(in Binding, value int) {
	pos := math.Max(0, math.Min(float64(value)/127, 1))
	if learner := pn.learning.Load(); learner != nil && (!in.Note || learner.takesNotes()) && pn.learning.CompareAndSwap(learner, nil) {
		code := pack(in)
		pn.release(in, learner, pn.params) // a controller moves one control
		learner.cc.Store(code)
		learner.caught.Store(true)
		learner.lastPos = pos
		pn.bindings.Add(1)
	}
	for _, par := range pn.params {
		b, ok := par.Binding()
		if !ok || b.Note != in.Note || b.Controller != in.Controller || (b.Channel != 0 && b.Channel != in.Channel) {
			continue
		}
		if in.Note {
			par.fromNote()
		} else {
			par.fromMIDI(pos)
		}
	}
	for _, mt := range pn.mountList() { // the panels mounted get it too
		if child := mt.Mounted(); child != nil && child != pn {
			child.receive(in, value)
		}
	}
}

// takesNotes tells whether a control can be bound to a note: buttons and toggles
func (par *Param) takesNotes() bool { return par.Kind == ButtonKind || par.Kind == ToggleKind }

// fromNote presses a button or switches a toggle over
func (par *Param) fromNote() {
	switch par.Kind {
	case ButtonKind:
		par.Press()
	case ToggleKind:
		par.set(1 - math.Round(par.Value()))
	}
}

// fromMIDI moves a control from a controller at a position (0-1)
func (par *Param) fromMIDI(pos float64) {
	last := par.lastPos
	par.lastPos = pos
	switch par.Kind {
	case ButtonKind:
		if pos >= 0.5 && last < 0.5 {
			par.Press()
		}
	case ToggleKind:
		par.set(math.Round(pos))
	case SelectKind:
		par.set(math.Round(pos * par.Max))
	case FaderKind, KnobKind:
		if par.pickup && !par.caught.Load() {
			// soft takeover: follow once the controller reaches the control (moves past it, or close)
			at := par.Position(par.Value())
			if math.Abs(pos-at) > 0.02 && (last-at)*(pos-at) > 0 {
				return
			}
			par.caught.Store(true)
		}
		par.set(par.At(pos))
	}
}

// Pickup makes a fader or knob follow its controller only once the controller has reached the
// control's value (soft takeover): no jump when the control was moved elsewhere (on the page, by a
// snapshot) and the knob on the controller is somewhere else
func Pickup() Option { return func(par *Param) { par.pickup = true } }

// BindCC binds a controller to the control: channel 1-16, or 0 for any channel. A controller moves
// one control: it is taken from a control that had it
func (par *Param) BindCC(channel, controller int) *Param {
	if par.Kind == MeterKind {
		panic(fmt.Sprintf("panel: a meter (%q) can't be moved by a controller", par.Name))
	}
	return par.bind(Binding{Channel: channel, Controller: controller})
}

// BindNote binds a note to a button or toggle (or the tile of a cue list): channel 1-16, or 0 for
// any channel. A note on presses the button, switches the toggle over
func (par *Param) BindNote(channel, note int) *Param {
	if !par.takesNotes() {
		panic(fmt.Sprintf("panel: only buttons and toggles can be bound to a note, not %q", par.Name))
	}
	return par.bind(Binding{Channel: channel, Controller: note, Note: true})
}

// bind binds a controller or note, taking it from a control that had it
func (par *Param) bind(b Binding) *Param {
	if par.panel != nil {
		par.panel.release(b, par, par.panel.Params())
		par.panel.bindings.Add(1)
	}
	par.cc.Store(pack(b))
	return par
}

// release takes a controller or note from the controls (but one) bound to it: on the same channel,
// or on any channel when one of the two is
func (pn *Panel) release(b Binding, keep *Param, params []*Param) {
	for _, par := range params {
		code := par.cc.Load()
		if par == keep || code == 0 {
			continue
		}
		if o := unpack(code); o.Note == b.Note && o.Controller == b.Controller && (o.Channel == b.Channel || o.Channel == 0 || b.Channel == 0) {
			par.cc.CompareAndSwap(code, 0)
		}
	}
}

// Unbind takes the controller from the control
func (par *Param) Unbind() {
	par.cc.Store(0)
	if par.panel != nil {
		par.panel.bindings.Add(1)
	}
}

// CC returns the controller bound to the control, and false when it has none (or a note)
func (par *Param) CC() (channel, controller int, ok bool) {
	b, ok := par.Binding()
	if !ok || b.Note {
		return 0, 0, false
	}
	return b.Channel, b.Controller, true
}

// Binding returns the controller or note bound to the control, and false when it has none
func (par *Param) Binding() (Binding, bool) {
	code := par.cc.Load()
	if code == 0 {
		return Binding{}, false
	}
	return unpack(code), true
}

// Learn arms the control: the next control change binds its controller to it (or the next note,
// for a button or toggle). Learning another control, or StopLearning, disarms it
func (par *Param) Learn() {
	if par.Kind != MeterKind && par.panel != nil {
		par.panel.learning.Store(par)
		par.panel.bindings.Add(1)
	}
}

// Learning returns the control waiting for a controller, nil when none is
func (pn *Panel) Learning() *Param { return pn.learning.Load() }

// StopLearning disarms the control waiting for a controller
func (pn *Panel) StopLearning() {
	pn.learning.Store(nil)
	pn.bindings.Add(1)
}

// Binding is a controller (or a note) bound to a control
type Binding struct {
	Channel    int  `json:"channel"`        // 1-16, 0 is any
	Controller int  `json:"controller"`     // the controller, or the note
	Note       bool `json:"note,omitempty"` // a note instead of a controller
}

// Bindings returns the controller or note of every control that has one, by name
// (and what the mounts keep, as "mount/...": the controls of every song of a playlist)
func (pn *Panel) Bindings() map[string]Binding {
	bs := map[string]Binding{}
	for _, par := range pn.Params() {
		if b, ok := par.Binding(); ok {
			bs[par.Name] = b
		}
	}
	for _, mt := range pn.mountList() {
		if k, ok := mt.m.(MountBinder); ok {
			for key, b := range k.BindingsMount() {
				bs[mt.Name+"/"+key] = b
			}
		}
	}
	return bs
}

// SetBindings binds the controllers of a map from Bindings; the controls not in it are unbound
// (the mounts that keep bindings get theirs)
func (pn *Panel) SetBindings(bs map[string]Binding) {
	for _, par := range pn.Params() {
		if b, ok := bs[par.Name]; ok && par.Kind != MeterKind && (!b.Note || par.takesNotes()) {
			par.cc.Store(pack(b))
		} else {
			par.cc.Store(0)
		}
	}
	for _, mt := range pn.mountList() {
		if k, ok := mt.m.(MountBinder); ok {
			mine := map[string]Binding{}
			for key, b := range bs {
				if rest, ok := strings.CutPrefix(key, mt.Name+"/"); ok {
					mine[rest] = b
				}
			}
			k.SetBindingsMount(mine)
		}
	}
	pn.bindings.Add(1)
}

// SaveBindings writes the bindings to a file as JSON, LoadBindings reads them back: the mapping of
// a controller, kept apart from the values
func (pn *Panel) SaveBindings(path string) error {
	data, err := json.MarshalIndent(pn.Bindings(), "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// LoadBindings reads bindings written by SaveBindings
func (pn *Panel) LoadBindings(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var bs map[string]Binding
	if err := json.Unmarshal(data, &bs); err != nil {
		return err
	}
	pn.SetBindings(bs)
	return nil
}

// KeepBindings keeps the bindings in a file (those of the songs of a playlist too, see MountBinder):
// it loads them now (when the file is there) and saves them every time they change from then on, so
// what you learn is there the next time. Saving goes on in the background as long as the program
// runs; an error saving is left out
func (pn *Panel) KeepBindings(path string) error {
	if err := pn.LoadBindings(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	saved := fmt.Sprint(pn.Bindings()) // before: a change right after counts
	go func() {
		for range time.Tick(keepInterval) {
			// compared every time: the mounted panels (the songs) learn too
			if now := fmt.Sprint(pn.Bindings()); now != saved && pn.SaveBindings(path) == nil {
				saved = now
			}
		}
	}()
	return nil
}

// keepInterval is how often KeepBindings looks for changes
const keepInterval = 500 * time.Millisecond

// midiState is what a panel keeps for MIDI
type midiState struct {
	learning atomic.Pointer[Param]
	bindings atomic.Int64 // counts changes of the bindings and learning, for the page
}
