# gate fx

The **gate** effect (called `gate fx`, not the [gate](gate.md) of the Song group) is a dynamics gate: below a threshold it turns the level down, all the way to its range (a gate) or by a ratio (a downward expander), with attack, hold and release. It cleans up noise and long tails between notes, and chops a sound short.

## Tutorial

1. Cable a synth (or a drum voice) to the gate's `in` and its `out` to the mixer. Play notes with a tail, or a sound with a steady hiss.
2. Raise **threshold** (dB) from -80: when it passes the level of the quiet parts they are turned down. Listen for the tail of each note being cut off.
3. **range** is how far it turns down (dB): 80 is silence, 12 only a dip. **ratio** at 100 is a hard gate; lower, to 2 or 4, is an expander that turns the quiet parts down gradually.
4. **attack** (ms) is how fast it opens: very short keeps the start of a note, a longer one rounds it. **hold** (ms) keeps it open after the sound falls under the threshold, so it does not chatter. **release** (ms) is how fast it closes: short is chopped, long is a natural fade.

```example Tight gate
[set]
threshold = -35
ratio = 100
range = 80
attack = 0.5
hold = 40
release = 60
```

```example Gentle expander
[set]
threshold = -50
ratio = 2
range = 20
attack = 5
hold = 20
release = 300
```

5. **side cut** is a high pass on the detector (Hz), so low rumble does not hold the gate open. A mod cable from an lfo or a [lane](lane.md) to `threshold` or `range` makes it move.

## Settings

| Setting | Range | What it does |
|---|---|---|
| threshold | -80 to 0 | dB below which it turns down |
| ratio | 1 to 100 | 2 is a 1:2 expander; 100 is a gate |
| range | 1 to 100 | dB, the most it turns down |
| attack | 0.1 to 100 | ms |
| hold | 0 to 500 | ms |
| release | 5 to 2000 | ms |
| side cut | 20 to 500 | Hz, high pass on the detector |

## See also

[compressor](compressor.md), [transient](transient.md), [gate](gate.md) (the Song group gate, which gates notes), [lane](lane.md).
