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
	Disconnect(int, Modular, int) bool
	ValueDisconnect(int, Modular, int) bool
	DisconnectAll()
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
	valueInputs  []valueLink // who sends values to the module, to disconnect it
	config       Config
	env          *Env          // the environment of the patch the module belongs to
	rng          *rand.Rand    // the module's own random stream, taken on first use
	cpu          time.Duration // time spent in Synthesize while profiling
	quietIn      int           // samples the input and output have been silent (modules with a tail)
	quietOut     bool          // the last output was silent
	skipping     bool          // skipping its work while silent
}

// Tailer is a module that can skip its work while it is silent: when its input and output have
// been silent for longer than its tail (in seconds), it stops synthesizing and outputs silence,
// until the input sounds again. The tail is the longest silence after which sound can still come
// out, like the echo time of a delay or the pre delay of a reverb (a ringing reverb keeps
// sounding, so it keeps working until its tail is silent anyway).
// A negative tail never skips, for example a frozen granulator
type Tailer interface {
	Tail() float64
}

// TailInputs is a Tailer whose silence depends on some of its inputs only: a vocoder is silent
// when its carrier is, whatever its modulator does
type TailInputs interface {
	TailInputs() []int
}

// Latent is a module whose whole signal comes out late (a vocoder sounds a frame late): the mixer
// delays the other strips to line them up. Modules that are only partly late (a reverb mixed with
// its dry signal) report 0, their latency is a pre delay
type Latent interface {
	SignalLatency() int
}

// valueLink is a value connection seen from the module receiving: the module sending, its output
// and the input
type valueLink struct {
	from    Modular
	out, in int
}

// moduleOf returns the Module of a modular (every module embeds one), nil for none
func moduleOf(m Modular) *Module {
	if b, ok := m.(interface{ base() *Module }); ok {
		return b.base()
	}
	return nil
}

func (m *Module) base() *Module { return m }

// silence is the level below which audio counts as silent (-120 dB)
const silence = 1e-6

// channelsSilent returns true when every sample of the sockets is silent
func channelsSilent(sockets []*Socket) bool {
	for _, s := range sockets {
		for _, ch := range s.Channels {
			for _, x := range ch {
				if x > silence || x < -silence {
					return false
				}
			}
		}
	}
	return true
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

	// a module with a tail skips its work while its input and its output are silent
	tailer, quiet := m.self.(Tailer)
	if quiet {
		inputs := m.inputs
		if ti, ok := m.self.(TailInputs); ok {
			inputs = inputs[:0:0]
			for _, i := range ti.TailInputs() {
				inputs = append(inputs, m.inputs[i])
			}
		}
		if channelsSilent(inputs) {
			m.quietIn += m.config.BufferSize
		} else {
			m.quietIn = 0
		}
		tail := tailer.Tail()
		if tail >= 0 && m.quietOut && float64(m.quietIn) > tail*m.config.SampleRate {
			if !m.skipping {
				for _, out := range m.outputs {
					out.Clear()
				}
				m.skipping = true
			}
			return
		}
		m.skipping = false
	}

	if profiling {
		start := time.Now()
		m.self.Synthesize()
		m.cpu += time.Since(start)
	} else {
		m.self.Synthesize()
	}

	if quiet {
		m.quietOut = channelsSilent(m.outputs)
		if !m.quietOut {
			m.quietIn = 0 // the tail counts from the last sound in or out (a delay between echoes)
		}
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
	if b := moduleOf(to); b != nil {
		b.valueInputs = append(b.valueInputs, valueLink{from: m.self, out: outIndex, in: inIndex})
	}
}

// Disconnect takes away the audio connection from an output of the module to an input of another
// (every one, when they were connected more than once), and tells whether there was one. Change
// the connections of a patch that plays on the audio thread: with Psalm.Do
func (m *Module) Disconnect(outIndex int, to Modular, inIndex int) bool {
	if outIndex < 0 || outIndex >= len(m.outputs) || inIndex < 0 || inIndex >= to.NumInputs() {
		return false
	}
	out := m.outputs[outIndex].RemoveConnection(to, inIndex)
	in := to.Input(inIndex).RemoveConnection(m.self, outIndex)
	return out || in
}

// ValueDisconnect takes away the value connection from a value output of the module to a value
// input of another, and tells whether there was one
func (m *Module) ValueDisconnect(outIndex int, to Modular, inIndex int) bool {
	conns := m.valueOutputs[outIndex]
	kept := make([]*Connection, 0, len(conns))
	for _, c := range conns {
		if c.Module != to || c.Index != inIndex {
			kept = append(kept, c)
		}
	}
	if len(kept) == len(conns) {
		return false
	}
	m.valueOutputs[outIndex] = kept // a new list: a Send going on keeps the one it has
	if b := moduleOf(to); b != nil {
		links := make([]valueLink, 0, len(b.valueInputs))
		for _, l := range b.valueInputs {
			if l.from != m.self || l.out != outIndex || l.in != inIndex {
				links = append(links, l)
			}
		}
		b.valueInputs = links
	}
	return true
}

// DisconnectAll takes away every connection of the module: the audio coming in and going out, the
// values it sends and the values sent to it. The module is on its own afterwards, ready to be
// removed from its patch (Patch.Remove does both) or connected elsewhere
func (m *Module) DisconnectAll() {
	for i, in := range m.inputs {
		for _, c := range in.Connections {
			c.Module.Disconnect(c.Index, m.self, i)
		}
	}
	for i, out := range m.outputs {
		for _, c := range out.Connections {
			m.Disconnect(i, c.Module, c.Index)
		}
	}
	for i, conns := range m.valueOutputs {
		for _, c := range conns {
			m.ValueDisconnect(i, c.Module, c.Index)
		}
	}
	for _, l := range m.valueInputs {
		l.from.ValueDisconnect(l.out, m.self, l.in)
	}
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
