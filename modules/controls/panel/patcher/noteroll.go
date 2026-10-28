package patcher

import (
	"math"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/noteroll"
	"github.com/almerlucke/psalm/tools/harmony"
)

// NoteRollKind is the note editor: a piano roll in its side panel, a loop of notes (a length in
// beats) that plays with the tempo of the patch, its notes out to the synths its cables go to
func NoteRollKind() Kind {
	return Kind{Name: "noteroll", Group: "Generators", Outs: []Port{{Name: "notes", Type: Notes}},
		Make: func(string) (*Made, error) {
			roll := noteroll.NewRoll()
			roll.Link(rollLink(roll))
			return &Made{Module: roll,
				State:    func() (string, error) { return roll.StateText(), nil },
				SetState: roll.SetStateText,
				Notes:    func(targets []psalm.Receiver) { roll.SetTargets(targets...) },
				Edit: func(page *panel.Panel, name, _ string) Settings {
					page.Section("Notes")
					noteroll.Add(page, name, roll)
					return roll
				}}, nil
		}}
}

// rollLink is what a roll in a patch knows of it: where the song is, its sections, and the chords of the harmony over a clip
func rollLink(roll *noteroll.Roll) noteroll.Link {
	return noteroll.Link{
		Where: func(tick psalm.CtrlTick) (string, int64, bool) {
			song := SongOf(roll.Env())
			ppq := roll.Config().PPQ
			pos, ok := song.At(tick, ppq)
			if !ok || pos.Ended {
				return "", 0, false
			}
			bpb := song.Transport().Form().BeatsPerBar
			in := (pos.Beat - float64(pos.Section.StartBar)*bpb) * float64(ppq)
			return pos.Section.Name, int64(math.Round(float64(tick.TickTimestamp) - in)), true
		},
		Sections: func() []noteroll.Section {
			tr := SongOf(roll.Env()).Transport()
			if tr == nil {
				return nil
			}
			form := tr.Form()
			out := make([]noteroll.Section, 0, len(form.Sections))
			for _, s := range form.Sections {
				out = append(out, noteroll.Section{Name: s.Name, Beats: float64(s.Bars) * form.BeatsPerBar})
			}
			return out
		},
		Scale: func() (int, string, bool) {
			h := HarmonyOf(roll.Env())
			if h == nil {
				return 0, "", false
			}
			pc, err := harmony.PitchClass(Keys[h.Key])
			if err != nil {
				return 0, "", false
			}
			return pc, Scales[h.Scale], true
		},
		Chords: func(section string, beats float64) []noteroll.ChordSpan {
			h := HarmonyOf(roll.Env())
			if h == nil {
				return nil
			}
			bpb := 4.0
			if tr := SongOf(roll.Env()).Transport(); tr != nil {
				bpb = tr.Form().BeatsPerBar
			}
			return h.Spans(section, beats, bpb)
		},
	}
}
