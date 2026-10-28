# supersaw

`synths/supersaw`

The supersaw of the Roland JP-8000: seven detuned sawtooths, the sound of trance pads and leads.
The detune and the mix of the center saw against the others follow the curves Adam Szabo measured
on the original; the saws are band limited, start at random phases, spread over the stereo field
and go through a filter with its envelope.

```go
pad := supersaw.NewPoly(supersaw.TrancePad(), 8).AddTo(p)
```

| Setting | Meaning |
|---|---|
| `Saws` | 0 is 7 |
| `Detune`, `Mix` | 0-1 each, as on the JP-8000 |
| `Stereo` | 0-1, the saws left and right |
| `Sub` | 0-1, a square an octave down |
| `HighPass` | Takes out what lies below the note, as the original does |
| `Cutoff`, `Resonance`, `FilterEnvAmount`, `KeyTrack`, `Velocity` | The filter (Hz, 0-1, octaves, 0-1, octaves) |
| `FilterEnv`, `AmpEnv` | ADSR (ms) |
| `Glide` | Seconds |
| `LFORate`, `LFOSync`, `LFOCutoff`, `LFODetune` | An LFO (Hz, or beats a cycle) on the cutoff (octaves) and the detune |

**Value inputs**: 1 cutoff, 2 detune. Presets: `hypersaw` (sixteen saws, a wall of sound), `wobbleBass`
(the filter in eighth notes), `breathing` (the detune swelling), `swarm` (a four second rise),
`glideLead` (sliding: play it mono legato), `trancePad`, `lead`, `pluck`, `hoover`, `stab`,
`unisonBass`.

**Messages**: `pitch`, `frequency`, `velocity`, `level`, `detune`, `mix`, `stereo`, `saws`, `sub`, `cutoff`, `resonance`, `filterEnvAmount`, and the [pitch bend, mod wheel and aftertouch](../../synths.md#pitch-bend-mod-wheel-and-aftertouch).

## Matrix, ensemble, high pass

An 8 slot [modulation matrix](../subtractive.md#modulation-matrix): sources filter env, amp env, lfo 1 (`lfoRate`,
`lfoCutoff`, `lfoDetune`), lfo 2 (a second sine or triangle, `lfo2Rate`, `lfo2Shape`, running when a
slot uses it), velocity, key, random, wheel and Ext 1-4 (value inputs 5-8); destinations `detune`,
`mix`, `stereo`, `sub`, `cutoff` (octaves), `resonance`, `pitch` (semitones), `level`, `lfoRate` and
`ensemble`.

`ensemble` (0-1) adds a built-in string-machine chorus: three modulated delays of about 7 ms per
channel, swept slowly at `ensembleRate` Hz (default 0.6). It makes a centered voice wide and moves it;
at 0 it is off. `hpCutoff` (20-2000 Hz, 20 is off) is an extra high pass in series before the filter;
the `HighPass` setting that follows the note stays as it was.
