package psalm

import (
	"math/rand/v2"
	"time"
)

type CtrlTick struct {
	SampleTimestamp int64
	TickTimestamp   int64
	SampleOffset    int
	BPM             float64
	PPQ             int
}

type Receiver interface {
	Receive(any, int)
	ReceiveMessage(Message)
}

type Modular interface {
	Selfie[Modular]
	Receiver
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
	Config() Config
	CtrlTick(CtrlTick)
	Send(any, int)
	Muted() bool
	SetMuted(bool) Modular
}

type Module struct {
	hasRun       bool
	muted        bool
	id           string
	self         Modular
	inputs       []*Socket
	outputs      []*Socket
	valueOutputs map[int][]*Connection
	config       Config
	env          *Env          // the environment of the patch the module belongs to
	rng          *rand.Rand    // the module's own random stream, taken on first use
	cpu          time.Duration // time spent in Synthesize while profiling
}

// profiling makes every module measure the time it spends synthesizing
var profiling bool

// SetProfiling turns the measuring of the time every module spends synthesizing on or off (off
// by default, it costs a little time itself). Read it with CPUTime
func SetProfiling(on bool) {
	profiling = on
}

// CPUTime returns the time the module spent synthesizing while profiling was on
func (m *Module) CPUTime() time.Duration {
	return m.cpu
}

// ResetCPUTime sets the measured time back to 0
func (m *Module) ResetCPUTime() {
	m.cpu = 0
}

func NewModule(ins []int, outs []int) *Module {
	env := CurrentEnv()
	config := env.Config()

	m := &Module{
		env:          env,
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

func (m *Module) Muted() bool {
	return m.muted
}

func (m *Module) SetMuted(muted bool) Modular {
	m.muted = muted
	return m.self
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

	if profiling {
		start := time.Now()
		m.self.Synthesize()
		m.cpu += time.Since(start)
	} else {
		m.self.Synthesize()
	}

	if m.muted {
		for _, out := range m.outputs {
			out.Clear()
		}
	}
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

// Env returns the environment of the patch the module belongs to (the current one when it was
// created)
func (m *Module) Env() *Env {
	return m.env
}

// BPM returns the current tempo of the module's patch
func (m *Module) BPM() float64 {
	return m.env.BPM()
}

// Rand returns the module's own random stream, from its environment's seed
func (m *Module) Rand() *rand.Rand {
	if m.rng == nil {
		m.rng = m.env.NewRand()
	}
	return m.rng
}

// Config returns the config of the module when it was created (sample rate, buffer size, PPQ;
// read the live tempo with BPM)
func (m *Module) Config() Config {
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
