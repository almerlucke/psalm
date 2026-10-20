// Editors: every synth and every effect with its controls on a page, drawn from what each describes
// of itself (params, the editor plugin of the panel): choose a synth and a preset, play it on the
// page's keyboard (the computer's keys: A W S E D ..., or a MIDI keyboard), shape it, and send it
// through a chain of up to three effects, each with its controls. Above the controls of each: its
// presets, and yours (type a name, Save: kept in editors-presets/<synth or effect>). The page's own
// presets (at the top) keep a whole setup: the synth, the chain, every setting of every page
// (editors-presets/setups). The DX7 has its own editor, with the algorithm drawn. Open
// http://localhost:8087.
//
//	go run ./cmd/examples/editors
//	go run ./cmd/examples/editors -synth wavetable -effect chorus,tape,shimmer -theme midnight
package main

import (
	"flag"
	"fmt"
	"log"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/keyboard"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/dxeditor"
	"github.com/almerlucke/psalm/modules/controls/panel/editor"
	"github.com/almerlucke/psalm/modules/controls/panel/widgets"
	"github.com/almerlucke/psalm/modules/effects/chorus"
	"github.com/almerlucke/psalm/modules/effects/compressor"
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/psalm/modules/effects/crusher"
	"github.com/almerlucke/psalm/modules/effects/eq"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/effects/flanger"
	"github.com/almerlucke/psalm/modules/effects/follower"
	"github.com/almerlucke/psalm/modules/effects/freeverb"
	"github.com/almerlucke/psalm/modules/effects/gain"
	"github.com/almerlucke/psalm/modules/effects/gate"
	"github.com/almerlucke/psalm/modules/effects/granulator"
	"github.com/almerlucke/psalm/modules/effects/phaser"
	"github.com/almerlucke/psalm/modules/effects/pingpong"
	"github.com/almerlucke/psalm/modules/effects/pitchshift"
	"github.com/almerlucke/psalm/modules/effects/ringmod"
	"github.com/almerlucke/psalm/modules/effects/rotary"
	"github.com/almerlucke/psalm/modules/effects/saturate"
	"github.com/almerlucke/psalm/modules/effects/shimmer"
	"github.com/almerlucke/psalm/modules/effects/tape"
	"github.com/almerlucke/psalm/modules/effects/transient"
	"github.com/almerlucke/psalm/modules/effects/tremolo"
	"github.com/almerlucke/psalm/modules/effects/widener"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/modules/poly"
	"github.com/almerlucke/psalm/params"
	"github.com/almerlucke/psalm/synths/drumsynth"
	"github.com/almerlucke/psalm/synths/dx"
	"github.com/almerlucke/psalm/synths/grains"
	"github.com/almerlucke/psalm/synths/modal"
	"github.com/almerlucke/psalm/synths/organ"
	"github.com/almerlucke/psalm/synths/piano"
	"github.com/almerlucke/psalm/synths/pulsar"
	"github.com/almerlucke/psalm/synths/shaper"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/synths/supersaw"
	"github.com/almerlucke/psalm/synths/vocal"
	"github.com/almerlucke/psalm/synths/waveguide"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
	"github.com/almerlucke/sndfile"
)

// instrument is a synth with its presets: how to describe one, its page
type instrument struct {
	name     string
	synth    psalm.Modular
	start    string                         // the preset it starts with
	presets  []string                       // sorted
	describe func(preset string) params.Set // nil: an editor of its own (the DX7)
	make     func(preset string) psalm.Modular
	page     *panel.Panel
}

// slots is how many effects can be chained
const slots = 3

// presetsOf returns the names of presets, sorted
func presetsOf[T any](m map[string]T) []string { return slices.Sorted(maps.Keys(m)) }

// effect is an effect and its page
type effect struct {
	name string
	fx   psalm.Modular
	set  func() params.Set
	page *panel.Panel
}

// chooser is a mount showing the page chosen; it keeps the settings of all its pages in a setup
// (a preset of the main page), by the name of the page
type chooser struct {
	names []string
	pages []*panel.Panel
	now   atomic.Int64
}

func (c *chooser) add(name string, page *panel.Panel) {
	c.names = append(c.names, name)
	c.pages = append(c.pages, page)
}

// SnapshotMount keeps every page's settings, "<page>/<setting>"
func (c *chooser) SnapshotMount() map[string]float64 {
	values := map[string]float64{}
	for i, page := range c.pages {
		for k, v := range page.Snapshot() {
			values[c.names[i]+"/"+k] = v
		}
	}
	return values
}

// RestoreMount sets every page to its settings in a setup
func (c *chooser) RestoreMount(values map[string]float64) {
	byPage := map[string]map[string]float64{}
	for k, v := range values {
		name, key, ok := strings.Cut(k, "/")
		if !ok {
			continue
		}
		if byPage[name] == nil {
			byPage[name] = map[string]float64{}
		}
		byPage[name][key] = v
	}
	for i, page := range c.pages {
		if vs, ok := byPage[c.names[i]]; ok {
			page.Restore(vs)
		}
	}
}

func (c *chooser) Mounted() *panel.Panel {
	if i := int(c.now.Load()); i >= 0 && i < len(c.pages) {
		return c.pages[i]
	}
	return nil
}

// player sends notes to the synth chosen
type player struct {
	instruments []*instrument
	now         int
}

func (pl *player) ReceiveMessage(msg psalm.Message) { pl.instruments[pl.now].synth.Receive(msg, 0) }
func (pl *player) Receive(v any, _ int) {
	if m, ok := psalm.MsgVal(v); ok {
		pl.ReceiveMessage(m)
	}
}

func main() {
	addr := flag.String("addr", "localhost:8087", "the address of the page")
	theme := flag.String("theme", "default", "the theme of the page")
	startSynth := flag.String("synth", "subtractive", "the synth to start with")
	startEffect := flag.String("effect", "none", "the effects to start with, in order, up to three: chorus,tape")
	presetDir := flag.String("presets", "editors-presets", "the folder your presets are kept in, a folder for every synth and effect")
	device := flag.String("device", "", "a raw MIDI device to play from too (empty: the first keyboard, if there is one)")
	flag.Parse()

	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	// --- the synths: each made at a preset of its own (prefer, else the first), its presets, how one
	// is described ---
	table := gen.Brightness(32)
	var instruments []*instrument
	add := func(name string, presets []string, prefer string, make func(preset string) psalm.Modular, describe func(string) params.Set) {
		start := presets[0]
		if slices.Contains(presets, prefer) {
			start = prefer
		}
		synth := make(start)
		synth.(interface {
			AddTo(psalm.Patcher) psalm.Modular
		}).AddTo(p)
		instruments = append(instruments, &instrument{name: name, synth: synth, presets: presets, start: start, describe: describe, make: make})
	}
	add("subtractive", presetsOf(subtractive.Presets), "lead",
		func(n string) psalm.Modular { return subtractive.NewPoly(subtractive.Presets[n](), 12) },
		func(n string) params.Set { return subtractive.Describe(subtractive.Presets[n]()) })
	add("dx", presetsOf(dx.Presets), "bellPad",
		func(n string) psalm.Modular { return dx.NewPoly(dx.Presets[n](), 12, 2) }, nil)
	add("wavetable", presetsOf(wavetable.Presets), "pad",
		func(n string) psalm.Modular { return wavetable.NewPoly(wavetable.Presets[n](table), 12) },
		func(n string) params.Set { return wavetable.Describe(wavetable.Presets[n](table)) })
	add("supersaw", presetsOf(supersaw.Presets), "lead",
		func(n string) psalm.Modular { return supersaw.NewPoly(supersaw.Presets[n](), 12) },
		func(n string) params.Set { return supersaw.Describe(supersaw.Presets[n]()) })
	add("shaper", presetsOf(shaper.Presets), "glass",
		func(n string) psalm.Modular { return shaper.NewPoly(shaper.Presets[n](), 12) },
		func(n string) params.Set { return shaper.Describe(shaper.Presets[n]()) })
	add("pulsar", presetsOf(pulsar.Presets), "vowelLead",
		func(n string) psalm.Modular { return pulsar.NewPoly(pulsar.Presets[n](), 12) },
		func(n string) params.Set { return pulsar.Describe(pulsar.Presets[n]()) })
	add("organ", presetsOf(organ.Presets), "jazz",
		func(n string) psalm.Modular { return organ.New(organ.Presets[n]()) },
		func(n string) params.Set { return organ.Describe(organ.Presets[n]()) })
	add("piano", presetsOf(piano.Presets), "grand",
		func(n string) psalm.Modular { return piano.NewPoly(piano.Presets[n](), 24) },
		func(n string) params.Set { return piano.Describe(piano.Presets[n]()) })
	add("waveguide", presetsOf(waveguide.Presets), "guitar",
		func(n string) psalm.Modular { return waveguide.NewPoly(waveguide.Presets[n](), 12) },
		func(n string) params.Set { return waveguide.Describe(waveguide.Presets[n]()) })
	add("modal", presetsOf(modal.Presets), "vibraphone",
		func(n string) psalm.Modular { return modal.NewPoly(modal.Presets[n](), 16) },
		func(n string) params.Set { return modal.Describe(modal.Presets[n]()) })
	add("vocal", presetsOf(vocal.Presets), "choir",
		func(n string) psalm.Modular { return vocal.NewPoly(vocal.Presets[n](), 8) },
		func(n string) params.Set { return vocal.Describe(vocal.Presets[n]()) })
	add("drumsynth", presetsOf(drumsynth.Kits), "909",
		func(n string) psalm.Modular { return drumsynth.NewPoly(drumsynth.Config{Kit: drumsynth.Kits[n]()}, 16) },
		func(n string) params.Set { return drumsynth.Describe(drumsynth.Config{Kit: drumsynth.Kits[n]()}) })
	if choir, err := sndfile.NewMipMapSoundFile[float32]("assets/sounds/various/mixkit-angelical-choir-654.wav", 4); err == nil {
		add("grains", presetsOf(grains.Presets), "choirPad",
			func(n string) psalm.Modular { return grains.NewPoly(grains.Presets[n](choir, 57), 8) },
			func(n string) params.Set { return grains.Describe(grains.Presets[n](choir, 57)) })
	}

	// --- the effects: a chain of slots, each playing one of them or none; every slot has effects of
	// its own (the same effect in two slots, set two ways) ---
	vienna := convolver.IRFromSoundFile(sndfile.MustSoundFile[float32]("assets/IR44100/Musikvereinsaal.wav"))
	makeEffects := func() []*effect {
		var effects []*effect
		fx := func(name string, m psalm.Modular, set func() params.Set) {
			m.(interface {
				AddTo(psalm.Patcher) psalm.Modular
			}).AddTo(p)
			effects = append(effects, &effect{name: name, fx: m, set: set})
		}
		{
			c := chorus.New(chorus.DefaultConfig())
			fx("chorus", c, func() params.Set { return chorus.Describe(c.Settings()) })
			f := flanger.New(flanger.DefaultConfig())
			fx("flanger", f, func() params.Set { return flanger.Describe(f.Settings()) })
			ph := phaser.New(phaser.DefaultConfig())
			fx("phaser", ph, func() params.Set { return phaser.Describe(ph.Settings()) })
			tr := tremolo.New(tremolo.DefaultConfig())
			fx("tremolo", tr, func() params.Set { return tremolo.Describe(tr.Settings()) })
			ro := rotary.New(rotary.DefaultConfig())
			fx("rotary", ro, func() params.Set { return rotary.Describe(ro.Settings()) })
			rm := ringmod.New(ringmod.DefaultConfig())
			fx("ringmod", rm, func() params.Set { return ringmod.Describe(rm.Settings()) })
			wah := follower.NewWah(follower.DefaultWah())
			fx("wah", wah, func() params.Set { return follower.DescribeWah(wah.Settings()) })
			sa := saturate.New(saturate.DefaultConfig())
			fx("saturate", sa, func() params.Set { return saturate.Describe(sa.Settings()) })
			cr := crusher.New(crusher.DefaultConfig())
			fx("crusher", cr, func() params.Set { return crusher.Describe(cr.Settings()) })
			ta := tape.New(tape.DefaultConfig())
			fx("tape", ta, func() params.Set { return tape.Describe(ta.Settings()) })
			pg := pingpong.New(pingpong.DefaultConfig())
			fx("pingpong", pg, func() params.Set { return pingpong.Describe(pg.Settings()) })
			ps := pitchshift.New(7, 60, 0.5)
			fx("pitchshift", ps, func() params.Set { return pitchshift.Describe(ps) })
			gr := granulator.New(granulator.DefaultConfig())
			fx("granulator", gr, func() params.Set { return granulator.Describe(gr.Settings()) })
			fd := fdnreverb.New(fdnreverb.DefaultConfig())
			fx("fdnreverb", fd, func() params.Set { return fdnreverb.Describe(fd.Settings()) })
			fv := freeverb.New(0.7, 0.6, 0.9, 0.3, 0.7)
			fx("freeverb", fv, func() params.Set { return freeverb.Describe(fv) })
			sh := shimmer.New(shimmer.DefaultConfig())
			fx("shimmer", sh, func() params.Set { return shimmer.Describe(sh.Settings()) })
			cv := convolver.New(vienna, convolver.Config{PartitionSize: 512, Mix: 0.35, Gain: 0.7})
			fx("hall", cv, func() params.Set { return convolver.Describe(cv.Settings()) })
			co := compressor.New(compressor.DefaultConfig())
			fx("compressor", co, func() params.Set { return compressor.Describe(co.Settings()) })
			ga := gate.New(gate.DefaultConfig())
			fx("gate", ga, func() params.Set { return gate.Describe(ga.Settings()) })
			ts := transient.New(transient.DefaultConfig())
			fx("transient", ts, func() params.Set { return transient.Describe(ts.Settings()) })
			e := eq.New(eq.DefaultConfig())
			fx("eq", e, func() params.Set { return eq.Describe(e.Settings()) })
			wi := widener.New(widener.DefaultConfig())
			fx("widener", wi, func() params.Set { return widener.Describe(wi.Settings()) })
		}

		return effects
	}
	var chain [slots][]*effect
	for i := range chain {
		chain[i] = makeEffects()
	}
	var fxNames []string
	for _, e := range chain[0] {
		fxNames = append(fxNames, e.name)
	}

	// the synths into a bus, the bus through the effects chosen, in order, into the out the mixer plays
	bus, out := gain.New(0), gain.New(0)
	bus.AddTo(p)
	out.AddTo(p)
	for _, in := range instruments {
		bus.In(in.synth)
	}
	mix := mixer.New(p)
	mix.Channel("out", out, mixer.StripConfig{Volume: -3})
	p.Out(mix.Master())
	choice := [slots]int{-1, -1, -1} // the effect of each slot, -1 none (on the audio thread)
	var connected []psalm.Modular    // the chain as it is connected
	rewire := func() {               // on the audio thread, between buffers
		for i := 1; i < len(connected); i++ {
			connected[i-1].Disconnect(0, connected[i], 0)
		}
		connected = []psalm.Modular{bus}
		for s, i := range choice {
			if i >= 0 {
				connected = append(connected, chain[s][i].fx)
			}
		}
		connected = append(connected, out)
		for i := 1; i < len(connected); i++ {
			connected[i].In(connected[i-1])
		}
	}
	for s, name := range strings.Split(*startEffect, ",") {
		if s < slots {
			choice[s] = slices.Index(fxNames, strings.TrimSpace(name))
		}
	}
	rewire()

	// --- the pages: one for every synth and every effect of every slot, the ones chosen mounted ---
	t, ok := panel.Themes[*theme]
	if !ok {
		log.Fatalf("theme %q: one of %v", *theme, slices.Sorted(maps.Keys(panel.Themes)))
	}
	synthPages := &chooser{}
	for _, in := range instruments {
		in.page = panel.New(p, in.name)
		in.page.SetTheme(t)
		if in.describe == nil { // the DX7: its own editor, its presets in its bank
			var bank []dx.Patch // the one it starts with first
			order := append([]string{in.start}, slices.DeleteFunc(slices.Clone(in.presets), func(n string) bool { return n == in.start })...)
			for _, n := range order {
				patch := dx.Presets[n]()
				patch.Name = n
				bank = append(bank, patch)
			}
			de := dxeditor.Add(in.page, in.name, in.synth.(*poly.Poly[*dx.Voice]), bank[0], dxeditor.Bank(bank))
			if err := de.Keep(filepath.Join(*presetDir, in.name)); err != nil {
				log.Fatal(err)
			}
		} else {
			// its presets above the controls: the synth's own, and yours, kept in a folder of its own
			ed := editor.Add(in.page, in.name, in.synth, in.describe(in.start), panel.NoDefault())
			// a factory preset is the synth made at it, swapped in (the voices of a Poly, the
			// whole state of the organ and the piano)
			if sw := poly.NewSwap(p.Do, in.synth, func(n string) psalm.Modular {
				var m psalm.Modular
				p.Within(func() { m = in.make(n) })
				return m
			}); sw.Can() {
				ed.FactoryApply(in.presets, in.describe, sw.Load)
			} else {
				ed.Factory(in.presets, in.describe)
			}
			if err := ed.Keep(filepath.Join(*presetDir, in.name)); err != nil {
				log.Fatal(err)
			}
		}
		synthPages.add(in.name, in.page)
	}
	var slotPages [slots]*chooser
	for s := range chain {
		slotPages[s] = &chooser{}
		slotPages[s].now.Store(int64(choice[s]))
		for _, e := range chain[s] {
			e.page = panel.New(p, e.name)
			e.page.SetTheme(t)
			ed := editor.Add(e.page, e.name, e.fx, e.set(), panel.NoDefault())
			if err := ed.Keep(filepath.Join(*presetDir, e.name)); err != nil { // the effect's presets, the same in every slot
				log.Fatal(err)
			}
			slotPages[s].add(e.name, e.page)
		}
	}

	pl := &player{instruments: instruments}
	pl.now = max(0, slices.IndexFunc(instruments, func(in *instrument) bool { return in.name == *startSynth }))
	synthPages.now.Store(int64(pl.now))

	ctl := panel.New(p, "Editors")
	ctl.SetTheme(t)
	// setups: the synth, the chain and every setting of every page, saved by name on the page
	if err := ctl.Presets(filepath.Join(*presetDir, "setups")); err != nil {
		log.Fatal(err)
	}
	ctl.Section("Play")
	var synthNames []string
	for _, in := range instruments {
		synthNames = append(synthNames, in.name)
	}
	ctl.Select("Synth", synthNames, func(i int) {
		if off, ok := instruments[pl.now].synth.(interface{ AllNotesOff() }); ok && i != pl.now {
			off.AllNotesOff()
		}
		pl.now = i
		synthPages.now.Store(int64(i))
	}, panel.Default(float64(pl.now)))
	for s := range slots {
		ctl.Select(fmt.Sprintf("Effect %d", s+1), append([]string{"none"}, fxNames...), func(i int) {
			slotPages[s].now.Store(int64(i - 1))
			p.Do(func() { choice[s] = i - 1; rewire() })
		}, panel.Default(float64(choice[s]+1)))
	}
	widgets.Keys(ctl, "Keys", widgets.Play(pl), widgets.From(36), widgets.Octaves(4))
	widgets.Scope(ctl, "Scope", mix.Master())
	ctl.Fader("Volume", -40, 6, panel.Linear, panel.Message(mix.Master(), "volume"), panel.Unit("dB"), panel.Default(0), panel.Smooth(0.03))
	ctl.Meter("Out", mix.Master())
	ctl.Section("Synth")
	ctl.Mount("Synth page", synthPages)
	for s := range slots {
		ctl.Section(fmt.Sprintf("Effect %d", s+1))
		ctl.Mount(fmt.Sprintf("Effect page %d", s+1), slotPages[s])
	}

	// a MIDI keyboard, when there is one: it plays the synth chosen
	if d, err := midi.OpenDevice(*device); err == nil {
		defer d.Close()
		keyboard.New(d, keyboard.Config{Dynamics: 1, Controls: map[int]string{midi.CCModWheel: "modWheel"}, HoldNotes: true,
			PitchBend: "pitchBend", Pressure: "pressure"}, pl).AddTo(p)
		log.Printf("playing from %s too", d.Path())
	} else if *device != "" {
		log.Fatal(err)
	}

	go func() {
		if err := ctl.Serve(*addr); err != nil {
			log.Fatal(err)
		}
	}()
	defer ctl.Close()
	log.Printf("open http://%s: %d synths, %d effects in %d slots", *addr, len(instruments), len(fxNames), slots)
	if err := p.Listen(); err != nil {
		log.Fatal(err)
	}
}
