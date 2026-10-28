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
lead := dx.NewPoly(dx.FlutePreset(), 2, psalm.Stereo).AddTo(p)
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
whatever plays the grid tells the page `SetPlaying`, `SetPlayhead` and `SetLength` (per row). A pattern can have fewer steps than the grid
(`SetPatternSteps`, `DefaultSteps`; `SetActive` takes a pattern out of the rotation `Next` goes through, with the page's "In rotation" box; the page has a Steps box, a snapshot keeps them as
"Grid/B/steps", the sequencer comes round and switches patterns there). Grids
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
`ctl.Mount(...).Hide()` keeps a mount off the page while it is still served and reached by MIDI,
for a plugin to show it where it wants: the patcher shows a module's settings in a window of its own, inside the page, so; a mounter that is an `ArgMounter` also serves a panel by an argument, `mount/<name>@<arg>/`.

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
where it is (the song, the sequencers, a reverb's tail) and starting goes on from there, the notes
that were playing let go (they ring out their release, not the rest of their length); the meters
show silence while it stands. Changes made while it is stopped are applied when it plays again.
**Restart** next to it goes back to the start: the song, the parts, the streams and the step
sequencers play from the beginning (a section asked for next is forgotten, the conditions of the
steps count from the start), the notes playing are let go and ring out their release, a reverb
or a delay its tail. A synth lets go of its notes by `AllNotesOff()` (`psalm.NoteReleaser`: the
poly synths, the organ and the piano have it). Pressed while stopped, the patch
starts from the beginning when it plays again. `ctl.SetPlayer(pl)` gives the buttons to anything
with `AudioStart`, `AudioStop` and `Running` (and `Restart` for the restart button; `nil` takes them
away). `psalm.Psalm`'s `AudioStart` and `AudioStop` can be called from any goroutine and do nothing
when it already plays or stands still; `Running()` tells which.

**The bar at the top**: `.Top()` on a control puts it in the page's header, beside Start/Stop and
Restart, small: a fader or a knob as a short slider with its value (a tap on the value types one,
Enter takes it, Esc doesn't), a meter as a bar; a plugin's control is told to draw itself small (the
keys widget becomes a button showing them in a sidebar, the scope a strip of the wave). A section
left with nothing else isn't shown.

```go
ctl.Fader("Volume", -40, 6, panel.Linear, panel.Message(master, "volume"), panel.Unit("dB")).Top()
ctl.Meter("Out", master).Top()
widgets.Scope(ctl, "Scope", master).Top()
```

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
`Screenshot`, `Serve` a panel for a test, `Eventually` for the Go side, and an audio `Engine` that
asks for buffers on a goroutine of its own, so a patch plays while a test drives its page. The panel's tests use it for
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
for a variable of the theme; `theme()` is the theme's name, and `frame(iframe)` gives a frame of
the plugin showing a panel (`mount/<name>/?embed`) its height, the theme, light or dark and learn
mode, as the page's own frames. A plugin's script is served at `plugins/<name>/plugin.js`, its styles
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
| `Keys(ctl, name, play, From(48), Octaves(3))` | A keyboard: lower on a key is louder, sliding plays the keys passed; the computer's keys play the first one on a page (A W S E D F T G Y H U J K, Z and X an octave down and up). `Play(synth)` plays a synth; or pass a function of your own (the keyboard example pushes MIDI). Its Sidebar button puts it upright in a sidebar on the right of the window (the page makes room, a patch over the whole screen too), where ⇥ hides it to a tab on the edge and the tab brings it back, × puts it back on the page; the browser keeps where it is. At the top of the page (`.Top()`) it is a button that shows and hides it in the sidebar (hidden at first) |
| `Pads(ctl, name, Hit(kit, "kick", "snare", ...), columns)` | Pads, louder in the middle; the number keys hit the first nine |
| `Envelope(ctl, name, synth, "amp", ADSR{...})` | An attack, decay, sustain and release to drag into shape, sent as the envelope messages every synth takes ([envelopes by message](synths.md#envelopes-by-message)) |
| `Scope(ctl, name, module)` | The sound of a module: its wave (steady on its rising zero crossings) and its spectrum; a click switches between them. At the top of the page (`.Top()`): a strip of the wave |
| `Readout(ctl, name, f)` | A line of text the program writes, asked from `f` on every page update. With `.Top()` it is small, beside the volume: `widgets.Readout(ctl, "Time", pt.SongClock).Top()` shows the time of the song of a patcher patch, "1:23 / 4:00" (the time in the song and the length of a pass) |

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

`Factory` only moves the controls, and a preset is more than its description: a waveguide string's model
(pluck, bow, membrane, pipe), a modal synth's mode list, a vocal's word and the matrix slots are not
settings the editor has controls for, so choosing a bowed preset while the page started on a plucked one
leaves the synth plucked. `ed.FactoryApply(names, describe, apply)` does better: choosing a preset
places the controls at the described values without sending them (`Param.Place`), then calls
`apply(name)`. With a `poly.Swap` that builds the synth at the preset off the audio thread, `apply` swaps
the new voices into the live synth between two buffers (`Poly.AdoptVoices`), so the sound is exactly
the synth made at that preset. Notes sounding fade out, and mono and legato stay as set.
`Editor.Base()` is the preset chosen last. The patcher does this for every polyphonic synth in its
catalog (`Made.Rebuild`) and records the chosen preset on the node, so a saved patch keeps it. Presets of
your own still go through `Apply` (their settings on the synth as it is). The organ and the piano are
single modules with no voices, so they take over the whole inner state of an instrument built from the
preset (`Organ.Adopt`: the tonewheels, scanner and Leslie; `Piano.Adopt`: the string bank, bridge,
soundboard convolution and pedals). `poly.Swap` takes any `poly.Adopter` (`Adopt(other) bool`): the new
instrument is built off the audio thread and swapped in between two buffers through `Psalm.Do`, without
allocating; what was sounding fades out over one buffer, and held keys and pedals come back up (a
pedal held on the keyboard has to be pressed again). The instrument handed to `Adopt` shares its
internals with the adopter afterwards and must not be used again. In code:

```go
swap := poly.NewSwap(p.Do, synth, func(name string) psalm.Modular { return waveguide.NewPoly(waveguide.Presets[name](), 12) })
ed.FactoryApply(names, describe, swap.Load)
```

A test in the patcher catalog (`TestFactoryPresetsAreThePresetsMade`; `AUDIT=1` runs every preset of
every synth) chooses presets through the editor's own load path and compares the sound with the synth
made at that preset.

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

`dxeditor.SysExFolder(dir)` turns on a SysEx browser. The editor lists the `.syx` files of the folder in a
"sysex" row (the refresh button looks again). Choosing a file makes its voices the program list,
whether it is a 32 voice bulk dump or single dumps; "built in" returns to the bank given with `Bank`.
"save patch" writes the patch as it is (with its exact operator values) as a single voice dump, "save
bank" the bank shown as a 32 voice dump (the first 32 voices). Both use the name in the box (letters,
digits, spaces and `- _ . ( )`; the editor adds `.syx` and never accepts a path). A file that exists is
only replaced after the page asks you to confirm. In the patcher set `catalog.Config.SysEx` to a
folder; the `dxedit` example has `-sysex` (default `dx-sysex`). The folder is made on the first save.

**The patcher** (`modules/controls/panel/patcher`) makes a modular patch on the page. Add modules
from a palette, move them by their heads, and draw cables from an output to an input of the same
kind: audio (the sound), notes (a keyboard to the synths it plays) and modulation (an LFO, a random
source or the mod wheel to any setting of a module, over a part of its range: drawn to a module's
mod input, the setting and the range are chosen below the patch). A tap on a module chooses it, a double
click opens a window for it, inside the page (a frame over the patch, not a browser window, so it has no address bar): its script (a box of lines) and its settings
(its editor, with its presets and yours), what is wrong with what is typed below the boxes. For a plugin that has a window of its own that one opens, and with Alt held a
double click opens ours. A window has a title to move it by, a corner to size it by and an × (Esc, in a box or on the page, closes the one in front; closing takes what
is typed); there is one for each module and they stay open together, a double click on a module that has one brings it to the front. A window keeps its size and place
(in the browser's local storage, per kind of module: `psalm.patcher.window.<patcher>.<kind>`) and the next window of that kind opens there. What is in a window is
`plugins/patcher/module.html` in a frame, which the page tells its module as it changes. The page of one module is served at `mount/<name>@<id>/` (a mounter that gives a panel by an argument: `panel.ArgMounter`). Click a cable to take it
away (or Delete); Delete (or Backspace) takes the modules chosen away with their cables. Patches are kept by name in a folder: the modules, where they are, the
cables and every setting. Every change is made between two buffers while it plays.

Several modules move together: shift-click (or ctrl, cmd) adds a module to the ones chosen or takes
it out, a drag over the empty patch draws a rectangle that chooses every module it touches (with
shift, adding to them); dragging one of them moves them all. Delete takes the chosen away (asking
first when there are several), Esc lets go of them. Ctrl+C (cmd on a Mac) copies them with their
settings, scripts and the cables between them, Ctrl+V pastes them a little further each time (chosen
then, to drag them where they go), Ctrl+D duplicates, Ctrl+X cuts; the copy is the patcher's, so it
outlives the modules copied. From code: `Copy(ids...)`, `Paste()`. A cable is taken hold of a little beside its
line too. **Log** opens the log of the patch over the patch: a line for everything the scripts `log()`, each starting with the time and the module (`14:03:22.510  script m3  now: Creation`), the newest below, with **Clear log**; the button counts the lines. An error of a module (a script that doesn't compile, a text that can't be read) is a toast in the corner of the page, with the module's name, gone after twelve seconds or when tapped, and the module gets a red edge while it has the error (no text is on a module: a status line, such as what a sampler made of its sounds, is in its window); the same error is shown once. **Full screen** puts the patch over the whole screen (Esc, or the button again, brings it
back). In its window a script has room to work: Ctrl+Enter (or Run) runs it, its error or what
it logs is below it (and on its module), what was typed is taken when it closes; the window's corner
makes it bigger or smaller.

```go
kinds := catalog.Kinds(catalog.Config{Input: keys, IR: hall}) // every synth and effect, a keyboard
pt := patcher.New(p, ctl, "Patch", kinds, patcher.Keep("patches"), patcher.Presets("editors-presets"))
mix.Channel("patch", pt.Out(), mixer.StripConfig{})
```

**The sampler and the grains** are modules of the patch too, with their sounds as a text in the window. The sampler's text has a line for every
sound: `36 assets/sounds/various/tabla_loop.wav hits 9 0.08` (the first key, the file, how it is cut: `hits` rise-in-dB and least gap, `change`
for a sung line, `pauses` depth, least pause and longest piece for speech, `even n`, `notes from to pitch ...` (the played notes of a recording, in seconds and MIDI pitch, five keys apart), `whole hiKey root`, or `sfz` for an SFZ instrument; `vol=`, `pan=` and `name=` after it, and `loop=1.5` to make a sound go round while its key is down, the end fading into the start over 1.5 seconds: the sea, an ambience); the pieces of a sound are on the
keys from the first, and a script plays them with `note(key, velocity, steps)` (see [Synths](synths.md#slices)). A text that can't be read
(a missing file, a mode it doesn't know) leaves the sounds as they were. The grains synth's text has a line for each sound: `name file pitch`;
the first plays, and a script picks another with `set({sound: "name"})`. `cmd/songs/mercy` plays both from scripts.

The **vocoder** is an effect with two inputs, `voice` and `carrier`: connect a sampler (or any strip) to the first and a synth to the second, and the synth says what the voice says. It sounds a
frame late, which the mixer makes up for on the other strips, so the voice can also go to the mixer as it is and the two are mixed dry and wet by their levels.

The **input** (Sources) is the sound input of the computer, a microphone or a line in, to sing over a piece: its `gain` (dB) and `mono` (the left channel on both sides) are its settings, and it is a
strip like any other, so it can go through effects and the sends of the mixer. It reads what the program passes in `catalog.Config.Audio` (`p.PatchInput(0)`); without it (a render) it is silent. Use headphones, or
the speakers sing back into it. The **recorder** (Effects) lets the sound through as it is and, while its `record` switch is on, writes it to a 32 bit float WAV file in the folder `recordings` (named after the module
and the time); the log of the patch says when the file is made and when it is closed. Put one on the master to keep the whole piece with the voice, and one between the input and the mixer to keep the voice dry. The
sound you hear is a little later than the piece (the sound card), so what you sing is a little late against it, in the recording too.

The **audiolooper** (Effects) is a looper for sound, like a looper pedal. The sound that goes through it is recorded into a loop of `bars` bars (1 to 16, 4/4) on the beat of the patch, and the loop plays on, pass
after pass, over the sound that comes in (`input level` is how loud that is, `loop level` the loop). `record` starts a layer at the start of the next pass and it ends with the pass after you switch it off; a new layer
can be recorded on top of the others as long as you like, and they play together. The page of the module shows a cell for every bar with its four beats lit as they pass (the first beat red while a layer is recorded,
and it says how long until the recording starts), and a row for every layer with its sound and a button that takes that layer away alone; `undo` takes the last one, `clear` all of them, `export to WAV` writes
every layer to `recordings/loop-<time>-layer<n>.wav` (one pass each, so it loops), or with `export files` set to `one file` the layers together to `loop-<time>.wav`. The sound card is late: what you sing or play in time is heard, and recorded, a
little after the beat, so a layer would play late; `play layers earlier` (milliseconds, try 30 to 120 and listen) puts it back in time. A layer plays at the speed it was recorded at: after a change of the tempo,
clear and record again. The module runs only when its output reaches the patch's output.

The tempo of the patch is in the bar at the top of the page (`pt.Tempo()`, 30 to 300 bpm, a value
typed too), and kept with a patch: loading one sets its tempo.

**Generators** make music in the patch, on two more kinds of cable: triggers (a moment: the
control tick it falls on, so notes start on their sample) and values (a number, or a chord). They
are `patcher.Generators()`, built on genny:

| Module | In | Out | Settings |
|---|---|---|---|
| clock | | trigger | division (1/4 to 1/32, triplets), swing, chance; a pattern typed in its window, looping: `x . x x` a hit or a rest a step, `x3` a roll in a step, `[x x x]` a group filling a step, `*2` a step (or group) lasting 2 steps, `x?` a hit by chance (`x?25`); a trigger carries how long its hit lasts |
| sequence | trigger, reset | value, after (a trigger once the value is out) | values typed in its window (numbers, note names `C3 Eb3`, chords `[C3 E3 G3]`), the order: in order, shuffled, random, up and down, converge, alternate, backwards (genny's sequence, bucket, arpeggio) |
| notes | trigger, pitch, velocity, length (steps) | notes | the pitch, velocity, length and transposition for the inputs not patched. A length is in steps of the clock that triggered the note (at 1/16 and 120 bpm a step is 125 ms: a length of 8 is a second) |
| quantize | value | value | a key and a scale (tools/harmony): every pitch, every note of a chord, to its nearest note in it |
| arp | notes, clock (trigger) | notes | an arpeggiator: the notes it gets (a keyboard, a note editor) are held, and on every trigger of the clock it plays the next, in a mode (up, down, up & down, down & up, as played, random, chord) over 1-4 octaves; gate (how much of a step a note lasts), hold (the notes stay after the keys, until new ones), transpose |
| looper | notes | notes | a live looper for notes: what comes in (a keyboard, a note editor) goes on to the synths as it is, and is recorded into a loop of `bars` bars (1 to 16, 4/4) on the grid of the patch, so it is always on the beat and follows the tempo. `record` starts a layer at the start of the next pass and it ends with the pass after you switch it off; the layers play together on every pass (`play`) and a new one can be recorded on top of them as often as you like. `quantize` (off, 1/4, 1/8, 1/16, 1/32) puts the start and the length of the notes you play on a grid; `undo` takes the last layer away, `clear` all of them. `export to MIDI` writes the layers to `recordings/loop-<time>.mid` (a track for each layer and the tempo of the patch, the loop repeated `loop repeated` times) and the log of the patch says where. The side panel shows the loop: a cell for every bar with its four beats lit as they pass (the first beat is red while a layer is recorded, and it says how many beats until recording starts), and a row for every layer with its notes and a button that takes that layer away, the others going on. `export to script` writes a script of the patcher for each layer, `recordings/loop-<time>-layer<n>.js`: a table of its notes ([step, length, pitch, velocity], with note names) and a `trigger` that plays them, on its own `clock` (a sixteenth, or a finer division if the notes need it); put it in a script module with its notes cable to a synth, and edit it as you like. `script files` chooses a file for every layer (each can go to a synth of its own) or one file with all the layers (`loop-<time>.js`, through one output). |
| song | | bar, section (triggers), number (value), progress (mod) | its sections typed (`Intro 4, Verse 8, Chorus 8`; `3/4` among them makes the bar three beats); tiles to play a section next or switch one off, repeat, jump at, loop; tiles of its parts (its gates) |
| lane | | value, mod | a curve per section typed (`Intro: 0 → 1`, `Verse: 1`, `Chorus: 0.2 → 0.8 smooth`; lin, exp, smooth): through the section as it plays |
| automation | | lane 1 … lane N (mod) | lanes of points over the bars of the song, drawn on the page (the Lanes button) or typed (`Cutoff: 0 0.2, 6 0.8 smooth, 12 0.5 step`: a position in bars, a value 0-1, how it goes on: lin, smooth, step, exp); presets of 2, 4, 8 or 16 lanes, every lane a mod output of its own; without a song the lanes go round over its `bars` |
| gate | in (audio), notes, trigger | out, notes, trigger | its name (its tile) and the sections it plays in (`Verse, Chorus`, `not Intro`, `all`) typed; on (its tile in the song's side panel switches it) |
| part | | notes | notes written per step (`C3 Eb3 . G3 -`: `.` a rest, `-` held longer, `C3+Eb3` a chord, `[C3 D3 Eb3]` a group in a step, `*2` longer, `?` by chance), per section too; the clock's division, swing, chance; velocity, length, transpose; **chord notes** follow a harmony module (`@1` the root, `@3` the fifth, `@b` the bass, `@c` the chord, `'` and `,` an octave up and down), so a figure is written once for every chord; `register` and `bass` are the lowest notes they sit at |
| harmony | | chord (value), bass (value), change (trigger) | the chord track of the patch (one a patch, like the song): chords typed as symbols (`Dm7`, `Bb/D`) or roman numerals in its key and scale (`i VI V7`), a bar each, `*2` for two bars, `-` to hold, `.` for none; per section (`Chorus: F Am Dm Bb`), looping over the section. The parts with chord notes, a script (`harmony()`) and the note editor (the chords over the clip, the tones of the chord lit on its keys) ask it |
| noteroll | | notes | the note editor, a piano roll in the side panel that plays with the tempo of the patch: select with a rectangle, move, resize and copy-drag several notes, copy and paste at a cursor, duplicate (ctrl+D repeats a selection or a region of the ruler), undo and redo, a velocity lane, quantize and legato, zoom and scroll, the keys of a tool (V select, D draw, E erase); a **clip for every section of the song** (the loop plays where a section has none) that can be copied to other sections, and the chords of the harmony module shown over it. Kept with the patch |

A notes module plays the triggers of a control tick at its end, after every other module had its
tick: the values feeding it are up to date whatever order the cables were drawn in. Texts (a clock's
pattern, a sequence's values, a script) are typed in the module's window (a double tap on it), in
a box of lines each, as a script: Ctrl+Enter or Run takes them, or the panel closing; a pattern or
values may go over lines. One that can't be read is shown in red with what is wrong with it below,
the module goes on with the one before. They are kept with
the patch.

**A song** gives a patch sections. A song module's sections are typed in its window (a name and
a number of bars each: `Intro 4, Verse 8, Chorus 8, Outro 4`); a transport (arrange) plays them on
the patch's clock, from the start again on a restart. Its tiles (in its window) play a section
next, switch one off (skipped), with repeat (the section playing again and again), when a section
asked for starts (the end of the section playing, or the next bar) and loop; the bar at the top of
the page shows the section playing, its bar and the one asked for. Its outputs: a trigger every bar
and at every section (a note of a step lasts a bar), the section's number and how far through it
(mod). The patch follows it:

- a sequence's values and a clock's pattern can be per section: a line `Verse: C3 Eb3 G3` (or
  `Intro: x . . .`) each, the lines without a section for the others (with only sections given, the
  others are silent). A sequence starts its values over at every section, a clock counts its
  pattern from the section's start.
- a script has `song` (`song.section`, `song.index`, `song.bar` in the section, `song.bars`,
  `song.progress`, `song.loop`, `song.sections`, `song.playing`, and `song.beats` to the bar with `song.beat` in it) and calls `section(name, index)`
  when a section starts (its "song" preset: a chord a bar from the song's bar trigger, other chords
  in the chorus).

- a lane gives a value per section: a curve each, through the section as it plays; its mod output
  to any setting (a filter opening over the intro), its value output to a value input.
- an automation module is a lane drawn instead of typed, with as many lanes as its preset says, each a mod output. The Lanes button in the bar above the patch opens
  the lanes over its whole width, a row for every lane over the bars of the song (its sections on the ruler): a click adds a point (drag to place it, Shift for off
  the grid, the snap is chosen at the top), a drag moves a point, a double-click gives it the next shape (the way it goes on to the next point: `lin`, `smooth`,
  `step`, `exp`), Delete or a right click or an Alt-click takes it away, the name of a lane is typed in its row, and a line and a dot show where the song is and
  what every lane is. What is drawn is kept as the text of the module's `lanes` box (a line a lane), which can be typed instead. A position is in bars from the
  start of the song (the form of its song module), so the lanes follow the sections wherever the song goes.
  A lane records: the **●** button of its row (the lane has to be cabled to a setting first, the row says where: `→ module · setting`) arms it, and its cable lets go of the
  setting while the song plays; move the setting yourself (its knob on the page of its module) and the move is written into the lane, over the bars it took, thinned
  to the points that matter (what the lane had there is taken away, outside the move it is as it was), and the move is over when the setting has been still for a
  moment. The button lights while it writes; press it again and the lane plays what it has.
- a gate makes a part of the song: sound, notes and triggers go through it in its sections only
  (a note let go always goes through), and when it is on. The song's window has a tile for
  every gate of the patch, by its name: a tap switches the part off and on while it plays, the
  parts playing light up.
- a part plays notes written per step, as a clock plays its pattern: per section too.

The **setlist** example plays the patches kept in a folder as the songs of a playlist
(modules/playlist): each built in a patch of its own when it is needed; one with a song ends with
it (its loop off: skipping a section ends it sooner), one without plays until Next. Its page has
the songs, the sections of the song playing and the patch playing itself (its patcher), to change
while it plays. A patch as a playlist song:

```go
playlist.Song{Name: name, Build: func(sp *psalm.Psalm) playlist.Built {
	ctl := panel.New(sp, name)
	pt := patcher.New(sp, ctl, "Patch", catalog.Kinds(cfg), patcher.Keep("patches"))
	_ = pt.Load(name)
	b := playlist.Built{Out: pt.Out(), Panel: ctl}
	if tr := patcher.SongOf(sp.Env()).Transport(); tr != nil {
		b.Transport, b.Length = tr, tr.Form().Beats()*60/pt.Tempo().Value()
	}
	return b
}}
```

One song a patch: the last made; `patcher.SongOf(env)` is the song of a patch for modules of your
own (`At(tick, ppq)`: where it is).

The full reference for generators and scripts (every setting, default and option) is in [Scripting](scripting.md).

**Scripts** (`patcher/script`) are modules of JavaScript (goja). A script's top level runs when it
starts, to make its generators; then `trigger(ev)` on every trigger (`ev.step`, the step of the
clock, and `ev.seconds`, how long a step is) and `value(v, port)` on every value on its input `a` or
`b`. Its outputs: notes, values `a` and `b`, a trigger and a mod output.

```js
const minor = scale("C minor")
const walk = genny.rwalk(60, 48, 72, 0.15)          // a random walk
const lengths = genny.bucket([1, 1, 2, 0.5])        // shuffled, every one once before again

function trigger(ev) {
  note(minor.snap(walk.next()), 0.8, lengths.next()) // pitch, velocity, length in steps
  if (ev.step % 16 === 0) note(["C2", "G2"], 0.9, 8) // a chord, by note names
}
```

| In a script | |
|---|---|
| `note(pitch, velocity = 0.8, steps = 1, delay = 0)` | a note (or a chord: an array) to the synths its notes cable goes to, played at the end of the tick, or `delay` steps later (to play off the grid) |
| `send(port, v)`, `fire()`, `mod(v)` | a value (a number, a note name, a chord) on output `"a"` or `"b"`; a trigger; the mod output's value (0-1) |
| `input(port)`, `log(...)` | the last value on input `"a"` or `"b"`; a line in the log of the patch (the Log button at the top) |
| `pitch("C3")`, `scale("C minor")`, `bpm()` | a pitch; a scale: `snap(p)`, `pitch(degree, octave)`, `notes(lo, hi)`, `contains(p)`, `step(p, n)`; the tempo |
| `shared` | an object every script of the patch sees |
| `genny.sequence(values, loop = true)` | the values in order |
| `genny.bucket(values, "shuffle" \| "random", loop = true)` | every one once before again, or any one every time |
| `genny.arpeggio(values, "up" \| "converge" \| "alternate" \| "random", "none" \| "exclusive" \| "inclusive", reverse)` | an arpeggio, mirrored |
| `genny.repeat(gen, n[, n2])` | n values of one that loops, one that ends n times over (n to n2 at random); then it ends |
| `genny.and(...gens)`, `genny.or("random" \| "shuffle", ...gens)` | one after the other (each to its end), looping; one of them every time |
| `genny.combine(...gens)`, `genny.continuous(gen)`, `genny.constant(v)` | a value of each (an array); one that ends started over; always v |
| `genny.rand(lo, hi)`, `genny.rwalk(origin, min, max, step, "clip" \| "bounce" \| "jump")` | random numbers; a random walk |
| `genny.from(v)` | a generator of a function (called every time), a `function*` (started over on a reset), an iterator |
| `song`, `section(name, index)` | where the song of the patch is (a song module): `song.section`, `song.bar`, `song.progress`...; called when a section starts |

A generator gives `next()`, `done()`, `reset()` and `take(n)`; where one takes generators it takes
an array (a sequence of it), a function or a `function*` too. Every script of a patch runs in one
runtime (`script.NewRuntime()`, one per `catalog.Kinds`), each in a scope of its own, on the audio
thread. A script is compiled when it is typed (Ctrl+Enter, or leaving its box): one with an error
is shown in red with the line of it, and the script before it plays on. A call has a time budget
(`script.StartBudget` to start, `script.CallBudget` on every trigger and value): one going on too
long is stopped, its error shown, and it is not called again until it is typed anew; the rest of the
patch plays on. Its presets: a walk, an arpeggio of chords, a phrase of a `function*`, values to
other modules.

A `patcher.Kind` is a kind of module: its ports and how one is made at a preset (`Make`), giving the
module, how it describes its settings (for the editor and for modulation), and for a source of
notes or modulation what it plays (`Notes`) or what it is now (`Mod`, 0-1, asked once a buffer).
`patcher.Sources()` are the patcher's own: an LFO (a rate or a number of beats, five waves), a
random source (a new value at a rate, gliding to it) and a mixer of 4, 8 or 16 channels (its
preset): every channel at a level, panned, muted or not and sent to four buses, A to D, after its
level; its outputs are the main mix and sends A to D (patch a send into a reverb, a delay, a granulator or a shimmer and its
output back into the return of the same letter, each at a level of its own);
a note editor (`noteroll`, package `panel/noteroll`: a piano roll in the window to draw a loop of notes on, 1 to 64 beats, that plays with the patch's tempo and sends its notes to the synths it is cabled to; kept with the patch), a log (Tools) that shows the last 12 values and triggers sent to it in a box of its side panel (`Made.Log`), and sends them on;
`patcher/catalog` adds a keyboard (notes, and the mod wheel and pressure as modulation), every
synth with its presets (the DX7 with its own editor), the drum machine (the `stepseq` grid of the
drum machine example in its window, 16 patterns of up to 32 steps (16 to start with), a sample for every row chosen from the folder (kept with the patch by a number from its path), a `pattern` input (a value, from 0: a song's section number, a sequence) and a `next` input (a trigger, through the patterns in the rotation: A to D to start with) that ask for a pattern, switching when the pattern comes round (`Made.ValueIn`: the module that takes the value cables, when it isn't the one heard), a kit of samples from `Config.Drums`, "assets/sounds/drums"
by default; left out without them; `Made.Added` gives the patch what an editor adds, its sequencer)
and every effect. From code: `Add`, `Connect`,
`Disconnect`, `Remove`, `Move`, `SetText`, `Texts`, `TextError`, `Status`, `Copy`, `Paste`, `Save`, `Load`, `Clear`,
`Nodes`, `Cables`. A module's `Made.Texts` and `SetText` give it settings of text (`Help`: what to
know to type one, shown above its box in the window, a point a line, `code` in backquotes), `Made.Late` a function at the end of every control tick (a notes module's),
`Made.Status` a line shown under it (a script's error), `Made.Attach` gives it the patch's log (`Journal`, `pt.Journal()`: `Add(source, text)`). A module whose ports
depend on how it is made (the mixer's channels) gives them in `Made.Ins` and `Made.Outs`. Cables
are drawn over the modules, so one going back (a return) stays in sight, the ports over them.

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
