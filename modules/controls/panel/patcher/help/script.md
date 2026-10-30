# script

A module of **JavaScript** for what the other modules can't say: a rule, a chain, a walk. It makes generators once, then reacts to triggers and values. It has inputs `trigger`, `a` and `b`, and outputs `notes`, `a`, `b`, `trigger` and `mod`. Start from a preset (walk, arpeggio, function*, chain, values, song) and change it.

## How it runs

1. When the text is taken (Ctrl+Enter, or **Run**), it is compiled. A syntax error is shown with its line, and the script before it plays on.
2. At the next tick the **top level** runs once: this is where generators, scales and state are made. Running it again starts every generator and counter over.
3. After that, `trigger(ev)` is called on every trigger, `value(v, port)` on every value that comes in on `a` or `b`, and `section(name, index)` when a section of the [song](song.md) starts. Any of them may be left out.
4. Notes made with `note()` are played at the end of the tick.

Scripts run on the audio thread, so keep them short: a call may take 5 ms.

## Tutorial

1. Add a script, a [clock](clock.md) and a synth. Cables: clock `trigger` to the script's `trigger`, the script's `notes` to the synth.
2. A note on every trigger:

```example A note a step
[code]
function trigger(ev) {
  note(pitch("C3"), 0.8, 1)
}
```

`note(pitch, velocity, steps, delay)`: a pitch (a number or a name, or an array for a chord), how loud (0 to 1), how long in steps of the clock, and how late in steps.

3. Make a generator once, ask it for a value each time. A random walk that stays in a range:

```example A walk in the scale
[code]
const minor = scale("C minor")
const walk = genny.rwalk(60, 48, 72, 0.15)   // made once, kept between triggers

function trigger(ev) {
  note(minor.snap(walk.next()), 0.7, 1)
}
```

4. Rhythms too: `genny.euclid(5, 8)` is `true` on 5 of 8 steps.

```example A euclidean rhythm
[code]
const hits = genny.euclid(5, 8)
const pitches = genny.bucket(["C3", "Eb3", "G3"].map(pitch))

function trigger(ev) {
  if (hits.next()) note(pitches.next(), 0.8, 0.5)
}
```

5. Follow the chords. `harmony()` is the chord on now (or `null`); `chord.close(lowest)` gives its notes:

```example Chord tones from the harmony
[code]
let step = 0

function trigger(ev) {
  const c = harmony()
  if (!c) return
  const tones = c.tones(48, 72)
  note(tones[step++ % tones.length], 0.7, 1)
}
```

6. Follow the song. `song.section`, `song.bar`, `song.progress` and `section(name, index)` say where it is:

```example A different note in each section
[code]
const roots = {Verse: "C3", Chorus: "F3"}

function trigger(ev) {
  note(pitch(roots[song.section] || "C3"), 0.8, 1)
}

function section(name, index) {
  log("now", name)
}
```

7. Use more outputs: `send("a", v)` puts a value on output `a`, `mod(v)` sets the mod output (0 to 1) that moves any setting, `fire(delay, count)` sends a trigger (a ratchet or an off-beat), and `clock("1/16")` gives the script a clock of its own so it needs no cable.

## What it knows

| | |
|---|---|
| `note(pitch, velocity, steps, delay)` | play a note or a chord |
| `set(key, value)` | a setting of the synths it is cabled to (`cutoff`, `morph`...) |
| `send(port, v)`, `mod(v)`, `fire(delay, count)`, `clock(division, swing)` | outputs and its own clock |
| `input(port)` | the last value on `a` or `b` |
| `pitch("C3")`, `chord("Dm7")`, `scale("C minor")`, `harmony()` | notes, chords, scales, the chord on now |
| `song` | `section`, `index`, `bar`, `bars`, `progress`, `playing`, `sections` |
| `shared` | an object every script of the patch sees |
| `genny` | generators: `sequence`, `bucket`, `arpeggio`, `markov`, `walk`, `euclid`, `rwalk`, `logistic`, `choice`, `ramp`... each has `next()`, `done()`, `reset()` and `take(n)` |
| `log(...)` | a line in the log of the patch (the **Log** button) |

A generator can also be built from an array (a looping sequence), a function, or a `function*`. The complete list, with every option, is in the scripting reference of the docs.
