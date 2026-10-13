package follower

import (
	"testing"

	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/params/paramstest"
)

func TestDescribe(t *testing.T) {
	f := New(DefaultConfig())
	paramstest.Check(t, func() params.Set { return Describe(f.Settings()) }, f.ReceiveMessage)
	w := NewWah(DefaultWah())
	paramstest.Check(t, func() params.Set { return DescribeWah(w.Settings()) }, w.ReceiveMessage)
}
