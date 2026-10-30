# drummachine

A **drum machine** with a kit of samples and a step grid in its window. Seven rows (kick, snare, clap, hat, open, perc, 808) are played by a step sequencer that follows the tempo of the patch. The kind is only there when the drum samples are in the folder `assets/sounds/drums`.

## Tutorial

1. Add a drummachine and cable its `out` to the mixer. Start the patch: a beat is already in the first pattern.
2. Open the module. The grid in its window has a row for each sound and 16 steps (up to 32). Tap a step to switch it, drag to paint, open a step to set its velocity, chance and more.
3. The group **Samples** has a list for each row: choose any file of the folder for it. The choice is kept with the patch.
4. There are 16 patterns. Patterns A to D are in the rotation to start with. Switch pattern in the window, or ask for one from the patch.

## In the patch

- `pattern` (a value) asks for a pattern by its number from 0: a [song](song.md) section number, a [sequence](sequence.md). It switches when the pattern comes round.
- `next` (a trigger) goes to the next pattern of the rotation.

## Feel

| Setting | What it does |
|---|---|
| Swing | Delays every second step. |
| Humanize | Random timing, in ms. |
| Humanize velocity | Random loudness. |
| Fill | Plays the steps made for a fill. |
| level, length, M, S (per row) | The level in dB, the length of the row in steps (rows of different lengths drift against each other), mute and solo. |

The closed hat cuts the open one.

To make drums from oscillators instead of samples, see [drumsynth](drumsynth.md).
