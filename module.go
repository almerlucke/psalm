package psalm

type CtrlTick struct {
	SampleTimestamp int64
	TickTimestamp   int64
	SampleOffset    int
	BPM             int
	PPQ             int
}

type Modular interface {
	Selfie[Modular]
	ID() string
	SetID(string) Modular
	HasRun() bool
	SetHasRun(bool)
	Run()
	ResetRun()
	Synthesize()
	NumInputs() int
	NumOutputs() int
	Input(int) *Socket
	Output(int) *Socket
	Inputs() []*Socket
	Outputs() []*Socket
	Connect(int, Modular, int)
	AddInputConnection(int, *Connection)
	AddOutputConnection(int, *Connection)
	In(...any) Modular
	ValueConnect(int, Modular, int)
	ValueIn(...any) Modular
	AddTo(Patcher) Modular
	Config() *Config
	CtrlTick(CtrlTick)
	Receive(any, int)
	Send(any, int)
	ReceiveMessage(Message)
}

type Module struct {
	hasRun       bool
	id           string
	self         Modular
	inputs       []*Socket
	outputs      []*Socket
	valueOutputs map[int][]*Connection
	config       *Config
}

func NewModule(ins []int, outs []int) *Module {
	config := CurrentConfig()
	if config == nil {
		panic("No config set")
	}

	m := &Module{
		inputs:       make([]*Socket, len(ins)),
		outputs:      make([]*Socket, len(outs)),
		valueOutputs: make(map[int][]*Connection),
		config:       config,
	}

	for i, n := range ins {
		m.inputs[i] = &Socket{
			Channels: NewChannels(n, config.BufferSize),
		}
	}

	for i, n := range outs {
		m.outputs[i] = &Socket{
			Channels: NewChannels(n, config.BufferSize),
		}
	}

	m.self = m

	return m
}

func NewControl() *Module {
	return NewModule(nil, nil)
}

func (m *Module) ID() string {
	return m.id
}

func (m *Module) SetID(id string) Modular {
	m.id = id
	return m.self
}

func (m *Module) Self() Modular {
	return m.self
}

func (m *Module) SetSelf(self Modular) {
	m.self = self
}

func (m *Module) HasRun() bool {
	return m.hasRun
}

func (m *Module) SetHasRun(hasRun bool) {
	m.hasRun = hasRun
}

func (m *Module) Run() {
	if m.hasRun {
		return
	}

	m.hasRun = true

	for _, input := range m.inputs {
		for _, conn := range input.Connections {
			conn.Module.Run()
		}
		input.ProcessIncoming()
	}

	m.self.Synthesize()
}

func (m *Module) ResetRun() {
	m.hasRun = false
}

func (m *Module) Synthesize() {
}

func (m *Module) NumInputs() int {
	return len(m.inputs)
}

func (m *Module) NumOutputs() int {
	return len(m.outputs)
}

func (m *Module) Input(index int) *Socket {
	return m.inputs[index]
}

func (m *Module) Output(index int) *Socket {
	return m.outputs[index]
}

func (m *Module) Inputs() []*Socket {
	return m.inputs
}

func (m *Module) Outputs() []*Socket {
	return m.outputs
}

func (m *Module) Connect(outIndex int, to Modular, inIndex int) {
	m.self.AddOutputConnection(outIndex, &Connection{
		Index:  inIndex,
		Module: to,
	})
	to.AddInputConnection(inIndex, &Connection{
		Index:  outIndex,
		Module: m.self,
	})
}

func (m *Module) AddInputConnection(index int, conn *Connection) {
	m.inputs[index].AddConnection(conn)
}

func (m *Module) AddOutputConnection(index int, conn *Connection) {
	m.outputs[index].AddConnection(conn)
}

func (m *Module) In(args ...any) Modular {
	for _, conn := range ParseConnectionDescs(args...) {
		conn.Module.Connect(conn.OutIndex, m.self, conn.InIndex)
	}

	return m.self
}

func (m *Module) ValueConnect(outIndex int, to Modular, inIndex int) {
	m.valueOutputs[outIndex] = append(m.valueOutputs[outIndex], &Connection{Index: inIndex, Module: to})
}

func (m *Module) ValueIn(args ...any) Modular {
	for _, conn := range ParseConnectionDescs(args...) {
		conn.Module.ValueConnect(conn.OutIndex, m.self, conn.InIndex)
	}

	return m.self
}

func (m *Module) Config() *Config {
	return m.config
}

func (m *Module) AddTo(p Patcher) Modular {
	p.Add(m.self)
	return m.self
}

func (m *Module) CtrlTick(tick CtrlTick) {
	// STUB
}

func (m *Module) Receive(value any, index int) {
	// STUB
}

func (m *Module) Send(value any, index int) {
	for _, conn := range m.valueOutputs[index] {
		conn.Module.Receive(value, conn.Index)
	}
}

func (m *Module) ReceiveMessage(msg Message) {

}
