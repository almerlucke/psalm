# notes

Turns **triggers into notes** for the instruments it is cabled to. A [clock](clock.md) says when, a [sequence](sequence.md) says which pitch, and this module says how loud and how long. A [voice](voice.md) has all three inside it.

## Tutorial

1. Cable a clock's `trigger` to the `trigger` input of a notes module, and its `notes` output to a synth. Every trigger plays a note on the module's `pitch` (48, a C).
2. The settings:

| | |
|---|---|
| pitch | 0 to 127; used when nothing is cabled to the `pitch` input. A chord plays every note |
| velocity | 0 to 1 |
| length | in steps of the clock that triggered it (at 1/16 and 120 bpm a step is 125 ms) |
| transpose | semitones, added |

3. Cable a value to an input and it wins over the setting: a [sequence](sequence.md) to `pitch` for a melody, a random or a lane (a mod cable) to `velocity` for accents that move.
4. The note sounds at the end of the tick, when the values cabled in are up to date. To be sure a value is there first, cable the sequence's `after` output to this module's `trigger` instead of the clock's.

A trigger carries the velocity of the hit that made it (a clock's `vel` lane, its groove), so accents in the clock reach the notes without a cable.
