# Blocks

`blocks/...` contains plain DSP building blocks: no sockets or messages, just structs that
process samples. The modules are built from them, and you can use them to build your own.

## fft

`blocks/fft`

| API | Meaning |
|---|---|
| `fft.New(n)` | In place radix-2 complex FFT of size `n` (power of 2, panics otherwise) |
| `Forward(re, im)` | Forward transform |
| `Inverse(re, im)` | Inverse transform, including the 1/n scaling |
| `fft.HannWindow(n)` | Periodic Hann window, suited for overlap-add |

Used by the vocoder and the convolver.

## Filters

### svf

`blocks/filters/svf` - zero-delay feedback state variable filter (12 dB/oct).

| API | Meaning |
|---|---|
| `svf.New(cutoff, resonance, sampleRate)` | Resonance 0-1 |
| `Set(cutoff, resonance)` | Update both |
| `SetFrequencyQ(cutoff, q)` | Set the damping from Q instead (for bandpass based designs) |
| `Tick(x) (low, band, high)` | All three basic outputs at once |
| `Process(x, mode)` | One output: `LowPass`, `BandPass`, `HighPass`, `Notch`, `Peak`, `AllPass` |
| `K()` | Damping factor (1/Q), `band * K()` is a bandpass with 0 dB peak gain |

### ladder

`blocks/filters/ladder` - Moog style 4 pole ladder from four zero-delay feedback one pole stages
with a resolved feedback loop, input saturation and multimode outputs.

| API | Meaning |
|---|---|
| `ladder.New(cutoff, resonance, sampleRate)` | Resonance 0-1 (feedback 0-4) |
| `Set(cutoff, resonance)`, `SetDrive(drive)` | Update settings |
| `Process(x, mode)` | `LowPass24`, `LowPass12`, `BandPass24`, `BandPass12`, `HighPass24`, `HighPass12` |

### biquad

`blocks/filters/biquad` - second order filter with the audio EQ cookbook coefficients.

| API | Meaning |
|---|---|
| `biquad.New(type, cutoff, q, gainDB, sampleRate)` | Types: `LowPass`, `HighPass`, `BandPass`, `Notch`, `Peak`, `LowShelf`, `HighShelf`, `AllPass` |
| `Set(type, cutoff, q, gainDB)`, `SetFrequency(cutoff)` | Update settings |
| `Process(x)` | Filter a sample |

### formant

`blocks/filters/formant` - three parallel resonant bandpass filters tuned to the first three
formants of a vowel (average adult male values).

| API | Meaning |
|---|---|
| `formant.New(vowel, q, sampleRate)` | Vowel 0-4 (a e i o u), fractions morph |
| `Set(vowel, q)`, `SetShift(shift)` | Update settings, shift scales all formant frequencies |
| `Process(x)` | Filter a sample |

## pitchshift

`blocks/pitchshift` - delay line pitch shifter with two crossfading read heads.

| API | Meaning |
|---|---|
| `pitchshift.New(semitones, windowMs, sampleRate)` | Window 30-100 ms |
| `SetSemitones(s)`, `SetRatio(r)` | Shift by semitones or a frequency ratio (2 is an octave up) |
| `Process(x)` | Shift a sample |
| `Latency()` | Average delay in samples |

## blosc

`blocks/oscillators/blosc` - band limited oscillator (polyBLEP / polyBLAMP), which avoids the
aliasing of naive waveforms.

`blosc.New(frequency, sampleRate, phase, pulseWidth, amplitude, waveform)`, then
`Tick(phaseMod, sync, syncPhase)` per sample. `Phase()` returns the current phase (0-1).

| Waveform | Uses pulse width |
|---|---|
| `Sine`, `Cosine` | |
| `Triangle` | |
| `Square` | |
| `Rectangle` | yes |
| `Sawtooth`, `Ramp` (falling saw) | |
| `ModifiedTriangle` | yes |
| `ModifiedSquare` | yes |
| `HalfWaveRectifiedSine`, `FullWaveRectifiedSine` | |
| `TriangularPulse` | yes |
| `TrapezoidFixed` | |
| `TrapezoidVariable` | yes |

## delay

`blocks/delay` - a simple delay line: `delay.New(length)`, `Write(x)`, and fractional reads
`ReadLinear(location)` and `ReadHermite(location)`. Used by chorus, flanger and pingpong.
