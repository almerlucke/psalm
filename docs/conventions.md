# Conventions

How psalm modules behave, so modules from different packages fit together.

## Channels

Synths (poly synths and their voices) and effects are **stereo** by default; a `NumChannels` of
0 means stereo. Mono and stereo connect freely: a mono output feeds both sides of a stereo input,
a stereo output into a mono input is mixed down. Sources like `lfo` and `blosc` are mono.

## Units

| Quantity | Unit |
|---|---|
| Volume of a strip or master | dB (0 is unity, -120 is silence) |
| Amplitude of a note, mix, feedback, depth, amount | 0-1 |
| Frequency, cutoff, formant | Hz |
| Pitch | MIDI note numbers (60 is middle C, fractions allowed), shifts in semitones |
| Musical time (note lengths, echo times, swing units) | beats |
| Effect times (pre delay, grain size, window) | milliseconds |
| Envelope times (ADSR) | milliseconds |
| Section and song lengths | bars |
| Loudness | LUFS (EBU R128, see [rendering](rendering.md#meter)) |

## Levels

Synth presets are calibrated: a preset plays a note at amplitude 1 at **-12 LUFS**, or, when
that would push its peaks over -1 dBFS (percussive and sparse sounds, whose attack reaches the
peak first), as loud as it can with peaks at -1 dBFS. So presets of different synths can be swapped
without changing the mix much, and a part at amplitude 1 leaves headroom for a few more. Every synth
has a `Level` setting (a DX patch too) that scales its output after everything else, the presets
set it. The `synths/levels` test measures every preset and keeps them calibrated; when you add a
preset, set its `Level` so it passes. Default configs are not calibrated.

## Messages and settings

Every setting of a config can be changed while playing with a message whose key is the field
name in lower camel case (`Cutoff` is `"cutoff"`, `GrainSize` is `"grainSize"`), sent with
`ReceiveMessage` or as a message to value input 0. Poly synths pass note messages to the voice
that plays the note.

## Value inputs

Value input 0 takes triggers (poly synths) and messages; settings that are worth modulating with
an LFO or a stream (plain numbers) are on inputs from 1:

| Module | Value inputs |
|---|---|
| `subtractive` | 1 cutoff, 2 resonance, 3 pulse width, 4 vowel |
| `fm` | 1 mod depth, 2 feedback |
| `grains` | 1 position, 2 density, 3 scan, 4 spray, 5 grain size |
| `pulsar` | 1 formant, 2 duty, 3 masking, 4 second formant |
| `wavetable` | 1 position, 2 cutoff, 3 resonance |
| `filters` | 1 cutoff, 2 resonance, 3 vowel |
| `granulator` | 1 freeze, 2 delay, 3 density, 4 grain size, 5 pitch |
| `shimmer` | 1 shimmer |
| `chorus` | 1 rate, 2 depth, 3 mix |
| `flanger` | 1 rate, 2 depth, 3 feedback, 4 mix |
| `pingpong` | 1 time (beats), 2 feedback, 3 mix |
| `mixer` strip | 1 volume (dB), 2 pan, 3 mute |
| `mixer` master | 1 volume (dB) |
| `player` | 1 speed, 2 amp, 3 start offset, 4 end offset, 5 offset range |
| `lfo` | 0 frequency, 1 min, 2 max, 3 waveform |

Two older modules number their settings from 0: `pan` (0 position) and `freeverb` (0 wet, 1 dry,
2 room size, 3 damp, 4 width, 5 mode). Their messages follow the convention above.

## Tempo, randomness and silence

Modules read their patch's environment while they play, never global state:

- **Tempo**: `m.BPM()`, so they follow tempo changes of their own patch (an arrangement tempo lane).
  Musical times (echo times, note lengths) are in beats and follow the tempo.
- **Randomness**: `m.Rand()`, the module's own stream from the patch seed, so `psalm.Seed` makes a
  render repeat.
- **Silence**: modules with a tail implement `psalm.Tailer` (`Tail()` in seconds): they rest while
  their input and output have been silent longer than their tail. Effects implement it, a frozen
  granulator returns -1 to keep playing. `psalm.TailInputs` names the inputs that decide the silence
  (the vocoder's carrier).

## Latency

A module whose whole signal comes out late implements `psalm.Latent` (`SignalLatency()` in
samples): the vocoder, a fully wet convolver. The [mixer](mixer.md) lines up the strips playing
into the same group or the master by delaying the earlier ones (`mix.Align`, automatic before the
first buffer); effect bus returns are not compensated, a reverb's latency is a pre delay. Modules
that are only partly late (a convolver mixed with its dry signal) report 0. Outside a mixer, line
signals up yourself with the `latency` module.
