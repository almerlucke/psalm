package organ

import (
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/params/paramstest"
)

func TestDescribe(t *testing.T) {
	for name, f := range Presets {
		o := New(f())
		t.Run(name, func(t *testing.T) {
			paramstest.Check(t, func() params.Set { return Describe(o.cfg) }, func(m psalm.Message) { o.ReceiveMessage(m) })
		})
	}
}
