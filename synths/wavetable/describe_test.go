package wavetable

import (
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/params/paramstest"
)

func TestDescribe(t *testing.T) {
	// the Basic table: gen imports this package, so an internal test can't use gen.Brightness
	table := Basic()
	for name, f := range Presets {
		v := New(f(table))
		t.Run(name, func(t *testing.T) {
			paramstest.Check(t, func() params.Set { return Describe(v.cfg) }, func(m psalm.Message) { v.ReceiveMessage(m) })
		})
	}
}
