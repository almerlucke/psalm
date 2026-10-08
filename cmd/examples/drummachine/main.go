// Drum machine: a step sequencer with a kit of samples, played and edited on a web page.
//
// Seven rows (kick, snare, clap, closed and open hihat, percussion, an 808 bass), 16 steps, four
// patterns. On the page: tap a step to switch it, drag to paint, right click (or long press, or
// shift click) a step to edit its velocity, chance, roll, nudge, pitch and condition; tap a row's
// name for its tools (clear, shift, Euclid, random, copy the pattern). The pattern tabs switch
// patterns when the pattern comes round. Swing, humanizing and the fill switch act on the whole
// beat, every row has its level, length (rows loop apart), mute and solo. The open hihat is choked
// by the closed one.
//
// The starting patterns show what a step can do: A is a beat with ghost notes, a hihat roll that
// happens now and then, a clap on every second time round and a snare roll only with the fill; B
// rolls up into a fill; C and D are empty, to write your own.
//
//	go run ./cmd/examples/drummachine                    then open http://localhost:8080
//	go run ./cmd/examples/drummachine -addr :8080        from a phone or tablet on the network
//	go run ./cmd/examples/drummachine -midi              the knobs of a MIDI controller too (MIDI learn)
//	go run ./cmd/examples/drummachine -render beat.wav   render 8 bars of pattern A to a file
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/midi"
	"github.com/almerlucke/psalm/modules/controls/keyboard"
	"github.com/almerlucke/psalm/modules/controls/panel"
	"github.com/almerlucke/psalm/modules/controls/stepseq"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/synths/drums"
	"github.com/almerlucke/psalm/tools/render"
	"github.com/almerlucke/sndfile"
)

const kitDir = "assets/sounds/drums/"

// the kit: a row each, by its sample
var kit = []struct{ name, file string }{
	{"kick", "kick/Cymatics - Humble Friday Kick - E.wav"},
	{"snare", "snare/Cymatics - Humble King Snare 1 - A.wav"},
	{"clap", "clap/Cymatics - Humble Stars Clap.wav"},
	{"hat", "hihat/Cymatics - Humble Closed Hihat 4.wav"},
	{"open", "hihat/Cymatics - Humble Open Hihat 1.wav"},
	{"perc", "percussion/Cymatics - Odyssey Percussion 3.wav"},
	{"808", "808/Cymatics - Humble 808 1 - D.wav"},
}

func main() {
	addr := flag.String("addr", "localhost:8087", "the address of the page")
	useMIDI := flag.Bool("midi", false, "move the controls from a MIDI controller too (MIDI learn on the page)")
	renderTo := flag.String("render", "", "render 8 bars to this WAV file instead of playing")
	seed := flag.Uint64("seed", 0, "seed for the chances and the humanizing: the same seed plays the same")
	flag.Parse()
	if *seed != 0 {
		psalm.Seed(*seed)
	}

	psalm.PushConfig(psalm.Config{BPM: 96, PPQ: 240, SampleRate: 44100, BufferSize: 256})
	var p *psalm.Psalm
	if *renderTo != "" {
		p = psalm.New(nil)
	} else {
		p = psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	}
	defer p.Close()

	// --- the kit: a drum synth playing the samples by name ---

	bank := sndfile.SoundBank[float32]{}
	var rows []string
	for _, k := range kit {
		bank[k.name] = sndfile.MustMipMapSoundFile[float32](kitDir+k.file, 4)
		rows = append(rows, k.name)
	}
	kitSynth := drums.NewDrums(bank, 24, 2).AddTo(p)

	// --- the panel, the grid and the sequencer playing it ---

	ctl := panel.New(p, "Drum machine")
	ctl.Section("Beat")
	grid := stepseq.NewGrid(ctl, "Beat", rows, 16, 4)
	seq := stepseq.New(p, grid, stepseq.Config{StepsPerBeat: 4, Swing: 0.56, Humanize: 4, HumanizeVelocity: 0.06})
	for _, r := range rows {
		var opts []stepseq.RowOption
		if r == "hat" || r == "open" {
			opts = append(opts, stepseq.Choke(1)) // the closed hihat cuts the open one
		}
		seq.Row(r, stepseq.Sound(kitSynth, r), opts...)
	}
	writePatterns(grid)
	seq.Controls(ctl)

	// --- the mix: the kit through a little hall ---

	mix := mixer.New(p)
	hall := fdnreverb.DefaultConfig()
	hall.Decay = 1.6
	hall.Mix = 1
	mix.Bus("hall", mixer.StripConfig{}, fdnreverb.New(hall).AddTo(p))
	kitStrip := mix.Channel("kit", kitSynth, mixer.StripConfig{Volume: -12, Sends: map[string]float64{"hall": 0.12}})
	p.Out(mix.Master())

	ctl.Section("Mix")
	ctl.Fader("Tempo", 60, 180, panel.Linear, panel.Func(func(v float64) { p.Env().SetBPM(v) }), panel.Default(96), panel.Step(1), panel.Unit("BPM"))
	ctl.Fader("Volume", -40, 6, panel.Linear, panel.Message(kitStrip, "volume"), panel.Default(-12), panel.Unit("dB"), panel.Smooth(0.03))
	ctl.Knob("Hall", 0, 0.6, panel.Linear, panel.Message(kitStrip, "send.hall"), panel.Default(0.12))
	ctl.Meter("Master", mix.Master())

	if *renderTo != "" {
		stats, err := render.ToFile(p, *renderTo, render.Options{Seconds: 8 * 4 * 60 / 96.0, Tail: 2})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(stats)
		fmt.Print(mix.Meters())
		return
	}

	if *useMIDI {
		d, err := midi.OpenDevice("")
		if err != nil {
			log.Fatalf("%v (go run ./cmd/examples/keyboard -list shows the devices)", err)
		}
		defer d.Close()
		kb := keyboard.New(d, keyboard.Config{Controls: map[int]string{}})
		kb.AddTo(p)
		ctl.MIDI(kb)
		if err := ctl.KeepBindings("drummachine-midi.json"); err != nil {
			log.Fatal(err)
		}
	}
	go func() {
		if err := ctl.Serve(*addr); err != nil {
			log.Fatal(err)
		}
	}()
	defer ctl.Close()
	fmt.Printf("open http://%s (Ctrl-C to stop)\n", *addr)
	_ = p.Listen()
}

// steps writes a row of a pattern from text, a character per step: X an accent, x a hit, o a
// ghost note, . nothing
func steps(g *panel.Grid, pattern int, row, text string) {
	velocities := map[rune]float64{'X': 1, 'x': 0.75, 'o': 0.4}
	r := g.Row(row)
	i := 0
	for _, c := range text {
		if c == ' ' || c == '|' {
			continue
		}
		g.Set(pattern, r, i, stepseq.Velocity, velocities[c])
		i++
	}
}

// writePatterns writes the starting patterns: A a beat that shows what a step can do, B a fill
func writePatterns(g *panel.Grid) {
	set := func(pattern int, row string, step, field int, v float64) { g.Set(pattern, g.Row(row), step, field, v) }
	condition := func(name string) float64 {
		for i, c := range stepseq.Conditions {
			if c == name {
				return float64(i)
			}
		}
		return 0
	}

	// A: a beat with ghost notes
	steps(g, 0, "kick", "X... ..x. X... .o..")
	steps(g, 0, "snare", ".... X..o .... X..x")
	steps(g, 0, "clap", ".... x... .... x...")
	steps(g, 0, "hat", "x.x. xox. x.x. xoxx")
	steps(g, 0, "open", ".... .... .... ..x.")
	steps(g, 0, "perc", "..o. ...o ..o. o...")
	steps(g, 0, "808", "X... ..x. x... ....")
	set(0, "clap", 4, stepseq.Condition, condition("1:2")) // the clap on beat 2 every second time round
	set(0, "clap", 12, stepseq.Condition, condition("2:2"))
	set(0, "hat", 14, stepseq.Roll, 3) // a hihat roll now and then
	set(0, "hat", 14, stepseq.RollChance, 0.5)
	set(0, "perc", 2, stepseq.Chance, 0.6) // the percussion comes and goes
	set(0, "perc", 7, stepseq.Chance, 0.5)
	set(0, "perc", 10, stepseq.Chance, 0.6)
	set(0, "perc", 12, stepseq.Pitch, 5)
	set(0, "808", 6, stepseq.Pitch, 3) // a bass line on the 808: D, F, A
	set(0, "808", 8, stepseq.Pitch, 7)
	set(0, "snare", 15, stepseq.Roll, 4) // a snare roll into the next bar, only with the fill
	set(0, "snare", 15, stepseq.RollRamp, 0.7)
	set(0, "snare", 15, stepseq.Condition, condition("fill"))
	set(0, "kick", 13, stepseq.Nudge, 0.1) // the ghost kick a little late

	// B: the beat rolling up into a fill
	steps(g, 1, "kick", "X... ..x. X.x. ....")
	steps(g, 1, "snare", "..o. X..o ..o. XxXX")
	steps(g, 1, "hat", "x.x. x.x. x.x. ....")
	steps(g, 1, "open", "...x ...x ...x ....")
	steps(g, 1, "808", "X... ..x. x.x. ....")
	for _, s := range []int{12, 13, 14, 15} {
		set(1, "snare", s, stepseq.Roll, float64(s-10)) // 2, 3, 4, 5 hits: faster and faster
		set(1, "snare", s, stepseq.RollRamp, 0.5)
	}
	set(1, "808", 10, stepseq.Pitch, 5)
}
