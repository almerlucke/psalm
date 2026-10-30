# widener

A **stereo widener**. It splits the sound into its middle (what both sides share) and its sides (what differs): more sides is wider, no sides is mono. The low end can be kept in the middle, so a wide pad does not make the bass wobble between the speakers.

## Tutorial

1. Cable a stereo source (a pad, a chorus, a reverb) to the widener's `in` and its `out` to the mixer. A mono source has no sides, so there is nothing to widen: put it after a [reverb](reverb.md) or a chorus.
2. Move **width**: 0 is mono, 1 the sound as it is, 2 twice as wide, up to 3. Listen for the sound spreading beyond the speakers, and for the middle getting thin when it is too wide.
3. **mono below** is a frequency (Hz): below it the sound is kept in the middle. Raise it if the low end of a wide sound feels unsteady.

```example A wide pad
[set]
width = 1.6
monoBelow = 150
```

```example Check in mono
[set]
width = 0
monoBelow = 120
```

4. A mod cable from an lfo or a [lane](lane.md) to `width` makes the sound breathe in the stereo field.

## Settings

| Setting | Range | What it does |
|---|---|---|
| width | 0 to 3 | 0 mono, 1 as it is, 2 twice as wide |
| mono below | 20 to 1000 | Hz, below it the sound is kept in the middle |

## See also

[reverb](reverb.md), [eq](eq.md), [lane](lane.md).
