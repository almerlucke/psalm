# saturate

**Saturation and distortion**: the sound is driven into a curve that rounds its peaks off, which adds harmonics. A little warms a bass or drums, a lot is fuzz. It runs oversampled, so the added harmonics do not fold back as harsh aliasing.

## Tutorial

1. Add a synth (or drums), a saturate and the mixer. Cable the synth's audio out to the saturate `in` and its `out` to the mixer.
2. Choose the **shape**: `soft` (tape-like, odd harmonics), `tube` (lopsided, even harmonics too), `hard` (clipped, fuzz) or `fold` (folded back: buzzy, changing a lot with the drive).
3. Raise **drive** (dB into the curve) from 0: first the sound gets warmer and denser, then rougher. Around 24 is heavy.
4. Distortion makes the sound louder, so use **output** (dB after the curve) to bring the level back. Lower **tone** (a lowpass after the curve, in Hz) to tame the fizz.
5. **bias** shifts the curve off-centre, which adds even harmonics and a different, more lopsided character.
6. **mix** blends dry and saturated: parallel saturation at 0.3 to 0.5 keeps the punch and adds the grit.

```example Warm bass
[set]
shape = 0
drive = 9
tone = 9000
output = -4
mix = 1
```

```example Fuzz
[set]
shape = 2
drive = 24
tone = 4000
output = -12
mix = 1
```

```example Buzzy fold, parallel
[set]
shape = 3
drive = 18
bias = 0.2
output = -8
mix = 0.4
```

7. A mod cable from a [lane](lane.md), an lfo or a random module to **drive** makes the grit swell during a song.

## Settings

| Setting | Range | What it does |
|---|---|---|
| shape | soft, tube, hard, fold | the curve |
| drive | 0 to 36 dB | how hard the curve is driven |
| bias | -1 to 1 | shifts the curve: more even harmonics |
| tone | 500 to 20000 Hz | lowpass after the curve |
| output | -24 to 12 dB | level after the curve |
| mix | 0 to 1 | 0 dry, 1 saturated |

## Related

[crusher](crusher.md), [tape](tape.md) (saturates its echoes), [rotary](rotary.md) (has a drive), [lane](lane.md).
