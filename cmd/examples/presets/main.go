// Presets: a tour of the showcase presets of the synths (the shaper synth's all fifty-two), each playing what suits it for two bars: a
// chord for pads and drones, a line for leads, an arpeggio for plucks, a bass line for basses, a beat
// for the drum kits. The name of each prints as it starts.
//
//	go run ./cmd/examples/presets                          all of them
//	go run ./cmd/examples/presets -synth vocal             one synth
//	go run ./cmd/examples/presets -render presets.wav      render it
package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/drumsynth"
	"github.com/almerlucke/psalm/synths/modal"
	"github.com/almerlucke/psalm/synths/organ"
	"github.com/almerlucke/psalm/synths/shaper"
	"github.com/almerlucke/psalm/synths/supersaw"
	"github.com/almerlucke/psalm/synths/vocal"
	"github.com/almerlucke/psalm/synths/waveguide"
	"github.com/almerlucke/psalm/tools/harmony"
	"github.com/almerlucke/psalm/tools/render"
)

type (
	S = arrange.Section
	N = score.Note
)

// style is what a preset plays
type style int

const (
	pad   style = iota // a chord held
	drone              // one low note held
	lead               // a melody
	slow               // a slow melody (for sounds that swell in)
	arp                // a chord broken into sixteenths
	bass               // a bass line in eighths
	held               // a bass note a bar long (for presets that move by themselves)
	hits               // a few single notes, ringing
	beat               // a drum pattern
)

// entry is a preset on the tour
type entry struct {
	synth, preset string
	style         style
	octave        int // the melody or chord moved by octaves
	make          func(p *psalm.Psalm) psalm.Modular
}

func main() {
	only := flag.String("synth", "", "the synth whose presets to play (empty: all)")
	renderTo := flag.String("render", "", "render to this WAV file instead of playing")
	flag.Parse()
	psalm.PushConfig(psalm.Config{BPM: 110, PPQ: 240, SampleRate: 44100, BufferSize: 512})
	var p *psalm.Psalm
	if *renderTo != "" {
		p = psalm.New(nil)
	} else {
		p = psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	}
	defer p.Close()

	org := func(f func() organ.Config) func(*psalm.Psalm) psalm.Modular {
		return func(p *psalm.Psalm) psalm.Modular { return organ.New(f()).AddTo(p) }
	}
	wg := func(f func() waveguide.Config) func(*psalm.Psalm) psalm.Modular {
		return func(p *psalm.Psalm) psalm.Modular { return waveguide.NewPoly(f(), 8).AddTo(p) }
	}
	md := func(f func() modal.Config) func(*psalm.Psalm) psalm.Modular {
		return func(p *psalm.Psalm) psalm.Modular { return modal.NewPoly(f(), 8).AddTo(p) }
	}
	vc := func(f func() vocal.Config) func(*psalm.Psalm) psalm.Modular {
		return func(p *psalm.Psalm) psalm.Modular { return vocal.NewPoly(f(), 6).AddTo(p) }
	}
	sh := func(f func() shaper.Config) func(*psalm.Psalm) psalm.Modular {
		return func(p *psalm.Psalm) psalm.Modular { return shaper.NewPoly(f(), 8).AddTo(p) }
	}
	ss := func(f func() supersaw.Config, voices int) func(*psalm.Psalm) psalm.Modular {
		return func(p *psalm.Psalm) psalm.Modular { return supersaw.NewPoly(f(), voices).AddTo(p) }
	}
	kit := func(f func() drumsynth.Kit) func(*psalm.Psalm) psalm.Modular {
		return func(p *psalm.Psalm) psalm.Modular {
			return drumsynth.NewPoly(drumsynth.Config{Kit: f()}, 12).AddTo(p)
		}
	}
	tour := []entry{
		{"organ", "screamer", pad, 0, org(organ.Screamer)},
		{"organ", "warble", pad, 0, org(organ.Warble)},
		{"organ", "chime", arp, 0, org(organ.Chime)},
		{"organ", "growlBass", bass, 0, org(organ.GrowlBass)},
		{"organ", "seasick", pad, 0, org(organ.Seasick)},
		{"waveguide", "glassHarp", arp, 0, wg(waveguide.GlassHarp)},
		{"waveguide", "wire", arp, 0, wg(waveguide.Wire)},
		{"waveguide", "eBow", slow, -1, wg(waveguide.EBow)},
		{"waveguide", "ghostBow", slow, -2, wg(waveguide.GhostBow)},
		{"waveguide", "scrape", lead, -2, wg(waveguide.Scrape)},
		{"modal", "alienBell", hits, 0, md(modal.AlienBell)},
		{"modal", "gamelan", lead, 0, md(modal.Gamelan)},
		{"modal", "crystal", arp, 1, md(modal.Crystal)},
		{"modal", "steelPan", lead, 0, md(modal.SteelPan)},
		{"modal", "metalPlate", hits, -1, md(modal.MetalPlate)},
		{"modal", "spaceTine", hits, 1, md(modal.SpaceTine)},
		{"vocal", "talkbox", pad, -1, vc(vocal.Talkbox)},
		{"vocal", "wahChoir", pad, 0, vc(vocal.WahChoir)},
		{"vocal", "throatDrone", drone, 0, vc(vocal.ThroatDrone)},
		{"vocal", "whisper", pad, 0, vc(vocal.Whisper)},
		{"vocal", "alien", lead, 0, vc(vocal.Alien)},
		{"vocal", "babble", pad, 0, vc(vocal.Babble)},
		{"supersaw", "hypersaw", pad, 0, ss(supersaw.Hypersaw, 8)},
		{"supersaw", "wobbleBass", held, 0, ss(supersaw.WobbleBass, 2)},
		{"supersaw", "breathing", pad, 0, ss(supersaw.Breathing, 8)},
		{"supersaw", "swarm", pad, 0, ss(supersaw.Swarm, 8)},
		{"supersaw", "glideLead", lead, 0, ss(supersaw.GlideLead, 1)}, // one voice: it glides note to note
		{"shaper", "subSine", held, 0, sh(shaper.SubSine)},
		{"shaper", "kickSub", hits, -3, sh(shaper.KickSub)},
		{"shaper", "reese", held, 0, sh(shaper.Reese)},
		{"shaper", "acid", bass, 0, sh(shaper.Acid)},
		{"shaper", "wobble", held, 0, sh(shaper.Wobble)},
		{"shaper", "pulseBass", bass, 0, sh(shaper.PulseBass)},
		{"shaper", "fmBass", bass, 0, sh(shaper.FMBass)},
		{"shaper", "growl", held, 0, sh(shaper.Growl)},
		{"shaper", "subFold", held, 0, sh(shaper.SubFold)},
		{"shaper", "supersawLead", lead, 0, sh(shaper.SupersawLead)},
		{"shaper", "squareLead", lead, 0, sh(shaper.SquareLead)},
		{"shaper", "syncLead", lead, 0, sh(shaper.SyncLead)},
		{"shaper", "pluckLead", lead, 0, sh(shaper.PluckLead)},
		{"shaper", "fmLead", lead, 0, sh(shaper.FMLead)},
		{"shaper", "fluteLead", slow, 0, sh(shaper.FluteLead)},
		{"shaper", "brassLead", lead, 0, sh(shaper.BrassLead)},
		{"shaper", "vowel", slow, 0, sh(shaper.Vowel)},
		{"shaper", "reed", lead, 0, sh(shaper.Reed)},
		{"shaper", "warmPad", pad, 0, sh(shaper.WarmPad)},
		{"shaper", "stringPad", pad, 0, sh(shaper.StringPad)},
		{"shaper", "choirPad", pad, 0, sh(shaper.ChoirPad)},
		{"shaper", "airPad", pad, 0, sh(shaper.AirPad)},
		{"shaper", "darkPad", pad, -1, sh(shaper.DarkPad)},
		{"shaper", "evolve", pad, 0, sh(shaper.Evolve)},
		{"shaper", "glassPad", pad, 0, sh(shaper.GlassPad)},
		{"shaper", "foldPad", pad, 0, sh(shaper.FoldPad)},
		{"shaper", "morphPad", pad, 0, sh(shaper.MorphPad)},
		{"shaper", "swarm", pad, 0, sh(shaper.Swarm)},
		{"shaper", "organ", pad, 0, sh(shaper.Organ)},
		{"shaper", "pluck", arp, 0, sh(shaper.Pluck)},
		{"shaper", "harp", arp, 0, sh(shaper.Harp)},
		{"shaper", "kalimba", arp, 1, sh(shaper.Kalimba)},
		{"shaper", "marimba", arp, 0, sh(shaper.Marimba)},
		{"shaper", "ePiano", pad, 0, sh(shaper.EPiano)},
		{"shaper", "clav", arp, -1, sh(shaper.Clav)},
		{"shaper", "vibes", arp, 0, sh(shaper.Vibes)},
		{"shaper", "stab", pad, 0, sh(shaper.Stab)},
		{"shaper", "musicBox", arp, 1, sh(shaper.MusicBox)},
		{"shaper", "pwm", arp, -1, sh(shaper.PWM)},
		{"shaper", "bellMorph", hits, 0, sh(shaper.BellMorph)},
		{"shaper", "glassBell", hits, 0, sh(shaper.GlassBell)},
		{"shaper", "churchBell", hits, -1, sh(shaper.ChurchBell)},
		{"shaper", "gong", hits, -2, sh(shaper.Gong)},
		{"shaper", "metal", hits, -1, sh(shaper.Metal)},
		{"shaper", "glass", hits, 0, sh(shaper.Glass)},
		{"shaper", "drone5", drone, 0, sh(shaper.Drone5)},
		{"shaper", "subDrone", drone, -1, sh(shaper.SubDrone)},
		{"shaper", "crush", arp, -1, sh(shaper.Crush)},
		{"shaper", "laser", hits, 0, sh(shaper.Laser)},
		{"shaper", "riser", drone, -1, sh(shaper.Riser)},
		{"shaper", "impact", hits, -3, sh(shaper.Impact)},
		{"shaper", "zap", hits, 0, sh(shaper.Zap)},
		{"drumsynth", "electro", beat, 0, kit(drumsynth.KitElectro)},
		{"drumsynth", "trap", beat, 0, kit(drumsynth.KitTrap)},
		{"drumsynth", "industrial", beat, 0, kit(drumsynth.KitIndustrial)},
	}
	var chosen []entry
	for _, e := range tour {
		if *only == "" || strings.EqualFold(e.synth, *only) {
			chosen = append(chosen, e)
		}
	}
	if len(chosen) == 0 {
		log.Fatalf("no presets of %q", *only)
	}

	// a section of two bars each
	var sections []S
	for _, e := range chosen {
		sections = append(sections, S{Name: e.synth + " " + e.preset, Bars: 2})
	}
	form := arrange.NewForm(sections...)
	chords := harmony.MustKey("D minor").MustProgression("i VI")

	mix := mixer.New(p)
	hall := fdnreverb.DefaultConfig()
	hall.Decay, hall.Mix = 2.2, 1
	mix.Bus("hall", mixer.StripConfig{}, fdnreverb.New(hall).AddTo(p))

	for _, e := range chosen {
		synth := e.make(p)
		name := e.synth + " " + e.preset
		up := float64(12 * e.octave)
		form.Part(p, synth, name, score.Config{Velocity: 0.75}, arrange.Only(name, func(s S) []N {
			return play(e.style, up, chords)
		}))
		send := 0.25
		if e.style == beat || e.style == bass || e.style == held {
			send = 0.08
		}
		mix.Channel(name, synth, mixer.StripConfig{Volume: -8, Sends: map[string]float64{"hall": send}})
	}
	arrangement := arrange.NewArrangement(form).PrintSections(true)
	arrangement.AddTo(p)
	p.Out(mix.Master())

	if *renderTo != "" {
		stats, err := render.ToFile(p, *renderTo, render.Options{Seconds: arrangement.Duration(), Tail: 4})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(stats)
		return
	}
	fmt.Println("playing, Ctrl-C to stop")
	_ = p.Listen()
}

// play returns two bars of a style (the last half bar left for the sound to ring out)
func play(st style, up float64, chords []harmony.Chord) []N {
	c1, c2 := chords[0], chords[1]
	switch st {
	case pad:
		return []N{score.Chord(addAll(c1.Close(57), up), 3.5), score.Chord(addAll(c2.Close(57), up), 3), score.Rest(1.5)}
	case drone:
		return []N{score.Chord([]float64{38 + up}, 6.5), score.Rest(1.5)}
	case lead:
		return score.Transpose(score.MustParse(`D5:1 F5:0.5 A5:0.5 C6:1.5 A5:0.5 | Bb5:1 A5:0.5 G5:0.5 F5:1 r:1`), up)
	case slow:
		return score.Transpose(score.MustParse(`D5:2 F5:2 | A5:3 r:1`), up)
	case arp:
		var ns []N
		for _, c := range []harmony.Chord{c1, c2} {
			v := addAll(c.Close(62), up)
			ns = append(ns, score.Notes(0.25, v[0], v[1], v[2], v[0]+12, v[2], v[1], v[0], v[1]+12)...)
			ns = append(ns, score.Notes(0.25, v[0], v[1], v[2], v[0]+12, v[2]+12, v[1]+12, v[0]+12, v[2])...)
		}
		return ns
	case bass:
		r1, r2 := c1.BassPitch(38)+up, c2.BassPitch(38)+up
		return score.Shape(score.Notes(0.5, r1, r1, r1+12, r1, 0, r1+7, r1+10, r1+12, r2, r2, r2+12, r2, 0, r2+7, r2+5, 0),
			score.Hold(0.8))
	case held:
		return []N{score.Chord([]float64{c1.BassPitch(38) + up}, 3.5), score.Rest(0.5), score.Chord([]float64{c2.BassPitch(38) + up}, 3), score.Rest(1)}
	case hits:
		return score.Transpose(score.MustParse(`D5:1.5 A5:1.5 F5:1 | C6:2.5 r:1.5`), up)
	case beat:
		bar := []N{}
		hit := func(sound string, beats float64) N {
			return N{Pitches: []float64{60}, Beats: beats, Params: map[string]any{"sound": sound}}
		}
		// kick, hat, snare and the others in sixteenths, over two bars
		pattern := []string{"kick", "hat", "hat", "kick", "snare", "hat", "kick", "hat",
			"hat", "kick", "hat", "hat", "snare", "hat", "clap", "open",
			"kick", "hat", "tomHigh", "kick", "snare", "hat", "kick", "cowbell",
			"tomLow", "kick", "tomMid", "hat", "snare", "rim", "clap", "cymbal"}
		for _, sound := range pattern {
			bar = append(bar, hit(sound, 0.25))
		}
		return bar
	}
	return nil
}

func addAll(v []float64, up float64) []float64 {
	out := make([]float64, len(v))
	for i := range v {
		out[i] = v[i] + up
	}
	return out
}
