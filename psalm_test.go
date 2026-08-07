package psalm

import (
	"testing"
)

type TestModule struct {
	*Module
	value float32
}

func NewTestModule(value float32) *TestModule {
	tm := &TestModule{Module: NewModule([]int{1}, []int{1}), value: value}
	tm.SetSelf(tm)
	return tm
}

func (tm *TestModule) Synthesize() {
	for i := range tm.config.BufferSize {
		tm.outputs[0].Channels[0][i] = tm.inputs[0].Channels[0][i] + tm.value
	}
}

func TestConnections(t *testing.T) {
	Debugger.SetDebug(true)

	p := NewPatch([]int{1}, []int{1})
	p.SetID("p")
	tm1 := NewTestModule(0.3)
	tm1.SetID("tm1")
	tm1.In(p.PatchInput(0))
	p.Add(tm1).Out(tm1)

	p2 := NewPatch(nil, []int{1})
	p2.SetID("p2")
	tm2 := NewTestModule(0.5)
	tm2.SetID("tm2")
	p2.Add(tm2)
	p.In(tm2)
	p2.Add(p).Out(p)

	p2.Run()

	t.Logf("value=%f", p2.Output(0).Channels[0][0])
}
