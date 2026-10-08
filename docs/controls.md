# Controls

Control modules have no audio, they create and route values and messages at control (tick)
time. A typical sequencing chain is:

```
timer  --bang + tick-->  templ  --trigger message-->  synth (poly)
```

For written music (melodies, chords, parts in a song form) use [score](#score) and
[arrange](#arrange) instead: notes in beats go straight into a synth, and a song is written section
by section.

To play live from MIDI (a keyboard, a sequencer), see [MIDI](midi.md): the `keyboard` module
turns MIDI events into these messages.

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

`Shape(notes, shapers...)` gives a figure its accents and lengths: `Velocities(v...)` gives the
notes their velocities in turn (starting again at the first when there are more notes, an accent
pattern), `Hold(beats)` sets how long every note sounds:

```go
// a broken chord in the left hand, accented on the beat, held under the pedal
score.Shape(score.Notes(0.5, 39, 46, 51, 55, 58, 55, 51, 46),
	score.Velocities(0.6, 0.4, 0.45, 0.4), score.Hold(3))
```

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
| `Tuning` | none | Plays the pitches (MIDI) in a tuning, sent as `"frequency"`: `harmony.EqualTemperament(440)`, or just intonation in a key, `harmony.MustKey("Eb major").Just(440)` (see [Harmony](harmony.md#tuning)) |
| `Loop` | false | Start again after the last note |
| `Length` | notes | Loop length in beats |

A chord is sent as one trigger per pitch. `SetGain(g)` scales the notes that follow, `SetNotes`
replaces them while playing.

### Feel

`Config.Feel` takes the notes off the grid, like players:

```go
feel := score.Feel{Swing: 0.64, Timing: 8, Velocity: 0.15, Accents: []float64{1, 0.6, 0.8, 0.6}}
seq := score.NewSequencer(notes, score.Config{Amp: 0.3, Loop: true, Feel: feel})
```

| Setting | Meaning |
|---|---|
| `Swing` | Where the second note of a swung pair lands: 0 (or 0.5) is straight, 0.58 light, 0.66 a triplet feel, 0.75 hard |
| `SwingUnit` | Beats per swung note: 0.5 swings 8ths (the default), 0.25 16ths (a shuffle) |
| `Timing` | ± milliseconds of random timing per note, the notes of a chord move together |
| `Spread` | ± milliseconds of random timing per note within a chord (an ensemble never starts exactly together) |
| `Strum` | Milliseconds between the notes of a chord, low to high (negative is high to low) |
| `Push` | Milliseconds every note moves: negative plays ahead of the beat, positive lays back |
| `Velocity` | ± random variation of velocity and amplitude (0.1 is ±10%) |
| `Accents`, `AccentUnit`, `BeatsPerBar` | An amplitude per unit (1 beat by default) through the bar, for strong and weak beats |
| `Seed` | The same seed gives the same feel, 0 is a new feel every run |

Swing warps time within every pair of swing units, so notes that aren't on the grid and the
lengths of notes stretch along (a swung first 8th sounds longer). The random parts are drawn
again every time the loop starts, so repeats are never exactly the same. A note on the very first
beat of a sequence can't play early. `SetFeel(feel)` changes the feel while playing, for example
per section from an `arrange` event.

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

// a part: the notes per section, nil where it doesn't play (and a bar without notes is a rest)
form.Part(p, pad, "pad", score.Config{Amp: 0.3}, func(s arrange.Section) []score.Note {
	if !arrange.In(s, "Verse", "Outro") {
		return nil
	}
	return arrange.PerBar(s, func(bar int) []score.Note {
		if bar == 0 {
			return nil // the pad enters in the second bar
		}
		return []score.Note{score.Chord(chords[bar%4], 4)}
	})
})

// automation and events
arrange.NewArrangement(form).
	AutomateMessage(arrange.Sections{"Intro": arrange.Exp(250, 12000)}.Else(12000), filter, "cutoff"). // the filter opens over the intro
	AutomateMessage(arrange.Sections{"Outro": form.Span("Outro").Fade(-6, 2, 8)}.Else(arrange.Silent), rain, "volume").
	At("Verse", 0, player.Bang).
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
section bar by bar (a bar returning nil rests until the end of the bar: a bar of rest, or
nothing when a longer note before it lasts through it), `arrange.In(s, names...)` checks the
section and `arrange.Only(section, score)` plays a score written for one section only.
`arrange.Bars(section, func(s, bar) []score.Note)` is both in one, the common case: a part written
bar by bar for one section.

```go
form.Part(p, piano, "left hand", score.Config{Amp: 0.5}, arrange.Bars("Verse", func(s arrange.Section, bar int) []score.Note {
	if bar == 0 {
		return nil // enters in the second bar
	}
	return []score.Note{score.Chord(chords[bar%4].Close(48), 4)}
}))
```

**Automation**: a lane has a `Curve` per section (a function of the progress 0-1 through the
section) and a value for the other sections. `arrange.Sections{"Intro": curve}` is a lane by
itself, 0 in the other sections; `.Else(v)` gives them another value (the same as
`arrange.Lane{Default: v, Sections: ...}`). Curves: `Const(v)`, `Linear(a, b)`, `Exp(a, b)` (even
steps per octave, for frequencies) and `Smooth(a, b)`, or your own function. Send a lane to:

| Method | Target |
|---|---|
| `AutomateMessage(lane, module, key)` | A setting by name: `"cutoff"` of a filter, `"volume"` of a mixer strip, any message key |
| `Automate(lane, module, input)` | A value input by number (a filter cutoff and a strip volume are input 1) |
| `AutomateFunc(lane, func(v float64))` | Your own function |

Values are updated 48 times per beat and only sent when they change.

**Bars**: curves run over the whole section; a `Span` places them at bars instead (counted from
0, fractions allowed). `form.Span(name)` is the span of a section, `arrange.Span(12)` any number
of bars:

| Curve | |
|---|---|
| `span.Between(from, to, curve)` | The curve from one bar to another, its first value before and its last after |
| `span.From(bar, curve)`, `span.Last(bars, curve)` | The curve from a bar to the end, or over the last bars |
| `span.Fade(dB, from, to)` | A level from one bar to another, faded in and out (over a bar, even in amplitude), `arrange.Silent` outside: a sound that comes in for part of a section. A bar beyond the section (`-1`, or its bars + 1) leaves out the fade at that edge, for a sound that carries on into the next section |
| `span.Steps(v0, v1, ...)` | A value for every bar, the last one holding on: `Steps(0, 0, 0.5, 1)` |
| `span.On(from, to)` | 1 from one bar to another, 0 outside: a switch for part of a section |
| `span.Pedal(from, to)` | A sustain pedal changed at every bar line: up for a moment to clear the chord, down for the bar, up outside the bars |

```go
day := form.Span("Day 1")
arrangement.
	AutomateMessage(arrange.Sections{"Day 1": day.Between(6, 7, arrange.Exp(300, 18000))}.Else(18000), filter, "cutoff").
	AutomateMessage(arrange.Sections{"Day 6": form.Span("Day 6").Pedal(6, 12)}, piano, "sustain")
```

**Events**: `OnSection(f)` and `OnBar(f)` call a function at the start of every section and bar
with the position; `At(section, bar, f)` calls a function at the start of one bar (from 0), every
time the song gets there: to start a recording (`player.Bang`) or change a sound at a moment.

**Tempo**: `Tempo(lane)` automates the tempo in BPM. The arrangement sets the tempo of its patch
(`Env().SetBPM`), the patch and its sequencers follow it; other patches keep their own tempo. Where the lane is 0 (a section
without a curve, with the default `Default` of 0) the tempo returns to where it started, so a
ritardando at the end only needs a curve for the last section:

```go
arrangement.Tempo(arrange.Sections{"Outro": arrange.Smooth(120, 90)})
arrangement.Tempo(arrange.Sections{"Outro": form.Span("Outro").Last(4, arrange.Smooth(120, 90))}) // the last 4 bars
```

The tempo is updated 48 times per beat (changes smaller than
0.01 BPM are skipped), so a ritardando is smooth. The song position is counted in ticks, so parts
stay together; note lengths are converted to seconds when a note starts. The `pingpong` delay
follows the tempo too. Modules given a time in milliseconds computed from the tempo once (with
`timing.BPMToMilli`) don't.
`arrangement.Schedule()` returns the start time of every section following the tempo lane.

**Streams**: a part can also be played by generators instead of written notes. A `Stream` has a
`Rhythm` generator (the length of every step in beats, a negative step is a rest) and a `Message`
template (a genny `template.Template`): generators in it give every note new values and a slice
splits a note into a chord, like `templ`. `Amp` and `Legato` are a number or a generator.

```go
glass := &arrange.Stream{
	Rhythm: bucket.NewLoop(0.5, 0.5, 0.25, 0.25, -0.5, 1.0),
	Message: template.Template{
		"pitch":   wanderingPitch,                              // a genny.Generator[float64]
		"formant": template.NewParameter("formant", 2400.0),   // from the section
	},
	Amp: 0.25,
}
form.Stream(p, synth, "glass", func(s arrange.Section) *arrange.Stream {
	if s.Name == "Intro" {
		return nil // silent
	}
	return glass
})
```

The function is asked again at the start of every section (also when the song loops): return
`nil` for silence, the same stream as the section before to go on without a break, or another
stream to start it at the start of the section. Generators keep their own position (reset them to
start over). Streams play live, so a looping song is never played the same way twice, while
`psalm.Seed` still makes it repeat when the random choices come from psalm (`psalm.Rand()`, for
example in a `function.New` generator). The section dynamics scale the notes of streams too.

A stream drives any module, not only poly synths, with its `Output`:

| Output | Sends every step | For |
|---|---|---|
| `arrange.Notes` (default) | A trigger per note, with its amplitude and length | Poly synths |
| `arrange.Messages` | The messages of the template as they are (a slice splits them) | Any module taking messages: effects, filters, mixer strips, templ, players (add `"bang": true` for modules that play on a bang) |
| `arrange.Values` | The `"value"` of the template as a plain number | A value input: a filter cutoff, a mixer strip volume or pan |

`form.Stream` connects the streamer to value input 0 of a module; `form.NewStream(p, name, score)`
returns one that isn't connected, to connect to another input or to several modules:

```go
grains := &arrange.Stream{Output: arrange.Messages, Rhythm: sequence.NewLoop(4.0),
	Message: template.Template{"grainSize": bucket.NewLoop(60.0, 200.0, 350.0)}}
form.Stream(p, granulator, "grains", func(s arrange.Section) *arrange.Stream { return grains })

pan := form.NewStream(p, "pan", func(s arrange.Section) *arrange.Stream {
	return &arrange.Stream{Output: arrange.Values, Rhythm: sequence.NewLoop(1.0),
		Message: template.Template{"value": wanderingPan}}
})
mix.Strip("glass").ValueIn(pan, 0, 2) // the pan input of the strip
```

**Section parameters**: `Section.Params` are the parameters of a section, which makes sections
scenes. Streams put them in their templates: a `template.NewParameter("formant", 2400.0)` entry
takes the section's `"formant"` (and keeps its value until a section sets it again).
`arrangement.SendParams(modules...)` sends them as a message to modules at the start of every
section: `templ` modules take them as template parameters, synths and effects as settings (a
granulator takes `mix`, `freeze`, `octave`, ...).

```go
arrange.Section{Name: "Freeze", Bars: 8, Params: map[string]any{"formant": 2000.0, "mix": 0.7, "freeze": 1.0}}
arrangement.SendParams(gran)
```

**Playing in sections**: `Play(module, sections...)` makes a module sound only in those sections:
it is unmuted at their start and muted at the start of the others.

```go
arrangement.Play(mix.Strip("choir"), "Chorus", "Outro") // a mixer strip mutes with its 10 ms fade
arrangement.Play(drumKit, "Verse", "Chorus")            // other modules with SetMuted
```

Mixer strips (anything with `SetMute`) mute with their short fade, other modules with
`SetMuted`: a hard cut, while the module keeps running so its generators go on. A section name the
form doesn't have panics, so a typo shows when the song is built. A part that should not play at
all in a section is better written as a rest (a nil score or stream): it costs nothing and a
ringing sound finishes naturally.

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
