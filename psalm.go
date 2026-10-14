package psalm

import (
	"errors"
	"log"
	"math"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/almerlucke/psalm/audio"
	"github.com/almerlucke/psalm/tools/signal"
	"github.com/almerlucke/sndfile/writer"
	"github.com/dh1tw/gosamplerate"
)

type Psalm struct {
	*Patch

	// Control
	samplesPerTick  atomic.Uint64 // float64 bits: the tempo can change from any goroutine
	sampleCnt       float64
	sampleTimestamp int64
	tickTimestamp   int64

	// Audio
	engine  audio.Engine
	audioMu sync.Mutex  // serializes starting and stopping (from the main goroutine, a panel's page)
	running atomic.Bool // the engine is started
	restart atomic.Bool // back to the start at the next buffer
	letGo   atomic.Bool // let go of the notes playing at the next buffer (stopped, restarted)

	// changes to make on the audio thread, between two buffers
	doMu  sync.Mutex
	doing []func()

	host  *Psalm      // the patch this one plays within (NewSub), nil for a patch of its own
	held  atomic.Bool // the clock stands still (HoldClock)
	delay float64     // samples the clock waits before its next tick (DelayStart), after a restart
}

// Do makes a change to the patch where it is safe: on the audio thread between two buffers while
// it plays, at once while it doesn't (stopped, or rendering offline from the goroutine calling).
// Connecting, disconnecting, adding and removing modules while the patch plays goes through Do:
//
//	p.Do(func() {
//		p.Remove(oldSynth)
//		newSynth.AddTo(p)
//		mixer.Channel(...)
//	})
//
// The changes are made in the order they were asked for, all of one function in the same pause, so
// the patch never plays half a change. Keep them short: the audio waits for them. It can be called
// from any goroutine, but not from within f (or a function f calls Do from)
func (p *Psalm) Do(f func()) {
	if p.host != nil { // within another patch: on its audio thread
		p.host.Do(f)
		return
	}
	p.audioMu.Lock()
	defer p.audioMu.Unlock()
	if !p.running.Load() {
		p.doPending() // what was asked before it stopped first
		f()
		return
	}
	p.doMu.Lock()
	p.doing = append(p.doing, f)
	p.doMu.Unlock()
}

// doPending makes the changes asked for; on the audio thread it never waits for the lock (a change
// being asked for right then is made at the next buffer)
func (p *Psalm) doPending() {
	if !p.doMu.TryLock() {
		return
	}
	fs := p.doing
	p.doing = nil
	p.doMu.Unlock()
	for _, f := range fs {
		f()
	}
}

// Restarter is a module that keeps time of its own (a sequencer, an arrangement): Restart sets it
// back to the start of the patch. It is called on the audio thread, by Psalm.Restart
type Restarter interface {
	Restart()
}

// NoteReleaser is a module that plays notes (a synth): AllNotesOff lets go of every note it plays,
// held or of a duration, into its release. It is called on the audio thread, when the patch stops or
// restarts
type NoteReleaser interface {
	AllNotesOff()
}

// Restart sets the patch back to its start at the next buffer: the clock goes back to 0 and every
// module that keeps time of its own (a Restarter, also in patches within the patch) starts again,
// so the song, the parts and the sequencers play from the beginning. The notes playing are let go
// (a NoteReleaser's), what rings (a release, a reverb) rings out. It can be called from any
// goroutine, also while the patch is stopped (it starts from the beginning when it plays again)
func (p *Psalm) Restart() { p.restart.Store(true) }

// releaseNotes lets go of the notes of every module that plays them, in patches within patches too
func releaseNotes(modules []Modular) {
	for _, m := range modules {
		if r, ok := m.(NoteReleaser); ok {
			r.AllNotesOff()
		}
		if sub, ok := m.(interface{ Modules() []Modular }); ok {
			releaseNotes(sub.Modules())
		}
	}
}

// restartModules restarts the modules that keep time, in patches within patches too (a Restarter
// restarts what it holds itself: a patch within a patch with a clock of its own, NewSub)
func restartModules(modules []Modular) {
	for _, m := range modules {
		if r, ok := m.(Restarter); ok {
			r.Restart()
			continue
		}
		if sub, ok := m.(interface{ Modules() []Modular }); ok {
			restartModules(sub.Modules())
		}
	}
}

// New creates a patch to play through an audio engine (nil renders offline). It creates the
// environment of the patch (its config, tempo and random streams) and makes it current: create
// the modules of the patch after it
func New(engine audio.Engine) *Psalm {
	p := &Psalm{
		engine: engine,
	}

	// the configured settings (PushConfig), not the live state of the patch built before
	cfg := CurrentStackConfig()
	if engine != nil {
		engine.SetCallback(p.audioCallback)
		err := engine.Init()
		if err != nil {
			panic(err)
		}
		cfg.SampleRate = engine.GetSampleRate()
		cfg.BufferSize = engine.GetBufferSize()
	}
	return p.init(cfg)
}

// NewSub creates a patch to play within this one, with a clock and an environment of its own: its
// own tempo (Env().SetBPM), its own random streams, its ticks counted from 0 when it starts to
// play. A song of a playlist is one. Like New it makes its environment current: build its modules
// right after it (and make the environment of this patch current again after, SetCurrentEnv, to
// build more of this one). It has the sample rate and buffer size of this patch, the tempo and PPQ
// of the config (PushConfig). To build one on another goroutine (in the background, while other
// goroutines build too), use NewSubWith and Within instead.
//
// It plays when it runs: the module it is added to, or connected to, runs it once per buffer, and
// then it ticks its own modules. The control ticks of the patch around it don't reach them
func (p *Psalm) NewSub() *Psalm {
	cfg := CurrentStackConfig()
	cfg.SampleRate, cfg.BufferSize = p.config.SampleRate, p.config.BufferSize
	sub := &Psalm{host: p}
	return sub.init(cfg)
}

// NewSubWith is NewSub for building on another goroutine (a song built in the background): it has a
// config of its own (the tempo and PPQ; the sample rate and buffer size are always the ones of this
// patch) and makes nothing current for the program. Build its modules within it:
//
//	sub := p.NewSubWith(cfg)
//	sub.Within(func() { synth := subtractive.NewPoly(...).AddTo(sub) })
//
// Other goroutines building modules meanwhile build them in their own patch
func (p *Psalm) NewSubWith(cfg Config) *Psalm {
	cfg.SampleRate, cfg.BufferSize = p.config.SampleRate, p.config.BufferSize
	sub := &Psalm{host: p}
	env := NewEnv(cfg)
	within(env, func() { sub.build(env) })
	return sub
}

// Within runs f with the environment of the patch current for the goroutine calling only: the
// modules f creates belong to this patch, whatever other goroutines build meanwhile. Build a patch
// made with NewSubWith within it
func (p *Psalm) Within(f func()) { within(p.Env(), f) }

// HoldClock makes the clock of the patch stand still, or go on: no ticks, so no new notes and no
// moving song, while the sound goes on (notes that sound ring out, a reverb tails). A song that
// ended in a playlist holds its clock
func (p *Psalm) HoldClock(hold bool) { p.held.Store(hold) }

// ClockHeld tells whether the clock stands still
func (p *Psalm) ClockHeld() bool { return p.held.Load() }

// Ticks returns the ticks of the patch's clock since it started (or restarted): where a song is,
// for a playlist bringing the next one in on its bar lines. Read it on the audio thread
func (p *Psalm) Ticks() int64 { return p.tickTimestamp }

// SamplesUntil returns the samples from the start of the next buffer until the tick at a position
// (counted from 0, as modules see it in TickTimestamp) comes, at the current tempo. Read it on the
// audio thread
func (p *Psalm) SamplesUntil(position int64) float64 {
	spt := p.SamplesPerTick()
	return float64(position-p.tickTimestamp)*spt + (spt - p.sampleCnt)
}

// SamplesPerTick returns the samples between two ticks at the current tempo
func (p *Psalm) SamplesPerTick() float64 { return math.Float64frombits(p.samplesPerTick.Load()) }

// DelayStart makes the clock wait a number of samples before its next tick: a song starting
// within a buffer, sample exact (its first notes come that many samples into the buffer). Call it
// on the audio thread, before the patch runs
func (p *Psalm) DelayStart(samples float64) { p.delay += samples }

// CtrlTick does nothing: a patch with a clock of its own ticks its modules itself, when it runs
func (p *Psalm) CtrlTick(CtrlTick) {}

// init creates the environment and the patch, and makes the environment current
func (p *Psalm) init(cfg Config) *Psalm {
	env := NewEnv(cfg)
	SetCurrentEnv(env)
	return p.build(env)
}

// build creates the patch in the current environment (env)
func (p *Psalm) build(env *Env) *Psalm {
	p.Patch = NewPatch([]int{2}, []int{2})
	p.SetSelf(p)
	p.SetID("psalm")
	p.SetSkipInputs(true)

	env.OnTempo(p.updateCtrlTick)

	p.updateCtrlTick()

	return p
}

// OnConfigChanged follows a config change
func (p *Psalm) OnConfigChanged() {
	p.updateCtrlTick()
}

func (p *Psalm) updateCtrlTick() {
	p.samplesPerTick.Store(math.Float64bits(p.Env().TickDuration() * p.config.SampleRate))
}

func (p *Psalm) ToSoundFile(filePath string, fileFormat writer.FileFormat, numSeconds float64, sampleRate float64, normalize bool) error {
	config := p.config
	inputSampleRate := config.SampleRate
	numChannels := 2
	writeBuffers := make([][]float32, numChannels)
	pipe := p.Output(0).Channels

	for i := 0; i < numChannels; i++ {
		writeBuffers[i] = pipe[i]
	}

	wr, err := writer.NewWithOptions(filePath, fileFormat, numChannels, sampleRate, writer.Options{
		InputConverter:    writer.NewChannelConverter[float32](config.BufferSize, 2),
		Normalize:         normalize,
		ConvertSampleRate: inputSampleRate != sampleRate,
		InputSampleRate:   inputSampleRate,
		SrConvQuality:     gosamplerate.SRC_SINC_BEST_QUALITY,
	})
	if err != nil {
		return err
	}

	defer func() {
		_ = wr.Close()
	}()

	numCycles := int64(math.Ceil((config.SampleRate * numSeconds) / float64(config.BufferSize)))

	if numCycles == 0 {
		return nil
	}

	for numCycles > 1 {
		p.ResetRun()
		p.Run()
		err = wr.Write(writeBuffers, false)
		if err != nil {
			return err
		}
		numCycles--
	}

	p.ResetRun()
	p.Run()

	return wr.Write(writeBuffers, true)
}

// AudioStart starts playing through the audio engine; the patch goes on from where it stopped.
// Starting a patch that plays does nothing. It can be called from any goroutine
func (p *Psalm) AudioStart() error {
	if p.engine == nil {
		return errors.New("engine not initialized")
	}
	p.audioMu.Lock()
	defer p.audioMu.Unlock()
	if p.running.Load() {
		return nil
	}
	if err := p.engine.Start(); err != nil {
		return err
	}
	p.running.Store(true)
	return nil
}

// AudioStop stops playing: the patch stands still (every module where it is) until AudioStart,
// when the notes that were playing are let go (they ring out their release; a reverb its tail).
// Stopping a patch that doesn't play does nothing. It can be called from any goroutine
func (p *Psalm) AudioStop() error {
	if p.engine == nil {
		return errors.New("engine not initialized")
	}
	p.audioMu.Lock()
	defer p.audioMu.Unlock()
	if !p.running.Load() {
		return nil
	}
	if err := p.engine.Stop(); err != nil {
		return err
	}
	p.running.Store(false)
	p.letGo.Store(true)
	p.doPending() // the changes still waiting for a buffer are made now
	return nil
}

// Running tells whether the patch plays through its audio engine
func (p *Psalm) Running() bool { return p.running.Load() }

// Live tells whether the patch has an audio engine to play through (it renders offline without)
func (p *Psalm) Live() bool { return p.engine != nil }

func (p *Psalm) Listen() error {
	err := p.AudioStart()
	if err != nil {
		return err
	}

	log.Printf("playing audio until terminated...")

	why := signal.WaitForStop(syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	log.Printf("stopping audio (%s)", why)

	return p.AudioStop()
}

func (p *Psalm) Close() {
	if p.engine != nil {
		p.audioMu.Lock()
		defer p.audioMu.Unlock()
		_ = p.engine.Close()
		p.running.Store(false)
	}
}

func (p *Psalm) audioCallback(in, out [][]float32) {
	inputPipe := p.inputModules[0].Output(0).Channels

	for i := 0; i < p.config.BufferSize; i++ {
		for j := 0; j < 2; j++ {
			inputPipe[j][i] = in[j][i]
		}
	}

	p.ResetRun()
	p.Run()

	channels := p.outputs[0].Channels

	for i := 0; i < p.config.BufferSize; i++ {
		for j := 0; j < 2; j++ {
			out[j][i] = channels[j][i]
		}
	}
}

func (p *Psalm) Run() {
	if p.hasRun { // within another patch it runs once per buffer, however many pull it
		return
	}
	p.doPending()
	if p.restart.Swap(false) {
		p.sampleTimestamp, p.tickTimestamp, p.sampleCnt = 0, 0, 0
		restartModules(p.modules)
		p.letGo.Store(true)
	}
	if p.letGo.Swap(false) { // stopped or restarted: no note goes on for good
		releaseNotes(p.modules)
	}
	if p.delay != 0 {
		p.sampleCnt -= p.delay
		p.delay = 0
	}
	if p.held.Load() {
		p.Patch.Run()
		return
	}
	for n := range p.config.BufferSize {
		p.sampleTimestamp++
		p.sampleCnt += 1.0
		for p.sampleCnt >= p.SamplesPerTick() { // read every time: a tick can change the tempo (a tempo lane)
			p.sampleCnt -= p.SamplesPerTick()
			p.tickTimestamp++
			p.Patch.CtrlTick(CtrlTick{ // counted from 0: the counts less the one just counted
				SampleTimestamp: p.sampleTimestamp - 1,
				TickTimestamp:   p.tickTimestamp - 1,
				SampleOffset:    n,
			})
		}
	}

	p.Patch.Run()
}
