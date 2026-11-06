# compressor

A **compressor** turns the level down when the sound gets louder than a threshold, by a ratio. Loud and quiet parts come closer together: drums get tighter, a bass more even, a mix glued. It uses a soft knee, and its detector follows the level, not each wave, so a fast attack does not distort a bass.

## Tutorial

1. Cable a synth (or a drum voice) to the compressor's `in` and its `out` to the mixer. Play something with loud and soft notes.
2. Lower **threshold** (in dB) from 0 and listen: once it is below the loudest notes they get quieter. The louder the sound goes past the threshold, the more it is turned down.
3. Raise **ratio**. At 2 the part above the threshold is halved, at 4 it is a quarter, at 20 it is almost a limiter. Listen for the sound getting flatter and smaller.
4. Move **attack** (ms): a slow attack lets the start of a note through, so drums keep their snap; a fast one catches it and softens the hit. Move **release** (ms): short and the level comes back at once and can pump; long and it stays down longer and sounds smooth.
5. The compression makes the sound quieter: raise **makeup** (dB) to bring it back, or set **auto makeup** to 1 to let it give half the reduction back.
6. **mix** blends in the dry sound (parallel compression): in between 0 and 1 you keep the punch of the dry hits and add the density of the compressed sound.

```example Glue on a drum bus
[set]
threshold = -20
ratio = 3
knee = 6
attack = 25
release = 120
autoMakeup = 1
```

```example Heavy and pumping
[set]
threshold = -30
ratio = 10
attack = 2
release = 80
makeup = 6
mix = 0.7
```

7. **side cut** is a high pass on the detector (Hz): the low end then does not set the compressor to work, so a kick does not pump the rest. A mod cable from an lfo or a [lane](lane.md) to a setting makes it move.

8. **sidechain.** The second input is a sidechain: cable a sound there (a kick) and it, not the sound in `in`, decides when the compressor works. Cable a pad to `in` and the kick to `sidechain`, and every kick pulls the pad down and lets it back over the **release**: the pump of dance music. Set **side cut** low (20 Hz), so the low end of the kick counts, a low threshold and a fast attack, a **release** of 100 to 250 ms, and a ratio of 3 to 6.

```example Ducked by a kick
[set]
threshold = -30
ratio = 4
knee = 6
attack = 2
release = 180
sideCut = 20
```

## Settings

| Setting | Range | What it does |
|---|---|---|
| threshold | -60 to 0 | dB above which it compresses |
| ratio | 1 to 20 | 4 is 4:1; 20 is close to a limiter |
| knee | 0 to 24 | dB, softness around the threshold (0 is hard) |
| attack | 0.1 to 200 | ms |
| release | 5 to 2000 | ms |
| makeup | -12 to 24 | dB of gain after it |
| mix | 0 to 1 | 0 dry, 1 compressed |
| auto makeup | 0 or 1 | gives back half the reduction |
| side cut | 20 to 500 | Hz, high pass on the detector |

## See also

[gate fx](gate%20fx.md), [transient](transient.md), [eq](eq.md), [lane](lane.md).
