# phaser

A **phaser** sends the sound through a chain of allpass filters whose frequency an LFO sweeps, and mixes it with the dry sound. The result is notches that move up and down: softer and more vocal than a flanger. Good on electric pianos, strings, pads and guitars.

## Tutorial

1. Add a synth, a phaser and the mixer. Cable the synth's audio out to the phaser `in` and the phaser `out` to the mixer. Play a held, bright chord.
2. Start with **mix** at 0.5, where the notches are deepest. At 1 you only hear the shifted sound, which is a vibrato.
3. Move **rate** for the speed of the sweep. Or set **sync** above 0 to give the length of a sweep in beats, which follows the tempo.
4. **min** and **max** are the lowest and highest frequency of the sweep (even over the octaves). A narrow range sounds like a small wobble, a wide one like a big swoosh.
5. **stages** sets the number of allpass stages: more stages, more notches. **feedback** makes the notches deeper and more resonant; negative values are different in colour.
6. **stereo** moves the sweeps of left and right apart.

```example Classic slow phase
[set]
rate = 0.3
stages = 4
min = 300
max = 2500
feedback = 0.4
mix = 0.5
```

```example Deep, resonant, 8 stages
[set]
rate = 0.15
stages = 8
min = 200
max = 4000
feedback = 0.75
stereo = 0.4
mix = 0.5
```

```example Four beats per sweep
[set]
sync = 4
stages = 6
min = 250
max = 3000
feedback = 0.6
```

7. A mod cable from a [lane](lane.md), an lfo or a random module to **feedback**, **min** or **max** moves the sound while it plays.

## Settings

| Setting | Range | What it does |
|---|---|---|
| rate | 0.01 to 10 Hz | speed of the sweep |
| sync | 0 to 32 beats | one sweep in beats instead, follows the tempo (when above 0) |
| min | 20 to 10000 Hz | low end of the sweep |
| max | 100 to 20000 Hz | high end of the sweep |
| stages | 1 to 12 | allpass stages (4 is the classic, more means more notches) |
| feedback | -0.95 to 0.95 | deeper, more resonant notches |
| stereo | 0 to 1 | how far apart left and right sweep |
| mix | 0 to 1 | 0.5 is the deepest notches |

## Related

[flanger](flanger.md), [chorus](chorus.md), [lane](lane.md).
