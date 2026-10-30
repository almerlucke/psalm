# granulator

A **granulator** records its input into a buffer and plays short windowed slices of it, **grains**, with their own pitch, position and stereo place. It turns any sound into clouds, stutters and frozen textures. With **freeze** the recording stops and the grains keep playing the captured moment while the input moves on.

## Tutorial

1. Add a synth, a granulator and the mixer. Cable the synth's audio out to the granulator `in` and its `out` to the mixer. Play a held chord or a melody.
2. **density** is grains per second and **grainSize** the length of each in milliseconds. Many grains of medium size give a smooth cloud; few short grains give a crackling, sparse texture.
3. **delay** is how many ms behind the input the grains read, and **spray** randomly spreads that position. More spray blurs the sound into a wash; none keeps it clear.
4. **pitch** shifts the grains in semitones; **pitchSpread** detunes each grain randomly. **octave** is the chance a grain plays an octave up, **reverse** the chance it plays backwards.
5. **shape** changes the grain window from smooth (0) to percussive (1, a stutter). **jitter** randomizes the time between grains; **stereoSpread** gives each grain a random pan.
6. **feedback** sends the grains back into the buffer: a pitched-up cloud stacks (a fifth up stacks fifths).
7. Set **freeze** to 1 while a chord rings: the chord is held as a texture. Set it back to 0 to follow the input again.

```example Soft cloud
[set]
density = 40
grainSize = 150
delay = 200
spray = 150
stereoSpread = 0.8
mix = 0.5
```

```example Frozen shimmer, octave grains
[set]
freeze = 1
density = 30
grainSize = 200
pitchSpread = 0.2
octave = 0.3
mix = 0.8
```

```example Stutter
[set]
density = 12
grainSize = 60
shape = 1
jitter = 0.1
spray = 10
mix = 0.7
```

8. A mod cable from a [lane](lane.md), an lfo or a random module to **freeze**, **pitch**, **density** or **delay** makes the texture evolve. Freeze is on above 0.5.

## Settings

| Setting | Range | What it does |
|---|---|---|
| freeze | 0 or 1 | stop recording, keep playing the buffer |
| delay | 0 to 4000 ms | how far behind the input the grains read |
| density | 0.1 to 200 /s | grains per second |
| grainSize | 5 to 1000 ms | length of a grain |
| pitch | -24 to 24 | semitones |
| spray | 0 to 1000 ms | random spread of grain positions |
| pitchSpread | 0 to 12 | random detune per grain, semitones |
| octave | 0 to 1 | chance of an octave up |
| reverse | 0 to 1 | chance of playing backwards |
| shape | 0 to 1 | grain window, smooth to percussive |
| jitter | 0 to 1 | randomness of the time between grains |
| stereoSpread | 0 to 1 | random pan per grain |
| feedback | 0 to 0.98 | grains fed back into the buffer |
| mix | 0 to 1 | 0 dry, 1 wet |

## Related

[pitchshift](pitchshift.md), [pingpong](pingpong.md), [lane](lane.md).
