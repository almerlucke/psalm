package main

// "Creation": the seven days of Genesis 1 and 2, to the glory of God the Father, the Son and the
// Holy Spirit. "In the beginning was the Word, and the Word was with God, and the Word was God.
// All things were made by him" (John 1:1, 3): the Word through whom all was made is Jesus Christ,
// our Messiah and Redeemer, and the piece ends at his cross (Colossians 1:16, 20).
//
// Every day is an evening and a morning ("and the evening and the morning were the first day"):
// six bars of evening, six of morning. The seventh day, the day God blessed and rested, is 21 bars
// (three times seven): the Doxology on the Old Hundredth ("Praise Father, Son, and Holy Ghost",
// Louis Bourgeois, 1551) and a plagal Amen.
//
//	day 1  light       the earth without form and void (an open fifth, no third), the Spirit upon
//	                   the waters (a breathing choir), John 1:1 spoken into the darkness; then "Let
//	                   there be light": the darkness lifts and the first major chord sounds
//	day 2  firmament   the waters divided: strings very high and very low, nothing between them; in
//	                   the morning the heaven fills the middle (a choir)
//	day 3  land        the dry land (timpani, bass) and the seas (waves); in the morning the plants
//	                   "yielding seed after his kind": a canon, flute, oboe and clarinet, every voice
//	                   the same melody
//	day 4  lights      night: the moon (flute) and the stars (celesta, bells); morning: the sun rises
//	                   (horns) and "the morning stars sang together" (Job 38:7, the choir)
//	day 5  creatures   the great creatures of the sea (a low formant voice), the birds (flute and
//	                   piccolo birdsong) over the sea
//	day 6  mankind     the beasts of the earth; man in the image of God, male and female: the soprano
//	                   sings the theme, the horn answers with the figure of the cross, the piano
//	day 7  rest        the Doxology on organ and choir in a church, the Amen, the cross in the horn
//	                   leading into the last chord
//
// The divine order in the sound: three beats to the bar (the Trinity) in E flat major (three flats,
// the key of Bach's trinitarian St. Anne prelude and fugue), tuned in just intonation: every note
// a pure ratio of the harmonic series God put into every vibrating string, and only the chords
// that sound pure in it (I, IV, V, vi and iii). The verse of every day is printed as it begins
// (King James Version).
//
//	go run ./cmd/songs/creation                                     play it
//	go run ./cmd/songs/creation -tuning equal                       in equal temperament instead
//	go run ./cmd/songs/creation -render creation.wav -stems -seed 7 render it, a stem per strip
//	go run ./cmd/songs/creation -profile                            the time every module takes
//
// The same piece is also made as a patch, with the patcher's modules and scripts to read and change while it plays: cmd/songs/creationpatch.
//
// The files: form.go (the days, their chords and voicings), melodies.go, orchestra.go (the
// instruments on the mixer), parts.go (what every instrument plays, day by day) and arrangement.go
// (the light, the voices of creation coming in, the tempo).

import (
	"flag"
	"fmt"

	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/tools/harmony"
	"github.com/almerlucke/psalm/tools/render"
)

func main() {
	var (
		renderTo  = flag.String("render", "", "render one pass of the piece to this WAV file instead of playing it")
		stems     = flag.Bool("stems", false, "with -render, also write every mixer strip to its own file")
		profile   = flag.Bool("profile", false, "render the piece offline and report the time every module takes")
		seed      = flag.Uint64("seed", 0, "seed for the random parts (grains, feel): the same seed plays the same, 0 is new every run")
		tuningArg = flag.String("tuning", "just", "just (pure ratios) or equal (equal temperament)")
		a4        = flag.Float64("a4", 440, "the reference pitch of A4 in Hz")
	)
	flag.Parse()
	if *seed != 0 {
		psalm.Seed(*seed)
	}
	var tune score.Tuning
	switch *tuningArg {
	case "just":
		tune = key.Just(*a4) // every note a pure ratio of E flat
	case "equal":
		tune = harmony.EqualTemperament(*a4)
	default:
		panic(fmt.Sprintf("tuning %q: just or equal", *tuningArg))
	}

	psalm.PushConfig(psalm.Config{BPM: bpm, PPQ: 240, SampleRate: 44100, BufferSize: 1024})
	var p *psalm.Psalm
	if *renderTo != "" || *profile {
		p = psalm.New(nil)
	} else {
		p = psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	}
	defer p.Close()

	o := newOrchestra(p)
	if writeParts(p, o, tune) == 0 {
		fmt.Println("harmony check: every part fits its chords")
	}
	arrangement := o.arrangement()
	arrangement.AddTo(p)
	p.Out(o.mix.Master())

	fmt.Println("Creation")
	fmt.Println("In the beginning was the Word, and the Word was with God, and the Word was God. All things were made by him. (John 1:1, 3)")
	fmt.Print(arrangement.Schedule())

	opts := render.Options{Seconds: arrangement.Duration(), Tail: 12}
	switch {
	case *profile:
		fmt.Print(render.RunProfiled(p, opts))
	case *renderTo != "":
		var sources map[string]psalm.Modular
		if *stems {
			sources = render.MixerStems(o.mix)
		}
		stats, err := render.Stems(p, *renderTo, opts, sources)
		if err != nil {
			panic(err)
		}
		fmt.Println(stats)
		fmt.Print(o.mix.Meters())
	default:
		_ = p.Listen()
	}
}
