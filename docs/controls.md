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

**It follows the clock of its patch.** A timer that misses control ticks (taken out of the patch for
a while, in a patch that isn't processed) catches up silently when the ticks reach it again: its
durations go by without bangs (bursts and random draws included), and it goes on where it would
have been, in time with the rest. So a part can be stopped by taking its timer out and come back on
the beat, instead of running muted all the time. A timer added while the patch plays starts where it
would be had it run from the start of the clock; `timer.New(...).FromNow()` starts it at its first
tick instead (its first bang there). A restart (`Psalm.Restart`, or the clock going back) starts it
again from its first duration.

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
position at any tick of the song as written and `form.Schedule()` the start time of every section.

**The transport**: `form.Transport()` is the song clock of the form. It says where the song is and
decides which section comes next, and that can change while the song plays: every part of the form
follows it (parts, streams, lanes, the tempo, events). Without changes it plays the sections in
order, exactly as before.

```go
tr := form.Transport()
tr.SetEnabled("Bridge", false) // switched off: skipped, until it is on again (or queued)
tr.Queue("Chorus")             // play the chorus next; Queue("") takes it back
tr.SetRepeat(true)             // the section playing again and again
tr.SetJump(arrange.AtBar)      // a queued section starts at the next bar, not at the end of the section
tr.Controls(ctl, "Song")       // all of it on a panel
```

A queued section starts from its first bar, on a bar line, so the song stays in time; going back
(or repeating) counts as a loop, so streams start again. A section queued before the song starts
is where it starts; one queued after a song that doesn't loop has ended starts it again at once.
`Playing()` returns the section playing and how far it is, `Order()` the sections a pass plays now;
`Duration()` and `Schedule()` of the arrangement leave the sections that are off out (a render
plays the song without them). All of it can be called from any goroutine.

**Parts on and off**: the parts of a form (`Part`, `Stream`) can be switched off and on while the
song plays, like the channels of a mixer but in time: a part that is off starts no new notes (what
sounds rings out) and keeps its place, so it comes back on the beat; it stays off through the
sections until it is switched on. `form.Parts()` lists them, `form.SetPartEnabled("bass", false)`,
`PartEnabled`, `PartPlaying` (it has notes in the section playing). `form.PartControls(ctl, name)`
puts them on a panel: a tile per part, tap it to switch the part (the parts playing in the section
light up, a part that is off says so); which parts are on is kept in presets. A playlist shows the
parts of the song playing: `list.PartControls(ctl, name)`. (`score.Sequencer.SetEnabled` and
`Streamer.SetEnabled` do it for one part.)

`tr.Controls(ctl, name)` adds a cue list to a panel (a tile per section: tap to play it next, the
dot turns it on or off, the section playing shows how far it is), a Repeat switch (`"<name>
repeat"`) and a Jump at choice (`"<name> jump"`). They are kept in presets. Every tile is a button
named `"<name>/<section>"` that a MIDI pad can play (MIDI learn on the page, or `BindNote`).

## panel

`modules/controls/panel`

Control a patch while it plays: faders, knobs, toggles, buttons and selects declared in code next
to the patch, bound to the settings of its modules, and meters showing levels. `Serve` makes a web
page of them: open it in a browser on the same computer, or on a phone or tablet on the network.

```go
ctl := panel.New(p, "Synth")

ctl.Section("Filter")
ctl.Fader("Cutoff", 200, 12000, panel.Exp, panel.Message(filter, "cutoff"), panel.Default(2400), panel.Unit("Hz"), panel.Smooth(0.05))
ctl.Knob("Resonance", 0, 0.95, panel.Linear, panel.Message(filter, "resonance"))

ctl.Section("Play")
ctl.Fader("Tempo", 60, 160, panel.Linear, panel.Func(func(v float64) { p.Env().SetBPM(v) }), panel.Default(96), panel.Step(1))
ctl.Toggle("Sustain", panel.Message(piano, "sustain"))
ctl.Button("Bells", bells.Bang)
ctl.Select("Preset", []string{"warm", "bright"}, func(i int) { /* switch */ })
ctl.Meter("Master", mix.Master())

go ctl.Serve("localhost:8087") // ":8087" to reach it from other devices
_ = p.Listen()
```

| Control | |
|---|---|
| `Fader(name, min, max, curve, target, options...)` | A vertical fader. `panel.Linear` moves in even steps (levels in dB, amounts), `panel.Exp` in even steps per octave (frequencies, times; the range above 0) |
| `Knob(...)` | The same as a knob, dragged up and down |
| `Toggle(name, target)` | On (1) and off (0): a pedal, a mute, a freeze |
| `Button(name, func())` | Calls the function every time it is pressed: start a sample, go to the next scene |
| `Select(name, options, func(i int))` | A choice between options |
| `Meter(name, module)` | The peak and RMS level of the output of a module: a synth, a mixer strip, the master |
| `Section(name)` | Starts a group: the controls after it are shown together |

**Grids**: `ctl.Grid(name, rows, steps, patterns, fields...)` adds a grid: rows of steps in
patterns, every cell holding the values of the fields (`panel.Field{Name, Min, Max, Default, Step,
Unit, Options}`; the first field is the level, 0 is off). The page draws it as a step grid: tap a
cell to switch it, drag to paint, right click (long press, shift click) to edit all its fields,
tap a row name for its tools (clear, shift, Euclid, random), pattern tabs on top with Copy (the
pattern on view to another) and Clear. `grid.Value`, `grid.Set`, `grid.Clear`, `grid.Copy(from,
to)`, `grid.SetPattern` work from any goroutine;
whatever plays the grid tells the page `SetPlaying`, `SetPlayhead` and `SetLength` (per row). Grids
are in snapshots (as `name/pattern/row/step/field`, the cells away from their defaults). The step
sequencer below plays one.

**Cue lists**: `ctl.Cues(name, cuer)` adds a row of tiles for anything with named cues, one
playing and one asked for next (a `panel.Cuer`: `CueNames`, `CueState`, `QueueCue`, `EnableCue`).
Tap a tile to ask for it next, tap its dot to turn it on or off. Which cues are on is in snapshots
(as `name/cue`, 1 or 0). Every tile is a button too, named `"<list>/<cue>"`: tap a tile in MIDI
learn mode and hit a pad, or `ctl.Param("Song/Chorus").BindNote(0, 36)` (`cues.Tile(i)` is the
same button). The song transport of `arrange` is one (`Transport.Controls`). With
`panel.CueSlots(n)` the cues can change while it plays (the sections of the song a playlist plays):
the page redraws the tiles when they change, the buttons are named by position (`"Sections/1"` to
`"Sections/<n>"`) and which cues are on is not in snapshots. With `panel.CueToggles()` the cues
are switches (the parts of a song): a tap switches a cue on or off instead of asking for it, and the
cues sounding now (`CueState.Active`) light up.

**Mounts**: `ctl.Mount(name, mounter)` shows another panel inside this one: the one a
`panel.Mounter` gives now (`Mounted() *Panel`), and another one when it gives another, like the
panel of the song a playlist plays (`playlist.SongControls`). The page shows it as on its own page
(it is served under `/mount/<name>/`), the control changes and notes of MIDI reach it, and MIDI
learn on the page binds its controls. A mounter that is a `panel.MountKeeper` (`SnapshotMount`,
`RestoreMount`) keeps values in this panel's snapshots and presets too, as `"<mount>/..."`: the
playlist keeps the controls of every song this way. A mounter that is a `panel.MountBinder`
(`BindingsMount`, `SetBindingsMount`) keeps MIDI bindings the same way: they are in `Bindings()` as
`"<mount>/..."`, so `KeepBindings` saves and loads what was learned on every song's controls.

**Labels**: `panel.Label("M")` shows a short text on the page instead of the name (names have to
be unique, labels can repeat).

**Targets**: `panel.Message(module, key)` sends the value as a message (`"cutoff"`, `"volume"`,
`"send.hall"`, `"sustain"`, any setting a module takes by name), `panel.Input(module, index)` to a
value input, `panel.Func(f)` calls a function, `panel.Targets(...)` sends to several.

**Options**: `Default(v)` (the start value, sent when the patch starts, and where a double click
returns; the minimum otherwise), `Unit("Hz")`, `Step(1)` (round to steps), `Smooth(seconds)`
(glide to a new value instead of jumping, evenly along the travel: through the octaves on an
exponential fader).

**Threads**: the panel is a control module in the patch. Changes (from the page, or from your own
code on any goroutine with `ctl.Param("Cutoff").Set(v)` and `Press()`) are applied on the audio
thread at the next control tick, and the audio thread never waits for them; button functions and
select functions run there too. Declare the controls before the patch plays.

**MIDI**: `ctl.MIDI(kb)` connects a keyboard (from `keyboard`): the control changes (CC) of the
controller move the controls bound to them. Bind in code, `ctl.Param("Cutoff").BindCC(0, 74)`
(channel 1-16, 0 for any), or learn: press **MIDI learn** on the page, tap a control and move a
knob or fader on the controller (`Param.Learn()` from code). A controller moves one control: binding
it again takes it from the control that had it. A fader or knob follows the controller over its
travel (by its curve), a toggle is on from 64 up, a button is pressed when the controller goes up
past 64, a select picks an option over the travel. `panel.Pickup()` makes a fader or knob wait
until the controller reaches its value (soft takeover), so it doesn't jump after it was moved on the
page. Buttons and toggles take notes too, for pads that send notes: `BindNote(channel, note)`, or
MIDI learn with a pad (a note on presses a button, switches a toggle over; `ctl.NoteOn(channel,
note, velocity)` takes notes from any other source). For a keyboard whose keys play a synth,
`ctl.MIDIControllers(kb)` takes only its knobs and faders: a key then never presses a button or
gets learned. Binding a controller or note takes it from
the control that had it, also when one of the two listens on any channel. `Bindings`,
`SetBindings`, `SaveBindings(path)` and `LoadBindings(path)` keep the mapping of a controller;
`ctl.ControlChange(channel, controller, value)` takes control changes from any other MIDI source.

**Snapshots**: `ctl.Snapshot()` returns every value by name, `Restore(values)` sets them,
`Save(path)` and `Load(path)` keep them in a JSON file: presets of the patch.

**Start and stop**: a panel made for a patch that plays through a sound card has a Start/Stop
button in the page's header (the space bar does the same). Stopping stands the whole patch still
where it is (the song, the sequencers, a ringing note) and starting goes on from there; the meters
show silence while it stands. Changes made while it is stopped are applied when it plays again.
**Restart** next to it goes back to the start: the song, the parts, the streams and the step
sequencers play from the beginning (a section asked for next is forgotten, the conditions of the
steps count from the start), sounds that ring go on ringing. Pressed while stopped, the patch
starts from the beginning when it plays again. `ctl.SetPlayer(pl)` gives the buttons to anything
with `AudioStart`, `AudioStop` and `Running` (and `Restart` for the restart button; `nil` takes them
away). `psalm.Psalm`'s `AudioStart` and `AudioStop` can be called from any goroutine and do nothing
when it already plays or stands still; `Running()` tells which.

`p.Restart()` (from any goroutine) sets the clock of the patch back to 0 at the next buffer and
calls `Restart()` on every module that keeps time of its own: a `psalm.Restarter`, also in patches
within the patch. The arrangement, the parts (`score.Sequencer`), the streams, the step sequencer,
`dispatch`, the `timer` (its durations from the first, `steps` too), `templ` (its generators from
their start: sequences start again, random ones go on drawing) and the `lfo` (its starting phase)
are; a module of your own that counts time can be one too.

**Presets**: `ctl.Presets("presets")` gives the panel a folder for setups (made when it isn't
there), a JSON file each. The page then has a presets bar: type a name and Save, pick one and Load
or Delete. From code: `SavePreset(name)`, `LoadPreset(name)`, `DeletePreset(name)`,
`PresetNames()`. A name is letters, digits, spaces, `-`, `_` and `.` (up to 64), so it stays in
the folder. Download and Upload on the page keep a setup in a file of your own, with or without
the folder.

**Tests of the page**: `modules/controls/panel/paneltest` drives pages in headless Chrome (the
DevTools protocol on a pipe, no other dependencies): `Start`, `Open` a page and wait for it,
`Eval`, `WaitFor`, `Tap`, `Key`, `Mouse` (press, move, release, the wheel), `Center` of an element,
`Screenshot`, `Serve` a panel for a test, `Eventually` for the Go side. The panel's tests use it for
every kind of control, the tiles of cue lists, mounts, the grid, MIDI learn, Start/Stop and Restart,
presets, themes and plugins; the plugins' for their own pages: the widgets (the keyboard letting go
whatever happens to the pointer, the XY pad, pads, envelope, scope), the editor (every kind of
control, presets saved and loaded on the page), the DX editor (the algorithm, one of your own, the
bank), the piano roll. Any error a page throws fails its test. They run with `go test` when Chrome
is installed (`PSALM_CHROME` names another browser), and are skipped with `-short`. A plugin of
your own tests its page the same way:

```go
func TestMain(m *testing.M) { paneltest.Main(m) }

func TestPage(t *testing.T) {
	b := paneltest.Start(t)
	tb := b.Open(t, paneltest.Serve(t, ctl.Handler()), `document.querySelector(".xy")`)
	x, y := tb.Center(`document.querySelector(".xy")`, 0.5)
	tb.Mouse("mousePressed", x, y, nil)
	paneltest.Eventually(t, "the pad pressed", func() bool { return pad.Value() > 0 })
}
```

**Themes**: the look of the page is a set of CSS variables, and a `Theme` gives them values, for
light and for dark mode, plus CSS of its own. The page has a choice of the themes in
`panel.Themes` (`default`, `midnight`, `paper`, `amber`, `contrast`, `neon`) and of light, dark or
the system's mode, remembered in the browser; a mounted panel follows the page it is in.
`ctl.SetTheme(t)` gives a panel its own theme, the one its page starts with:

```go
ctl.SetTheme(panel.Theme{
	Name:  "studio",
	Light: map[string]string{"accent": "#e0662b", "radius": "4px", "font": "Inter, sans-serif"},
	Dark:  map[string]string{"bg": "#101010", "card": "#1a1a1a", "accent": "#ff8a3d"},
	CSS:   `section h2 { letter-spacing: .2em; }`,
})
```

| Variable | |
|---|---|
| `bg`, `card`, `ink`, `muted`, `line` | The page, the cards of the sections, the text, quiet text, borders |
| `track`, `accent`, `accent-ink` | The travel of faders and knobs, what is on, the text on it |
| `meter-low`, `meter-mid`, `meter-high` | The meters (and the status light) |
| `shadow`, `radius` | The shadow and the corners of the cards (buttons get two thirds of the radius) |
| `font`, `fader-height` | The font, and the height of faders and meters |

A theme without `Dark` looks the same in both modes. Its CSS can change anything: every control is
in an element with `data-kind` and `data-name` (`[data-name="Cutoff"] .fader { ... }`).

**Plugins** draw controls of their own: a `Plugin` is a script, styles and files, and `Custom`
adds a control of a kind its script draws. The script registers a builder for the kind with
`psalm.register(kind, (p, api) => ...)`, getting the control as the page sees it and returning its
element and how it changes:

```js
psalm.register("xy", (p, api) => {
	const pad = document.createElement("div");
	pad.className = "xy";
	pad.addEventListener("pointermove", e => api.message(p.name, {x: e.offsetX, y: e.offsetY}));
	return {
		el: api.labelled(p, pad),      // under its name, as the page's own controls are
		show(value) {},                // its value changed (here, elsewhere, from code)
		update(state) {},              // its state changed (Push, Live)
		restyle() {},                  // the theme changed (for a canvas: read api.css again)
		bindable: true,                // MIDI learn can bind its value
	};
});
```

```go
//go:embed xy.js
var xyJS string

ctl.Use(panel.Plugin{Name: "xy", JS: xyJS, CSS: ".xy { width: 200px; height: 200px; }"})
ctl.Custom("xy", "Pad", panel.Message(filter, "cutoff"), panel.Range(200, 8000, panel.Exp),
	panel.Data(map[string]any{"grid": 8}),                  // sent once: p.data
	panel.Live(func() any { return lfo.Phase() }),          // asked for 20 times a second: update(state)
	panel.OnMessage(func(m json.RawMessage) { /* ... */ }), // api.message, on the audio thread
)
```

A custom control has a value like a fader (0 to 1, or its `Range`) bound to a target, kept in
snapshots and presets, set with `api.send(name, value)`. `Data` is anything it needs to draw it,
`Live` (asked for on the page's goroutine) or `Push` (from any goroutine) give it a state that
changes, `OnMessage` gets what it sends back with `api.message`. The `api` has the page's own
helpers too: `labelled`, `drag`, `position`, `valueAt`, `format`, `press`, `post` and `css(name)`
for a variable of the theme. A plugin's script is served at `plugins/<name>/plugin.js`, its styles
at `plugin.css` (before the theme's, so a theme can restyle it) and its `Files` under
`plugins/<name>/`. A control of a kind no plugin draws shows as a note on the page instead.

A custom control can have **parts**: values of their own that its plugin draws, not shown by
themselves (the y of an XY pad, the decay of an envelope). `par.Part(name, min, max, curve, target,
options...)` adds one; it is a control like any other (a target, presets, `api.send(name, value)`),
the builder gets them in `p.parts` and its `part(name, value)` is called when one changes.
`panel.OnBuffer(f)` calls `f` on the audio thread at the start of every buffer (a scope reading a
module's output, as the meters do). `panel.NoDefault()`, for any control, keeps its default from
the target when the patch starts: the target keeps its own setting until the control moves.

**Widgets** (`modules/controls/panel/widgets`) are plugins to play and shape synths with, all
following the theme:

| Widget | |
|---|---|
| `XY(ctl, name, x, y Axis)` | A pad moving two values at once, x across and y up (a cutoff and a resonance); arrow keys move it, a double click goes back |
| `Keys(ctl, name, play, From(48), Octaves(3))` | A keyboard: lower on a key is louder, sliding plays the keys passed; the computer's keys play the first one on a page (A W S E D F T G Y H U J K, Z and X an octave down and up). `Play(synth)` plays a synth; or pass a function of your own (the keyboard example pushes MIDI) |
| `Pads(ctl, name, Hit(kit, "kick", "snare", ...), columns)` | Pads, louder in the middle; the number keys hit the first nine |
| `Envelope(ctl, name, synth, "amp", ADSR{...})` | An attack, decay, sustain and release to drag into shape, sent as the envelope messages every synth takes ([envelopes by message](synths.md#envelopes-by-message)) |
| `Scope(ctl, name, module)` | The sound of a module: its wave (steady on its rising zero crossings) and its spectrum; a click switches between them |

```go
ctl.Section("Synth")
widgets.Keys(ctl, "Keys", widgets.Play(lead), widgets.From(48), widgets.Octaves(3))
widgets.XY(ctl, "Filter",
	widgets.Axis{Name: "cutoff", Min: 200, Max: 8000, Curve: panel.Exp, To: panel.Message(lead, "cutoff")},
	widgets.Axis{Name: "resonance", Max: 0.9, To: panel.Message(lead, "resonance")})
widgets.Envelope(ctl, "Amp", lead, "amp", widgets.ADSR{Attack: 5, Decay: 200, Sustain: 0.7, Release: 300})
widgets.Pads(ctl, "Drums", widgets.Hit(kit, "kick", "snare", "hat", "clap"))
widgets.Scope(ctl, "Out", mix.Master())
```

`cmd/examples/widgets` has them all, on nine synths to switch between.

**Editors of every synth and effect** (`modules/controls/panel/editor`): every synth and effect
describes its settings (`params`: the message keys it takes, their ranges, units, groups and where
they are now, and its envelopes): `subtractive.Describe(cfg)`, `chorus.Describe(ch.Settings())`.
`editor.Add` draws them as cards by group: a knob for every value, a switch, a choice, and the
envelopes to drag (the widgets' envelope; knobs for one with fewer stages). Every setting is a value
of the panel named `"<name> <key>"`: presets keep them, MIDI learn binds them (learn mode, tap a
knob), code sets them (`ed.Setting("cutoff").Set(2000)`). `ed.Load(set)` moves them all to another
description: a preset.

```go
lead := subtractive.NewPoly(subtractive.LeadPreset(), 12).AddTo(p)
ed := editor.Add(ctl, "Lead", lead, subtractive.Describe(subtractive.LeadPreset()))
ctl.Select("preset", names, func(i int) { ed.Load(subtractive.Describe(subtractive.Presets[names[i]]())) })
ch := chorus.New(chorus.DefaultConfig()).AddTo(p)
editor.Add(ctl, "Chorus", ch, chorus.Describe(ch.Settings()))
```

**Presets of a synth.** Above the controls the editor has its presets: `ed.Factory(names, describe)`
gives it the synth's own (choosing one moves every control to it), `ed.Keep(dir)` keeps yours in a
folder, a JSON file each (the settings by message key): type a name and Save, pick one of yours to
load it back, Delete it. From code: `ed.Presets().Save(name)`, `Load(name, mine)`, `Delete(name)`,
`Mine()`. The DX editor keeps patches of its own the same way (`Keep`; an algorithm of your own is
kept with them), and `editor.NewStore` gives a plugin of your own the same store.

```go
ed := editor.Add(ctl, "Lead", lead, subtractive.Describe(subtractive.LeadPreset()))
ed.Factory(slices.Sorted(maps.Keys(subtractive.Presets)), func(n string) params.Set {
	return subtractive.Describe(subtractive.Presets[n]())
})
ed.Keep("presets/subtractive")
```

The settings start where the description has them and are sent there when the patch starts;
`panel.NoDefault()` keeps the synth's own until one moves. `params/paramstest` checks a description:
every key a message the module takes, every value landing where it says. `cmd/examples/editors` has
them all: every synth with its presets, every effect, the one chosen of each on its own page
(mounted).

**The DX editor** (`modules/controls/panel/dxeditor`) edits the six operator FM synth on the page,
as a DX7 with a screen: the algorithm as a diagram (click through the 32, or "my own": the carriers
by clicking the operators, the modulations a higher operator makes into a lower one in a grid, the
feedback loop), each operator in a row (its frequency as a ratio or fixed, coarse and fine, detune,
level, velocity sensitivity, rate scaling, amplitude modulation, its envelope drawn; open it for
the rates and levels and the keyboard scaling), the LFO and the pitch envelope, and a bank of
patches to load from. Drag a number up or down (Shift slower), turn it with the wheel, double click
for its default. Every setting is a control of the panel named as the dx messages are ("DX
op3Coarse"), so presets keep them, MIDI learn binds them, code sets them.

```go
synth := dx.NewPoly(patch, 16, 2).AddTo(p)
banks, _ := dx.LoadSysEx("assets/dx7/rom1a.syx")
dxeditor.Add(ctl, "DX", synth, patch, dxeditor.Bank(banks))
```

`modules/controls/panel/pianoroll` is a plugin to start from: the notes of a MIDI file scrolling
past a playhead as a `midi.Player` plays them (coloured by channel, the muted ones dimmed); a click
or drag moves the player there, the wheel zooms. `pianoroll.Add(ctl, "Notes", player)`.

**The page** follows the system's light or dark mode (or the one chosen on it) and works with touch (several faders at once
on a tablet). Drag a fader or knob (Shift for fine steps on a knob), use the arrow keys on a
focused fader, double click to return to the default. Every open page shows the same values.
`ctl.Handler()` gives the page and its JSON API as an `http.Handler` to serve yourself:

| Route | |
|---|---|
| `GET /` | The page |
| `GET /api/panel` | The controls: kinds, ranges, options and values |
| `POST /api/set` | `{"name": "Cutoff", "value": 2400}` sets a control |
| `POST /api/press` | `{"name": "Bells"}` presses a button |
| `GET /api/events` | The values and meters as they change (server sent events, 20 times a second) |
| `GET /api/snapshot`, `POST /api/snapshot` | All values, and restoring them (`?download` to save it as a file) |
| `POST /api/message` | `{"name": "Pad", "data": {...}}` sends anything to a custom control's `OnMessage` |
| `GET /theme.css` | The stylesheet of a theme (`?name=amber`; the panel's own without) |
| `GET /plugins/<name>/...` | A plugin's `plugin.js`, `plugin.css` and files |
| `GET /api/presets` | `{"enabled": true, "names": [...]}`: the saved presets |
| `POST /api/presets` | `{"action": "save", "name": "Groove"}` (or `"load"`, `"delete"`), answers with the names |
| `POST /api/learn` | `{"name": "Cutoff"}` arms a control for MIDI learn, `{"name": ""}` stops |
| `POST /api/unbind` | `{"name": "Cutoff"}` takes its controller from a control |
| `POST /api/audio` | `{"running": false}` stops the sound of the patch, `true` starts it again, `{"restart": true}` goes back to the start |
| `/mount/<name>/...` | The page and API of the panel mounted at a mount (`/mount/Song/api/set`, ...) |
| `POST /api/cues` | `{"name": "Song", "queue": 2}` asks for a cue next (-1 takes it back), `{"name": "Song", "cue": 3, "on": false}` switches one off |
| `POST /api/grid` | `{"name": "Beat", "pattern": 1, "set": [[pattern, row, step, field, value], ...]}` sets cells of a grid and/or asks for a pattern |

The server is plain `net/http`: there is no login, so serve it on `localhost` unless you trust the
network it is on. `ctl.Close()` stops it (the pages lose their connection, the address is free
again): `defer ctl.Close()` after starting `Serve`. When the address is taken, `Serve` says so,
most likely an earlier run that is still going.

`p.Listen()` plays until Ctrl-C (or a SIGTERM), and also stops when the program was started by
`go run` and `go run` goes away (stopped from an editor, or killed): otherwise the program it
started would play on, holding the sound card and the panel's port.

## stepseq

`modules/controls/stepseq`

A step sequencer playing a grid of a panel: rows of steps (a row per sound), every step with its
own velocity, chance, roll, nudge, pitch and condition, in patterns, edited on the panel's page
while it plays.

```go
ctl := panel.New(p, "Drums")
grid := stepseq.NewGrid(ctl, "Beat", []string{"kick", "snare", "hat", "open"}, 16, 4)
seq := stepseq.New(p, grid, stepseq.Config{StepsPerBeat: 4, Swing: 0.56, Humanize: 4}).
	Row("kick", stepseq.Sound(kit, "kick")).
	Row("snare", stepseq.Sound(kit, "snare")).
	Row("hat", stepseq.Sound(kit, "hat"), stepseq.Choke(1)).
	Row("open", stepseq.Sound(kit, "open"), stepseq.Choke(1))
seq.Controls(ctl) // swing, humanize, fill; level, length, mute and solo per row
go ctl.Serve("localhost:8087")
```

**A step** (the fields of the grid, `stepseq.Velocity` ... `stepseq.Condition` in code):

| Field | |
|---|---|
| velocity | 0 is off |
| chance | The chance it plays, 0-1: random skips |
| roll | Hits the step is split into, 1-8 (a ratchet) |
| roll chance | The chance the roll happens, else a single hit |
| roll ramp | The velocities of the roll: 1 rises from nothing, -1 falls away, 0 keeps them |
| nudge | Early (-0.5 step) or late (0.5 step) |
| pitch | Semitones: the speed of a sample, the pitch of a note |
| condition | When it plays: `always`; `A:B`, the Ath of every B times the row comes round (`1:2`, `2:4` ...); `fill` and `not fill` (the fill switch); `first` and `not first` (the first time since the pattern started) |

**A row** plays `stepseq.Sound(drums, "kick")` (a sound of a drum synth, `drums.NewDrums`) or
`stepseq.Note(synth, pitch, beats)` (a note of any synth). Options: `stepseq.Choke(group)` (a hit
stops the last hit of the other rows of the group: an open hihat cut by the closed one) and
`stepseq.Length(steps)` (rows of other lengths loop apart: polymeter).

**The sequencer** counts steps from the patch's clock (`StepsPerBeat` 4 is 16ths, 3 triplets),
so it follows the tempo and stays with the parts and the arrangement. It works a step ahead (a
step nudged early is there in time) and plays every hit at its sample. Swing (0.5 straight to
0.75) moves every second step, humanizing moves hits by ± milliseconds and their velocity by ± a
part. A new pattern starts when the pattern playing comes round; the rows and the conditions start
counting again. Settings, safe from any goroutine: `SetSwing`, `SetHumanize`, `SetFill`, and per
row `SetLevel` (dB), `SetMute`, `SetSolo`, `SetLength`; mute, solo and the fill count right away
(a hit checks them when it plays). `seq.Controls(ctl)` adds them all to the panel: a card for the
feel and a card per row.

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
	dispatch.NewSetting(reverb, bar*16, psalm.Message{"mix": 0.6}),
	dispatch.NewEvent(bells, bar*24, dispatch.Message, psalm.Message{"bang": true}),
).AddTo(p)
```

| Event type | Effect on `Who` at tick `When` |
|---|---|
| `dispatch.Mute` | `SetMuted(true)` |
| `dispatch.Unmute` | `SetMuted(false)` |
| `dispatch.Message` | `ReceiveMessage(What)` |

Events are played once, sorted by time (events at the same tick in the order given). More can be
added with `AddEvents`.

**It follows the clock of its patch.** Mutes, unmutes and settings (`NewSetting`, or an event with
`State` set) are states; other messages are moments (a note, a bang). A dispatch that misses
control ticks (out of the patch for a while) doesn't fire the events it missed all at once when the
ticks reach it again: the states are set as they should be by now (only the latest mute of every
module, its settings merged, later keys over earlier ones), the moments are left out. A dispatch
added while the patch plays does the same with the events before its first tick. A restart (or the
clock going back) dispatches from the start again.

## logger

`modules/controls/logger`

`logger.New(slogLogger)` logs every value it receives (with its input index), handy to inspect
what flows through value connections.
