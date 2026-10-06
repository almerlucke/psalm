package main

// One jazzy loop played four ways, to hear what feel does: first exactly on the grid like a
// machine, then swung 8ths, a 16th shuffle, and finally swung and played like people: timing and
// dynamics that vary every bar, accented hihats, a strummed electric piano and a bass that lays
// back a little. The feel of every part changes per section (SetFeel at the start of a section).
//
// The music is written with score, arrange and harmony: a ii V I VI progression in C, voiced with
// voice leading, a walking bass with chromatic approach notes, comping on the off beats

import (
	"fmt"
	"maps"
	"slices"

	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/drums"
	"github.com/almerlucke/psalm/synths/fm"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/tools/harmony"
	"github.com/almerlucke/sndfile"
)

var form = arrange.NewForm(
	arrange.Section{Name: "Robot", Bars: 4},
	arrange.Section{Name: "Swing", Bars: 4},
	arrange.Section{Name: "Shuffle", Bars: 4},
	arrange.Section{Name: "Human", Bars: 4},
)

// the feel of every part per section
var feels = map[string]map[string]score.Feel{
	"Robot": {},
	"Swing": {
		"hats": {Swing: 0.66}, "snare": {Swing: 0.66}, "kick": {Swing: 0.66},
		"bass": {Swing: 0.66}, "keys": {Swing: 0.66},
	},
	"Shuffle": {
		"hats": {Swing: 0.62, SwingUnit: 0.25}, "snare": {Swing: 0.62, SwingUnit: 0.25}, "kick": {Swing: 0.62, SwingUnit: 0.25},
		"bass": {Swing: 0.62, SwingUnit: 0.25}, "keys": {Swing: 0.62, SwingUnit: 0.25},
	},
	"Human": {
		"hats":  {Swing: 0.64, Timing: 6, Velocity: 0.15, Accents: []float64{1, 0.5, 0.75, 0.5}, AccentUnit: 0.25},
		"snare": {Swing: 0.64, Timing: 8, Velocity: 0.2},
		"kick":  {Swing: 0.64, Timing: 4, Velocity: 0.1},
		"bass":  {Swing: 0.64, Timing: 6, Push: 12, Velocity: 0.12}, // laid back
		"keys":  {Swing: 0.64, Timing: 10, Strum: 18, Velocity: 0.15},
	},
}

// drum turns a pattern of steps ("x" a hit, "." a rest) lasting step beats into notes playing a
// sound
func drum(sound string, step float64, pattern string) []score.Note {
	var notes []score.Note
	for _, c := range pattern {
		switch c {
		case 'x':
			notes = append(notes, score.Note{Pitches: []float64{60}, Beats: step, Params: map[string]any{"sound": sound, "speed": 1.0}})
		case '.':
			notes = append(notes, score.Rest(step))
		}
	}
	return notes
}

func main() {
	psalm.PushConfig(psalm.Config{BPM: 108, PPQ: 240, SampleRate: 44100, BufferSize: 1024})

	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	// --- harmony: ii V I VI in C, voice led ---
	chords := harmony.MustProgression("Dm7 G7 Cmaj7 A7")
	voicings := [][]float64{harmony.MustChord("Dm7").Close(60)}
	for _, c := range chords[1:] {
		voicings = append(voicings, harmony.VoiceLead(voicings[len(voicings)-1], c))
	}

	// a walking bass: root, third, fifth, then a semitone below the next root
	var walk []score.Note
	for i, c := range chords {
		root := c.BassPitch(36)
		next := chords[(i+1)%len(chords)].BassPitch(36)
		walk = append(walk, score.Notes(1, root, root+float64(c.Intervals[1]), root+7, next-1)...)
	}

	// comping on the off beats: the and of 1 and the and of 3
	var comp []score.Note
	for _, v := range voicings {
		comp = append(comp, score.Rest(0.5), score.Chord(v, 1.5), score.Rest(0.5), score.Chord(v, 1.5))
	}

	// --- instruments ---
	kit := sndfile.SoundBank[float32]{
		"kick":  sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/kick/Cymatics - Humble Sit Down Kick - D.wav", 4),
		"snare": sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/snare/Cymatics - Humble Adequate Snare - E.wav", 4),
		"hat":   sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/hihat/Cymatics - Humble Closed Hihat 4.wav", 4),
	}
	drumKit := drums.NewDrums(kit, 16, 2).AddTo(p)
	bass := subtractive.NewPoly(subtractive.BassPreset(), 2).AddTo(p)
	keys := fm.NewPoly(fm.EPianoPreset(), 12).AddTo(p)

	// every part plays its loop in every section
	loop := func(notes []score.Note) arrange.Score {
		return func(s arrange.Section) []score.Note {
			return score.Repeat(notes, s.Bars/int(score.Beats(notes)/form.BeatsPerBar))
		}
	}
	seqs := map[string]*score.Sequencer{
		"hats":  form.Part(p, drumKit, "hats", score.Config{Amp: 0.2}, loop(drum("hat", 0.25, "x.xxx.xxx.xxx.xx"))), // 16ths
		"snare": form.Part(p, drumKit, "snare", score.Config{Amp: 0.4}, loop(drum("snare", 0.5, "..x...x."))),
		"kick":  form.Part(p, drumKit, "kick", score.Config{Amp: 0.55}, loop(drum("kick", 0.5, "x.....x."))),
		"bass":  form.Part(p, bass, "bass", score.Config{Amp: 0.45, Legato: 0.85}, loop(walk)),
		"keys":  form.Part(p, keys, "keys", score.Config{Amp: 0.12, Legato: 0.6, Velocity: 0.7}, loop(comp)),
	}

	// the walking bass and the comping fit the chords
	timeline := harmony.Timeline(4, chords...)
	fmt.Print(harmony.Report("bass", harmony.Check(walk, timeline)))
	fmt.Print(harmony.Report("keys", harmony.Check(comp, timeline)))

	// --- mix ---
	mix := mixer.New(p)
	room := fdnreverb.DefaultConfig()
	room.Decay = 1.2
	room.Size = 0.4
	room.Mix = 1
	mix.Bus("room", mixer.StripConfig{}, fdnreverb.New(room).AddTo(p))
	mix.Channel("drums", drumKit, mixer.StripConfig{Volume: -5, Sends: map[string]float64{"room": 0.15}})
	mix.Channel("bass", bass, mixer.StripConfig{Pan: -0.1})
	mix.Channel("keys", keys, mixer.StripConfig{Pan: 0.25, Sends: map[string]float64{"room": 0.3}})

	// --- the feel changes per section ---
	names := slices.Sorted(maps.Keys(seqs))
	arrange.NewArrangement(form).
		OnSection(func(pos arrange.Position) {
			fmt.Printf("bar %2d  %s\n", pos.Bar+1, pos.Section.Name)
			for _, name := range names {
				seqs[name].SetFeel(feels[pos.Section.Name][name])
			}
		}).
		AddTo(p)

	fmt.Print(form.Schedule())
	p.Out(mix.Master())

	_ = p.Listen()
}
