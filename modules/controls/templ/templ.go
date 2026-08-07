package templ

import (
	"fmt"

	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
)

type Templ struct {
	*psalm.Module
	templ  template.Template
	params map[string]any
}

func New(templ template.Template) *Templ {
	g := &Templ{
		Module: psalm.NewControl(),
		templ:  templ,
	}

	g.SetSelf(g)

	return g
}

func (t *Templ) Receive(value any, index int) {
	var (
		msg   psalm.Message
		isMsg bool
	)

	if msg, isMsg = psalm.MsgVal(value); isMsg {
		for k, v := range msg {
			if k != "bang" {
				t.params[k] = v
			}
		}
	}

	if psalm.IsBang(value) {
		switch index {
		case 0:
			if t.templ.Done() {
				break
			}

			t.resolveParams()

			msgs := t.templ.Generate()

			if t.templ.Done() {
				t.Send(psalm.Bang, 1)
			}

			for _, msg = range msgs {
				t.Send(msg, 0)
			}
		case 1:
			t.templ.Reset()
		default:
			break
		}
	} else if !isMsg {
		t.params[fmt.Sprintf("controlInput(%d)", index)] = value
	}
}

func (t *Templ) resolveParams() {
	var params []*template.Parameter
	for k, v := range t.params {
		params = append(params, template.NewParameter(k, v))
	}

	t.templ.SetParameters(params)
}
