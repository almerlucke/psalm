# Generators in scripts (`genny`)

A **generator** hands out values one at a time: a melody, a rhythm, a modulation signal. In a script
(`script` module, see [scripting.md](scripting.md)) every generator is a `Gen`, made by a function of
the global `genny`, and you call it once per step:

```js
const melody = genny.sequence([60, 62, 64, 67])   // made once, kept between triggers
function trigger(ev) {
  note(melody.next(), 0.8, 1)
}
```

Generators are small and nest: a mutating phrase on a euclidean rhythm, quantized to a scale, swung.
The package is [genny](https://github.com/almerlucke/genny); this page is the script side of all of it.

- [The `Gen` object](#the-gen-object)
- [What you can give where a generator is wanted](#what-you-can-give-where-a-generator-is-wanted)
- [Conventions](#conventions)
- [Building blocks](#building-blocks): `sequence` `bucket` `arpeggio` `constant` `from` `map` `choice` `markov` `walk` `and` `or` `combine` `flatten` `repeat` `continuous`
- [Rhythm and time](#rhythm-and-time): `euclid` `euclidPattern` `divider` `poly` `rests` `gated` `pulseGate` `ratchet` `ratchetPattern` `automaton` `section` `split`
- [Timing and velocity](#timing-and-velocity): `swing` `humanize` `jitter` `accent`
- [Pitch and harmony](#pitch-and-harmony): `scale` `equalScale` `justScale` `chord` `diatonic` `voiceLeader` `progression` and the chord functions
- [Rows](#rows): `row` `retrograde` `inversion` `serial` `matrix`
- [Variation and structure](#variation-and-structure): `mutate` `lsystem` `grammar` `fibonacci` `collatz` `thueMorse` `primes` `learn` `pick` `shifting`
- [Constraints and sieves](#constraints-and-sieves): `allowIntervals` `blockIntervals` `noRepeats` `noDuplicates` `intervalWalk` `sieve` `funnel`
- [Lists](#lists): `jump` `stutter`
- [Numbers over time](#numbers-over-time): `rand` `rwalk` `ramp` `lfo` `slew` `noise` `lissajous` `logistic` `henon` and the attractors
- [Distributions](#distributions): `gaussian` `exponential` `beta` `cauchy` `weibull` `pink` `lehmer`
- [Tendency masks](#tendency-masks): `tendency` `tendencyChoice` `spray`
- [Lines of note events](#lines-of-note-events): `line.*` `counterpoint` `chordTones` `isorhythm` `phase` `learnStyle`
- [Forms](#forms): `fugue` `strettoDistances` `harmony.*` `polyrhythm.*` `tuning`
- [Stochastic music](#stochastic-music): `poisson` `density` `cloud` `freeStochastic`
- [Game of Life](#game-of-life): `life`
- [Evolving a phrase](#evolving-a-phrase): `evolve`
- [Shapers](#shapers)
- [Randomness and seeds](#randomness-and-seeds)
- [Recipes](#recipes)

## The `Gen` object

| Member | Does |
|---|---|
| `g.next()` | the next value |
| `g.take(n)` | the next `n` values as an array (at most 4096) |
| `g.done()` | `true` when a generator that can end has ended; a generator that loops is never done |
| `g.reset()` | start again from the beginning |
| `g.map(f)` | a generator of the values of this one through a function: `g.map(v => v + 12)` |
| `g.shape(shaper)` | a generator of the values through a [shaper](#shapers) |

A generator that **ends** (a `sequence` with `loop = false`, `repeat`, `ramp`, `collatz` without a loop)
keeps giving its last value once it is done. `done()` turns `true` right after the last value is taken,
not a call later.

## What you can give where a generator is wanted

Every argument that says *gen* takes any of these:

| You give | It is |
|---|---|
| another `Gen` | itself |
| an array | a sequence of it, looping: `[60, 64, 67]` |
| a function | called for every value, never done (`reset()` does nothing) |
| a `function*` | its values, started again by `reset()`; done when it returns |
| an object with `next()` | an iterator (it cannot start over) |
| any other value | always that value |

`genny.from(x)` makes a `Gen` of any of these.

```js
let n = 0
const counter = genny.from(() => n++)                    // 0, 1, 2 ...
const tune = genny.from(function* () { yield 60; yield 67 })   // 60, 67, then done
const pedal = genny.and(tune, [48])                      // 60, 67, then 48 for ever
```

## Conventions

- **Gates** are `true`/`false`. **Rests** are `null`: a generator that rests gives `null` for a step
  that does not play (`if (v !== null) note(v, 1, 1)`).
- **Positions are in steps** and the generator knows nothing about time: you call `next()` on a step
  of a clock (see `clock` in scripting.md) or of the song. A swing is a *fraction of a step* to delay by
  (`note(p, vel, steps, delay)`).
- **Defaults** are written in the signatures below; an argument can be left out or `undefined`.
- **Names** of modes are strings; a wrong name is an error that lists the right ones.
- A constructor that can never work throws a `TypeError` that names the function
  (`divider: clock: n must be at least 1`), so the mistake shows when the script is taken. `next()` never throws.
- **Empty input** is allowed for lists: a `sequence`, `bucket`, `and`, `or` or `arpeggio` without values
  is done at once.
- Where a value has a natural limit it is clamped instead of rejected (more pulses than steps, a
  probability above 1).
- **Made once.** Create generators outside `trigger` (at the top of the script, or behind a `let ready`
  flag): inside `trigger` you would start a new one every step. They keep their state across triggers.

---

## Building blocks

### `genny.sequence(values, loop = true)`

The values in order. With `loop = false` it ends after the last one.

```js
genny.sequence([1, 2, 3]).take(5)          // 1, 2, 3, 1, 2
const once = genny.sequence([1, 2], false)
once.take(3)                               // 1, 2, 2   (done: it keeps the last)
once.done()                                // true
once.reset(); once.next()                  // 1
```

### `genny.bucket(values, mode = "shuffle", loop = true)`

| mode | gives |
|---|---|
| `"shuffle"` | every value once, in random order, before any comes again (a bag of tiles) |
| `"random"` | any one every time, repeats allowed |

```js
const drum = genny.bucket(["kick", "snare", "hat", "hat"])      // each of the four once per round
const wild = genny.bucket([60, 62, 64], "random")
const oneRound = genny.bucket([1, 2, 3], "shuffle", false)       // ends after one round
```

### `genny.arpeggio(values, mode = "up", mirror = "none", reverse = false)`

The notes of a chord as an arpeggio, looping.

| mode | order of `[1, 2, 3, 4]` |
|---|---|
| `"up"` | 1 2 3 4 |
| `"converge"` | 1 4 2 3 (from the outside in) |
| `"alternate"` | 1 2 1 3 1 4 1 2 (the first note between each of the others) |
| `"random"` | any note every time |

`mirror` plays the pattern back on its way down: `"none"`, `"exclusive"` (the end notes are not played
twice: 1 2 3 2 | 1 2 3 2), `"inclusive"` (they are: 1 2 3 3 2 1 | 1 2 3 ...). `reverse = true` turns the order around.

```js
genny.arpeggio([60, 64, 67], "up", "exclusive").take(6)   // 60 64 67 64 60 64
genny.arpeggio([60, 64, 67], "up", "none", true).take(3)  // 67 64 60
```

### `genny.constant(v)`

Always `v`. A plain value does the same where a generator is wanted.

### `genny.from(x)`

A `Gen` of an array, a function, a `function*`, an iterator or a value; see
[what you can give](#what-you-can-give-where-a-generator-is-wanted).

### `genny.map(gen, f)` / `gen.map(f)`

The values of a generator through a function. The function may keep state.

```js
const lead = genny.sequence([0, 2, 4, 7]).map(d => 60 + d)
```

### `genny.choice(values, weights)`

Any one value every time; the more weight the more often. `weights` default to equal; there must be one
per value.

```js
const dyn = genny.choice(["p", "mf", "f"], [5, 3, 1])
```

### `genny.markov(table, start)`

A Markov chain. `table` is `{ state: { next: weight, ... } }`; the states are values (their names:
`"C3"`, `"60"`), every state must go somewhere and every weight must be 0 or more. `start` is the first
state; it defaults to the first key (keys that look like numbers sort first in JavaScript, so give
`start` for those).

```js
const m = genny.markov({
  C3: { C3: 1, E3: 2, G3: 2 },
  E3: { C3: 1, G3: 3 },
  G3: { C3: 3, E3: 1 },
}, "C3")
```

### `genny.walk(dims, values)`

A random walk through a grid, one step along one dimension at a time. `values` is the grid laid out
first dimension first; `dims` its size, each 2 or more. Good for a melody that can only move to
neighbours.

```js
const grid = genny.walk([4, 3], [60, 62, 64, 65,  67, 69, 71, 72,  74, 76, 77, 79])
```

### `genny.and(...gens)`

One after the other, each to its end; then it starts over. A generator that does not end (an array
loops) never gives way, so give ones that end: `genny.sequence(x, false)`, `genny.repeat(...)`.

```js
genny.and(genny.sequence([1, 2], false), genny.sequence([9], false)).take(5)   // 1 2 9 1 2
```

### `genny.or(mode = "random", ...gens)`

One of the generators every time it starts (when the one playing is done, or every value for
continuous ones). `"random"` picks any; `"shuffle"` plays each once before any again.

### `genny.combine(...gens)`

One value of each, together, as an array: a chord from three melodies.

```js
genny.combine([60, 62], [64, 65], [67, 69]).take(2)   // [60,64,67], [62,65,69]
```

### `genny.flatten(gen)`

A generator of arrays (chords, bars) as their values one by one.

### `genny.repeat(gen, n, n2)`

`n` values of a generator that loops; or a generator that ends, played `n` times over, then it ends.
With `n2` the count is chosen from `n` to `n2` at random each time it starts.

```js
genny.repeat([1, 2], 3).take(5)                           // 1 2 1, then done: the last value stays (1 1)
genny.repeat(genny.sequence([1, 2], false), 2).take(5)    // 1 2 1 2 2
```

### `genny.continuous(gen)`

A generator that ends, started over every time it does. Turns a one-off phrase into a loop.

---

## Rhythm and time

### `genny.euclid(hits, steps, rotate = 0)`

`true` on `hits` pulses spread evenly over `steps`, `false` between, looping (Bjorklund's algorithm).
`rotate` moves the start.

```js
genny.euclid(3, 8).take(8)      // x . . x . . x .
genny.euclid(5, 12, 2)          // five in twelve, turned two steps
```

### `genny.euclidPattern(steps, pulses, rotation = 0)`

The same as an array of booleans, to look at it whole or build another gate from it.

### `genny.divider(n, offset = 0)`

`true` on every `n`th step, starting at `offset`.

```js
genny.divider(4).take(8)        // x . . . x . . .
```

### `genny.poly(lengths)`

Cycles of different lengths run together. Every step gives an array with one boolean per cycle (`true`
where that cycle begins). `genny.poly([3, 4])` is the 3-against-4.

```js
const p = genny.poly([3, 4])
const [a, b] = p.next()         // a: the three cycle starts, b: the four cycle starts
```

### `genny.rests(gen, p = 0.5)`

The values of a generator, each played with chance `p` (0 to 1), else `null`. The generator advances
every step, so a rest does not shift the melody.

```js
const tune = genny.rests([60, 62, 64, 67], 0.7)
const v = tune.next()           // a note, or null
```

### `genny.gated(gen, gate)`

The values of a generator where the gate says `true`, else `null`. The melody and the rhythm are
separate generators, which is the point.

```js
const rhythm = genny.euclid(5, 8)
const line = genny.gated(genny.arpeggio([60, 64, 67, 72]), rhythm)
```

### `genny.pulseGate(divisions, level)`

A gate by the metric weight of the pulses (Barlow's indispensability): `divisions` is the meter as
nested divisions, `[3, 2, 2]` a bar of three in two in two = 12 pulses. The strong pulses are most
important. `level` thins the rhythm: **0 plays every pulse, a higher level only the more important
ones**, a level of the number of pulses or more plays none. `level` can be a generator, a number that
changes: an envelope or an LFO fades a rhythm in and out.

```js
genny.pulseGate([3, 2, 2], 0).take(12)                         // all twelve
genny.pulseGate([3, 2, 2], 6).take(12)                         // every other pulse: x . x . x . ...
const fade = genny.pulseGate([3, 2, 2], genny.ramp(48, 12, 0)) // fills in over 48 steps
```

### `genny.ratchet(p = 0.3, max = 4)` / `genny.ratchetPattern(counts)`

A step that is repeated: a roll. Every step gives an array of the **starts** of its repeats, as
fractions of the step. `[0]` is no roll; `[0, 0.5]` two; `[0, 0.25, 0.5, 0.75]` four.
`ratchet` rolls at random: chance `p`, up to `max` repeats. `ratchetPattern` plays the counts in order.

```js
const roll = genny.ratchetPattern([1, 1, 2, 1, 3, 1, 1, 4])
function trigger(ev) {
  for (const t of roll.next()) note(36, 0.8, 0.2, t)   // delay is a fraction of a step
}
```

### `genny.automaton(rule = 30, width = 16, random = false)` / `genny.automatonGate(rule, width, random)`

A one dimensional cellular automaton (rule 30, 90 and 110 are the famous ones). `automaton` gives one
row at a time as an array of booleans; `automatonGate` reads the rows one cell after the other as a
gate. A start of one cell in the middle, or random cells with `random = true`.

```js
const gate = genny.automatonGate(90, 16)       // a gate that never quite repeats
```

### `genny.section(parts, loop = true)`

Plays parts for a number of steps each: A for 4 steps, B for 8, around again. `parts` is
`[{ gen, steps, restart }]`: `steps` is the number of values (0 or less: until the generator is done),
`restart` starts the generator again every time the part begins; otherwise it carries on where it was.

```js
const form = genny.section([
  { gen: [60, 62, 64, 65], steps: 8 },
  { gen: [67, 65, 64, 62], steps: 8, restart: true },
])
```

### `genny.split(value, lower, prob = 0.5, branches = 2)` / `genny.splitting(...)`

A duration cut recursively into `branches` equal parts (2 or more). Each part is cut again with chance
`prob`, but only if its parts would not be shorter than `lower` (which must be above 0). `split` gives
the parts in time order as an array that adds up to `value` (with chance it may not be cut at all:
`[1]`); `splitting` is a looping generator that cuts a new one each cycle.

```js
genny.split(1, 0.125, 0.7)      // e.g. [0.5, 0.25, 0.125, 0.125]
const rhythm = genny.splitting(4, 0.25, 0.6, 3)
```

---

## Timing and velocity

These give **numbers per step** to add to the timing or the velocity of a note.

### `genny.swing(amount = 0.2, humanize = 0)`

The delay of each step as a fraction of a step: every second step late by `amount`, the others on the
beat. `humanize` adds a random shift up to that much.

```js
const sw = genny.swing(0.2)
note(p, vel, 1, sw.next())
```

### `genny.humanize(amount = 0.05)`

A random shift up to `amount` either way, in the range -amount to amount.

### `genny.jitter(gen, amount = 0.05)`

The numbers of a generator, each moved at random by up to `amount` either way.

### `genny.accent(groups, { downbeat = 1, group = 0.8, other = 0.6 })`

Velocities by the meter. `groups` is the grouping: `[2, 2, 3]` is 7/8 as 2+2+3, `[3, 3]` is 6/8. The
first step of the bar gets `downbeat` (default 1), the first step of each other group `group` (0.8), all
other steps `other` (0.55).

```js
const vel = genny.accent([2, 2, 3], { downbeat: 1, group: 0.75, other: 0.5 })
vel.take(7)                      // 1, 0.5, 0.75, 0.5, 0.75, 0.5, 0.5
genny.accent([2, 2, 3]).take(7)  // the defaults: 1, 0.55, 0.8, 0.55, 0.8, 0.55, 0.55
```

---

## Pitch and harmony

### `genny.scale(root, steps = "major", period = 12)`

A scale: a root pitch and ascending steps within a period (12 semitones). `steps` is a name or an
array of semitones above the root.

Names: `major`, `minor`, `harmonicMinor`, `melodicMinor`, `dorian`, `phrygian`, `lydian`,
`mixolydian`, `locrian`, `majorPentatonic`, `minorPentatonic`, `blues`, `wholeTone`, `diminished`,
`chromatic`, and Messiaen's seven modes of limited transposition, `messiaen1` (the whole tone scale) to `messiaen7`.

It returns an object:

| Member | Does |
|---|---|
| `s.degree(i)` | the pitch of degree `i`: 0 is the root, negative degrees below it, degrees past the end go up an octave (`period`) |
| `s.quantize(x)` | the nearest pitch of the scale |
| `s.degrees(gen)` | a generator of the pitches of the degrees a generator gives |
| `s.walk(start = 0, lo = -7, hi = 7, maxStep = 2, leap = 0.1, maxLeap = 4)` | a melody that walks over the degrees in steps of up to `maxStep`, and with chance `leap` leaps up to `maxLeap`; stays between `lo` and `hi` |
| `s.len()`, `s.root()` | number of steps, the root |

```js
const a = genny.scale(57, "minor")
a.degree(0)                                  // 57
a.degree(7)                                  // 69 (an octave up)
a.quantize(58)                               // 57 or 59: the nearest of A minor
const tune = a.degrees(genny.rwalk(0, -7, 7, 0.2).map(Math.round))
const wander = a.walk(0, -4, 7)
```

### `genny.messiaen(root, mode)`, `genny.transpositions(steps)`

`messiaen(60, 2)` is the second mode of limited transposition (the octatonic scale, semitone and tone
alternating), the same as `genny.scale(60, "messiaen2")`. Modes 1 to 7. `transpositions` tells how many
different transpositions a set of steps (a name or an array) has before it maps onto itself: 2 for the
whole tone scale (`"messiaen1"`), 3 for mode 2, 12 for an ordinary scale.

### `genny.sclScale(root, text)`

A scale from a **Scala `.scl` file**: give the text of the file, or the name of one that comes with
genny: `"12tet"`, `"just5"` (Ptolemy's 5-limit just chromatic scale), `"meantone"`, `"pythagorean"`,
`"bohlenPierce"` (13 steps to the tritave). The last pitch of the file is the period, so Bohlen-Pierce
repeats at the tritave, not the octave. The pitches are fractional semitones; the synth plays them as
they are. Notes are cents (`701.955`, with a dot) or ratios (`3/2`); a file that does not ascend is an error.

```js
const bp = genny.sclScale(48, "bohlenPierce")
bp.degree(13)                                      // a tritave above the root
const own = genny.sclScale(60, `! mine.scl
A five note scale
5
200.0
400.0
3/2
5/3
2/1
`)
```

### `genny.equalScale(root, divisions)`

A scale of equal steps of the octave: `divisions = 19` or `24` (quarter tones) for the other equal
temperaments.

### `genny.justScale(root, ratios)`

A scale from frequency ratios, in just intonation. `ratios` ascend from 1.

```js
const just = genny.justScale(60, [1, 9/8, 5/4, 4/3, 3/2, 5/3, 15/8])
```

Pitches are semitones and may be fractions; the synths take them as they come (a fraction is a detune).

### `genny.chord(root, kind = "major")`

The pitches of a chord as an array. `kind` is a name or an array of intervals in semitones.

Names: `major`, `minor`, `diminished`, `augmented`, `sus2`, `sus4`, `major6`, `minor6`, `major7`,
`minor7`, `dominant7`, `minorMajor7`, `halfDiminished`, `diminished7`, `add9`, `major9`, `minor9`,
`dominant9`.

```js
genny.chord(60, "minor7")        // [60, 63, 67, 70]
genny.chord(60, [0, 5, 10])      // a chord of fourths
```

### `genny.diatonic(scale, degree, size = 3)`

The chord on a degree of a scale (from `genny.scale`): `size` notes stacked in thirds of the scale.

```js
genny.diatonic(genny.scale(60, "major"), 1, 3)   // [62, 65, 69]: D minor in C
genny.diatonic(genny.scale(60, "major"), 4, 4)   // the seventh chord on G
```

### Chord functions

| Call | Gives |
|---|---|
| `genny.invert(chord, n)` | the chord inverted `n` times (the lowest note an octave up) |
| `genny.open(chord)` | an open voicing (every second note an octave down) |
| `genny.transpose(chord, semitones)` | the chord moved |
| `genny.harmonic(hz, harmonics, distortion = 1, sub = false)` | a spectral chord: partial `n` of a tone of `hz` sounds at `hz * n^distortion` (partial numbers like `[1, 2, 3, 5, 7]`, 1 is the fundamental). 1 is the harmonic series, above 1 stretches the partials apart, below 1 squeezes them (0: unisons). `sub` divides instead (`hz / n^distortion`: a series that goes down). The pitches are fractional MIDI notes, microtonal as the spectrum asks |
| `genny.fmChord(carrierHz, modHz, orders = 3)` | the sidebands of an FM pair as a chord |
| `genny.ringChord(aHz, bHz)` | the sum and difference tones of two tones |

### `genny.voiceLeader(low, high)`

An object that turns chords into the voicing that moves the least from the last one, inside the range
`low`..`high`. `leader.next(chord)` gives the voiced chord, `leader.reset()` forgets the last one.

```js
const leader = genny.voiceLeader(48, 72)
leader.next(genny.chord(60, "major"))     // a close voicing
leader.next(genny.chord(65, "major"))     // the next one with the fewest moves
```

### `genny.progression(kind = "pop", start = 0)`

Chord degrees as a Markov chain, a generator of whole numbers (0 is I, 3 is IV, 4 is V, 5 is vi...).

`kind` is `"pop"` (I, IV, V, vi), `"classical"` (tonic, subdominant, dominant, back), `"jazz"`,
`"blues"`, or **your own table**: `{ degree: [[to, weight], ...] }`. A degree without moves ends the
progression; a move to a degree that is not in the table goes back to the start.

```js
const prog = genny.progression("classical", 0)
const own = genny.progression({ 0: [[3, 2], [4, 1]], 3: [[4, 1]], 4: [[0, 1]] }, 0)
const key = genny.scale(60, "major")
const chordOfBar = genny.diatonic(key, prog.next(), 3)
```

---

## Rows

Twelve tone rows as arrays of pitch classes, and the classical operations on them.

| Call | Gives |
|---|---|
| `genny.row(n = 12)` | a random row: each of the `n` pitch classes once |
| `genny.retrograde(row)` | backwards |
| `genny.inversion(row)` | upside down (the intervals turned around) |
| `genny.retroInversion(row)` | both |
| `genny.transposeRow(row, semitones)` | moved |
| `genny.serial(row, forms)` | a looping generator of the forms one after the other; each form is `[op, transpose]`, `op` one of `"prime"`, `"retro"`, `"invert"`, `"retroInvert"` |
| `genny.matrix(row)` | the twelve tone matrix: an array of rows |

```js
const r = genny.row()
const piece = genny.serial(r, [["prime", 0], ["retro", 0], ["invert", 7], ["retroInvert", 7]])
const pitches = piece.map(pc => 48 + pc)
```

---

## Variation and structure

### `genny.mutate(values, { replace = 0, swap = 0, rotate = 0, reverse = 0, with })`

A looping phrase that changes a little every time it has been played. Each option is the **chance (0 to
1)** of that change after each pass:

| option | change |
|---|---|
| `replace` | one note is replaced: by `with()` (a function that makes a value), or, without `with`, by a copy of another note of the phrase |
| `swap` | two notes swap places |
| `rotate` | the phrase rotates one step, left or right |
| `reverse` | the phrase is reversed |

`reset()` restores the original phrase.

```js
const phrase = genny.mutate([60, 62, 64, 67, 64, 62], {
  swap: 0.4, rotate: 0.2, replace: 0.1, with: () => 60 + 2 * Math.floor(Math.random() * 6),
})
```

(For repeatable renders, draw random numbers from genny itself, not `Math.random`; see
[Randomness and seeds](#randomness-and-seeds).)

### `genny.lsystem(axiom, rules, iterations, symbols, loop = true)`

Self-similar phrases from Lindenmayer rewriting. `rules` rewrite each letter (`{ A: "AB", B: "A" }`),
`iterations` is how many times; `symbols` gives each letter its value (`{ A: 60, B: 67 }`).
A letter without a rule stays. The result is a sequence of the symbols, looping.

```js
genny.lsystem("A", { A: "AB", B: "A" }, 5, { A: 60, B: 67 }).take(8)   // the Fibonacci word as notes
```

### `genny.grammar(rules, start, depth = 4)`

A context free grammar with weighted alternatives. **A new phrase is derived every cycle.** `rules` gives
each name its alternatives as `[weight, symbols]`; a symbol that is `{ rule: "name" }` is replaced by
that rule, any other value is output. `depth` limits the recursion. A rule that is never defined or
that cannot end is an error when the grammar is made.

```js
const g = genny.grammar({
  Phrase: [[3, [{ rule: "Motif" }, { rule: "Motif" }]], [1, [{ rule: "Motif" }, 67]]],
  Motif:  [[1, [60, 62, 64]], [1, [64, 62, 60]]],
}, "Phrase")
g.take(12)
```

### Number sequences

| Call | Gives |
|---|---|
| `genny.fibonacci(mod = 12)` | 0 1 1 2 3 5 8 ... modulo `mod` (2 or more): a melody in pitch classes |
| `genny.collatz(start, loop = true)` | the hailstone sequence from `start` (6 3 10 5 16 8 4 2 1), then it ends, or loops |
| `genny.thueMorse()` | 0 1 1 0 1 0 0 1 ... the Thue-Morse sequence |
| `genny.primes()` | 2 3 5 7 11 13 ... |

```js
genny.thueMorse().take(8)                                 // a rhythm that never repeats
genny.fibonacci(7).map(n => 60 + [0, 2, 4, 5, 7, 9, 11][n])
```

### `genny.learn(values, order = 2)`

A Markov chain that learns the style of an example (an array of numbers) and plays new music in it.
`order` is how many previous notes it looks at: 1 is loose, higher repeats the source more.

```js
const bach = [60, 62, 64, 65, 67, 65, 64, 62, 60, 64, 67, 72]
const style = genny.learn(bach, 2)
style.take(16)
```

### `genny.pick(items, distribution = "uniform", ...params)`

An item at random by a curve over the list, to make the first, the last or the middle ones likely.

| distribution | params (defaults) | favours |
|---|---|---|
| `"uniform"` | | all the same |
| `"linear"` | | the first items |
| `"exponential"` | `rate = 3` | the first items, steeper |
| `"gaussian"` | `mean` = the middle item, `sd` = a sixth of the list (at least 1) | the items around `mean`; both are in **items** (0 is the first, `mean = 2, sd = 1` the third and its neighbours) |
| `"beta"` | `a = 0.5`, `b = 0.5` | the ends (a, b below 1) or the middle (above 1) |
| `"cauchy"` | `center = 0.5`, `scale = 0.1` (positions 0 to 1 over the list) | around `center`, with now and then a far pick |
| `"weibull"` | `spread = 1`, `shape = 1` (position 0 to 1) | the start for a shape below 1, around `spread` for a larger one |
| `"triangle"` | `mode = 0.5` (position 0 to 1) | the chance rises to `mode` and falls again |
| `"ratio"` | `...weights` (whole numbers) | the items by those weights |

```js
const low = genny.pick([36, 38, 41, 43, 46], "exponential", 4)
const mid = genny.pick(pitches, "gaussian", 2, 1)            // around the third item
const by  = genny.pick(["a", "b"], "ratio", 3, 1)             // a three times as often
```

### `genny.shifting(items, every, ...weightSets)`

An item by weights; **the weights change every `every` values** to the next set (and around). A melody
that slowly drifts from low to high:

```js
const g = genny.shifting([48, 55, 60, 67], 16, [1, 0, 0, 0], [1, 1, 0, 0], [0, 1, 1, 0], [0, 0, 1, 1])
```

---

## Constraints and sieves

### Constraints on a number generator

These wrap a generator of numbers; when its value breaks the rule they **draw again**, up to `stop`
times (default 100) and then let the last one through, so they never block.

| Call | Rule |
|---|---|
| `genny.allowIntervals(gen, intervals, opts)` | the step from the last value must be one of `intervals` (semitones, absolute) |
| `genny.blockIntervals(gen, intervals, opts)` | the step must not be one of them |
| `genny.noRepeats(gen, opts)` | never the same number twice in a row |
| `genny.noDuplicates(gen, n, opts)` | no number twice in a group of `n` |
| `genny.intervalWalk(lo, hi, intervals)` | a melody (whole numbers) that only moves by the intervals, up or down at random, within `lo`..`hi`; it starts in the middle |

`opts` is `{ stop, pitchClass }`: `stop` the number of tries; `pitchClass: true` compares modulo 12 (an
octave is a repeat or the same interval).

```js
const wild = genny.rand(48, 72).map(Math.round)
const tune = genny.blockIntervals(genny.noRepeats(wild), [6, 11, 1])    // no tritones, sevenths, semitones
const walk = genny.intervalWalk(48, 72, [1, 2, 3, 5])
```

### `genny.sieve(mod, start, lo, hi)` / `genny.sieveOf(lo, hi, values)`

Xenakis sieves: the members of a residue class (`mod 3 from 0`: 0 3 6 9 ...) between `lo` and `hi`, or
a list of numbers. Combine them with logic; use the result as a pitch set, a gate or a filter.

| Member | Does |
|---|---|
| `s.values()` | the members as an array |
| `s.contains(x)` | is `x` a member |
| `s.union(t)`, `s.intersection(t)`, `s.difference(t)`, `s.complement()` | a sieve |
| `s.gen(loop = true)` | a generator of the members, in order |
| `s.gate(length)` | a gate that is `true` on the steps that are members, over `length` steps, looping |
| `s.scale(base)` | the members as pitches above `base` |

```js
const a = genny.sieve(3, 0, 0, 23)
const b = genny.sieve(4, 1, 0, 23)
const rhythm = a.union(b).gate(24)                  // a rhythm in 24 steps
const pitches = a.intersection(b).scale(36)
```

### `genny.funnel(values, allowed)`

Each value moved to the nearest of the allowed values. An array in, an array out.

---

## Lists

### `genny.jump(values, chunk, step, leap)`

A looping read of a list in chunks: from the start position (0) it plays `chunk` values, each `step`
positions after the previous one, then the start moves `leap` positions and the next chunk begins.
Positions wrap around the list; `step` and `leap` may be negative. `jump([0,1,2,3,4,5,6,7], 3, 1, 4)` plays
0 1 2, 4 5 6, 0 1 2, ...

### `genny.stutter(gen, n, k)`

After every `n` values the last one is repeated `k` more times.

```js
genny.stutter([1, 2, 3], 1, 1).take(6)       // 1 1 2 2 3 3
genny.stutter(melody, 4, 2)                  // every fourth note rings three times
```

---

## Numbers over time

### `genny.rand(lo = 0, hi = 1)`

A random number in the range, uniform.

### `genny.rwalk(origin, min, max, step = 0.1, mode = "bounce")`

A random walk from `origin` within `min`..`max`. Each step moves by up to `step` times the range. At an
edge: `"clip"` holds, `"bounce"` turns back, `"jump"` goes back to the origin.

### `genny.ramp(steps, from, to, curve = 1, loop = false)`

`steps` values (2 or more) from one number to the other, along a curve (1 straight, above 1 slow to
start); then it ends, or loops.

```js
const fadeIn = genny.ramp(32, 0, 1, 2)
```

### `genny.lfo(shape = "sine", period = 64, { pulseWidth, unipolar })`

A wave: one cycle in `period` values (steps). Values from -1 to 1, or 0 to 1 with `unipolar: true`.

| shape | |
|---|---|
| `"sine"`, `"triangle"` | smooth |
| `"saw"` | rises from -1 to 1 and falls back at once |
| `"square"` | 1 for the first `pulseWidth` of the cycle, then -1 |
| `"sampleHold"` | holds a new random value for each cycle |
| `"smoothRandom"` | glides to a new random value each cycle |

```js
const vib = genny.lfo("sine", 16, { unipolar: true })
const cutoff = vib.map(v => 400 + 3000 * v)
```

### `genny.slew(gen, rise = 0.1, fall = rise)`

A limit on how fast the numbers of a generator may move: per value at most `rise` up and `fall` down.
Smooths a stepped signal or a random one.

### `genny.noise(step = 0.1, octaves = 1, { unipolar })`

Smooth (Perlin style) noise that drifts: `step` is how far along the noise it moves per value (small
is slow), `octaves` adds finer detail. -1 to 1, or 0 to 1 unipolar.

### `genny.lissajous(a = 3, b = 2, delta = 0, steps = 64)`

Two related modulation signals, `[x, y]` per value, a Lissajous curve with frequencies `a` and `b` and
phase difference `delta` (radians); one round takes `steps` values. Pan and cutoff that move together.

### `genny.logistic(a = 3.9, x0 = 0.5)`, `genny.henon(a = 1.4, b = 0.3)`

The logistic map (0 to 1; chaotic above about 3.57) and the x of the Hénon map (about -1.3 to 1.3).

### Attractors

Each gives an array per value, the state of the system.

| Call | Gives (defaults) |
|---|---|
| `genny.lorenz(sigma = 10, rho = 28, beta = 8/3, dt = 0.01)` | `[x, y, z]` |
| `genny.rossler(a = 0.2, b = 0.2, c = 5.7, dt = 0.05)` | `[x, y, z]` |
| `genny.duffing()` | `[x, y]` |
| `genny.pendulum()` | `[x, y]` |
| `genny.standardMap(k = 1)` | `[x, y]` |

```js
const l = genny.lorenz()
const pitch = l.map(([x, y, z]) => 36 + Math.round(z))          // z runs about 0 to 50
```

---

## Distributions

Random numbers that fall by a curve, as generators of numbers.

| Call | Gives |
|---|---|
| `genny.gaussian(mean = 0, sd = 1, lo, hi)` | a bell curve; with `lo` and `hi` the values stay between them |
| `genny.exponential(mean = 1, lo, hi)` | many small values, a few large |
| `genny.beta(a = 0.5, b = 0.5, lo = 0, hi = 1)` | between `lo` and `hi`: a, b below 1 favour the ends, above 1 the middle |
| `genny.cauchy(location = 0, scale = 1, lo, hi)` | like a gaussian with heavy tails: now and then a wild value |
| `genny.weibull(threshold = 0, spread = 1, shape = 1, lo, hi)` | a skewed spread starting at `threshold` |
| `genny.pink(rows = 8)` | pink (1/f) noise, 0 to 1: slow drifts with detail at every scale; `rows` is the octaves of detail |
| `genny.lehmer(seed = 1)` | a pseudo random sequence 0 to 1 that is **the same every time** for a seed, not tied to `Seed` |

```js
const velocity = genny.gaussian(0.7, 0.1, 0.3, 1)
const drift = genny.pink().map(v => 0.2 + 0.6 * v)
```

---

## Tendency masks

A **mask** is a band `lo`..`hi` that moves over time; the values are drawn inside it (Koenig). It makes
a melody that opens up from a single note, or a density that rises.

### `genny.tendency(segments, { loop = false, beta })`

`segments` is `[{ n, from: [lo, hi], to: [lo, hi] }]`: for `n` values the lower bound goes from `from[0]`
to `to[0]` and the upper from `from[1]` to `to[1]`. Without `to` the band stays at `from`; a segment with
`n = 1` is just the band `from`. At the end the generator is done (it keeps the last band), unless
`loop: true`. `beta: [a, b]` draws toward the edges (a, b below 1) or the middle (above 1) of the band.

```js
// a melody that starts on one note and spreads to two octaves
const open = genny.tendency([
  { n: 16, from: [60, 60], to: [48, 72] },
  { n: 16, from: [48, 72], to: [60, 60] },
], { loop: true }).map(Math.round)
```

### `genny.tendencyChoice(items, segments, { loop, beta })`

The same mask, but it picks **items** of a list: the band is a position in the list, 0 (first) to 1 (last).

### `genny.spray(values, spread)`

An array of values, each moved at random by up to `spread` either way. A chord that is not quite in tune.

---

## Lines of note events

Some generators make **whole lines** instead of values one at a time. A line is an array of
note events, objects `{ pitch, start, duration = 1, velocity = 0.8 }` with `start` and `duration` in
steps (the same unit as the clock steps of the script). You play one by looking up the events that start
on a step:

```js
const theme = [60, 62, 64, 60].map((p, i) => ({ pitch: p, start: i * 2, duration: 2 }))
const phrase = genny.line.canon(theme, 8, 7, 3)[1]            // the second voice of a canon
function trigger(ev) {
  for (const e of phrase) if (e.start === ev.step % 64) note(e.pitch, e.velocity, e.duration)
}
```

### `genny.line.*`: what is done to a line

Each takes events and gives new ones.

| Call | Does |
|---|---|
| `line.transpose(events, semitones)` | moved up or down |
| `line.stretch(events, factor)` | starts and durations times `factor`; `augment` and `diminish` are the same (`factor` above 1 for a slower line, below 1 for a faster) |
| `line.invert(events, axis, scale)` | mirrored around the pitch `axis`; **with a scale** (from `genny.scale`) the mirror is diatonic: degrees are mirrored, so it stays in the scale. Without, chromatic: `2 * axis - pitch` |
| `line.retrograde(events)` | backwards in time (durations stay): the line keeps its first start and its length |
| `line.shift(events, steps)` | later, or earlier when negative |
| `line.merge(...lines)` | together, in order of start |
| `line.slice(events, from, to)` | the notes that start from `from` up to `to`, moved so `from` is 0 |
| `line.loop(events, n, length)` | `n` times one after the other, each cycle `length` steps long |
| `line.quantize(events, scale)` | every pitch to the nearest of a scale |
| `line.canon(events, delay, interval, voices)` | an **array of lines**: the line, then again `delay` steps later and `interval` semitones higher, and so on for `voices` voices |
| `line.thin(events, gate)` | only the notes where a gate generator says `true` (a value for each note) |

```js
const theme = [60, 62, 64, 60].map((p, i) => ({ pitch: p, start: i * 2, duration: 2 }))
const key = genny.scale(60, "major")
const answer = genny.line.invert(genny.line.shift(theme, 8), 62, key)
const both = genny.line.merge(theme, answer)
```

### `genny.toLine(pitches, length = 1)`

Pitches as events one after the other, each `length` steps long.

### `genny.counterpoint(cantus, options)`

A line **against a cantus firmus** (an array of pitches, one whole note each), by the rules of first
species counterpoint: consonances, begin and end on a perfect consonance, no parallel fifths or
octaves, no big leaps. It gives `{ line, solved }`. When no line follows every rule `solved` is `false`
and `line` is the best it found (see `checkCounterpoint`).

| Option | Default | |
|---|---|---|
| `side` | `"above"` | `"above"` or `"below"` the cantus |
| `second` | `false` | `true`: second species, two notes to each note of the cantus (experimental) |
| `scale` | the pitch classes of the cantus | limits the pitches to a scale (`genny.scale`) |
| `maxLeap` | 9 | the largest leap, in semitones |
| `maxDistance` | 19 | the largest distance to the cantus, in semitones |
| `low`, `high` | no limit | the range (only when `high > low`) |
| `budget` | 200000 | search steps before it gives up |

`genny.checkCounterpoint(cantus, line, second = false)` gives the rules a line breaks, as an array of
texts (empty when it is right).

```js
const cantus = [62, 65, 64, 62, 67, 65, 69, 67, 65, 64, 62]       // D dorian
const cp = genny.counterpoint(cantus, { side: "above" })
cp.solved                                                          // true
genny.checkCounterpoint(cantus, cp.line)                           // [] : no rule broken
genny.checkCounterpoint([60, 62], [67, 69])                        // a text for the parallel fifths
const parts = genny.line.merge(genny.toLine(cantus, 4), genny.toLine(cp.line, 4))
```

### `genny.chordTones(chords, bars, options)`

A melody over chords, **one chord to a bar**: chord tones on the strong beats, scale tones between,
moving by small steps. `chords` is an array of chords (each an array of pitches) or a generator of them;
`bars` the number of bars. Gives events.

| Option | Default | |
|---|---|---|
| `scale` | required | the passing tones (`genny.scale`) |
| `barSteps` | 16 | steps in a bar |
| `beatSteps` | 4 | steps in a beat (`barSteps` should be a multiple of it) |
| `maxLeap` | 3 | the largest leap, in scale degrees |
| `low`, `high` | 60, 84 | the range, at least an octave |

```js
const key = genny.scale(60, "major")
const prog = [0, 3, 4, 0].map(d => genny.diatonic(key, d, 3))
const tune = genny.chordTones(prog, 4, { scale: key })
```

### `genny.isorhythm(talea, color, length)`

Ars nova **isorhythm**: a rhythm (the *talea*, durations in steps, `0` a rest) repeated against a row
of pitches (the *color*) of another length, so the two come back together only after
`genny.isorhythmPeriod(talea, color)` notes. Gives events for `length` steps.

`genny.isorhythmSteps(talea, color)` is the same on a grid of steps: `{ pitch, length }`, two generators
to use together, one value for each step: `pitch` gives the pitch on the step a note starts on and
`null` while it sounds, `length` the length of the note that starts there (1 elsewhere).

```js
genny.isorhythm([2, 1, 1, 2, 1], [60, 62, 64, 67], 32)       // talea of 5 against color of 4
genny.isorhythmPeriod([2, 1, 1, 2, 1], [60, 62, 64, 67])     // 20
```

### `genny.phase(pattern, cycles, drift)`

Steve Reich's **phasing**: two voices play the same pattern, the second a little faster (`drift` steps
a cycle, between 0 and the length of the pattern), so it slowly pulls ahead and, after
`genny.phaseCycles(pattern, drift)` cycles, is in unison again. Gives `{ first, second }`, two lines.
`genny.phaseLength(pattern)` is the length of one cycle.

```js
const pat = [64, 66, 71, 73, 74, 66, 64, 73, 71, 66, 74, 73].map((p, i) => ({ pitch: p, start: i, duration: 1 }))
const { first, second } = genny.phase(pat, 12, 0.25)
```

### `genny.learnStyle(events, order = 2, { intervals, ratios })`

Learns **pitches and durations together** from an example line and plays new music in its style (a
Markov chain over note and length). Of notes that sound at once the highest is taken, and a rest after
a note lengthens it. `order` is how many notes it looks back: 1 is loose, 2 or 3 follow the example more
closely (the example needs more notes than `order`).

| Option | |
|---|---|
| `intervals: true` | learns the intervals between the pitches: the style is the same in any key |
| `ratios: true` | learns the ratios of the durations: the same at any speed |

The style is an object: `next()` the next `{ pitch, duration }`, `phrase(n)` `n` of them, `fill(length)`
notes up to a total duration (the last one shortened), `events(n, start, velocity)` `n` notes as events,
`reset()`, `transpose(semitones)` and `scale(factor)` for another style that is moved, or faster or slower.

```js
const example = [60, 62, 64, 65, 67, 65, 64, 62].map((p, i) => ({ pitch: p, start: i, duration: 1 }))
const style = genny.learnStyle(example, 1, { intervals: true })
style.transpose(7).phrase(8)
```

---

## Forms

Whole forms of genny: a fugue, functional harmony in four voices, exact polyrhythm and other tunings.
They make lines of note events (the times in steps, the velocities from 0 to 1) like `genny.line`.
Most of them draw on the random stream, so seed first.

### `genny.fugue(subject, options)`

The exposition of a **fugue**: the voices enter one after the other with the subject and the answer, the voice
before it plays a countersubject. The subject is `{ degrees, durations }` (scale degrees from the tonic, 0 is the
tonic; steps) or `{ pitches, durations }` (all notes of the key). Gives an object to go on with.

| Option | Default | |
|---|---|---|
| `scale` | required | the key, `genny.scale(62, "minor")` |
| `voices` | 3 | |
| `answer` | `"tonal"` | `"tonal"` or `"real"` |
| `stepsPerBar`, `beat` | 16, half a bar | the bar and the grid of accents (steps) |
| `centers`, `order` | top voice at 74, a fifth lower each | the middle pitches of the voices, and the order they enter in |
| `species`, `velocity`, `attempts` | 2, 0.8, 400 | the countersubject |

| Member | Does |
|---|---|
| `f.episode({ voices, repeats = 3, step = -1, shift = 0, fragment })` | a sequence from the head of the subject; gives the shift to aim a middle entry at |
| `f.middle(voice, "subject" or "answer", shift, withCounter = false)` | an entry in a new key area (2 the relative major, 4 the dominant, 3 the subdominant) |
| `f.stretto(distance, voices)` | overlapping entries; `false` (and nothing added) when the distance does not work |
| `f.close({ bars = 4, leadingTone = false })` | the ending: a pedal on the dominant, then the tonic |
| `f.voices()` | the notes of every voice, the highest first: an array of lines |
| `f.end()`, `f.check()` | where it ends (steps); the rules it breaks, as texts |
| `f.sections()`, `f.entries()` | `{ name, start, end }` and `{ voice, start, kind, shift, section }` |
| `f.warning`, `f.hasCounter` | set when no countersubject was found (the voices then rest) |

`genny.strettoDistances(subject, { scale, answer, voices = 2, beat = 4 })` gives the distances at which a stretto works.

```js
const key = genny.scale(62, "minor")
const f = genny.fugue({ degrees: [0, 2, 4, 2, 0, -1, 0], durations: [4, 2, 2, 4, 2, 1, 1] }, { scale: key, voices: 3 })
f.episode({ repeats: 2 })
f.middle(0, "answer", 4, true)
f.close({ bars: 2 })
const notes = []
f.voices().forEach((voice, i) => voice.forEach(n => notes.push({ ...n, channel: i })))   // a channel for each voice
```

### `genny.harmony`

**Functional harmony**: chords with a function (tonic, subdominant, dominant) in phrases that end in cadences, written
in four voices by the rules of four part writing. A key is `"D minor"`, `"C"` (major) or `{ tonic: "D4", minor: true }`.
A chord is `{ name, numeral, function, root, intervals, pitchClasses, of, borrowed, seventh, key }`.

| Function | Does |
|---|---|
| `harmony.generator({ key, phraseLength = 4, cadences, seventh = 0, secondary = 0, borrowed = 0 })` | `g.next()` a chord, `g.phrase(n, cadence = "authentic")` a phrase, `g.setKey(key)`. `cadences` is weights: `{ authentic: 2, half: 1, plagal, deceptive }` |
| `harmony.diatonic(key, degree, seventh = false)` | the chord on a degree (0 is I, 4 is V) |
| `harmony.secondaryDominant(key, degree, seventh)` | the dominant of a degree (`V/vi`), `null` when it has none |
| `harmony.borrowed(key, degree)` | the chord from the parallel key (iv in a major key), `null` when it is the same |
| `harmony.modulate(from, to)` | the chords into another key: a pivot, then the dominant seventh |
| `harmony.realize(chords, { variety = 0, ranges })` | four voices (soprano first) with the smoothest voice leading: `{ voicings: [{ chord, notes: [s, a, t, b] }], violations: [text] }` |
| `harmony.voices(result, { length = 16, velocity = 0.7, start = 0 })` | the voicings as four lines of notes, a chord to `length` steps |

```js
const H = genny.harmony
const key = "A minor"
const chords = H.generator({ key, phraseLength: 4 }).phrase(4, "authentic")
const four = H.voices(H.realize(chords), { length: 16 })      // soprano, alto, tenor, bass
```

### `genny.polyrhythm`

**Exact polyrhythm**: times are beats as fractions, so 3 against 2 never drifts. A time is a number, `"2/3"` or `[2, 3]`.

| Function | Does |
|---|---|
| `polyrhythm.layer(options)` | pulses spread evenly: `{ pulses, span = 1 }`; or `{ every: period }`; or `{ tuplet: [n, m], period = 1 }` (n in the time of m periods). Also `offset`, `pattern` (booleans), `accent` (0 to 1), `pitch = 60`, `lo`, `hi` (velocities, 0 to 1), `gate = 0.5`, `euclid: [steps, pulses, rotation]`, `barlow: { soft, level, divisions }`. `layer.events(from, to, stepsPerBeat = 4)` gives its notes, `layer.period` its pulse |
| `polyrhythm.cycle(layers)` | the beats after which all layers are together again |
| `polyrhythm.realign(layers)` | the beats at which each layer comes back in line with the first |
| `polyrhythm.grid(layers)` | the finest raster all pulses fall on: `{ unit, steps, cells, text }` |
| `polyrhythm.meter(unit, groups)` | an additive meter, 3+3+2: `{ accents, onsets, length, units, signature, layer }` |
| `polyrhythm.modulation(old, new)`, `polyrhythm.ratio(n, m)` | a metric modulation: `m.tempo(bpm)`, `m.factor()`, `m.carryEvents(events)` |

```js
const P = genny.polyrhythm
const three = P.layer({ tuplet: [3, 2], period: 1, pitch: 60 })      // 3 in the time of 2 beats
const two = P.layer({ every: 1, pitch: 67 })
const line = genny.line.merge(three.events(0, 4), two.events(0, 4))
P.cycle([three, two])                                                 // 2 beats
```

### `genny.tuning(options)`

Other tunings, as pitches in fractional semitones (a plugin or a synth that takes a pitch with decimals plays them).
`{ edo: 31 }` or `{ edo: 13, period: 1901.955 }` (a period in cents), `{ just: [1, 9/8, 5/4, 3/2], period }`,
`{ cents: [0, 204, 386], period = 1200 }` or `{ scl: text }` (a Scala file); `root = 60`.

| Member | Does |
|---|---|
| `t.pitch(degree)`, `t.note(degree, octave)` | the pitch of a degree, of a degree in an octave |
| `t.snap(p)`, `t.nearest(p)` | the nearest pitch of the tuning, its degree |
| `t.cent(degree)`, `t.len()`, `t.pitches(lo, hi)` | cents above the root, degrees in a period, the pitches in a range |

## Stochastic music

Xenakis's way: not notes but **distributions** of notes.

| Call | Gives |
|---|---|
| `genny.poisson(rate, length)` | the times (in steps) of events that come at random, `rate` to a step on average, over `length` steps: an array, ascending |
| `genny.poissonInt(mean)` | a whole number by Poisson's law (the number of events in a step) |
| `genny.density(gen)` | a gate whose chance of `true` follows the numbers of a generator: 0 never, 1 always. A rhythm of random hits that thins and thickens |
| `genny.freeStochastic(length, density, meanDuration, pitchMean, pitchSpread, scale)` | a section of "free stochastic music" as events: Poisson onsets (`density` events a step), exponential durations (mean `meanDuration`), Gaussian pitches around `pitchMean` with `pitchSpread` semitones, velocity 0.3 to 0.9. `scale` (optional) quantizes |

### `genny.cloud(options, steps)`

A **cloud** of notes for `steps` steps, as events. In each step the number of notes is Poisson around the
density, the onsets uniform inside the step.

| Option | Default | |
|---|---|---|
| `density` | required | the mean number of notes in a step: a number, a function of the step number, or a generator |
| `pitch` | required | a generator of pitches, or `[lo, hi]` for any pitch between |
| `velocity` | 0.8 | a generator (0 to 1) |
| `duration` | 1 | a generator of lengths in steps |
| `scale` | none | quantizes the pitches to a scale |

```js
const key = genny.scale(48, "majorPentatonic")
const swell = genny.cloud({
  density: step => 0.2 + 3 * Math.sin(Math.PI * step / 64),     // thickens, then thins
  pitch: [48, 84], duration: genny.rand(0.5, 3), scale: key,
}, 64)
```

---

## Game of Life

### `genny.life(width, height, rule = "B3/S23", density = 0, { pattern, x, y })`

A grid of cells that are born and die (a two dimensional cellular automaton on a torus: the edges wrap
around). Use its rows and columns as rhythms and pitches that evolve.

`rule` is in B/S notation (**B**orn with this many neighbours / **S**urvive with): `"B3/S23"` Conway's
Life, `"B36/S23"` HighLife, `"B3678/S34678"` DayAndNight, `"B2/S"` Seeds, or your own. `density` (0 to 1)
fills the grid at random; `pattern` puts a shape at `x`, `y`: `"glider"`, `"blinker"`, `"block"`,
`"rPentomino"` (chaotic for a thousand generations), `"toad"`, `"lwss"` (a spaceship), or an array of
rows with `#` for a live cell (`["##.", ".##"]`).

| Member | Does |
|---|---|
| `l.step(n = 1)` | the next generation |
| `l.alive(x, y)`, `l.set(x, y, alive)` | read or change a cell |
| `l.randomize(density)` | fill at random |
| `l.row(y)`, `l.column(x)` | the cells as an array of booleans |
| `l.population()`, `l.generation()`, `l.width()`, `l.height()` | numbers |
| `l.toString()` | a picture |
| `l.rowGate(y)`, `l.columnGate(x)` | a gate that reads the cells of a row (column) one after the other; the grid takes a step after each pass |
| `l.pitches(y, scale, degrees)` | the pitches of the scale degrees `degrees` (one for each cell of the row): `null` where the cell is dead |
| `l.density()` | a number generator: the part of the cells that is alive (a slow modulation that follows the life) |

```js
const l = genny.life(16, 8, "B3/S23", 0, { pattern: "rPentomino", x: 6, y: 2 })
const key = genny.scale(57, "minor")
const tune = l.pitches(3, key, [0, 2, 4, 5, 7, 9, 11, 12, 0, 2, 4, 5, 7, 9, 11, 12])
const hits = l.rowGate(5)
const brightness = l.density().map(d => 500 + 6000 * d)
```

---

## Evolving a phrase

### `genny.evolve(options, perCycle = 1)`

A **genetic algorithm** for phrases of whole numbers (pitches, or degrees of a scale). A population of
phrases is rated by a fitness, the better ones are crossed and mutated, and the best phrase is played;
after every phrase the population evolves `perCycle` generations, so the music **improves as it plays**.

| Option | Default | |
|---|---|---|
| `population` | 30 | phrases |
| `length` | 8 | notes in a phrase |
| `min`, `max` | 48, 72 | the lowest and highest note |
| `fitness` | required | a function `phrase => number` (higher is better), or a list `[[kind, weight, argument], ...]` |
| `tournament` | 3 | phrases that compete for each parent: higher presses harder |
| `crossover` | 0.9 | the chance that two parents are combined (negative: never) |
| `mutation` | 0.1 | the chance per note that it changes (half by a step up or down, otherwise to any note) |
| `elitism` | 1 | the best phrases that pass unchanged, so the best never gets worse |
| `start` | none | phrases for the first population (the rest is random) |

Fitness kinds: `"smoothness"` (small leaps), `"intervalVariety"` (different intervals), `"arch"` (the
shape rises to the middle and falls), `["contour", weight, [0, 0.5, 1, ...]]` (a shape of values 0 to 1),
`["inScale", weight, scale]`, `["target", weight, phrase]` (as near as possible to a phrase).

The result is `{ player, step(), run(n), best(), phrase(), generation() }`: `player` a generator of the
notes of the playing phrase; `step()` one generation; `run(n)` n generations (gives the best phrase);
`best()` the best phrase now; `phrase()` the phrase being played.

```js
const key = genny.scale(60, "dorian")
const ga = genny.evolve({
  length: 8, min: 60, max: 79, mutation: 0.2,
  fitness: [["smoothness", 1], ["arch", 2], ["inScale", 3, key]],
}, 2)
function trigger(ev) {
  note(ga.player.next(), 0.8, 1)           // better and better phrases
}
```

---

## Shapers

Functions of a number to a number, for turning one generator into another (a unit ramp into two
octaves of scale). `gen.shape(s)` puts one after a generator, `s.shape(x)` calls it on a number, a note
name or every pitch of a chord. They are documented in [scripting.md](scripting.md#shapers).

---

## Randomness and seeds

Every random choice in the generators (`rand`, `rwalk`, `bucket`, `choice`, `markov`, `walk`,
`arpeggio` `"random"`, `or`, `repeat` with two counts, `rests`, `mutate`, `grammar`, `pick`, the distributions, `tendency`, `learn`, `swing` with humanize, `split`, ...) draws from **one stream** that
`psalm.Seed(n)` (the `-seed` flag of the example programs) seeds, so a render with the same seed and
patch repeats. Without a seed every run is different.

Generators that draw **when they are made** (`noise`, `lfo`, `bucket`, `automaton` with `random = true`) take their values from the stream at that moment; make them in the
same order every run (at the top of the script) so the seed gives the same piece.

`genny.lehmer(seed)` is a separate, self contained sequence; `Math.random` is not seeded by anything.

---

## Recipes

**A melody on a euclidean rhythm, in a scale, with swing.**

```js
const key = genny.scale(57, "minor")
const rhythm = genny.euclid(5, 8)
const pitch = key.degrees(genny.rwalk(0, -7, 7, 0.25).map(Math.round)).map(p => p + 12)
const tune = genny.gated(pitch, rhythm)
const sw = genny.swing(0.15)

function trigger(ev) {
  const p = tune.next()
  const delay = sw.next()
  if (p !== null) note(p, 0.8, 1, delay)
}
```

**Chords from a progression with smooth voice leading.**

```js
const key = genny.scale(60, "major")
const prog = genny.progression("pop", 0)
const leader = genny.voiceLeader(48, 72)

function trigger(ev) {
  if (ev.step % 12 !== 0) return
  const c = leader.next(genny.diatonic(key, prog.next(), 4))
  for (const p of c) note(p, 0.6, 12)
}
```

**A phrase that changes, a rhythm that thins and fills.**

```js
const phrase = genny.mutate([60, 63, 65, 67, 70, 67, 65, 63], { swap: 0.5, rotate: 0.2 })
const density = genny.lfo("triangle", 96, { unipolar: true }).map(v => Math.round(8 * (1 - v)))
const gate = genny.pulseGate([4, 2, 2], density)
const line = genny.gated(phrase, gate)
```

**A drifting sound.**

```js
const cutoff = genny.slew(genny.noise(0.05, 3, { unipolar: true }), 0.02).map(v => 300 + 5000 * v)
const pan = genny.lissajous(3, 2, 0, 128).map(([x]) => x)
```

**Serial music.**

```js
const r = genny.row()
const form = genny.serial(r, [["prime", 0], ["invert", 5], ["retro", 0], ["retroInvert", 5]])
const piece = genny.stutter(form.map(pc => 48 + pc), 6, 1)
```
