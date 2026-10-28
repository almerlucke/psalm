# melodist

`modules/controls/panel/patcher/melodist.go`, a kind of the [patcher](../controls.md#panel) (group Generators). In: none. Out: `notes`.

A melody that is made **while the song plays**. Where the [note editor](noteroll.md) makes a melody once and keeps it in a clip, the melodist makes a pass of it (a section of the song, else so many beats) at a time, over the chords of the [harmony](harmony.md), in its scale, in phrases that repeat and vary (`tools/melody`). The next pass is made ahead, off the audio thread, so it is there at the bar line.

## From pass to pass

Each pass of a section is made from the pass before it of **the same section**, changed by `mutate`:

| mutate | |
|---|---|
| 0 | the same melody every time through the section |
| about 0.2 to 0.5 | the melody again, changed: some notes move a step or two in the scale, some repeat the note before, some are split in two, some are silent; the notes on the beat are fitted to the chord again, and it ends on the root |
| 1 | a new melody every time |

So the second chorus is the first chorus, varied. The first time a section is played it is made new (from the seed and the name of the section). Without a song a pass is `length` beats. **Freeze** keeps what each section has.

## Settings

| | |
|---|---|
| form, contour, rhythm, phrase | as the Melody panel of the [note editor](noteroll.md): the form (`free`, `AA'`, `ABAB`, `AABA`, `call`, or a **grammar** typed in the box of the window), the shape of a phrase, the rhythm, how long a phrase is |
| busy, rests, smooth, chord tones | how many notes, how often it rests, steps against leaps, how much the notes on the beat keep to the chord |
| low, high, key, scale | the range; the scale is that of the harmony, or a key and scale of its own |
| mutate, seed, freeze | see above; the seed makes the same melody every time |
| second voice | a counter-line goes with the melody (as the roll's Counter-line: consonant, against the melody, no parallel fifths or octaves), to the same synth |
| length | beats a pass without a song |

## Buttons of the window

- **Draw again**: forgets what the sections had, and the seed goes on by one: the next time through each section is a new melody
- **Keep in the roll**: writes the melody that plays into the clip of its section in the first note editor of the patch, to edit it there (the loop without a song)

The tile says the section, the pass and the number of notes (`Chorus · pass 3 · 28 notes`).

## Go

`NewMelodist`, `ReceiveMessage` (the settings by key), `SetGrammar`, `Passes`, `Current`, `Keep`, `DrawAgain`; `melody.Mutate` in `tools/melody` does the changes.

Tests: `melodist_test.go`, `TestMutate` in `tools/melody`.
