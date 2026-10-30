package patcher

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"

	"github.com/almerlucke/psalm/modules/controls/panel/chordmodel"
)

// VoiceKinds are the voices the grid of the sections can add: what each one is
var VoiceKinds = []string{"rhythm", "melody", "melodist"}

// belowAll is a place under the modules of the patch, for what is added from the grid
func (pt *Patcher) belowAll() float64 {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	y := 0.0
	for _, n := range pt.nodes {
		if n.ID != "out" && n.Y > y {
			y = n.Y
		}
	}
	return y + 110
}

// instrumentIn is the notes input of an instrument (a module that takes notes and makes sound), -1: it has none
func (pt *Patcher) instrumentIn(id string) int {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	n := pt.node(id)
	if n == nil {
		return -1
	}
	return slices.IndexFunc(n.ins(), func(p Port) bool { return p.Type == Notes })
}

// AddVoice adds what plays one thing in the sections of the song and gives it a name: a rhythm (a clock with a pattern and a notes module that plays it), a melody (a part, notes
// written per section) or a melodist (a melody made while it plays). It plays on the instrument when it is given (an id of the patch), and the ids of the modules it made come back
func (pt *Patcher) AddVoice(kind, name, instrument string) ([]string, error) {
	name = strings.TrimSpace(name)
	y := pt.belowAll()
	var ids []string
	var out string // the module whose notes go to the instrument
	switch kind {
	case "rhythm":
		clock, err := pt.Add("clock", "", 300, y)
		if err != nil {
			return nil, err
		}
		notes, err := pt.Add("notes", "", 780, y)
		if err != nil {
			return nil, err
		}
		if _, err := pt.Connect(Cable{From: clock, To: notes}); err != nil {
			return nil, err
		}
		_ = pt.SetText(clock, "pattern", "x . . . x . . .")
		if name != "" {
			_ = pt.SetName(clock, name)
			_ = pt.SetName(notes, name+" notes")
		}
		ids, out = []string{clock, notes}, notes
	case "melody":
		part, err := pt.Add("part", "", 300, y)
		if err != nil {
			return nil, err
		}
		if name != "" {
			_ = pt.SetName(part, name)
		}
		ids, out = []string{part}, part
	case "melodist":
		m, err := pt.Add("melodist", "", 300, y)
		if err != nil {
			return nil, err
		}
		if name != "" {
			_ = pt.SetName(m, name)
		}
		ids, out = []string{m}, m
	default:
		return nil, fmt.Errorf("a voice is one of %v", VoiceKinds)
	}
	if instrument != "" {
		in := pt.instrumentIn(instrument)
		if in < 0 {
			return ids, errors.New("that is not an instrument")
		}
		if _, err := pt.Connect(Cable{From: out, To: instrument, In: in}); err != nil {
			return ids, err
		}
	}
	return ids, nil
}

// ChordsInfo is what the grid of the sections knows of the chord model of the patch
type ChordsInfo struct {
	Live bool `json:"live"` // the chords are drawn again every time the song leaves a section
}

// chordModel is the first chord model of the patch (made, with the harmony it writes to, when the patch has none)
func (pt *Patcher) chordModel(make bool) (*chordmodel.Module, string, error) {
	find := func(kind string) string {
		pt.mu.Lock()
		defer pt.mu.Unlock()
		for _, n := range pt.nodes {
			if n.Kind == kind {
				return n.ID
			}
		}
		return ""
	}
	if !make && find("chordmodel") == "" {
		return nil, "", nil
	}
	if find("harmony") == "" {
		if _, err := pt.Add("harmony", "", 60, pt.belowAll()); err != nil {
			return nil, "", err
		}
	}
	id := find("chordmodel")
	if id == "" {
		var err error
		if id, err = pt.Add("chordmodel", "", 60, pt.belowAll()); err != nil {
			return nil, "", err
		}
		_ = pt.SetName(id, "chord model")
	}
	m, ok := pt.Module(id).(*chordmodel.Module)
	if !ok {
		return nil, "", errors.New("no chord model")
	}
	return m, id, nil
}

// Chords is what the grid asks of the chord model: "draw" new chords for a section ("" the lines without a section) and write them to the harmony, "live" to have them drawn again
// every time the song leaves a section (on) or not. The model and the harmony are added when the patch has none; the key and scale of the harmony are the model's
func (pt *Patcher) Chords(action, section string, on bool) error {
	m, _, err := pt.chordModel(true)
	if err != nil {
		return err
	}
	// the key and scale of the harmony, when the model has them too
	pt.mu.Lock()
	hid := ""
	for _, n := range pt.nodes {
		if n.Kind == "harmony" {
			hid = n.ID
			break
		}
	}
	pt.mu.Unlock()
	if s := pt.SettingsOf(hid); s != nil {
		v := s.Values()
		if sc := int(v["scale"]); sc >= 0 && sc < len(Scales) {
			if i := slices.Index(chordmodel.Scales, Scales[sc]); i >= 0 {
				m.SetKey(int(v["key"]), i)
			}
		}
	}
	switch action {
	case "draw":
		was := m.State().Target
		m.SetTarget(section)
		m.Reroll(rand.Uint64N(1 << 32))
		m.Bake()
		m.SetTarget(was)
	case "live":
		m.SetTarget("")
		m.SetLive(on, 0)
	default:
		return errors.New("draw or live")
	}
	return nil
}
