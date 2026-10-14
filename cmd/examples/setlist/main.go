// Setlist: play patches of the patcher one after the other, as the songs of a playlist. Every patch
// kept in the patches folder (the patcher example's) is a song: built when it is needed, in a patch
// of its own, with its own tempo. A patch with a song module (its sections) ends when its song does
// (its loop is off then: skip a section and it ends sooner, repeat one and it waits); one without
// plays until Next. The page shows the songs (tap one to play it next, its dot to switch it off),
// the sections of the song playing (its tiles, Repeat, Jump at) and the patch playing itself, as on
// the patcher's page: change it while it plays. Changes are not kept: save them in the patcher.
// Play the page's keys (the Keys button), the computer's keys or a MIDI keyboard.
//
//	go run ./cmd/examples/setlist                          every patch in patches, by name
//	go run ./cmd/examples/setlist -songs "song example,testing"
//	go run ./cmd/examples/setlist -mix                     one into the other, like a DJ set
package main

import (
	"cmp"
	"flag"
	"fmt"
	"log"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher"
	"github.com/almerlucke/psalm/modules/controls/panel/patcher/catalog"
	"github.com/almerlucke/psalm/modules/controls/panel/widgets"
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/modules/playlist"
	"github.com/almerlucke/sndfile"
)

// inputs reads several inputs as one, in order of offset
type inputs []midi.Input

func (in inputs) Read(events []midi.Event, frames int, sampleRate float64) []midi.Event {
	start := len(events)
	for _, i := range in {
		events = i.Read(events, frames, sampleRate)
	}
	slices.SortStableFunc(events[start:], func(a, b midi.Event) int { return cmp.Compare(a.Offset, b.Offset) })
	return events
}

func main() {
	addr := flag.String("addr", "localhost:8087", "the address of the page")
	theme := flag.String("theme", "default", "the theme of the page")
	patches := flag.String("patches", "patches", "the folder the patches are kept in (the patcher example's)")
	songs := flag.String("songs", "", "the patches to play, by name, by commas (empty: all of them, by name)")
	presets := flag.String("presets", "editors-presets", "the folder the modules' presets of your own are kept in")
	device := flag.String("device", "", "a raw MIDI device to play from too (empty: the first keyboard, if there is one)")
	djMix := flag.Bool("mix", false, "mix the songs into each other like a DJ set (on a bar, the tempo matched)")
	flag.Parse()

	psalm.PushConfig(psalm.Config{BPM: 120, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	t, ok := panel.Themes[*theme]
	if !ok {
		log.Fatalf("theme %q: one of %v", *theme, slices.Sorted(maps.Keys(panel.Themes)))
	}

	// the patches to play: those kept in the folder, by name
	var names []string
	files, _ := os.ReadDir(*patches)
	for _, f := range files {
		if n, ok := strings.CutSuffix(f.Name(), ".json"); ok && !f.IsDir() {
			names = append(names, n)
		}
	}
	slices.Sort(names)
	if *songs != "" {
		var chosen []string
		for _, n := range strings.Split(*songs, ",") {
			n = strings.TrimSpace(n)
			if !slices.Contains(names, n) {
				log.Fatalf("no patch %q in %s: %v", n, *patches, names)
			}
			chosen = append(chosen, n)
		}
		names = chosen
	}
	if len(names) == 0 {
		log.Fatalf("no patches in %s: make some with the patcher example (go run ./cmd/examples/patcher)", *patches)
	}

	// what the keyboards of the patches play from: the page's keys, and a MIDI keyboard
	keys := midi.NewQueue()
	in := inputs{keys}
	if d, err := midi.OpenDevice(*device); err == nil {
		defer d.Close()
		in = append(in, d)
		log.Printf("playing from %s too", d.Path())
	} else if *device != "" {
		log.Fatal(err)
	}
	hall := convolver.IRFromSoundFile(sndfile.MustSoundFile[float32]("assets/IR44100/Musikvereinsaal.wav"))

	// a song: the patch kept under its name, made again in a patch of its own
	song := func(name string) playlist.Song {
		return playlist.Song{Name: name, Build: func(sp *psalm.Psalm) playlist.Built {
			ctl := panel.New(sp, name)
			ctl.SetTheme(t)
			pt := patcher.New(sp, ctl, "Patch", catalog.Kinds(catalog.Config{Input: in, IR: hall}),
				patcher.Keep(*patches), patcher.Presets(*presets), patcher.Theme(t))
			if err := pt.Load(name); err != nil {
				log.Printf("%s: %v", name, err)
			}
			b := playlist.Built{Out: pt.Out(), Panel: ctl}
			if tr := patcher.SongOf(sp.Env()).Transport(); tr != nil { // its song: the patch ends with it
				b.Transport = tr
				b.Length = tr.Form().Beats() * 60 / pt.Tempo().Value()
			}
			return b
		}}
	}
	var list []playlist.Song
	for _, n := range names {
		list = append(list, song(n))
	}
	cfg := playlist.Config{Gap: 1, Loop: true}
	if *djMix {
		cfg.Crossfade, cfg.OnBar, cfg.Phrase, cfg.MatchTempo = 4, true, 1, true
	}
	pl := playlist.New(p, cfg, list...)
	defer pl.Close()

	mix := mixer.New(p)
	mix.Channel("songs", pl, mixer.StripConfig{Volume: -3})
	p.Out(mix.Master())

	ctl := panel.New(p, "Setlist")
	ctl.SetTheme(t)
	ctl.Fader("Volume", -40, 6, panel.Linear, panel.Message(mix.Master(), "volume"), panel.Unit("dB"), panel.Default(0), panel.Smooth(0.03)).Top()
	ctl.Meter("Out", mix.Master()).Top()
	widgets.Scope(ctl, "Scope", mix.Master()).Top()
	widgets.Keys(ctl, "Keys", func(note int, velocity float64, on bool) {
		if on {
			keys.Push(midi.NoteOnMsg(1, note, max(1, int(velocity*127))))
		} else {
			keys.Push(midi.NoteOffMsg(1, note, 0))
		}
	}, widgets.From(36), widgets.Octaves(4)).Top()
	ctl.Section("Songs")
	pl.Controls(ctl, "Songs")
	ctl.Section("Sections")
	pl.SectionControls(ctl, "Sections")
	ctl.Section("Song")
	pl.SongControls(ctl, "Song")

	if err := p.AudioStart(); err != nil {
		log.Fatal(err)
	}
	go func() {
		if err := ctl.Serve(*addr); err != nil {
			log.Fatal(err)
		}
	}()
	defer ctl.Close()
	fmt.Printf("the setlist of %d patches: http://%s\n", len(names), *addr)
	select {}
}
