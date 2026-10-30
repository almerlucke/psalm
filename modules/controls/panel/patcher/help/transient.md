# transient

A **transient shaper** finds the start of every hit and its decay, and turns them up or down separately: snappier or softer drums, more or less ring, whatever the level (a threshold does not matter, as it does for a compressor).

## Tutorial

1. Cable a drum voice (or any plucked sound) to the transient's `in` and its `out` to the mixer. Play a pattern with clear hits.
2. Raise **attack** (dB): the start of each hit gets louder, the hits get snappier. Lower it below 0 and the hits get softer, as if the drum were played gently.
3. Raise **sustain** (dB): the decay after the hit gets louder, so there is more body and ring. Lower it and the sound gets drier and shorter.
4. **output** (dB) sets the level after it, **mix** blends the shaped sound with the dry one.

```example Snappy and dry
[set]
attack = 9
sustain = -8
mix = 1
```

```example Softer, more ring
[set]
attack = -6
sustain = 6
output = -2
```

5. A mod cable from an lfo or a [lane](lane.md) to `attack` or `sustain` makes the character change over time.

## Settings

| Setting | Range | What it does |
|---|---|---|
| attack | -24 to 24 | dB, the start of a hit (negative: down) |
| sustain | -24 to 24 | dB, the decay (negative: down) |
| output | -24 to 12 | dB after it |
| mix | 0 to 1 | 0 dry, 1 shaped |

## See also

[compressor](compressor.md), [gate fx](gate%20fx.md), [eq](eq.md), [lane](lane.md).
