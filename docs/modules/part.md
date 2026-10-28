# part

`modules/controls/panel/patcher/part.go`, a kind of the [patcher](../controls.md#panel). In: none. Out: `notes`.

Notes written per step, played on the clock of the patch. Typed in the module's window; see also [scripting](../scripting.md#song-lane-gate-and-part).

## Notes

| | |
|---|---|
| `C3 Eb3 G3` | a note a step |
| `.` | a rest |
| `-` | the note before held a step longer |
| `C3+Eb3` | a chord |
| `[C3 D3 Eb3]` | a group in one step |
| `*2` | longer |
| `?` | by chance |
| `^3` | a ratchet: 3 hits in the note |
| `%2/4` | a condition: only the second time through in every four (`%!2/4`, `%fill` in the last bar of a section, `%first`) |

Per section too (`Chorus: …`). Settings: the clock's division, swing, chance, groove and drift, and its **lanes** (a value a step: chance, velocity, timing, length, pitch; as bars to drag in the window, see [scripting](../scripting.md#clock)); velocity, length, transpose.

## Chord notes

Notes that follow the [harmony](harmony.md), so a figure is written once for every chord.

| | |
|---|---|
| `@1` | the root |
| `@3` | the fifth |
| `@b` | the bass |
| `@c` | the chord |
| `@v` | the chord voice led from the one before: each note moves as little as it can, so the same chord shape moves in a smooth line (it stays while the chord does) |
| `'` and `,` | an octave up and down (`@1'`) |

`register` and `bass` settings are the lowest notes they sit at.

Tests: `harmony_test.go` (chord notes), `patcher_test.go`.
