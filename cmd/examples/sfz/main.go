// SFZ: play a sampled instrument (an SFZ file and its sounds) from a MIDI keyboard.
//
// The sounds an SFZ file names are found next to it (and its default_path). When they live
// somewhere else, -sounds gives the folder they are in instead: the paths in the file are taken
// from there. At the start it prints what it loaded: the regions, the keys they cover, the opcodes
// the sampler doesn't know (those are left out).
//
// The keyboard plays the velocity into the instrument (its regions choose by it, and play louder).
// Its controllers go to the instrument as they are: the mod wheel (controller 1) blends the
// dynamic layers of instruments that crossfade on it, or changes their attack and volume; the
// sustain pedal holds the notes as the instrument says (its sustain_lo is where it is down; with
// sustain_sw=off the keyboard holds them). Instruments with keyswitches list their articulations at the start;
// pressing a keyswitch key prints the one chosen. Without a keyboard, -demo plays a phrase.
//
//	go run ./cmd/examples/sfz -sfz ~/Downloads/Timpani/Timpani.sfz
//	go run ./cmd/examples/sfz -sfz piano.sfz -sounds ~/samples/piano    the sounds elsewhere
//	go run ./cmd/examples/sfz -sfz glass.sfz -rename samples=glass        a folder named otherwise
//	go run ./cmd/examples/sfz -sfz piano.sfz -device /dev/snd/midiC3D0    a given MIDI device
//	go run ./cmd/examples/sfz -sfz piano.sfz -demo                        no keyboard: a phrase
package main

import (
	"flag"
	"fmt"
	"log"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
	sfzPath := flag.String("sfz", "", "the SFZ file of the instrument")
	sounds := flag.String("sounds", "", "the folder the sounds are in, when not next to the SFZ file")
	rename := flag.String("rename", "", "a folder of the paths in the file that is named otherwise on disk: old=new")
	mipmaps := flag.Bool("mipmaps", false, "keep band limited copies of the sounds, for regions played far above their key (more memory)")
	device := flag.String("device", "", "the raw MIDI device (empty is the first keyboard)")
	demo := flag.Bool("demo", false, "play a phrase instead of reading a keyboard")
	voices := flag.Int("voices", 32, "notes sounding at once")
	transpose := flag.Int("transpose", 0, "semitones")
	volume := flag.Float64("volume", -6, "dB")
	hallSend := flag.Float64("hall", 0.2, "how much goes to the hall (0-1)")
	monitor := flag.Bool("monitor", false, "print every MIDI event")
	buffer := flag.Int("buffer", 256, "the buffer size (the latency)")
	flag.Parse()
	if *sfzPath == "" {
		flag.Usage()
		os.Exit(2)
	}

	// --- the instrument ---

	load := sampler.LoadFile
	if *mipmaps {
		load = sampler.MipMaps(4)
	}
	sfzDir := filepath.Dir(*sfzPath)
	var from, to string
	if *rename != "" {
		var ok bool
		if from, to, ok = strings.Cut(*rename, "="); !ok {
			log.Fatalf("-rename %q: old=new", *rename)
		}
	}
	start := time.Now()
	inst, err := sampler.LoadSFZWith(*sfzPath, func(path string) (sampler.Sound, error) {
		// the path as the file has it (from the SFZ file's folder), then where it really is
		if rel, err := filepath.Rel(sfzDir, path); err == nil && *sounds != "" {
			path = filepath.Join(*sounds, rel)
		}
		if from != "" {
			path = renameFolder(path, from, to)
		}
		return load(path)
	})
	if err != nil {
		log.Fatal(err)
	}
	describe(inst, time.Since(start))

	// --- the engine, the keyboard, the hall ---

	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: *buffer})
	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	var input midi.Input
	if *demo {
		q := midi.NewQueue()
		input = q
		go playDemo(q, inst)
	} else {
		d, err := midi.OpenDevice(*device)
		if err != nil {
			log.Fatalf("%v (go run ./cmd/examples/keyboard -list shows the devices, -demo plays without one)", err)
		}
		defer d.Close()
		log.Printf("reading %s", d.Path())
		input = d
	}

	synth := sampler.NewPoly(sampler.DefaultConfig(inst), *voices).AddTo(p)
	// the instrument takes the velocity itself, and every controller as itself ("cc1" the mod wheel, "cc64" the sustain pedal); without a pedal of
	// its own the keyboard holds the notes
	controls := map[int]string{}
	for cc := range 120 {
		if cc != midi.CCSustain || !inst.NoSustain {
			controls[cc] = fmt.Sprintf("cc%d", cc)
		}
	}
	kb := keyboard.New(input, keyboard.Config{Transpose: *transpose, Controls: controls, HoldNotes: inst.NoSustain, PitchBend: "pitchBend"}, synth)
	switches, _ := inst.Switches()
	if len(switches) > 0 {
		kb.OnEvent(func(e midi.Event, _ psalm.CtrlTick) {
			if e.Message.Kind() != midi.NoteOn || e.Message.Velocity() == 0 {
				return
			}
			for _, s := range switches {
				if s.Key == e.Message.Note()+*transpose {
					log.Printf("articulation: %s", label(s))
				}
			}
		})
	}
	if *monitor {
		kb.OnEvent(func(e midi.Event, _ psalm.CtrlTick) {
			if k := e.Message.Kind(); k != midi.Clock && k != midi.ActiveSensing {
				log.Printf("%v", e.Message)
			}
		})
	}
	kb.AddTo(p)

	mix := mixer.New(p)
	hall := fdnreverb.DefaultConfig()
	hall.Decay, hall.Mix = 2.2, 1
	mix.Bus("hall", mixer.StripConfig{}, fdnreverb.New(hall).AddTo(p))
	mix.Channel("instrument", synth, mixer.StripConfig{Volume: *volume, Sends: map[string]float64{"hall": *hallSend}})
	p.Out(mix.Master())

	log.Printf("playing, Ctrl-C to stop")
	if err := p.Listen(); err != nil {
		log.Fatal(err)
	}
}

// renameFolder replaces a folder of a path by another name ("samples" by "glass")
func renameFolder(path, from, to string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for i, part := range parts {
		if part == from {
			parts[i] = to
		}
	}
	return filepath.FromSlash(strings.Join(parts, "/"))
}

// describe prints what was loaded
func describe(inst *sampler.Instrument, took time.Duration) {
	lo, hi := inst.Keys()
	sounds := map[string]bool{}
	release, loops, rr := 0, 0, 0
	for _, r := range inst.Regions {
		sounds[r.Name] = true
		if r.Trigger == sampler.OnRelease {
			release++
		}
		if r.Loop == sampler.LoopContinuous || r.Loop == sampler.LoopSustain {
			loops++
		}
		if r.SeqLength > 1 || r.HiRand-r.LoRand < 1 {
			rr++
		}
	}
	fmt.Printf("%d regions, %d sounds, keys %s to %s (%d to %d), loaded in %v\n",
		len(inst.Regions), len(sounds), noteName(lo), noteName(hi), lo, hi, took.Round(time.Millisecond))
	if loops > 0 || release > 0 || rr > 0 {
		fmt.Printf("%d looped, %d release regions, %d taking turns (round robin or random)\n", loops, release, rr)
	}
	if switches, def := inst.Switches(); len(switches) > 0 {
		fmt.Println("keyswitches:")
		for _, s := range switches {
			mark := ""
			if s.Key == def {
				mark = " (at the start)"
			}
			fmt.Printf("  %-4s %s%s\n", noteName(s.Key), label(s), mark)
		}
	}
	if len(inst.CC) > 0 {
		var ccs []string
		for _, cc := range slices.Sorted(maps.Keys(inst.CC)) {
			ccs = append(ccs, fmt.Sprintf("cc%d=%g", cc, inst.CC[cc]))
		}
		fmt.Printf("controllers start at %s\n", strings.Join(ccs, ", "))
	}
	if len(inst.Unsupported) > 0 {
		var names []string
		for name, n := range inst.Unsupported {
			names = append(names, fmt.Sprintf("%s (%d)", name, n))
		}
		slices.Sort(names)
		fmt.Printf("left out, the sampler doesn't know them: %s\n", strings.Join(names, ", "))
	}
}

// label returns the name of a keyswitch, its key when it has none
func label(s sampler.Switch) string {
	if s.Label != "" {
		return s.Label
	}
	return noteName(s.Key)
}

func noteName(key int) string {
	if key < 0 {
		return "-"
	}
	return fmt.Sprintf("%s%d", []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}[key%12], key/12-1)
}

// playDemo plays a rising and falling phrase over the keys of the instrument, again and again
func playDemo(q *midi.Queue, inst *sampler.Instrument) {
	lo, hi := inst.Keys()
	mid := (lo + hi) / 2
	if hi-lo > 24 {
		lo, hi = mid-12, mid+12
	}
	var keys []int
	for k := lo; k <= hi; k += max(1, (hi-lo)/8) {
		keys = append(keys, k)
	}
	for i := len(keys) - 2; i > 0; i-- {
		keys = append(keys, keys[i])
	}
	for {
		for i, k := range keys {
			q.Push(midi.NoteOnMsg(1, k, 50+(i*37)%70))
			time.Sleep(350 * time.Millisecond)
			q.Push(midi.NoteOffMsg(1, k, 0))
			time.Sleep(50 * time.Millisecond)
		}
		time.Sleep(time.Second)
	}
}
