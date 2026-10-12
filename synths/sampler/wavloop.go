package sampler

import (
	"encoding/binary"
	"io"
	"os"
	"strings"
)

// Looper is a sound with a loop of its own, as a WAV file can have (its smpl chunk)
type Looper interface {
	Loop() (start, end int64, ok bool)
}

// looped is a sound and the loop its file holds
type looped struct {
	Sound
	start, end int64
}

func (l *looped) Loop() (int64, int64, bool) { return l.start, l.end, true }

// withLoop returns a sound with the loop of its WAV file, when it has one
func withLoop(snd Sound, path string) Sound {
	if !strings.HasSuffix(strings.ToLower(path), ".wav") {
		return snd
	}
	start, end, ok := wavLoop(path)
	if !ok || start < 0 || end <= start || end >= snd.NumFrames() {
		return snd
	}
	return &looped{Sound: snd, start: start, end: end}
}

// wavLoop reads the first loop of a WAV file's smpl chunk (frames, the end the last frame of the
// loop), skipping past the audio, not reading it
func wavLoop(path string) (start, end int64, ok bool) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, false
	}
	defer f.Close()
	var head [12]byte
	if _, err := io.ReadFull(f, head[:]); err != nil || string(head[0:4]) != "RIFF" || string(head[8:12]) != "WAVE" {
		return 0, 0, false
	}
	for {
		var ch [8]byte
		if _, err := io.ReadFull(f, ch[:]); err != nil {
			return 0, 0, false
		}
		size := int64(binary.LittleEndian.Uint32(ch[4:]))
		if string(ch[0:4]) != "smpl" {
			if _, err := f.Seek(size+size&1, io.SeekCurrent); err != nil {
				return 0, 0, false
			}
			continue
		}
		if size < 36+24 {
			return 0, 0, false
		}
		data := make([]byte, 36+24)
		if _, err := io.ReadFull(f, data); err != nil {
			return 0, 0, false
		}
		if binary.LittleEndian.Uint32(data[28:]) == 0 { // no loops
			return 0, 0, false
		}
		loop := data[36:]
		return int64(binary.LittleEndian.Uint32(loop[8:])), int64(binary.LittleEndian.Uint32(loop[12:])), true
	}
}
