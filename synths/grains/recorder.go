package grains

import (
	"math"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/almerlucke/psalm"
)

// Recorder listens to the sound that comes in and keeps the last seconds of it, for grains to read: a granular synth that plays
// what it hears (a voice, an instrument, another synth) as it is played. It passes the sound on, so it can sit in a chain.
// The recorder of a patch is given to a voice as its source: cfg.Source = rec.Source()
//
// The position of the grains is how far back they look: 0 is just now, 1 is the oldest second there is. A grain that reads
// faster than the sound is written is kept behind the writing, so it never reads what is not there yet
type Recorder struct {
	*psalm.Module
	ring   []float64
	head   int // where the next frame is written
	count  int
	freeze atomic.Bool
	name   atomic.Pointer[string]
}

// recorders are the recorders of the patch by name, for a text that names the sound a voice plays (RecorderNamed)
var recorders sync.Map

// RecorderNamed is the recorder that has a name (the last one that took it), nil for none
func RecorderNamed(name string) *Recorder {
	if r, ok := recorders.Load(strings.TrimSpace(name)); ok {
		return r.(*Recorder)
	}
	return nil
}

// DefaultRecorderName is the name a recorder has until it is given another
const DefaultRecorderName = "live"

// SetName gives the recorder a name, so a voice can be told to listen to it (see RecorderNamed)
func (r *Recorder) SetName(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	if old := r.name.Load(); old != nil && *old != name {
		recorders.CompareAndDelete(*old, r)
	}
	r.name.Store(&name)
	recorders.Store(name, r)
}

// NewRecorder makes a recorder that keeps seconds of sound (of a stereo input: its two channels mixed)
func NewRecorder(seconds float64) *Recorder {
	n := max(64, int(math.Round(seconds*psalm.SampleRate())))
	r := &Recorder{
		Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}),
		ring:   make([]float64, n),
	}
	r.SetSelf(r)
	r.SetName(DefaultRecorderName)
	return r
}

// Source is the recorder as a source of grains
func (r *Recorder) Source() Source { return &liveSource{r: r} }

// Freeze stops (true) or goes on (false) with writing: the grains then play what was heard until now, a held moment
func (r *Recorder) Freeze(on bool) { r.freeze.Store(on) }

func (r *Recorder) Receive(value any, index int) {
	if msg, ok := psalm.MsgVal(value); ok {
		r.ReceiveMessage(msg)
	}
}

func (r *Recorder) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("freeze", func(f float64) { r.freeze.Store(f > 0.5) })
	msg.IfStrVal("name", r.SetName)
}

func (r *Recorder) Synthesize() {
	ins := r.Input(0).Channels
	outs := r.Output(0).Channels
	n := r.Config().BufferSize
	for c := range outs {
		copy(outs[c][:n], ins[c][:n])
	}
	if r.freeze.Load() {
		return
	}
	for i := range n {
		x := 0.0
		for c := range ins {
			x += float64(ins[c][i])
		}
		r.ring[r.head] = x / float64(len(ins))
		r.head++
		if r.head == len(r.ring) {
			r.head = 0
		}
	}
	r.count = min(len(r.ring), r.count+n)
}

// Tail: the recorder has no sound of its own
func (r *Recorder) Tail() float64 { return 0 }

// LiveSource is the source of the recorder that has a name, found when the grains need it: the recorder may be made later than the
// voice that listens to it (a patch loads its modules one after the other), or replaced. Until there is one the source is silent
func LiveSource(name string) Source { return &namedSource{name: strings.TrimSpace(name)} }

// namedSource asks the recorder of a name for its source every time the recorder it found is not the one with the name any more
type namedSource struct {
	name string
	r    *Recorder
	src  *liveSource
}

func (s *namedSource) get() *liveSource {
	if r := RecorderNamed(s.name); r != s.r {
		s.r = r
		s.src = nil
		if r != nil {
			s.src = &liveSource{r: r}
		}
	}
	return s.src
}

func (s *namedSource) Rate() float64     { return psalm.SampleRate() }
func (s *namedSource) Depth() int        { return 1 }
func (s *namedSource) Loudness() float64 { return 0 }
func (s *namedSource) Ready() bool       { l := s.get(); return l != nil && l.Ready() }
func (s *namedSource) Span() float64 {
	if l := s.get(); l != nil {
		return l.Span()
	}
	return 1
}
func (s *namedSource) Locate(position, scanned, spray, lead float64) (float64, float64, float64) {
	return s.get().Locate(position, scanned, spray, lead)
}
func (s *namedSource) Read(dst []float64, pos, inc float64, depth int, lo, hi float64) float64 {
	return s.get().Read(dst, pos, inc, depth, lo, hi)
}

// liveSource reads the ring of a recorder
type liveSource struct{ r *Recorder }

func (s *liveSource) Rate() float64     { return psalm.SampleRate() }
func (s *liveSource) Span() float64     { return float64(len(s.r.ring)) }
func (s *liveSource) Depth() int        { return 1 }
func (s *liveSource) Loudness() float64 { return 0 }
func (s *liveSource) Ready() bool       { return s.r.count > 2 }

func (s *liveSource) Locate(position, scanned, spray, lead float64) (float64, float64, float64) {
	n := float64(s.r.count)
	// how far behind the writing: position of the grain's look back, moved by the scan, never so near that the grain gets ahead of the writing
	back := max(0.0, min(position, 1.0))*(n-1) + scanned + spray
	back = max(lead+1, min(back, n-1))
	return wrap(float64(s.r.head)-back, float64(len(s.r.ring))), 0, float64(len(s.r.ring))
}

func (s *liveSource) Read(dst []float64, pos, inc float64, _ int, _, hi float64) float64 {
	ring := s.r.ring
	n := len(ring)
	for i := range dst {
		k := int(pos)
		f := pos - float64(k)
		a := ring[k]
		b := ring[(k+1)%n]
		dst[i] = a + (b-a)*f
		pos += inc
		pos -= math.Floor(pos/hi) * hi
	}
	return pos
}
