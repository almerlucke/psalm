# harmony

`modules/controls/panel/patcher/harmony.go`, a kind of the [patcher](../controls.md#panel) (group Song). In: none. Out: `chord` (value), `bass` (value), `change` (trigger).

The chord track of the patch. One a patch (the last made), like the song. The [parts](part.md) with chord notes, scripts (`harmony()`), the [note editor](noteroll.md) and the [chord model](chordmodel.md) all ask it.

## Writing chords

Chords are typed in the module's window, as symbols (`Dm7`, `F#m7b5`, `Bb/D`, `Csus4`; see [Harmony](../harmony.md)) or as roman numerals in its key and scale (`i`, `VI`, `V7`, `bVII`).

| | |
|---|---|
| `Dm7*2` | the chord lasts 2 bars (`*0.5` half a bar); the default is one |
| `-` | holds the chord before it a bar longer |
| `.` | no chord |
| `\|`, spaces, new lines | between chords |

The chords loop over the bars of a section. With a song, one line per section: `Chorus: F Am Dm Bb`; sections without a line get the lines without a name. Without a song they loop from the start of the patch. A text that can't be read is shown in red and the one before stays.

## Settings

| Setting | Meaning |
|---|---|
| key, scale | what the roman numerals mean |
| low | the lowest note of the chord it sends (the root at or above it); 24 to 84 |

## Outputs

- `chord`: the chord as an array of pitches (root position from `low`), sent when it changes
- `bass`: its bass note, an octave under `low`
- `change`: a trigger at every change

## Go

`HarmonyOf(env)` returns the harmony of a patch; `ChordAt(tick)` the chord at a tick; `Spans(section, beats, beatsPerBar)` the chords of a section as an editor shows them; `ParseChords(text, scale)`.

Tests: `harmony_test.go`.
