# pitchshift

A **pitch shifter**: it moves the whole sound up or down by a number of semitones and mixes the result with the dry sound. With a shift of 7 and a mix of 0.5 you hear the original with a fifth above it. It has low latency and is smooth on dense material.

## Tutorial

1. Add a synth, a pitchshift and the mixer. Cable the synth's audio out to the pitchshift `in` and its `out` to the mixer. Play a note or a chord.
2. Set **semitones**: 12 is an octave up, -12 an octave down, 7 a fifth, fractions detune slightly. With **mix** at 0 you only hear the original; at 1 only the shifted sound.
3. A mix of 0.5 gives a harmony line that follows what you play. Small shifts (0.1 to 0.3 semitones) at a mix of 0.5 thicken a sound like a detune.
4. On long, exposed notes a slight warble can be heard in the shifted sound; it is mild on chords and drums, and the effect is best for harmonies, octaves and detune.

```example Fifth above
[set]
semitones = 7
mix = 0.5
```

```example Octave down
[set]
semitones = -12
mix = 0.6
```

```example Thin detune
[set]
semitones = 0.2
mix = 0.5
```

5. A mod cable from a [lane](lane.md) or a random module to **semitones** makes the shift change over time. Stepwise values give a moving harmony; the shifted sound slides as the value changes. Put the output into a [granulator](granulator.md) or a delay for more textures.

## Settings

| Setting | Range | What it does |
|---|---|---|
| semitones | -24 to 24 | the shift (fractions allowed) |
| mix | 0 to 1 | 0 dry, 1 shifted |

## Related

[granulator](granulator.md) (also shifts pitch, in grains), [ringmod](ringmod.md) (a shift that detunes the harmonics), [chorus](chorus.md), [lane](lane.md).
