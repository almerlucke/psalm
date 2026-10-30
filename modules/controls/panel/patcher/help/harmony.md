# harmony

The **chord track** of the patch. The [parts](part.md) with chord notes, the [melodists](melodist.md) and the [chord model](chordmodel.md) ask it which chord is on. One a patch (the last made). A [song](song.md) has one inside it, with chords per section; add a harmony on its own for a patch without a song.

## Tutorial

1. Add a harmony and type chords, a bar each, as symbols (`Dm7`, `F#m7b5`, `Bb/D`, `Csus4`):

```example A turnaround
[chords]
Dm7 G7 Cmaj7 Fmaj7
```

2. Make a chord last longer with `*2` (two bars) or `*0.5` (half a bar). `-` holds the chord before a bar longer, `.` is no chord:

```example Different lengths
[chords]
Am7*2 Dm7 G7*0.5 G7sus4*0.5 Cmaj7*2 .
```

3. Or write roman numerals, in the key and scale of the settings, so the chords move when you change the key:

```example In a minor key
[chords]
i VI III VII
[set]
key = 9     # A
scale = 11  # minor
```

4. With a song, one line a section: `Chorus: F Am Dm Bb`. The lines without a section are for the others.
5. Let the parts follow: a part with `@1 @3 @5`, a melodist.

## Outputs

| | |
|---|---|
| chord | the chord as an array of pitches, in root position from `low`, sent when it changes |
| bass | its bass note, an octave under `low` |
| change | a trigger at every change of chord |

Cable `chord` to a synth that takes chords, and `change` to anything that should happen on a new chord.

## Settings

| | |
|---|---|
| key, scale | what the numerals mean |
| low | the lowest note of the chord it sends (24 to 84) |
