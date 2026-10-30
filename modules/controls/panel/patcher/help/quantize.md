# quantize

Moves every value it gets to the **nearest note of a scale**. A random or a lane in, a melody in the key out. A chord is quantized note by note.

## Tutorial

1. Add a [sequence](sequence.md) with some values, a [clock](clock.md), a quantize and a [notes](notes.md) module. The sequence's `value` goes to the quantize's `in`, and its `out` to the `pitch` of the notes. Whatever the sequence says, you hear notes of the scale.
2. Choose the **key** (C, Db, D, ...) and the **scale** (minor, major, dorian, blues, pentatonic, whole tone, ...):

```example A minor pentatonic
[set]
key = 9        # A
scale = 12     # minor pentatonic
```

3. Feed it something that wanders: a random pitch, or a lane (`value` output) rising over a section. The result is always in the key, so a pitch that drifts everywhere still sounds like it belongs.
4. Chromatic (the scale `chromatic`) leaves every value as it is, rounded to a semitone.

For melodies that keep to the chords as well as the scale, see [melodist](melodist.md) and [part](part.md).
