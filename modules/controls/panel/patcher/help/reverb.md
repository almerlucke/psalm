# reverb

A **reverb** puts the sound in a space. It is a feedback delay network of eight modulated delay lines: a dense, smooth tail whose length you set in seconds. Reach for it to give a dry synth a room, a plate or a hall, or to push something into the background. For a tail that rises in pitch see [shimmer](shimmer.md).

## Tutorial

1. Add a synth and a reverb. Cable the synth's audio `out` to the reverb's `in`, and the reverb's `out` to the mixer (or the next effect). Play something short, like a pluck: the tail is easier to hear after a short sound.
2. Move **mix**: 0 is only the dry sound, 1 only the reverb. For an effect on a single sound 0.2 to 0.4 is a good start.
3. Move **decay**, the time the tail takes to die away (in seconds, at low frequencies). Short is a room, long is a hall or a cathedral.
4. **damping** makes the highs die faster than the lows: more damping gives a darker, softer tail. **size** is the size of the room.
5. **pre delay** is the time before the reverb starts. A little (20 to 60 ms) keeps the start of a note clear of its own tail.

```example Small room
[set]
decay = 0.6
size = 0.35
damping = 0.6
mix = 0.2
```

```example Big dark hall
[set]
decay = 6
size = 0.9
damping = 0.7
preDelay = 40
mix = 0.35
```

6. **diffusion** is the softness and density of the attack of the tail. **mod depth** and **mod rate** move the delay lines a little, which removes the metallic ringing a long tail can have. **width** is 0 for mono and 1 for full stereo.
7. A mod cable from an lfo, a [lane](lane.md) or a random to a setting makes it move: on `mix` it swells the reverb over a piece.

## Settings

| Setting | Range | What it does |
|---|---|---|
| size | 0 to 1 | room size (the length of the delay lines) |
| decay | 0.1 to 30 | reverb time in seconds |
| damping | 0 to 1 | how much faster the highs decay |
| pre delay | 0 to 490 | milliseconds before the reverb starts |
| diffusion | 0 to 1 | softness and density of the attack |
| mod depth | 0 to 4 | delay modulation in ms |
| mod rate | 0.05 to 5 | modulation rate in Hz |
| width | 0 to 1 | 0 mono, 1 full stereo |
| mix | 0 to 1 | 0 dry, 1 wet |

## See also

[shimmer](shimmer.md), the same reverb with a pitch shifter in its loop; [eq](eq.md) before it to keep low rumble out of the tail; [lane](lane.md).
