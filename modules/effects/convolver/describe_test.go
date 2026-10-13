package convolver

import (
	"testing"

	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/params/paramstest"
)

func TestDescribe(t *testing.T) {
	m := New([][]float64{{1, 0.5, 0.25, 0.125}}, DefaultConfig())
	paramstest.Check(t, func() params.Set { return Describe(m.Settings()) }, m.ReceiveMessage)
}
