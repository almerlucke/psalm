package psalm

import (
	"math"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"
)

// Env is the environment of a patch: its config (sample rate, buffer size, PPQ), its live tempo
// and its random streams. Every psalm.New creates one and makes it current; the modules created
// after it belong to it and keep a pointer to it. Two patches each have their own environment, so
// their tempo and random numbers don't touch: build patches one after the other, then render them
// in parallel if you like
type Env struct {
	config  Config
	bpm     atomic.Uint64 // float64 bits, the live tempo
	mu      sync.Mutex
	seed    uint64
	streams uint64
	shared  *rand.Rand
	onTempo []func()
}

// randomStream separates psalm's random numbers from other PCG streams with the same seed
const randomStream = 0x2545f4914f6cdd1d

var (
	envMu      sync.Mutex
	currentEnv *Env
	nextSeed   *uint64 // set by Seed, used by the environments created after it
)

// NewEnv creates an environment with a config (not current, see psalm.New)
func NewEnv(config Config) *Env {
	e := &Env{config: config}
	e.bpm.Store(math.Float64bits(config.BPM))
	envMu.Lock()
	seed := uint64(time.Now().UnixNano())
	if nextSeed != nil {
		seed = *nextSeed
	}
	envMu.Unlock()
	e.reseed(seed)
	return e
}

// CurrentEnv returns the environment new modules belong to: the one of the last psalm.New
func CurrentEnv() *Env {
	envMu.Lock()
	defer envMu.Unlock()
	if currentEnv == nil {
		currentEnv = &Env{config: CurrentStackConfig()}
		currentEnv.bpm.Store(math.Float64bits(currentEnv.config.BPM))
		currentEnv.reseed(uint64(time.Now().UnixNano()))
	}
	return currentEnv
}

// SetCurrentEnv makes an environment current, the modules created after it belong to it
func SetCurrentEnv(e *Env) {
	envMu.Lock()
	currentEnv = e
	envMu.Unlock()
}

// Config returns the config with the current tempo
func (e *Env) Config() Config {
	cfg := e.config
	cfg.BPM = e.BPM()
	return cfg
}

// BPM returns the current tempo
func (e *Env) BPM() float64 {
	return math.Float64frombits(e.bpm.Load())
}

// SetBPM changes the tempo of the patch while it plays
func (e *Env) SetBPM(bpm float64) {
	if bpm <= 0 {
		return
	}
	e.bpm.Store(math.Float64bits(bpm))
	e.mu.Lock()
	listeners := e.onTempo
	e.mu.Unlock()
	for _, f := range listeners {
		f()
	}
}

// TickDuration returns the duration of a control tick in seconds at the current tempo
func (e *Env) TickDuration() float64 {
	return 60.0 / (e.BPM() * float64(e.config.PPQ))
}

// OnTempo calls f after every tempo change
func (e *Env) OnTempo(f func()) {
	e.mu.Lock()
	e.onTempo = append(e.onTempo, f)
	e.mu.Unlock()
}

// setConfig replaces the config (from PushConfig and PopConfig)
func (e *Env) setConfig(cfg Config) {
	e.mu.Lock()
	e.config = cfg
	e.mu.Unlock()
	e.SetBPM(cfg.BPM)
}

// Seed seeds the random streams of the environment: modules that take their stream after it (on
// their first random number) repeat exactly with the same seed. Seed before building the patch
func (e *Env) Seed(seed uint64) {
	e.reseed(seed)
}

func (e *Env) reseed(seed uint64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.seed = seed
	e.streams = 0
	e.shared = rand.New(rand.NewPCG(seed, randomStream))
}

// NewRand returns a new random stream, each module takes its own: they don't share state while
// rendering, and with the same seed (and the same patch) every stream repeats
func (e *Env) NewRand() *rand.Rand {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.streams++
	return rand.New(rand.NewPCG(e.seed+e.streams*0x9e3779b97f4a7c15, randomStream^e.streams))
}

// Rand returns the shared random stream of the environment, for code outside modules running on
// the patch goroutine. Modules use their own stream (Module.Rand)
func (e *Env) Rand() *rand.Rand {
	return e.shared
}

// Seed seeds the patches created after it (psalm.New), so a patch built and rendered after the
// same seed sounds exactly the same. Without a seed every run is different. It also seeds the
// current environment while none of its modules took a random stream yet (Seed before building),
// but never a patch that is already built: seeding the next patch doesn't change the one before
func Seed(seed uint64) {
	envMu.Lock()
	s := seed
	nextSeed = &s
	envMu.Unlock()
	e := CurrentEnv()
	e.mu.Lock()
	fresh := e.streams == 0
	e.mu.Unlock()
	if fresh {
		e.Seed(seed)
	}
}

// Rand returns the shared random stream of the current environment (see Env.Rand)
func Rand() *rand.Rand {
	return CurrentEnv().Rand()
}
