// Slicer: a recording cut into pieces, each on a key of its own, played from a MIDI keyboard. The pieces are found where the level
// jumps (the hits of a loop, the syllables of a phrase: -rise is how many dB, -gap the least seconds between two) or the recording is
// divided in equal ones (-slices 16). It prints the pieces and their keys at the start. Without a keyboard, -demo plays the pieces in their own
// order (the loop as it was), then shuffled.
//
//	go run ./cmd/examples/slicer -demo                                          the tabla loop, cut at its hits
//	go run ./cmd/examples/slicer -file assets/sounds/various/John_1-1.wav -rise 7 -gap 0.15 -demo
//	go run ./cmd/examples/slicer -file assets/sounds/chants/Factus.est.repente.VII.Ton.wav -by change -demo   a sung line, cut where the sound changes
//	go run ./cmd/examples/slicer -file loop.wav -slices 16 -first 48            sixteen equal pieces from C2... of the keyboard
package main

import (
	"cmp"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/keyboard"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/sampler"
)

func main() {
	file := flag.String("file", "assets/sounds/chants/Factus.est.repente.VII.Ton.wav", "the recording")
	even := flag.Int("slices", 0, "cut it in this many equal pieces (0: at the hits)")
	by := flag.String("by", "hits", "where to cut: at the hits of the level (hits), where the sound changes (change: a sung line), at the pauses (pauses: speech; -rise is the depth in dB, -gap the least pause, -max the longest piece), or as the verses and refrains of a recitation (verses: -verses n)")
	rise := flag.Float64("rise", 0, "hits: dB the level must rise (default 9); change: how many times the average the change must be (default 1.1)")
	verses := flag.Int("verses", 26, "verses: the verses of a recitation, each followed by a refrain (two pieces each)")
	longest := flag.Float64("max", 4, "pauses: the longest piece in seconds")
	gap := flag.Float64("gap", 0.08, "the least seconds between two hits")
	first := flag.Int("first", 36, "the key of the first piece")
	device := flag.String("device", "", "the raw MIDI device (empty is the first keyboard)")
	demo := flag.Bool("demo", false, "play the pieces instead of reading a keyboard")
	volume := flag.Float64("volume", -6, "dB")
	hallSend := flag.Float64("hall", 0.15, "how much goes to the hall (0-1)")
	flag.Parse()

	snd, err := sampler.LoadFile(*file)
	if err != nil {
		log.Fatal(err)
	}
	var cuts []int64
	if *even > 0 {
		cuts = sampler.EvenCuts(snd, *even)
	} else if *by == "verses" {
		cuts = sampler.VerseCuts(snd, *verses, cmp.Or(*rise, 20))
	} else if *by == "pauses" {
		cuts = sampler.PauseCuts(snd, cmp.Or(*rise, 30), *gap, *longest)
	} else if *by == "change" {
		cuts = sampler.ChangeCuts(snd, cmp.Or(*rise, 1.1), *gap)
	} else {
		cuts = sampler.OnsetCuts(snd, cmp.Or(*rise, 9), *gap)
	}
	regions := sampler.Slices(snd, *first, cuts)
	inst, err := sampler.NewInstrument(regions...)
	if err != nil {
		log.Fatal(err)
	}
	sr := snd.SampleRate()
	fmt.Printf("%s: %.2f s in %d pieces\n", *file, float64(snd.NumFrames())/sr, len(regions))
	for i, r := range regions {
		fmt.Printf("  key %3d  from %6.3f s  %5.3f s long\n", r.LoKey, float64(cuts[i])/sr, float64(cuts[i+1]-cuts[i])/sr)
	}

	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	var input midi.Input
	if *demo {
		q := midi.NewQueue()
		input = q
		go playDemo(q, regions, cuts, sr)
	} else {
		d, err := midi.OpenDevice(*device)
		if err != nil {
			log.Fatalf("%v (-demo plays without a keyboard)", err)
		}
		defer d.Close()
		input = d
	}
	synth := sampler.NewPoly(sampler.DefaultConfig(inst), 16).AddTo(p)
	keyboard.New(input, keyboard.Config{PitchBend: "pitchBend"}, synth).AddTo(p)

	mix := mixer.New(p)
	hall := fdnreverb.DefaultConfig()
	hall.Decay, hall.Mix = 1.8, 1
	mix.Bus("hall", mixer.StripConfig{}, fdnreverb.New(hall).AddTo(p))
	mix.Channel("slices", synth, mixer.StripConfig{Volume: *volume, Sends: map[string]float64{"hall": *hallSend}})
	p.Out(mix.Master())
	log.Printf("playing, Ctrl-C to stop")
	if err := p.Listen(); err != nil {
		log.Fatal(err)
	}
}

// playDemo plays the pieces one after the other, each for its own length, as the recording went; then in a shuffled order, again and again
func playDemo(q *midi.Queue, regions []sampler.Region, cuts []int64, sr float64) {
	order := make([]int, len(regions))
	for i := range order {
		order[i] = i
	}
	for round := 0; ; round++ {
		if round%2 == 1 {
			rand.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
		} else {
			for i := range order {
				order[i] = i
			}
		}
		for _, i := range order {
			q.Push(midi.NoteOnMsg(1, regions[i].LoKey, 100))
			time.Sleep(time.Duration(float64(cuts[i+1]-cuts[i]) / sr * float64(time.Second)))
			q.Push(midi.NoteOffMsg(1, regions[i].LoKey, 0))
		}
	}
}
