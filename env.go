package psalm

import (
	"math"
	"math/rand/v2"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/almerlucke/genny"
)

// Env is the environment of a patch: its config (sample rate, buffer size, PPQ), its live tempo
// and its random streams. Every psalm.New creates one and makes it current; the modules created
// after it belong to it and keep a pointer to it. Two patches each have their own environment, so
// their tempo and random numbers don't touch: build patches one after the other, then render them
// in parallel if you like
type Env struct {
	config  Config
	bpm     atomic.Uint64 // float64 bits, the live tempo
	ppq     atomic.Int64  // the PPQ of the config, read with every tick length
	mu      sync.Mutex
	seed    uint64
	streams uint64
	shared  *rand.Rand
	onTempo []func()
	live    atomic.Bool // the patch plays through an audio engine, in real time (not a render)
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
	e.ppq.Store(int64(config.PPQ))
	envMu.Lock()
	seed := uint64(time.Now().UnixNano())
	if nextSeed != nil {
		seed = *nextSeed
	}
	envMu.Unlock()
	e.reseed(seed)
	return e
}

// scopes holds the environments made current for one goroutine only (Psalm.Within): a song built in
// the background while other goroutines build modules of their own patch
var (
	scopes     sync.Map // goroutine id -> *Env
	scopeCount atomic.Int32
)

// goroutineID returns the id of the goroutine calling (from the first line of its stack trace:
// "goroutine 42 [running]:"); only asked while a goroutine builds within a patch of its own
func goroutineID() uint64 {
	var buf [64]byte
	b := buf[:runtime.Stack(buf[:], false)]
	b = b[len("goroutine "):]
	var id uint64
	for _, c := range b {
		if c < '0' || c > '9' {
			break
		}
		id = id*10 + uint64(c-'0')
	}
	return id
}

// within makes an environment current for the goroutine calling while f runs
func within(e *Env, f func()) {
	id := goroutineID()
	prev, had := scopes.Load(id)
	scopes.Store(id, e)
	scopeCount.Add(1)
	defer func() {
		if had {
			scopes.Store(id, prev)
		} else {
			scopes.Delete(id)
		}
		scopeCount.Add(-1)
	}()
	f()
}

// CurrentEnv returns the environment new modules belong to: the one of the patch the goroutine
// builds within (Psalm.Within), else the one of the last psalm.New
func CurrentEnv() *Env {
	if scopeCount.Load() > 0 {
		if e, ok := scopes.Load(goroutineID()); ok {
			return e.(*Env)
		}
	}
	envMu.Lock()
	defer envMu.Unlock()
	if currentEnv == nil {
		currentEnv = &Env{config: CurrentStackConfig()}
		currentEnv.bpm.Store(math.Float64bits(currentEnv.config.BPM))
		currentEnv.ppq.Store(int64(currentEnv.config.PPQ))
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

// Live says whether the patch plays in real time, through an audio engine: what runs on the audio thread has a deadline.
// A render (psalm.New(nil)) has none: it takes the time it takes
func (e *Env) Live() bool { return e.live.Load() }

// Config returns the config with the current tempo
func (e *Env) Config() Config {
	e.mu.Lock()
	cfg := e.config
	e.mu.Unlock()
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
	return 60.0 / (e.BPM() * float64(e.ppq.Load()))
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
	e.ppq.Store(int64(cfg.PPQ))
	e.mu.Unlock()
	e.SetBPM(cfg.BPM)
}

// Seed seeds the random streams of the environment: modules that take their stream after it (on
// their first random number) repeat exactly with the same seed. Seed before building the patch
func (e *Env) Seed(seed uint64) {
	e.reseed(seed)
	genny.Seed(seed) // the generators (genny) draw from one stream of their own
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
	genny.Seed(seed) // the generators (genny) too: scripts, sequences and walks repeat with it
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
