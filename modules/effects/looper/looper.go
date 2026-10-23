// Package looper is a live looper for audio, like a looper pedal: the sound that goes through it is recorded into a loop of a number of bars (4/4) on
// the beat of the patch, and the loop plays on, pass after pass, over the sound that comes in. Recording begins at the start of the next pass and ends
// at the end of a pass, and what was recorded is a layer: layers play together, and a new one can be recorded on top of them as long as you like. A layer
// can be taken away by itself, or the last one (undo), or all of them (clear), and the layers are written to WAV files (export). The loop follows the
// control ticks of the patch, so what is played in time is in time on every pass; the length of a pass is that of the tempo when the layer began, and a
// change of the tempo makes a layer play at the old speed against the beat (clear, and record again).
package looper

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/blocks/mix"
	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/sndfile/writer"
)

// Columns is how many columns of peaks a layer is shown with
const Columns = 240

// Config describes a looper
type Config struct {
	Bars   int     // the bars of the loop, 1 to 16
	Level  float64 // how loud the loop is, 0 to 1.5
	Dry    float64 // how loud the sound that comes in is on the way through, 0 to 1
	Early  float64 // milliseconds a layer is played earlier than it was recorded, 0 to 250: the sound card is late (what you play is heard and recorded after the beat)
	Record bool    // a layer is recorded from the next pass; off, it ends with the pass
	Play   bool    // the layers play
	Files  int     // export: 0 a file for every layer, 1 one file with the layers together
}

// Files are the ways the layers are exported
var Files = []string{"one per layer", "one file"}

// DefaultConfig is a loop of two bars
func DefaultConfig() Config { return Config{Bars: 2, Level: 1, Dry: 1, Play: true} }

// Describe describes the settings of a looper (see params); undo, clear and export are switches that let go by themselves
func Describe(cfg Config) params.Set {
	return params.Set{Name: "looper", Params: []params.Param{
		params.K("Loop", "bars", 1, 16, float64(cfg.Bars), params.Step(1)),
		params.S("Loop", "record", cfg.Record),
		params.S("Loop", "play", cfg.Play),
		params.K("Mix", "level", 0, 1.5, cfg.Level, params.Label("loop level")),
		params.K("Mix", "dry", 0, 1, cfg.Dry, params.Label("input level")),
		params.K("Mix", "early", 0, 250, cfg.Early, params.Unit("ms"), params.Label("play layers earlier")),
		params.S("Layers", "undo", false),
		params.S("Layers", "clear", false),
		params.S("Export", "export", false, params.Label("export to WAV")),
		params.C("Export", "files", Files, cfg.Files, params.Label("export files")),
	}}
}

// layer is a recording of a pass: it is never changed after it is made
type layer struct {
	id    int
	audio [2][]float32
	peaks atomic.Pointer[[]float32]
}

// Looper is the module: input 0 is stereo audio, which goes on to output 0 together with the loop
type Looper struct {
	*psalm.Module
	cfg Config

	// the audio thread
	layers    []*layer
	cur       *layer // the layer being recorded
	recording bool
	synced    bool // a pass has begun: the position in the loop is known
	pos       int
	wrapAt    int // the sample of the buffer a pass begins at, -1 for none
	nextID    int
	undo      bool
	clear     bool
	export    bool
	level     mix.Smoother
	dry       mix.Smoother
	spare     chan [2][]float32 // a buffer to record in, made ahead
	col       int
	colMax    float32

	// what the page shows and asks for
	snap  atomic.Pointer[[]*layer]
	where atomic.Int64 // pos
	span  atomic.Int64 // samples in a pass as the page counts them
	armed atomic.Bool
	live  atomic.Bool
	bars  atomic.Int64
	curPk [Columns]atomic.Uint32 // the peaks of the layer being recorded, float32 bits
	mu    sync.Mutex
	drops []int

	dir      string
	log      func(string)
	released func(key string)
}

// New creates a looper
func New(cfg Config) *Looper {
	if cfg.Bars <= 0 {
		cfg.Bars = 2
	}
	cfg.Bars = min(cfg.Bars, 16)
	sr := psalm.SampleRate()
	l := &Looper{Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}), cfg: cfg, wrapAt: -1, dir: "recordings",
		level: mix.NewSmoother(cfg.Level, mix.SmoothMs, sr), dry: mix.NewSmoother(cfg.Dry, mix.SmoothMs, sr), spare: make(chan [2][]float32, 1)}
	l.bars.Store(int64(cfg.Bars))
	l.SetSelf(l)
	return l
}

// Settings returns the settings
func (l *Looper) Settings() Config { return l.cfg }

// OnEvent sets where the looper tells what it does (the log of the patch)
func (l *Looper) OnEvent(f func(string)) { l.log = f }

// OnRelease sets what is called when a switch that lets go by itself (undo, clear, export) has: the page shows it off again
func (l *Looper) OnRelease(f func(key string)) { l.released = f }

// SetDir sets the folder the WAV files are written in
func (l *Looper) SetDir(dir string) { l.dir = dir }

func (l *Looper) say(format string, args ...any) {
	if l.log != nil {
		l.log(fmt.Sprintf(format, args...))
	}
}

func (l *Looper) let(key string) {
	if l.released != nil {
		l.released(key)
	}
}

func (l *Looper) Receive(value any, _ int) {
	if msg, ok := psalm.MsgVal(value); ok {
		l.ReceiveMessage(msg)
	}
}

func (l *Looper) ReceiveMessage(msg psalm.Message) {
	msg.IfIntVal("bars", func(b int) { l.cfg.Bars = max(1, min(b, 16)); l.bars.Store(int64(l.cfg.Bars)) })
	msg.IfFltVal("level", func(v float64) { l.cfg.Level = max(0, min(v, 1.5)) })
	msg.IfFltVal("dry", func(v float64) { l.cfg.Dry = max(0, min(v, 1)) })
	msg.IfFltVal("early", func(v float64) { l.cfg.Early = max(0, min(v, 250)) })
	msg.IfFltVal("record", func(v float64) {
		l.cfg.Record = v >= 0.5
		if l.cfg.Record {
			l.prepare()
		}
	})
	msg.IfIntVal("files", func(f int) { l.cfg.Files = max(0, min(f, 1)) })
	msg.IfFltVal("play", func(v float64) { l.cfg.Play = v >= 0.5 })
	msg.IfFltVal("undo", func(v float64) { l.undo = v >= 0.5 })
	msg.IfFltVal("clear", func(v float64) { l.clear = v >= 0.5 })
	msg.IfFltVal("export", func(v float64) { l.export = v >= 0.5 })
}

// ticks is the length of a pass in ticks
func (l *Looper) ticks() int64 { return int64(l.cfg.Bars) * 4 * int64(l.Config().PPQ) }

// samples is the length of a pass in samples at the tempo now, with a few to spare
func (l *Looper) samples() int {
	return int(math.Ceil(float64(l.ticks())*l.Config().SampleRate*60/(l.BPM()*float64(l.Config().PPQ)))) + 4
}

// prepare makes the buffer of the next layer ahead, off the audio thread
func (l *Looper) prepare() {
	if len(l.spare) > 0 {
		return
	}
	n := l.samples()
	go func() {
		select {
		case l.spare <- [2][]float32{make([]float32, n), make([]float32, n)}:
		default:
		}
	}()
}

// CtrlTick notes the tick a pass begins at (and where in the buffer it is), for the next buffer
func (l *Looper) CtrlTick(tick psalm.CtrlTick) {
	if tick.TickTimestamp%l.ticks() == 0 {
		l.wrapAt = tick.SampleOffset
	}
}

// buffer takes the buffer of a layer: the one made ahead when it is long enough
func (l *Looper) buffer() [2][]float32 {
	n := l.samples()
	var b [2][]float32
	select {
	case b = <-l.spare:
	default:
	}
	if len(b[0]) < n {
		b = [2][]float32{make([]float32, n), make([]float32, n)}
	} else {
		clear(b[0])
		clear(b[1])
	}
	return b
}

// startPass is the beginning of a pass: a layer that was being recorded is done, and the next one begins when it is asked for
func (l *Looper) startPass() {
	if l.recording {
		cur := l.cur
		for c := range cur.audio {
			cur.audio[c] = cur.audio[c][:min(l.pos, len(cur.audio[c]))]
		}
		l.cur, l.recording = nil, false
		if l.pos > 0 {
			l.layers = append(slices.Clone(l.layers), cur)
			l.publish()
			go l.peaksOf(cur)
			l.say("layer %d: %.1f seconds", len(l.layers), float64(l.pos)/l.Config().SampleRate)
		}
	}
	l.pos, l.synced = 0, true
	l.span.Store(int64(l.samples() - 4))
	if l.cfg.Record {
		l.cur = &layer{id: l.nextID, audio: l.buffer()}
		l.nextID++
		l.recording = true
		for i := range l.curPk {
			l.curPk[i].Store(0)
		}
		l.col, l.colMax = 0, 0
		l.say("recording layer %d", len(l.layers)+1)
		l.prepare()
	}
	l.live.Store(l.recording)
}

// publish makes the list of layers the page and the export read
func (l *Looper) publish() { s := slices.Clone(l.layers); l.snap.Store(&s) }

// peaksOf measures a layer for the page (a goroutine of its own)
func (l *Looper) peaksOf(ly *layer) {
	pk := make([]float32, Columns)
	n := len(ly.audio[0])
	for c := range ly.audio {
		for i, v := range ly.audio[c] {
			col := min(Columns-1, i*Columns/max(1, n))
			pk[col] = max(pk[col], float32(math.Abs(float64(v))))
		}
	}
	ly.peaks.Store(&pk)
}

// Drop asks for a layer to be taken away, by its number (done in the next buffer)
func (l *Looper) Drop(id int) {
	l.mu.Lock()
	l.drops = append(l.drops, id)
	l.mu.Unlock()
}

func (l *Looper) Synthesize() {
	l.armed.Store(l.cfg.Record && !l.recording)
	l.mu.Lock()
	drops := l.drops
	l.drops = nil
	l.mu.Unlock()
	for _, id := range drops {
		if i := slices.IndexFunc(l.layers, func(ly *layer) bool { return ly.id == id }); i >= 0 {
			l.layers = slices.Delete(slices.Clone(l.layers), i, i+1)
			l.publish()
			l.say("layer %d taken away: %d layers left", i+1, len(l.layers))
		}
	}
	if l.undo {
		l.undo = false
		l.let("undo")
		if n := len(l.layers); n > 0 {
			l.layers = slices.Clone(l.layers[:n-1])
			l.publish()
			l.say("undo: %d layers left", len(l.layers))
		}
	}
	if l.clear {
		l.clear = false
		l.let("clear")
		l.layers = nil
		l.publish()
		l.say("cleared")
	}
	if l.export {
		l.export = false
		l.let("export")
		l.exportWAV()
	}

	in, out := l.Input(0).Channels, l.Output(0).Channels
	play := l.cfg.Play
	early := int(l.cfg.Early * l.Config().SampleRate / 1000)
	for i := range out[0] {
		if i == l.wrapAt {
			l.startPass()
		}
		dry := float32(l.dry.Next(l.cfg.Dry))
		lv := float32(l.level.Next(l.cfg.Level))
		var loop [2]float32
		if l.synced {
			if play {
				for _, ly := range l.layers {
					if n := len(ly.audio[0]); l.pos < n {
						at := l.pos + early // the end of the layer comes round to its beginning: what was late is where it was meant to be
						if at >= n {
							at -= n
						}
						loop[0] += ly.audio[0][at]
						loop[1] += ly.audio[1][at]
					}
				}
			}
			if l.recording && l.pos < len(l.cur.audio[0]) {
				a, b := in[0][i], in[min(1, len(in)-1)][i]
				l.cur.audio[0][l.pos], l.cur.audio[1][l.pos] = a, b
				col := min(Columns-1, l.pos*Columns/max(1, int(l.span.Load())))
				if col != l.col {
					l.flush()
					l.col = col
				}
				l.colMax = max(l.colMax, abs32(a), abs32(b))
			}
			l.pos++
		}
		for c := range out {
			out[c][i] = in[min(c, len(in)-1)][i]*dry + loop[c]*lv
		}
	}
	l.flush()
	l.wrapAt = -1
	l.where.Store(int64(l.pos))
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// flush hands the peak of the column being recorded to the page
func (l *Looper) flush() {
	if l.colMax > math.Float32frombits(l.curPk[l.col].Load()) {
		l.curPk[l.col].Store(math.Float32bits(l.colMax))
	}
	l.colMax = 0
}

// Layers says how many layers there are and whether one is being recorded
func (l *Looper) Layers() (int, bool) {
	if s := l.snap.Load(); s != nil {
		return len(*s), l.live.Load()
	}
	return 0, l.live.Load()
}

// LayerView is a layer as the page shows it: its number and its peaks (nothing until they are measured)
type LayerView struct {
	ID    int       `json:"id"`
	Peaks []float32 `json:"peaks"`
}

// State is what the page gets as the loop plays: the position is in beats, like the looper of notes
type State struct {
	Layers    []LayerView `json:"layers"`
	CurPeaks  []float32   `json:"curPeaks"`
	Recording bool        `json:"recording"`
	Bars      int         `json:"bars"`
	Pos       float64     `json:"pos"`
	BPS       float64     `json:"bps"`
	Armed     bool        `json:"armed"`
}

// State returns what the page shows (from any goroutine)
func (l *Looper) State() any {
	st := State{Layers: []LayerView{}, CurPeaks: []float32{}, Recording: l.live.Load(), Bars: int(l.bars.Load()), BPS: l.BPM() / 60, Armed: l.armed.Load()}
	if span := l.span.Load(); span > 0 {
		st.Pos = math.Round(float64(l.where.Load())/float64(span)*float64(st.Bars)*4*1000) / 1000
	}
	if s := l.snap.Load(); s != nil {
		for _, ly := range *s {
			v := LayerView{ID: ly.id, Peaks: []float32{}}
			if pk := ly.peaks.Load(); pk != nil {
				v.Peaks = *pk
			}
			st.Layers = append(st.Layers, v)
		}
	}
	if st.Recording {
		for i := range l.curPk {
			st.CurPeaks = append(st.CurPeaks, math.Float32frombits(l.curPk[i].Load()))
		}
	}
	return st
}

// exportWAV writes every layer to a file in the folder, loop-<time>-layer<n>.wav, or (the setting "files") the layers together in one, loop-<time>.wav
// (from a goroutine of its own): each is one pass, so it loops
func (l *Looper) exportWAV() {
	s := l.snap.Load()
	if s == nil || len(*s) == 0 {
		l.say("nothing to export: no layers yet")
		return
	}
	layers, sr, say, dir := *s, l.Config().SampleRate, l.say, l.dir
	early, one := int(l.cfg.Early*sr/1000), l.cfg.Files == 1
	go func() {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			say("error: can't make %s: %v", dir, err)
			return
		}
		stamp := time.Now().Format("20060102-150405")
		rotate := func(audio [2][]float32) [2][]float32 { // as it is played: earlier
			var out [2][]float32
			for c := range out {
				n := len(audio[c])
				out[c] = make([]float32, n)
				for i := range out[c] {
					out[c][i] = audio[c][(i+early)%max(1, n)]
				}
			}
			return out
		}
		write := func(name string, audio [2][]float32) bool {
			path := filepath.Join(dir, name)
			w, err := writer.New(path, writer.WAV, 2, sr, writer.NewChannelConverter[float32](len(audio[0]), 2))
			if err == nil {
				err = w.Write([][]float32{audio[0], audio[1]}, false)
				if cerr := w.Close(); err == nil {
					err = cerr
				}
			}
			if err != nil {
				say("error: can't write %s: %v", path, err)
				return false
			}
			say("saved %s (%.1f s)", path, float64(len(audio[0]))/sr)
			return true
		}
		longest := 0
		for i, ly := range layers {
			if !one && !write(fmt.Sprintf("loop-%s-layer%d.wav", stamp, i+1), rotate(ly.audio)) {
				return
			}
			longest = max(longest, len(ly.audio[0]))
		}
		if one {
			var all [2][]float32
			for c := range all {
				all[c] = make([]float32, longest)
				for _, ly := range layers {
					r := rotate(ly.audio)
					for i, v := range r[c] {
						all[c][i] += v
					}
				}
			}
			write(fmt.Sprintf("loop-%s.wav", stamp), all)
		}
	}()
}
