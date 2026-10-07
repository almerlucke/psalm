// Keyboard: play the piano from a MIDI keyboard, live, in the Musikverein.
//
// The notes play with the velocity curve chosen (-curve), the pedals work as on a piano (sustain
// with half pedaling, sostenuto, soft) and a program change switches the piano (the presets in
// alphabetical order). Every MIDI event is printed with -monitor.
//
//	go run ./cmd/examples/keyboard -list                  the MIDI devices
//	go run ./cmd/examples/keyboard                        the first keyboard, through portaudio
//	go run ./cmd/examples/keyboard -device /dev/snd/midiC1D0 -preset felt -curve soft
//	go run ./cmd/examples/keyboard -jack                  JACK audio and a JACK MIDI port (sample accurate)
//	go run ./cmd/examples/keyboard -demo                  no keyboard: a phrase played as MIDI
package main

import (
	"flag"
	"fmt"
	"log"
	"slices"
	"strconv"
	"time"

	"github.com/almerlucke/psalm"
	jackengine "github.com/almerlucke/psalm/audio/jack"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/keyboard"
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/piano"
	"github.com/almerlucke/sndfile"
)

func main() {
	list := flag.Bool("list", false, "list the MIDI devices")
	device := flag.String("device", "", "the raw MIDI device (empty is the first keyboard)")
	useJack := flag.Bool("jack", false, "play through JACK, with a JACK MIDI input port")
	demo := flag.Bool("demo", false, "play a phrase as MIDI instead of reading a keyboard")
	preset := flag.String("preset", "felt", "the piano to start with")
	curve := flag.String("curve", "linear", "the velocity curve: soft, linear, hard or an exponent")
	channel := flag.Int("channel", 0, "the MIDI channel (1-16), 0 is all")
	transpose := flag.Int("transpose", 0, "semitones")
	monitor := flag.Bool("monitor", true, "print every MIDI event")
	buffer := flag.Int("buffer", 256, "the buffer size (the latency) with portaudio")
	flag.Parse()

	if *list {
		devices, err := midi.Devices()
		if err != nil {
			log.Fatal(err)
		}
		for _, d := range devices {
			fmt.Println(d)
		}
		return
	}

	cfg := keyboard.Config{Channel: *channel, Transpose: *transpose}
	switch *curve {
	case "soft":
		cfg.Curve = keyboard.Soft
	case "linear":
		cfg.Curve = keyboard.Linear
	case "hard":
		cfg.Curve = keyboard.Hard
	default:
		c, err := strconv.ParseFloat(*curve, 64)
		if err != nil || c <= 0 {
			log.Fatalf("curve %q: soft, linear, hard or an exponent above 0", *curve)
		}
		cfg.Curve = keyboard.Curve(c)
	}

	// the engine and the input
	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: *buffer})
	var (
		p     *psalm.Psalm
		input midi.Input
	)
	switch {
	case *useJack:
		engine := jackengine.New("psalm")
		in, err := engine.MidiIn("midi_in")
		if err != nil {
			log.Fatal(err)
		}
		input = in
		p = psalm.New(engine)
		log.Printf("JACK: connect a keyboard to psalm:midi_in, psalm:out_0 and out_1 to the outputs")
	case *demo:
		q := midi.NewQueue()
		input = q
		go playDemo(q)
		p = psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	default:
		d, err := midi.OpenDevice(*device)
		if err != nil {
			log.Fatalf("%v (see -list, or play without a keyboard with -demo)", err)
		}
		defer d.Close()
		log.Printf("reading %s", d.Path())
		input = d
		p = psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	}
	defer p.Close()

	// the pianos, on the mixer with the hall: idle pianos rest, so all can be there
	mix := mixer.New(p)
	vienna := convolver.IRFromSoundFile(sndfile.MustSoundFile[float32]("assets/IR44100/Musikvereinsaal.wav"))
	mix.Bus("hall", mixer.StripConfig{}, convolver.New(vienna, convolver.Config{PartitionSize: 512, Mix: 1, Gain: 0.7}).AddTo(p))
	var names []string
	for name := range piano.Presets {
		names = append(names, name)
	}
	slices.Sort(names)
	if !slices.Contains(names, *preset) {
		log.Fatalf("preset %q: one of %v", *preset, names)
	}
	pianos := map[string]psalm.Modular{}
	for _, name := range names {
		synth := piano.NewPoly(piano.Presets[name](), 24).AddTo(p)
		pianos[name] = synth
		mix.Channel(name, synth, mixer.StripConfig{Volume: -3.5, Sends: map[string]float64{"hall": 0.45}})
	}
	p.Out(mix.Master())

	// the keyboard: the velocity goes to the piano, which has its own response, so the amplitude
	// stays 1 (Dynamics 0)
	kb := keyboard.New(input, cfg, pianos[*preset])
	kb.OnProgram(func(program int, tick psalm.CtrlTick) {
		name := names[program%len(names)]
		kb.AllNotesOff(tick)
		kb.SetTargets(pianos[name])
		log.Printf("program %d: %s", program, name)
	})
	if *monitor {
		kb.OnEvent(func(e midi.Event, _ psalm.CtrlTick) {
			if k := e.Message.Kind(); k != midi.Clock && k != midi.ActiveSensing {
				log.Printf("%v (frame %d)", e.Message, e.Offset)
			}
		})
	}
	kb.AddTo(p)

	log.Printf("%s, %s velocity curve; program changes switch between %v", *preset, *curve, names)
	if err := p.Listen(); err != nil {
		log.Fatal(err)
	}
}

// playDemo plays a phrase into a queue as a keyboard would: a few chords with the sustain pedal
// changed on every one, then a soft line, over and over
func playDemo(q *midi.Queue) {
	type ev struct {
		beats float64
		msg   midi.Message
	}
	var score []ev
	at := 0.0
	chord := func(beats float64, velocity int, notes ...int) {
		score = append(score, ev{at, midi.ControlChangeMsg(1, midi.CCSustain, 0)})
		for _, n := range notes {
			score = append(score, ev{at + 0.02, midi.NoteOnMsg(1, n, velocity)})
		}
		score = append(score, ev{at + 0.1, midi.ControlChangeMsg(1, midi.CCSustain, 127)})
		for _, n := range notes {
			score = append(score, ev{at + beats*0.6, midi.NoteOffMsg(1, n, 0)})
		}
		at += beats
	}
	chord(2, 70, 45, 57, 64, 72)
	chord(2, 60, 41, 57, 65, 69)
	chord(2, 80, 43, 55, 62, 71)
	chord(2, 50, 40, 55, 64, 67)
	for i, n := range []int{76, 74, 72, 71, 72, 67, 69, 64} {
		score = append(score, ev{at, midi.NoteOnMsg(1, n, 40+i*5)}, ev{at + 0.45, midi.NoteOffMsg(1, n, 0)})
		at += 0.5
	}
	chord(4, 55, 33, 57, 60, 64, 69)
	score = append(score, ev{at, midi.ControlChangeMsg(1, midi.CCSustain, 0)})
	at += 1
	slices.SortStableFunc(score, func(a, b ev) int {
		switch {
		case a.beats < b.beats:
			return -1
		case a.beats > b.beats:
			return 1
		}
		return 0
	})

	beat := 60 / 72.0 * float64(time.Second)
	for {
		start := time.Now()
		for _, e := range score {
			time.Sleep(time.Until(start.Add(time.Duration(e.beats * beat))))
			q.Push(e.msg)
		}
		time.Sleep(time.Until(start.Add(time.Duration(at * beat))))
	}
}
