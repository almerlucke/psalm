// Step 17: a playlist, songs one after the other.
//
// Every step so far was one song in one patch. A playlist plays songs one after the other in one
// patch, and every song is a patch of its own: its own tempo, its own instruments and hall, its own
// clock that starts at 0 when the song starts. A song is a name and a function building it:
//
//	func morning(p *psalm.Psalm) playlist.Built {
//		p.Env().SetBPM(72)                          the song's own tempo
//		...                                         a form, parts, an arrangement, a mixer, added to p
//		return playlist.Built{Out: mix.Master(), Length: arrangement.Duration()}
//	}
//
//	list := playlist.New(p, playlist.Config{Gap: 1}, playlist.Song{Name: "Morning", Build: morning}, ...)
//
// Built says what to hear (the master of the song's mixer) and how long the song plays. When a
// song reaches its length its clock holds: no new notes, and the form doesn't loop back to its
// start, while what sounds rings out (the last chord, the hall). Then the next song starts:
//
//	Gap        seconds between two songs, the song before ringing on
//	Crossfade  seconds the next song starts early, the two fading into each other
//	Loop       after the last song the first again
//
// Playing through the sound card, a song is built when it is needed: the next one in the background
// while one plays, so it starts in time; it is built in its own patch, so the rest of the program
// can go on building meanwhile. A render builds every
// song at the start, so it is the same every time, and list.Duration() is the length of the list.
//
// On the page (list.Controls, step 14): a tile per song, tap one to play it next or its dot to pass
// it over (step 16 has the same tiles for sections); Next skips to the next song; Start/Stop and
// Restart (back to the first song) in the header. From code: list.QueueSong("Evening"),
// list.Skip(), p.Restart().
//
//	go run ./cmd/examples/tutorial/17-playlist                  then open http://localhost:8087
//	go run ./cmd/examples/tutorial/17-playlist -crossfade 3     the songs fading into each other
//	go run ./cmd/examples/tutorial/17-playlist -render list.wav
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/almerlucke/psalm"
	"github.com/almerlucke/psalm/cmd/examples/tutorial/tutorial"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/modules/playlist"
	"github.com/almerlucke/psalm/synths/dx"
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
	crossfade := flag.Float64("crossfade", 0, "seconds the songs fade into each other (0: one after the other, with a gap)")
	p := tutorial.New(100) // the tempo of the patch around the songs: every song sets its own

	// --- the playlist: three songs, each built in a patch of its own ---

	list := playlist.New(p, playlist.Config{Gap: 1, Crossfade: *crossfade, Loop: !tutorial.Rendering()},
		playlist.Song{Name: "Morning", Build: morning},
		playlist.Song{Name: "Groove", Build: groove},
		playlist.Song{Name: "Evening", Build: evening},
	)
	defer list.Close()

	// --- the patch around it: a mixer and the page ---

	mix := mixer.New(p)
	songs := mix.Channel("songs", list, mixer.StripConfig{})

	if !tutorial.Rendering() {
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
		fmt.Printf("the playlist: http://%s\n", *addr)
	}
	tutorial.PlayUntil(p, mix.Master(), list.Duration(), list.Done) // rendering: until the last song rang out
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

// morning: a piano and a flute at 72, in G, ending on a long chord
func morning(p *psalm.Psalm) playlist.Built {
	p.Env().SetBPM(72)
	form := arrange.NewForm(S{Name: "Verse", Bars: 3}, S{Name: "End", Bars: 1})
	chords := harmony.MustKey("G major").MustProgression("I V vi IV")
	keys := piano.NewPoly(piano.WarmPreset(), 16).AddTo(p)
	flute := dx.NewPoly(dx.FlutePreset(), 2, psalm.Stereo).AddTo(p)
	form.Part(p, keys, "piano", score.Config{Velocity: 0.5, Feel: played}, func(s S) []N {
		return arrange.PerBar(s, func(bar int) []N {
			if s.Name == "End" {
				c := chords[3]
				return []N{score.Chord(append([]float64{c.BassPitch(43)}, c.Close(59)...), 4)}
			}
			c := chords[bar].Close(59)
			return score.Notes(1, chords[bar].BassPitch(43), c[0], c[1], c[2])
		})
	})
	form.Part(p, flute, "flute", score.Config{Amp: 0.7, Feel: played}, arrange.Only("Verse", func(S) []N {
		return score.MustParse(`B5:2 D6:1 B5 | A5:3 D5:1 | E5:2 G5:1 B5`)
	}))
	arrangement := arrange.NewArrangement(form)
	arrangement.AddTo(p)
	mix := withHall(p, 2.4)
	mix.Channel("piano", keys, mixer.StripConfig{Volume: -10, Sends: map[string]float64{"hall": 0.3}})
	mix.Channel("flute", flute, mixer.StripConfig{Volume: -12, Pan: 0.2, Sends: map[string]float64{"hall": 0.35}})
	return playlist.Built{Out: mix.Master(), Length: arrangement.Duration()}
}

// groove: bass and keys at 104, in C minor
func groove(p *psalm.Psalm) playlist.Built {
	p.Env().SetBPM(104)
	form := arrange.NewForm(S{Name: "Groove", Bars: 4})
	chords := harmony.MustKey("C minor").MustProgression("i VI III VII")
	bass := subtractive.NewPoly(subtractive.BassPreset(), 2).AddTo(p)
	keys := dx.NewPoly(dx.EPianoPreset(), 12, psalm.Stereo).AddTo(p)
	form.Part(p, bass, "bass", score.Config{Velocity: 0.7, Feel: played}, func(s S) []N {
		return arrange.PerBar(s, func(bar int) []N {
			r := chords[bar].BassPitch(36)
			return score.Shape(score.Notes(0.5, r, 0, r, r+12, 0, r+7, r, 0), score.Velocities(0.9, 0, 0.6, 0.7, 0, 0.6, 0.5, 0))
		})
	})
	form.Part(p, keys, "keys", score.Config{Velocity: 0.5, Feel: played}, func(s S) []N {
		return arrange.PerBar(s, func(bar int) []N {
			c := score.Chord(chords[bar].Close(60), 0.5)
			return score.Shape([]N{score.Rest(0.5), c, score.Rest(1), score.Rest(0.5), c, score.Rest(1)}, score.Hold(0.3))
		})
	})
	arrangement := arrange.NewArrangement(form)
	arrangement.AddTo(p)
	mix := withHall(p, 1.6)
	mix.Channel("bass", bass, mixer.StripConfig{Volume: -17})
	mix.Channel("keys", keys, mixer.StripConfig{Volume: -17, Pan: -0.25, Sends: map[string]float64{"hall": 0.2}})
	return playlist.Built{Out: mix.Master(), Length: arrangement.Duration()}
}

// evening: a pad and a felt piano at 60, in F, ending on a long chord
func evening(p *psalm.Psalm) playlist.Built {
	p.Env().SetBPM(60)
	form := arrange.NewForm(S{Name: "Evening", Bars: 4})
	chords := harmony.MustKey("F major").MustProgression("I iii IV I")
	pad := wavetable.NewPoly(wavetable.PadPreset(gen.Brightness(24)), 8).AddTo(p)
	felt := piano.NewPoly(piano.FeltPreset(), 16).AddTo(p)
	form.Part(p, pad, "pad", score.Config{Amp: 0.5}, func(s S) []N {
		return arrange.PerBar(s, func(bar int) []N { return []N{score.Chord(chords[bar].Close(57), 4)} })
	})
	form.Part(p, felt, "piano", score.Config{Velocity: 0.45, Feel: played}, func(s S) []N {
		return arrange.PerBar(s, func(bar int) []N {
			v := chords[bar].Close(65)
			if bar == 3 {
				return []N{score.Chord(append([]float64{chords[bar].BassPitch(41)}, v...), 4)}
			}
			return []N{score.Chord([]float64{chords[bar].BassPitch(41)}, 1), score.Chord(v, 3)}
		})
	})
	arrangement := arrange.NewArrangement(form)
	arrangement.AddTo(p)
	mix := withHall(p, 3.2)
	mix.Channel("pad", pad, mixer.StripConfig{Volume: -18, Sends: map[string]float64{"hall": 0.4}})
	mix.Channel("piano", felt, mixer.StripConfig{Volume: -8, Sends: map[string]float64{"hall": 0.35}})
	return playlist.Built{Out: mix.Master(), Length: arrangement.Duration()}
}
