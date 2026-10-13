package freeverb

import (
	"testing"

	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/params/paramstest"
)

func TestDescribe(t *testing.T) {
	m := New(0.7, 0.6, 0.9, 0.1, 0.7)
	paramstest.Check(t, func() params.Set { return Describe(m) }, m.ReceiveMessage)
}
