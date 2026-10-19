# Scripting and generators in the patcher

The patcher (`modules/controls/panel/patcher`, see [Controls](controls.md#panel)) makes music in a
patch with **generators**: modules that send triggers, values and notes on cables. This page is the
reference for them: what each module takes, its settings and their defaults, the text you type in its
side panel, and the **script** module, which runs JavaScript with genny's generators.

- [Cables](#cables)
- [Typing into a module](#typing-into-a-module)
- [clock](#clock)
- [sequence](#sequence)
- [notes](#notes)
- [quantize](#quantize)
- [arp](#arp)
- [Song, lane, gate and part](#song-lane-gate-and-part)
- [script](#script)
  - [How a script runs](#how-a-script-runs)
  - [The API](#the-api)
  - [Generators](#generators)
  - [Limits and errors](#limits-and-errors)
  - [The presets](#the-presets)
- [From Go](#from-go)

## Cables

| Kind | Colour | What goes over it |
|---|---|---|
| audio | blue | the sound |
| notes | green | notes from a keyboard, `notes`, `part`, `arp`, `noteroll` or a script to the synths it is cabled to |
| trigger | purple | a moment: the control tick it falls on (so a note starts on its sample), the step of the clock and how long it lasts |
| value | grey | a number, a pitch, or a chord (an array of pitches) |
| modulation | yellow | 0 to 1, over a part of the range of any setting |

A trigger carries `step` (the step of the clock it is in) and `seconds` (how long one step is at the
tempo of the patch). Notes made from a trigger last a number of those steps.

The tempo is the one in the bar at the top of the page (30 to 300 bpm), kept with the patch.

## Typing into a module

Double tap a module to open its side panel. A module of text (a clock's pattern, a sequence's values,
a script) has a box of lines there, with what to know to type it shown above it.

- **Ctrl+Enter** (or **Run**) takes the text; so does leaving the box or closing the panel.
- A text that can't be read is shown in red, with the reason (and the line, for a script). The module
  keeps the one before it and plays on.
- Pitches are numbers (MIDI: 60 is middle C) or note names: `C3`, `F#4`, `Bb2`. **C3 is 48**, so
  `C4` is 60.
- With a [song module](#song-lane-gate-and-part), a clock's pattern, a sequence's values and a
  part's notes can be written per section: a line `Verse: ...` each, and the lines without a section
  are for the others. With only sections given, the others are silent.

## clock

Sends a trigger on the steps of the patch's tempo.

| Setting | Range | Default | |
|---|---|---|---|
| division | `1/4`, `1/8`, `1/8T`, `1/16`, `1/16T`, `1/32` | `1/16` | how long a step is; `T` is a triplet |
| swing | 0.5 to 0.75 | 0.5 | every second step later (0.5 straight; what is in the step goes with it) |
| chance | 0 to 1 | 1 | the chance of every hit, multiplied with the chance of the hit in the pattern |
| pattern | text | `x` | a hit on every step |

The pattern loops. In it:

| | |
|---|---|
| `x` (also `X`, `o`, `1`) | a hit |
| `.` (also `-`, `_`, `0`) | a rest |
| `x3` | a roll: 3 hits in one step (1 to 64) |
| `[x x x]` | a group filling one step (a triplet); `[x . x x]` with rests; groups can hold groups and rolls |
| `*2` after a step or group | it lasts 2 steps (`x*2`, `.*4`, `*0.5`); a note of that trigger lasts as long |
| `?` after a hit or group | by chance, 50% (`x?25`: 25%) |

Spaces, `|` and new lines are left out, so `x . x . | x . x x` is a bar of 8 steps.
`1` and `0` are single steps (`1010` is four steps, never a roll).

A clock follows the patch's clock: a restart, a jump back, a new division or pattern, or a new section
starts it over from where it is. After a jump in tempo it plays no burst of old hits.

## sequence

A value on every trigger, from a list.

| | |
|---|---|
| in | `trigger` (the next value), `reset` (back to the start) |
| out | `value`, and `after`: a trigger once the value is out (to chain modules that need the value first) |
| values | text; default `C3 Eb3 G3 Bb3` |
| order | default `in order` |

Values are numbers and note names, separated by spaces, commas or new lines; a chord is in brackets:
`[C3 Eb3 G3]` (it is sent as one value, an array of pitches).

| Order | |
|---|---|
| in order | 1 2 3 4 1 2 3 4... |
| shuffled | every value once before any again (genny's bucket) |
| random | any one every time |
| up and down | up, then back down without repeating the ends |
| converge | from the outside in |
| alternate | first, last, second, second to last... |
| backwards | the list reversed |

All orders loop. Changing the values or the order makes the generator again, from its start. With a
song, a section starts its values over each time it starts.

## notes

Turns triggers into notes for the synths it is cabled to (notes output).

| In | Setting | Range | Default | |
|---|---|---|---|---|
| trigger | | | | plays a note at the end of the tick |
| pitch | pitch | 0 to 127 | 48 | used when nothing is cabled to the input; a chord plays every note |
| velocity | velocity | 0 to 1 | 0.8 | |
| length | length | 0.05 to 16 | 0.9 | in **steps of the clock that triggered it**: at 1/16 and 120 bpm a step is 125 ms, so 8 is a second |
| | transpose | -36 to 36 | 0 | semitones, added to the pitch |

A cabled input wins over its setting.

## quantize

Moves every value it gets (a pitch, or every note of a chord) to the nearest note of a scale.
Settings: `key` (C, Db, D, Eb, E, F, F#, G, Ab, A, Bb, B; default C) and `scale` (the modes of
`tools/harmony`, see [Harmony](harmony.md); default `minor`).

## arp

An arpeggiator. In: `notes` (held notes from a keyboard or a note editor) and `clock` (a trigger);
out: `notes`. On every trigger of the clock it plays the next held note.

| Setting | Range | Default | |
|---|---|---|---|
| mode | up, down, up & down, down & up, as played, random, chord | up | |
| octaves | 1 to 4 | 1 | how many octaves it climbs |
| gate | 0.05 to 1 | 0.8 | how much of a step a note lasts |
| hold | on/off | off | the notes stay after the keys are let go, until new ones are played |
| transpose | -36 to 36 | 0 | semitones |

## Song, lane, gate and part

These follow the form of a song. One song a patch (the last made).

| Module | Text | |
|---|---|---|
| song | `Intro 4, Verse 8, Chorus 8, Outro 4` (default) | sections with a number of bars; outputs: a trigger every bar and at every section, the section's number, and how far through it (mod). Its tiles play a section next, switch one off, repeat or loop |
| lane | a curve a line: `Intro: 0 → 1`, `Chorus: 0.2 → 0.8 smooth` (`lin`, `exp`, `smooth`); default `0 → 1` | a value per section, moving through it as it plays; mod output (0-1) to any setting, or a value output |
| gate | the sections it plays in: `Verse, Chorus`, `not Intro`, `all` | sound, notes and triggers pass in those sections only, and when it is on. Its name is a tile in the song's side panel. A note let go always goes through |
| part | notes a step each: `C3 Eb3 G3 - Bb3 . G3 Eb3` (default) | `.` a rest, `-` the note before held a step longer, `C3+Eb3+G3` a chord, `[C3 D3 Eb3]` a group in a step, `*2`, `?25` as in a clock's pattern. Division, swing, chance, velocity, length (of its step), transpose; per section too |

## script

A module of JavaScript ([goja](https://github.com/dop251/goja), ES6 with generators). It makes
generators once, then reacts to triggers and values.

| | |
|---|---|
| in | `trigger`, `a`, `b` (values) |
| out | `notes`, `a`, `b` (values), `trigger`, `mod` |

### How a script runs

1. **Compile.** When the text is taken (Ctrl+Enter) it is compiled off the audio thread. A syntax
   error is shown with its line and the script before it plays on.
2. **Start.** At the next control tick, on the audio thread, the **top level** runs once. This is
   where generators, scales and tables are made, and where state starts:

   ```js
   const minor = scale("C minor")
   const walk = genny.rwalk(60, 48, 72, 0.15)   // made once, kept between triggers
   let count = 0                                // so is this

   function trigger(ev) { ... }                 // found by name when the top level is done
   ```

   The top level runs inside a function of its own, so `const` and `let` are private to the script.
   The functions `trigger`, `value` and `section` are looked up by name when it returns; any of them
   may be left out.
3. **Run.** On every trigger `trigger(ev)` is called; on every value on `a` or `b`, `value(v, port)`;
   when a section of the song starts, `section(name, index)`.
4. **Notes** made with `note()` are collected and played at the end of the tick.

Running it again (Ctrl+Enter) runs the top level again in a new scope: **every generator and counter
starts over**. Only `shared` is kept. A top level that throws leaves the script without its
functions (it does nothing until it is typed again); an error inside `trigger` or `value` is shown
and the script goes on.

All scripts of a patch share one runtime but each has a scope of its own. They run on the audio
thread, so keep them short.

The order within a tick: the song's position is brought up to date (and `section` called if a section
starts) **before** `trigger` is called, so `song.section` is always the section the trigger falls in.

### The API

| | |
|---|---|
| `trigger(ev)` | `ev.step` is the step of the clock, `ev.seconds` how long a step is |
| `value(v, port)` | `v` is the value, `port` is `"a"` or `"b"` |
| `section(name, index)` | a section of the song starts (also again after a restart of the song) |
| `note(pitch, velocity = 0.8, steps = 1)` | a note, or a chord (an array), to the synths its notes cable goes to. Velocity is clamped to 0-1. Steps are of the last trigger's `ev.seconds` (a sixteenth at the tempo when there has been none); the shortest note is 5 ms |
| `send(port, v)` | a value on output `"a"` or `"b"`: a number, a note name (sent as its pitch), or an array (a chord) |
| `fire()` | a trigger on the trigger output, carrying the event being handled |
| `mod(v)` | the value of the mod output, clamped to 0-1; it stays until set again |
| `input(port)` | the last value that came in on `"a"` or `"b"` (`undefined` before one) |
| `log(...)`, `console.log(...)` | shown under the module; arrays and objects as JSON. Only the last call is kept |
| `pitch("C3")` | a note name as a number (numbers pass through) |
| `bpm()` | the tempo of the patch |
| `scale("C minor")` | a key and a mode (as in [Harmony](harmony.md)) with `snap(p)` (nearest note), `pitch(degree, octave)`, `notes(lo, hi)` (all pitches between), `contains(p)`, `step(p, n)` (n scale steps from p) |
| `shared` | an object every script of the patch sees; it lives as long as the patch's runtime |
| `song` | `playing`, `section` (`""` when none), `index` (-1), `bar` (in the section), `bars`, `progress` (0-1 through the section), `loop`, `sections` (the names) |
| `genny` | the generators below |

### Generators

A generator is made once (at the top level) and asked for a value each time:

| Method | |
|---|---|
| `next()` | the next value |
| `done()` | true once a generator that does not loop has ended; always false for ones that loop |
| `reset()` | start over |
| `take(n)` | an array of the next n values (at most 4096) |

Wherever a generator is wanted, you can give: another generator; **an array** (a sequence of it, looping);
**a function** (called for every value, never done, `reset()` does nothing); **a `function*`** (its
values, started again by `reset()`); an iterator object with `next()` (cannot restart); or any other
value (always that value). `genny.from(x)` makes a generator of any of these.

| Constructor | Gives | Default |
|---|---|---|
| `genny.sequence(values, loop)` | the values in order | `loop = true` |
| `genny.bucket(values, mode, loop)` | `"shuffle"`: every value once, in random order, before any again; `"random"`: any one every time | `"shuffle"`, `loop = true` |
| `genny.arpeggio(values, mode, mirror, reverse)` | `mode`: `"up"`, `"converge"`, `"alternate"`, `"random"`; `mirror`: `"none"`, `"exclusive"` (up and back down without repeating the top), `"inclusive"` (repeating it); `reverse` flips the direction | `"up"`, `"none"`, `false`; loops |
| `genny.repeat(gen, n)` | n values of a generator that loops, or one that ends played n times over, then it ends | |
| `genny.repeat(gen, n, n2)` | the same with n chosen from n to n2 at random each time it starts | |
| `genny.and(...gens)` | one after the other, each to its end; loops | |
| `genny.or(mode, ...gens)` | one of them every time; `"random"` or `"shuffle"` (each once before again) | `"random"` |
| `genny.combine(...gens)` | a value of each at once, as an array | |
| `genny.continuous(gen)` | one that ends, started over every time it does | |
| `genny.constant(v)` | always v | |
| `genny.rand(lo, hi)` | a random number in the range (give both) | `hi = 1` |
| `genny.rwalk(origin, min, max, step, mode)` | a random walk from `origin` within `min`..`max`; each step moves by up to `step` times the range; at an edge `"clip"` holds, `"bounce"` turns back, `"jump"` goes back to the origin | `step = 0.1`, `"bounce"` |

The list arguments of `and`, `or` and `combine` can be given as separate arguments or as one array.
A wrong option name is an error that lists the names it knows.

A `function*` ends like any other generator, and `genny.and`/`genny.repeat` use that:

```js
function* phrase() { for (const p of ["C3", "D3", "Eb3", "G3"]) yield pitch(p) }
const notes = genny.repeat(genny.from(phrase), 2)  // the phrase twice, then done()
```

A generator made from a `function*` or an iterator is one value ahead: `done()` is true right after
its last value has come out. After the end it keeps giving that last value.

### Limits and errors

| | |
|---|---|
| `script.StartBudget` | 50 ms for the top level |
| `script.CallBudget` | 5 ms for every `trigger`, `value` and `section` call |

A call that runs longer is stopped, its error is shown on the module (`it took too long`), and the
script is **not called again until it is typed anew**. The rest of the patch plays on. Errors thrown by
your code or by the API (an unknown port, a bad pitch, a wrong option) show with the script's own
line number. A script with an infinite loop therefore cannot hang the sound.

### The presets

`walk` (a random walk snapped to a scale, with shuffled lengths), `arpeggio` (chords arpeggiated with
a bass note under each), `function*` (a phrase twice and an answer, with `and` and `repeat`),
`values` (a pitch on `a`, a trigger and a wave on `mod`, to other modules), `song` (a chord a bar,
others in the chorus). They are in `patcher/script/kind.go` and a good place to start.

```js
// every trigger: a note of a random walk, in C minor
const minor = scale("C minor")
const walk = genny.rwalk(60, 48, 72, 0.15)
const lengths = genny.bucket([1, 1, 2, 0.5])

function trigger(ev) {
  note(minor.snap(walk.next()), 0.8, lengths.next())
}
```

## From Go

```go
kinds := catalog.Kinds(catalog.Config{Input: keys, IR: hall}) // includes patcher.Sources(), patcher.Generators() and a script kind
// to build your own set: one runtime per set, so its scripts share `shared`
rt := script.NewRuntime()
kinds = append(kinds, script.Kind(rt))
```

`catalog.Kinds` makes a runtime of its own. A script can also be
used directly: `script.New(rt)`, `SetCode(text)` (compiles; returns the error), `Status()`. The
generators have constructors too (`patcher.NewClock`, `NewSequence`, `NewNote`, `NewQuantize`,
`NewArp`) with the fields shown above (`Division`, `Swing`, `Chance`, `Order`, `Pitch`, ...), and
`patcher.ParsePattern` and `patcher.ParseValues` read the text of a pattern and of values.
