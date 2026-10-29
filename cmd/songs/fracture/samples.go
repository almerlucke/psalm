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
	maxSec     float64 // the pieces longer than this are left out: a piece plays to its end, and a long one is a sound that does not stop
}

const firstKey = 36

var (
	// a sung recording cut into syllables (the short ones only), the loop of a tabla cut at its strokes
	vocalCut = cutSound{"chops", "assets/sounds/goedertierenheid.wav", 9, 0.06, 0, 0.4}
	tablaCut = cutSound{"slices", "assets/sounds/various/tabla_loop_4bars_90bpm-86005.wav", 9, 0.08, 0, 0.5}
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
	sr := snd.SampleRate()
	var keys []int
	for i := 0; i+1 < len(cuts) && firstKey+i <= 127; i++ {
		if c.maxSec <= 0 || float64(cuts[i+1]-cuts[i])/sr <= c.maxSec {
			keys = append(keys, firstKey+i)
		}
	}
	return keys, nil
}
