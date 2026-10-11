// FX: a short groove where every effect has a job, to hear them in a mix.
//
//	drums      the kit through a transient shaper (snappier) and a glue compressor; the hihats
//	           through a bit crusher
//	bass       tube saturation, then a compressor ducked by the kick on its sidechain (the pump)
//	organ      a rotary speaker: slow in the intro, fast halfway the groove, slow again for the break
//	keys       an electric piano through a phaser and a tremolo in 8ths
//	clav       an auto-wah: every note quacks open
//	pad        an EQ (the low end out, some air) and a stereo widener; in the break a second pad
//	           is gated by the hihats on a sidechain
//	bells      a frequency shifter: a slow, wide beating
//	lead       a tape echo, wavering, its repeats darker and warmer
//
//	go run ./cmd/examples/fx                    play it
//	go run ./cmd/examples/fx -render fx.wav     render it
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/effects/compressor"
	"github.com/almerlucke/psalm/modules/effects/crusher"
	"github.com/almerlucke/psalm/modules/effects/eq"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/effects/follower"
	"github.com/almerlucke/psalm/modules/effects/gate"
	"github.com/almerlucke/psalm/modules/effects/phaser"
	"github.com/almerlucke/psalm/modules/effects/ringmod"
	"github.com/almerlucke/psalm/modules/effects/rotary"
	"github.com/almerlucke/psalm/modules/effects/saturate"
	"github.com/almerlucke/psalm/modules/effects/tape"
	"github.com/almerlucke/psalm/modules/effects/transient"
	"github.com/almerlucke/psalm/modules/effects/tremolo"
	"github.com/almerlucke/psalm/modules/effects/widener"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/drums"
	"github.com/almerlucke/psalm/synths/fm"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
	"github.com/almerlucke/psalm/tools/harmony"
	"github.com/almerlucke/psalm/tools/render"
	"github.com/almerlucke/sndfile"
)

type (
	S = arrange.Section
	N = score.Note
)

const kit = "assets/sounds/drums/"

func main() {
	renderTo := flag.String("render", "", "render to this WAV file instead of playing")
	seed := flag.Uint64("seed", 0, "seed for everything random")
	flag.Parse()
	if *seed != 0 {
		psalm.Seed(*seed)
	}
	psalm.PushConfig(psalm.Config{BPM: 98, PPQ: 240, SampleRate: 44100, BufferSize: 512})
	var p *psalm.Psalm
	if *renderTo != "" {
		p = psalm.New(nil)
	} else {
		p = psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	}
	defer p.Close()

	form := arrange.NewForm(S{Name: "Intro", Bars: 4}, S{Name: "Groove", Bars: 8}, S{Name: "Break", Bars: 4})
	chords := harmony.MustKey("D minor").MustProgression("i VI III VII")
	chordOf := func(s S, bar int) harmony.Chord { return chords[(s.StartBar+bar)%4] }

	// --- the drums: three kits, so the kick and the hihats can go their own ways ---

	bank := func(names ...string) sndfile.SoundBank[float32] {
		files := map[string]string{
			"kick":  "kick/Cymatics - Humble Friday Kick - E.wav",
			"snare": "snare/Cymatics - Humble King Snare 1 - A.wav",
			"hat":   "hihat/Cymatics - Humble Closed Hihat 4.wav",
			"open":  "hihat/Cymatics - Humble Open Hihat 1.wav",
		}
		b := sndfile.SoundBank[float32]{}
		for _, n := range names {
			b[n] = sndfile.MustMipMapSoundFile[float32](kit+files[n], 4)
		}
		return b
	}
	kicks := drums.NewDrums(bank("kick"), 4, 2).AddTo(p)
	snares := drums.NewDrums(bank("snare"), 4, 2).AddTo(p)
	hats := drums.NewDrums(bank("hat", "open"), 8, 2).AddTo(p)

	// beat writes a bar from text: X an accent, x a hit, o a ghost, . nothing (16ths)
	beat := func(sound, text string) []N {
		amp := map[rune]float64{'X': 1, 'x': 0.75, 'o': 0.35}
		var ns []N
		for _, c := range text {
			if c == ' ' {
				continue
			}
			if a, ok := amp[c]; ok {
				ns = append(ns, N{Pitches: []float64{60}, Beats: 0.25, Amp: a, Params: map[string]any{"sound": sound, "speed": 1.0}})
			} else {
				ns = append(ns, score.Rest(0.25))
			}
		}
		return ns
	}
	drumPart := func(synth psalm.Modular, name string, pattern func(s S, bar int) []N) {
		form.Part(p, synth, name, score.Config{Feel: score.Feel{Timing: 3, Velocity: 0.05}}, func(s S) []N {
			return arrange.PerBar(s, func(bar int) []N { return pattern(s, bar) })
		})
	}
	drumPart(kicks, "kick", func(s S, bar int) []N {
		if s.Name == "Intro" && bar < 2 {
			return nil
		}
		return beat("kick", "X... ..x. X... .x..")
	})
	drumPart(snares, "snare", func(s S, bar int) []N {
		if s.Name == "Intro" {
			return nil
		}
		if s.Name == "Groove" && bar == 7 {
			return beat("snare", ".... X..o .... XxXX")
		}
		return beat("snare", ".... X..o .... X...")
	})
	drumPart(hats, "hats", func(s S, bar int) []N {
		if s.Name == "Break" {
			return beat("hat", "x.x. x.x. x.x. x.x.")
		}
		return append(beat("hat", "x.xo x.xo x.xo x."), beat("open", "x.")...)
	})

	// --- the instruments ---

	bass := subtractive.NewPoly(subtractive.BassPreset(), 2).AddTo(p)
	organ := fm.NewPoly(fm.OrganPreset(), 8).AddTo(p)
	keys := fm.NewPoly(fm.EPianoPreset(), 8).AddTo(p)
	clav := fm.NewPoly(fm.ClavinetPreset(), 6).AddTo(p)
	pad := wavetable.NewPoly(wavetable.PadPreset(gen.Brightness(32)), 8).AddTo(p)
	gated := wavetable.NewPoly(wavetable.PadPreset(gen.Brightness(40)), 8).AddTo(p)
	bells := fm.NewPoly(fm.BellPreset(), 6).AddTo(p)
	lead := fm.NewPoly(fm.FlutePreset(), 2).AddTo(p)
	played := score.Feel{Timing: 6, Velocity: 0.08}

	form.Part(p, bass, "bass", score.Config{Velocity: 0.8, Feel: played}, func(s S) []N {
		if s.Name == "Intro" {
			return nil
		}
		return arrange.PerBar(s, func(bar int) []N {
			r := chordOf(s, bar).BassPitch(33)
			return score.Shape(score.Notes(0.5, r, 0, r+12, r, 0, r+7, r+10, r), score.Velocities(0.9, 0, 0.6, 0.8, 0, 0.6, 0.5, 0.7))
		})
	})
	form.Part(p, organ, "organ", score.Config{Amp: 0.6}, func(s S) []N {
		return arrange.PerBar(s, func(bar int) []N { return []N{score.Chord(chordOf(s, bar).Close(57), 4)} })
	})
	form.Part(p, keys, "keys", score.Config{Velocity: 0.55, Feel: played}, arrange.Only("Groove", func(s S) []N {
		return arrange.PerBar(s, func(bar int) []N {
			c := score.Chord(chordOf(s, bar).Close(62), 1)
			return score.Shape([]N{score.Rest(0.5), c, score.Rest(0.5), c, score.Rest(1)}, score.Hold(0.6))
		})
	}))
	form.Part(p, clav, "clav", score.Config{Velocity: 0.7, Feel: played}, arrange.Only("Groove", func(s S) []N {
		return arrange.PerBar(s, func(bar int) []N {
			v := chordOf(s, bar).Close(55)
			c := score.Chord(v[:2], 0.25)
			return score.Shape([]N{c, score.Rest(0.25), c, c, score.Rest(0.5), c, score.Rest(0.25), c, score.Rest(1.75)}, score.Hold(0.15))
		})
	}))
	form.Part(p, pad, "pad", score.Config{Amp: 0.5}, func(s S) []N {
		if s.Name == "Break" {
			return nil
		}
		return arrange.PerBar(s, func(bar int) []N { return []N{score.Chord(chordOf(s, bar).Close(64), 4)} })
	})
	form.Part(p, gated, "gated pad", score.Config{Amp: 0.6}, arrange.Only("Break", func(s S) []N {
		return arrange.PerBar(s, func(bar int) []N { return []N{score.Chord(chordOf(s, bar).Close(60), 4)} })
	}))
	form.Part(p, bells, "bells", score.Config{Velocity: 0.4}, arrange.Only("Break", func(s S) []N {
		return score.MustParse(`A5:2 F5:2 | C6:2 A5:2 | G5:2 E5:2 | A5:4`)
	}))
	form.Part(p, lead, "lead", score.Config{Amp: 0.7, Feel: score.Feel{Timing: 10, Push: 6}}, arrange.Only("Groove", func(s S) []N {
		return score.MustParse(`r:4 | r:2 D6:0.5 F6:0.5 A6:1 | G6:3 r:1 | r:4 |
			r:2 A6:0.5 C7:0.5 D7:1 | C7:1.5 A6:0.5 G6:2 | r:4 | r:4`)
	}))

	// --- the effects ---

	drumsSnap := transient.New(transient.Config{Attack: 6, Sustain: -2, Mix: 1}).AddTo(p)
	drumsGlue := compressor.New(compressor.Config{Threshold: -16, Ratio: 2.5, Knee: 6, Attack: 15, Release: 120, Makeup: 2, Mix: 0.7}).AddTo(p)
	hatsCrush := crusher.New(crusher.Config{Bits: 10, Rate: 16000, Mix: 0.6}).AddTo(p).In(hats)
	bassWarm := saturate.New(saturate.Config{Shape: saturate.Tube, Drive: 10, Tone: 5000, Output: -5, Mix: 1}).AddTo(p).In(bass)
	bassDuck := compressor.New(compressor.Ducker(9, 140)).AddTo(p)
	bassDuck.In(bassWarm, 0, 0, kicks, 0, 1) // the bass on 0, the kick on the sidechain
	leslie := rotary.New(rotary.DefaultConfig())
	leslie.AddTo(p).In(organ)
	keysPhase := phaser.New(phaser.Config{Stages: 4, Sync: 4, Min: 300, Max: 2400, Feedback: 0.5, Stereo: 0.25, Mix: 0.5}).AddTo(p).In(keys)
	keysTrem := tremolo.New(tremolo.Config{Sync: 0.5, Depth: 0.35, Mode: tremolo.Pan}).AddTo(p).In(keysPhase)
	clavWah := follower.NewWah(follower.WahConfig{Kind: follower.BandPass, Min: 400, Max: 2600, Q: 5, Gain: 14, Attack: 3, Release: 90, Mix: 0.9}).AddTo(p).In(clav)
	padCfg := eq.DefaultConfig()
	padCfg.LowCut = 220
	padCfg.Mid1 = eq.Band{Freq: 450, Gain: -3, Q: 1}
	padCfg.High = eq.Band{Freq: 9000, Gain: 3}
	padEQ := eq.New(padCfg).AddTo(p).In(pad)
	padWide := widener.New(widener.Config{Width: 1.6, MonoBelow: 150}).AddTo(p).In(padEQ)
	padGate := gate.New(gate.Config{Threshold: -38, Range: 30, Attack: 1, Hold: 40, Release: 70}).AddTo(p)
	padGate.In(gated, 0, 0, hats, 0, 1) // the hihats open the gate
	bellShift := ringmod.New(ringmod.Config{Mode: ringmod.Shift, Freq: 2, Stereo: -4, Mix: 0.5}).AddTo(p).In(bells)
	leadEcho := tape.New(tape.Config{Time: 0.75, Feedback: 0.5, Wow: 0.35, Flutter: 0.2, HighCut: 3500, LowCut: 200, Mix: 0.35}).AddTo(p).In(lead)

	// the rotary speeds up halfway the groove and slows down for the break
	arrangement := arrange.NewArrangement(form).
		At("Groove", 4, func() { leslie.SetFast(true) }).
		At("Break", 0, func() { leslie.SetFast(false) }).
		PrintSections(true)
	arrangement.AddTo(p)

	// --- the mix ---

	mix := mixer.New(p)
	hall := fdnreverb.DefaultConfig()
	hall.Decay, hall.Mix = 2.2, 1
	mix.Bus("hall", mixer.StripConfig{}, fdnreverb.New(hall).AddTo(p))
	mix.Group("drums", mixer.StripConfig{Volume: -9, Sends: map[string]float64{"hall": 0.08}}, drumsSnap, drumsGlue)
	mix.Channel("kick", kicks, mixer.StripConfig{Group: "drums", Volume: -3})
	mix.Channel("snare", snares, mixer.StripConfig{Group: "drums", Volume: -3})
	mix.Channel("hats", hatsCrush, mixer.StripConfig{Group: "drums", Volume: -10, Pan: 0.2})
	mix.Channel("bass", bassDuck, mixer.StripConfig{Volume: -9})
	mix.Channel("organ", leslie, mixer.StripConfig{Volume: -16, Sends: map[string]float64{"hall": 0.2}})
	mix.Channel("keys", keysTrem, mixer.StripConfig{Volume: -15, Sends: map[string]float64{"hall": 0.2}})
	mix.Channel("clav", clavWah, mixer.StripConfig{Volume: -22, Pan: -0.3})
	mix.Channel("pad", padWide, mixer.StripConfig{Volume: -18, Sends: map[string]float64{"hall": 0.3}})
	mix.Channel("gated pad", padGate, mixer.StripConfig{Volume: -16, Sends: map[string]float64{"hall": 0.2}})
	mix.Channel("bells", bellShift, mixer.StripConfig{Volume: -16, Pan: 0.3, Sends: map[string]float64{"hall": 0.4}})
	mix.Channel("lead", leadEcho, mixer.StripConfig{Volume: -7, Pan: 0.1, Sends: map[string]float64{"hall": 0.2}})
	p.Out(mix.Master())

	if *renderTo != "" {
		stats, err := render.ToFile(p, *renderTo, render.Options{Seconds: arrangement.Duration(), Tail: 4})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(stats)
		fmt.Print(mix.Meters())
		return
	}
	fmt.Println("playing, Ctrl-C to stop")
	_ = p.Listen()
}
