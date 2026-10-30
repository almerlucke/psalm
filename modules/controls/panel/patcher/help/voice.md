# voice

One thing that plays in a song, in one module: a **clock** (when), a list of **pitches** (what) and the **notes** that sound them (how loud, how long). It is a clock, a sequence and a notes module with the cables between them taken out. Its `notes` output goes to an instrument.

Use it for the parts you write by hand: a kick, a hat, a bass figure, an ostinato. For a part that follows the chords, see [part](part.md); for a melody that makes itself, see [melodist](melodist.md).

## Tutorial

1. Add a voice and an instrument (a synth or a drum machine). Cable the voice's `notes` output to the instrument.
2. Open the voice. The **pattern** says when it plays, a step each: `x` is a hit, `.` a rest. With the default `x` it hits on every step.
3. Type a pattern and press Ctrl+Enter:

```example Four on the floor
[pattern]
x . . . x . . . x . . . x . . .
[set]
division = 3   # 1/16: a step is a sixteenth
pitch = 36     # the note it plays (C1)
velocity = 0.9
length = 0.8
```

4. Give the pattern a feel. **groove** moves some steps later and some softer; **swing** delays every second step; **chance** makes every hit by chance.
5. Give it pitches. Without **pitches** the voice plays its one `pitch` setting. With a list it plays the next one at every hit:

```example A bass figure
[pattern]
x . x x . x . x
[values]
C2 C2 Eb2 G2
[set]
division = 3
order = 0      # in order
velocity = 0.8
length = 0.6
```

The list and the pattern have their own lengths, so they drift against each other: a pattern of 8 steps with 4 pitches comes round every 8 steps, a pattern of 8 steps with 3 pitches only every 24. That is how a short figure keeps changing.

6. Let it change. **mutate** drops some hits and adds others from pass to pass (the same **seed** makes the same changes); **freeze** keeps what you like.

```example A hat that wanders
[pattern]
x x x x x x x x
[lanes]
vel: 100 40 70 40
prob: 100 100 100 60
[set]
division = 3
pitch = 42
mutate = 0.3
seed = 7
velocity = 0.5
length = 0.3
```

## Per section

With a [song](song.md) in the patch, any of the three texts can be written per section: a line `Chorus: x x x x` each, and the lines without a section are for the others. With only sections given, the others are silent. So one voice can play a different pattern in every part of the song.

## Lanes

A **lane** gives every step a value of its own: `vel` (velocity, percent), `prob` (chance, percent), `shift` (timing), `len` (length) and `pitch` (semitones added). A lane has its own length and goes round on it, which makes polymeter. In the window they are bars to drag.

## Settings

| | |
|---|---|
| division, swing, groove, drift | the clock's: how long a step is and how it feels |
| chance, mutate, seed, freeze | how much the pattern varies |
| order | the order of the pitches: in order, shuffled, random, up and down, converge, alternate, backwards |
| pitch | the one pitch, when there are no pitches (0 to 127) |
| velocity, length, transpose | the notes': how loud, how long (in steps), semitones added |

The full pattern syntax (rolls, groups, euclidean rhythms `E(5,16)`, conditions `%2/4`) is in [clock](clock.md); the pitches are written as in [sequence](sequence.md); what the notes do is in [notes](notes.md).
