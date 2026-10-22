// Package recorder records the sound that goes through it to a WAV file (32 bit float, stereo) while its record switch is on: put it
// on the master to record the whole piece, or on a voice to record it dry. The sound goes on to its output as it is. The file
// is written next to the playing, on a goroutine of its own, so the audio thread never waits for the disk; it is named after the
// module and the time, and the function set with OnSaved says when it is closed.
package recorder

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/sndfile/writer"
)

// Config describes a recorder
type Config struct {
	Record bool   // recording
	Dir    string // the folder of the files, "" is "recordings"
	Name   string // the first part of the name of the files, "" is "recording"
}

// Describe describes the settings of a recorder set as cfg (see params)
func Describe(cfg Config) params.Set {
	return params.Set{Name: "recorder", Params: []params.Param{params.S("Recorder", "record", cfg.Record)}}
}

// Recorder is the module: input 0 is stereo audio, which also goes to output 0
type Recorder struct {
	*psalm.Module
	cfg    Config
	want   atomic.Bool
	frames chan [2][]float32 // the buffers of the recording going on, nil when not recording (on the audio thread)
	drops  *atomic.Int64
	onFile func(path string, err error)
	onDone func(path string, seconds float64, dropped int64, err error)
}

// New creates a recorder
func New(cfg Config) *Recorder {
	r := &Recorder{Module: psalm.NewModule([]int{psalm.Stereo}, []int{psalm.Stereo}), cfg: cfg}
	if r.cfg.Dir == "" {
		r.cfg.Dir = "recordings"
	}
	if r.cfg.Name == "" {
		r.cfg.Name = "recording"
	}
	r.want.Store(cfg.Record)
	r.SetSelf(r)
	return r
}

// Settings returns the settings
func (r *Recorder) Settings() Config {
	c := r.cfg
	c.Record = r.want.Load()
	return c
}

// SetRecord starts or stops the recording (at the next buffer)
func (r *Recorder) SetRecord(on bool) { r.want.Store(on) }

// OnStarted is called, from the goroutine of the recording, when a file is made (or could not be)
func (r *Recorder) OnStarted(f func(path string, err error)) { r.onFile = f }

// OnSaved is called, from the goroutine of the recording, when the file is closed: its path, the seconds in it, how many buffers
// were lost (the disk could not keep up) and the error
func (r *Recorder) OnSaved(f func(path string, seconds float64, dropped int64, err error)) {
	r.onDone = f
}

func (r *Recorder) Receive(value any, _ int) {
	if msg, ok := psalm.MsgVal(value); ok {
		r.ReceiveMessage(msg)
	}
}

func (r *Recorder) ReceiveMessage(msg psalm.Message) {
	msg.IfFltVal("record", func(v float64) { r.SetRecord(v >= 0.5) })
}

func (r *Recorder) Synthesize() {
	in, out := r.Input(0).Channels, r.Output(0).Channels
	for c := range out {
		copy(out[c], in[min(c, len(in)-1)])
	}
	want := r.want.Load()
	switch {
	case want && r.frames == nil:
		r.frames = make(chan [2][]float32, 8192) // about 47 seconds of buffers of 256, for a slow disk
		r.drops = new(atomic.Int64)
		go r.write(r.frames, r.drops, r.Config().SampleRate, r.Config().BufferSize)
	case !want && r.frames != nil:
		close(r.frames)
		r.frames = nil
	}
	if r.frames == nil {
		return
	}
	var f [2][]float32
	for c := range f {
		f[c] = append([]float32(nil), out[c]...)
	}
	select {
	case r.frames <- f:
	default:
		r.drops.Add(1)
	}
}

// write is the goroutine of a recording: it makes the file, writes the buffers that arrive and closes the file when they stop
func (r *Recorder) write(frames <-chan [2][]float32, drops *atomic.Int64, sr float64, bs int) {
	path := filepath.Join(r.cfg.Dir, fmt.Sprintf("%s-%s.wav", r.cfg.Name, time.Now().Format("20060102-150405")))
	var w *writer.Writer
	err := os.MkdirAll(r.cfg.Dir, 0o755)
	if err == nil {
		w, err = writer.New(path, writer.WAV, 2, sr, writer.NewChannelConverter[float32](bs, 2))
	}
	if r.onFile != nil {
		r.onFile(path, err)
	}
	n := 0
	for f := range frames { // always drained, so the audio thread never blocks, even when the file could not be made
		if err == nil {
			if werr := w.Write([][]float32{f[0], f[1]}, false); werr != nil {
				err = werr
			}
			n++
		}
	}
	if w != nil {
		if cerr := w.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}
	if r.onDone != nil {
		r.onDone(path, float64(n*bs)/sr, drops.Load(), err)
	}
}
