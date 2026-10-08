// Keyboard: play the synths from a MIDI keyboard, live, in the Musikverein.
//
// Every synth that plays notes is there: the piano, subtractive, pulsar, wavetable, fm, dx and
// grains, each with all its presets. Bank select (controller 0) switches the synth (in the order above) and a
// program change its preset (in alphabetical order); typed in the terminal, a synth or a preset
// name (or both: "wavetable glass") switches too, "list" shows them all.
//
// The notes play with the velocity curve chosen (-curve). The piano takes the velocity itself and
// has its pedals (sustain with half pedaling, sostenuto, soft); fm and dx take the velocity too,
// the other synths play it as the amplitude, and the sustain pedal holds their notes. Every MIDI
// event is printed with -monitor.
//
// A control panel runs alongside on a web page (http://localhost:8080, -addr to move it, empty to
// leave it out): the sound (every synth and preset, previous and next, all notes off), the
// transposition and the velocity curve, the volume and the hall, and meters. Press MIDI learn on
// the page, tap a control and turn a knob on the keyboard: the knob moves it from then on (the
// bindings are kept in keyboard-midi.json).
//
//	go run ./cmd/examples/keyboard -list                  the MIDI devices
//	go run ./cmd/examples/keyboard                        the first keyboard, through portaudio
//	go run ./cmd/examples/keyboard -device /dev/snd/midiC1D0 -preset felt -curve soft
//	go run ./cmd/examples/keyboard -synth subtractive -preset brass
//	go run ./cmd/examples/keyboard -jack                  JACK audio and a JACK MIDI port (sample accurate)
//	go run ./cmd/examples/keyboard -demo                  no keyboard: a phrase played as MIDI
package main

import (
	"bufio"
	"cmp"
	"flag"
	"fmt"
	"log"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/almerlucke/psalm"
	jackengine "github.com/almerlucke/psalm/audio/jack"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/keyboard"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/dx"
	"github.com/almerlucke/psalm/synths/fm"
	"github.com/almerlucke/psalm/synths/grains"
	"github.com/almerlucke/psalm/synths/piano"
	"github.com/almerlucke/psalm/synths/pulsar"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
	"github.com/almerlucke/sndfile"
)

// instrument is a synth with all its presets, ready to play
type instrument struct {
	name     string
	presets  []string // alphabetical
	synths   map[string]psalm.Modular
	keys     func(keyboard.Config) keyboard.Config // how the keyboard plays it
	hall     float64                               // the send to the hall
	selected atomic.Int64                          // the preset played last (the terminal reads it too)
}

// build makes an instrument of a synth: every preset by its constructor
func build(name string, presets []string, newSynth func(preset string) psalm.Modular, keys func(keyboard.Config) keyboard.Config, hall float64) *instrument {
	slices.Sort(presets)
	in := &instrument{name: name, presets: presets, synths: map[string]psalm.Modular{}, keys: keys, hall: hall}
	for _, preset := range presets {
		in.synths[preset] = newSynth(preset)
	}
	return in
}

// the piano takes the velocity itself (so the amplitude stays 1) and has its pedals
func pianoKeys(cfg keyboard.Config) keyboard.Config {
	cfg.Dynamics, cfg.Controls, cfg.HoldNotes = 0, nil, false
	return cfg
}

// most synths play the velocity as the amplitude, the sustain pedal holds their notes
func synthKeys(cfg keyboard.Config) keyboard.Config {
	cfg.Dynamics, cfg.Controls, cfg.HoldNotes = 1, map[int]string{}, true
	return cfg
}

// fm and dx take the velocity themselves (it brightens them), the sustain pedal holds their notes
func velocityKeys(cfg keyboard.Config) keyboard.Config {
	cfg.Dynamics, cfg.Controls, cfg.HoldNotes = 0, map[int]string{}, true
	return cfg
}

// merged reads several inputs as one, the events in order
type merged []midi.Input

func (m merged) Read(events []midi.Event, frames int, sampleRate float64) []midi.Event {
	start := len(events)
	for _, in := range m {
		events = in.Read(events, frames, sampleRate)
	}
	slices.SortStableFunc(events[start:], func(a, b midi.Event) int { return cmp.Compare(a.Offset, b.Offset) })
	return events
}

func main() {
	list := flag.Bool("list", false, "list the MIDI devices")
	device := flag.String("device", "", "the raw MIDI device (empty is the first keyboard)")
	useJack := flag.Bool("jack", false, "play through JACK, with a JACK MIDI input port")
	demo := flag.Bool("demo", false, "play a phrase as MIDI instead of reading a keyboard")
	synth := flag.String("synth", "piano", "the synth to start with: piano, subtractive, pulsar, wavetable, fm, dx or grains")
	preset := flag.String("preset", "", "the preset to start with (empty: felt for the piano, else the first)")
	curve := flag.String("curve", "linear", "the velocity curve: soft, linear, hard or an exponent")
	channel := flag.Int("channel", 0, "the MIDI channel (1-16), 0 is all")
	transpose := flag.Int("transpose", 0, "semitones")
	monitor := flag.Bool("monitor", true, "print every MIDI event")
	buffer := flag.Int("buffer", 256, "the buffer size (the latency) with portaudio")
	addr := flag.String("addr", "localhost:8087", "the address of the control panel page (empty: no panel)")
	bindingsFile := flag.String("bindings", "keyboard-midi.json", "the file the MIDI learn bindings of the panel are kept in")
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
		p        *psalm.Psalm
		input    midi.Input
		commands = midi.NewQueue() // what is typed in the terminal, as MIDI
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

	instruments := newInstruments(p)
	mix := mixInstruments(p, instruments)
	p.Out(mix.Master())

	// the instrument and preset to start with
	current := slices.IndexFunc(instruments, func(in *instrument) bool { return in.name == *synth })
	if current < 0 {
		log.Fatalf("synth %q: one of %v", *synth, names(instruments))
	}
	in := instruments[current]
	if *preset == "" && in.name == "piano" {
		*preset = "felt"
	}
	if *preset != "" {
		i := slices.Index(in.presets, *preset)
		if i < 0 {
			log.Fatalf("%s preset %q: one of %v", in.name, *preset, in.presets)
		}
		in.selected.Store(int64(i))
	}
	var playing atomic.Int64 // the instrument playing, for the terminal
	playing.Store(int64(current))
	synthOf := func(in *instrument) psalm.Modular { return in.synths[in.presets[in.selected.Load()]] }

	// the keyboard plays the selected preset. Bank select chooses the synth the next program
	// change switches to (as MIDI has it: keyboards send it along with their program changes), the
	// program change its preset (here, on the audio thread)
	kb := keyboard.New(merged{input, commands}, in.keys(cfg), synthOf(in))

	// every sound (a synth and a preset), in order, for the panel
	type sound struct{ synth, preset int }
	var sounds []sound
	var labels []string
	for i, in := range instruments {
		for j, name := range in.presets {
			sounds = append(sounds, sound{i, j})
			labels = append(labels, in.name+" · "+name)
		}
	}
	soundIndex := func(synth, preset int) int {
		return slices.Index(sounds, sound{synth, preset})
	}
	var soundSelect *panel.Param // the sound on the panel, following every switch

	// switchTo plays another synth and preset (on the audio thread): the notes of the one left are
	// released, the pedals of a piano left come up
	switchTo := func(synth, preset int, tick psalm.CtrlTick) {
		in := instruments[synth]
		preset = (preset%len(in.presets) + len(in.presets)) % len(in.presets)
		if synth == current && int64(preset) == in.selected.Load() {
			return
		}
		old := instruments[current]
		kb.AllNotesOff(tick)
		if old.name == "piano" {
			synthOf(old).Receive(psalm.Message{"sustain": 0.0, "sostenuto": 0.0, "soft": 0.0}, 0)
		}
		current = synth
		in.selected.Store(int64(preset))
		kb.SetConfig(in.keys(cfg))
		kb.SetTargets(synthOf(in))
		playing.Store(int64(current))
		if soundSelect != nil {
			soundSelect.Set(float64(soundIndex(synth, preset)))
		}
		log.Printf("%s: %s", in.name, in.presets[preset])
	}

	// bank select chooses the synth the next program change switches to (as MIDI has it: keyboards
	// send it along with their program changes), the program change its preset
	bank := current
	kb.OnControl(midi.CCBankSelect, func(value float64, _ psalm.CtrlTick) {
		if b := int(value*127 + 0.5); b < len(instruments) {
			bank = b
		}
	})
	kb.OnProgram(func(program int, tick psalm.CtrlTick) { switchTo(bank, program, tick) })
	if *monitor {
		kb.OnEvent(func(e midi.Event, _ psalm.CtrlTick) {
			if k := e.Message.Kind(); k != midi.Clock && k != midi.ActiveSensing {
				log.Printf("%v (frame %d)", e.Message, e.Offset)
			}
		})
	}
	kb.AddTo(p)

	// --- the control panel: the sound, the keyboard and the mix on a web page, its controls
	// movable from the knobs of the keyboard too (MIDI learn on the page) ---

	if *addr != "" {
		ctl := panel.New(p, "Keyboard")

		ctl.Section("Sound")
		soundSelect = ctl.Select("Sound", labels, func(i int) { switchTo(sounds[i].synth, sounds[i].preset, psalm.CtrlTick{}) },
			panel.Default(float64(soundIndex(current, int(in.selected.Load())))))
		step := func(by int) {
			next := (soundIndex(current, int(instruments[current].selected.Load())) + by + len(sounds)) % len(sounds)
			switchTo(sounds[next].synth, sounds[next].preset, psalm.CtrlTick{})
		}
		ctl.Button("Previous", func() { step(-1) })
		ctl.Button("Next", func() { step(1) })
		ctl.Button("All notes off", func() {
			kb.AllNotesOff(psalm.CtrlTick{})
			synthOf(instruments[current]).Receive(psalm.Message{"sustain": 0.0, "sostenuto": 0.0, "soft": 0.0}, 0)
		})

		ctl.Section("Keys")
		ctl.Fader("Transpose", -24, 24, panel.Linear, panel.Func(func(v float64) {
			cfg.Transpose = int(v)
			kb.SetConfig(instruments[current].keys(cfg))
		}), panel.Default(float64(*transpose)), panel.Step(1), panel.Unit("st"))
		curves := []keyboard.Curve{keyboard.Soft, keyboard.Linear, keyboard.Hard}
		curveNames := []string{"soft", "linear", "hard"}
		if !slices.Contains(curves, cfg.Curve) { // an exponent of its own from -curve
			curves = append(curves, cfg.Curve)
			curveNames = append(curveNames, *curve)
		}
		ctl.Select("Velocity curve", curveNames, func(i int) {
			cfg.Curve = curves[i]
			kb.SetConfig(instruments[current].keys(cfg))
		}, panel.Default(float64(slices.Index(curves, cfg.Curve))))

		ctl.Section("Mix")
		ctl.Fader("Volume", -40, 6, panel.Linear, panel.Message(mix.Master(), "volume"), panel.Unit("dB"), panel.Default(0), panel.Smooth(0.03))
		ctl.Fader("Hall", -40, 6, panel.Linear, panel.Message(mix.Strip("hall"), "volume"), panel.Unit("dB"), panel.Default(0), panel.Smooth(0.03))
		ctl.Meter("Master", mix.Master())
		ctl.Meter("Hall out", mix.Strip("hall"))

		ctl.MIDI(kb)
		if err := ctl.KeepBindings(*bindingsFile); err != nil {
			log.Fatal(err)
		}
		go func() {
			if err := ctl.Serve(*addr); err != nil {
				log.Fatal(err)
			}
		}()
		log.Printf("the control panel: http://%s", *addr)
	}

	go readCommands(commands, max(*channel, 1), instruments, &playing)

	log.Printf("%s: %s, %s velocity curve; bank select or a synth name switches between %v, a program change or a preset name the preset (\"list\" shows them)",
		in.name, in.presets[in.selected.Load()], *curve, names(instruments))
	if err := p.Listen(); err != nil {
		log.Fatal(err)
	}
}

// newInstruments makes every synth that plays notes with all its presets (grains when its
// recording is there)
func newInstruments(p *psalm.Psalm) []*instrument {
	table := gen.Brightness(32)
	instruments := []*instrument{
		build("piano", slices.Collect(maps.Keys(piano.Presets)), func(n string) psalm.Modular {
			return piano.NewPoly(piano.Presets[n](), 24).AddTo(p)
		}, pianoKeys, 0.45),
		build("subtractive", slices.Collect(maps.Keys(subtractive.Presets)), func(n string) psalm.Modular {
			return subtractive.NewPoly(subtractive.Presets[n](), 12).AddTo(p)
		}, synthKeys, 0.3),
		build("pulsar", slices.Collect(maps.Keys(pulsar.Presets)), func(n string) psalm.Modular {
			return pulsar.NewPoly(pulsar.Presets[n](), 12).AddTo(p)
		}, synthKeys, 0.3),
		build("wavetable", slices.Collect(maps.Keys(wavetable.Presets)), func(n string) psalm.Modular {
			return wavetable.NewPoly(wavetable.Presets[n](table), 12).AddTo(p)
		}, synthKeys, 0.3),
		build("fm", slices.Collect(maps.Keys(fm.Presets)), func(n string) psalm.Modular {
			return fm.NewPoly(fm.Presets[n](), 12).AddTo(p)
		}, velocityKeys, 0.3),
		build("dx", slices.Collect(maps.Keys(dx.Presets)), func(n string) psalm.Modular {
			return dx.NewPoly(dx.Presets[n](), 12, 2).AddTo(p)
		}, velocityKeys, 0.3),
	}
	choir, err := sndfile.NewMipMapSoundFile[float32]("assets/sounds/various/mixkit-angelical-choir-654.wav", 4)
	if err != nil {
		log.Printf("grains left out: %v", err)
		return instruments
	}
	return append(instruments, build("grains", slices.Collect(maps.Keys(grains.Presets)), func(n string) psalm.Modular {
		return grains.NewPoly(grains.Presets[n](choir, 57), 8).AddTo(p)
	}, synthKeys, 0.3))
}

// mixInstruments puts the synths on a mixer with the hall: idle synths rest, so all can be there.
// The presets are calibrated to the same loudness, every channel gets the same volume
func mixInstruments(p *psalm.Psalm, instruments []*instrument) *mixer.Mixer {
	mix := mixer.New(p)
	vienna := convolver.IRFromSoundFile(sndfile.MustSoundFile[float32]("assets/IR44100/Musikvereinsaal.wav"))
	mix.Bus("hall", mixer.StripConfig{}, convolver.New(vienna, convolver.Config{PartitionSize: 512, Mix: 1, Gain: 0.7}).AddTo(p))
	for _, in := range instruments {
		for _, preset := range in.presets {
			mix.Channel(in.name+"/"+preset, in.synths[preset], mixer.StripConfig{Volume: -3.5, Sends: map[string]float64{"hall": in.hall}})
		}
	}
	return mix
}

func names(instruments []*instrument) []string {
	var n []string
	for _, in := range instruments {
		n = append(n, in.name)
	}
	return n
}

// readCommands turns what is typed in the terminal into MIDI on a channel: a synth name, a preset
// name or number (of the synth playing, or after a synth name) a bank select and a program change
func readCommands(q *midi.Queue, channel int, instruments []*instrument, playing *atomic.Int64) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		words := strings.Fields(strings.ToLower(scanner.Text()))
		if len(words) == 0 {
			continue
		}
		if words[0] == "list" {
			for i, in := range instruments {
				fmt.Printf("%d %s: %s\n", i, in.name, strings.Join(in.presets, ", "))
			}
			continue
		}
		synth := int(playing.Load())
		if i := slices.IndexFunc(instruments, func(in *instrument) bool { return in.name == words[0] }); i >= 0 {
			synth = i
			words = words[1:]
			if len(words) == 0 {
				// the synth, at the preset it played last
				q.Push(midi.ControlChangeMsg(channel, midi.CCBankSelect, synth))
				q.Push(midi.ProgramChangeMsg(channel, int(instruments[synth].selected.Load())))
				continue
			}
		}
		presets := instruments[synth].presets
		program, err := strconv.Atoi(words[0])
		if err != nil {
			program = slices.IndexFunc(presets, func(p string) bool { return strings.ToLower(p) == words[0] })
		}
		if program < 0 || program >= len(presets) {
			fmt.Printf("%q: a synth (%s), a preset of %s (%s), a number or \"list\"\n",
				words[0], strings.Join(names(instruments), ", "), instruments[synth].name, strings.Join(presets, ", "))
			continue
		}
		q.Push(midi.ControlChangeMsg(channel, midi.CCBankSelect, synth))
		q.Push(midi.ProgramChangeMsg(channel, program))
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
