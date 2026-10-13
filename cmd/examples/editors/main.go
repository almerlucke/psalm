// Editors: every synth and every effect with its controls on a page, drawn from what each describes
// of itself (params, the editor plugin of the panel): choose a synth and a preset, play it on the
// page's keyboard (the computer's keys: A W S E D ..., or a MIDI keyboard), shape it, and send it
// through any effect. Above the controls of each: its presets, and yours (type a name, Save: kept
// in editors-presets/<synth>, -presets to move them). The DX7 has its own editor, with the algorithm drawn. Open
// http://localhost:8087.
//
//	go run ./cmd/examples/editors
//	go run ./cmd/examples/editors -synth wavetable -effect shimmer -theme midnight
package main

import (
	"flag"
	"log"
	"maps"
	"path/filepath"
	"slices"
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
	"github.com/almerlucke/psalm/synths/fm"
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

// instrument is a synth with its presets: how to describe one, its page
type instrument struct {
	name     string
	synth    psalm.Modular
	start    string                         // the preset it starts with
	presets  []string                       // sorted
	describe func(preset string) params.Set // nil: an editor of its own (the DX7)
	page     *panel.Panel
}

// presetsOf returns the names of presets, sorted
func presetsOf[T any](m map[string]T) []string { return slices.Sorted(maps.Keys(m)) }

// effect is an effect and its page
type effect struct {
	name string
	fx   psalm.Modular
	set  func() params.Set
	page *panel.Panel
}

// chooser is a mount showing the page chosen
type chooser struct {
	pages []*panel.Panel
	now   atomic.Int64
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
	startEffect := flag.String("effect", "none", "the effect to start with")
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
		instruments = append(instruments, &instrument{name: name, synth: synth, presets: presets, start: start, describe: describe})
	}
	add("subtractive", presetsOf(subtractive.Presets), "lead",
		func(n string) psalm.Modular { return subtractive.NewPoly(subtractive.Presets[n](), 12) },
		func(n string) params.Set { return subtractive.Describe(subtractive.Presets[n]()) })
	add("fm", presetsOf(fm.Presets), "ePiano",
		func(n string) psalm.Modular { return fm.NewPoly(fm.Presets[n](), 12) },
		func(n string) params.Set { return fm.Describe(fm.Presets[n]()) })
	add("dx", presetsOf(dx.Presets), "bellPad",
		func(n string) psalm.Modular { return dx.NewPoly(dx.Presets[n](), 12, 2) }, nil)
	add("wavetable", presetsOf(wavetable.Presets), "pad",
		func(n string) psalm.Modular { return wavetable.NewPoly(wavetable.Presets[n](table), 12) },
		func(n string) params.Set { return wavetable.Describe(wavetable.Presets[n](table)) })
	add("supersaw", presetsOf(supersaw.Presets), "lead",
		func(n string) psalm.Modular { return supersaw.NewPoly(supersaw.Presets[n](), 12) },
		func(n string) params.Set { return supersaw.Describe(supersaw.Presets[n]()) })
	add("pd", presetsOf(pd.Presets), "brass",
		func(n string) psalm.Modular { return pd.NewPoly(pd.Presets[n](), 12) },
		func(n string) params.Set { return pd.Describe(pd.Presets[n]()) })
	add("pulsar", presetsOf(pulsar.Presets), "vowelLead",
		func(n string) psalm.Modular { return pulsar.NewPoly(pulsar.Presets[n](), 12) },
		func(n string) params.Set { return pulsar.Describe(pulsar.Presets[n]()) })
	add("westcoast", presetsOf(westcoast.Presets), "pluck",
		func(n string) psalm.Modular { return westcoast.NewPoly(westcoast.Presets[n](), 12) },
		func(n string) params.Set { return westcoast.Describe(westcoast.Presets[n]()) })
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

	// --- the effects: the synths play through the one chosen ---
	vienna := convolver.IRFromSoundFile(sndfile.MustSoundFile[float32]("assets/IR44100/Musikvereinsaal.wav"))
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

	// the synths into a bus, the bus through the effect chosen into the slot the mixer plays
	bus, slot := gain.New(0), gain.New(0)
	bus.AddTo(p)
	slot.AddTo(p)
	for _, in := range instruments {
		bus.In(in.synth)
	}
	mix := mixer.New(p)
	mix.Channel("out", slot, mixer.StripConfig{Volume: -3})
	p.Out(mix.Master())
	current := -1         // the effect playing, -1 none
	plug := func(i int) { // on the audio thread, between buffers
		if current < 0 {
			bus.Disconnect(0, slot, 0)
		} else {
			old := effects[current].fx
			bus.Disconnect(0, old, 0)
			old.Disconnect(0, slot, 0)
		}
		current = i
		if i < 0 {
			slot.In(bus)
			return
		}
		effects[i].fx.In(bus)
		slot.In(effects[i].fx)
	}
	start := slices.IndexFunc(effects, func(e *effect) bool { return e.name == *startEffect })
	plug(start)

	// --- the pages: one for every synth and effect, the one chosen mounted ---
	t, ok := panel.Themes[*theme]
	if !ok {
		log.Fatalf("theme %q: one of %v", *theme, slices.Sorted(maps.Keys(panel.Themes)))
	}
	synthPages, fxPages := &chooser{}, &chooser{}
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
			ed.Factory(in.presets, in.describe)
			if err := ed.Keep(filepath.Join(*presetDir, in.name)); err != nil {
				log.Fatal(err)
			}
		}
		synthPages.pages = append(synthPages.pages, in.page)
	}
	for _, e := range effects {
		e.page = panel.New(p, e.name)
		e.page.SetTheme(t)
		ed := editor.Add(e.page, e.name, e.fx, e.set(), panel.NoDefault())
		if err := ed.Keep(filepath.Join(*presetDir, e.name)); err != nil {
			log.Fatal(err)
		}
		fxPages.pages = append(fxPages.pages, e.page)
	}

	pl := &player{instruments: instruments}
	pl.now = max(0, slices.IndexFunc(instruments, func(in *instrument) bool { return in.name == *startSynth }))
	synthPages.now.Store(int64(pl.now))
	fxPages.now.Store(int64(start))

	ctl := panel.New(p, "Editors")
	ctl.SetTheme(t)
	ctl.Section("Play")
	var synthNames, fxNames []string
	for _, in := range instruments {
		synthNames = append(synthNames, in.name)
	}
	fxNames = append(fxNames, "none")
	for _, e := range effects {
		fxNames = append(fxNames, e.name)
	}
	ctl.Select("Synth", synthNames, func(i int) {
		if off, ok := instruments[pl.now].synth.(interface{ AllNotesOff() }); ok && i != pl.now {
			off.AllNotesOff()
		}
		pl.now = i
		synthPages.now.Store(int64(i))
	}, panel.Default(float64(pl.now)))
	ctl.Select("Effect", fxNames, func(i int) {
		fxPages.now.Store(int64(i - 1))
		p.Do(func() { plug(i - 1) })
	}, panel.Default(float64(start+1)))
	widgets.Keys(ctl, "Keys", widgets.Play(pl), widgets.From(36), widgets.Octaves(4))
	widgets.Scope(ctl, "Scope", mix.Master())
	ctl.Fader("Volume", -40, 6, panel.Linear, panel.Message(mix.Master(), "volume"), panel.Unit("dB"), panel.Default(0), panel.Smooth(0.03))
	ctl.Meter("Out", mix.Master())
	ctl.Section("Synth")
	ctl.Mount("Synth page", synthPages)
	ctl.Section("Effect")
	ctl.Mount("Effect page", fxPages)

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
	log.Printf("open http://%s: %d synths, %d effects", *addr, len(instruments), len(effects))
	if err := p.Listen(); err != nil {
		log.Fatal(err)
	}
}
