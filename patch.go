package psalm

import (
	"fmt"
)

type internalInputModule struct {
	*Module
}

func newInternalInputModule(n int) *internalInputModule {
	return &internalInputModule{
		Module: NewModule(nil, []int{n}),
	}
}

func (m *internalInputModule) Receive(value any, index int) {
	m.Send(value, index)
}

type internalOutputModule struct {
	*Module
	p Patcher
}

func newInternalOutputModule(p Patcher, n int) *internalOutputModule {
	return &internalOutputModule{
		Module: NewModule([]int{n}, nil),
		p:      p,
	}
}

func (m *internalOutputModule) Receive(value any, index int) {
	m.p.Send(value, index)
}

type Patcher interface {
	Modular
	Add(m Modular) Patcher
	PatchInput(index int) Modular
	PatchOutput(index int) Modular
	Out(...any) Patcher
	ValueOut(...any) Patcher
}

type Patch struct {
	*Module
	modules       []Modular
	inputModules  []*internalInputModule
	outputModules []*internalOutputModule
	skipInputs    bool
}

func NewPatch(ins []int, outs []int) *Patch {
	p := &Patch{Module: NewModule(ins, outs)}
	p.SetSelf(p)

	inputModules := make([]*internalInputModule, len(ins))
	for i, numChannels := range ins {
		inputModules[i] = newInternalInputModule(numChannels)
	}

	outputModules := make([]*internalOutputModule, len(outs))
	for i, numChannels := range outs {
		outputModules[i] = newInternalOutputModule(p, numChannels)
	}

	p.inputModules = inputModules
	p.outputModules = outputModules

	return p
}

func (p *Patch) SetSkipInputs(skip bool) {
	// this leaves room for manually setting input buffers (to pipe audio input into patch for instance)
	p.skipInputs = skip
}

func (p *Patch) Add(m Modular) Patcher {
	p.modules = append(p.modules, m)
	return p
}

func (p *Patch) ResetRun() {
	p.hasRun = false

	for _, input := range p.inputModules {
		input.ResetRun()
	}
	for _, output := range p.outputModules {
		output.ResetRun()
	}
	for _, m := range p.modules {
		m.ResetRun()
	}
}

func (p *Patch) PatchInput(index int) Modular {
	return p.inputModules[index]
}

func (p *Patch) PatchOutput(index int) Modular {
	return p.outputModules[index]
}

func (p *Patch) SetID(id string) Modular {
	for i, m := range p.inputModules {
		m.SetID(fmt.Sprintf("%s_input_%d", id, i+1))
	}
	for i, m := range p.outputModules {
		m.SetID(fmt.Sprintf("%s_output_%d", id, i+1))
	}
	p.Module.SetID(id)
	return p.self
}

func (p *Patch) Run() {
	if p.hasRun {
		return
	}

	p.hasRun = true

	if !p.skipInputs {
		// request runs from outer connected modules
		for _, input := range p.inputs {
			for _, conn := range input.Connections {
				conn.Module.Run()
			}
			input.ProcessIncoming()
		}

		// copy external input to internal input modules's output
		for i, input := range p.inputs {
			p.inputModules[i].Output(0).Copy(input)
		}
	}

	// let internal patched modules run by pulling from internal output modules
	for _, outputModule := range p.outputModules {
		outputModule.Run()
	}

	// copy internal output modules's input to external output
	for i, output := range p.outputs {
		output.Copy(p.outputModules[i].Input(0))
	}

	p.self.Synthesize()

	if p.muted {
		for _, out := range p.outputs {
			out.Clear()
		}
	}
}

func (p *Patch) Out(args ...any) Patcher {
	for _, conn := range ParseConnectionDescs(args...) {
		conn.Module.Connect(conn.OutIndex, p.outputModules[conn.InIndex], 0)
	}

	return p.self.(Patcher)
}

func (p *Patch) ValueOut(args ...any) Patcher {
	for _, conn := range ParseConnectionDescs(args...) {
		conn.Module.ValueConnect(conn.OutIndex, p.outputModules[conn.InIndex], 0)
	}

	return p.self.(Patcher)
}

func (p *Patch) CtrlTick(tick CtrlTick) {
	for _, m := range p.modules {
		m.CtrlTick(tick)
	}
}

func (p *Patch) Receive(value any, index int) {
	p.inputModules[index].Receive(value, 0)
}
