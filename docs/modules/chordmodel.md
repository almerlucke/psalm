# chordmodel

`modules/controls/panel/chordmodel` and `patcher/chordmodel.go`, a kind of the [patcher](../controls.md#panel). In: none. Out: `notes` (to hear a progression).

A Markov model of chord progressions (`tools/chordmodel`) with an editor on the page. Kept with the patch.

## The model

A matrix of how likely each chord (roman numerals of a key: `I ii iii IV V vi vii°`) is to follow each other chord.

- **Drag** a cell up or down, use the wheel, or right-click for never.
- **Presets** for styles: pop, minor, jazz, rock.
- **Teach it**: type progressions in numerals; it learns what follows what. With `memory`, it learns which two chords lead to which.
- **Lengths**: how long a chord lasts (½, 1, 2, 4 bars), by weights.
- **End chord**: a chord the progression ends on.

## Progressions

A progression is drawn from the model, a card a chord. On a card: lock it, choose another chord, set its bars.

| Button | |
|---|---|
| new progression | draws all the chords |
| draw the others again | draws only the chords not locked, each fitting the chords on both sides |
| hear | plays it through the synths its notes output goes to |
| write to the harmony | puts it in the [harmony](harmony.md) of the patch: the line of a section, as chords or numerals |

## Go

`chordmodel.Model` (`Fill`, `Reroll` with locks and a one chord lookahead, `Learn`, `LearnText`, `Text`) in `tools/chordmodel`; the module is `chordmodel.Module`.

Tests: `tools/chordmodel`, `chordmodel_test.go`, `TestPageChordModel`.
