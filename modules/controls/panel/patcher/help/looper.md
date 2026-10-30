# looper

A **live looper for notes**. What comes in (a keyboard, a note editor) goes on to the synths as it is, and is recorded into a loop of a few bars on the grid of the patch, so it is always on the beat and follows the tempo. Layers play together on every pass, and you can record a new one on top as often as you like.

## Tutorial

1. Add a keyboard, a looper and a synth. Cables: keyboard `notes` to looper `notes`, looper `notes` to the synth. You can play through it already.
2. Set the **bars** of the loop (1 to 16, in 4/4) and **quantize** the notes you play to a grid (off, 1/4, 1/8, 1/16, 1/32), so the loop is tight even when you are not:

```example A two bar loop, tight
[set]
bars = 2
quantize = 3   # 1/16
play = 1
```

3. Press **record**. The layer starts at the start of the next pass (the side panel counts down the beats) and ends with the pass after you switch record off. Play something.
4. The layer plays from now on. Record another on top: bass first, then chords, then a tune.
5. **undo** takes the last layer away; **clear** takes all of them; the side panel has a button on every layer to take that one away while the others go on. **play** switches the loop off and on.
6. Keep it. **export** writes the layers to `recordings/loop-<time>.mid` (a track a layer, the tempo of the patch, the loop repeated **repeats** times). **script** writes a script for each layer (or one file for all, **scriptFiles**) that plays its notes from a script module, to edit as you like.

The loop is on the grid of the patch, so with a [song](song.md) it stays on the beat when the tempo changes. For a loop you edit by hand, use a [noteroll](noteroll.md).
