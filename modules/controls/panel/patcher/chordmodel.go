package patcher

import (
	"fmt"
	"strings"
	"sync"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/chordmodel"
)

// patchers are the patches by the environment of their modules, for a module to find the patch it is in (to write into another of its modules)
var patchers sync.Map // *psalm.Env -> *Patcher

// ChordModelKind is the chord model: a matrix of how likely each chord is to follow another, a progression drawn from it, and a button that writes it into the harmony of
// the patch; it plays the progression through the synths its notes output is cabled to
func ChordModelKind() Kind {
	return Kind{Name: "chordmodel", Group: "Song", Outs: []Port{{Name: "notes", Type: Notes}},
		Make: func(string) (*Made, error) {
			m := chordmodel.New()
			m.SetLink(chordmodel.Link{
				Sections: func() []chordmodel.Section {
					tr := SongOf(m.Env()).Transport()
					if tr == nil {
						return nil
					}
					var out []chordmodel.Section
					for _, s := range tr.Form().Sections {
						out = append(out, chordmodel.Section{Name: s.Name, Bars: s.Bars})
					}
					return out
				},
				Bake: func(section, chords string) error { return bakeChords(m.Env(), section, chords) },
			})
			return &Made{Module: m,
				State:    func() (string, error) { return m.StateText(), nil },
				SetState: m.SetStateText,
				Notes:    func(targets []psalm.Receiver) { m.SetTargets(targets...) },
				Edit: func(page *panel.Panel, name, _ string) Settings {
					page.Section("Chords")
					chordmodel.Add(page, name, m)
					return noSettings{}
				}}, nil
		}}
}

// noSettings is the settings of a module that keeps its sound as a state
type noSettings struct{}

func (noSettings) Values() map[string]float64 { return map[string]float64{} }
func (noSettings) Apply(map[string]float64)   {}

// bakeChords writes chords into the harmony of the patch: the line of a section (the others keep theirs), or what is not a section's line
func bakeChords(env *psalm.Env, section, chords string) error {
	v, ok := patchers.Load(env)
	if !ok {
		return fmt.Errorf("this module is not in a patch")
	}
	pt := v.(*Patcher)
	pt.mu.Lock()
	id, text := "", ""
	for _, n := range pt.nodes {
		if n.Kind == "harmony" {
			id, text = n.ID, n.texts["chords"]
			break
		}
	}
	pt.mu.Unlock()
	if id == "" {
		return fmt.Errorf("the patch has no harmony module to write the chords to: add one")
	}
	if err := pt.SetText(id, "chords", mergeChords(text, section, chords)); err != nil {
		return err
	}
	if e := pt.TextError(id, "chords"); e != "" {
		return fmt.Errorf("the harmony would not take the chords: %s", e)
	}
	return nil
}

// mergeChords puts chords in the text of a harmony: in the line of a section (added when it has none), or, for no section, in place of the lines that are no section's
func mergeChords(text, section, chords string) string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if section == "" {
		out := []string{chords}
		for _, l := range lines {
			if sectionLabel.MatchString(l) {
				out = append(out, l)
			}
		}
		return strings.Join(out, "\n")
	}
	done := false
	for i, l := range lines {
		if m := sectionLabel.FindStringSubmatch(l); m != nil && m[1] == section {
			lines[i], done = section+": "+chords, true
			break
		}
	}
	if !done {
		if strings.TrimSpace(strings.Join(lines, "")) == "" {
			lines = nil
		}
		lines = append(lines, section+": "+chords)
	}
	return strings.Join(lines, "\n")
}
