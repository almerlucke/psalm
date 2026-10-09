// Playlist: three songs one after the other, each a patch of its own with its own tempo, built
// while the song before plays. A song that ends rings out (its hall tails) under the start of the
// next. On the page: a tile per song (tap one to play it next, its dot to pass it over), Next to
// skip, Start/Stop and Restart (back to the first song) in the header.
//
//	go run ./cmd/examples/playlist                     then open http://localhost:8087
//	go run ./cmd/examples/playlist -crossfade 4        the songs fading into each other
//	go run ./cmd/examples/playlist -render list.wav    the three songs to a file
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/modules/playlist"
	"github.com/almerlucke/psalm/synths/fm"
	"github.com/almerlucke/psalm/synths/piano"
	"github.com/almerlucke/psalm/synths/pulsar"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
	"github.com/almerlucke/psalm/tools/harmony"
	"github.com/almerlucke/psalm/tools/render"
)

type (
	S = arrange.Section
	N = score.Note
)

func main() {
	addr := flag.String("addr", "localhost:8087", "the address of the page")
	renderTo := flag.String("render", "", "render the playlist to this WAV file instead of playing")
	crossfade := flag.Float64("crossfade", 0, "seconds the songs fade into each other (0: one after the other)")
	gap := flag.Float64("gap", 1, "seconds between the songs (without a cross fade)")
	loop := flag.Bool("loop", true, "after the last song the first again (not when rendering)")
	seed := flag.Uint64("seed", 0, "seed for the feel: the same seed plays the same")
	flag.Parse()
	if *seed != 0 {
		psalm.Seed(*seed)
	}

	psalm.PushConfig(psalm.Config{BPM: 100, PPQ: 240, SampleRate: 44100, BufferSize: 512})
	var p *psalm.Psalm
	if *renderTo != "" {
		p = psalm.New(nil)
	} else {
		p = psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	}
	defer p.Close()

	// --- the playlist: every song is built in a patch of its own, when it is needed ---

	list := playlist.New(p, playlist.Config{Gap: *gap, Crossfade: *crossfade, Loop: *loop && *renderTo == ""},
		playlist.Song{Name: "Morning", Build: morning},
		playlist.Song{Name: "Groove", Build: groove},
		playlist.Song{Name: "Evening", Build: evening},
	)
	defer list.Close()

	mix := mixer.New(p)
	songs := mix.Channel("songs", list, mixer.StripConfig{})
	p.Out(mix.Master())

	if *renderTo != "" {
		stats, err := render.ToFile(p, *renderTo, render.Options{Seconds: list.Duration(), Tail: 8})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(stats)
		return
	}

	ctl := panel.New(p, "Playlist")
	ctl.Section("Songs")
	list.Controls(ctl, "Songs")
	ctl.Section("Mix")
	ctl.Fader("Volume", -40, 6, panel.Linear, panel.Message(songs, "volume"), panel.Default(0), panel.Unit("dB"), panel.Smooth(0.03))
	ctl.Meter("Master", mix.Master())
	go func() {
		if err := ctl.Serve(*addr); err != nil {
			log.Fatal(err)
		}
	}()
	defer ctl.Close()
	fmt.Printf("open http://%s (Ctrl-C to stop)\n", *addr)
	_ = p.Listen()
}

// hall gives a song a mixer with a hall
func hall(p *psalm.Psalm, decay float64) *mixer.Mixer {
	mix := mixer.New(p)
	cfg := fdnreverb.DefaultConfig()
	cfg.Decay = decay
	cfg.Mix = 1
	mix.Bus("hall", mixer.StripConfig{}, fdnreverb.New(cfg).AddTo(p))
	return mix
}

var played = score.Feel{Timing: 8, Velocity: 0.08}

// morning: a piano and a flute at 72, in G
func morning(p *psalm.Psalm) playlist.Built {
	p.Env().SetBPM(72)
	form := arrange.NewForm(S{Name: "Intro", Bars: 2}, S{Name: "Verse", Bars: 4}, S{Name: "End", Bars: 2})
	chords := harmony.MustKey("G major").MustProgression("I V vi IV")
	keys := piano.NewPoly(piano.WarmPreset(), 16).AddTo(p)
	flute := fm.NewPoly(fm.FlutePreset(), 2).AddTo(p)
	form.Part(p, keys, "piano", score.Config{Velocity: 0.5, Feel: played}, func(s S) []N {
		return arrange.PerBar(s, func(bar int) []N {
			c := chords[bar%4]
			if s.Name == "End" && bar == 1 {
				return []N{score.Chord(append([]float64{c.BassPitch(43)}, c.Close(59)...), 4)}
			}
			return score.Notes(1, c.BassPitch(43), c.Close(59)[0], c.Close(59)[1], c.Close(59)[2])
		})
	})
	form.Part(p, flute, "flute", score.Config{Amp: 0.7, Feel: played}, arrange.Only("Verse", func(s S) []N {
		return score.MustParse(`B5:2 D6:1 B5 | A5:3 D5:1 | E5:2 G5:1 B5 | C6:4`)
	}))
	arrangement := arrange.NewArrangement(form)
	arrangement.AddTo(p)
	mix := hall(p, 2.4)
	mix.Channel("piano", keys, mixer.StripConfig{Volume: -10, Sends: map[string]float64{"hall": 0.3}})
	mix.Channel("flute", flute, mixer.StripConfig{Volume: -12, Pan: 0.2, Sends: map[string]float64{"hall": 0.35}})
	return playlist.Built{Out: mix.Master(), Length: arrangement.Duration()}
}

// groove: bass, keys and a pulsar mallet at 104, in C minor
func groove(p *psalm.Psalm) playlist.Built {
	p.Env().SetBPM(104)
	form := arrange.NewForm(S{Name: "A", Bars: 4}, S{Name: "B", Bars: 4, Dynamics: 1.1})
	chords := harmony.MustKey("C minor").MustProgression("i VI III VII")
	bass := subtractive.NewPoly(subtractive.BassPreset(), 2).AddTo(p)
	keys := fm.NewPoly(fm.EPianoPreset(), 12).AddTo(p)
	mallet := pulsar.NewPoly(pulsar.MalletPreset(), 8).AddTo(p)
	form.Part(p, bass, "bass", score.Config{Velocity: 0.7, Feel: played}, func(s S) []N {
		return arrange.PerBar(s, func(bar int) []N {
			r := chords[bar%4].BassPitch(36)
			return score.Shape(score.Notes(0.5, r, 0, r, r+12, 0, r+7, r, 0), score.Velocities(0.9, 0, 0.6, 0.7, 0, 0.6, 0.5, 0))
		})
	})
	form.Part(p, keys, "keys", score.Config{Velocity: 0.5, Feel: played}, func(s S) []N {
		return arrange.PerBar(s, func(bar int) []N {
			c := score.Chord(chords[bar%4].Close(60), 0.5)
			return score.Shape([]N{score.Rest(0.5), c, score.Rest(1), score.Rest(0.5), c, score.Rest(1)}, score.Hold(0.3))
		})
	})
	form.Part(p, mallet, "mallet", score.Config{Amp: 0.5, Feel: played}, arrange.Only("B", func(s S) []N {
		return arrange.PerBar(s, func(bar int) []N {
			return harmony.Arp{Pattern: harmony.UpDown, Beats: 0.25, Steps: 16}.Notes(chords[bar%4].Close(72))
		})
	}))
	arrangement := arrange.NewArrangement(form)
	arrangement.AddTo(p)
	mix := hall(p, 1.6)
	mix.Channel("bass", bass, mixer.StripConfig{Volume: -17})
	mix.Channel("keys", keys, mixer.StripConfig{Volume: -19, Pan: -0.25, Sends: map[string]float64{"hall": 0.2}})
	mix.Channel("mallet", mallet, mixer.StripConfig{Volume: -21, Pan: 0.3, Sends: map[string]float64{"hall": 0.3}})
	return playlist.Built{Out: mix.Master(), Length: arrangement.Duration()}
}

// evening: a pad and a felt piano at 60, in F, ending on a long chord
func evening(p *psalm.Psalm) playlist.Built {
	p.Env().SetBPM(60)
	form := arrange.NewForm(S{Name: "Evening", Bars: 6})
	chords := harmony.MustKey("F major").MustProgression("I iii IV ii V I")
	pad := wavetable.NewPoly(wavetable.PadPreset(gen.Brightness(24)), 8).AddTo(p)
	felt := piano.NewPoly(piano.FeltPreset(), 16).AddTo(p)
	form.Part(p, pad, "pad", score.Config{Amp: 0.5}, func(s S) []N {
		return arrange.PerBar(s, func(bar int) []N { return []N{score.Chord(chords[bar].Close(57), 4)} })
	})
	form.Part(p, felt, "piano", score.Config{Velocity: 0.45, Feel: played}, func(s S) []N {
		return arrange.PerBar(s, func(bar int) []N {
			v := chords[bar].Close(65)
			if bar == 5 {
				return []N{score.Chord(append([]float64{chords[bar].BassPitch(41)}, v...), 4)}
			}
			return []N{score.Chord([]float64{chords[bar].BassPitch(41)}, 1), score.Chord(v, 3)}
		})
	})
	arrangement := arrange.NewArrangement(form)
	arrangement.AddTo(p)
	mix := hall(p, 3.2)
	mix.Channel("pad", pad, mixer.StripConfig{Volume: -18, Sends: map[string]float64{"hall": 0.4}})
	mix.Channel("piano", felt, mixer.StripConfig{Volume: -8, Sends: map[string]float64{"hall": 0.35}})
	return playlist.Built{Out: mix.Master(), Length: arrangement.Duration()}
}
