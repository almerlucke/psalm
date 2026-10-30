# lfo

A **low frequency oscillator**: a slow wave from 0 to 1 that you cable to a setting (a filter cutoff, a pan, a level) to make it move. Its `out` is a **mod** output. See [random](random.md) for a source that jumps instead of waving, and [lane](lane.md) for curves that follow the form of a song.

## Tutorial

1. Add an lfo and a module with a setting to move, for instance an effect such as [chorus](chorus.md) or a synth. Draw a cable from the lfo's `out` to the module's mod input and choose the setting it moves (the inspector under the patch lists them).
2. Choose how much it moves: the cable has a **range**, a low and a high end of the setting that the 0 to 1 of the lfo is mapped onto. A narrow range is a gentle sway, a wide one a sweep.
3. Set the speed in Hz with **rate** (0.01 to 40), or tie it to the tempo with **beats**: the number of beats of one cycle (4 is a bar of 4/4, 0.5 an eighth note). With **beats** above 0 the rate is ignored, and it keeps time with the song when the tempo changes.

```example One cycle a bar
[set]
beats = 4
wave = 1       # triangle
phase = 0
```

4. Choose the **wave**: sine, triangle, saw up, saw down or square. A square switches between the two ends (a tremolo that chops), a saw up rises and falls back.

```example A slow sine sweep
[set]
rate = 0.2
beats = 0      # use the rate
wave = 0       # sine
```

5. **phase** (0 to 1) is where in its cycle it starts. Two lfos with the same rate and different phases move two settings against each other, for instance the left and right of a wide sound.

```example A square, twice a beat
[set]
beats = 0.5
wave = 4       # square
phase = 0.25
```

## Settings

| | |
|---|---|
| rate | Hz, 0.01 to 40 |
| beats | the length of a cycle in beats (0 to 32); 0 uses the rate |
| wave | sine, triangle, saw up, saw down, square |
| phase | 0 to 1, where a cycle starts |
