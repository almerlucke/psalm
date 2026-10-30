# flanger

A **flanger** mixes the input with a copy delayed by a few milliseconds, and the delay sweeps up and down. The two cancel at some frequencies, so notches sweep through the sound: the jet-plane whoosh. It works best on rich, bright material: drums, guitars, noisy or saw-wave sounds.

## Tutorial

1. Add a synth (or drums), a flanger and the mixer. Cable the synth's audio out to the flanger `in` and the flanger `out` to the mixer.
2. Listen with **mix** at 0.5: the deepest notches come when the dry and delayed sound are equal. Toward 0 you hear less effect, toward 1 only the delayed copy.
3. Move **rate** (speed of the sweep). Or set **beats** above 0: the length of one sweep in beats, which follows the tempo (and replaces rate while it is above 0).
4. **manual** is the shortest delay (where the sweep starts, so the highest notches) and **depth** how far the sweep reaches above it. A small manual with a large depth is a wide sweep.
5. **feedback** is the strength of the resonance: positive rings metallic, negative sounds hollow. Try both signs.
6. **stereo** sets how far the right sweep is out of phase with the left. **shape** is `sine` or `triangle`.

```example Slow jet sweep
[set]
rate = 0.15
manual = 1
depth = 6
feedback = 0.7
stereo = 0.5
mix = 0.5
```

```example A sweep every two bars
[set]
beats = 8
manual = 0.8
depth = 8
feedback = 0.6
mix = 0.5
```

```example Hollow, negative feedback
[set]
rate = 0.3
manual = 2
depth = 3
feedback = -0.8
mix = 0.5
```

7. A mod cable from a [lane](lane.md), an lfo or a random module to **feedback**, **depth** or **manual** makes the effect move on its own.

## Settings

| Setting | Range | What it does |
|---|---|---|
| rate | 0.02 to 10 Hz | speed of the sweep |
| beats | 0 to 32 | length of a sweep in beats; used instead of rate when above 0 |
| manual | 0.1 to 10 ms | the shortest delay |
| depth | 0 to 20 ms | how far the sweep reaches above manual |
| feedback | -0.95 to 0.95 | positive metallic, negative hollow |
| stereo | 0 to 1 | how far the right sweep is out of phase |
| shape | sine, triangle | shape of the sweep |
| mix | 0 to 1 | 0 dry, 1 only the delayed copy; 0.5 makes the deepest notches |

## Related

[chorus](chorus.md), [phaser](phaser.md), [lane](lane.md).
