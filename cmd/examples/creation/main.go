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
//	go run ./cmd/examples/creation                                     play it
//	go run ./cmd/examples/creation -tuning equal                       in equal temperament instead
//	go run ./cmd/examples/creation -render creation.wav -stems -seed 7 render it, a stem per strip
//	go run ./cmd/examples/creation -profile                            the time every module takes

import (
	"flag"
	"fmt"
	"math"
	"slices"

	"github.com/almerlucke/genny/float/envelopes/adsr"
	"github.com/almerlucke/psalm"
	paengine "github.com/almerlucke/psalm/audio/portaudio"
	"github.com/almerlucke/psalm/blocks/filters/ladder"
	"github.com/almerlucke/psalm/blocks/filters/svf"
	"github.com/almerlucke/psalm/modules/controls/arrange"
	"github.com/almerlucke/psalm/modules/controls/score"
	"github.com/almerlucke/psalm/modules/effects/convolver"
	"github.com/almerlucke/psalm/modules/effects/fdnreverb"
	"github.com/almerlucke/psalm/modules/effects/pingpong"
	"github.com/almerlucke/psalm/modules/effects/pitchshift"
	"github.com/almerlucke/psalm/modules/effects/shimmer"
	"github.com/almerlucke/psalm/modules/effects/vocoder"
	"github.com/almerlucke/psalm/modules/filters"
	"github.com/almerlucke/psalm/modules/mixer"
	"github.com/almerlucke/psalm/modules/player"
	"github.com/almerlucke/psalm/synths/dx"
	"github.com/almerlucke/psalm/synths/fm"
	"github.com/almerlucke/psalm/synths/grains"
	"github.com/almerlucke/psalm/synths/piano"
	"github.com/almerlucke/psalm/synths/pulsar"
	"github.com/almerlucke/psalm/synths/subtractive"
	"github.com/almerlucke/psalm/synths/wavetable"
	"github.com/almerlucke/psalm/synths/wavetable/gen"
	"github.com/almerlucke/psalm/tools/harmony"
	"github.com/almerlucke/psalm/tools/render"
	"github.com/almerlucke/sndfile"
)

const (
	bpm     = 72
	restBPM = 63 // the seventh day
	evening = 6  // bars of every day before the morning
)

// the days
const (
	day1 = "Day 1: light"
	day2 = "Day 2: firmament"
	day3 = "Day 3: land"
	day4 = "Day 4: lights"
	day5 = "Day 5: creatures"
	day6 = "Day 6: mankind"
	day7 = "Day 7: rest"
)

// --- the form: seven days, three beats to the bar ---

var form = func() *arrange.Form {
	verse := func(text string) map[string]any { return map[string]any{"verse": text} }
	f := arrange.NewForm(
		arrange.Section{Name: day1, Bars: 12, Dynamics: 1.3, Params: verse("And God said, Let there be light: and there was light. (Genesis 1:3)")},
		arrange.Section{Name: day2, Bars: 12, Params: verse("And God said, Let there be a firmament in the midst of the waters, and let it divide the waters from the waters. (Genesis 1:6)")},
		arrange.Section{Name: day3, Bars: 12, Params: verse("And God said, Let the earth bring forth grass, the herb yielding seed, and the fruit tree yielding fruit after his kind. (Genesis 1:11)")},
		arrange.Section{Name: day4, Bars: 12, Params: verse("And God said, Let there be lights in the firmament of the heaven to divide the day from the night; and let them be for signs, and for seasons, and for days, and years. (Genesis 1:14)")},
		arrange.Section{Name: day5, Bars: 12, Params: verse("And God said, Let the waters bring forth abundantly the moving creature that hath life, and fowl that may fly above the earth in the open firmament of heaven. (Genesis 1:20)")},
		arrange.Section{Name: day6, Bars: 12, Dynamics: 1.1, Params: verse("So God created man in his own image, in the image of God created he him; male and female created he them. And God saw every thing that he had made, and, behold, it was very good. (Genesis 1:27, 31)")},
		arrange.Section{Name: day7, Bars: 21, Params: verse("And God blessed the seventh day, and sanctified it: because that in it he had rested from all his work which God created and made. (Genesis 2:3)\n" +
			"For by him were all things created, that are in heaven, and that are in earth, visible and invisible: all things were created by him, and for him. " +
			"And, having made peace through the blood of his cross, by him to reconcile all things unto himself. (Colossians 1:16, 20)")},
	)
	f.BeatsPerBar = 3 // the Trinity
	return f
}()

// --- harmony: E flat major, a progression for every day ---

var key = harmony.MustKey("Eb major")

// the chords of every day, one per bar: the evening, then the morning
var days = map[string]string{
	day1: "I5 I5 I5 I5 I5 I5  I IV I V IV I",                    // without form and void (no third), then light
	day2: "vi vi IV IV V V  I V vi iii IV I",                    // the waters divided
	day3: "I I IV IV V V  I IV I IV I IV",                       // the land rises, the plants grow on I and IV
	day4: "vi IV I V vi V  I iii IV I V I",                      // night, then the sun
	day5: "vi vi iii iii IV V  I IV V I IV V",                   // the deep, then the birds
	day6: "I V vi IV I V  I vi IV V IV I",                       // the beasts, then man: very good
	day7: "I vi IV I I iii IV V I V vi I iii I V7 I IV I I I I", // the Doxology, the Amen (IV I) and the rest
}

// bars holds the chord and the voicings of every bar of the song
type bar struct {
	chord harmony.Chord
	high  []float64 // upper voices (violins)
	mid   []float64 // middle voices (choir, organ, horns)
	root  float64   // the bass
}

var bars = func() []bar {
	var bs []bar
	high, mid := []float64(nil), []float64(nil)
	for _, s := range form.Sections {
		chords := key.MustProgression(days[s.Name])
		if len(chords) != s.Bars {
			panic(fmt.Sprintf("%s: %d chords for %d bars", s.Name, len(chords), s.Bars))
		}
		for _, c := range chords {
			if high == nil {
				high, mid = c.Close(67), c.Close(55)
			} else {
				high, mid = harmony.VoiceLead(high, c), harmony.VoiceLead(mid, c)
			}
			bs = append(bs, bar{chord: c, high: high, mid: mid, root: c.BassPitch(34)})
		}
	}
	return bs
}()

// at returns a bar of a section
func at(s arrange.Section, b int) bar { return bars[s.StartBar+b] }

// timeline lays out the chord of every bar, for the harmony check
func timeline() []harmony.TimedChord {
	var tl []harmony.TimedChord
	for i, b := range bars {
		tl = append(tl, harmony.TimedChord{Beat: float64(i) * form.BeatsPerBar, Beats: form.BeatsPerBar, Chord: b.chord})
	}
	return tl
}

// --- tuning: just intonation in E flat ---

// justRatios are the pure ratios of the twelve semitones above the tonic (five limit): the scale
// of E flat is 1, 9/8, 5/4, 4/3, 3/2, 5/3 and 15/8, so I, IV, V, vi and iii are pure triads
var justRatios = [12]float64{1, 16.0 / 15, 9.0 / 8, 6.0 / 5, 5.0 / 4, 4.0 / 3, 45.0 / 32, 3.0 / 2, 8.0 / 5, 5.0 / 3, 9.0 / 5, 15.0 / 8}

const tonic = 63 // E flat 4

// tuning turns a MIDI pitch into a frequency
type tuning func(pitch float64) float64

func justTuning(a4 float64) tuning {
	eb := a4 * math.Exp2((tonic-69)/12.0) // the tonic where equal temperament has it
	return func(pitch float64) float64 {
		steps := int(math.Round(pitch)) - tonic
		octave := math.Floor(float64(steps) / 12)
		return eb * math.Exp2(octave) * justRatios[steps-int(octave)*12]
	}
}

func equalTuning(a4 float64) tuning {
	return func(pitch float64) float64 { return a4 * math.Exp2((pitch-69)/12) }
}

// tuned returns a score that plays its pitches as frequencies of a tuning
func tuned(sc arrange.Score, tune tuning) arrange.Score {
	return func(s arrange.Section) []score.Note {
		notes := sc(s)
		for i := range notes {
			fs := make([]float64, len(notes[i].Pitches))
			for k, p := range notes[i].Pitches {
				fs[k] = tune(p)
			}
			notes[i].Pitches = fs
		}
		return notes
	}
}

// --- melodies ---

// the theme of man, made in the image of God (day 6, over I vi IV V IV I)
var theme = score.MustParse("G4:2 Bb4:1 | C5:2 Eb5:1 | Eb5:1 D5 C5 | Bb4:2 D5:1 | C5:1 Bb4 Ab4 | G4:1 F4 Eb4")

// the horn answers: its fourth bar and fifth bar hold the figure of the cross (Bb G C Bb: the first
// and last note the beam, the second and third crossing it)
var answer = score.MustParse("Eb3:3 | C3:3 | Ab2:3 | Bb2:1 G2 C3 | Bb2:3 | Eb3:3")

// cross is the figure of the cross by itself, two bars
var cross = score.MustParse("Bb3:1 G3 C4 | Bb3:3")

// the moon (day 4, over vi IV I V vi V)
var moon = score.MustParse("Eb5:2 C5:1 | C5:2 Ab4:1 | Bb4:2 G4:1 | F4:2 Bb4:1 | C5:1 D5 Eb5 | D5:3")

// the canon of the plants, every phrase two bars over I and IV: the flute sings A B C, the oboe
// follows two bars later with A B, the clarinet two more with A, an octave down
var (
	seedA = score.MustParse("Bb4:1 G4 Eb4 | C5:2 Ab4:1")
	seedB = score.MustParse("G4:2 Bb4:1 | Ab4:1 C5 Eb5")
	seedC = score.MustParse("Eb5:2 D5:1 | C5:3")
)

// the Doxology: the Old Hundredth in E flat, a long first and last note in every line, three beats
// to the bar
var doxology = score.MustParse(
	"Eb4:3 | Eb4:1 D4 C4 | Bb3:1 Eb4 F4 | G4:3 | " + // Praise God, from whom all blessings flow
		"G4:3 | G4:1 G4 F4 | Eb4:1 Ab4 G4 | F4:3 | " + // praise him, all creatures here below
		"Eb4:3 | F4:1 G4 F4 | Eb4:1 C4 D4 | Eb4:3 | " + // praise him above, ye heavenly host
		"Bb4:3 | G4:1 Eb4 F4 | Ab4:1 G4 F4 | Eb4:3") // praise Father, Son, and Holy Ghost

// arp is six eighth notes up and down a chord
func arp(pitches []float64) []score.Note {
	ps := slices.Clone(pitches)
	slices.Sort(ps)
	order := []int{0, 1, 2, 3, 2, 1}
	var notes []score.Note
	for _, i := range order {
		notes = append(notes, score.N(ps[min(i, len(ps)-1)], 0.5))
	}
	return notes
}

// birdsong is a bar of quick calls on the tones of a chord
func birdsong(c harmony.Chord, alt bool) []score.Note {
	p := c.Close(72)
	p = append(p, p[0]+12)
	if alt {
		return score.Join(score.Notes(0.25, p[3], p[2], p[3], p[2]), score.Notes(1, 0), score.Notes(0.5, p[1], p[3]))
	}
	return score.Join(score.Notes(0.25, p[1], p[2]), score.Notes(0.5, p[3], 0), score.Notes(0.25, p[2], p[3]), score.Notes(1, p[1]))
}

// stars is a bar of slow twinkling high in a chord
func stars(c harmony.Chord, b int) []score.Note {
	p := c.Close(79)
	switch b % 3 {
	case 0:
		return score.Notes(1, p[2], 0, p[0])
	case 1:
		return score.Join(score.Notes(0.5, 0, p[1]), score.Notes(2, p[2]))
	}
	return score.Notes(1.5, p[0]+12, p[1])
}

// --- curves for the arrangement ---

// pedalChanged is a sustain pedal changed at every bar line of the bars from..to of a section of a
// number of bars: up just long enough to clear the old chord, then down; up outside them
func pedalChanged(bars, from, to int) arrange.Curve {
	return func(progress float64) float64 {
		at := progress * float64(bars)
		if at < float64(from) || at >= float64(to) || at-math.Floor(at) < 0.05 {
			return 0
		}
		return 1
	}
}

// swell is a level in dB in a section, faded in and out at its edges over a part of it
func swell(db float64, from, to float64) arrange.Curve {
	const edge = 0.06
	return func(progress float64) float64 {
		if progress < from || progress > to {
			return mixer.MinDB
		}
		in := math.Min((progress-from)/edge, 1)
		out := math.Min((to-progress)/edge, 1)
		return mixer.MinDB + (db-mixer.MinDB)*math.Pow(math.Min(in, out), 0.15)
	}
}

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
	var tune tuning
	switch *tuningArg {
	case "just":
		tune = justTuning(*a4)
	case "equal":
		tune = equalTuning(*a4)
	default:
		panic(fmt.Sprintf("tuning %q: just or equal", *tuningArg))
	}

	psalm.PushConfig(psalm.Config{BPM: bpm, PPQ: 240, SampleRate: 44100, BufferSize: 1024})

	offline := *renderTo != "" || *profile
	var p *psalm.Psalm
	if offline {
		p = psalm.New(nil)
	} else {
		p = psalm.New(paengine.New(psalm.SampleRate(), psalm.BufferSize()))
	}
	defer p.Close()

	// --- the mixer: a hall and the heavens as effect buses; creation plays through the darkness
	// that lifts on the first day, the choir and the organ sing in a church ---

	mix := mixer.New(p)
	type sends = map[string]float64

	hcfg := fdnreverb.DefaultConfig()
	hcfg.Decay = 4.2
	hcfg.Size = 0.85
	hcfg.Damping = 0.5
	hcfg.PreDelay = 30
	hcfg.Mix = 1
	mix.Bus("hall", mixer.StripConfig{}, fdnreverb.New(hcfg).AddTo(p))

	scfg := shimmer.DefaultConfig()
	scfg.Reverb.Decay = 8
	scfg.Shimmer = 0.5
	scfg.Mix = 1
	mix.Bus("heavens", mixer.StripConfig{}, shimmer.New(scfg).AddTo(p))

	darkness := filters.NewLadder(ladder.LowPass12, 300, 0.1, 1.0)
	darkness.AddTo(p)
	mix.Group("creation", mixer.StripConfig{}, darkness)

	choirClean := filters.NewSVF(svf.HighPass, 110, 0.1)
	choirClean.AddTo(p)
	church := convolver.IRFromSoundFile(sndfile.MustSoundFile[float32]("assets/IR44100/St Nicolaes Church.wav"))
	mix.Group("church", mixer.StripConfig{Volume: -2}, choirClean,
		convolver.New(church, convolver.Config{PartitionSize: 4096, Mix: 0.4, Gain: 2.2}).AddTo(p))

	channel := func(name string, m psalm.Modular, cfg mixer.StripConfig) *mixer.Strip {
		if cfg.Group == "" {
			cfg.Group = "creation"
		}
		return mix.Channel(name, m, cfg)
	}

	// --- the waters, the land and the living: recordings of creation itself ---

	recording := func(name, file string, pan float64) *mixer.Strip {
		pl := player.New(player.Config{SndFile: sndfile.MustMipMapSoundFile[float32](file, 4), Loop: true, Speed: 1, Amp: 1}).AddTo(p)
		return channel(name, pl, mixer.StripConfig{Volume: mixer.MinDB, Pan: pan, Sends: sends{"hall": 0.15}})
	}
	waves := recording("waves", "assets/sounds/various/boat_waves-6099.wav", 0)
	seaBirds := recording("sea and birds", "assets/sounds/various/mixkit-sea-waves-with-birds-loop-1185.wav", 0.2)
	gulls := recording("gulls", "assets/sounds/various/seagulls-by-the-sea-7042.wav", -0.3)
	animals := recording("animals", "assets/sounds/various/animals.wav", 0)
	// the children laugh once, from the start of the recording (it is started on its bar)
	laughter := player.New(player.Config{SndFile: sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/mixkit-laughing-children-indoors-427.wav", 4), Speed: 1, Amp: 1})
	laughter.AddTo(p)
	children := channel("children", laughter, mixer.StripConfig{Volume: mixer.MinDB, Pan: 0.1, Sends: sends{"hall": 0.25}})

	// --- the void and the Spirit ---

	void := wavetable.NewPoly(wavetable.DronePreset(gen.Random(64, 6, 32, 7)), 4).AddTo(p)
	channel("void", void, mixer.StripConfig{Volume: -3, Sends: sends{"hall": 0.4}})

	spirit := pulsar.NewPoly(pulsar.ChoirPreset(), 12).AddTo(p)
	channel("spirit", spirit, mixer.StripConfig{Group: "church", Sends: sends{"heavens": 0.2}})

	// the Word: John 1:1 spoken through a pad of open fifths, heard clearly in the darkness
	word := player.New(player.Config{
		SndFile: sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/John_1-1.wav", 4),
		Speed:   1.0,
		Amp:     0.8,
	})
	word.AddTo(p)
	wordCfg := wavetable.PadPreset(gen.Brightness(16))
	wordCfg.Position = 1
	wordCfg.PositionLFODepth = 0
	wordCfg.Filter = wavetable.FilterOff
	wordCfg.Unison = 3
	wordCfg.AmpEnv.AttackDuration = 200
	wordPad := wavetable.NewPoly(wordCfg, 8).AddTo(p)
	voc := vocoder.New(vocoder.DefaultConfig())
	voc.AddTo(p)
	voc.In(word, 0, 0, wordPad, 0, 1)
	mix.Channel("the word", voc, mixer.StripConfig{Volume: -3, Sends: sends{"hall": 0.35, "heavens": 0.1}})

	// --- strings ---

	violinsCfg := wavetable.StringsPreset(gen.Brightness(64))
	violinsCfg.Position = 0.8
	violins := wavetable.NewPoly(violinsCfg, 16).AddTo(p)
	channel("violins", violins, mixer.StripConfig{Pan: -0.35, Sends: sends{"hall": 0.45}})

	celli := subtractive.NewPoly(subtractive.StringsPreset(), 8).AddTo(p)
	channel("celli", celli, mixer.StripConfig{Pan: 0.35, Sends: sends{"hall": 0.4}})

	pizz := fm.NewPoly(fm.PizzicatoPreset(), 4).AddTo(p)
	channel("pizzicato", pizz, mixer.StripConfig{Pan: 0.45, Sends: sends{"hall": 0.3}})

	harp := fm.NewPoly(fm.HarpPreset(), 12).AddTo(p)
	channel("harp", harp, mixer.StripConfig{Pan: 0.55, Sends: sends{"hall": 0.5}})

	timpani := fm.NewPoly(fm.TimpaniPreset(), 4).AddTo(p)
	channel("timpani", timpani, mixer.StripConfig{Pan: -0.1, Sends: sends{"hall": 0.45}})

	// --- winds and brass ---

	flute := fm.NewPoly(fm.FlutePreset(), 4).AddTo(p)
	piccolo := pitchshift.New(12, 60, 0.15).AddTo(p).In(flute) // on the fifth day, the birds an octave above
	channel("flute", piccolo, mixer.StripConfig{Pan: -0.15, Sends: sends{"hall": 0.4, "heavens": 0.1}})

	oboe := fm.NewPoly(fm.OboePreset(), 2).AddTo(p)
	channel("oboe", oboe, mixer.StripConfig{Pan: 0.2, Sends: sends{"hall": 0.4}})

	clarinet := fm.NewPoly(fm.ClarinetPreset(), 2).AddTo(p)
	channel("clarinet", clarinet, mixer.StripConfig{Pan: -0.4, Sends: sends{"hall": 0.4}})

	horns := dx.NewPoly(dx.SoftBrass(), 12, psalm.Stereo).AddTo(p)
	channel("horns", horns, mixer.StripConfig{Pan: -0.2, Sends: sends{"hall": 0.45}})

	horn := fm.NewPoly(fm.FrenchHornPreset(), 2).AddTo(p)
	channel("horn", horn, mixer.StripConfig{Pan: 0.25, Sends: sends{"hall": 0.4}})

	// --- the lights: celesta and bells, stars in the heavens ---

	celesta := fm.NewPoly(fm.CelestaPreset(), 8).AddTo(p)
	celestaEcho := pingpong.New(pingpong.Config{Time: 0.833, Feedback: 0.35, Mix: 0.25, HighCut: 6000, LowCut: 250}).AddTo(p).In(celesta)
	channel("celesta", celestaEcho, mixer.StripConfig{Pan: 0.3, Sends: sends{"heavens": 0.8}})

	bellPad := dx.NewPoly(dx.BellPad(), 12, psalm.Stereo).AddTo(p)
	channel("bell pad", bellPad, mixer.StripConfig{Sends: sends{"hall": 0.4, "heavens": 0.3}})

	// tuned bells: every partial a whole number times the note (the 1, 2 and 4 of the music box, the
	// 7 of its modulator only adding harmonics), ringing long and soft
	bellCfg := fm.MusicBoxPreset()
	bellCfg.Ops[0].Env = *adsr.NewSetting(1.0, 4.0, 0.0, 6000.0, 1.0, 3000.0)
	bellCfg.Ops[1].Env = *adsr.NewSetting(1.0, 4.0, 0.0, 3500.0, 1.0, 2000.0)
	bellCfg.Ops[2].Env = *adsr.NewSetting(1.0, 4.0, 0.0, 1800.0, 1.0, 1000.0)
	bellCfg.Ops[3].Level = 0.1
	bellCfg.Ops[3].Env = *adsr.NewSetting(1.0, 4.0, 0.0, 300.0, 1.0, 200.0)
	bells := fm.NewPoly(bellCfg, 4).AddTo(p)
	channel("bells", bells, mixer.StripConfig{Pan: 0.15, Sends: sends{"hall": 0.5, "heavens": 0.3}})

	// --- the living: the creatures of the sea, mankind ---

	deep := pulsar.NewPoly(pulsar.FormantPadPreset(), 4).AddTo(p)
	channel("the deep", deep, mixer.StripConfig{Sends: sends{"hall": 0.5}})

	grand := piano.NewPoly(piano.GrandPreset(), 24).AddTo(p)
	channel("piano", grand, mixer.StripConfig{Pan: -0.05, Sends: sends{"hall": 0.35}})

	soprano := pulsar.NewPoly(pulsar.VowelLeadPreset(), 2).AddTo(p)
	channel("soprano", soprano, mixer.StripConfig{Group: "church", Sends: sends{"heavens": 0.15}})

	choirRec := sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/mixkit-angelical-choir-654.wav", 4)
	choir := grains.NewPoly(grains.ChoirPad(choirRec, 60), 12).AddTo(p)
	channel("choir", choir, mixer.StripConfig{Group: "church"})

	organ := fm.NewPoly(fm.ChurchOrganPreset(), 12).AddTo(p)
	channel("organ", organ, mixer.StripConfig{Group: "church"})

	// --- the scores ---

	type S = arrange.Section
	type N = score.Note
	tl := timeline()
	clashes := 0
	feels := map[string]score.Feel{
		"violins":   {Timing: 8, Spread: 12, Velocity: 0.06},
		"celli":     {Timing: 8, Spread: 10, Velocity: 0.06},
		"pizzicato": {Timing: 6, Velocity: 0.1, Accents: []float64{1, 0.8, 0.85}, BeatsPerBar: 3},
		"harp":      {Timing: 5, Velocity: 0.1},
		"timpani":   {Timing: 4, Velocity: 0.08},
		"flute":     {Timing: 10, Push: 6, Velocity: 0.08},
		"oboe":      {Timing: 10, Push: 6, Velocity: 0.08},
		"clarinet":  {Timing: 10, Push: 6, Velocity: 0.08},
		"horns":     {Timing: 10, Spread: 15, Velocity: 0.06},
		"horn":      {Timing: 12, Push: 4, Velocity: 0.08},
		"soprano":   {Timing: 10, Push: 6, Velocity: 0.08},
		"spirit":    {Timing: 10, Spread: 20},
		"choir":     {Timing: 10, Spread: 20},
		"organ":     {Timing: 4, Spread: 6},
		"celesta":   {Timing: 6, Velocity: 0.12},
		"bell pad":  {Strum: 25, Velocity: 0.05},
		"bells":     {Strum: 40},
		"piano":     {Timing: 6, Strum: 18, Velocity: 0.06},
		"the deep":  {Timing: 20},
	}

	// part plays a score in the tuning; every part is checked against the chords first (the void
	// holds its fifth through the chords on purpose)
	part := func(synth psalm.Modular, name string, amp float64, legato float64, sc arrange.Score) {
		form.Part(p, synth, name, score.Config{Amp: amp, Legato: legato, Velocity: 0.8, Key: "frequency", Feel: feels[name]}, tuned(sc, tune))
		if name == "void" {
			return
		}
		issues := harmony.Check(form.Compile(name, sc), tl)
		clashes += len(issues)
		fmt.Print(harmony.Report(name, issues))
	}
	perBar := func(s S, f func(b int, br bar) []N) []N {
		return arrange.PerBar(s, func(b int) []N { return f(b, at(s, b)) })
	}
	morning := func(b int) bool { return b >= evening }
	// the first day sounds out when the light comes, its evening stays as quiet as the darkness
	dark := 0.9 / 1.3

	// the void: "the earth was without form, and void; and darkness was upon the face of the deep",
	// an open fifth; it returns under the last chord of the seventh day
	part(void, "void", 0.111, 1.0, func(s S) []N {
		switch s.Name {
		case day1:
			return []N{{Pitches: []float64{39, 46}, Beats: 18, Hold: 19, Amp: dark}, score.Rest(18)}
		case day7:
			return []N{score.Rest(51), {Pitches: []float64{39, 46, 51}, Beats: 12, Hold: 12}}
		}
		return nil
	})

	// "and the Spirit of God moved upon the face of the waters": a breathing choir in the darkness,
	// the heaven between the waters on the second morning, the morning stars on the fourth
	part(spirit, "spirit", 0.061, 1.0, func(s S) []N {
		switch s.Name {
		case day1:
			return perBar(s, func(b int, br bar) []N {
				if morning(b) {
					return []N{score.Chord(br.mid, 3)}
				}
				return []N{{Pitches: []float64{51, 58}, Beats: 3, Velocity: 0.6, Amp: dark}}
			})
		case day2, day4:
			return perBar(s, func(b int, br bar) []N {
				if !morning(b) {
					return []N{score.Rest(3)}
				}
				return []N{score.Chord(br.mid, 3)}
			})
		case day7:
			return perBar(s, func(b int, br bar) []N { return []N{score.Chord(br.mid, 3)} })
		}
		return nil
	})

	// the Word sounds through open fifths, two readings in the darkness of the first evening
	part(wordPad, "the word", 0.0976, 1.0, func(s S) []N {
		if s.Name != day1 {
			return nil
		}
		return []N{score.Rest(6), {Pitches: []float64{39, 46, 51, 58}, Beats: 12, Amp: dark}, score.Rest(18)}
	})

	part(violins, "violins", 0.126, 1.0, func(s S) []N {
		switch s.Name {
		case day1, day4, day6:
			return perBar(s, func(b int, br bar) []N {
				if !morning(b) {
					return []N{score.Rest(3)}
				}
				return []N{score.Chord(br.high, 3)}
			})
		case day2:
			// the waters above, very high
			return perBar(s, func(b int, br bar) []N { return []N{score.Chord(score.Pitches(br.high, 12), 3)} })
		case day7:
			return perBar(s, func(b int, br bar) []N {
				if b == s.Bars-4 {
					return []N{{Pitches: br.high, Beats: 12, Hold: 12}} // the last chord rings
				} else if b > s.Bars-4 {
					return nil
				}
				return []N{score.Chord(br.high, 3)}
			})
		}
		return nil
	})

	part(celli, "celli", 0.145, 1.0, func(s S) []N {
		switch s.Name {
		case day2:
			// the waters below, very low
			return perBar(s, func(b int, br bar) []N { return []N{score.Chord([]float64{br.root, br.root + 7}, 3)} })
		case day3, day5, day6:
			return perBar(s, func(b int, br bar) []N { return []N{score.Chord([]float64{br.root + 12, br.root + 19}, 3)} })
		case day7:
			return perBar(s, func(b int, br bar) []N {
				if b == s.Bars-4 {
					return []N{{Pitches: []float64{br.root + 12, br.root + 19}, Beats: 12, Hold: 12}}
				} else if b > s.Bars-4 {
					return nil
				}
				return []N{score.Chord([]float64{br.root + 12, br.root + 19}, 3)}
			})
		}
		return nil
	})

	// the land walks: the dry land on the third evening, the beasts on the sixth
	part(pizz, "pizzicato", 0.25, 0.5, func(s S) []N {
		if !arrange.In(s, day3, day6) {
			return nil
		}
		return perBar(s, func(b int, br bar) []N {
			if morning(b) {
				return []N{score.Rest(3)}
			}
			r := br.root + 12
			return score.Notes(1, r, r+7, r+12)
		})
	})

	// the waters flow on the second morning
	part(harp, "harp", 0.13, 1.6, func(s S) []N {
		if s.Name != day2 {
			return nil
		}
		return perBar(s, func(b int, br bar) []N {
			if !morning(b) {
				return []N{score.Rest(3)}
			}
			return arp(append([]float64{br.root + 12}, br.mid...))
		})
	})

	// the dry land appears, the beasts tread, the Amen
	part(timpani, "timpani", 0.2, 1.0, func(s S) []N {
		switch s.Name {
		case day1:
			return perBar(s, func(b int, br bar) []N {
				if b != evening {
					return []N{score.Rest(3)}
				}
				return []N{{Pitches: []float64{39}, Beats: 3, Velocity: 0.9}} // let there be light
			})
		case day3, day6:
			return perBar(s, func(b int, br bar) []N {
				if morning(b) {
					return []N{score.Rest(3)}
				}
				r := br.root
				for r < 36 {
					r += 12
				}
				return []N{{Pitches: []float64{r}, Beats: 2, Velocity: 0.6}, {Pitches: []float64{r}, Beats: 1, Velocity: 0.4}}
			})
		case day7:
			return perBar(s, func(b int, br bar) []N {
				if b != s.Bars-4 {
					return []N{score.Rest(3)}
				}
				return []N{{Pitches: []float64{39}, Beats: 3, Velocity: 0.6}}
			})
		}
		return nil
	})

	// the plants in canon on the third morning, the moon on the fourth evening, the birds on the
	// fifth morning
	part(flute, "flute", 0.21, 0.95, func(s S) []N {
		switch s.Name {
		case day3:
			return score.Join([]N{score.Rest(18)}, seedA, seedB, seedC)
		case day4:
			return score.Join(moon, []N{score.Rest(18)})
		case day5:
			return perBar(s, func(b int, br bar) []N {
				if !morning(b) {
					return []N{score.Rest(3)}
				}
				return birdsong(br.chord, b%2 == 1)
			})
		}
		return nil
	})

	part(oboe, "oboe", 0.2, 0.95, func(s S) []N {
		if s.Name != day3 {
			return nil
		}
		return score.Join([]N{score.Rest(24)}, seedA, seedB)
	})

	part(clarinet, "clarinet", 0.25, 0.95, func(s S) []N {
		if s.Name != day3 {
			return nil
		}
		return score.Join([]N{score.Rest(30)}, score.Transpose(seedA, -12))
	})

	// light breaks, the sun rises, the beasts, the seventh day
	part(horns, "horns", 0.14, 0.95, func(s S) []N {
		switch s.Name {
		case day1, day4:
			return perBar(s, func(b int, br bar) []N {
				if !morning(b) {
					return []N{score.Rest(3)}
				}
				return []N{score.Chord(score.Pitches(br.mid, -12), 3)}
			})
		case day6:
			return perBar(s, func(b int, br bar) []N {
				if morning(b) {
					return []N{score.Rest(3)}
				}
				return []N{score.Chord(score.Pitches(br.mid, -12), 3)}
			})
		}
		return nil
	})

	// Adam answers Eve with the figure of the cross; on the seventh day the cross leads through the
	// Amen into the rest
	part(horn, "horn", 0.33, 0.95, func(s S) []N {
		switch s.Name {
		case day6:
			return score.Join([]N{score.Rest(18)}, answer)
		case day7:
			return score.Join([]N{score.Rest(48)}, cross, []N{score.Rest(9)})
		}
		return nil
	})

	// the stars of the first morning and of the fourth night
	part(celesta, "celesta", 0.217, 0.9, func(s S) []N {
		switch s.Name {
		case day1:
			return perBar(s, func(b int, br bar) []N {
				if !morning(b) {
					return []N{score.Rest(3)}
				}
				return stars(br.chord, b)
			})
		case day4:
			return perBar(s, func(b int, br bar) []N {
				if morning(b) {
					return []N{score.Rest(3)}
				}
				return stars(br.chord, b)
			})
		}
		return nil
	})

	part(bellPad, "bell pad", 0.0471, 1.0, func(s S) []N {
		if !arrange.In(s, day1, day4) {
			return nil
		}
		return perBar(s, func(b int, br bar) []N {
			if (s.Name == day1) != morning(b) {
				return []N{score.Rest(3)}
			}
			return []N{score.Chord(br.mid, 3)}
		})
	})

	// the bells ring once, as the Amen comes home
	part(bells, "bells", 0.12, 1.0, func(s S) []N {
		if s.Name != day7 {
			return nil
		}
		return []N{score.Rest(51), {Pitches: []float64{63, 70, 75}, Beats: 12, Hold: 12, Velocity: 0.5}}
	})

	// the great creatures of the sea, low in the deep
	part(deep, "the deep", 0.2, 1.0, func(s S) []N {
		if s.Name != day5 {
			return nil
		}
		return perBar(s, func(b int, br bar) []N {
			if morning(b) {
				return []N{score.Rest(3)}
			}
			return []N{{Pitches: []float64{br.root}, Beats: 3, Hold: 3.2}}
		})
	})

	// mankind: the hands of man on the piano
	part(grand, "piano", 0.2, 1.0, func(s S) []N {
		if s.Name != day6 {
			return nil
		}
		return perBar(s, func(b int, br bar) []N {
			if !morning(b) {
				return []N{score.Rest(3)}
			}
			return arp(append([]float64{br.root + 12, br.root + 19}, br.mid...))
		})
	})

	// Eve sings the theme of man; on the seventh day the soprano leads the Doxology
	part(soprano, "soprano", 0.34, 0.95, func(s S) []N {
		switch s.Name {
		case day6:
			return score.Join([]N{score.Rest(18)}, theme)
		case day7:
			return score.Join(doxology, []N{score.Rest(15)})
		}
		return nil
	})

	// the choir: the light, the morning stars, the Doxology
	part(choir, "choir", 0.11, 1.0, func(s S) []N {
		switch s.Name {
		case day1, day4, day6:
			return perBar(s, func(b int, br bar) []N {
				if !morning(b) {
					return []N{score.Rest(3)}
				}
				return []N{score.Chord(br.mid, 3)}
			})
		case day7:
			return perBar(s, func(b int, br bar) []N {
				if b == s.Bars-4 {
					return []N{{Pitches: br.mid, Beats: 12, Hold: 12}}
				} else if b > s.Bars-4 {
					return nil
				}
				return []N{score.Chord(br.mid, 3)}
			})
		}
		return nil
	})

	// the organ plays the Doxology in four parts and holds the Amen
	part(organ, "organ", 0.1, 1.0, func(s S) []N {
		if s.Name != day7 {
			return nil
		}
		return perBar(s, func(b int, br bar) []N {
			v := append([]float64{br.root}, br.mid...)
			if b == s.Bars-4 {
				return []N{{Pitches: v, Beats: 12, Hold: 12}}
			} else if b > s.Bars-4 {
				return nil
			}
			return []N{score.Chord(v, 3)}
		})
	})

	if clashes == 0 {
		fmt.Println("harmony check: every part fits its chords")
	}

	// --- the arrangement: the light, the voices of creation, the pedal, the rest ---

	// start plays a recording from its start when a lane rises
	start := func(pl *player.Player) func(v float64) {
		last := 0.0
		return func(v float64) {
			if v > 0.5 && last <= 0.5 {
				pl.Bang()
			}
			last = v
		}
	}
	// onBars rises at bars of a section of a number of bars
	onBars := func(bars float64, at ...float64) arrange.Curve {
		return func(progress float64) float64 {
			b := progress * bars
			for _, a := range at {
				if b >= a && b < a+0.1 {
					return 1
				}
			}
			return 0
		}
	}
	arrangement := arrange.NewArrangement(form).
		Tempo(arrange.Lane{Sections: map[string]arrange.Curve{day7: func(progress float64) float64 {
			if progress < 17.0/21 {
				return restBPM
			}
			return arrange.Smooth(restBPM, 48)((progress - 17.0/21) * 21 / 4)
		}}}).
		// "Let there be light": the darkness lifts in the first bar of the first morning
		Automate(arrange.Lane{Default: 18000, Sections: map[string]arrange.Curve{day1: func(progress float64) float64 {
			light := float64(evening) / 12
			if progress < light {
				return 300
			}
			return arrange.Exp(300, 18000)(math.Min((progress-light)*12, 1))
		}}}, darkness, 1).
		// the Word is spoken at the third and the fifth bar of the first day
		AutomateFunc(arrange.Lane{Sections: map[string]arrange.Curve{day1: onBars(12, 2, 4)}}, start(word)).
		Automate(arrange.Lane{Default: mixer.MinDB, Sections: map[string]arrange.Curve{
			day1: swell(-14, 0, 0.5), // the deep
			day2: swell(-8, 0, 1),    // the waters
			day3: swell(-8, 0, 0.5),  // the seas gathered together
		}}, waves, 1).
		Automate(arrange.Lane{Default: mixer.MinDB, Sections: map[string]arrange.Curve{day5: swell(-8, 0, 1)}}, seaBirds, 1).
		Automate(arrange.Lane{Default: mixer.MinDB, Sections: map[string]arrange.Curve{day5: swell(-16, 0.5, 1)}}, gulls, 1).
		Automate(arrange.Lane{Default: mixer.MinDB, Sections: map[string]arrange.Curve{day6: swell(-10, 0, 0.55)}}, animals, 1).
		// "be fruitful, and multiply" (Genesis 1:28): the children laugh through the last four bars of
		// the sixth day
		AutomateFunc(arrange.Lane{Sections: map[string]arrange.Curve{day6: onBars(12, 8)}}, start(laughter)).
		Automate(arrange.Lane{Default: mixer.MinDB, Sections: map[string]arrange.Curve{day6: swell(-3, 8.0/12, 1)}}, children, 1).
		// the piano's sustain pedal changes at every bar of the sixth morning
		AutomateMessage(arrange.Lane{Sections: map[string]arrange.Curve{day6: pedalChanged(12, evening, 12)}}, grand, "sustain").
		OnSection(func(pos arrange.Position) {
			fmt.Printf("\n%s\n%s\n", pos.Section.Name, pos.Section.Params["verse"])
		})
	arrangement.AddTo(p)

	fmt.Println("Creation")
	fmt.Println("In the beginning was the Word, and the Word was with God, and the Word was God. All things were made by him. (John 1:1, 3)")
	fmt.Print(arrangement.Schedule())

	p.Out(mix.Master())

	opts := render.Options{Seconds: arrangement.Duration(), Tail: 12}
	switch {
	case *profile:
		fmt.Print(render.RunProfiled(p, opts))
	case *renderTo != "":
		var sources map[string]psalm.Modular
		if *stems {
			sources = render.MixerStems(mix)
		}
		stats, err := render.Stems(p, *renderTo, opts, sources)
		if err != nil {
			panic(err)
		}
		fmt.Println(stats)
		fmt.Print(mix.Meters())
	default:
		_ = p.Listen()
	}
}
