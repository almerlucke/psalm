# Scripting and generators in the patcher

The patcher (`modules/controls/panel/patcher`, see [Controls](controls.md#panel)) makes music in a
patch with **generators**: modules that send triggers, values and notes on cables. This page is the
reference for them: what each module takes, its settings and their defaults, the text you type in its
window, and the **script** module, which runs JavaScript with genny's generators.

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
  - [Shapers](#shapers)
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

Double tap a module to open its window. A module of text (a clock's pattern, a sequence's values,
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
| groove | straight, mpc 54, mpc 58, mpc 62, shuffle, laid back, push, dilla, backbeat | straight | the feel of the steps of a bar: some later, some softer (`tools/rhythm`) |
| pattern | text | `x` | a hit on every step |
| drift | 0 to 0.3 | 0 | how far the timing wanders, in steps, along a smooth curve (not jitter), the same every time |
| mutate | 0 to 1 | 0 | how much the pattern changes from pass to pass: some hits dropped, some rests become hits (at most a quarter of the steps at 1); the same seed makes the same changes |
| seed | 0 to 999 | 0 | the seed of the mutations |
| freeze | switch | off | keep the pattern of the pass it is on: a pattern that evolved and sounds right stays |
| lanes | text | | a value a step, see below |

The pattern loops. In it:

| | |
|---|---|
| `x` (also `X`, `o`, `1`) | a hit |
| `.` (also `-`, `_`, `0`) | a rest |
| `x3` | a roll: 3 hits in one step (1 to 64) |
| `[x x x]` | a group filling one step (a triplet); `[x . x x]` with rests; groups can hold groups and rolls |
| `*2` after a step or group | it lasts 2 steps (`x*2`, `.*4`, `*0.5`); a note of that trigger lasts as long |
| `?` after a hit or group | by chance, 50% (`x?25`: 25%) |
| `%2/4` after a hit or group | a condition: only the second time through the pattern in every four (`%!2/4`: the other three); `%fill` only in the last bar of a section, `%first` only the first time through (`%!fill`, `%!first`) |
| `!` after a hit or group | it stays when the pattern mutates (`x!`; before a `%` condition) |
| `E(5,16,+1)` | a function that goes on with the passes through the pattern: `+1` a step more every pass (the hits turn round), `2+1` from 2; `CA(30,16,+)` a generation more every pass (the automaton runs on: a new row each time), `M(x . x x;16;3/4)` a new rhythm every 4 passes |
| `E(5,16)` | a euclidean rhythm: 5 hits spread as evenly as they can be over 16 steps (`E(5,16,2)` turned 2 steps to the right) |
| `A(3+3+2)` | an additive rhythm: a hit on the first step of every group |
| `M(x . x x . x . .;16;3)` | a rhythm like the example: a Markov chain on its steps (the last two decide the next), the steps to make, a seed |
| `CA(30,16)` | a row of an elementary cellular automaton: the rule (0 to 255; 30 and 90 make nice ones), the cells, the generation (default half the cells), a seed |

Spaces, `|` and new lines are left out, so `x . x . | x . x x` is a bar of 8 steps.
`1` and `0` are single steps (`1010` is four steps, never a roll).

**Lanes** give every step a value of its own, a lane a line: `prob: 100 50 . 80` (the chance the step plays, percent), `vel:` (its velocity, percent), `shift:` (how late, percent of a step; negative: early), `len:` (how long, percent) and `pitch:` (semitones added; for a part). A `.` is the default there. A lane goes round on **its own length**: a `prob` lane of 3 steps under a pattern of 4 drifts, which is polymeter. The velocity reaches the notes module and the parts (a clock's trigger carries it); the groove and the lanes work together.

In a module's window the lanes are bars to drag under their text: a row a lane, a bar a step (a drag sets its value, Alt-click or a right click makes it a gap, + and − change the length of the lane). Typing and dragging change each other.

For phasing (Reich), two clocks with patterns of 12 and 13 steps drift against each other; so do lanes of different lengths on one clock.

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
| song | `Intro 4, Verse 8, Chorus 8, Outro 4` (default) | sections with a number of bars; outputs: a trigger every bar and at every section, the section's number, and how far through it (mod). Its tiles play a section next, switch one off, repeat or loop. With `chords` it has the harmony inside it and the chord model of its window (see [controls](controls.md#panel)) |
| lane | a curve a line: `Intro: 0 → 1`, `Chorus: 0.2 → 0.8 smooth` (`lin`, `exp`, `smooth`); default `0 → 1` | a value per section, moving through it as it plays; mod output (0-1) to any setting, or a value output |
| automation | a line a lane: `Cutoff: 0 0.2, 6 0.8 smooth, 12 0.5 step` (a position in bars from the start, a value 0-1, then `lin`, `smooth`, `step` or `exp`); `bars` for the round without a song | lanes of points over the bars of the song, drawn on the page (the Lanes button) or typed, or recorded from a knob moved while the song plays (the ● of a lane); a mod output (0-1) for every lane (the preset says how many) |
| gate | the sections it plays in: `Verse, Chorus`, `not Intro`, `all` | sound, notes and triggers pass in those sections only, and when it is on. Its name is a tile in the song's window. A note let go always goes through |
| part | notes a step each (the `^3` after a note ratchets it into 3 hits, and `%` conditions as in a pattern; lanes as in a clock, `pitch:` too; chord notes follow the harmony, `@v` the chord voice led from the one before): `C3 Eb3 G3 - Bb3 . G3 Eb3` (default) | `.` a rest, `-` the note before held a step longer, `C3+Eb3+G3` a chord, `[C3 D3 Eb3]` a group in a step, `*2`, `?25` as in a clock's pattern. **Chord notes** follow a harmony module: `@1` the root, `@2` the next tone of the chord (the third), `@3` the fifth, `@4` the seventh (a triad: the root an octave up), `@-1` the tone below the root, `@b` the bass, `@c` the whole chord, `'` after one an octave up and `,` down, `C3+@3` together; a figure written once follows the chords (`@1 @3 @5 @3`). The root of `@1` is the first one at or above the part's `register` (48), the bass above `bass` (36); without a harmony (or a chord) they rest. Division, swing, chance, velocity, length (of its step), transpose; per section too |
| harmony | chords a bar each: `Dm7 G7*2 Cmaj7` (symbols, or roman numerals in its key and scale: `i VI V7`), `*0.5` half a bar, `-` the chord held a bar longer, `.` none; per section: a line `Chorus: F Am Dm Bb` each | the chord track of the patch (one a patch, like the song): the chords loop over the bars of a section. Outputs: the chord (a value: an array of pitches in root position from its lowest note), its bass, and a trigger when the chord changes. A part with chord notes and a script (`harmony()`) ask it for the chord at the moment they play |

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
| `note(pitch, velocity = 0.8, steps = 1, delay = 0)` | a note, or a chord (an array), to the synths its notes cable goes to. Velocity is clamped to 0-1. Steps are of the last trigger's `ev.seconds` (a sixteenth at the tempo when there has been none); the shortest note is 5 ms. `delay` is steps later (0.1: a tenth of a step), to play off the grid: a chord not quite together, a strum, a note behind the beat; the nearest tick (a few milliseconds), a negative delay plays on the step |
| `set(key, value)`, `set({key: value, ...})` | settings of the synths its notes cable goes to, by the keys of their pages (`morph`, `macro1`, `cutoff`, `ampAttack`, `filterType`...), sent at the end of the tick before its notes: numbers, `true`/`false` (1/0) and strings. `chainA` to `chainD` take the text of a chain of the [shaper synth](synths/shaper.md#shaper), read by the script first: a chain that can't be read is an error of the script with its column. At most 256 a tick |
| `send(port, v)` | a value on output `"a"` or `"b"`: a number, a note name (sent as its pitch), or an array (a chord) |
| `fire(delay = 0, count = 1)` | a trigger on the trigger output, carrying the event being handled. `delay` is in steps (`fire(0.5)`: halfway the next step, so swing and off-beats), `count` triggers are spread over one step, the first after the delay (`fire(0, 3)`: a ratchet) |
| `clock(division = "1/16", swing = 0.5)` | the script keeps its own clock on the tempo of the patch: `trigger(ev)` on its steps, with no clock cabled. `"1/4"`, `"1/8"`, `"1/8T"`, `"1/16"`, `"1/16T"` or `"1/32"`, swing 0.5 to 0.75; `clock(false)` stops it. Combine it with `fire` for a script that is the clock of other modules, with a rhythm of its own (`genny.euclid`, a markov chain, the song's section) |
| `mod(v)` | the value of the mod output, clamped to 0-1; it stays until set again |
| `input(port)` | the last value that came in on `"a"` or `"b"` (`undefined` before one) |
| `log(...)`, `console.log(...)` | a line in the log of the patch (the **Log** button in the bar at the top), starting with the time and the module (`script m3`); arrays and objects as JSON |
| `pitch("C3")` | a note name as a number (numbers pass through) |
| `chord("Dm7")` | a chord of a symbol ([Harmony](harmony.md): `m`, `7`, `maj7`, `sus4`, `m7b5`, `Dm/C` with a bass...): `close(lowest)` the chord in root position with the root at or above `lowest`, `bass(lowest)`, `tones(low, high)` every chord tone between two pitches, `contains(p)`, `voiceLead(prev)` the chord for the same number of voices as an array of pitches, moving as little as it can; `name`, `root` (a pitch class) |
| `bpm()` | the tempo of the patch |
| `harmony()` | the chord the [harmony](#song-lane-gate-and-part) module has now, as `chord("Dm7")` gives one (`null`: no harmony, or no chord) |
| `scale("C minor")` | a key and a mode (as in [Harmony](harmony.md)) with `snap(p)` (nearest note), `pitch(degree, octave)`, `notes(lo, hi)` (all pitches between), `contains(p)`, `step(p, n)` (n scale steps from p) |
| `chord("Dm7")` | a chord of a symbol (`m`, `7`, `maj7`, `sus4`, `m7b5`, `Dm/C` with a bass...; see [Harmony](harmony.md)): `close(lowest)` the chord in root position with the root at or above `lowest`, `bass(lowest)`, `tones(low, high)` every chord tone between two pitches, `contains(p)`, `voiceLead(prev)` the chord for the same number of voices as an array of pitches, moving as little as it can; `name`, `root` (a pitch class). A symbol that can't be read is an error of the script |
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
| `genny.choice(values, weights)` | any one every time, the more weight the more often (`choice(["C3", "G3"], [3, 1])`); weights default to equal | |
| `genny.markov(table, start)` | a chain: `table` is `{state: {next: weight, ...}}`, every state a value (its name: `"C3"`, `"60"`), every state must go somewhere. `start` defaults to the first state; keys that look like numbers sort first in JavaScript, so give `start` for those | first key |
| `genny.walk(dims, values)` | a random walk through a grid, one step along one dimension at a time (`walk([4, 3], twelveValues)`, first dimension first); every dimension needs 2 values or more | |
| `genny.logistic(a, x0)` | the logistic map, 0 to 1; chaotic above about 3.57 | `a = 3.9`, `x0 = 0.5` |
| `genny.henon(a, b)` | the x of the Hénon map, about -1.3 to 1.3 | `1.4`, `0.3` |
| `genny.ramp(steps, from, to, curve, loop)` | `steps` values from one to the other (2 or more), along a curve (1 straight, above 1 slow to start); then it ends and `done()` is true, or with `loop` it stays on `to` | `0`, `1`, `1`, `false` |
| `genny.euclid(hits, steps, rotate)` | `true` on `hits` spread evenly over `steps` (3 of 8: `x..x..x.`), `false` between, looping; `rotate` moves the start | `0` |
| `genny.flatten(gen)` | a generator of arrays (chords, bars) as their values one by one | |
| `genny.map(gen, f)`, `gen.map(f)` | the values of a generator through a function | |
| `genny.rand(lo, hi)` | a random number in the range | `lo = 0`, `hi = 1` |
| `genny.rwalk(origin, min, max, step, mode)` | a random walk from `origin` within `min`..`max`; each step moves by up to `step` times the range; at an edge `"clip"` holds, `"bounce"` turns back, `"jump"` goes back to the origin | `step = 0.1`, `"bounce"` |

The generators of genny's music packages (rhythm, scales and chords, rows, mutation, grammars, sieves, LFOs, noise, distributions, tendency masks and more) are `genny.*` too, with every default, mode and an example, in [genny.md](genny.md).

The list arguments of `and`, `or` and `combine` can be given as separate arguments or as one array.
Every random choice in the generators (`rand`, `rwalk`, `bucket`, `choice`, `markov`, `walk`, `arpeggio` `"random"`, `or`, `repeat` with two counts) draws from one stream that `psalm.Seed(n)` (and `Env.Seed`) also seeds, so a render with the same seed and patch repeats; from Go alone, `genny.Seed(n)`. The clock's chances and the arp's random mode use the module's own stream, seeded the same way.

A wrong option name is an error that lists the names it knows.

A `function*` ends like any other generator, and `genny.and`/`genny.repeat` use that:

```js
function* phrase() { for (const p of ["C3", "D3", "Eb3", "G3"]) yield pitch(p) }
const notes = genny.repeat(genny.from(phrase), 2)  // the phrase twice, then done()
```

`gen.map(f)` works on any generator: `genny.sequence(pitches).map(p => p + 12)`.

A generator made from a `function*` or an iterator is one value ahead: `done()` is true right after
its last value has come out. After the end it keeps giving that last value.

### Shapers

A shaper is a function of a number to a number. Put one after a generator to bend its values, or call
it by hand. They work on any number (a pitch, a velocity, a walk), not only on audio: `genny.shapers`
are genny's shapers.

```js
const walk = genny.rwalk(0.5, 0, 1, 0.2)
const pitches = walk.shape(genny.shapers.series(
  genny.shapers.mult(24),       // 0..1 to two octaves
  genny.shapers.add(48),        // from C3
  genny.shapers.quantize(1),    // whole semitones
))
// or with a function: walk.shape(x => 48 + Math.round(x * 24))

function trigger() { note(pitches.next()) }
```

`gen.shape(s)` and `genny.shape(gen, s)` give a generator of the values through `s`; `s.shape(x)`
shapes one value: a number, a note name, or a chord (every pitch of it). Wherever a shaper is wanted a
function of a number works too. A shaper that gives NaN or infinity (a division by 0) is an error of
the script, shown on the module.

| `genny.shapers.` | `x` becomes |
|---|---|
| `add(a)`, `mult(m)` | `x + a`, `m * x` |
| `linear(scale, shift)`, `bipolar()`, `unipolar()` | `x * scale + shift`; 0..1 to -1..1; -1..1 to 0..1 |
| `mod(m)`, `ripple(m)` | the remainder after dividing by `m` (not 0); `x` plus it |
| `abs()`, `sin()`, `tri()` | the absolute value; `sin(2π x)`; a triangle over 0..1 |
| `mirror(bottom, top)` | folded back into the range, as a ball bounces |
| `clip("hard", low, high)`, `clip("soft", amount)` | cut off at the edges (-1 to 1 by default); an arctangent curve |
| `pulse(w)` | 1 below `w`, else 0 |
| `quantize(q)` | the nearest multiple of `q` (not 0): `quantize(12)` to octaves |
| `chebyshev({1: 1, 3: 0.3})` | the harmonics (1 to 64) at their levels: a waveshaper |
| `table("sine", n)` or `table([0, 3, 1])` | read over 0..1: a wave (`sine`, `cosine`, `sawtooth`, `triangle`, `square`, each also `uni ...` for 0..1; `n` points, 256 to start; the end is the start again) or your values, with the points between them drawn straight |
| `fn(f)` | a function of a number as a shaper (also what a function stands for) |
| `chain(text, macros)` | a chain written as text, as the [shaper synth](synths/shaper.md#shaper) reads it: `chain("mult 3 \| sin")`, `chain("mult m1[1,5] \| sin", [0.3])` (the macros `m1` to `m4`, 0.5 to start); a text that can't be read is an error of the script |
| `thru()` | `x` |
| `series(a, b, ...)` | `x` through each in turn |
| `parallel(a, b, ...)` | `x` through each, the results added |
| `mixer([a, b], [la, lb])` | the results added at levels (1 to start) |
| `interp(index, a, b, ...)` | between neighbours: index 0 is the first, 1 the last |
| `route(i, a, b, ...)` | the shaper at index `i` |
| `supersaw(a1, m1, m2)`, `hardsync(a1)`, `softsync(a1)`, `trimod(m)`, `voyager()`, `pwmod()` | the waves of synth emulations over a phase 0..1 |

### Limits and errors

| | |
|---|---|
| `script.StartBudget` | 50 ms for the top level |
| `script.CallBudget` | 5 ms for every `trigger`, `value` and `section` call |
| `script.RenderBudget` | 20 s, instead of the two above, when the patch renders (`psalm.New(nil)`): there is no deadline offline, so a busy machine or a pause of the garbage collector must not stop a script; only one that never ends is |

A call that runs longer is stopped, its error is shown on the module (`it took too long`), and the
script is **not called again until it is typed anew**. The rest of the patch plays on. Errors thrown by
your code or by the API (an unknown port, a bad pitch, a wrong option) show with the script's own
line number. A script with an infinite loop therefore cannot hang the sound.

### The presets

`walk` (a random walk snapped to a scale, with shuffled lengths), `arpeggio` (chords arpeggiated with
a bass note under each), `function*` (a phrase twice and an answer, with `and` and `repeat`),
`chain` (a markov chain of notes, an euclidean rhythm and weighted lengths), `values` (a pitch on `a`, a trigger and a wave on `mod`, to other modules), `song` (a chord a bar,
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
