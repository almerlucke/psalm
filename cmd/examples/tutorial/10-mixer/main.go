// Step 10: the mixer.
//
// Every instrument gets a channel on the mixer: its volume (dB), its pan (-1 left to 1 right) and
// how much of it goes to each effect bus.
//
//	mix.Channel("keys", keys, mixer.StripConfig{Volume: -2, Pan: -0.3, Sends: map[string]float64{"hall": 0.3}})
//
// A bus is an effect shared by many channels: a reverb, a delay. Each channel sends some of its
// sound to it (its send amount), the bus returns the effect on its own channel. Set the effect
// fully wet (Mix 1): the dry sound stays on the channels.
//
// A group is a sub mix: channels with Group set play into it, through the effects inserted in it,
// then to the master. Useful to treat instruments together (all strings through one filter, one
// chorus on the band) or to set their level together.
//
// The channels are modules like any other: lanes and events can change their "volume", "pan",
// "mute" and "send.<bus>". Here the first section is dry (the sends at 0), the second wet, and in
// the outro a recording of waves fades in under the music.
//
//	go run ./cmd/examples/tutorial/10-mixer
//	go run ./cmd/examples/tutorial/10-mixer -render mixer.wav   also prints the level of every channel
package main

import (
	"fmt"

	"github.com/almerlucke/psalm/cmd/examples/tutorial/tutorial"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/effects/chorus"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/effects/pingpong"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/modules/player"
	"github.com/almerlucke/psalm/synths/fm"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
	"github.com/almerlucke/psalm/tools/harmony"
	"github.com/almerlucke/sndfile"
)

type sends = map[string]float64

func main() {
	p := tutorial.New(96)

	form := arrange.NewForm(
		arrange.Section{Name: "Dry", Bars: 4},
		arrange.Section{Name: "Wet", Bars: 4},
		arrange.Section{Name: "Outro", Bars: 4},
	)
	chords := harmony.MustKey("C major").MustProgression("I vi IV V")

	keys := fm.NewPoly(fm.EPianoPreset(), 12).AddTo(p)
	bass := subtractive.NewPoly(subtractive.BassPreset(), 2).AddTo(p)
	pad := wavetable.NewPoly(wavetable.PadPreset(gen.Brightness(32)), 8).AddTo(p)
	lead := fm.NewPoly(fm.FlutePreset(), 2).AddTo(p)
	waves := player.New(player.Config{SndFile: sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/boat_waves-6099.wav", 4), Loop: true, Speed: 1, Amp: 1})
	waves.AddTo(p)

	form.Part(p, keys, "keys", score.Config{Amp: 0.18}, func(s arrange.Section) []score.Note {
		return arrange.PerBar(s, func(bar int) []score.Note {
			chord := score.Chord(chords[bar%4].Close(60), 1)
			return score.Shape(score.Repeat([]score.Note{chord}, 4), score.Velocities(0.6, 0.4), score.Hold(0.6))
		})
	})
	form.Part(p, bass, "bass", score.Config{Amp: 0.3, Legato: 0.8}, func(s arrange.Section) []score.Note {
		return arrange.PerBar(s, func(bar int) []score.Note {
			r := chords[bar%4].BassPitch(36)
			return score.Notes(2, r, r+7)
		})
	})
	form.Part(p, pad, "pad", score.Config{Amp: 0.15, Legato: 1}, func(s arrange.Section) []score.Note {
		return arrange.PerBar(s, func(bar int) []score.Note { return []score.Note{score.Chord(chords[bar%4].Close(48), 4)} })
	})
	form.Part(p, lead, "lead", score.Config{Amp: 0.35, Velocity: 0.6}, func(s arrange.Section) []score.Note {
		return score.MustParse(`E5:1 G5 C6:2 | E5:1 A5 C6:2 | A5:1 F5 C5:2 | D5:1 G5 B4:2`)
	})

	mix := mixer.New(p)

	// two buses: a hall reverb and an echo, both fully wet
	hall := fdnreverb.DefaultConfig()
	hall.Decay = 3
	hall.Mix = 1
	mix.Bus("hall", mixer.StripConfig{}, fdnreverb.New(hall).AddTo(p))
	mix.Bus("echo", mixer.StripConfig{}, pingpong.New(pingpong.Config{Time: 0.75, Feedback: 0.4, Mix: 1, HighCut: 5000, LowCut: 300}).AddTo(p))

	// a group: the keys and the bass play into it, through a chorus
	mix.Group("band", mixer.StripConfig{Volume: -1}, chorus.New(chorus.Config{Rate: 0.4, Depth: 4, Delay: 20, Voices: 2, Spread: 1, Tone: 9000, Mix: 0.3}).AddTo(p))

	// the channels: level, place in the stereo field, sends
	keysChannel := mix.Channel("keys", keys, mixer.StripConfig{Pan: -0.3, Group: "band", Sends: sends{"hall": 0.3}})
	mix.Channel("bass", bass, mixer.StripConfig{Group: "band"})
	padChannel := mix.Channel("pad", pad, mixer.StripConfig{Volume: -3, Pan: 0.3, Sends: sends{"hall": 0.5}})
	leadChannel := mix.Channel("lead", lead, mixer.StripConfig{Pan: 0.1, Sends: sends{"hall": 0.3, "echo": 0.35}})
	wavesChannel := mix.Channel("waves", waves, mixer.StripConfig{Volume: arrange.Silent, Sends: sends{"hall": 0.2}})

	// the first section dry: the sends of every channel at 0 there, back to their amount after
	dry := func(amount float64) arrange.Lane { return arrange.Sections{"Dry": arrange.Const(0)}.Else(amount) }
	outro := form.Span("Outro")
	arrangement := arrange.NewArrangement(form).
		AutomateMessage(dry(0.3), keysChannel, "send.hall").
		AutomateMessage(dry(0.5), padChannel, "send.hall").
		AutomateMessage(dry(0.3), leadChannel, "send.hall").
		AutomateMessage(dry(0.35), leadChannel, "send.echo").
		// the waves come in over the outro and carry on to its end (a bar past it: no fade out)
		AutomateMessage(arrange.Sections{"Outro": outro.Fade(-4, 0, 5)}.Else(arrange.Silent), wavesChannel, "volume").
		PrintSections(true)
	arrangement.AddTo(p)

	tutorial.Play(p, mix.Master(), arrangement.Duration())
	fmt.Print(mix.Meters()) // the level of every channel, after rendering
}
