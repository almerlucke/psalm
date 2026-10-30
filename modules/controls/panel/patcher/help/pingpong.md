# pingpong

A **stereo delay** whose echoes bounce between left and right. The echo time is in beats and follows the tempo, also when it changes while playing (it glides like tape). Reach for it on leads, plucks and vocals-like lines to add space and movement without a reverb.

## Tutorial

1. Add a synth, a pingpong and the mixer. Cable the synth's audio out to the pingpong `in` and its `out` to the mixer. Play separate notes with gaps between them.
2. Set **time** in beats: 0.5 is an eighth, 0.75 a dotted eighth, 1 a quarter. The echoes land on the grid of the clock.
3. **feedback** is how much quieter each echo is than the one before. Low gives one or two repeats, high a long trail (at most 0.98).
4. **stereo** at 0 is a real ping pong: everything bounces, starting on the left. At 1 it is a plain stereo delay where each side echoes its own input.
5. **highCut** and **lowCut** (Hz) filter every repeat, so the echoes become darker and thinner and stay out of the way of the dry sound.
6. **mix** blends echoes with the dry sound.

```example Dotted-eighth bounce
[set]
time = 0.75
feedback = 0.4
stereo = 0
highCut = 6000
lowCut = 120
mix = 0.3
```

```example Long dark trail
[set]
time = 1
feedback = 0.7
highCut = 2500
lowCut = 200
mix = 0.35
```

```example Plain stereo slap
[set]
time = 0.25
feedback = 0.2
stereo = 1
mix = 0.2
```

7. A mod cable from a [lane](lane.md), an lfo or a random module to **feedback** or **mix** gives echoes only at chosen moments.

## Settings

| Setting | Range | What it does |
|---|---|---|
| time | 0.0625 to 4 beats | echo time, follows the tempo |
| feedback | 0 to 0.98 | how much quieter each echo is |
| stereo | 0 to 1 | 0 ping pong, 1 plain stereo delay |
| highCut | 500 to 20000 Hz | repeats get darker |
| lowCut | 20 to 2000 Hz | repeats get thinner |
| mix | 0 to 1 | 0 dry, 1 wet |

## Related

[tape](tape.md) (echoes that wobble and saturate), [lane](lane.md), [clock](clock.md).
