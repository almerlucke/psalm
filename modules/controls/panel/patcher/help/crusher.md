# crusher

A **bit crusher**: it lowers the number of bits (the coarse steps of an old sampler, grainy noise) and the sample rate (samples are held, giving a metallic aliasing). For lo-fi drums, 8-bit leads and broken textures.

## Tutorial

1. Add a synth or drums, a crusher and the mixer. Cable the audio out to the crusher `in` and its `out` to the mixer.
2. Lower **bits** from 24 toward 1: first nothing changes, then a grainy noise appears, and at a few bits the sound turns to harsh steps. Fractions move smoothly between the steps.
3. Lower **rate** (the sample rate the sound is held at, in Hz) from 44100: the highs fold down into metallic, aliased tones. Below a few thousand Hz the sound becomes dark and ringing.
4. **output** (dB) sets the level after the crusher; **mix** blends with the dry sound, so you can add some crunch beneath a clean sound.

```example Lo-fi drums
[set]
bits = 8
rate = 11025
output = 0
mix = 1
```

```example Broken radio
[set]
bits = 5
rate = 4000
output = -3
mix = 0.7
```

```example Gentle grit, parallel
[set]
bits = 10
rate = 22050
mix = 0.35
```

5. A mod cable from a [lane](lane.md), an lfo or a random module to **bits** or **rate** makes the damage change during the song.

## Settings

| Setting | Range | What it does |
|---|---|---|
| bits | 1 to 24 | bit depth (fractions allowed) |
| rate | 100 to 44100 Hz | sample rate the sound is held at |
| output | -24 to 24 dB | level after the effect |
| mix | 0 to 1 | 0 dry, 1 crushed |

## Related

[saturate](saturate.md), [ringmod](ringmod.md), [lane](lane.md).
