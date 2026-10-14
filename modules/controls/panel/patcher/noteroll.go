package patcher

import (
	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/noteroll"
)

// NoteRollKind is the note editor: a piano roll in its side panel, a loop of notes (a length in
// beats) that plays with the tempo of the patch, its notes out to the synths its cables go to
func NoteRollKind() Kind {
	return Kind{Name: "noteroll", Group: "Generators", Outs: []Port{{Name: "notes", Type: Notes}},
		Make: func(string) (*Made, error) {
			roll := noteroll.NewRoll()
			return &Made{Module: roll,
				Notes: func(targets []psalm.Receiver) { roll.SetTargets(targets...) },
				Edit: func(page *panel.Panel, name, _ string) Settings {
					page.Section("Notes")
					noteroll.Add(page, name, roll)
					return roll
				}}, nil
		}}
}
