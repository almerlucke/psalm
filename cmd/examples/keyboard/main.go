// Keyboard: play the synths from a MIDI keyboard, live, in the Musikverein.
//
// Every synth that plays notes is there: the piano, subtractive, pulsar, wavetable, fm, dx, organ,
// waveguide (plucked and bowed strings), modal (mallets and bells), vocal, pd, westcoast, supersaw,
// drumsynth (its kits are its presets) and grains, each with all its presets. Bank select
// (controller 0) switches the synth (in the order above) and a program change its preset (in
// alphabetical order); typed in the terminal, a synth or a preset name (or both: "wavetable
// glass") switches too, "list" shows them all.
//
// The notes play with the velocity curve chosen (-curve). The piano takes the velocity itself and
// has its pedals (sustain with half pedaling, sostenuto, soft); fm, dx, waveguide, modal, westcoast
// and drumsynth take the velocity too, the organ doesn't feel it (as a real one), the other synths
// play it as the amplitude, and the sustain pedal holds their notes. The pitch bend bends every
// synth but the piano (two semitones), the mod wheel does something of its own on each: opens the
// filter, brightens, brings in a vibrato, moves the vowel or the table; pressing harder on a held
// key (aftertouch) does the same. -route sends a controller (or the bend, or the aftertouch) to
// any setting of the synth instead, scaled to its range: -route 1:cutoff:300:8000:exp makes the
// mod wheel sweep the cutoff (the settings each synth takes are listed under Messages in
// docs/synths.md). Every MIDI event is printed
// with -monitor.
//
// A control panel runs alongside on a web page (http://localhost:8087, -addr to move it, empty to
// leave it out): the sound (every synth and preset, previous and next, all notes off), the
// transposition and the velocity curve, the routes (where the mod wheel and the aftertouch go: the
// synth's own, or a setting such as the cutoff, the fold or the vowel, and how much of its range),
// the volume and the hall, and meters. Press MIDI learn on the page, tap a control and turn a knob
// on the keyboard: the knob moves it from then on (the bindings are kept in keyboard-midi.json).
// The page has a keyboard of its own too (the computer's keys play it: A W S E D ...), played as
// MIDI, and the amplitude envelope of the synth playing to drag into shape.
// Record on the panel (or -record, from the start) records what is played; switched off (or at the
// end) it is written to a MIDI file, recording-<date>-<time>.mid, to play again with the midifile
// example.
//
//	go run ./cmd/examples/keyboard -list                  the MIDI devices
//	go run ./cmd/examples/keyboard                        the first keyboard, through portaudio
//	go run ./cmd/examples/keyboard -device /dev/snd/midiC1D0 -preset felt -curve soft
//	go run ./cmd/examples/keyboard -synth subtractive -preset brass
//	go run ./cmd/examples/keyboard -synth supersaw -preset glideLead -mono   one note at a time, sliding
//	go run ./cmd/examples/keyboard -synth westcoast -route 1:fold:0:1 -route pressure:fm:0:3
//	                                                      the wheel on the fold, the aftertouch on the FM index
//	go run ./cmd/examples/keyboard -jack                  JACK audio and a JACK MIDI port (sample accurate)
//	go run ./cmd/examples/keyboard -demo                  no keyboard: a phrase played as MIDI
//	go run ./cmd/examples/keyboard -record                record what is played to a MIDI file
package main

import (
	"bufio"
	"cmp"
	"flag"
	"fmt"
	"log"
	"maps"
	"math"
	"os"
	"path/filepath"
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
	"github.com/almerlucke/psalm/modules/controls/panel/widgets"
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/drumsynth"
	"github.com/almerlucke/psalm/synths/dx"
	"github.com/almerlucke/psalm/synths/grains"
	"github.com/almerlucke/psalm/synths/modal"
	"github.com/almerlucke/psalm/synths/organ"
	"github.com/almerlucke/psalm/synths/pd"
	"github.com/almerlucke/psalm/synths/piano"
	"github.com/almerlucke/psalm/synths/pulsar"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/synths/supersaw"
	"github.com/almerlucke/psalm/synths/vocal"
	"github.com/almerlucke/psalm/synths/waveguide"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
	"github.com/almerlucke/psalm/synths/westcoast"
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

// expressive are the hands on the synths: the pitch bend, the mod wheel (controller 1), and the
// aftertouch (pressing harder on a held key: it adds to the wheel)
var expressive = map[int]string{midi.CCModWheel: "modWheel"}

// most synths play the velocity as the amplitude, the sustain pedal holds their notes, they bend
// and take the mod wheel
func synthKeys(cfg keyboard.Config) keyboard.Config {
	cfg.Dynamics, cfg.Controls, cfg.HoldNotes, cfg.PitchBend, cfg.Pressure, cfg.PolyPressure = 1, expressive, true, "pitchBend", "pressure", "pressure"
	return cfg
}

// fm, dx and the physical models take the velocity themselves (it brightens them, or plays the
// mallet or the string harder), the sustain pedal holds their notes
func velocityKeys(cfg keyboard.Config) keyboard.Config {
	cfg.Dynamics, cfg.Controls, cfg.HoldNotes, cfg.PitchBend, cfg.Pressure, cfg.PolyPressure = 0, expressive, true, "pitchBend", "pressure", "pressure"
	return cfg
}

// destination is a setting the mod wheel or the aftertouch can be sent to from the panel: its
// message names (the synths that take it), its range
type destination struct {
	label    string
	names    []string
	min, max float64
	exp      bool
}

// destinations are what the panel offers, the first the synth's own (the built in wheel and
// aftertouch). A synth that doesn't take a setting doesn't hear it
var destinations = []destination{
	{"built in", nil, 0, 0, false},
	{"cutoff (subtractive, wavetable, supersaw)", []string{"cutoff"}, 200, 12000, true},
	{"resonance (subtractive, wavetable, supersaw, westcoast)", []string{"resonance"}, 0, 0.95, false},
	{"filter envelope (subtractive, wavetable, supersaw)", []string{"filterEnvAmount"}, 0, 6, false},
	{"vowel (vocal, subtractive formant)", []string{"vowel"}, 0, 4, false},
	{"brightness (fm, dx)", []string{"modDepth", "modScale"}, 0.2, 2, false},
	{"brightness (vocal, waveguide pluck)", []string{"brightness"}, 0, 1, false},
	{"vibrato depth (fm, wavetable, pulsar, vocal, pd, waveguide)", []string{"vibratoDepth"}, 0, 1, false},
	{"position (wavetable table, grains sample)", []string{"position"}, 0, 1, false},
	{"detune (supersaw)", []string{"detune"}, 0, 1, false},
	{"detune cents (wavetable, pd)", []string{"detune"}, 0, 50, false},
	{"saw mix (supersaw)", []string{"mix"}, 0, 1, false},
	{"distortion (pd)", []string{"dcw1", "dcw2"}, 0, 1, false},
	{"fold (westcoast)", []string{"fold"}, 0, 1, false},
	{"symmetry (westcoast)", []string{"symmetry"}, -1, 1, false},
	{"fm index (westcoast)", []string{"fm"}, 0, 3, false},
	{"formant (pulsar)", []string{"formant"}, 200, 4000, true},
	{"grain density (grains)", []string{"density"}, 2, 80, true},
	{"grain spray (grains)", []string{"spray"}, 0, 0.5, false},
	{"breath (vocal)", []string{"breath"}, 0, 1, false},
	{"bow pressure (waveguide)", []string{"bowPressure"}, 0.2, 0.9, false},
	{"mallet hardness (modal)", []string{"hardness"}, 0, 1, false},
	{"tremolo (modal)", []string{"tremoloDepth"}, 0, 1, false},
	{"drive (organ)", []string{"drive"}, 0, 1, false},
	{"drawbar 4' (organ)", []string{"drawbar4"}, 0, 8, false},
	{"noise (subtractive)", []string{"noiseLevel"}, 0, 1, false},
	{"pulse width (subtractive)", []string{"pulseWidth"}, 0.05, 0.95, false},
}

// at returns the setting a part of the way through the range (0-1)
func (d destination) at(part float64) float64 {
	if d.exp {
		return d.min * math.Pow(d.max/d.min, part)
	}
	return d.min + (d.max-d.min)*part
}

// panelRoute is a route chosen on the panel: the destination of a source, and the part of its
// range the controller moves through
type panelRoute struct {
	source    int
	dest      int
	low, high float64 // 0-1
}

// routes returns the keyboard routes of a panel route
func (pr *panelRoute) routes() []keyboard.Route {
	d := destinations[pr.dest]
	var rs []keyboard.Route
	for _, name := range d.names {
		rs = append(rs, keyboard.Route{Name: name, Min: d.at(pr.low), Max: d.at(pr.high), Exp: d.exp})
	}
	return rs
}

// shaping sends what the envelope on the page sets to the synth playing, and keeps it for the next
type shaping struct {
	synth func() psalm.Modular
	sent  map[string]float64
}

func (s *shaping) ReceiveMessage(msg psalm.Message) {
	for k, v := range msg {
		if f, ok := v.(float64); ok {
			s.sent[k] = f
		}
	}
	s.synth().Receive(msg, 0)
}

func (s *shaping) Receive(value any, _ int) {
	if msg, ok := psalm.MsgVal(value); ok {
		s.ReceiveMessage(msg)
	}
}

// replay gives a synth the envelope moved on the page (nothing when it wasn't)
func (s *shaping) replay(m psalm.Modular) {
	if len(s.sent) == 0 {
		return
	}
	msg := psalm.Message{}
	for k, v := range s.sent {
		msg[k] = v
	}
	m.Receive(msg, 0)
}

// short writes a number for the terminal: whole above 100, three digits below
func short(v float64) string {
	if math.Abs(v) >= 100 {
		return strconv.Itoa(int(math.Round(v)))
	}
	return strconv.FormatFloat(v, 'g', 3, 64)
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
	synth := flag.String("synth", "piano", "the synth to start with: piano, subtractive, pulsar, wavetable, fm, dx, organ, waveguide, modal, vocal, pd, westcoast, supersaw, drumsynth or grains")
	preset := flag.String("preset", "", "the preset to start with (empty: felt for the piano, else the first)")
	curve := flag.String("curve", "linear", "the velocity curve: soft, linear, hard or an exponent")
	channel := flag.Int("channel", 0, "the MIDI channel (1-16), 0 is all")
	transpose := flag.Int("transpose", 0, "semitones")
	monitor := flag.Bool("monitor", true, "print every MIDI event")
	var routes []string
	flag.Func("route", "send a controller to any setting of the synths, scaled: source:name:min:max[:exp|:int], the source a controller (1 the mod wheel), bend or pressure (\"1:cutoff:300:8000:exp\"); repeatable", func(text string) error {
		if _, _, err := keyboard.ParseRoute(text); err != nil {
			return err
		}
		routes = append(routes, text)
		return nil
	})
	mono := flag.Bool("mono", false, "play the synths mono legato: one note at a time, sliding between them where the synth has a glide")
	buffer := flag.Int("buffer", 256, "the buffer size (the latency) with portaudio")
	addr := flag.String("addr", "localhost:8087", "the address of the control panel page (empty: no panel)")
	bindingsFile := flag.String("bindings", "keyboard-midi.json", "the file the MIDI learn bindings of the panel are kept in")
	presetDir := flag.String("presets", "keyboard-presets", "the folder the presets saved on the panel's page are kept in")
	record := flag.Bool("record", false, "record what is played from the start, written to a MIDI file in -recordings when it stops")
	recordings := flag.String("recordings", ".", "the folder recordings are written to (recording-<date>-<time>.mid)")
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
	for _, text := range routes { // the routes of a source together
		source, r, _ := keyboard.ParseRoute(text)
		if cfg.Routes == nil {
			cfg.Routes = map[int][]keyboard.Route{}
		}
		cfg.Routes[source] = append(cfg.Routes[source], r)
	}
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
	if *mono {
		for _, in := range instruments {
			for _, synth := range in.synths {
				synth.Receive(psalm.Message{"mono": 1.0, "legato": 1.0}, 0) // the poly synths take it, the organ and the piano don't
			}
		}
	}
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
	// the routes: the -route flags, and the ones chosen on the panel (a source routed on the panel
	// replaces its flags; "built in" goes back to them)
	flagRoutes := cfg.Routes
	cfg.Routes = nil
	panelRoutes := []*panelRoute{{source: midi.CCModWheel, high: 1}, {source: keyboard.RoutePressure, high: 1}}
	keysFor := func(in *instrument) keyboard.Config {
		c := in.keys(cfg)
		c.Routes = maps.Clone(flagRoutes)
		if c.Routes == nil {
			c.Routes = map[int][]keyboard.Route{}
		}
		for _, pr := range panelRoutes {
			if pr.dest > 0 {
				c.Routes[pr.source] = pr.routes()
			}
		}
		return c
	}
	// what is played on the keyboard (not typed in the terminal) can be recorded to a MIDI file
	recorder := midi.Record(input)
	kb := keyboard.New(merged{recorder, commands}, keysFor(in), synthOf(in))
	reroute := func() { kb.SetConfig(keysFor(instruments[current])) } // on the audio thread

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
	shaped := &shaping{synth: func() psalm.Modular { return synthOf(instruments[current]) }, sent: map[string]float64{}}

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
		kb.SetConfig(keysFor(in))
		kb.SetTargets(synthOf(in))
		shaped.replay(synthOf(in)) // the envelope moved on the page, on the new synth too
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

		// a keyboard on the page (the computer's keys too: A W S E D ...), played as MIDI: the curve, the
		// routes and the switches apply as to the real one; the amplitude envelope of the synth playing
		ctl.Section("Play")
		widgets.Keys(ctl, "Page keys", func(note int, velocity float64, on bool) {
			ch := max(*channel, 1)
			if on {
				commands.Push(midi.NoteOnMsg(ch, note, max(1, int(velocity*127))))
			} else {
				commands.Push(midi.NoteOffMsg(ch, note, 0))
			}
		}, widgets.From(48), widgets.Octaves(3))
		widgets.Envelope(ctl, "Amp envelope", shaped, "amp", widgets.ADSR{Attack: 5, Decay: 300, Sustain: 0.7, Release: 400}, panel.NoDefault())

		ctl.Section("Keys")
		ctl.Fader("Transpose", -24, 24, panel.Linear, panel.Func(func(v float64) {
			cfg.Transpose = int(v)
			reroute() // the keys with the routes
		}), panel.Default(float64(*transpose)), panel.Step(1), panel.Unit("st"))
		curves := []keyboard.Curve{keyboard.Soft, keyboard.Linear, keyboard.Hard}
		curveNames := []string{"soft", "linear", "hard"}
		if !slices.Contains(curves, cfg.Curve) { // an exponent of its own from -curve
			curves = append(curves, cfg.Curve)
			curveNames = append(curveNames, *curve)
		}
		ctl.Select("Velocity curve", curveNames, func(i int) {
			cfg.Curve = curves[i]
			reroute() // the keys with the routes
		}, panel.Default(float64(slices.Index(curves, cfg.Curve))))

		// the routes: the mod wheel and the aftertouch sent to a setting of the synth, the part of
		// its range they move through
		ctl.Section("Routes")
		names := make([]string, len(destinations))
		for i, d := range destinations {
			names[i] = d.label
		}
		for _, pr := range panelRoutes {
			label := "Wheel"
			if pr.source == keyboard.RoutePressure {
				label = "Aftertouch"
			}
			last := label + ": built in"
			changed := func() { // the route chosen, in the terminal too (once it changes)
				reroute()
				now := label + ": built in"
				if pr.dest > 0 {
					d := destinations[pr.dest]
					now = fmt.Sprintf("%s: %s, %s to %s", label, d.label, short(d.at(pr.low)), short(d.at(pr.high)))
				}
				if now != last {
					log.Print(now)
					last = now
				}
			}
			ctl.Select(label+" to", names, func(i int) { pr.dest = i; changed() })
			ctl.Knob(label+" from", 0, 100, panel.Linear, panel.Func(func(v float64) { pr.low = v / 100; changed() }),
				panel.Default(0), panel.Unit("%"), panel.Step(1), panel.Label("from"))
			ctl.Knob(label+" up to", 0, 100, panel.Linear, panel.Func(func(v float64) { pr.high = v / 100; changed() }),
				panel.Default(100), panel.Unit("%"), panel.Step(1), panel.Label("up to"))
		}

		ctl.Section("Record")
		recordOn := 0.0
		if *record {
			recordOn = 1
		}
		ctl.Toggle("Record", panel.Func(func(v float64) {
			switch {
			case v > 0.5 && !recorder.Recording():
				recorder.Start()
				log.Print("recording")
			case v <= 0.5 && recorder.Recording():
				recorder.Stop()
				bpm := kb.Env().BPM()
				go saveRecording(recorder, bpm, *recordings) // not on the audio thread
			}
		}), panel.Default(recordOn))

		ctl.Section("Mix")
		ctl.Fader("Volume", -40, 6, panel.Linear, panel.Message(mix.Master(), "volume"), panel.Unit("dB"), panel.Default(0), panel.Smooth(0.03))
		ctl.Fader("Hall", -40, 6, panel.Linear, panel.Message(mix.Strip("hall"), "volume"), panel.Unit("dB"), panel.Default(0), panel.Smooth(0.03))
		ctl.Meter("Master", mix.Master())
		ctl.Meter("Hall out", mix.Strip("hall"))

		ctl.MIDIControllers(kb) // the knobs; the keys play the synth (MIDI learn doesn't take them)
		if err := ctl.KeepBindings(*bindingsFile); err != nil {
			log.Fatal(err)
		}
		if err := ctl.Presets(*presetDir); err != nil { // a sound with its keys and mix, saved by name
			log.Fatal(err)
		}
		go func() {
			if err := ctl.Serve(*addr); err != nil {
				log.Fatal(err)
			}
		}()
		defer ctl.Close()
		log.Printf("the control panel: http://%s", *addr)
	}

	go readCommands(commands, max(*channel, 1), instruments, &playing)

	log.Printf("%s: %s, %s velocity curve; bank select or a synth name switches between %v, a program change or a preset name the preset (\"list\" shows them)",
		in.name, in.presets[in.selected.Load()], *curve, names(instruments))
	if *record {
		recorder.Start()
		log.Printf("recording; written to %s when it stops", *recordings)
	}
	if err := p.Listen(); err != nil {
		log.Fatal(err)
	}
	if recorder.Recording() {
		recorder.Stop()
		saveRecording(recorder, kb.Env().BPM(), *recordings)
	}
}

// saveRecording writes what was recorded to a MIDI file named by the time, at a tempo
func saveRecording(r *midi.Recorder, bpm float64, dir string) {
	if r.Len() == 0 {
		log.Print("nothing recorded")
		return
	}
	path := filepath.Join(dir, "recording-"+time.Now().Format("20060102-150405")+".mid")
	if err := r.File(bpm).WriteFile(path); err != nil {
		log.Printf("the recording: %v", err)
		return
	}
	log.Printf("recorded %d messages to %s (play it: go run ./cmd/examples/midifile %s)", r.Len(), path, path)
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
		build("dx", slices.Collect(maps.Keys(dx.Presets)), func(n string) psalm.Modular {
			return dx.NewPoly(dx.Presets[n](), 12, 2).AddTo(p)
		}, velocityKeys, 0.3),
		build("organ", slices.Collect(maps.Keys(organ.Presets)), func(n string) psalm.Modular {
			return organ.New(organ.Presets[n]()).AddTo(p)
		}, velocityKeys, 0.25),
		build("waveguide", slices.Collect(maps.Keys(waveguide.Presets)), func(n string) psalm.Modular {
			return waveguide.NewPoly(waveguide.Presets[n](), 12).AddTo(p)
		}, velocityKeys, 0.3),
		build("modal", slices.Collect(maps.Keys(modal.Presets)), func(n string) psalm.Modular {
			return modal.NewPoly(modal.Presets[n](), 16).AddTo(p)
		}, velocityKeys, 0.3),
		build("vocal", slices.Collect(maps.Keys(vocal.Presets)), func(n string) psalm.Modular {
			return vocal.NewPoly(vocal.Presets[n](), 8).AddTo(p)
		}, synthKeys, 0.35),
		build("pd", slices.Collect(maps.Keys(pd.Presets)), func(n string) psalm.Modular {
			return pd.NewPoly(pd.Presets[n](), 12).AddTo(p)
		}, synthKeys, 0.3),
		build("westcoast", slices.Collect(maps.Keys(westcoast.Presets)), func(n string) psalm.Modular {
			return westcoast.NewPoly(westcoast.Presets[n](), 12).AddTo(p)
		}, velocityKeys, 0.3),
		build("supersaw", slices.Collect(maps.Keys(supersaw.Presets)), func(n string) psalm.Modular {
			return supersaw.NewPoly(supersaw.Presets[n](), 12).AddTo(p)
		}, synthKeys, 0.3),
		build("drumsynth", slices.Collect(maps.Keys(drumsynth.Kits)), func(n string) psalm.Modular {
			return drumsynth.NewPoly(drumsynth.Config{Kit: drumsynth.Kits[n]()}, 16).AddTo(p)
		}, velocityKeys, 0.15),
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
