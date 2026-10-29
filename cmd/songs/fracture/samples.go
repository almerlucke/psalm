package main

import (
	"fmt"
	"os"

	"github.com/almerlucke/psalm/synths/sampler"
)

// a recording cut at its hits into pieces, a piece on each key from firstKey: the sampler of the glitch plays them in any order
type cutSound struct {
	name, path string
	rise, gap  float64 // how a hit is found: dB over the 50 ms before, and the seconds between two
	vol        float64 // dB
}

const firstKey = 36

var (
	// a blurry vocal cut into syllables and breaths, the loop of a tabla cut at its strokes
	vocalCut = cutSound{"chops", "assets/sounds/various/Cymatics - Blurry Vocal - 80 BPM F Min.wav", 6, 0.06, 0}
	tablaCut = cutSound{"slices", "assets/sounds/various/tabla_loop_4bars_90bpm-86005.wav", 9, 0.08, 0}
)

// there says whether the recording is there (the piece plays without the ones that are not)
func (c cutSound) there() bool {
	_, err := os.Stat(c.path)
	return err == nil
}

// line is the sound of the sampler: the recording cut at its hits
func (c cutSound) line() string {
	return fmt.Sprintf("%d %q hits %g %g name=%s vol=%g", firstKey, c.path, c.rise, c.gap, c.name, c.vol)
}

// keys are the keys of the pieces
func (c cutSound) keys() ([]int, error) {
	snd, err := sampler.LoadFile(c.path)
	if err != nil {
		return nil, err
	}
	cuts := sampler.OnsetCuts(snd, c.rise, c.gap)
	var keys []int
	for i := 0; i+1 < len(cuts) && firstKey+i <= 127; i++ {
		keys = append(keys, firstKey+i)
	}
	return keys, nil
}
