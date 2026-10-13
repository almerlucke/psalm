package chorus

import (
	"testing"

	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/params/paramstest"
)

func TestDescribe(t *testing.T) {
	c := New(DefaultConfig())
	paramstest.Check(t, func() params.Set { return Describe(c.Settings()) }, c.ReceiveMessage)
}
