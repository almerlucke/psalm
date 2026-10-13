package westcoast

import (
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/params/paramstest"
)

func TestDescribe(t *testing.T) {
	for name, f := range Presets {
		v := New(f())
		t.Run(name, func(t *testing.T) {
			paramstest.Check(t, func() params.Set { return Describe(v.cfg) }, func(m psalm.Message) { v.ReceiveMessage(m) })
		})
	}
}
