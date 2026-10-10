// Package mixer is a mixing desk: channel strips with volume (dB), pan, mute, solo, sends and
// meters, effect buses (sends summed into effects, returned on a strip), groups (sub mixes with
// insert effects) and a master with a lookahead limiter.
//
//	mix := mixer.New(p)
//	mix.Bus("hall", mixer.StripConfig{}, fdnreverb.New(wetHall).AddTo(p))
//	mix.Group("strings", mixer.StripConfig{Volume: -3}, filter)
//	mix.Channel("violins", violins, mixer.StripConfig{Pan: -0.4, Group: "strings", Sends: map[string]float64{"hall": 0.3}})
//	p.Out(mix.Master())
//
// Buses and groups are created before the strips that play into them. Effects and inserts are
// modules you create and add to the patch, the mixer wires them
package mixer

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/meter"
)

// Mixer holds the strips and the master
type Mixer struct {
	aligned bool
	p       psalm.Patcher
	master  *Master
	strips  atomic.Pointer[[]*Strip] // in the order they were created; replaced, never changed (the audio thread reads it)
	inputs  map[string]psalm.Modular
}

// New creates a mixer with a master, added to the patch
func New(p psalm.Patcher) *Mixer {
	m := &Mixer{p: p, inputs: map[string]psalm.Modular{}}
	m.master = newMaster(m)
	m.master.AddTo(p)
	return m
}

// Master returns the master, connect it to the output: p.Out(mix.Master())
func (m *Mixer) Master() *Master {
	return m.master
}

// Strip returns a strip (channel, group or bus return) by name
func (m *Mixer) Strip(name string) *Strip {
	for _, s := range m.list() {
		if s.name == name {
			return s
		}
	}
	return nil
}

// Strips returns the strips (channels, groups and bus returns) in the order they were created
func (m *Mixer) Strips() []*Strip {
	return m.list()
}

// list returns the strips
func (m *Mixer) list() []*Strip {
	if l := m.strips.Load(); l != nil {
		return *l
	}
	return nil
}

func (m *Mixer) setStrips(l []*Strip) { m.strips.Store(&l) }

// Channel adds a strip for a source (any module with an audio output 0)
func (m *Mixer) Channel(name string, source psalm.Modular, cfg StripConfig) *Strip {
	s := m.add(name, cfg, false)
	s.kind = channelStrip
	s.In(source)
	return s
}

// Group adds a sub mix: strips with Group set to its name play into it, through the inserts (in
// order), then the group strip
func (m *Mixer) Group(name string, cfg StripConfig, inserts ...psalm.Modular) *Strip {
	in := newSum()
	in.AddTo(m.p)
	m.inputs[name] = in
	s := m.add(name, cfg, true)
	s.kind = groupStrip
	s.In(chain(in, inserts))
	s.owned = append([]psalm.Modular{in}, inserts...)
	return s
}

// Bus adds an effect bus: the sends of the strips are summed into the effects (in order), their
// output returns on a strip. Set the effects fully wet (a reverb mix of 1), the dry signal stays
// on the strips
func (m *Mixer) Bus(name string, cfg StripConfig, effects ...psalm.Modular) *Strip {
	in := newSum()
	in.AddTo(m.p)
	m.inputs["bus:"+name] = in
	s := m.add(name, cfg, true)
	s.kind = busStrip
	s.In(chain(in, effects))
	s.owned = append([]psalm.Modular{in}, effects...)
	return s
}

// Remove takes a strip out of the mixer: it fades out over some seconds (no click), then it is
// disconnected and removed from the patch. A group or a bus goes with what it is made of (its input
// and its inserts or effects); a group can only go when no strip plays into it any more, a bus
// leaves the sends to it going nowhere. A channel's source stays in the patch: remove it too
// (p.FadeRemove(source, seconds)) or play it elsewhere. Call it on the audio thread while the patch
// plays (Psalm.Do), like Patch.Remove
func (m *Mixer) Remove(name string, seconds float64) error {
	s := m.Strip(name)
	if s == nil {
		return fmt.Errorf("mixer: no strip named %q", name)
	}
	if s.kind == groupStrip {
		for _, other := range m.list() {
			if other.cfg.Group == name && !other.removing {
				return fmt.Errorf("mixer: %s still plays into group %q: remove it first", other.name, name)
			}
		}
	}
	if s.removing {
		return nil
	}
	s.removing = true
	detach := func() {
		m.setStrips(slices.DeleteFunc(slices.Clone(m.list()), func(o *Strip) bool { return o == s }))
		m.p.Remove(s)
		for _, mod := range s.owned {
			m.p.Remove(mod)
		}
		switch s.kind {
		case groupStrip:
			delete(m.inputs, name)
		case busStrip:
			delete(m.inputs, "bus:"+name)
		}
	}
	if seconds <= 0 {
		detach()
		return nil
	}
	s.FadeOut(seconds, detach)
	return nil
}

// add creates a strip and routes its output and sends
func (m *Mixer) add(name string, cfg StripConfig, safe bool) *Strip {
	if m.Strip(name) != nil {
		panic(fmt.Sprintf("mixer: a strip named %q already exists", name))
	}
	if cfg.Sends == nil {
		cfg.Sends = map[string]float64{}
	}
	// check the routing first, so a mistake leaves the mixer as it was
	if _, ok := m.inputs[cfg.Group]; cfg.Group != "" && !ok {
		panic(fmt.Sprintf("mixer: %s plays into group %q, which doesn't exist (create groups first)", name, cfg.Group))
	}
	for bus := range cfg.Sends {
		if _, ok := m.inputs["bus:"+bus]; !ok {
			panic(fmt.Sprintf("mixer: %s sends to bus %q, which doesn't exist (create buses first)", name, bus))
		}
	}
	s := newStrip(m, name, cfg, safe)
	s.SetID(name) // for profiles
	s.AddTo(m.p)
	m.setStrips(append(slices.Clone(m.list()), s))

	// output 0 to the group or the master
	if cfg.Group == "" {
		m.master.In(s)
	} else {
		m.inputs[cfg.Group].In(s)
	}

	// the sends
	for i, bus := range s.sends {
		m.inputs["bus:"+bus].In(s, i+1, 0)
	}
	return s
}

// Align lines up the strips that play into the same group or the master: a strip whose signal is
// late (a vocoder, a fully wet convolver, anything reporting psalm.Latent along its sources and
// inserts) makes the mixer delay the other strips by the difference, so they arrive together. The
// strips delay their input, so their sends stay lined up too. Effect bus returns are left as they
// are, the latency of a reverb is a natural pre delay. It runs by itself before the first buffer,
// call it again after adding strips while playing
func (m *Mixer) Align() {
	m.aligned = true
	for _, s := range m.list() {
		s.setDelay(0)
	}
	var align func(dest string)
	align = func(dest string) {
		var members []*Strip
		for _, s := range m.list() {
			if s.cfg.Group == dest && s.kind != busStrip {
				members = append(members, s)
			}
		}
		arrivals := make([]int, len(members))
		latest := 0
		for i, s := range members {
			if s.kind == groupStrip {
				align(s.name) // its members first
			}
			arrivals[i] = upstreamLatency(s, map[psalm.Modular]int{})
			latest = max(latest, arrivals[i])
		}
		for i, s := range members {
			s.setDelay(latest - arrivals[i])
		}
	}
	align("")
}

// upstreamLatency returns the latency of the signal arriving at the input of a module: the most
// over every path of the latencies reported along it
func upstreamLatency(mod psalm.Modular, memo map[psalm.Modular]int) int {
	most := 0
	for _, in := range mod.Inputs() {
		for _, c := range in.Connections {
			most = max(most, pathLatency(c.Module, memo))
		}
	}
	return most
}

// pathLatency returns the latency at the output of a module
func pathLatency(mod psalm.Modular, memo map[psalm.Modular]int) int {
	if v, ok := memo[mod]; ok {
		return v
	}
	memo[mod] = 0 // a loop counts as no latency
	own := 0
	if l, ok := mod.(psalm.Latent); ok {
		own = l.SignalLatency()
	}
	v := own + upstreamLatency(mod, memo)
	memo[mod] = v
	return v
}

// chain connects modules in series after a first one and returns the last
func chain(first psalm.Modular, rest []psalm.Modular) psalm.Modular {
	last := first
	for _, mod := range rest {
		mod.In(last)
		last = mod
	}
	return last
}

func (m *Mixer) anySolo() bool {
	for _, s := range m.list() {
		if s.cfg.Solo && !s.safe {
			return true
		}
	}
	return false
}

// ResetMeters starts measuring again
func (m *Mixer) ResetMeters() {
	for _, s := range m.list() {
		s.meter.reset()
	}
	m.master.meter.reset()
	m.master.loud.Reset()
	m.master.reduction = 0
}

// Meters returns the peak and RMS level of every strip and the master since the meters were reset,
// and the most gain reduction of the limiter
func (m *Mixer) Meters() string {
	var b strings.Builder
	line := func(name string, peak, rms float64) {
		fmt.Fprintf(&b, "%-14s peak %6.1f dB  rms %6.1f dB\n", name, peak, rms)
	}
	for _, s := range m.list() {
		peak, rms := s.Levels()
		line(s.name, peak, rms)
	}
	peak, rms := m.master.Levels()
	line("master", peak, rms)
	fmt.Fprintf(&b, "%-14s %.1f LUFS integrated, %.1f LUFS short-term\n", "loudness", m.master.loud.Integrated(), m.master.loud.ShortTerm())
	fmt.Fprintf(&b, "%-14s %.1f dB\n", "limiter", -m.master.Reduction())
	return b.String()
}

// sum is a stereo summing point (sockets add up all their connections)
type sum struct {
	*psalm.Module
}

func newSum() *sum {
	s := &sum{Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo})}
	s.SetSelf(s)
	return s
}

func (s *sum) Synthesize() {
	s.Output(0).Channels.Copy(s.Input(0).Channels)
}

// MasterConfig holds the master settings
type MasterConfig struct {
	Volume    float64 // dB
	Ceiling   float64 // dB, the limiter keeps peaks below this (0 is off)
	Lookahead float64 // milliseconds the limiter looks ahead (the latency of the master)
	Release   float64 // milliseconds the limiter takes to let go
}

// DefaultMasterConfig limits at -1 dB with 3 ms of lookahead
func DefaultMasterConfig() MasterConfig {
	return MasterConfig{Ceiling: -1, Lookahead: 3, Release: 150}
}

// Master sums the strips, applies the volume and a lookahead limiter: it sees peaks coming and
// lowers the gain smoothly before they arrive, so nothing clips and nothing distorts. Value input 1
// is the volume (dB), messages "volume" and "ceiling"
type Master struct {
	*psalm.Module
	mixer     *Mixer
	cfg       MasterConfig
	vol       float64
	delayL    []float64
	delayR    []float64
	pos       int
	count     int64  // samples processed
	queue     []need // sliding minimum of the needed gains over the lookahead
	head      int    // first entry of the queue
	gain      float64
	meter     levelMeter
	loud      *meter.Loudness
	reduction float64 // most gain reduction (linear) since the meters were reset
}

func newMaster(m *Mixer) *Master {
	ms := &Master{
		Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}),
		mixer:  m,
		gain:   1,
	}
	ms.loud = meter.NewLoudness(ms.Config().SampleRate)
	ms.SetID("master")
	ms.SetSelf(ms)
	ms.SetConfig(DefaultMasterConfig())
	return ms
}

// SetConfig changes the master settings
func (ms *Master) SetConfig(cfg MasterConfig) {
	if cfg.Lookahead <= 0 {
		cfg.Lookahead = 3
	}
	if cfg.Release <= 0 {
		cfg.Release = 150
	}
	ms.cfg = cfg
	ms.vol = DB(cfg.Volume)
	n := max(1, int(cfg.Lookahead*0.001*ms.Config().SampleRate))
	ms.delayL = make([]float64, n)
	ms.delayR = make([]float64, n)
	ms.queue = ms.queue[:0]
	ms.head = 0
	ms.pos = 0
}

// need is the gain a sample needs to stay below the ceiling
type need struct {
	index int64
	gain  float64
}

// push adds the need of the newest sample and returns the lowest need of the last n+1 samples:
// from the sample leaving the lookahead delay up to the newest. Needs that can never be the
// lowest again (an older, higher one) are dropped, so this costs little per sample
func (ms *Master) push(g float64, n int) float64 {
	for len(ms.queue) > ms.head && ms.queue[len(ms.queue)-1].gain >= g {
		ms.queue = ms.queue[:len(ms.queue)-1]
	}
	ms.queue = append(ms.queue, need{ms.count, g})
	for ms.queue[ms.head].index < ms.count-int64(n) {
		ms.head++
	}
	if ms.head > 1024 { // compact now and then
		ms.queue = append(ms.queue[:0], ms.queue[ms.head:]...)
		ms.head = 0
	}
	ms.count++
	return ms.queue[ms.head].gain
}

// Settings returns the master settings
func (ms *Master) Settings() MasterConfig { return ms.cfg }

// Latency returns the delay of the limiter lookahead in samples
func (ms *Master) Latency() int { return len(ms.delayL) }

// Levels returns the peak and RMS level of the master output in dB since the meters were reset
func (ms *Master) Levels() (float64, float64) { return ms.meter.levels() }

// Loudness returns the loudness meter of the master output (EBU R128, LUFS)
func (ms *Master) Loudness() *meter.Loudness { return ms.loud }

// Reduction returns the most gain reduction of the limiter in dB since the meters were reset
func (ms *Master) Reduction() float64 {
	if ms.reduction <= 0 {
		return 0
	}
	return -ToDB(1 - ms.reduction)
}

func (ms *Master) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		ms.ReceiveMessage(msg)
		return
	}
	if index == 1 {
		ms.cfg.Volume = psalm.FltVal[float64](value)
	}
}

func (ms *Master) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("volume", func(v float64) { ms.cfg.Volume = v })
	msg.IfFltVal("ceiling", func(v float64) { ms.cfg.Ceiling = v })
}

func (ms *Master) Synthesize() {
	var (
		sr      = ms.Config().SampleRate
		in      = ms.Input(0).Channels
		out     = ms.Output(0).Channels
		n       = len(ms.delayL)
		smooth  = 1 - math.Exp(-1/(0.02*sr))
		attack  = 1 - math.Exp(-1/(float64(n)/4)) // reaches the needed gain within the lookahead
		release = 1 - math.Exp(-1/(ms.cfg.Release*0.001*sr))
		target  = DB(ms.cfg.Volume)
		limit   = ms.cfg.Ceiling != 0
		ceiling = DB(ms.cfg.Ceiling)
	)

	for i := range out[0] {
		ms.vol += (target - ms.vol) * smooth
		l := float64(in[0][i]) * ms.vol
		r := float64(in[1][i]) * ms.vol

		if !limit {
			out[0][i] = float32(l)
			out[1][i] = float32(r)
			ms.meter.add(l, r)
			ms.loud.Add(l, r)
			continue
		}

		// the gain this sample needs to stay below the ceiling, and the lowest need until it
		// leaves the lookahead
		g := 1.0
		if peak := math.Max(math.Abs(l), math.Abs(r)); peak > ceiling {
			g = ceiling / peak
		}
		minNeed := ms.push(g, n)

		// the gain falls fast enough to be there when the peak leaves the delay, rises slowly
		if minNeed < ms.gain {
			ms.gain += (minNeed - ms.gain) * attack
		} else {
			ms.gain += (minNeed - ms.gain) * release
		}

		// delay the audio by the lookahead
		pos := ms.pos
		dl, dr := ms.delayL[pos], ms.delayR[pos]
		ms.delayL[pos], ms.delayR[pos] = l, r
		ms.pos = (pos + 1) % n

		ol, or := dl*ms.gain, dr*ms.gain
		// whatever the smoothing still lets through is clipped at the ceiling
		ol = math.Max(-ceiling, math.Min(ol, ceiling))
		or = math.Max(-ceiling, math.Min(or, ceiling))

		out[0][i] = float32(ol)
		out[1][i] = float32(or)
		ms.meter.add(ol, or)
		ms.loud.Add(ol, or)
		ms.reduction = math.Max(ms.reduction, 1-ms.gain)
	}
}

// Tail lets a summing point rest while silent
func (s *sum) Tail() float64 {
	return 0
}
