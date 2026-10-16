package main

// "Exodus": the Israelites leave Egypt, the sea parts, they wander through the desert and come to the
// promised land. Everything sounds from the physically modelled instruments of the library: the
// waveguide plucked strings, pipes and drums, the modal bells and bars, the piano and the organ.
// Nothing else (no samples, no oscillator synths) but the voice that speaks the verses of every chapter
// (clips of the Piper speech synthesizer, see assets/sounds/exodus; the music gives way a little under it). The effects colour it without hardening it: granulator clouds on the
// glass harmonica and the shakuhachi that open out with the story, a chorus on the clarinets and the harp,
// tape and ping pong echoes on the oud and the kalimba, a slow rotary speaker on the soft organ, and the
// shimmer (a reverb with an octave up in its tail) that the sea, the waters, the desert, the years and the
// land send their bells, glass, flutes and harp into.
//
//	Egypt              slaves' lament in D minor: an organ pedal drone, the frame drum of the work, the anvil
//	                   of the brick makers, an alto flute
//	The night          Passover: twelve strokes of a bell, a heart beating faster, midnight
//	The exodus         the march out: clarinets, recorder, marimba, taiko, bodhran and tambourine, the pillar of
//	                   cloud and fire in the organ
//	Pharaoh pursues    galloping drums, a rising glass harmonica, the cry to the LORD
//	The sea            the wind (the piano sweeping the keyboard, the harp rising, space tines rolling), the walls
//	                   of water (the organ, the harp falling, the tines rolling harder), the crossing on dry ground
//	                   (a bansuri, angel bells)
//	The waters return  tam-tam, gong, drums, a fast roll of space tines, the piano falling, silence
//	Miriam's song      the dance on the far shore in F major: recorder, tambourines, bodhran, harp
//	The wilderness     a bansuri alone, an oud, drops of manna (kalimba), a bowl; the heat
//	Sinai              thunder, the ten words as ten strokes of a bell, the organ's full chorus
//	Forty years        the lament remembered on the shakuhachi
//	The promised land  the lament lifted into D major, bells and the full organ
//
//	go run ./cmd/songs/exodus                                      play it
//	go run ./cmd/songs/exodus -render exodus.wav -stems -seed 7    render it, a stem per mixer strip
//	go run ./cmd/songs/exodus -from sea                            start at a chapter (a name or a number)
//	go run ./cmd/songs/exodus -profile                             the time every module takes
//
// The files: form.go (the chapters, their chords and voicings), parts.go and song.go (the tunes
// and what every instrument plays), orchestra.go (the instruments on the mixer), arrangement.go
// (the tempo, the pedal).

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/tools/render"
)

func main() {
	var (
		renderTo = flag.String("render", "", "render one pass of the piece to this WAV file instead of playing it")
		stems    = flag.Bool("stems", false, "with -render, also write every mixer strip to its own file")
		profile  = flag.Bool("profile", false, "render the piece offline and report the time every module takes")
		from     = flag.String("from", "", "start at a chapter, by name or number (-from sea, -from 5): the chapters before it are skipped")
		seed     = flag.Uint64("seed", 0, "seed for the random parts (feel, breath): the same seed plays the same, 0 is new every run")
	)
	flag.Parse()
	if *seed != 0 {
		psalm.Seed(*seed)
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
	if writeParts(p, o) == 0 {
		fmt.Println("harmony check: every part fits its chords")
	}
	arrangement := o.arrangement()
	if *from != "" {
		if err := startFrom(*from); err != nil {
			fmt.Println(err)
			os.Exit(2)
		}
	}
	arrangement.AddTo(p)
	p.Out(o.mix.Master())

	fmt.Println("Exodus")
	fmt.Print(arrangement.Schedule())

	opts := render.Options{Seconds: arrangement.Duration(), Tail: 10}
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

// startFrom skips the chapters before the one asked for: the number (from 1) or the name, or part of
// the name, in any case ("sea" is The sea, "land" The promised land)
func startFrom(want string) error {
	want = strings.ToLower(strings.TrimSpace(want))
	first := -1
	if n, err := strconv.Atoi(want); err == nil && n >= 1 && n <= len(chapters) {
		first = n - 1
	}
	for i, c := range chapters {
		if first < 0 && strings.Contains(strings.ToLower(c.name), want) {
			first = i
		}
	}
	if first < 0 {
		var names []string
		for i, c := range chapters {
			names = append(names, fmt.Sprintf("%d %s", i+1, c.name))
		}
		return fmt.Errorf("-from %q: no such chapter, the chapters are: %s", want, strings.Join(names, ", "))
	}
	tr := form.Transport()
	for _, c := range chapters[:first] {
		tr.SetEnabled(c.name, false)
	}
	return nil
}
