package templ

import (
	"testing"

	"github.com/almerlucke/genny/sequence"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
)

type pitches struct {
	*psalm.Module
	got []any
}

func (p *pitches) Receive(v any, _ int) {
	if m, ok := psalm.MsgVal(v); ok {
		p.got = append(p.got, m["pitch"])
	}
}

func TestRestart(t *testing.T) {
	psalm.New(nil)
	tp := New(template.Template{"pitch": sequence.NewLoop(60.0, 62.0, 64.0)})
	rec := &pitches{Module: psalm.NewControl()}
	rec.SetSelf(rec)
	rec.ValueIn(tp)
	for range 2 {
		tp.Receive(psalm.Bang, 0)
	}
	tp.Restart()
	tp.Receive(psalm.Bang, 0)
	if len(rec.got) != 3 || rec.got[1] != 62.0 || rec.got[2] != 60.0 {
		t.Fatalf("after a restart the sequence should start again: %v", rec.got)
	}
}
