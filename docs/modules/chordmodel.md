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
| hear | plays it through the synths its notes output goes to: the chords voice led (each note moves as little as it can) with a bass note under them |
| write to the harmony | puts it in the [harmony](harmony.md) of the patch: the line of a section, as chords or numerals |

## Live

**Live** (a checkbox by the buttons) makes the progression change while the song plays: when the song **leaves the section** the chords are written to (every section when they are written to the whole harmony), the chords that are not locked are drawn again and written to the [harmony](harmony.md), so the next time through that section has other chords (verse 2 is not verse 1), and the editor shows them as they come. A chord that is locked stays. Without a song, **every N bars** (0: the length of the progression). It is drawn off the audio thread and lands on the harmony at the section boundary, never in the middle of a chord. The first time through is the progression you have; every pass after it is drawn from the model (`Live`, `SetLive`, `Drawn` in Go).

## Go

`chordmodel.Model` (`Fill`, `Reroll` with locks and a one chord lookahead, `Learn`, `LearnText`, `Text`) in `tools/chordmodel`; the module is `chordmodel.Module`.

Tests: `tools/chordmodel`, `chordmodel_test.go`, `TestPageChordModel`.
