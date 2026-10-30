# ringmod

Two effects in one. **ring** mode is a ring modulator: the sound is multiplied with a sine, so every frequency becomes its sum and its difference with the carrier: metallic, bell-like, inharmonic. **shift** mode is a frequency shifter: every frequency moves up or down by the same number of Hz, so the harmonics stop lining up. Use it for bells, robots and strange textures, or slow shifts for a beating, wide sound.

## Tutorial

1. Add a synth, a ringmod and the mixer. Cable the synth's audio out to the ringmod `in` and its `out` to the mixer. Play single notes.
2. In `ring` mode move **freq** (the carrier, in Hz). Low values (a few Hz) give a tremolo-like roughness, hundreds of Hz give a clangorous, metallic tone that no longer follows the note's harmonics.
3. Switch **mode** to `shift`. Now **freq** is the shift in Hz; a negative value shifts down. A few Hz gives a slow beating; larger values detune the sound into something inharmonic.
4. **stereo** is the number of Hz by which the right side differs from the left: a wide, moving image.
5. **mix** blends with the dry sound; around 0.5 you hear both the note and the metal.

```example Metallic ring
[set]
mode = 0
freq = 440
mix = 0.5
```

```example Slow wide shift
[set]
mode = 1
freq = 3
stereo = -6
mix = 0.5
```

```example Robot growl
[set]
mode = 0
freq = 80
mix = 0.8
```

6. A mod cable from a [lane](lane.md) or an lfo to **freq** makes the carrier sweep.

## Settings

| Setting | Range | What it does |
|---|---|---|
| mode | ring, shift | ring modulator or frequency shifter |
| freq | -4000 to 4000 Hz | the carrier, or the shift (negative shifts down) |
| stereo | -50 to 50 Hz | how many Hz the right side differs |
| mix | 0 to 1 | 0 dry, 1 only the modulated sound |

## Related

[pitchshift](pitchshift.md) (keeps harmonics in tune), [crusher](crusher.md), [lane](lane.md).
