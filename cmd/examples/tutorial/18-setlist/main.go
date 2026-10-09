// Step 18: a setlist, steering the songs and their sections.
//
// The playlist of step 17 and the transport of step 16 together: songs one after the other, and
// the sections of the song playing, both on the page. A song hands its transport to the playlist:
//
//	return playlist.Built{Out: mix.Master(), Length: arrangement.Duration(), Transport: form.Transport()}
//
// Then the song ends when its transport ended, wherever the sections took it: skip the bridge and
// the next song comes sooner, repeat the chorus and it waits. (Its form doesn't loop: the song ends
// with its last section. Length is still how long it plays as written: how far the song is, and
// how long a render is.)
//
//	list.Controls(ctl, "Songs")             a tile per song, Next
//	list.SectionControls(ctl, "Sections")   a tile per section of the song playing, Repeat, Jump at
//
// The section tiles change with the song. They work as in step 16: tap one to play it next, its
// dot to switch it off; Repeat holds the section playing, Jump at says when a section asked for
// starts. Repeat and Jump at count for every song. Every song has a section that is off as written
// (a bridge, a break, a night): turn it on, or tap it, while the song plays. A song played again
// is built again, so its sections start as written.
//
// Every song has controls of its own too: it makes its own panel in Build, bound to its own
// modules, and hands it to the playlist with the rest:
//
//	ctl := panel.New(p, "Groove")                                  in the song's patch
//	ctl.Fader("Tempo", 90, 120, panel.Linear, panel.Func(func(v float64) { p.Env().SetBPM(v) }), ...)
//	return playlist.Built{..., Panel: ctl}
//
//	list.SongControls(ctl, "Song")          the panel of the song playing
//
// The page shows the panel of the song playing, and another one when the next song starts. MIDI
// reaches it too, and what MIDI learn binds on it is kept for the song.
//
// MIDI: with -midi the pads of a controller can play songs and sections (MIDI learn on the page).
// The section tiles' buttons are named by position, "Sections/1", "Sections/2", ...: the pad bound
// to the second tile plays the second section of whatever song is playing.
//
// Presets keep the songs that are on, Repeat and Jump at, and the controls of every song (as
// "Song/Groove/Tempo"): loading one sets the song playing at once and the others when they play.
// A song keeps its controls as you left them when it plays again, also without presets.
//
//	go run ./cmd/examples/tutorial/18-setlist               then open http://localhost:8087
//	go run ./cmd/examples/tutorial/18-setlist -midi
//	go run ./cmd/examples/tutorial/18-setlist -render set.wav
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/cmd/examples/tutorial/tutorial"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/keyboard"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/modules/playlist"
	"github.com/almerlucke/psalm/synths/fm"
	"github.com/almerlucke/psalm/synths/piano"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
	"github.com/almerlucke/psalm/tools/harmony"
)

type (
	S = arrange.Section
	N = score.Note
)

func main() {
	addr := flag.String("addr", "localhost:8087", "the address of the panel's page")
	useMIDI := flag.Bool("midi", false, "play songs and sections from the pads of the first MIDI controller (MIDI learn on the page)")
	p := tutorial.New(100)

	list := playlist.New(p, playlist.Config{Gap: 1, Loop: !tutorial.Rendering()},
		playlist.Song{Name: "Morning", Build: morning},
		playlist.Song{Name: "Groove", Build: groove},
		playlist.Song{Name: "Evening", Build: evening},
	)
	defer list.Close()

	mix := mixer.New(p)
	songs := mix.Channel("songs", list, mixer.StripConfig{})

	if !tutorial.Rendering() {
		ctl := panel.New(p, "Setlist")
		ctl.Section("Songs")
		list.Controls(ctl, "Songs")
		ctl.Section("Sections")
		list.SectionControls(ctl, "Sections")
		ctl.Section("Song")
		list.SongControls(ctl, "Song")
		ctl.Section("Mix")
		ctl.Fader("Volume", -40, 6, panel.Linear, panel.Message(songs, "volume"), panel.Default(0), panel.Unit("dB"), panel.Smooth(0.03))
		ctl.Meter("Master", mix.Master())
		if err := ctl.Presets("setlist-presets"); err != nil {
			log.Fatal(err)
		}
		if *useMIDI {
			d, err := midi.OpenDevice("")
			if err != nil {
				log.Fatalf("%v (go run ./cmd/examples/keyboard -list shows the devices)", err)
			}
			defer d.Close()
			kb := keyboard.New(d, keyboard.Config{Controls: map[int]string{}}) // its pads and knobs only
			kb.AddTo(p)
			ctl.MIDI(kb)
			if err := ctl.KeepBindings("setlist-midi.json"); err != nil {
				log.Fatal(err)
			}
		}
		go func() {
			if err := ctl.Serve(*addr); err != nil {
				log.Fatal(err)
			}
		}()
		defer ctl.Close()
		fmt.Printf("the setlist: http://%s\n", *addr)
	}
	tutorial.Play(p, mix.Master(), list.Duration())
}

// withHall gives a song a mixer with a hall of its own
func withHall(p *psalm.Psalm, decay float64) *mixer.Mixer {
	mix := mixer.New(p)
	cfg := fdnreverb.DefaultConfig()
	cfg.Decay = decay
	cfg.Mix = 1
	mix.Bus("hall", mixer.StripConfig{}, fdnreverb.New(cfg).AddTo(p))
	return mix
}

var played = score.Feel{Timing: 8, Velocity: 0.08}

// finish adds the arrangement and gives the song to the playlist, with its transport and its panel
func finish(p *psalm.Psalm, form *arrange.Form, mix *mixer.Mixer, ctl *panel.Panel) playlist.Built {
	arrangement := arrange.NewArrangement(form)
	arrangement.AddTo(p)
	return playlist.Built{Out: mix.Master(), Length: arrangement.Duration(), Transport: form.Transport(), Panel: ctl}
}

// level adds a fader for the volume of a channel to a song's panel
func level(ctl *panel.Panel, name string, strip *mixer.Strip, db float64) {
	ctl.Fader(name, -40, 6, panel.Linear, panel.Message(strip, "volume"), panel.Default(db), panel.Unit("dB"), panel.Smooth(0.03))
}

// morning: a piano and a flute at 72, in G; the bridge is off
func morning(p *psalm.Psalm) playlist.Built {
	p.Env().SetBPM(72)
	form := arrange.NewForm(S{Name: "Intro", Bars: 1}, S{Name: "Verse", Bars: 2}, S{Name: "Bridge", Bars: 2}, S{Name: "End", Bars: 1})
	form.Transport().SetEnabled("Bridge", false)
	key := harmony.MustKey("G major")
	song, bridge := key.MustProgression("I V vi IV"), key.MustProgression("ii V iii vi")
	chordOf := func(s S, bar int) harmony.Chord {
		if s.Name == "Bridge" {
			return bridge[bar*2%4]
		}
		return song[(s.StartBar+bar)%4]
	}
	keys := piano.NewPoly(piano.WarmPreset(), 16).AddTo(p)
	flute := fm.NewPoly(fm.FlutePreset(), 2).AddTo(p)
	form.Part(p, keys, "piano", score.Config{Velocity: 0.5, Feel: played}, func(s S) []N {
		return arrange.PerBar(s, func(bar int) []N {
			c := chordOf(s, bar)
			v := c.Close(59)
			if s.Name == "End" {
				return []N{score.Chord(append([]float64{song[0].BassPitch(43)}, song[0].Close(59)...), 4)}
			}
			return score.Notes(1, c.BassPitch(43), v[0], v[1], v[2])
		})
	})
	form.Part(p, flute, "flute", score.Config{Amp: 0.7, Feel: played}, func(s S) []N {
		switch s.Name {
		case "Verse":
			return score.MustParse(`B5:2 D6:1 B5 | A5:3 D5:1`)
		case "Bridge":
			return score.MustParse(`C6:3 B5:1 | B5:4`)
		}
		return nil
	})
	mix := withHall(p, 2.4)
	pianoStrip := mix.Channel("piano", keys, mixer.StripConfig{Volume: -10, Sends: map[string]float64{"hall": 0.3}})
	fluteStrip := mix.Channel("flute", flute, mixer.StripConfig{Volume: -12, Pan: 0.2, Sends: map[string]float64{"hall": 0.35}})

	// the song's own controls: the piano, the flute and their hall
	ctl := panel.New(p, "Morning")
	ctl.Section("Levels")
	level(ctl, "Piano", pianoStrip, -10)
	level(ctl, "Flute", fluteStrip, -12)
	ctl.Knob("Hall", 0, 1, panel.Linear, panel.Targets(panel.Message(pianoStrip, "send.hall"), panel.Message(fluteStrip, "send.hall")), panel.Default(0.3))
	return finish(p, form, mix, ctl)
}

// groove: bass and keys at 104, in C minor; the break (keys alone) is off
func groove(p *psalm.Psalm) playlist.Built {
	p.Env().SetBPM(104)
	form := arrange.NewForm(S{Name: "A", Bars: 2}, S{Name: "B", Bars: 2, Dynamics: 1.1}, S{Name: "Break", Bars: 2}, S{Name: "Out", Bars: 2})
	form.Transport().SetEnabled("Break", false)
	chords := harmony.MustKey("C minor").MustProgression("i VI III VII")
	chordOf := func(s S, bar int) harmony.Chord { return chords[(s.StartBar+bar)%4] }
	bass := subtractive.NewPoly(subtractive.BassPreset(), 2).AddTo(p)
	keys := fm.NewPoly(fm.EPianoPreset(), 12).AddTo(p)
	form.Part(p, bass, "bass", score.Config{Velocity: 0.7, Feel: played}, func(s S) []N {
		if s.Name == "Break" {
			return nil
		}
		return arrange.PerBar(s, func(bar int) []N {
			r := chordOf(s, bar).BassPitch(36)
			return score.Shape(score.Notes(0.5, r, 0, r, r+12, 0, r+7, r, 0), score.Velocities(0.9, 0, 0.6, 0.7, 0, 0.6, 0.5, 0))
		})
	})
	form.Part(p, keys, "keys", score.Config{Velocity: 0.5, Feel: played}, func(s S) []N {
		return arrange.PerBar(s, func(bar int) []N {
			c := score.Chord(chordOf(s, bar).Close(60), 0.5)
			if s.Name == "Break" {
				return score.Shape(score.Repeat([]N{c}, 8), score.Velocities(0.7, 0.3, 0.5, 0.3), score.Hold(0.3))
			}
			return score.Shape([]N{score.Rest(0.5), c, score.Rest(1), score.Rest(0.5), c, score.Rest(1)}, score.Hold(0.3))
		})
	})
	mix := withHall(p, 1.6)
	bassStrip := mix.Channel("bass", bass, mixer.StripConfig{Volume: -20})
	keysStrip := mix.Channel("keys", keys, mixer.StripConfig{Volume: -20, Pan: -0.25, Sends: map[string]float64{"hall": 0.2}})

	// the song's own controls: its tempo (only this song's), the bass and the keys
	ctl := panel.New(p, "Groove")
	ctl.Section("Groove")
	ctl.Fader("Tempo", 90, 120, panel.Linear, panel.Func(func(v float64) { p.Env().SetBPM(v) }), panel.Default(104), panel.Step(1), panel.Unit("BPM"))
	level(ctl, "Bass", bassStrip, -20)
	level(ctl, "Keys", keysStrip, -20)
	ctl.Toggle("Mute bass", panel.Message(bassStrip, "mute"))
	return finish(p, form, mix, ctl)
}

// evening: a pad and a felt piano at 60, in F; the night (the pad alone, lower) is off
func evening(p *psalm.Psalm) playlist.Built {
	p.Env().SetBPM(60)
	form := arrange.NewForm(S{Name: "Dusk", Bars: 2}, S{Name: "Night", Bars: 2, Dynamics: 0.8}, S{Name: "Last", Bars: 1})
	form.Transport().SetEnabled("Night", false)
	chords := harmony.MustKey("F major").MustProgression("I iii IV I")
	night := harmony.MustKey("F major").MustProgression("vi iii")
	pad := wavetable.NewPoly(wavetable.PadPreset(gen.Brightness(24)), 8).AddTo(p)
	felt := piano.NewPoly(piano.FeltPreset(), 16).AddTo(p)
	form.Part(p, pad, "pad", score.Config{Amp: 0.5}, func(s S) []N {
		return arrange.PerBar(s, func(bar int) []N {
			c := chords[bar]
			if s.Name == "Night" {
				c = night[bar]
			}
			return []N{score.Chord(c.Close(53), 4)}
		})
	})
	form.Part(p, felt, "piano", score.Config{Velocity: 0.45, Feel: played}, func(s S) []N {
		switch s.Name {
		case "Dusk":
			return arrange.PerBar(s, func(bar int) []N {
				return []N{score.Chord([]float64{chords[bar].BassPitch(41)}, 1), score.Chord(chords[bar].Close(65), 3)}
			})
		case "Last":
			return []N{score.Chord(append([]float64{chords[0].BassPitch(41)}, chords[0].Close(65)...), 4)}
		}
		return nil
	})
	mix := withHall(p, 3.2)
	padStrip := mix.Channel("pad", pad, mixer.StripConfig{Volume: -18, Sends: map[string]float64{"hall": 0.4}})
	pianoStrip := mix.Channel("piano", felt, mixer.StripConfig{Volume: -8, Sends: map[string]float64{"hall": 0.35}})

	// the song's own controls: the pad and the piano
	ctl := panel.New(p, "Evening")
	ctl.Section("Levels")
	level(ctl, "Pad", padStrip, -18)
	level(ctl, "Piano", pianoStrip, -8)
	return finish(p, form, mix, ctl)
}
