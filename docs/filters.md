# Filters

`modules/filters`

One stereo `Filter` module that wraps any of the filter [blocks](blocks.md#filters). Each
channel is filtered independently. Cutoff, resonance and vowel glide smoothly over every buffer
(the cutoff in log frequency, so sweeps sound even), so they can be modulated without clicks.

```go
lp := filters.NewLadder(ladder.LowPass24, 600, 0.8, 2.0).AddTo(p).In(synth)
lp.ValueIn(lfo.New(true, 0.25, 200, 3000, lfo.Sine).AddTo(p), 0, 1) // sweep the cutoff
```

## Constructors

| Constructor | Filter | Resonance means |
|---|---|---|
| `filters.NewSVF(mode, cutoff, resonance)` | State variable filter, 12 dB/oct | 0-1 |
| `filters.NewLadder(mode, cutoff, resonance, drive)` | Moog style transistor ladder, 24 or 12 dB/oct | 0-1, self-oscillates near 1 |
| `filters.NewBiquad(type, cutoff, q, gainDB)` | Audio EQ cookbook biquad | Q (0.707 is flat for low and high pass) |
| `filters.NewFormant(vowel, q, shift)` | Vowel (formant) filter | Q of the formants (around 5-20) |

## Modes and types

| Filter | Modes (`mode` / `type` argument and `mode` message) |
|---|---|
| SVF (`svf.Mode`) | `LowPass`, `BandPass`, `HighPass`, `Notch`, `Peak`, `AllPass` |
| Ladder (`ladder.Mode`) | `LowPass24`, `LowPass12`, `BandPass24`, `BandPass12`, `HighPass24`, `HighPass12` |
| Biquad (`biquad.Type`) | `LowPass`, `HighPass`, `BandPass` (0 dB peak), `Notch`, `Peak` (EQ boost/cut), `LowShelf`, `HighShelf`, `AllPass` |
| Formant | vowel position 0-4: a, e, i, o, u (fractions morph between neighbours) |

## Settings

| Setting | Value input | Message | Used by |
|---|---|---|---|
| Cutoff (Hz) | 1 | `cutoff` | SVF, ladder, biquad |
| Resonance / Q | 2 | `resonance` or `q` | all |
| Vowel (0-4) | 3 | `vowel` | formant |
| Mode / type | - | `mode` | SVF, ladder, biquad |
| Drive | - | `drive` | ladder (input saturation, 1 is mild, unity gain for quiet signals) |
| Gain (dB) | - | `gainDB` | biquad peak and shelves |
| Shift | - | `shift` | formant (above 1 a smaller voice, below 1 a bigger one) |

`Clear()` resets the filter state.

## Choosing a filter

- **SVF**: clean, cheap, all-round 12 dB filter with notch and peak modes, great for sweeps.
- **Ladder**: the warm 24 dB sound of classic monosynths, with drive and screaming resonance.
  The band and high pass modes are mixed from the ladder stages.
- **Biquad**: precise EQ shapes (shelves, peaks, notches) for tone shaping a mix.
- **Formant**: makes any bright sound "speak" vowels, morph it with an LFO on input 3.

The subtractive synth has the SVF, ladder and formant filters built into its voices, see
[synths](synths.md#filter).
