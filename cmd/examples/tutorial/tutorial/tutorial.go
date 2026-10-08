// Package tutorial holds what every step of the tutorial needs besides the music: the flags, the
// patch and playing it. Step 1 writes all of it out; the other steps call New and Play, so they
// show only what they teach.
package tutorial

import (
	"flag"
	"fmt"

	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/tools/render"
)

var (
	renderTo = flag.String("render", "", "render to this WAV file instead of playing through the sound card")
	seed     = flag.Uint64("seed", 0, "seed for everything random (the feel, the generators): the same seed plays the same")
)

// New reads the flags and creates the patch at a tempo: playing through the sound card, or
// offline when the step renders to a file (-render)
func New(bpm float64) *psalm.Psalm {
	flag.Parse()
	if *seed != 0 {
		psalm.Seed(*seed)
	}
	psalm.PushConfig(psalm.Config{BPM: bpm, PPQ: 240, SampleRate: 44100, BufferSize: 512})
	if *renderTo != "" {
		return psalm.New(nil)
	}
	return psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
}

// Play sends a module to the speakers and plays until you stop it (Ctrl-C), or with -render writes
// seconds of it (and a few seconds for the sound to ring out) to the file
func Play(p *psalm.Psalm, out psalm.Modular, seconds float64) {
	defer p.Close()
	p.Out(out)
	if *renderTo == "" {
		fmt.Println("playing, Ctrl-C to stop")
		if err := p.Listen(); err != nil {
			panic(err)
		}
		return
	}
	stats, err := render.ToFile(p, *renderTo, render.Options{Seconds: seconds, Tail: 3})
	if err != nil {
		panic(err)
	}
	fmt.Println(stats)
}
