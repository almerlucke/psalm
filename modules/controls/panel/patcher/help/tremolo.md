# tremolo

A **tremolo** moves the level of the sound up and down; in **pan** mode it moves the sound from left to right instead (an auto-pan). Synced to the tempo it pulses on the beats of the patch: a chopped pad, a rhythmic gate, a slow sweep across the stereo field.

## Tutorial

1. Add a synth (a held pad is good), a tremolo and the mixer. Cable the synth's audio out to the tremolo `in` and its `out` to the mixer.
2. Play a long chord. With **sync** at 0.5 the level pulses every half beat. **sync** is the length of one cycle in beats (0.25 is sixteenth notes, 2 is two beats), and it follows the tempo.
3. Move **depth**: at 1 the sound goes down to silence on every cycle, lower is a gentler wobble.
4. Choose the **shape**: `sine` (smooth), `triangle`, or `square` (on and off, rounded so it does not click).
5. For a speed that does not follow the tempo set **rate** in Hz (setting it switches **sync** to 0).
6. Switch **mode** to `pan` and the sound travels left and right. In tremolo mode **stereo** moves the left and right apart (at 0.5 they take turns).

```example Sixteenth chop
[set]
mode = 0
shape = 2
sync = 0.25
depth = 0.9
```

```example Slow auto-pan
[set]
mode = 1
shape = 0
sync = 4
depth = 0.7
```

```example Vintage amp wobble
[set]
mode = 0
shape = 0
sync = 0
rate = 5
depth = 0.4
```

7. A mod cable from a [lane](lane.md), an lfo or a random module to **depth** brings the effect in and out over a song.

## Settings

| Setting | Range | What it does |
|---|---|---|
| mode | tremolo, pan | moves the level, or the place |
| shape | sine, triangle, square | shape of the movement |
| rate | 0.05 to 20 Hz | speed in Hz (switches sync off) |
| sync | 0 to 4 beats | one cycle in beats, follows the clock; 0 is off |
| depth | 0 to 1 | down to silence at 1, or all the way to the sides |
| stereo | 0 to 1 | tremolo: how far apart left and right are |

## Related

[rotary](rotary.md) (a tremolo with a Doppler effect), [lane](lane.md), [clock](clock.md).
