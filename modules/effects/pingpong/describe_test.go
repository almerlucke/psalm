package pingpong

import (
	"testing"

	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/params/paramstest"
)

func TestDescribe(t *testing.T) {
	m := New(DefaultConfig())
	paramstest.Check(t, func() params.Set { return Describe(m.Settings()) }, m.ReceiveMessage)
}
