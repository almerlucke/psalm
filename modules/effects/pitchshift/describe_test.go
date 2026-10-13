package pitchshift

import (
	"testing"

	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/params/paramstest"
)

func TestDescribe(t *testing.T) {
	p := New(7, 50, 0.5)
	paramstest.Check(t, func() params.Set { return Describe(p) }, p.ReceiveMessage)
}
