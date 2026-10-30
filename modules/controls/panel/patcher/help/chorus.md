# chorus

A **chorus** thickens a sound: it mixes the input with a few copies, each read through a short delay that moves slowly, so the copies drift a little in pitch and time against the original. One voice sounds like several. Reach for it on pads, keys, strings and clean guitars; it is also stereo, because the right side moves out of phase with the left.

## Tutorial

1. Add a synth, a chorus and the mixer. Cable the synth's audio out to the chorus `in`, and the chorus `out` to the mixer. Play a held chord.
2. Move **mix** from 0 (only the dry sound) up to about 0.4: listen for the sound getting wider and thicker. At 1 only the copies are heard.
3. Move **rate**: slow is a gentle shimmer, fast starts to sound like vibrato. Move **depth**, how far the delay moves: more depth means more pitch wobble.
4. **delay** is the centre delay in milliseconds. Short delays sit closer to a flanger, long ones to a doubler. **voices** is the number of copies per side (1 to 4).
5. **spread** is the stereo width (0 makes both sides move together). **feedback** feeds the copies back in, which gives a metallic, resonant chorus. **tone** is a lowpass on the copies that keeps them from getting bright and harsh.

```example Slow thick chorus
[set]
rate = 0.3
depth = 4
delay = 20
voices = 4
spread = 1
mix = 0.45
```

```example Fast shallow shimmer
[set]
rate = 2.5
depth = 1.5
delay = 10
voices = 2
mix = 0.35
```

```example Metallic, resonant
[set]
rate = 0.8
depth = 3
delay = 8
feedback = 0.6
tone = 5000
mix = 0.5
```

6. A mod cable from a [lane](lane.md), an lfo or a random module to a setting makes it move by itself, for instance to **depth** or **mix**. Settings glide a little when they change, so they do not click.

## Settings

| Setting | Range | What it does |
|---|---|---|
| rate | 0.05 to 10 Hz | speed of the modulation |
| depth | 0 to 10 ms | how far the delay moves either way |
| delay | 1 to 35 ms | the centre delay |
| voices | 1 to 4 | delayed copies per side |
| spread | 0 to 1 | stereo width |
| feedback | 0 to 0.9 | metallic, resonant chorus |
| tone | 500 to 20000 Hz | lowpass on the copies |
| mix | 0 to 1 | 0 dry, 1 only the copies |

## Related

[flanger](flanger.md) (a shorter delay with feedback), [phaser](phaser.md), [lane](lane.md).
