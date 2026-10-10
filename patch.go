package psalm

import (
	"fmt"
	"slices"
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
	Remove(m Modular) bool
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
	leaving       []Modular // fading out to be removed (FadeRemove)
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

// Remove takes a module out of the patch: it is disconnected from everything (DisconnectAll) and
// gets no control ticks anymore. A patch within the patch keeps its own modules and connections,
// so it can be added again later, or elsewhere. It tells whether the module was in the patch.
// Change a patch that plays on the audio thread: with Psalm.Do
func (p *Patch) Remove(m Modular) bool {
	kept := make([]Modular, 0, len(p.modules))
	for _, other := range p.modules {
		if other != m {
			kept = append(kept, other)
		}
	}
	if len(kept) == len(p.modules) {
		return false
	}
	p.modules = kept // a new list: a loop over the modules going on keeps the one it has
	if i := slices.Index(p.leaving, m); i >= 0 {
		p.leaving = slices.Delete(slices.Clone(p.leaving), i, i+1)
	}
	m.DisconnectAll()
	return true
}

// Modules returns the modules added to the patch
func (p *Patch) Modules() []Modular {
	return p.modules
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
	p.applyFade()

	// modules fading out to be removed that nothing heard this buffer: nothing to fade, gone now
	for _, m := range p.leaving {
		if !m.HasRun() {
			p.Remove(m)
		}
	}
}

// FadeRemove fades a module out over some seconds and then removes it (Remove): no click when it is
// sounding. A module nothing hears (a control module, or one nothing pulls audio from) is removed at
// the end of the next buffer. Call it where Remove is called: on the audio thread while the patch
// plays (Psalm.Do)
func (p *Patch) FadeRemove(m Modular, seconds float64) {
	b := moduleOf(m)
	if b == nil || len(b.outputs) == 0 || seconds <= 0 {
		p.Remove(m)
		return
	}
	p.leaving = append(p.leaving, m)
	b.FadeOut(seconds, func() { p.Remove(m) })
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
