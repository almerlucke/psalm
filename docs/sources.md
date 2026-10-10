# Sources

## player

`modules/player`

Plays (parts of) a sound file, at any speed, once or looping. It is also a `poly` voice, see
the `drums` synth in [synths](synths.md).

```go
pl := player.New(player.Config{
	SndFile: sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/John_1-1.wav", 4),
	Loop:    true,
	Speed:   1.0,
	Amp:     0.8,
}).AddTo(p)
```

Use mipmapped sound files (`sndfile.MustMipMapSoundFile`) so playing at high speed doesn't alias.

**Config**

| Field | Meaning |
|---|---|
| `SndFile` | The sound file to play |
| `SndBank`, `SndName` | Alternatively a sound bank and the name of the sound to start with |
| `NumChannels` | Output channels, defaults to the channels of the sound file |
| `StartOffset`, `EndOffset` | Region to play, 0-1 of the file (end 0 means the whole file) |
| `OffsetRange` | Length of the region from the start offset (0-1), overrides `EndOffset` when set |
| `Speed` | Playback speed, 1 is normal, 2 an octave up, negative plays backwards |
| `Amp` | Amplitude |
| `Loop` | Loop the region (playing starts right away) |
| `LoopOnNoteOn` | Loop while a note is held when used as a voice |
| `UseZeroCrossings` | Snap the region to zero crossings to avoid clicks |
| `FadeOnNoteOff` | Seconds a note off takes to fade the sound out and stop it (a choke, like an open hihat cut by the closed one); 0 lets a one shot play to its end |

**Value inputs**

| Input | Meaning |
|---|---|
| 0 | Bang (restart playback) or a message |
| 1 | Speed |
| 2 | Amplitude |
| 3 | Start offset (0-1) |
| 4 | End offset (0-1) |
| 5 | Offset range (0-1) |

**Messages**: `sound` (name in the sound bank), `speed`, `amp`, `startOffset`, `endOffset`,
`startOffsetSec`, `endOffsetSec` (in seconds), `offsetRange`, and `bang` to restart.

As a `poly` voice the note's amplitude sets the level, the note message can set any of the
message keys (for example `"sound"` and `"speed"` per hit), and a note duration ends a loop.

## blosc (oscillator module)

`modules/oscillators/blosc`

A band limited oscillator with one mono audio output, five mono audio inputs for per sample
modulation, and value inputs and messages for its settings.

```go
osc := blosc.New(220.0, 0.0, 0.5, 0.3, bloscc.Sawtooth).AddTo(p) // freq, phase, pulse width, amp, waveform

// a second oscillator hard synced to the first one (sync on its rising edges)
slave := blosc.New(330.0, 0.0, 0.5, 0.3, bloscc.Sawtooth).AddTo(p)
slave.In(osc, 0, blosc.InSync)
```

Audio and value inputs with the same index control the same parameter (the indices are
exported as constants):

| Index | Constant | Audio input (per sample) | Value input |
|---|---|---|---|
| 0 | `InFrequency` | Frequency in Hz (replaces the set frequency) | Set the frequency |
| 1 | `InFM` | Linear FM in Hz added to the frequency (through zero) | - |
| 2 | `InPulseWidth` | Pulse width 0-1 | Set the pulse width |
| 3 | `InSync` | Restart the phase on every rising edge above 0 | Any value restarts the phase once |
| 4 | `InSyncPhase` | Phase (0-1) to restart at on sync | Set the sync phase |
| 5 | `InWaveform` | - | Set the waveform |
| 6 | `InAmplitude` | - | Set the amplitude |

**Messages**: `frequency`, `pulseWidth`, `waveform`, `amplitude`, `syncPhase`, and `sync` to
restart the phase. From code: `SetFrequency`, `SetPulseWidth`, `SetWaveform`, `SetAmplitude`,
`SetSyncPhase` and `Sync`. The waveforms are listed in [blocks](blocks.md#blosc). For a full synth
voice use the [subtractive synth](synths.md#subtractive).

## lfo

`modules/oscillators/lfo`

A low frequency oscillator for modulation. It runs either at control rate, sending values to
value connections (the usual way to modulate module settings), or at audio rate as a mono audio
output.

```go
// sweep a filter cutoff between 200 and 3000 Hz every 4 seconds
filter.ValueIn(lfo.New(true, 0.25, 200, 3000, lfo.Sine).AddTo(p), 0, 1)
```

`lfo.New(controlRate, frequency, min, max, waveform, customTable...)`

| Argument | Meaning |
|---|---|
| `controlRate` | `true`: send a value on value output 0 every control tick, `false`: audio output 0 |
| `frequency` | Rate in Hz |
| `min`, `max` | Output range |
| `waveform` | `lfo.Sine`, `lfo.Triangle`, `lfo.Saw`, `lfo.Square`, `lfo.Rand` (smoothly interpolated random), `lfo.Custom` (uses the lookup table passed as last argument) |

**Value inputs**: 0 frequency, 1 min, 2 max, 3 waveform.
**Messages**: `frequency`, `minValue`, `maxValue`, `waveForm`.

At control rate it follows the clock of its patch: ticks it misses (out of the patch for a while)
go by, so it comes back at the phase it would have had; one added while the patch plays starts at
the phase it would have from the start; a restart starts it at its starting phase.

## granular

`modules/granular`

A generic granular synthesis engine: many short grains, each a slice of a source shaped by an
envelope, started according to parameters from a parameter generator.

```go
g := granular.New[P, S, E](numChannels, sourceFactory, envelopeFactory, grainPoolSize, paramGen)
```

| Part | Role |
|---|---|
| `P` (parameter) | Describes one grain: `Onset()` (samples until the next grain), `Duration()` (seconds), `Amplitude()` |
| `ParameterGenerator[P]` | Produces grain parameters, either per onset (`granular.Onset`: one grain at a time, each telling when the next starts) or per buffer (`granular.PerFrame`: a list of grains for the current buffer). It also receives the granulator's values and messages, so grains can be controlled live |
| `Source[P]` | Produces a grain's audio |
| `Envelope[P]` | Shapes a grain's amplitude |
| `grainPoolSize` | Maximum number of simultaneous grains |

Available sources and envelopes:

| Package | Kind | Grain parameters |
|---|---|---|
| `granular/sources/loop` | Source | `loop.Param`: `Sound`, `Offset`, `LoopRange`, `Crossfade`, `GrainOnset`, `GrainDuration`, `GrainAmplitude`, `Speed`, `Pan` - plays (and loops with a crossfade) a region of a sound file |
| `granular/envelopes/parabolic` | Envelope | none, a smooth parabolic shape |
| `granular/envelopes/trapezoidal` | Envelope | `Attack()`, `Release()` (fractions of the grain), `Smoothness()` |
| `granular/envelopes/adsr` | Envelope | `ADSRSetting()`: an ADSR whose stage durations are relative, scaled to the grain duration |

See `cmd/examples/granular` (`drops`, `heavenly`, `loopup`) for complete parameter generators.
