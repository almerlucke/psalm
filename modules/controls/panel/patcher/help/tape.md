# tape

A **tape echo**: every repeat is a little darker, warmer (filtered and saturated on the way round) and wavers with the wow and flutter of the tape. The echo time is in beats and follows the tempo. With feedback above 1 the echoes build up into the singing self-oscillation of a dub delay (the saturation keeps it in bounds).

## Tutorial

1. Add a synth (a lead or short notes), a tape and the mixer. Cable the synth's audio out to the tape `in` and its `out` to the mixer. Play a few separate notes.
2. Set **time** in beats: 0.5 is an eighth, 0.75 a dotted eighth, 1 a quarter. It follows the tempo; moving it while playing glides like a tape speed change.
3. **feedback** is how much of each echo returns: low gives a few repeats, higher a long tail. Above 1 the echoes grow into self-oscillation; turn it back down to let it die.
4. **highCut** and **lowCut** (Hz) make each repeat darker and thinner. **drive** (dB) is how much the repeats are saturated.
5. **wow** is the slow wavering of the tape pitch and **flutter** the fast trembling. At 0 the echo is steady; high values sound like a worn tape.
6. **mix** blends the echoes with the dry sound.

```example Warm dotted-eighth echo
[set]
time = 0.75
feedback = 0.45
wow = 0.3
flutter = 0.2
highCut = 4500
lowCut = 150
mix = 0.25
```

```example Dub echo
[set]
time = 0.75
feedback = 1.05
wow = 0.5
highCut = 2500
lowCut = 300
mix = 0.4
```

```example Worn tape slapback
[set]
time = 0.25
feedback = 0.25
wow = 0.7
flutter = 0.6
drive = 6
mix = 0.3
```

7. A mod cable from a [lane](lane.md) to **feedback** or **mix** lets you throw echoes onto chosen notes (the classic dub move). Moving **time** by cable makes the echoes bend.

## Settings

| Setting | Range | What it does |
|---|---|---|
| time | 0.0625 to 4 beats | echo time, follows the tempo |
| feedback | 0 to 1.2 | repeats; above 1 they build up |
| wow | 0 to 1 | slow wavering of the tape |
| flutter | 0 to 1 | fast trembling of the tape |
| drive | 1 to 24 dB | saturation on every repeat |
| highCut | 500 to 20000 Hz | repeats get darker |
| lowCut | 20 to 2000 Hz | repeats get thinner |
| mix | 0 to 1 | 0 dry, 1 wet |

## Related

[pingpong](pingpong.md) (a cleaner stereo delay), [saturate](saturate.md), [lane](lane.md).
