# Controls

Control modules have no audio, they create and route values and messages at control (tick)
time. A typical sequencing chain is:

```
timer  --bang + tick-->  templ  --trigger message-->  synth (poly)
```

For written music (melodies, chords, parts in a song form) use [score](#score) and
[arrange](#arrange) instead: notes in beats go straight into a synth, and a song is written section
by section.

## timer

`modules/controls/timer`

Sends a bang every N ticks, where N comes from a generator (`genny.Generator[float64]`). A
negative tick value is a rest: the timer waits that many ticks without banging.

| Constructor | Meaning |
|---|---|
| `timer.New(generator)` | Any float generator of tick durations (e.g. a `bucket`) |
| `timer.WithTicks(ticks)` | A constant interval, `timer.WithTicks(float64(psalm.PPQ()))` bangs every beat |
| `timer.WithSteps(steps)` | Follows a looping list of `steps.Step` (see below) |

**Value outputs**

| Output | Value |
|---|---|
| 0 | Message `{"bang": true, "tick": CtrlTick, "sender": id, "duration": seconds}` on every bang |
| 1 | The duration of the step in seconds |
| 2 | `psalm.Bang` when the generator is done and restarts (end of a sequence) |

The `tick` in output 0 lets templates and synths start notes sample accurately.

## steps

`modules/controls/timer/steps`

Step sequencing for `timer.WithSteps`. Each `steps.Step` lasts `Multiply` quarter notes
(1 when 0) and has optional variations:

| Field | Meaning |
|---|---|
| `Skip` | Rest instead of a hit |
| `Multiply` | Length in quarter notes (0.25 is a 16th) |
| `SkipChance` | Chance (0-1) the step is skipped |
| `Shuffle`, `ShuffleRand` | Delays the step for swing (0-1), with random variation |
| `BurstChance`, `NumBurst` | Chance to split the step into `NumBurst` fast repeats (rolls) |

Helpers:

- `steps.QuickSteps(args...)` builds steps from short values: `1`/`true` is a hit, `0`/`false`
  a rest, a float is a hit with that length (negative is a rest of that length), a `*steps.Step`
  is used as is.
- `steps.Multiply(steps, m)` scales the length of all steps, for example `0.25` for 16ths.

```go
// a 16th note pattern
t := timer.WithSteps(steps.Multiply(steps.QuickSteps(1, 0, 1, 1, 0, 1, 0, 0), 0.25))
```

## templ

`modules/controls/templ`

Generates messages from a `genny/template.Template` every time it receives a bang on input 0.
Template values can be constants or generators (`bucket`, sequences, ...), so every message can
differ. Messages that arrive without a bang (like a timer's `tick`) are stored as template
parameters.

| Input | Meaning |
|---|---|
| 0 | Bang: generate and send the next message(s) |
| 1 | Bang: reset the template (generators start over) |

| Output | Value |
|---|---|
| 0 | The generated message(s) |
| 1 | `psalm.Bang` when the template is done |

Helpers that build note trigger messages for `poly` synths (see [synths](synths.md)):

| Helper | Message |
|---|---|
| `templ.NewTrigger(amp, msg)` | A trigger without duration (the voice decides the length, like a drum sample) |
| `templ.NewDurationTrigger(duration, amp, msg)` | A note of `duration` seconds |

`msg` is the template for the per note message (for example `{"pitch": ..., "cutoff": ...}`),
which the voice receives with the note. `duration` and `amp` can be generators too.

```go
synth.ValueIn(templ.NewDurationTrigger(0.25, bucket.NewLoop(0.2, 0.3).SetMode(bucket.Random), template.Template{
	"pitch": sequence.NewLoop(48.0, 55.0, 60.0), // in order, looping
}).ValueIn(timer.WithTicks(float64(psalm.PPQ())/2).AddTo(p)))
```

### Ordered and random values

Pick the genny generator by how values should come out:

| Generator | Order |
|---|---|
| `sequence.NewLoop(values...)` | In order, looping. Use it for melodies, chord progressions and any values that must line up with other sequences (pitch with duration, per step settings) |
| `bucket.NewLoop(values...).SetMode(bucket.Random)` | A random value every time |
| `bucket.NewLoop(values...)` | Every value once per cycle, but **shuffled**: a bucket shuffles its values when created and every time it starts over, also in `bucket.Indexed` mode |

### Chords

When a generator in a template produces a slice, the template generates one message per slice
element, all at the same time. With a `poly` synth that plays a chord: every note goes to its
own voice.

```go
pad.ValueIn(templ.NewDurationTrigger(2.0, 0.1, template.Template{
	"pitch": sequence.NewLoop(
		[]float64{57, 60, 64}, // Am
		[]float64{53, 57, 60}, // F
		[]float64{48, 52, 55}, // C
		[]float64{55, 59, 62}, // G
	),
}).ValueIn(timer.WithTicks(4 * float64(psalm.PPQ())).AddTo(p)))
```

The rules:

- The longest slice decides the number of messages. Slices are paired by position (the second
  message gets the second element of every slice), not combined.
- Shorter slices repeat their last element, plain values are the same in every message. So a
  scalar `amplitude` or `cutoff` applies to all notes of the chord, a slice gives each note its
  own value (`"amplitude": sequence.NewLoop([]float64{0.3, 0.2, 0.2})`).
- It also works inside the note message of `NewTrigger` / `NewDurationTrigger`: a slice in the
  nested template splits the whole trigger.
- Only values from generators (and template parameters) are split. A constant slice written
  directly in the template is passed on as a slice.

## score

`modules/controls/score`

Notes written in beats, played into a poly synth by a `Sequencer` at the exact tick. One module per
part, instead of a timer, a template and sequences of pitches and durations kept in step by hand.

```go
melody := score.MustParse("A4:2 F4:1 E4 | D4:3 F4:1 | [D4 F4 A4]:4")
lead := fm.NewPoly(fm.FlutePreset(), 2).AddTo(p)
lead.ValueIn(score.NewSequencer(melody, score.Config{Amp: 0.3, Loop: true}).AddTo(p))
```

A `score.Note` is a note, a chord (several pitches) or a rest (none):

| Field | Meaning |
|---|---|
| `Pitches` | MIDI pitches (or frequencies, with `Config.Key` `"frequency"`) |
| `Beats` | Time until the next note |
| `Hold` | Beats the note sounds, 0 is `Beats` times the legato of the part |
| `Velocity` | Sent as `velocity` when above 0, 0 is the part default |
| `Amp` | Scales the amplitude of the note, 0 is 1 |
| `Params` | Extra values for the synth voice, for example `"position"` or `"formant"` |

Helpers: `N(pitch, beats)`, `Chord(pitches, beats)`, `Rest(beats)`, `Notes(beats, pitches...)`
(0 is a rest), `Chords(beats, chords...)`, `Join`, `Repeat`, `Transpose`, `With` (change every
note), `Beats` (total length), `Pitches` (move a chord) and `Pitch("F#3")`.

**Notation** (`Parse`, `MustParse`), tokens separated by spaces:

| Token | Meaning |
|---|---|
| `A3:1` | A note (name like `C4` = 60, `F#3`, `Bb2`, or a MIDI number) lasting 1 beat |
| `[D4 F4 A4]:2` | A chord |
| `r:1`, `-:1` | A rest |
| `E4:1@.6` | Velocity 0.6 |
| `C5:1_3` | Lasting 1 beat, sounding for 3 |
| `G4` | Without a length: as long as the note before (1 beat at the start) |
| `\|` | A bar line, ignored, for reading |

**Sequencer config** (`score.Config`):

| Setting | Default | Meaning |
|---|---|---|
| `Amp` | 1 | Amplitude of the notes |
| `Legato` | 0.9 | Part of its length a note sounds when it has no `Hold` |
| `Velocity` | 0 | Velocity of notes without one, 0 sends none (the synth default) |
| `Key` | `"pitch"` | Message key of the pitches, `"frequency"` for Hz |
| `Loop` | false | Start again after the last note |
| `Length` | notes | Loop length in beats |

A chord is sent as one trigger per pitch. `SetGain(g)` scales the notes that follow, `SetNotes`
replaces them while playing.

## arrange

`modules/controls/arrange`

Songs built from sections. A `Form` lists the sections, every part is written per section and
compiled into a sequencer over the whole song, and the `Arrangement` module follows the song
position to automate settings and call section and bar events.

```go
form := arrange.NewForm(
	arrange.Section{Name: "Intro", Bars: 8, Dynamics: 0.8},
	arrange.Section{Name: "Verse", Bars: 16},
	arrange.Section{Name: "Outro", Bars: 8},
)

// a part: the notes per section, nil where it doesn't play
form.Part(p, pad, "pad", score.Config{Amp: 0.3}, func(s arrange.Section) []score.Note {
	if !arrange.In(s, "Verse", "Outro") {
		return nil
	}
	return arrange.PerBar(s, func(bar int) []score.Note {
		return []score.Note{score.Chord(chords[bar%4], 4)}
	})
})

// automation and events
arrange.NewArrangement(form).
	Automate(arrange.Lane{Default: 12000, Sections: map[string]arrange.Curve{
		"Intro": arrange.Exp(250, 12000), // the filter opens over the intro
	}}, filter, 1).
	OnSection(func(pos arrange.Position) { fmt.Println(pos.Section.Name) }).
	AddTo(p)

fmt.Print(form.Schedule())
```

**Sections** have a `Name`, a number of `Bars` and `Dynamics` (scales the amplitude of every part
in the section, default 1). `NewForm` sets their `Index` and `StartBar`, 4 beats per bar, looping
(`form.BeatsPerBar`, `form.Loop`).

**Parts**: `form.Part(p, synth, name, config, score)` compiles the part, creates a sequencer that
loops with the song, connects it to the synth and adds it to the patch. A section the part doesn't
play in (nil) is a rest. A section that doesn't last exactly its bars panics with the part and
section name, so mistakes show when the song is built, not as notes drifting out of time.
`form.Compile(name, score)` returns the notes without a sequencer. `arrange.PerBar(s, f)` writes a
section bar by bar, `arrange.In(s, names...)` checks the section.

**Automation**: a `Lane` has a `Curve` per section (a function of the progress 0-1 through the
section) and a `Default` for the other sections. Curves: `Const(v)`, `Linear(a, b)`, `Exp(a, b)`
(even steps per octave, for frequencies) and `Smooth(a, b)`, or your own function. Send a lane to:

| Method | Target |
|---|---|
| `Automate(lane, module, input)` | A value input (a filter cutoff is input 1) |
| `AutomateMessage(lane, module, key)` | A message key, for settings without a value input |
| `AutomateFunc(lane, func(v float64))` | Your own function |

Values are updated 48 times per beat and only sent when they change.

**Tempo**: `Tempo(lane)` automates the tempo in BPM. The arrangement pushes a config with the new
tempo (`psalm.PushConfig`), psalm and the sequencers follow it. Where the lane is 0 (a section
without a curve, with the default `Default` of 0) the tempo returns to where it started, so a
ritardando at the end only needs a curve for the last section:

```go
arrangement.Tempo(arrange.Lane{Sections: map[string]arrange.Curve{"Outro": arrange.Smooth(120, 90)}})
```

The arrangement replaces its own config on every change, so the config stack doesn't grow, and
removes it when the tempo returns. The tempo is updated 48 times per beat (changes smaller than
0.01 BPM are skipped), so a ritardando is smooth. The song position is counted in ticks, so parts
stay together; note lengths are converted to seconds when a note starts. The `pingpong` delay
follows the tempo too. Modules given a time in milliseconds computed from the tempo once (with
`timing.BPMToMilli`) don't.
`arrangement.Schedule()` returns the start time of every section following the tempo lane.

**Events**: `OnSection(f)` at the start of every section, `OnBar(f)` at every bar, with a
`Position` (song beat and bar, the section, `SectionBar()`, `Progress` through the section and the
`Loop` count). `PrintSections(true)` prints sections as they start, `PositionAt(tick)` returns the
position at any tick and `form.Schedule()` the start time of every section.

## gen

`modules/controls/gen`

Wraps any `genny.Generator[T]`: on a bang on input 0 it sends the next generated value on
output 0, and a bang on output 1 when the generator is done. A bang on input 1 resets the
generator. Other values go to an optional handler (`gen.NewWithConfig(generator, gen.Config{Handler: ...})`).

## dispatch

`modules/controls/dispatch`

Schedules events at absolute tick times, for arranging a piece (muting and unmuting parts,
sending messages).

```go
bar := psalm.PPQ() * 4
dispatch.New(
	dispatch.NewEvent(drums, bar*8, dispatch.Unmute),
	dispatch.NewEvent(lead, bar*16, dispatch.Mute),
	dispatch.NewEvent(reverb, bar*16, dispatch.Message, psalm.Message{"mix": 0.6}),
).AddTo(p)
```

| Event type | Effect on `Who` at tick `When` |
|---|---|
| `dispatch.Mute` | `SetMuted(true)` |
| `dispatch.Unmute` | `SetMuted(false)` |
| `dispatch.Message` | `ReceiveMessage(What)` |

Events are played once, sorted by time. More can be added with `AddEvents`.

## logger

`modules/controls/logger`

`logger.New(slogLogger)` logs every value it receives (with its input index), handy to inspect
what flows through value connections.
