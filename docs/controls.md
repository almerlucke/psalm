# Controls

Control modules have no audio, they create and route values and messages at control (tick)
time. A typical sequencing chain is:

```
timer  --bang + tick-->  templ  --trigger message-->  synth (poly)
```

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
