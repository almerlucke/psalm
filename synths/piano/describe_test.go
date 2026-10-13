package piano

import (
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/params/paramstest"
)

func TestDescribe(t *testing.T) {
	for name, f := range Presets {
		p := NewPoly(f(), 2)
		t.Run(name, func(t *testing.T) {
			paramstest.Check(t, func() params.Set { return Describe(p.cfg) }, func(m psalm.Message) { p.ReceiveMessage(m) })
		})
	}
}
