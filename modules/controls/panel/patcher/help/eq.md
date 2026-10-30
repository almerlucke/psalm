# eq

A channel **equalizer**: a low cut, a low shelf, two peaking bands, a high shelf and a high cut. Use it to take the mud out of a pad, to bring a lead forward, or to darken a sound. Changes glide, so the settings can be moved by cables.

## Tutorial

1. Cable a synth to the eq's `in` and its `out` to the mixer. Play a sound with a lot in it, such as a saw pad. All gains start at 0 dB, so nothing changes yet.
2. Raise **low cut** (Hz): the lows go. Listen for the sound getting thinner. The cut is 12 dB per octave. Lower **high cut** from 20000 for a darker sound.
3. Each band has a frequency and a gain (dB); the two mid bands also have a **Q**, the narrowness of the bell (high is narrow). Set **mid 2 gain** to +6 and sweep **mid 2** (frequency): you hear the part of the sound it lifts. Use a negative gain to find and remove what you do not like.
4. **low** and **high** are shelves: everything below or above the frequency is lifted or lowered. **output** sets the level after the eq.

```example Clear a pad
[set]
lowCut = 180
mid1Freq = 400
mid1Gain = -3
mid2Freq = 3000
mid2Gain = 3
mid2Q = 1.2
highFreq = 10000
highGain = 2
```

```example Dark and warm
[set]
lowGain = 3
lowFreq = 150
highCut = 4000
highGain = -4
```

5. A mod cable from an lfo or a [lane](lane.md) to a frequency makes a filter sweep.

## Settings

| Setting | Range | What it does |
|---|---|---|
| low cut | 20 to 1000 | Hz, high pass (20 is off) |
| low, low gain | 20 to 1000 Hz, -18 to 18 dB | low shelf |
| mid 1, mid 1 gain, mid 1 Q | 40 to 18000 Hz, -18 to 18 dB, 0.1 to 10 | first bell |
| mid 2, mid 2 gain, mid 2 Q | as mid 1 | second bell |
| high, high gain | 1000 to 18000 Hz, -18 to 18 dB | high shelf |
| high cut | 1000 to 20000 | Hz, low pass (20000 is off) |
| output | -24 to 24 | dB after the eq |

## See also

[reverb](reverb.md), [compressor](compressor.md), [widener](widener.md), [lane](lane.md).
