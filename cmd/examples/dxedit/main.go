// Dxedit: edit the six operator FM synth on a page, as a DX7 with a screen. The algorithm as a
// diagram (click through the 32, or make one of your own: "my own", then the modulations and the
// carriers), every operator (frequency, level, detune, velocity, rate scaling, its envelope and
// keyboard scaling), the LFO and the pitch envelope, and a bank to load from: the built in patches
// and every SysEx bank in assets/dx7 (or the ones given). Play it on the page's keyboard (or the
// computer's keys: A W S E D ...) or a MIDI keyboard. Save patches of your own by name (kept in
// dx-presets, -presets to move them). Open http://localhost:8087.
//
//	go run ./cmd/examples/dxedit
//	go run ./cmd/examples/dxedit -device /dev/snd/midiC3D0 path/to/bank.syx
package main

import (
	"flag"
	"log"
	"maps"
	"path/filepath"
	"slices"

	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/keyboard"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/dxeditor"
	"github.com/almerlucke/psalm/modules/controls/panel/widgets"
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/dx"
	"github.com/almerlucke/sndfile"
)

func main() {
	addr := flag.String("addr", "localhost:8087", "the address of the page")
	device := flag.String("device", "", "a raw MIDI device to play it from too (empty: the first keyboard, if there is one)")
	theme := flag.String("theme", "default", "the theme of the page")
	mine := flag.String("presets", "dx-presets", "the folder the patches you save are kept in")
	sysex := flag.String("sysex", "dx-sysex", "the folder of SysEx files the page lists, loads as the bank and saves into")
	flag.Parse()

	// the bank: the built in patches, then every voice of the SysEx banks
	var bank []dx.Patch
	for _, name := range slices.Sorted(maps.Keys(dx.Presets)) {
		p := dx.Presets[name]()
		p.Name = name
		bank = append(bank, p)
	}
	files := flag.Args()
	if len(files) == 0 {
		files, _ = filepath.Glob("assets/dx7/*.syx")
		slices.Sort(files)
	}
	for _, f := range files {
		patches, err := dx.LoadSysEx(f)
		if err != nil {
			log.Printf("%s: %v", f, err)
			continue
		}
		bank = append(bank, patches...)
	}
	log.Printf("%d patches", len(bank))

	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	synth := dx.NewPoly(bank[0], 16, 2)
	synth.AddTo(p)
	mix := mixer.New(p)
	vienna := convolver.IRFromSoundFile(sndfile.MustSoundFile[float32]("assets/IR44100/Musikvereinsaal.wav"))
	mix.Bus("hall", mixer.StripConfig{}, convolver.New(vienna, convolver.Config{PartitionSize: 512, Mix: 1, Gain: 0.7}).AddTo(p))
	mix.Channel("dx", synth, mixer.StripConfig{Volume: -4, Sends: map[string]float64{"hall": 0.25}})
	p.Out(mix.Master())

	// a MIDI keyboard, when there is one: the velocity, the sustain pedal, the bend and the wheel
	if d, err := midi.OpenDevice(*device); err == nil {
		defer d.Close()
		keyboard.New(d, keyboard.Config{Controls: map[int]string{midi.CCModWheel: "modWheel"}, HoldNotes: true,
			PitchBend: "pitchBend", Pressure: "pressure"}, synth).AddTo(p)
		log.Printf("playing from %s too", d.Path())
	} else if *device != "" {
		log.Fatal(err)
	}

	ctl := panel.New(p, "DX")
	t, ok := panel.Themes[*theme]
	if !ok {
		log.Fatalf("theme %q: one of %v", *theme, slices.Sorted(maps.Keys(panel.Themes)))
	}
	ctl.SetTheme(t)
	ctl.Section("DX")
	ed := dxeditor.Add(ctl, "DX", synth, bank[0], dxeditor.Bank(bank), dxeditor.SysExFolder(*sysex))
	if err := ed.Keep(*mine); err != nil { // patches of your own
		log.Fatal(err)
	}
	ctl.Section("Play")
	widgets.Keys(ctl, "Keys", widgets.Play(synth), widgets.From(36), widgets.Octaves(4))
	widgets.XY(ctl, "Hands", widgets.Axis{Name: "wheel", Max: 1, To: panel.Message(synth, "modWheel")},
		widgets.Axis{Name: "brightness", Min: 0.2, Max: 2, To: panel.Message(synth, "modScale"), Default: 1})
	widgets.Scope(ctl, "Scope", mix.Master())
	ctl.Meter("Out", mix.Master())

	go func() {
		if err := ctl.Serve(*addr); err != nil {
			log.Fatal(err)
		}
	}()
	defer ctl.Close()
	log.Printf("open http://%s", *addr)
	if err := p.Listen(); err != nil {
		log.Fatal(err)
	}
}
