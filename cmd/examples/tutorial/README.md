# Arranging in psalm: a tutorial

Fifteen short programs, one idea each, from a single note to a whole song, and then playing it live from a control panel and a drum machine. Every step plays: you
hear what the code does while you read it. Read them in order, each builds on the one before.

Run a step from the root of the repository (the steps load sounds from `assets`):

```
go run ./cmd/examples/tutorial/01-notes                       play it, Ctrl-C to stop
go run ./cmd/examples/tutorial/02-form -render form.wav       or render it to a file
go run ./cmd/examples/tutorial/04-feel -seed 3                the same random choices every run
```

The explanation of every step is in the comment at the top of its `main.go`, and the code is
commented along the way. Step 1 writes out everything a program needs around the music; the other
steps leave that to the small [`tutorial`](tutorial/tutorial.go) package (`tutorial.New` and
`tutorial.Play`), so they show only what they teach.

## The steps

| Step | Teaches | You hear |
|---|---|---|
| [01-notes](01-notes/main.go) | Notes (`score.Note`), score notation (`E5:1 G5 C6:2@.8`), a sequencer playing a synth, the patch | Chords and a melody on an electric piano |
| [02-form](02-form/main.go) | The form: sections, parts written per section (`form.Part`), `nil` for silence, the schedule | Intro, verse, chorus, outro: the bass entering, a melody per section |
| [03-bars](03-bars/main.go) | Writing bar by bar (`arrange.PerBar`, `arrange.Bars`), a `nil` bar is a rest, keys and roman numerals, voicings (`Close`, `VoiceLead`, `BassPitch`, `Triad`), arpeggios | A voice-led pad, a bass that waits a bar, an arpeggio |
| [04-feel](04-feel/main.go) | Accents (`score.Shape`, `Velocities`, `Hold`), the feel of a part (timing, spread, swing, strum), section dynamics | The same groove on the grid, played like a person, swung, then loud and strummed |
| [05-harmony](05-harmony/main.go) | Checking parts against the chords (`harmony.Check`): avoid notes, passing and neighbour notes | A melody with two wrong notes (reported at the start), then corrected |
| [06-automation](06-automation/main.go) | The arrangement and its lanes (`arrange.Sections{...}.Else(v)`), curves (`Const`, `Linear`, `Exp`, `Smooth`), sending lanes to settings by name | A filter closing, opening and closing again, a pad fading in |
| [07-bars-in-time](07-bars-in-time/main.go) | Curves at bars with a `Span`: `Between`, `From`, `Last`, `Steps`, `On`, `Fade`, `Pedal` | A piano with its pedal changed every bar, an arpeggio growing bar by bar and switched off for two |
| [08-tempo](08-tempo/main.go) | The tempo lane: slower sections, an accelerando, a ritardando; the schedule following the tempo | A groove speeding up and slowing down |
| [09-events](09-events/main.go) | Things that happen at a moment: `At`, `OnSection`, `OnBar`, `Play` (a channel in some sections only) | Sleigh bells at every chorus, the keys swinging in the chorus only |
| [10-mixer](10-mixer/main.go) | Channels (volume, pan), buses (a reverb, an echo) and sends, groups with effects, automating sends and volumes | A dry section, a wet one, waves fading in under the outro |
| [11-streams](11-streams/main.go) | Parts played by generators (`arrange.Stream`): rhythm and message generators, section parameters, streams driving any module | A glass melody wandering over a pentatonic scale, never the same twice |
| [12-tuning](12-tuning/main.go) | Tuning a part (`score.Config.Tuning`): equal temperament and just intonation | The same organ chords in both: listen for the beating that disappears |
| [13-song](13-song/main.go) | Everything together | A short song: intro, verse, chorus and outro |
| [14-panel](14-panel/main.go) | A control panel on a web page (`panel`): faders, knobs, toggles, buttons, selects and meters bound to the patch, curves and smoothing, the threads, presets, MIDI learn | The loop of the earlier steps, changed live from a browser or a phone: the filter, the levels, the tempo, the arpeggio |
| [15-drum-machine](15-drum-machine/main.go) | The step sequencer (`stepseq`): a grid of steps on the panel, every step with velocity, chance, roll, nudge, pitch and condition; rows of their own length, choke groups, swing; patterns switched by the arrangement, copied from one to another; presets of the whole setup | A beat with ghost notes, rolls and random skips, a fill on the switch, pattern B for the chorus, along with a written bass |
| [16-transport](16-transport/main.go) | The transport (`form.Transport()`): sections switched on and off, queued, repeated, jumping at the end of a section or at the next bar; on the panel as a tile per section, played from the pads of a MIDI controller | A song with a bridge that only plays when you ask for it, steered live from the page |

## The concepts at a glance

**Notes** (`score`): a `Note` has `Pitches` (MIDI, 60 is middle C), `Beats`, and optionally
`Hold` (how long it sounds), `Velocity` and `Amp`. Write them in Go (`score.N`, `score.Chord`,
`score.Notes`, `score.Rest`) or in notation (`score.MustParse`). A `score.Config` sets how a part
plays: `Amp`, `Legato`, `Velocity`, `Feel`, `Tuning`.

**The form** (`arrange.Form`): sections with a name, bars and dynamics. A part is a function from
a section to its notes (`arrange.Score`); `form.Part` plays it on a synth through the whole song.
Helpers: `arrange.PerBar` and `arrange.Bars` (bar by bar), `arrange.Only` (one section),
`arrange.In` (several).

**The arrangement** (`arrange.Arrangement`): follows the song while it plays. Lanes give a value
per moment (`arrange.Sections{"Intro": curve}.Else(value)`), placed at bars with `form.Span(name)`,
sent to settings (`AutomateMessage`, `Automate`, `AutomateFunc`) or the tempo (`Tempo`). Events
call functions at a moment (`At`, `OnSection`, `OnBar`) or turn modules on per section (`Play`).

**Harmony** (`tools/harmony`): chords, keys and roman numerals, voicings, arpeggios, tunings, and
`Check` for wrong notes.

**The mixer** (`modules/mixer`): channels, buses and groups; every setting can be automated.

**The panel** (`modules/controls/panel`): controls declared next to the patch and bound to its
settings (`panel.Message`, `panel.Input`, `panel.Func`), served as a web page (`Serve`), movable
from a MIDI controller (MIDI learn), with presets (`Save`, `Load`).

**The step sequencer** (`modules/controls/stepseq`): a grid of steps on the panel (`ctl.Grid`),
played on the patch's clock; every step has a velocity, chance, roll, nudge, pitch and condition.

**The transport** (`form.Transport()`): the song clock of a form, deciding which section comes next:
sections switched on and off, queued, repeated, live from code or from the panel (`tr.Controls`).

## Further

- [docs/controls.md](../../../docs/controls.md): `score` and `arrange` in full
- [docs/harmony.md](../../../docs/harmony.md): chords, keys, the harmony check and tunings
- [docs/controls.md#panel](../../../docs/controls.md#panel): the panel in full, its page and its API
- The big examples use all of this: `cmd/examples/showcase` (an orchestral piece), `cmd/examples/creation` (the seven days of Genesis), `cmd/examples/piano` (a tour of the piano)
