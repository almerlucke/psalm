# part

Notes written a **step** each, played on the clock of the patch, with notes that can **follow the chords**. The bass line, the pad, the arpeggio: a figure written once that is right for every chord. Cable its `notes` output to an instrument. See also [voice](voice.md) (fixed pitches) and [harmony](harmony.md).

## Tutorial

1. Add a part and an instrument. The default plays `C3 Eb3 G3 - Bb3 . G3 Eb3`, a note a step. Notes are names or MIDI numbers (`C3` is 48).

| | |
|---|---|
| `C3 Eb3 G3` | a note a step |
| `.` | a rest |
| `-` | the note before held a step longer |
| `C3+Eb3` | a chord |
| `[C3 D3 Eb3]` | a group in one step |
| `*2`, `?`, `^3` | longer, by chance, a ratchet of 3 hits |

2. Follow the chords. Add a [song](song.md) with chords (or a [harmony](harmony.md)). Now write the notes as parts of the chord of the moment:

```example A bass that follows the chords
[notes]
@b . . @1 . . @b . . . @5 . . @b . @1'
[set]
division = 3
register = 36   # the root is the first one at or above this
bass = 28       # the bass note is the first one above this
velocity = 0.8
length = 0.6
```

| | |
|---|---|
| `@1` | the root |
| `@2` `@3` | the next tones of the chord (the third, the fifth) |
| `@4` | the seventh (a triad: the root an octave up) |
| `@b` | the bass |
| `@c` | the whole chord |
| `@v` | the chord voice led from the one before |
| `'` `,` | an octave up, an octave down (`@1'`) |

3. A figure written once follows the chords: `@1 @3 @5 @3` is an arpeggio in any key the harmony goes through.

```example An arpeggio
[notes]
@1 @3 @5 @3 @1' @3 @5 @3
[set]
division = 3
register = 48
velocity = 0.6
length = 0.9
```

4. A pad that holds the chord, each chord led from the one before so the notes move as little as they can:

```example A pad
[notes]
@v*16
[set]
division = 3
register = 48
velocity = 0.5
length = 1
```

5. Per section: a line `Chorus: @1 @3 @5 @3` each; the lines without a section are for the others. **Lanes** (`vel`, `len`, `pitch`...), **mutate**, **groove** and **swing** work as in a [clock](clock.md).

Without a harmony the chord notes rest.
