// Midifile: play a MIDI file on the synths, in the Musikverein.
//
// Every channel of the file gets a synth of its own, chosen by the General MIDI program it plays: a
// piano for the pianos, fm for the electric pianos, the harpsichord and the brass and reeds, the
// organ for the organs, waveguide for the guitars and the bowed strings, modal for the mallets and
// bells, vocal for the choirs, subtractive, supersaw, wavetable and pd for the synth leads and
// pads, pulsar and westcoast for the effects; channel 10 plays the drumsynth (-kit). A program
// change in the middle of the file switches the synth of its channel. With -orchestra (the folder
// of the Virtual Playing Orchestra) the strings, brass, woodwinds, choirs, harp, timpani and
// mallets play its samples instead. -synth gives a channel a
// synth of your own choosing (any synth and preset): -synth 2=supersaw/hoover. Each channel plays
// through a keyboard (modules/controls/keyboard), so the velocity, the sustain pedal, the pitch bend,
// the mod wheel and the aftertouch of the file all play; its volume (controller 7), expression (11),
// pan (10) and reverb send (91) set its strip on the mixer.
//
// A control panel runs alongside (http://localhost:8087, -addr to move it, empty to leave it
// out): play, stop, back to the start, where in the file (drag to jump), the speed, loop; for every
// channel its sound (as the file has it, or any synth and preset, drum kit or orchestra
// instrument), a mute and a volume; the master and the hall, and meters.
//
// Without a file it plays the first page of Bach's Prelude in C (BWV 846) on a piano with a cello
// under it, written as a MIDI file in memory (-write saves it).
//
//	go run ./cmd/examples/midifile song.mid
//	go run ./cmd/examples/midifile -list song.mid           the tracks, the channels and their synths
//	go run ./cmd/examples/midifile -synth 1=supersaw/lead -kit 909 -loop song.mid
//	go run ./cmd/examples/midifile -speed 0.8 -start 30 song.mid
//	go run ./cmd/examples/midifile -tempo 128 song.mid        every beat at 128
//	go run ./cmd/examples/midifile -clock /dev/snd/midiC3D0 song.mid   in step with a sequencer's clock
//	go run ./cmd/examples/midifile -orchestra assets/Virtual-Playing-Orchestra3 song.mid
//	go run ./cmd/examples/midifile -synth 3=sfz/Strings/harp-sustain -orchestra assets/Virtual-Playing-Orchestra3 song.mid
//	go run ./cmd/examples/midifile -render song.wav song.mid
//	go run ./cmd/examples/midifile -write prelude.mid       the prelude as a file
package main

import (
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
	"time"

	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/keyboard"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/drumsynth"
	"github.com/almerlucke/psalm/synths/dx"
	"github.com/almerlucke/psalm/synths/fm"
	"github.com/almerlucke/psalm/synths/modal"
	"github.com/almerlucke/psalm/synths/organ"
	"github.com/almerlucke/psalm/synths/pd"
	"github.com/almerlucke/psalm/synths/piano"
	"github.com/almerlucke/psalm/synths/pulsar"
	"github.com/almerlucke/psalm/synths/sampler"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/synths/supersaw"
	"github.com/almerlucke/psalm/synths/vocal"
	"github.com/almerlucke/psalm/synths/waveguide"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
	"github.com/almerlucke/psalm/synths/westcoast"
	"github.com/almerlucke/psalm/tools/render"
	"github.com/almerlucke/sndfile"
)

// generalMIDI is the synth and preset that plays each General MIDI program
var generalMIDI = [128]string{
	// piano
	"piano/grand", "piano/bright", "piano/vintage", "piano/honkyTonk", "fm/ePiano", "dx/glassKeys", "fm/harpsichord", "fm/clavinet",
	// chromatic percussion
	"fm/celesta", "modal/glockenspiel", "fm/musicBox", "modal/vibraphone", "modal/marimba", "modal/xylophone", "modal/tubularBells", "waveguide/steel",
	// organ
	"organ/jazz", "organ/gospel", "organ/rock", "organ/full", "organ/reed", "fm/accordion", "organ/reed", "fm/accordion",
	// guitar
	"waveguide/guitar", "waveguide/steel", "waveguide/muted", "waveguide/steel", "waveguide/muted", "fm/overdriveGuitar", "fm/overdriveGuitar", "waveguide/harp",
	// bass
	"waveguide/bass", "waveguide/bass", "fm/bass", "fm/fretlessBass", "fm/slapBass", "fm/slapBass", "subtractive/bass", "pd/resoBass",
	// strings
	"waveguide/violin", "waveguide/violin", "waveguide/cello", "waveguide/cello", "waveguide/slowStrings", "waveguide/pizzicato", "waveguide/harp", "fm/timpani",
	// ensemble
	"fm/strings", "waveguide/slowStrings", "subtractive/strings", "pd/strings", "vocal/aah", "vocal/ooh", "vocal/choir", "supersaw/stab",
	// brass
	"fm/trumpet", "pd/brass", "pd/brass", "fm/trumpet", "fm/frenchHorn", "dx/softBrass", "subtractive/brass", "wavetable/brass",
	// reed
	"fm/saxophone", "fm/saxophone", "fm/saxophone", "fm/saxophone", "fm/oboe", "fm/oboe", "fm/clarinet", "fm/clarinet",
	// pipe
	"fm/flute", "fm/flute", "organ/flute", "fm/panFlute", "subtractive/wind", "subtractive/flute", "organ/whistle", "subtractive/flute",
	// synth lead
	"subtractive/chipLead", "supersaw/lead", "organ/flute", "subtractive/lead", "pd/lead", "vocal/solo", "wavetable/lead", "subtractive/ladderLead",
	// synth pad
	"wavetable/pad", "subtractive/pad", "supersaw/trancePad", "vocal/choir", "waveguide/ghostBow", "pd/glassPad", "vocal/angels", "wavetable/sweep",
	// synth effects
	"pulsar/raindrops", "wavetable/drone", "modal/crystal", "subtractive/hollowPad", "wavetable/glass", "pulsar/insects", "modal/spaceTine", "westcoast/laserPing",
	// ethnic
	"fm/sitar", "waveguide/steel", "fm/koto", "fm/koto", "modal/kalimba", "organ/reed", "waveguide/violin", "fm/oboe",
	// percussive
	"modal/glockenspiel", "westcoast/bell", "modal/steelPan", "modal/woodBlock", "fm/taiko", "westcoast/bongo", "subtractive/synthTom", "subtractive/riser",
	// sound effects
	"waveguide/scrape", "vocal/whisper", "westcoast/livingDrone", "pulsar/insects", "westcoast/laserPing", "pulsar/motor", "pulsar/crackle", "westcoast/metal",
}

// orchestral are the programs the Virtual Playing Orchestra plays with -orchestra: its SFZ files,
// in the folder of the library (the sustained articulations, which take the velocity)
var orchestral = map[int]string{
	8: "Keys/celesta", 9: "Percussion/glockenspiel", 11: "Percussion/vibraphone-auto-damp",
	13: "Percussion/xylophone", 14: "Percussion/tubular-bells",
	40: "Strings/1st-violin-SOLO-sustain", 41: "Strings/viola-SOLO-sustain", 42: "Strings/cello-SOLO-sustain",
	43: "Strings/bass-SOLO-sustain", 44: "Strings/all-strings-SEC-tremolo", 45: "Strings/all-strings-SEC-pizzicato",
	46: "Strings/harp-sustain", 47: "Percussion/timpani-hit",
	48: "Strings/all-strings-SEC-sustain", 49: "Strings/all-strings-SEC-sustain",
	52: "Vocals/choir-MIXED-sustain", 53: "Vocals/choir-FEMALE-sustain",
	56: "Brass/trumpet-SOLO-sustain", 57: "Brass/trombone-SOLO-sustain", 58: "Brass/tuba-SOLO-sustain",
	59: "Brass/trumpet-SOLO-staccato", 60: "Brass/french-horn-SOLO-sustain", 61: "Brass/all-brass-SEC-sustain",
	68: "Woodwinds/oboe-SOLO-sustain", 69: "Woodwinds/english-horn-SOLO-sustain",
	70: "Woodwinds/bassoon-SOLO-sustain", 71: "Woodwinds/clarinet-SOLO-sustain",
	72: "Woodwinds/piccolo-SOLO-sustain", 73: "Woodwinds/flute-SOLO-sustain",
}

// orchestraDir is the folder of the Virtual Playing Orchestra (-orchestra), relative SFZ paths are
// in it
var orchestraDir string

// sound is a synth made to play a channel: how the keyboard plays it, how much goes to the hall
type sound struct {
	name  string // synth/preset
	synth psalm.Modular
	keys  func(keyboard.Config) keyboard.Config
	hall  float64
}

// expressive are the hands on the synths: the mod wheel (controller 1); the bend and the aftertouch
// go along
var expressive = map[int]string{midi.CCModWheel: "modWheel"}

// the piano takes the velocity itself (so the amplitude stays 1) and has its pedals
func pianoKeys(cfg keyboard.Config) keyboard.Config {
	cfg.Dynamics, cfg.Controls, cfg.HoldNotes = 0, nil, false
	return cfg
}

// most synths play the velocity as the amplitude, the sustain pedal holds their notes
func synthKeys(cfg keyboard.Config) keyboard.Config {
	cfg.Dynamics, cfg.Controls, cfg.HoldNotes, cfg.PitchBend, cfg.Pressure, cfg.PolyPressure = 1, expressive, true, "pitchBend", "pressure", "pressure"
	return cfg
}

// fm, dx and the physical models take the velocity themselves
func velocityKeys(cfg keyboard.Config) keyboard.Config {
	cfg.Dynamics, cfg.Controls, cfg.HoldNotes, cfg.PitchBend, cfg.Pressure, cfg.PolyPressure = 0, expressive, true, "pitchBend", "pressure", "pressure"
	return cfg
}

// drums are hits: they play to their end, the note offs of the file don't choke them
func drumKeys(cfg keyboard.Config) keyboard.Config {
	cfg = velocityKeys(cfg)
	cfg.Hits, cfg.HoldNotes = true, false
	return cfg
}

var wavetableTable = gen.Brightness(32)

// newSound makes a synth by its name, "synth/preset" (a drum kit for the drumsynth), not yet in a
// patch
func newSound(name string) (*sound, error) {
	synth, preset, _ := strings.Cut(name, "/")
	missing := func(presets []string) error {
		slices.Sort(presets)
		return fmt.Errorf("%s has no preset %q: %s", synth, preset, strings.Join(presets, ", "))
	}
	s := &sound{name: name, keys: velocityKeys, hall: 0.3}
	switch synth {
	case "sfz":
		// a sampled instrument: an SFZ file (in the orchestra's folder when not absolute)
		path := preset
		if !strings.HasSuffix(path, ".sfz") {
			path += ".sfz"
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(orchestraDir, path)
		}
		inst, err := sampler.LoadSFZ(path)
		if err != nil {
			return nil, err
		}
		s.synth, s.hall = sampler.NewPoly(sampler.DefaultConfig(inst), 32), 0.25
		// it takes the velocity itself and every controller as itself ("cc1" the mod wheel); without
		// a pedal of its own the keyboard holds the notes
		controls := map[int]string{}
		for cc := range 120 {
			if cc != midi.CCSustain || !inst.NoSustain {
				controls[cc] = fmt.Sprintf("cc%d", cc)
			}
		}
		s.keys = func(cfg keyboard.Config) keyboard.Config {
			cfg.Dynamics, cfg.Controls, cfg.HoldNotes, cfg.PitchBend, cfg.Pressure = 0, controls, inst.NoSustain, "pitchBend", "pressure"
			return cfg
		}
	case "piano":
		f, ok := piano.Presets[preset]
		if !ok {
			return nil, missing(slices.Collect(maps.Keys(piano.Presets)))
		}
		s.synth, s.keys, s.hall = piano.NewPoly(f(), 32), pianoKeys, 0.4
	case "fm":
		f, ok := fm.Presets[preset]
		if !ok {
			return nil, missing(slices.Collect(maps.Keys(fm.Presets)))
		}
		s.synth = fm.NewPoly(f(), 16)
	case "dx":
		f, ok := dx.Presets[preset]
		if !ok {
			return nil, missing(slices.Collect(maps.Keys(dx.Presets)))
		}
		s.synth = dx.NewPoly(f(), 16, 2)
	case "organ":
		f, ok := organ.Presets[preset]
		if !ok {
			return nil, missing(slices.Collect(maps.Keys(organ.Presets)))
		}
		s.synth, s.hall = organ.New(f()), 0.25
	case "waveguide":
		f, ok := waveguide.Presets[preset]
		if !ok {
			return nil, missing(slices.Collect(maps.Keys(waveguide.Presets)))
		}
		s.synth = waveguide.NewPoly(f(), 12)
	case "modal":
		f, ok := modal.Presets[preset]
		if !ok {
			return nil, missing(slices.Collect(maps.Keys(modal.Presets)))
		}
		s.synth = modal.NewPoly(f(), 16)
	case "westcoast":
		f, ok := westcoast.Presets[preset]
		if !ok {
			return nil, missing(slices.Collect(maps.Keys(westcoast.Presets)))
		}
		s.synth = westcoast.NewPoly(f(), 12)
	case "subtractive":
		f, ok := subtractive.Presets[preset]
		if !ok {
			return nil, missing(slices.Collect(maps.Keys(subtractive.Presets)))
		}
		s.synth, s.keys = subtractive.NewPoly(f(), 12), synthKeys
	case "supersaw":
		f, ok := supersaw.Presets[preset]
		if !ok {
			return nil, missing(slices.Collect(maps.Keys(supersaw.Presets)))
		}
		s.synth, s.keys = supersaw.NewPoly(f(), 12), synthKeys
	case "wavetable":
		f, ok := wavetable.Presets[preset]
		if !ok {
			return nil, missing(slices.Collect(maps.Keys(wavetable.Presets)))
		}
		s.synth, s.keys = wavetable.NewPoly(f(wavetableTable), 12), synthKeys
	case "pd":
		f, ok := pd.Presets[preset]
		if !ok {
			return nil, missing(slices.Collect(maps.Keys(pd.Presets)))
		}
		s.synth, s.keys = pd.NewPoly(f(), 12), synthKeys
	case "pulsar":
		f, ok := pulsar.Presets[preset]
		if !ok {
			return nil, missing(slices.Collect(maps.Keys(pulsar.Presets)))
		}
		s.synth, s.keys = pulsar.NewPoly(f(), 12), synthKeys
	case "vocal":
		f, ok := vocal.Presets[preset]
		if !ok {
			return nil, missing(slices.Collect(maps.Keys(vocal.Presets)))
		}
		s.synth, s.keys, s.hall = vocal.NewPoly(f(), 8), synthKeys, 0.35
	case "drumsynth":
		f, ok := drumsynth.Kits[preset]
		if !ok {
			return nil, missing(slices.Collect(maps.Keys(drumsynth.Kits)))
		}
		s.synth, s.keys, s.hall = drumsynth.NewPoly(drumsynth.Config{Kit: f()}, 16), drumKeys, 0.12
	default:
		return nil, fmt.Errorf("no synth %q: sfz, piano, fm, dx, organ, waveguide, modal, westcoast, subtractive, supersaw, wavetable, pd, pulsar, vocal or drumsynth", synth)
	}
	return s, nil
}

// part is a channel of the file: its synths (one for each program it plays), its keyboard, its
// strip on the mixer, the volume the file gives it and the one of the panel
type part struct {
	channel  int
	sounds   map[int]*sound    // by program
	order    []int             // the programs, in the order they come
	built    map[string]*sound // every sound made for it, by name (on the mixer)
	program  int               // the program the file plays now
	fixed    *sound            // the sound chosen on the panel, nil: as the file has it
	wanted   string            // the sound chosen last on the panel (it may still be made)
	playing  *sound
	kb       *keyboard.Keyboard
	group    *mixer.Strip
	volume   float64 // controller 7, 0-1
	express  float64 // controller 11, 0-1
	trim     float64 // dB, from the panel
	hallSent bool    // the file sends to the hall itself (controller 91)
}

// use plays the channel on another sound (on the audio thread): the notes of the one left let go,
// the pedals of a piano left up
func (pt *part) use(s *sound, tick psalm.CtrlTick) {
	if s == nil || s == pt.playing {
		return
	}
	pt.kb.AllNotesOff(tick)
	pt.playing.synth.Receive(psalm.Message{"sustain": 0.0, "sostenuto": 0.0, "soft": 0.0}, 0)
	pt.playing = s
	pt.kb.SetConfig(s.keys(keyboard.Config{Channel: pt.channel}))
	pt.kb.SetTargets(s.synth)
	if !pt.hallSent {
		pt.group.SetSend("hall", s.hall)
	}
}

// gain sets the strip of the channel: the volume and the expression of the file (as General MIDI
// has them, the square of each) and the trim of the panel
func (pt *part) gain() {
	db := 40*math.Log10(math.Max(pt.volume, 1e-3)) + 40*math.Log10(math.Max(pt.express, 1e-3)) + pt.trim
	pt.group.SetVolume(db)
}

// describe returns what the part plays
func (pt *part) describe() string {
	var s []string
	for _, prog := range pt.order {
		if pt.channel == midi.DrumChannel {
			s = append(s, "drums: "+pt.sounds[prog].name)
			continue
		}
		s = append(s, fmt.Sprintf("%s: %s", midi.ProgramName(prog), pt.sounds[prog].name))
	}
	return strings.Join(s, ", ")
}

func main() {
	list := flag.Bool("list", false, "show the tracks, the channels and the synths that play them, and stop")
	renderTo := flag.String("render", "", "render to this WAV file instead of playing")
	writeTo := flag.String("write", "", "write the file played (the prelude, without a file) to this MIDI file")
	loop := flag.Bool("loop", false, "start over at the end")
	speed := flag.Float64("speed", 1, "how fast to play: 1 at the tempo of the file")
	tempo := flag.Float64("tempo", 0, "play every beat at this tempo (quarter notes a minute) instead of the file's; 0 is the file's")
	clockDevice := flag.String("clock", "", "follow the MIDI clock of a raw MIDI device (a sequencer, a drum machine): its tempo, start, stop and continue")
	start := flag.Float64("start", 0, "seconds into the file to start at")
	orchestra := flag.String("orchestra", "", "the folder of the Virtual Playing Orchestra (assets/Virtual-Playing-Orchestra3): its samples play the strings, brass, woodwinds, choirs, harp, timpani and mallets")
	kit := flag.String("kit", "808", "the drum kit of channel 10: 808, 909, electro, trap or industrial")
	addr := flag.String("addr", "localhost:8087", "the address of the control panel page (empty: no panel)")
	buffer := flag.Int("buffer", 512, "the buffer size")
	monitor := flag.Bool("monitor", false, "print every MIDI event played")
	overrides := map[int]string{}
	flag.Func("synth", "play a channel on a synth of your choosing: channel=synth/preset (\"2=supersaw/hoover\", \"10=drumsynth/909\"); repeatable", func(text string) error {
		ch, name, ok := strings.Cut(text, "=")
		c, err := strconv.Atoi(ch)
		if !ok || err != nil || c < 1 || c > 16 {
			return fmt.Errorf("%q: channel=synth/preset, the channel 1-16", text)
		}
		if synth, _, _ := strings.Cut(name, "/"); synth != "sfz" { // an SFZ file is loaded when it plays
			if _, err := newSound(name); err != nil {
				return err
			}
		}
		overrides[c] = name
		return nil
	})
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: midifile [flags] [file.mid]\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	orchestraDir = *orchestra

	var file *midi.File
	title := "Prelude in C"
	if flag.NArg() > 0 {
		var err error
		if file, err = midi.ReadFile(flag.Arg(0)); err != nil {
			log.Fatal(err)
		}
		title = flag.Arg(0)
	} else {
		file = prelude()
	}
	if *writeTo != "" {
		if err := file.WriteFile(*writeTo); err != nil {
			log.Fatal(err)
		}
		log.Printf("written to %s", *writeTo)
		if flag.NArg() == 0 && *renderTo == "" && !*list {
			return
		}
	}
	channels := file.Channels()
	if len(channels) == 0 {
		log.Fatalf("%s plays no notes", title)
	}

	// the sound of every program of every channel
	soundOf := func(ch, program int) string {
		if name, ok := overrides[ch]; ok {
			return name
		}
		if ch == midi.DrumChannel {
			return "drumsynth/" + *kit
		}
		if path, ok := orchestral[program&127]; ok && *orchestra != "" {
			return "sfz/" + path
		}
		return generalMIDI[program&127]
	}
	if *list {
		describe(file, title, soundOf)
		return
	}

	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: *buffer})
	var p *psalm.Psalm
	if *renderTo != "" {
		p = psalm.New(nil)
	} else {
		p = psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	}
	defer p.Close()

	mix := mixer.New(p)
	vienna := convolver.IRFromSoundFile(sndfile.MustSoundFile[float32]("assets/IR44100/Musikvereinsaal.wav"))
	mix.Bus("hall", mixer.StripConfig{}, convolver.New(vienna, convolver.Config{PartitionSize: 512, Mix: 1, Gain: 0.7}).AddTo(p))

	player := midi.NewPlayer(file)
	player.SetLoop(*loop)
	player.SetSpeed(*speed)
	player.SetTempo(*tempo)
	inputs := midi.Tee(player, len(channels))
	// the synths are calibrated to the same loudness: the more channels, the lower each, so a busy
	// file leaves the limiter of the master little to do (two channels at -6 dB, eight at -12)
	level := -3 - 10*math.Log10(float64(len(channels)))
	var parts []*part
	for i, ch := range channels {
		pt := &part{channel: ch, sounds: map[int]*sound{}, built: map[string]*sound{}, volume: 100.0 / 127, express: 1}
		programs := file.Programs(ch)
		if len(programs) == 0 {
			programs = []int{0}
		}
		pt.program = programs[0]
		byName := pt.built // two programs played by the same synth share it
		for _, prog := range programs {
			name := soundOf(ch, prog)
			s, ok := byName[name]
			if !ok {
				var err error
				if s, err = newSound(name); err != nil {
					log.Fatal(err)
				}
				p.Add(s.synth)
				byName[name] = s
			}
			pt.sounds[prog] = s
			pt.order = append(pt.order, prog)
		}
		pt.playing = pt.sounds[programs[0]]
		group := fmt.Sprintf("channel %d", ch)
		pt.group = mix.Group(group, mixer.StripConfig{Sends: map[string]float64{"hall": pt.playing.hall}})
		for _, name := range slices.Sorted(maps.Keys(byName)) {
			mix.Channel(group+" "+name, byName[name].synth, mixer.StripConfig{Volume: level, Group: group})
		}
		pt.gain()

		pt.kb = keyboard.New(inputs[i], pt.playing.keys(keyboard.Config{Channel: ch}), pt.playing.synth)
		pt.kb.OnProgram(func(program int, tick psalm.CtrlTick) {
			if _, ok := pt.sounds[program]; ok {
				pt.program = program
			}
			if pt.fixed == nil { // a sound chosen on the panel stays
				pt.use(pt.sounds[pt.program], tick)
			}
		})
		pt.kb.OnControl(midi.CCVolume, func(v float64, _ psalm.CtrlTick) { pt.volume = v; pt.gain() })
		pt.kb.OnControl(midi.CCExpression, func(v float64, _ psalm.CtrlTick) { pt.express = v; pt.gain() })
		pt.kb.OnControl(midi.CCPan, func(v float64, _ psalm.CtrlTick) { pt.group.SetPan(math.Max(-1, (v*127-64)/63)) })
		pt.kb.OnControl(91, func(v float64, _ psalm.CtrlTick) { pt.hallSent = true; pt.group.SetSend("hall", v) })
		if *monitor {
			pt.kb.OnEvent(func(e midi.Event, tick psalm.CtrlTick) {
				if e.Message.IsChannel() {
					log.Printf("%7.3f s  %v", float64(tick.SampleTimestamp)/psalm.SampleRate(), e.Message)
				}
			})
		}
		pt.kb.AddTo(p)
		parts = append(parts, pt)
	}
	p.Out(mix.Master())

	// a MIDI clock to follow: its tempo (the keyboard sets the patch's from it), its start (from the
	// top), stop and continue
	if *clockDevice != "" && *renderTo == "" {
		d, err := midi.OpenDevice(*clockDevice)
		if err != nil {
			log.Fatal(err)
		}
		defer d.Close()
		clock := keyboard.New(d, keyboard.Config{FollowClock: true, Controls: map[int]string{}})
		clock.OnEvent(func(e midi.Event, _ psalm.CtrlTick) {
			switch e.Message.Kind() {
			case midi.Start:
				player.Seek(*start)
				player.Play()
			case midi.Continue:
				player.Play()
			case midi.Stop:
				player.Stop()
			}
		})
		clock.AddTo(p)
		player.FollowTempo(func() float64 { return clock.Env().BPM() })
		log.Printf("following the clock of %s: start it there", d.Path())
	}

	describe(file, title, soundOf)
	if *start > 0 {
		player.Seek(*start)
	}
	if *clockDevice == "" || *renderTo != "" {
		player.Play()
	}

	if *renderTo != "" {
		seconds := player.PlayTime(*start)
		if *loop {
			seconds *= 2 // twice through
		}
		stats, err := render.ToFile(p, *renderTo, render.Options{Seconds: seconds, Tail: 4})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(stats)
		fmt.Printf("the limiter took the loudest peaks down %.1f dB\n", math.Abs(mix.Master().Reduction()))
		return
	}

	if *addr != "" {
		ctl := panel.New(p, "MIDI file")

		ctl.Section("Transport")
		ctl.Button("Play", player.Play)
		ctl.Button("Stop", player.Stop)
		ctl.Button("Back to the start", func() { player.Seek(0) })
		// where the file is: it follows the player, dragged further than a second it jumps
		position := ctl.Fader("Position", 0, math.Max(1, math.Ceil(player.Duration())), panel.Linear, panel.Func(func(v float64) {
			if math.Abs(v-player.Position()) > 1.5 {
				player.Seek(v)
			}
		}), panel.Unit("s"), panel.Step(1), panel.Default(*start))
		go func() {
			for range time.Tick(500 * time.Millisecond) {
				position.Set(player.Position())
			}
		}()
		ctl.Knob("Speed", 0.25, 2, panel.Exp, panel.Func(player.SetSpeed), panel.Default(player.Speed()), panel.Step(0.01), panel.Unit("×"))
		loopOn := 0.0
		if *loop {
			loopOn = 1
		}
		ctl.Toggle("Loop", panel.Func(func(v float64) { player.SetLoop(v > 0.5) }), panel.Default(loopOn))
		if *clockDevice == "" {
			ctl.Knob("Tempo", 0, 240, panel.Linear, panel.Func(player.SetTempo), panel.Default(*tempo), panel.Step(1),
				panel.Unit("bpm"), panel.Label("tempo (0: the file's)"))
		}

		ctl.Section("Channels")
		choices := append([]string{"as the file has it"}, soundNames()...)
		for _, pt := range parts {
			label := fmt.Sprintf("%d %s", pt.channel, pt.describe())
			// another sound for the channel: made off the audio thread the first time (an SFZ file
			// takes a moment to load), then put on the mixer and played between two buffers
			ctl.Select(fmt.Sprintf("Sound %d", pt.channel), choices, func(i int) {
				if i == 0 {
					pt.wanted, pt.fixed = "", nil
					pt.use(pt.sounds[pt.program], psalm.CtrlTick{})
					return
				}
				name := choices[i]
				pt.wanted = name
				if s, ok := pt.built[name]; ok {
					pt.fixed = s
					pt.use(s, psalm.CtrlTick{})
					return
				}
				go func() {
					s, err := newSound(name)
					if err != nil {
						log.Printf("channel %d: %v", pt.channel, err)
						return
					}
					p.Do(func() {
						if _, ok := pt.built[name]; !ok {
							p.Add(s.synth)
							mix.Channel(fmt.Sprintf("channel %d %s", pt.channel, name), s.synth, mixer.StripConfig{Volume: level, Group: fmt.Sprintf("channel %d", pt.channel)})
							mix.Align()
							pt.built[name] = s
						}
						if pt.wanted == name {
							pt.fixed = pt.built[name]
							pt.use(pt.fixed, psalm.CtrlTick{})
							log.Printf("channel %d: %s", pt.channel, name)
						}
					})
				}()
			}, panel.Label("sound "+strconv.Itoa(pt.channel)))
			ctl.Toggle(fmt.Sprintf("Mute %d", pt.channel), panel.Func(func(v float64) { player.MuteChannel(pt.channel, v > 0.5) }),
				panel.Label("mute "+label))
			ctl.Fader(fmt.Sprintf("Volume %d", pt.channel), -30, 12, panel.Linear, panel.Func(func(v float64) { pt.trim = v; pt.gain() }),
				panel.Label("volume "+strconv.Itoa(pt.channel)), panel.Unit("dB"), panel.Default(0), panel.Step(0.5))
		}

		ctl.Section("Mix")
		ctl.Fader("Volume", -40, 6, panel.Linear, panel.Message(mix.Master(), "volume"), panel.Unit("dB"), panel.Default(0), panel.Smooth(0.03))
		ctl.Fader("Hall", -40, 6, panel.Linear, panel.Message(mix.Strip("hall"), "volume"), panel.Unit("dB"), panel.Default(0), panel.Smooth(0.03))
		ctl.Meter("Master", mix.Master())
		ctl.Meter("Hall out", mix.Strip("hall"))
		go func() {
			if err := ctl.Serve(*addr); err != nil {
				log.Fatal(err)
			}
		}()
		defer ctl.Close()
		log.Printf("the control panel: http://%s", *addr)
	}

	log.Printf("playing %s (%s), Ctrl-C to stop", title, clock(player.Duration()))
	if err := p.Listen(); err != nil {
		log.Fatal(err)
	}
}

// soundNames returns every sound a channel can play: each synth with its presets, the drum kits,
// the instruments of the orchestra (with -orchestra)
func soundNames() []string {
	var names []string
	add := func(synth string, presets []string) {
		slices.Sort(presets)
		for _, p := range presets {
			names = append(names, synth+"/"+p)
		}
	}
	add("piano", slices.Collect(maps.Keys(piano.Presets)))
	add("fm", slices.Collect(maps.Keys(fm.Presets)))
	add("dx", slices.Collect(maps.Keys(dx.Presets)))
	add("organ", slices.Collect(maps.Keys(organ.Presets)))
	add("waveguide", slices.Collect(maps.Keys(waveguide.Presets)))
	add("modal", slices.Collect(maps.Keys(modal.Presets)))
	add("westcoast", slices.Collect(maps.Keys(westcoast.Presets)))
	add("subtractive", slices.Collect(maps.Keys(subtractive.Presets)))
	add("supersaw", slices.Collect(maps.Keys(supersaw.Presets)))
	add("wavetable", slices.Collect(maps.Keys(wavetable.Presets)))
	add("pd", slices.Collect(maps.Keys(pd.Presets)))
	add("pulsar", slices.Collect(maps.Keys(pulsar.Presets)))
	add("vocal", slices.Collect(maps.Keys(vocal.Presets)))
	add("drumsynth", slices.Collect(maps.Keys(drumsynth.Kits)))
	if orchestraDir != "" {
		var files []string
		for _, f := range orchestral {
			if !slices.Contains(files, f) {
				files = append(files, f)
			}
		}
		add("sfz", files)
	}
	return names
}

// describe prints the tracks of a file and the synths its channels play on
func describe(f *midi.File, title string, soundOf func(ch, program int) string) {
	fmt.Printf("%s: format %d, %d tracks, %s\n", title, f.Format, len(f.Tracks), clock(f.Duration()))
	for i, t := range f.Tracks {
		var chs []int
		notes := 0
		for _, e := range t.Events {
			if e.Message.IsChannel() && !slices.Contains(chs, e.Message.Channel()) {
				chs = append(chs, e.Message.Channel())
			}
			if e.Message.Kind() == midi.NoteOn {
				notes++
			}
		}
		if name := t.Name(); name != "" || notes > 0 {
			fmt.Printf("  track %d %q: %d notes, channels %v\n", i+1, t.Name(), notes, chs)
		}
	}
	for _, ch := range f.Channels() {
		programs := f.Programs(ch)
		if len(programs) == 0 {
			programs = []int{0}
		}
		var s []string
		for _, prog := range programs {
			if ch == midi.DrumChannel {
				s = append(s, "drums → "+soundOf(ch, prog))
				break
			}
			s = append(s, fmt.Sprintf("%s → %s", midi.ProgramName(prog), soundOf(ch, prog)))
		}
		fmt.Printf("  channel %2d: %s\n", ch, strings.Join(s, ", "))
	}
}

// clock writes seconds as minutes and seconds
func clock(seconds float64) string {
	s := int(math.Round(seconds))
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

// prelude writes the first page of Bach's Prelude in C (BWV 846) as a MIDI file: every bar a
// chord broken the same way twice, the lowest two notes held, the pedal down through each half;
// a cello holds the bass. It slows down at the end
func prelude() *midi.File {
	bars := [][5]int{
		{60, 64, 67, 72, 76}, {60, 62, 69, 74, 77}, {59, 62, 67, 74, 77}, {60, 64, 67, 72, 76},
		{60, 64, 69, 76, 81}, {60, 62, 66, 69, 74}, {59, 62, 67, 74, 79}, {59, 60, 64, 67, 72},
		{57, 60, 64, 67, 72}, {50, 57, 62, 66, 72}, {55, 59, 62, 67, 71}, {55, 58, 64, 67, 73},
		{53, 57, 62, 69, 74}, {53, 56, 62, 65, 71}, {52, 55, 60, 67, 72}, {52, 53, 57, 60, 65},
		{50, 53, 57, 60, 65}, {43, 50, 55, 59, 65}, {48, 52, 55, 60, 64}, {48, 55, 58, 60, 64},
		{41, 53, 57, 60, 64}, {42, 48, 57, 60, 63}, {44, 53, 59, 60, 62}, {43, 53, 55, 59, 62},
	}
	const q = 480 // ticks a quarter
	const sixteenth = q / 4
	f := &midi.File{Format: 1, Division: q}
	var conductor, keys, cello midi.Track
	conductor.AddMeta(0, midi.MetaTrackName, []byte("Prelude in C, BWV 846"))
	conductor.AddMeta(0, midi.MetaTimeSignature, midi.TimeSignatureData(4, 4))
	conductor.AddMeta(0, midi.MetaTempo, midi.TempoData(66))
	keys.AddMeta(0, midi.MetaTrackName, []byte("piano"))
	keys.Add(0, midi.ProgramChangeMsg(1, 0))
	cello.AddMeta(0, midi.MetaTrackName, []byte("cello"))
	cello.Add(0, midi.ProgramChangeMsg(2, 42))
	cello.Add(0, midi.ControlChangeMsg(2, midi.CCVolume, 70))
	var at int64
	for b, chord := range bars {
		for half := range 2 {
			t := at + int64(half)*8*sixteenth
			end := t + 8*sixteenth
			keys.Add(t, midi.ControlChangeMsg(1, midi.CCSustain, 0))
			keys.Add(t+20, midi.ControlChangeMsg(1, midi.CCSustain, 110))
			for i, step := range []int{0, 1, 2, 3, 4, 2, 3, 4} {
				on := t + int64(i)*sixteenth
				off := on + sixteenth - 10
				if step < 2 {
					off = end - 10 // the two lowest held through the half
				}
				// the line leans into the top of each figure, the second half answers softer
				velocity := 52 + 6*min(step, 3) - 6*half + (b*7)%5
				keys.Add(on, midi.NoteOnMsg(1, chord[step], velocity))
				keys.Add(off, midi.NoteOffMsg(1, chord[step], 0))
			}
		}
		bass := chord[0] - 12
		cello.Add(at, midi.NoteOnMsg(2, bass, 60))
		cello.Add(at+16*sixteenth-30, midi.NoteOffMsg(2, bass, 0))
		at += 16 * sixteenth
	}
	// the last chord, held, slower and slower into it
	for i, bpm := range []float64{62, 56, 48} {
		conductor.AddMeta(at-int64(3-i)*4*sixteenth, midi.MetaTempo, midi.TempoData(bpm))
	}
	keys.Add(at, midi.ControlChangeMsg(1, midi.CCSustain, 0))
	keys.Add(at+20, midi.ControlChangeMsg(1, midi.CCSustain, 110))
	for _, n := range []int{36, 48, 55, 64, 72} {
		keys.Add(at, midi.NoteOnMsg(1, n, 58))
		keys.Add(at+4*q, midi.NoteOffMsg(1, n, 0))
	}
	keys.Add(at+4*q, midi.ControlChangeMsg(1, midi.CCSustain, 0))
	cello.Add(at, midi.NoteOnMsg(2, 36, 62))
	cello.Add(at+4*q, midi.NoteOffMsg(2, 36, 0))
	f.Tracks = []midi.Track{conductor, keys, cello}
	return f
}

func init() {
	// every program has a sound that exists
	for prog, name := range generalMIDI {
		if _, err := newSound(name); err != nil {
			fmt.Fprintf(os.Stderr, "program %d (%s): %v\n", prog, midi.ProgramName(prog), err)
			os.Exit(1)
		}
	}
}
