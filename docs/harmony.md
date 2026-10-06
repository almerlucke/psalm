# Harmony

`tools/harmony`

Music theory helpers for writing parts with [score](controls.md#score) and
[arrange](controls.md#arrange): chords from symbols, voicings and voice leading, scales and keys
with roman numerals, an arpeggiator, and a check that finds notes clashing with their chords.

## Chords

```go
c := harmony.MustChord("Bbmaj7")          // or ParseChord, which returns an error
chords := harmony.MustProgression("Dm Bb | Gm A")
c.Pitches(58)                             // root position from Bb3: 58 62 65 69
```

A symbol is a root (`D`, `Bb`, `F#`), a type and an optional bass (`G/B`, `Am/G`). Types: none
(major), `m`, `dim` (`°`, `o`), `aug` (`+`), `sus2`, `sus4`, `5`, `6`, `m6`, `7`, `maj7` (`M7`,
`Δ`), `m7`, `mMaj7`, `dim7`, `m7b5` (`ø`), `aug7`, `7sus4`, `9`, `maj9`, `m9`, `add9`, `madd9`,
`7b9`, `7#9`, `11`, `m11`, `13`.

| Function | |
|---|---|
| `c.PitchClasses()`, `c.Contains(pitch)` | The tones of the chord, in any octave |
| `c.Pitches(root)` | Root position from a MIDI root, a slash chord with its bass below |
| `c.Close(lowest)` | Root position with the root at or above a pitch |
| `c.BassPitch(lowest)` | The bass note at or above a pitch |
| `Inversion(pitches, n)` | Move the lowest note up an octave n times (negative: the highest down) |
| `Drop2(pitches)` | An open voicing: the second highest note down an octave |
| `VoiceLead(prev, chord)` | The chord for the same voices with the least movement, using every chord tone when there are enough voices |

```go
voices := harmony.MustChord("Dm").Close(62) // D4 F4 A4
for _, c := range harmony.MustProgression("Bb Gm A Dm") {
	voices = harmony.VoiceLead(voices, c) // D F Bb, D G Bb, C# E A, D F A
}
```

## Scales and keys

```go
key := harmony.MustKey("D minor")    // or MustScale("D", "minor")
key.Pitch(1, 4)                      // degree 1 in octave 4: D4 (62), degree 8 is an octave up
key.Snap(66)                         // the nearest pitch in the key: F4
key.Step(62, 2)                      // two steps up from D4: F4
key.Triad(6).Name                    // "Bb"; Seventh(5) is "Am7" (and "A7" in harmonic minor)
key.MustProgression("i VI iv V7")    // Dm Bb Gm A7
```

Modes: `major` (`ionian`), `minor` (`aeolian`), `harmonic minor`, `melodic minor`, `dorian`,
`phrygian`, `lydian`, `mixolydian`, `locrian`, `major pentatonic`, `minor pentatonic`, `blues`,
`whole tone`, `chromatic`. Chord names are spelled for the key (flats in D minor).

**Roman numerals** count the degrees of the key's own scale (in D minor `VII` is C): upper case is
major, lower case minor, `°` (`o`, `dim`) diminished, `ø` half diminished, `+` augmented, with an
optional type after it (`V7`, `IVmaj7`, `ii7`, `Vsus4`, `vii°7`). An accidental in front moves the
degree: `#vii°` is C#dim in D minor, `bII` is Eb.

## Arpeggiator

```go
arp := harmony.Arp{Pattern: harmony.UpDown, Octaves: 2, Beats: 0.25, Velocities: []float64{1, 0.6, 0.8, 0.6}}
notes := arp.Notes(harmony.MustChord("Am").Close(57)) // score notes
```

| Setting | Meaning |
|---|---|
| `Pattern` | `Up`, `Down`, `UpDown`, `DownUp` (without repeating the turning notes), `OutsideIn`, `RandomWalk` |
| `Octaves` | The chord repeated in this many octaves (0 is 1) |
| `Steps` | Number of notes, 0 is one pass through the pattern |
| `Beats` | Length of every note (0 is 0.5) |
| `Hold` | Beats every note sounds, 0 is the legato of the part |
| `Velocities` | Velocities in turn, for accents |
| `Seed` | For `RandomWalk`, the same seed gives the same arpeggio |

## Checking a part against its chords

```go
timeline := harmony.Timeline(4, harmony.MustProgression("Dm Bb Gm A")...) // a chord per bar
issues := harmony.Check(melody, timeline)
fmt.Print(harmony.Report("melody", issues))
```

`Check` finds notes that clash with the chord under them: a note a **semitone above a chord tone**
(an "avoid note", C# over C), unless it is a passing or neighbour note (at most a beat long, in a
single line, reached or left by a step). Sevenths, ninths and sixths are colour, not clashes; put
them in the chord symbol (`Dm9`) when the chord should hold them. A note **held into the next
chord** also clashes when it ends up a semitone from a chord tone, unless it only rings in for a
moment (a quarter beat). When the part is longer than the timeline, the timeline repeats.

Build a timeline for a whole song with `harmony.TimedChord{Beat, Beats, Chord}` per bar of the
form, and check every part with `form.Compile(name, score)`, as the showcase does at its start.
