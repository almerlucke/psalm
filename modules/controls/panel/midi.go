package panel

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
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

// cc is a binding: the channel (1-16, 0 any) and the controller, packed for an atomic, 0 is none
func packCC(channel, controller int) int32 { return int32(channel<<8|controller) + 1 }

func unpackCC(code int32) (channel, controller int) {
	code--
	return int(code >> 8), int(code & 0xff)
}

// MIDI connects a keyboard (from the keyboard package): its control changes move the controls bound
// to them, on the audio thread
func (pn *Panel) MIDI(kb *keyboard.Keyboard) *Panel {
	kb.OnEvent(func(e midi.Event, _ psalm.CtrlTick) {
		if e.Message.Kind() == midi.ControlChange {
			pn.ControlChange(e.Message.Channel(), e.Message.Controller(), e.Message.Value())
		}
	})
	return pn
}

// ControlChange takes a control change (channel 1-16, controller and value 0-127) from any MIDI
// source: a control learning takes the controller, the controls bound to it follow. Call it from
// one goroutine (the audio thread, as MIDI does)
func (pn *Panel) ControlChange(channel, controller, value int) {
	pos := math.Max(0, math.Min(float64(value)/127, 1))
	if learner := pn.learning.Swap(nil); learner != nil {
		code := packCC(channel, controller)
		for _, par := range pn.params {
			if par != learner {
				par.cc.CompareAndSwap(code, 0) // a controller moves one control
			}
		}
		learner.cc.Store(code)
		learner.caught.Store(true)
		learner.lastPos = pos
		pn.bindings.Add(1)
	}
	for _, par := range pn.params {
		code := par.cc.Load()
		if code == 0 {
			continue
		}
		ch, ctl := unpackCC(code)
		if ctl != controller || (ch != 0 && ch != channel) {
			continue
		}
		par.fromMIDI(pos)
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
	if par.panel != nil {
		code := packCC(channel, controller)
		for _, other := range par.panel.Params() {
			if other != par {
				other.cc.CompareAndSwap(code, 0)
			}
		}
		par.panel.bindings.Add(1)
	}
	par.cc.Store(packCC(channel, controller))
	return par
}

// Unbind takes the controller from the control
func (par *Param) Unbind() {
	par.cc.Store(0)
	if par.panel != nil {
		par.panel.bindings.Add(1)
	}
}

// CC returns the controller bound to the control, and false when it has none
func (par *Param) CC() (channel, controller int, ok bool) {
	code := par.cc.Load()
	if code == 0 {
		return 0, 0, false
	}
	channel, controller = unpackCC(code)
	return channel, controller, true
}

// Learn arms the control: the next control change binds its controller to it. Learning another
// control, or StopLearning, disarms it
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

// Binding is a controller bound to a control
type Binding struct {
	Channel    int `json:"channel"` // 1-16, 0 is any
	Controller int `json:"controller"`
}

// Bindings returns the controller of every control that has one, by name
func (pn *Panel) Bindings() map[string]Binding {
	bs := map[string]Binding{}
	for _, par := range pn.Params() {
		if ch, ctl, ok := par.CC(); ok {
			bs[par.Name] = Binding{ch, ctl}
		}
	}
	return bs
}

// SetBindings binds the controllers of a map from Bindings; the controls not in it are unbound
func (pn *Panel) SetBindings(bs map[string]Binding) {
	for _, par := range pn.Params() {
		if b, ok := bs[par.Name]; ok && par.Kind != MeterKind {
			par.cc.Store(packCC(b.Channel, b.Controller))
		} else {
			par.cc.Store(0)
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

// KeepBindings keeps the bindings in a file: it loads them now (when the file is there) and saves
// them every time they change from then on, so what you learn is there the next time. Saving goes
// on in the background as long as the program runs; an error saving is left out
func (pn *Panel) KeepBindings(path string) error {
	if err := pn.LoadBindings(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	seen, saved := pn.bindings.Load(), fmt.Sprint(pn.Bindings()) // before: a change right after counts
	go func() {
		for range time.Tick(keepInterval) {
			if n := pn.bindings.Load(); n != seen {
				seen = n
				if now := fmt.Sprint(pn.Bindings()); now != saved && pn.SaveBindings(path) == nil {
					saved = now
				}
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
