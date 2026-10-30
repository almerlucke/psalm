package patcher

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"

	"github.com/almerlucke/psalm"
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

// AddVoice adds what plays one thing in the sections of the song and gives it a name: a rhythm (a voice: a pattern, pitches and the notes that play them), a melody (a part, notes
// written per section) or a melodist (a melody made while it plays). It plays on the instrument when it is given (an id of the patch), and the ids of the modules it made come back
func (pt *Patcher) AddVoice(kind, name, instrument string) ([]string, error) {
	name = strings.TrimSpace(name)
	y := pt.belowAll()
	var ids []string
	var out string // the module whose notes go to the instrument
	switch kind {
	case "rhythm":
		voice, err := pt.Add("voice", "", 300, y)
		if err != nil {
			return nil, err
		}
		_ = pt.SetText(voice, "pattern", "x . . . x . . .")
		if name != "" {
			_ = pt.SetName(voice, name)
		}
		ids, out = []string{voice}, voice
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

// chordModelOf is the chord model of a module: the one inside a song, or a chord model module of its own
func chordModelOf(m psalm.Modular) *chordmodel.Module {
	switch v := m.(type) {
	case *SongForm:
		return v.Model()
	case *chordmodel.Module:
		return v
	}
	return nil
}

// chordModel is the chord model of the patch: the one inside its song, else a module of its own. With make, a patch without a song gets one, and it has the model
func (pt *Patcher) chordModel(make bool) (*chordmodel.Module, error) {
	find := func() *chordmodel.Module {
		pt.mu.Lock()
		defer pt.mu.Unlock()
		var alone *chordmodel.Module
		for _, n := range pt.nodes {
			if m := chordModelOf(madeModule(n)); m != nil {
				if n.Kind == "song" {
					return m
				}
				alone = m
			}
		}
		return alone
	}
	if m := find(); m != nil || !make {
		return m, nil
	}
	if _, err := pt.Add("song", "", 60, pt.belowAll()); err != nil {
		return nil, err
	}
	if m := find(); m != nil {
		return m, nil
	}
	return nil, errors.New("no chord model")
}

// Chords is what the grid asks of the chord model: "draw" new chords for a section ("" the lines without a section) and write them to the harmony, "live" to have them drawn again
// every time the song leaves a section (on) or not. The model and the harmony are added when the patch has none; the key and scale of the harmony are the model's
func (pt *Patcher) Chords(action, section string, on bool) error {
	m, err := pt.chordModel(true)
	if err != nil {
		return err
	}
	// the key and scale of the harmony, when the model has them too
	pt.mu.Lock()
	hid := ""
	for _, kind := range []string{"harmony", "song"} { // the one the chords are written to
		for _, n := range pt.nodes {
			if _, has := n.texts["chords"]; has && n.Kind == kind && hid == "" {
				hid = n.ID
			}
		}
	}
	pt.mu.Unlock()
	if s := pt.SettingsOf(hid); s != nil {
		v := s.Values()
		if sc, ok := v["scale"]; ok && int(sc) >= 0 && int(sc) < len(Scales) {
			if i := slices.Index(chordmodel.Scales, Scales[int(sc)]); i >= 0 {
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
