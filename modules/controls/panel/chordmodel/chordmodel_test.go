package chordmodel

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	cm "github.com/almerlucke/psalm/tools/chordmodel"
)

// a module draws progressions of the bars asked, keeps the chords locked when it draws the others again, learns, takes a preset, and is kept as a text
func TestModule(t *testing.T) {
	m := New()
	st := m.State()
	if len(st.Strip) == 0 || st.Model.States[0] != "I" || st.Preset != "pop" {
		t.Fatalf("a new module: %+v", st)
	}
	m.change(func(s *State) { s.Bars = 6 })
	m.Generate(5)
	bars := 0.0
	for _, s := range m.State().Strip {
		bars += s.Bars
	}
	if bars != 6 || m.State().Seed != 6 {
		t.Errorf("%v bars, seed %v", bars, m.State().Seed)
	}
	// the same seed, the same progression
	a := m.State().Strip
	m.Generate(5)
	if !slices.Equal(a, m.State().Strip) {
		t.Errorf("seed 5 twice: %v and %v", a, m.State().Strip)
	}
	// a locked chord stays
	strip := m.State().Strip
	strip[1].Lock, strip[1].State = true, 3
	m.SetStrip(strip)
	for seed := range uint64(40) {
		m.Reroll(seed)
		if s := m.State().Strip[1]; s.State != 3 || !s.Lock {
			t.Fatalf("the locked chord moved: %v", s)
		}
	}
	// text for the harmony, in the key
	m.SetStrip([]cm.Slot{{State: 0, Bars: 1}, {State: 4, Bars: 2}})
	m.change(func(s *State) { s.Key = 2 })
	if st := m.State(); st.Text() != "D A*2" {
		t.Errorf("text %q", st.Text())
	}
	m.change(func(s *State) { s.As = "numerals" })
	if st := m.State(); st.Text() != "I V*2" {
		t.Errorf("numerals %q", st.Text())
	}
	// a preset, and what was taught
	if !m.Preset("minor") || m.State().Model.States[0] != "i" || m.Preset("polka") {
		t.Error("presets")
	}
	m.Learn("i bVII VI v\ni bVII VI v")
	if st := m.State(); st.Preset != "" || !slices.Contains(st.Model.States, "bVII") || *m.err.Load() != "" {
		t.Errorf("after teaching: %v, error %q", st.Model.States, *m.err.Load())
	}
	m.Learn("i xyz")
	if *m.err.Load() == "" {
		t.Error("a bad numeral is not reported")
	}

	// kept as a text
	other := New()
	if err := other.SetStateText(m.StateText()); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(other.State().Model.States, m.State().Model.States) || !slices.Equal(other.State().Strip, m.State().Strip) || other.State().Key != 2 {
		t.Errorf("state kept: %+v", other.State())
	}
	if err := other.SetStateText(`{"model": {"states": ["I"]}}`); err == nil {
		t.Error("a state that makes no sense is taken")
	}
}

// the page's messages: a changed model, a new progression, a preset, teaching; the writing to the harmony is the link's
func TestMessages(t *testing.T) {
	m := New()
	send := func(v any) {
		b, _ := json.Marshal(v)
		m.message(b)
	}
	model := m.State().Model
	model.W[0][1] = 99
	send(map[string]any{"set": map[string]any{"model": model, "bars": 4, "key": 7, "scale": 1, "as": "numerals", "target": "Verse"}})
	st := m.State()
	if st.Model.W[0][1] != 99 || st.Bars != 4 || st.Key != 7 || st.Scale != 1 || st.As != "numerals" || st.Target != "Verse" || st.Preset != "" {
		t.Errorf("the set: %+v", st)
	}
	send(map[string]any{"set": map[string]any{"model": map[string]any{"states": []string{"I"}}}}) // nonsense: left as it is
	if m.State().Model.W[0][1] != 99 {
		t.Error("a model that makes no sense was taken")
	}
	send(map[string]any{"generate": 3})
	if len(m.State().Strip) == 0 || m.State().Seed != 4 {
		t.Errorf("generate: %+v", m.State())
	}
	send(map[string]any{"preset": "jazz"})
	if m.State().Model.States[0] != "Imaj7" {
		t.Errorf("preset: %v", m.State().Model.States)
	}
	send(map[string]any{"learn": "Imaj7 vi7 ii7 V7"})
	if len(m.State().Strip) == 0 || *m.err.Load() != "" {
		t.Errorf("learn: %q", *m.err.Load())
	}

	// the writing: through the link, and its error is the module's
	var wrote [2]string
	m.SetLink(Link{Bake: func(section, chords string) error { wrote = [2]string{section, chords}; return nil }})
	send(map[string]any{"bake": true})
	if wrote[0] != "Verse" || !strings.Contains(wrote[1], "I") || *m.err.Load() != "" {
		t.Errorf("baked %v, error %q", wrote, *m.err.Load())
	}
	m.SetLink(Link{Bake: func(string, string) error { return errors.New("no harmony") }})
	send(map[string]any{"bake": true})
	if *m.err.Load() != "no harmony" {
		t.Errorf("error %q", *m.err.Load())
	}
}
