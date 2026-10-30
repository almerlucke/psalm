# sequence

Gives a **value** on every trigger, from a list: a melody, a bass line, a set of chords. Cable a [clock](clock.md) to its `trigger` input and its `value` output to the `pitch` input of a [notes](notes.md) module. A [voice](voice.md) has this built in.

## Tutorial

1. Add a clock, a sequence and a notes module, and an instrument. Cables: clock `trigger` to sequence `trigger`, sequence `after` to notes `trigger` (so the value is there before the note sounds), sequence `value` to notes `pitch`, notes to the instrument.
2. Type values in the window. They are numbers (MIDI: 60 is middle C) or note names (`C3` is 48, `F#4`, `Bb2`):

```example Four notes
[values]
C3 Eb3 G3 Bb3
```

3. A chord goes in brackets and plays as one value:

```example A chord and a bass note
[values]
[C3 Eb3 G3] C2 [C3 F3 Ab3] C2
```

4. Change the **order**. `shuffled` plays every value once before any again; `random` any one; `up and down` bounces; `converge` goes from the outside in:

```example Shuffled
[values]
C3 D3 Eb3 G3 Ab3
[set]
order = 1
```

5. A **reset** trigger starts it from the first value, so a pattern can be locked to the bar.

## With a song

Values can be written per section: a line `Verse: C3 Eb3 G3` each, the others for the lines without one. A section starts its values over each time it starts.

## Orders

| order | |
|---|---|
| 0 in order | 1 2 3 4 |
| 1 shuffled | every value once before any again |
| 2 random | any one every time |
| 3 up and down | up, then down without repeating the ends |
| 4 converge | from the outside in |
| 5 alternate | first, last, second, second to last |
| 6 backwards | the list reversed |
