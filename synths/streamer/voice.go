package streamer

import (
	"math"
	"sync/atomic"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/modules/poly"
	"github.com/almerlucke/sndfile"
)

const (
	ringFrames = 1 << 16 // frames the reader can be ahead of the sound (a second and a half at 44.1 kHz)
	chunk      = 4096    // frames the reader reads at a time
	genBits    = 40      // the word the reader tells how far it is in: the note's number above, the frames below
	fadeIn     = 0.002   // seconds
)

// Config is a streamer playing an instrument
type Config struct {
	NumChannels int
	Instrument  *Instrument
	Level       float64 // output level (0 is treated as 1)
}

// DefaultConfig plays an instrument in stereo
func DefaultConfig(inst *Instrument) Config {
	return Config{NumChannels: psalm.Stereo, Instrument: inst, Level: 1}
}

// Voice plays one sound at a time from disk. A note makes a goroutine that reads the file on from the head in memory into a ring; the audio thread only reads
// the ring, and tells the reader how far it has got through atomic words
type Voice struct {
	*psalm.Module
	cfg      Config
	pitch    float64
	ring     []float32 // stereo frames, filled by the reader of the note
	fill     atomic.Uint64
	used     atomic.Int64  // the frame the sound is at: the reader keeps ahead of it
	curGen   atomic.Uint64 // the note the reader should serve; a reader of another note stops
	gen      uint64
	last     chan struct{} // closed when the reader of the last note has stopped
	underrun atomic.Int64

	e        *Entry
	active   bool
	released bool
	pos      float64 // frame of the sound (from the start of the part played)
	rate     float64
	total    int64 // frames the sound lasts, -1 for a loop
	amp      float64
	gl, gr   float32
	fade     float64 // 0 to 1 while the sound fades in, then 1
	fadeStep float64
	relGain  float64
	relStep  float64
	wait     int
	noteOff  int
}

var _ poly.Voice = (*Voice)(nil)

// New returns a voice; NewPoly a playable streamer
func New(cfg Config) *Voice {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Stereo
	}
	if cfg.Level <= 0 {
		cfg.Level = 1
	}
	v := &Voice{Module: psalm.NewModule(nil, []int{cfg.NumChannels}), cfg: cfg, pitch: 60}
	v.SetSelf(v)
	return v
}

// NewPoly returns a streamer of numVoices voices
func NewPoly(cfg Config, numVoices int) *poly.Poly[*Voice] {
	if cfg.NumChannels == 0 {
		cfg.NumChannels = psalm.Stereo
	}
	voices := make([]*Voice, numVoices)
	for i := range voices {
		voices[i] = New(cfg)
	}
	return poly.New[*Voice](cfg.NumChannels, voices)
}

// Underruns is how many times the reader was too slow and the sound had a gap
func (v *Voice) Underruns() int64 { return v.underrun.Load() }

func (v *Voice) ReceiveMessage(msg psalm.Message) {
	if inst, ok := msg["instrument"].(*Instrument); ok && inst != nil {
		v.cfg.Instrument = inst
	}
	msg.IfFltVal("pitch", func(p float64) { v.pitch = p })
	msg.IfFltVal("frequency", func(f float64) {
		if f > 0 {
			v.pitch = 69 + 12*math.Log2(f/440)
		}
	})
	msg.IfFltVal("level", func(f float64) { v.cfg.Level = f })
}

func (v *Voice) Receive(value any, _ int) {
	if msg, ok := psalm.MsgVal(value); ok {
		v.ReceiveMessage(msg)
	}
}

func (v *Voice) NoteOn(amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.note(0, amplitude, message, tick)
}

func (v *Voice) Note(duration float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick, _ psalm.Config) {
	v.note(duration, amplitude, message, tick)
}

func (v *Voice) note(duration, amplitude float64, message psalm.Message, tick psalm.CtrlTick) {
	v.ReceiveMessage(message)
	v.stop()
	inst := v.cfg.Instrument
	if inst == nil {
		return
	}
	e := inst.Lookup(int(math.Round(v.pitch)))
	if e == nil {
		return
	}
	sr := v.Config().SampleRate
	v.e = e
	v.amp = amplitude * v.cfg.Level * math.Pow(10, e.Volume/20)
	// the pan of a mono sound is a constant power pan; of a stereo sound a balance
	pan := e.Pan
	if e.channels == 1 {
		a := (pan + 1) * math.Pi / 4
		v.gl, v.gr = float32(math.Cos(a)*math.Sqrt2), float32(math.Sin(a)*math.Sqrt2)
	} else {
		v.gl, v.gr = float32(min(1, 1-pan)), float32(min(1, 1+pan))
	}
	v.pos = 0
	v.rate = e.sr / sr * e.Speed
	v.total = e.endF - e.startF
	if e.Loop {
		v.total = -1
	}
	v.fade, v.fadeStep = 0, 1/max(1, fadeIn*sr)
	v.relGain, v.relStep = 1, 0
	v.released = false
	v.wait = tick.SampleOffset
	v.noteOff = 0
	if duration > 0 {
		v.noteOff = max(1, int(duration*sr))
	}
	v.active = true
	// the reader of the note: after the one before it has stopped (they share the ring)
	if v.ring == nil {
		v.ring = make([]float32, 2*ringFrames)
	}
	v.gen = (v.gen + 1) & (1<<(64-genBits) - 1)
	v.used.Store(0)
	v.fill.Store(v.gen << genBits)
	v.curGen.Store(v.gen)
	after, done := v.last, make(chan struct{})
	v.last = done
	if int64(len(e.head)/2) < v.total || v.total < 0 {
		go v.read(e, v.gen, after, done)
	} else {
		close(done) // the whole sound is in memory
	}
}

// stop lets the reader of the note go
func (v *Voice) stop() {
	v.active = false
	v.gen = (v.gen + 1) & (1<<(64-genBits) - 1)
	v.curGen.Store(v.gen)
}

func (v *Voice) NoteOff() {
	if v.e == nil || !v.active || v.released || !(v.e.Loop || v.e.Hold) {
		return
	}
	v.released = true
	v.relStep = 1 / max(1, v.e.Release*v.Config().SampleRate)
}

func (v *Voice) Clear() {
	v.stop()
	v.Output(0).Clear()
}

func (v *Voice) IsActive() bool { return v.active }

// read is the goroutine of a note: it reads the file on from where the head ends into the ring, as far ahead of the sound as the ring allows
func (v *Voice) read(e *Entry, gen uint64, after <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	if after != nil {
		<-after
	}
	stale := func() bool { return v.curGen.Load() != gen }
	if stale() {
		return
	}
	f, err := sndfile.Open[float32](e.Path)
	if err != nil {
		v.underrun.Add(1)
		return
	}
	defer func() { _ = f.Close() }()
	headLen := int64(len(e.head) / 2)
	limit := e.endF
	if e.Loop {
		limit = e.loopEndF
	}
	pos := e.startF + headLen
	if pos >= limit && e.Loop {
		pos = e.loopStartF
	}
	if err := f.SeekFrame(pos); err != nil {
		v.underrun.Add(1)
		return
	}
	buf := make([]float32, chunk*e.channels)
	var written int64
	for !stale() {
		if written-max(0, v.used.Load()-headLen) > ringFrames-chunk-4 {
			time.Sleep(4 * time.Millisecond) // the ring is full
			continue
		}
		if pos >= limit {
			if !e.Loop {
				return
			}
			pos = e.loopStartF
			if err := f.SeekFrame(pos); err != nil {
				return
			}
		}
		n := min(chunk, limit-pos)
		got, err := f.ReadFrames(buf[:n*int64(e.channels)])
		if err != nil || got == 0 {
			pos = limit // the file is shorter than it said
			continue
		}
		for i := range got {
			l := buf[i*int64(e.channels)]
			r := l
			if e.channels > 1 {
				r = buf[i*int64(e.channels)+1]
			}
			at := ((written + i) % ringFrames) * 2
			v.ring[at], v.ring[at+1] = l, r
		}
		written += got
		pos += got
		for { // tell the audio thread, unless the note is no longer this one
			old := v.fill.Load()
			if old>>genBits != gen || v.fill.CompareAndSwap(old, gen<<genBits|uint64(written)) {
				break
			}
		}
	}
}

func (v *Voice) Synthesize() {
	out := v.Output(0)
	out.Clear()
	if !v.active {
		return
	}
	e := v.e
	chans := out.Channels
	headLen := int64(len(e.head) / 2)
	var avail int64 // frames of the ring read in
	if w := v.fill.Load(); w>>genBits == v.gen {
		avail = int64(w & (1<<genBits - 1))
	}
	var gap bool
	// frame is the sound at a frame: the head in memory, then what the reader has read
	frame := func(i int64) (float32, float32) {
		if i < 0 || (v.total >= 0 && i >= v.total) {
			return 0, 0
		}
		if i < headLen {
			return e.head[2*i], e.head[2*i+1]
		}
		k := i - headLen
		if k >= avail {
			gap = true
			return 0, 0
		}
		at := (k % ringFrames) * 2
		return v.ring[at], v.ring[at+1]
	}
	n := v.Config().BufferSize
	amp := float32(v.amp)
	for s := v.wait; s < n; s++ {
		i := int64(v.pos)
		fr := float32(v.pos - float64(i))
		l0, r0 := frame(i)
		l1, r1 := l0, r0
		if fr > 0 {
			l1, r1 = frame(i + 1)
		}
		g := amp * float32(v.fade) * float32(v.relGain)
		l, r := (l0+(l1-l0)*fr)*g*v.gl, (r0+(r1-r0)*fr)*g*v.gr
		if len(chans) == 1 {
			chans[0][s] = (l + r) * 0.5
		} else {
			chans[0][s], chans[1][s] = l, r
		}
		if v.fade < 1 {
			v.fade = min(1, v.fade+v.fadeStep)
		}
		if v.relStep > 0 {
			if v.relGain -= v.relStep; v.relGain <= 0 {
				v.active = false
				break
			}
		}
		if v.noteOff > 0 {
			if v.noteOff--; v.noteOff == 0 {
				v.NoteOff()
			}
		}
		v.pos += v.rate
		if v.total >= 0 && v.pos >= float64(v.total) {
			v.active = false
			break
		}
	}
	v.wait = 0
	if gap {
		v.underrun.Add(1)
	}
	v.used.Store(int64(v.pos))
	if !v.active {
		v.curGen.Store(0) // the reader has nothing more to do
	}
}
