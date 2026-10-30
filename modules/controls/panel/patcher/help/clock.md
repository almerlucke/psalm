# clock

Sends a **trigger** on the steps of the patch's tempo, in a pattern. Everything that happens on a beat in a patch starts with a clock: it triggers a [notes](notes.md) module, steps a [sequence](sequence.md), or moves a script. For a clock with its pitches and notes in one module, use a [voice](voice.md).

## Tutorial

1. Add a clock. By default it hits on every step (`x`) at a sixteenth (`division`).
2. Cable its `trigger` output to the `trigger` input of a notes module, and that to a synth. You hear the same note sixteen times a bar.
3. Write a pattern in the window. `x` is a hit, `.` a rest. It loops:

```example Off the beat
[pattern]
x . . x . . x .
[set]
division = 3
```

4. A **roll** is several hits in one step (`x3`); a **group** `[x x x]` squeezes a triplet into one step; `*2` makes a step last two:

```example Rolls and groups
[pattern]
x . [x x x] . x3 . x*2 .
[set]
division = 3
```

5. Let the machine write it. `E(5,16)` spreads 5 hits as evenly as it can over 16 steps; `A(3+3+2)` hits at the start of each group; `CA(30,16)` takes a row of a cellular automaton:

```example A euclidean rhythm that turns
[pattern]
E(5,16,+1)
[set]
division = 3
```

The `+1` makes the hits turn one step further at every pass.

6. Add life: **chance** (all hits), `x?25` (one hit, 25%), `%2/4` (only the second time in every four), **mutate** and **seed** (the pattern changes from pass to pass), **groove** and **swing** (the feel), **drift** (the timing wanders).

## Syntax

| | |
|---|---|
| `x` `.` | a hit, a rest |
| `x3` | a roll of 3 hits in the step |
| `[x x x]` | a group filling one step |
| `*2` | lasts 2 steps (`*0.5` half) |
| `?` `?25` | by chance, 50% or 25% |
| `%2/4` `%!2/4` `%fill` `%first` | conditions |
| `!` | stays when the pattern mutates |
| `E(5,16)` `A(3+3+2)` `M(...)` `CA(30,16)` | functions that write a rhythm |

## Lanes

A value for every step, a lane a line: `prob`, `vel`, `shift`, `len`, `pitch`. A `.` is the default. Each lane goes round on its own length. In the window they are bars to drag:

```example Accents in three against four
[pattern]
x x x x
[lanes]
vel: 100 50 70
[set]
division = 3
```

## Settings

| | |
|---|---|
| division | `1/4`, `1/8`, `1/8T`, `1/16`, `1/16T`, `1/32` (0 to 5) |
| swing | 0.5 (straight) to 0.75 |
| groove | straight, mpc 54/58/62, shuffle, laid back, push, dilla, backbeat |
| chance, drift, mutate, seed, freeze | variation |

With a [song](song.md), the pattern can be written per section (`Verse: x . x .`).
