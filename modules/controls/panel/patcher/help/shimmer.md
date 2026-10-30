# shimmer

A **shimmer** is a reverb whose tail is pitch shifted: every pass through the feedback loop is shifted (an octave up by default), so the tail blooms upward. It suits pads, long notes and ambient music. A low cut keeps the loop clean and the feedback backs off by itself when the tail gets loud, so a long decay stays at a sane level.

## Tutorial

1. Add a pad or a synth and a shimmer. Cable the synth's audio `out` to the shimmer's `in`, and the shimmer's `out` to the mixer.
2. Play a long chord. With **shimmer** at 0 it is a plain reverb. Raise it: more of the shifted signal is fed back and the tail climbs higher with every pass.
3. Change **semitones**: 12 is an octave up, 7 a fifth, -12 an octave down. Use a value that fits the key of the piece.
4. **decay** is the length of the tail in seconds; **mix** is how much of it you hear.
5. **low cut** removes the lows from the loop (in Hz). Raise it if the tail gets muddy.

```example Octave bloom
[set]
shimmer = 0.5
semitones = 12
decay = 8
mix = 0.4
```

```example A fifth up, gentle
[set]
shimmer = 0.3
semitones = 7
decay = 5
lowCut = 400
mix = 0.3
```

6. The reverb settings (**size**, **damping**, **pre delay**, **diffusion**, **mod depth**, **mod rate**, **width**) are those of the [reverb](reverb.md).
7. A mod cable from an lfo, a [lane](lane.md) or a random to `shimmer` or `mix` makes the bloom come and go.

## Settings

| Setting | Range | What it does |
|---|---|---|
| shimmer | 0 to 1 | amount of shifted signal fed back |
| semitones | -24 to 24 | the shift in the loop |
| low cut | 20 to 2000 | Hz, high pass in the loop |
| mix | 0 to 1 | 0 dry, 1 wet |
| size, decay, damping | | as the reverb (decay 0.1 to 30 s) |
| pre delay, diffusion | | as the reverb |
| mod depth, mod rate, width | | as the reverb |

## See also

[reverb](reverb.md), [eq](eq.md), [lane](lane.md).
