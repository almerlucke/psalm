package main

import (
	"fmt"

	"github.com/almerlucke/genny/bucket"
	"github.com/almerlucke/genny/sequence"
	"github.com/almerlucke/genny/template"
	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/modules/controls/templ"
	"github.com/almerlucke/psalm/modules/controls/timer"
	"github.com/almerlucke/psalm/modules/controls/timer/steps"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/effects/gain"
	"github.com/almerlucke/psalm/modules/effects/granulator"
	"github.com/almerlucke/psalm/synths/drums"
	"github.com/almerlucke/psalm/synths/dx"
	"github.com/almerlucke/sndfile"
)

// Music through the granulator, which changes scene every 4 bars: a cloud, an octave shimmer, a
// freeze, a reverse smear, a stutter and a feedback cascade. The scenes loop

const sceneBars = 4

// seq plays the values in order and loops
func seq[T any](values ...T) *sequence.Sequence[T] {
	return sequence.NewLoop(values...)
}

// random picks one of the values at random every time
func random[T any](values ...T) *bucket.Bucket[T] {
	return bucket.NewLoop(values...).SetMode(bucket.Random)
}

// scene is a set of granulator settings, freeze lists per bar of the scene whether the buffer is frozen
type scene struct {
	name     string
	settings map[string]float64
	freeze   [sceneBars]float64
}

var scenes = []scene{
	{"Cloud: a soft granular halo around the music", map[string]float64{
		"mix": 0.37, "delay": 150, "spray": 80, "density": 25, "grainSize": 120, "pitch": 0,
		"pitchSpread": 0.1, "octave": 0, "reverse": 0, "shape": 0, "feedback": 0,
	}, [sceneBars]float64{}},
	{"Shimmer: grains an octave up", map[string]float64{
		"mix": 0.5, "delay": 200, "spray": 120, "density": 35, "grainSize": 180, "pitch": 12,
		"pitchSpread": 0.1, "octave": 0, "reverse": 0, "shape": 0, "feedback": 0.2,
	}, [sceneBars]float64{}},
	{"Freeze: the first bar is captured and held while the music moves on", map[string]float64{
		"mix": 0.63, "delay": 300, "spray": 300, "density": 40, "grainSize": 250, "pitch": 0,
		"pitchSpread": 0.15, "octave": 0.2, "reverse": 0, "shape": 0, "feedback": 0,
	}, [sceneBars]float64{0, 1, 1, 1}},
	{"Reverse smear: long backward grains with a wide spray", map[string]float64{
		"mix": 0.63, "delay": 600, "spray": 400, "density": 20, "grainSize": 300, "pitch": 0,
		"pitchSpread": 0.2, "octave": 0, "reverse": 0.8, "shape": 0, "feedback": 0,
	}, [sceneBars]float64{}},
	{"Stutter: short percussive grains just behind the input chop up the drums", map[string]float64{
		"mix": 0.74, "delay": 40, "spray": 10, "density": 16, "grainSize": 50, "pitch": 0,
		"pitchSpread": 0, "octave": 0, "reverse": 0, "shape": 1, "feedback": 0,
	}, [sceneBars]float64{}},
	{"Feedback cascade: grains a fifth up fed back, stacking fifths", map[string]float64{
		"mix": 0.5, "delay": 250, "spray": 100, "density": 30, "grainSize": 200, "pitch": 7,
		"pitchSpread": 0.05, "octave": 0, "reverse": 0, "shape": 0, "feedback": 0.55,
	}, [sceneBars]float64{}},
}

func main() {
	psalm.PushConfig(psalm.Config{BPM: 100, PPQ: 240, SampleRate: 44100, BufferSize: 1024})

	p := psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	defer p.Close()

	ppq := float64(psalm.PPQ())
	beatSec := 60.0 / 100.0

	// --- the music: electric piano chords, a mallet arpeggio and light drums ---
	chords := [][]float64{{57, 60, 64}, {53, 57, 60}, {48, 55, 64}, {55, 59, 62}} // Am F C G

	epiano := dx.NewPoly(dx.EPianoPreset(), 12, psalm.Stereo).AddTo(p)
	epiano.ValueIn(templ.NewDurationTrigger(seq(1.2*beatSec, 2.2*beatSec), 0.132, template.Template{
		"pitch":    seq(chords[0], chords[0], chords[1], chords[1], chords[2], chords[2], chords[3], chords[3]),
		"velocity": random(0.6, 0.8, 1.0),
	}).ValueIn(timer.WithSteps(steps.QuickSteps(1.5, 2.5)).AddTo(p)))

	var arp []float64
	for _, c := range chords {
		up := []float64{c[0] + 12, c[1] + 12, c[2] + 12, c[1] + 12}
		arp = append(arp, append(up, up...)...)
	}
	mallet := dx.NewPoly(dx.MalletPreset(), 8, psalm.Stereo).AddTo(p)
	mallet.ValueIn(templ.NewDurationTrigger(0.3, 0.157, template.Template{
		"pitch":    seq(arp...),
		"velocity": seq(1.0, 0.6, 0.8, 0.6),
	}).ValueIn(timer.WithTicks(ppq / 2).AddTo(p)))

	bank := sndfile.SoundBank[float32]{
		"kick":  sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/kick/Cymatics - Humble Sit Down Kick - D.wav", 4),
		"snare": sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/snare/Cymatics - Humble Adequate Snare - E.wav", 4),
		"hat":   sndfile.MustMipMapSoundFile[float32]("assets/sounds/drums/hihat/Cymatics - Humble Closed Hihat 4.wav", 4),
	}
	kit := drums.NewDrums(bank, 16, 2).AddTo(p)
	for sound, pattern := range map[string][]any{
		"kick":  {1, 0, 0, 0, 0, 0, 1, 0, 1, 0, 0, 0, 0, 0, 0, 0},
		"snare": {0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0},
		"hat":   {1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 1},
	} {
		amp := map[string]float64{"kick": 0.5, "snare": 0.35, "hat": 0.15}[sound]
		kit.ValueIn(templ.NewTrigger(amp, template.Template{"sound": sound, "speed": 1.0}).
			ValueIn(timer.WithSteps(steps.Multiply(steps.QuickSteps(pattern...), 0.25)).AddTo(p)))
	}

	// --- the granulator, its scenes and the freeze ---
	gran := granulator.New(granulator.DefaultConfig())
	gran.AddTo(p)
	gran.In(epiano, 0, 0, mallet, 0, 0, kit, 0, 0)

	// every scene sends its settings at its first bar
	settings := template.Template{}
	for key := range scenes[0].settings {
		var values []float64
		for _, s := range scenes {
			values = append(values, s.settings[key])
		}
		settings[key] = seq(values...)
	}
	gran.ValueIn(templ.New(settings).ValueIn(timer.WithTicks(sceneBars * 4 * ppq).AddTo(p)))

	// freeze is sent every bar
	var freeze []float64
	for _, s := range scenes {
		freeze = append(freeze, s.freeze[:]...)
	}
	gran.ValueIn(templ.New(template.Template{"freeze": seq(freeze...)}).ValueIn(timer.WithTicks(4 * ppq).AddTo(p)))

	sceneSec := sceneBars * 4 * beatSec
	for i, s := range scenes {
		fmt.Printf("%6.1fs  %s\n", float64(i)*sceneSec, s.name)
	}
	fmt.Printf("%6.1fs  (loops)\n", float64(len(scenes))*sceneSec)

	rcfg := fdnreverb.DefaultConfig()
	rcfg.Decay = 2.5
	rcfg.Mix = 0.16

	p.Out(gain.New(-4.5).AddTo(p).In(fdnreverb.New(rcfg).AddTo(p).In(gran)))

	_ = p.Listen()
}
