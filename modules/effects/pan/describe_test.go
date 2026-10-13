package pan

import (
	"testing"

	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/params/paramstest"
)

func TestDescribe(t *testing.T) {
	m := New(0)
	paramstest.Check(t, func() params.Set { return Describe(m) }, m.ReceiveMessage)
}
