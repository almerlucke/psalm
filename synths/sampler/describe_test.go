package sampler

import (
	"testing"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/params/paramstest"
)

func TestDescribe(t *testing.T) {
	v := New(DefaultConfig(MustInstrument(NewRegion(NewSound(sr, sine(220, 1))).Keys(0, 127, 57))))
	paramstest.Check(t, func() params.Set { return Describe(v.cfg) }, func(m psalm.Message) { v.ReceiveMessage(m) })
}
