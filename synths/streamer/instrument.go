// Package streamer plays sound files from disk while they play, instead of from memory: a long recording (a piece of music, an hour of rain, a field recording)
// costs the memory of its first moments only, the rest is read from the file by a goroutine of the note and handed to the audio thread through a ring.
//
// A note starts at once: the first HeadSeconds of every file are kept in memory, and the goroutine has read on from there by the time they are played. A file
// is read with the streaming reader of the sndfile package (pure Go: wav, aiff, mp3, flac and ogg play, other formats with its libsndfile build tag). The audio thread
// never waits for the disk: when the reader falls behind (a slow disk) the sound has a gap and Voice.Underruns counts it.
package streamer

import (
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/almerlucke/psalm/synths/sampler"
	"github.com/almerlucke/sndfile"
)

// HeadSeconds is how much of the start of a sound is kept in memory
const HeadSeconds = 0.5

// Entry is a sound on a key: a file, and how it plays
type Entry struct {
	Name string
	Path string
	Key  int
	// Loop goes round between LoopStart and LoopEnd (seconds, by default the start and the end) while the key is down; a note off releases it
	Loop bool
	// Hold lets a note off release a sound that is not a loop (otherwise it plays to its end, like a drum)
	Hold               bool
	Speed              float64 // 1; the pitch goes with it
	Start, End         float64 // the part of the file to play, seconds (End 0: to the end)
	LoopStart, LoopEnd float64 // seconds
	Release            float64 // seconds a note off takes to fade the sound out (0.1)
	Volume             float64 // dB
	Pan                float64 // -1 to 1

	sr         float64
	channels   int
	frames     int64
	startF     int64
	endF       int64
	loopStartF int64
	loopEndF   int64
	head       []float32 // stereo frames from startF, as many as are in memory
}

// Seconds is how long the sound is, as it plays once
func (e *Entry) Seconds() float64 { return float64(e.endF-e.startF) / e.sr }

// Instrument is the sounds of a streamer by their keys
type Instrument struct {
	Entries []*Entry
	byKey   map[int]*Entry
}

// Lookup is the sound on a key; an instrument of one sound plays it on every key
func (in *Instrument) Lookup(key int) *Entry {
	if e, ok := in.byKey[key]; ok {
		return e
	}
	if len(in.Entries) == 1 {
		return in.Entries[0]
	}
	return nil
}

// NewInstrument is an instrument of entries, opened (their files read for what they are and their heads)
func NewInstrument(entries ...*Entry) (*Instrument, error) {
	in := &Instrument{byKey: map[int]*Entry{}}
	for _, e := range entries {
		if err := e.open(); err != nil {
			return nil, err
		}
		in.Entries = append(in.Entries, e)
		in.byKey[e.Key] = e
	}
	return in, nil
}

// Open is a sound of one file on a key, as it is
func Open(path string, key int) (*Instrument, error) {
	return NewInstrument(&Entry{Path: path, Key: key})
}

// open reads the file's header and the start of it into memory
func (e *Entry) open() error {
	if e.Speed <= 0 {
		e.Speed = 1
	}
	if e.Release <= 0 {
		e.Release = 0.1
	}
	if e.Name == "" {
		e.Name = strings.TrimSuffix(filepath.Base(e.Path), filepath.Ext(e.Path))
	}
	f, err := sndfile.Open[float32](e.Path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	info := f.Info()
	if info.Frames <= 0 || info.Channels <= 0 || info.SampleRate <= 0 {
		return fmt.Errorf("%s: no sound in it", e.Path)
	}
	e.sr, e.channels, e.frames = info.SampleRate, info.Channels, info.Frames
	frame := func(sec float64, def int64) int64 {
		if sec <= 0 {
			return def
		}
		return min(e.frames, int64(sec*e.sr))
	}
	e.startF = frame(e.Start, 0)
	e.endF = frame(e.End, e.frames)
	if e.endF <= e.startF {
		return fmt.Errorf("%s: it ends (%g s) before it starts (%g s)", e.Path, e.End, e.Start)
	}
	e.loopStartF = max(e.startF, frame(e.LoopStart, e.startF))
	e.loopEndF = min(e.endF, frame(e.LoopEnd, e.endF))
	if e.loopEndF <= e.loopStartF {
		return fmt.Errorf("%s: the loop ends (%g s) before it starts (%g s)", e.Path, e.LoopEnd, e.LoopStart)
	}
	// the head: what plays before the reader of the note has got on; with a loop it is up to the end of it
	limit := e.endF
	if e.Loop {
		limit = e.loopEndF
	}
	n := min(int64(HeadSeconds*e.sr), limit-e.startF)
	e.head = make([]float32, 0, 2*n)
	if err := f.SeekFrame(e.startF); err != nil {
		return err
	}
	buf := make([]float32, 4096*e.channels)
	for int64(len(e.head)/2) < n {
		want := min(4096, n-int64(len(e.head)/2))
		got, err := f.ReadFrames(buf[:want*int64(e.channels)])
		if err != nil {
			return err
		}
		if got == 0 {
			break
		}
		e.head = appendStereo(e.head, buf, int(got), e.channels)
	}
	return nil
}

// appendStereo adds frames of interleaved channels as stereo frames (a mono sound on both, more than two: the first two)
func appendStereo(dst, src []float32, frames, channels int) []float32 {
	for i := range frames {
		l := src[i*channels]
		r := l
		if channels > 1 {
			r = src[i*channels+1]
		}
		dst = append(dst, l, r)
	}
	return dst
}

// ParseInstrument makes an instrument from a text of one line for each sound:
//
//	# the key, the file, and anything of: loop hold  speed=1 start=sec end=sec loopstart=sec loopend=sec release=sec vol=dB pan=-1..1 name=rain
//	36 assets/sounds/various/rain-sound-188158.wav loop
//
// A file name with spaces is in double quotes.
//
// The second result says what every line made, to show it on the page. Empty lines and lines that begin with # are left out.
func ParseInstrument(text string) (*Instrument, []string, error) {
	var entries []*Entry
	for n, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := sampler.Fields(line)
		fail := func(format string, a ...any) error { return fmt.Errorf("line %d: %s", n+1, fmt.Sprintf(format, a...)) }
		if len(fields) < 2 {
			return nil, nil, fail("a key and a file: 36 rain.wav loop")
		}
		key, err := strconv.Atoi(fields[0])
		if err != nil || key < 0 || key > 127 {
			return nil, nil, fail("%q is not a key from 0 to 127", fields[0])
		}
		e := &Entry{Key: key, Path: fields[1]}
		for _, f := range fields[2:] {
			name, val, ok := strings.Cut(f, "=")
			if !ok {
				switch f {
				case "loop":
					e.Loop = true
				case "hold":
					e.Hold = true
				default:
					return nil, nil, fail("%q: loop, hold, or name=value", f)
				}
				continue
			}
			if name == "name" {
				e.Name = val
				continue
			}
			x, err := strconv.ParseFloat(val, 64)
			if err != nil || math.IsNaN(x) || math.IsInf(x, 0) {
				return nil, nil, fail("%q is not a number", val)
			}
			switch name {
			case "speed":
				e.Speed = x
			case "start":
				e.Start = x
			case "end":
				e.End = x
			case "loopstart":
				e.LoopStart = x
			case "loopend":
				e.LoopEnd = x
			case "release":
				e.Release = x
			case "vol":
				e.Volume = x
			case "pan":
				e.Pan = max(-1, min(1, x))
			default:
				return nil, nil, fail("%q: speed, start, end, loopstart, loopend, release, vol, pan or name", name)
			}
		}
		if err := e.open(); err != nil {
			return nil, nil, fail("%v", err)
		}
		entries = append(entries, e)
	}
	if len(entries) == 0 {
		return nil, nil, fmt.Errorf("no sounds")
	}
	in, err := NewInstrument(entries...)
	if err != nil {
		return nil, nil, err
	}
	var report []string
	for _, e := range in.Entries {
		kind := "stereo"
		if e.channels == 1 {
			kind = "mono"
		} else if e.channels > 2 {
			kind = fmt.Sprintf("%d channels", e.channels)
		}
		how := "plays to its end"
		switch {
		case e.Loop:
			how = "loops"
		case e.Hold:
			how = "stops at a note off"
		}
		report = append(report, fmt.Sprintf("%d %s: %d:%02d, %s, %.0f Hz, from disk (%.1f s in memory), %s", e.Key, e.Name, int(e.Seconds())/60, int(e.Seconds())%60,
			kind, e.sr, float64(len(e.head)/2)/e.sr, how))
	}
	return in, report, nil
}
