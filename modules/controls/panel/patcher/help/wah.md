# wah

An **auto-wah**: a resonant filter whose frequency follows the level of the sound. Every note opens the filter as it is played and it closes again as the note dies away: the quack of a funky clavinet or guitar. No pedal and no cable are needed; the playing itself moves it.

## Tutorial

1. Add a synth with plucky notes, a wah and the mixer. Cable the synth's audio out to the wah `in` and its `out` to the mixer. Play a rhythm of short notes.
2. Listen to the quack. **kind** is `band pass` (the classic wah) or `low pass`.
3. **min** and **max** are the frequencies in Hz of the filter for a quiet and a loud sound. Raise max for a brighter peak, lower min for a darker start.
4. **q** is the resonance: more is a sharper, more vocal quack.
5. **gain** (in dB) is the sensitivity: the louder the incoming sound, the more the filter opens. If it hardly moves, raise it; if it stays wide open, lower it.
6. **attack** and **release** (ms) are how fast it follows the level up and down. A short release gives a fast, snappy quack, a long one a slow close. **down** turns it around: a louder note sweeps the filter down.

```example Funky quack
[set]
kind = 0
min = 350
max = 2200
q = 5
gain = 12
attack = 3
release = 120
mix = 1
```

```example Slow low-pass swell
[set]
kind = 1
min = 200
max = 3000
q = 3
gain = 18
attack = 20
release = 600
mix = 1
```

```example Sweeping down
[set]
kind = 0
min = 300
max = 3500
q = 6
down = 1
mix = 0.8
```

7. The settings can also be moved by a mod cable, for instance a [lane](lane.md) to **q** or **mix**.

## Settings

| Setting | Range | What it does |
|---|---|---|
| kind | band pass, low pass | the filter |
| min | 50 to 10000 Hz | filter frequency for a quiet sound |
| max | 50 to 10000 Hz | filter frequency for a loud sound |
| q | 0.5 to 20 | resonance |
| gain | -12 to 36 dB | sensitivity to the level |
| attack | 0.1 to 500 ms | how fast it opens |
| release | 5 to 5000 ms | how fast it closes |
| down | 0 or 1 | a louder note sweeps it down |
| mix | 0 to 1 | 0 dry, 1 only the wah |

## Related

[phaser](phaser.md), [saturate](saturate.md), [lane](lane.md).
