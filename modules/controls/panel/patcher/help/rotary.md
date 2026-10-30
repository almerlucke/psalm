# rotary

A **rotary speaker** (a Leslie): inside, a horn for the highs and a drum for the lows turn, so the sound comes closer and goes away. That makes a vibrato (Doppler) and a rising and falling level, picked up by two microphones for a stereo image. Made for organ, electric piano and guitar.

## Tutorial

1. Add a synth (an organ-like sound), a rotary and the mixer. Cable the synth's audio out to the rotary `in` and its `out` to the mixer. Play held chords.
2. **fast** at 0 is the slow, gentle "chorale" speed; at 1 the speakers spin fast ("tremolo"). Switching takes time, as with the real thing: the horn gets there in about a second, the heavy drum in four to five. Listen to it speed up and slow down.
3. **drive** pushes the sound into the tube amplifier in front of the speaker (in dB): 0 is clean, more is gritty.
4. **horn** is the level of the horn against the drum, and **crossover** the frequency in Hz where the sound is split between them.
5. **spread** is the stereo width of the two microphones. **mix** blends with the dry sound (1 is only the speaker).

```example Slow chorale
[set]
fast = 0
drive = 3
horn = 0.5
spread = 1
mix = 1
```

```example Fast with some grit
[set]
fast = 1
drive = 9
horn = 0.6
mix = 1
```

6. A mod cable from a [lane](lane.md) to **fast** switches the speed at chosen moments in the song (values above 0.5 are fast).

## Settings

| Setting | Range | What it does |
|---|---|---|
| fast | 0 to 1 | slow (chorale) or fast (tremolo) speed |
| drive | 0 to 24 dB | tube amplifier in front of the speaker |
| horn | 0.01 to 1 | level of the horn against the drum |
| crossover | 150 to 4000 Hz | where the sound is split between drum and horn |
| spread | 0.01 to 1 | stereo width of the microphones |
| mix | 0 to 1 | 0 dry, 1 only the speaker |

## Related

[tremolo](tremolo.md), [chorus](chorus.md), [saturate](saturate.md), [lane](lane.md).
