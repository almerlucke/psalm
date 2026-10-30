package patcher

import (
	"strings"
	"sync/atomic"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/params"
)

// Voice is one thing that plays in the sections of a song, in one module: a clock with a step pattern, a sequence of pitches that goes with it (or the one pitch of the notes) and the
// notes that play them. It is a clock, a sequence and a notes module with the cables between them taken out: the settings, the pattern, the lanes and the values are all its own,
// per section of the song as in the three, and its notes output goes to the instrument. Without values it plays its one pitch
type Voice struct {
	*psalm.Module
	clock  *Clock
	seq    *Sequence
	note   *Note
	useSeq atomic.Bool
}

// NewVoice returns a voice of sixteenths, a hit every step, on one pitch
func NewVoice() *Voice {
	v := &Voice{Module: psalm.NewControl(), clock: NewClock(), seq: NewSequence(), note: NewNote()}
	v.SetSelf(v)
	v.seq.ValueConnect(0, v.note, 1) // the value of the sequence is the pitch of the notes
	return v
}

// CtrlTick plays the hits that fall on the tick: the next value of the sequence, if it has any, then the note (it sounds at the end of the tick, when the values are up to date)
func (v *Voice) CtrlTick(tick psalm.CtrlTick) {
	v.clock.Play(tick, func(ev Event) {
		if v.useSeq.Load() {
			v.seq.Receive(ev, 0)
		}
		v.note.Receive(ev, 0)
	})
}

// ReceiveMessage sets the settings of all three (their keys differ)
func (v *Voice) ReceiveMessage(msg psalm.Message) {
	v.clock.ReceiveMessage(msg)
	v.seq.ReceiveMessage(msg)
	v.note.ReceiveMessage(msg)
}

func (v *Voice) Receive(x any, _ int) {
	if m, ok := psalm.MsgVal(x); ok {
		v.ReceiveMessage(m)
	}
}

// SetPattern, SetLanes and SetValues set what it plays (from any goroutine); no values: the pitch of the notes
func (v *Voice) SetPattern(text string) error { return v.clock.SetPattern(text) }
func (v *Voice) SetLanes(text string) error   { return v.clock.SetLanes(text) }
func (v *Voice) SetValues(text string) error {
	if strings.TrimSpace(text) == "" {
		v.useSeq.Store(false)
		return nil
	}
	if err := v.seq.SetValues(text); err != nil {
		return err
	}
	v.useSeq.Store(true)
	return nil
}

// Describe describes its settings: the clock's (division, swing, chance, groove, drift, mutate, seed, freeze), the order of the values and the notes' (pitch, velocity, length, transpose)
func (v *Voice) Describe() params.Set {
	set := params.Set{Name: "voice"}
	set.Params = append(set.Params, v.clock.Describe().Params...)
	set.Params = append(set.Params, v.seq.Describe().Params...)
	set.Params = append(set.Params, v.note.Describe().Params...)
	return set
}

// voiceKind is the kind of voices
func voiceKind() Kind {
	return Kind{Name: "voice", Group: "Song", Outs: []Port{{Name: "notes", Type: Notes}},
		Make: func(string) (*Made, error) {
			v := NewVoice()
			return &Made{Module: v, Describe: func(string) params.Set { return v.Describe() },
				Texts: []Text{{Key: "pattern", Label: "pattern", Value: "x", Help: PatternHelp}, {Key: "lanes", Label: "lanes", Value: "", Help: LanesHelp},
					{Key: "values", Label: "pitches (without: the pitch setting)", Value: "", Help: ValuesHelp}},
				SetText: func(k, text string) error {
					switch k {
					case "lanes":
						return v.SetLanes(text)
					case "values":
						return v.SetValues(text)
					}
					return v.SetPattern(text)
				},
				Notes: v.note.SetTargets, Late: v.note.late}, nil
		}}
}
