package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/almerlucke/psalm/tools/harmony"
	"github.com/almerlucke/psalm/tools/tune"
	"github.com/almerlucke/sndfile"
	"github.com/almerlucke/sndfile/writer"
)

// The voice, in tune with the music: the recording of the psalm is followed for its pitch and its voiced parts (the vowels) are moved to the notes of the chord that is playing
// under that word, or to the notes of the scale of the section (tools/tune). The consonants and the breaths stay as they were. It is made once, offline, to a file; the piece
// plays that file in place of the recording (-voice).

// chordTimeline is the chord at every bar of the piece
type chordTimeline struct {
	starts []float64 // the second every bar begins, from the start of the piece
	chords []harmony.Chord
	scales []harmony.Scale // of the section the bar is in
	wonder float64         // the second the Wonder begins: the voice begins there
}

// timeline reads the chords of the sections (a chord a bar, "*n" for n bars, round for the bars of the section) at the tempo of the piece
func timeline() (chordTimeline, error) {
	var tl chordTimeline
	seconds := 0.0
	const step = 1.0 / 64 // of a beat
	for i, s := range sections {
		if s.name == "Wonder" { // the voice begins with the Wonder (scripts/voice.js)
			tl.wonder = seconds
		}
		var cycle []harmony.Chord
		for _, field := range strings.Fields(progressions[s.name]) {
			symbol, n, _ := strings.Cut(field, "*")
			bars := 1
			if n != "" {
				v, err := strconv.Atoi(n)
				if err != nil {
					return tl, fmt.Errorf("%s: %q: %w", s.name, field, err)
				}
				bars = v
			}
			c, err := harmony.ParseChord(symbol)
			if err != nil {
				return tl, fmt.Errorf("%s: %w", s.name, err)
			}
			for range bars {
				cycle = append(cycle, c)
			}
		}
		scale := harmony.MustScale("D", "major")
		if s.name == "Shadow" { // the relative minor
			scale = harmony.MustScale("B", "minor")
		}
		for b := range s.bars {
			tl.starts = append(tl.starts, seconds)
			tl.chords = append(tl.chords, cycle[b%len(cycle)])
			tl.scales = append(tl.scales, scale)
			for beat := 0.0; beat < beats; beat += step {
				seconds += step * 60 / tempoAt(i, float64(b)+beat/beats)
			}
		}
	}
	return tl, nil
}

// bar is the bar of the piece at a second of the voice (it begins with the Wonder)
func (tl chordTimeline) bar(voiceSeconds float64) int {
	t := voiceSeconds + tl.wonder
	i := 0
	for i+1 < len(tl.starts) && tl.starts[i+1] <= t {
		i++
	}
	return i
}

func (tl chordTimeline) chordTones(voiceSeconds float64) []int {
	return tl.chords[tl.bar(voiceSeconds)].PitchClasses()
}

func (tl chordTimeline) scaleTones(voiceSeconds float64) []int {
	s := tl.scales[tl.bar(voiceSeconds)]
	var pcs []int
	for pc := range 12 {
		if s.Contains(float64(60 + pc)) {
			pcs = append(pcs, pc)
		}
	}
	return pcs
}

// tuneOptions are the flags of -tune
type tuneOptions struct {
	mode     string  // chant (the chord tones, a word on one note) or snap (the scale, every moment on the nearest note)
	speed    float64 // milliseconds the voice takes to move to a note
	keep     float64 // for chant: how much of the rise and fall of the voice stays
	strength float64 // how far toward the notes
	offset   float64 // semitones added to the notes
}

// tuneVoice reads the recording at `in`, brings its pitch to the notes of the music and writes it to `out` (a stereo file, the same length)
func tuneVoice(in, out string, o tuneOptions) error {
	tl, err := timeline()
	if err != nil {
		return err
	}
	sf, err := sndfile.NewSoundFile[float32](in)
	if err != nil {
		return err
	}
	sr := sf.SampleRate()
	chans := sf.NumChannels()
	n := int(sf.NumFrames())
	x := make([]float64, n)
	var apart float64
	for i := range n {
		var s float64
		for c := range chans {
			s += float64(sf.Buffer(c, 0)[i])
		}
		x[i] = s / float64(chans)
		if chans > 1 {
			apart = max(apart, math.Abs(float64(sf.Buffer(0, 0)[i]-sf.Buffer(1, 0)[i])))
		}
	}
	if apart > 1e-3 {
		fmt.Printf("the two sides differ by %.4f: they are mixed to one voice\n", apart)
	}
	track := tune.Detect(x, sr, tune.DetectOptions{})
	voiced := 0
	for _, f := range track.F0 {
		if f > 0 {
			voiced++
		}
	}
	fmt.Printf("%.1f s of voice, %.0f%% with a pitch\n", float64(n)/sr, 100*float64(voiced)/float64(len(track.F0)))
	opts := tune.Options{Speed: o.speed, Keep: o.keep, Strength: o.strength, Offset: o.offset}
	switch o.mode {
	case "chant":
		opts.Mode, opts.Allowed = tune.Chant, tl.chordTones
	case "snap":
		opts.Mode, opts.Allowed = tune.Snap, tl.scaleTones
	default:
		return fmt.Errorf("mode %q: chant or snap", o.mode)
	}
	y := tune.Shift(x, track, tune.Target(track, opts))
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	const bs = 1024
	w, err := writer.New(out, writer.WAV, 2, sr, writer.NewChannelConverter[float32](bs, 2))
	if err != nil {
		return err
	}
	buf := [][]float32{make([]float32, bs), make([]float32, bs)}
	for at := 0; at < n; at += bs {
		clear(buf[0])
		clear(buf[1])
		for i := 0; i < bs && at+i < n; i++ {
			buf[0][i], buf[1][i] = float32(y[at+i]), float32(y[at+i])
		}
		if err := w.Write(buf, at+bs >= n); err != nil {
			return err
		}
	}
	if err := w.Close(); err != nil {
		return err
	}
	fmt.Println("wrote", out)
	return nil
}
