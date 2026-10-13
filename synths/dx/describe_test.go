package dx

import (
	"testing"

	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/params/paramstest"
)

func TestDescribe(t *testing.T) {
	for name, f := range Presets {
		v := New(f(), 2)
		t.Run(name, func(t *testing.T) {
			paramstest.Check(t, func() params.Set { return Describe(v.patch) }, v.ReceiveMessage)
		})
	}
}
