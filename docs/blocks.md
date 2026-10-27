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
| `fft.NewReal(n)` | FFT of real signals of size `n`: a complex FFT of half the size, untangled, about twice as fast; it returns the bins 0..n/2 (the others mirror them) |
| `Forward(x, re, im)` | `n` samples into `n/2+1` bins |
| `Inverse(re, im, x)` | The bins back into `n` samples, scaled so it undoes `Forward` |
| `fft.HannWindow(n)` | Periodic Hann window, suited for overlap-add |

Used by the vocoder and the convolver (the real FFT).

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
| `SetStageLevel(level)` | Where the soft clip of the stages carries up to (4 by default, about 1 clips a normal signal) |
| `SetOversample(factor)` | Runs the ladder 2 or 4 times a sample (`blocks/oversample`), so what the saturation makes above the Nyquist frequency does not fold back: 6 to 20 dB less alias on a driven ladder; the lowpass delays the sound by about a millisecond |
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

## mix

`blocks/mix` - what effects share to blend and change their settings. `mix.EqualPower(m)` returns
the dry and wet gains of an equal power crossfade (they keep the loudness when the wet signal
doesn't correlate with the dry one). `mix.LinearToEqualPower(m)` converts the mix of a linear
crossfade into the equal power mix with the same balance, plus the gain that makes it as loud.
`mix.NewSmoother(value, ms, sampleRate)` glides a setting: `Next(target)` every sample
(`mix.SmoothMs`, 30 ms, is the default speed).

## sine

`blocks/sine`

A fast sine for audio loops: a table of one cycle read with linear interpolation, accurate to
about 3e-7 (-130 dB) and about 2.5 times faster than `math.Sin`. `sine.Cycle(phase)` takes the
phase in cycles (1 is a full turn), `sine.Sin(x)` and `sine.Cos(x)` radians. The FM and DX
operators, the oscillators, LFOs, windows and pulsar pulses use it; use it where a sine is computed
every sample, keep `math.Sin` for coefficients and tables.

## expression

`blocks/expression`

What a player's hands do to a sound while it plays: the pitch bend, the mod wheel and the
aftertouch. Every synth but the piano keeps one per voice (see
[synths](synths.md#pitch-bend-mod-wheel-and-aftertouch)); use it in a synth of your own:

```go
type Voice struct {
	// ...
	expr expression.Expression
}

func (v *Voice) ReceiveMessage(msg psalm.Message) {
	v.expr.Receive(msg) // "pitchBend" -1 to 1, "bendRange" semitones, "modWheel" and "pressure" 0 to 1
	// ...
}

// every sample:
v.expr.Tick(sr)
freq := v.freq * v.expr.Ratio()              // the bend
cutoff := v.cutoff * math.Exp2(2*v.expr.Wheel()) // the wheel (and the aftertouch) open the filter
```

| API | Meaning |
|---|---|
| `Receive(msg)` | Takes `pitchBend`, `bendRange`, `modWheel` and `pressure` from a message |
| `Tick(sr)` | Glides a sample towards where they were moved (a few milliseconds: they never zip or click) |
| `Settle()` | Puts them there at once (for a voice that reads them once a buffer) |
| `Ratio()`, `Semitones()`, `Bend()` | The bend now: as a frequency ratio, in semitones, -1 to 1 |
| `Wheel()` | The mod wheel with the aftertouch added, 0 to 1 |
| `Pressure()` | The aftertouch alone |
| `Vibrato(sr, rate, depth)`, `VibratoRatio(...)` | A vibrato the wheel brings in: semitones, or the ratio with the bend |
| `Range` | Semitones of a full bend (0 is `DefaultRange`, 2) |

## oversample

`blocks/oversample`

Runs a function at a higher sample rate, for processing that makes overtones (saturation, folding)
without them folding back as aliasing: `oversample.New(factor, oversample.TapsPerPhase)`, then
`Process(x, func(x float64) float64)` per sample. The up and down sampling filters are a
Blackman windowed sinc (with `TapsPerPhase` the band up to 16 kHz stays within half a dB); `Latency()` is the delay they add in samples
(delay a dry signal by it to mix it back in), `Reset()` clears them. The saturator uses it.

