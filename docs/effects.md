# Effects

All effects in `modules/effects` have one stereo input (0) and one stereo output (0), unless noted.

## Reverbs

| Module | Character | Use it for |
|---|---|---|
| [freeverb](#freeverb) | Classic algorithmic (comb + allpass) | Quick, light, cheap reverb |
| [fdnreverb](#fdnreverb) | Dense, smooth feedback delay network | Rooms to huge halls with precise decay control |
| [shimmer](#shimmer) | FDN with a pitch shifter in the loop | Ethereal, rising ambient tails |
| [convolver](#convolver) | Convolution with an impulse response | Real spaces from recorded impulse responses, creative convolution |

### freeverb

`freeverb.New(roomSize, damp, width, wet, dry)`, all 0-1.

| Setting | Input | Message | Meaning |
|---|---|---|---|
| wet | 0 | `wet` | Reverb level |
| dry | 1 | `dry` | Direct signal level |
| roomSize | 2 | `roomSize` | Room size / decay length |
| damp | 3 | `damp` | High frequency damping |
| width | 4 | `width` | Stereo width |
| mode | 5 | `mode` | 0.5 and above freezes the reverb (endless tail, input muted) |

### fdnreverb

An 8 line feedback delay network: modulated delay lines mixed through an orthogonal (lossless)
Hadamard matrix, with damping filters tuned so the tail decays in exactly `Decay` seconds.

```go
cfg := fdnreverb.DefaultConfig()
cfg.Decay = 4.5
reverb := fdnreverb.New(cfg).AddTo(p).In(synth)
```

| Setting | Default | Message | Meaning |
|---|---|---|---|
| `Decay` | 2.5 | `decay` | Reverb time (RT60) in seconds at low frequencies |
| `Damping` | 0.5 | `damping` | 0-1, how much faster the highs decay (the tail gets darker) |
| `Size` | 0.6 | `size` | 0-1, room size (delay line lengths) |
| `PreDelay` | 20 | `preDelay` | Milliseconds before the reverb starts (max 490) |
| `Diffusion` | 0.7 | `diffusion` | 0-1, softness and density of the attack |
| `ModDepth` | 0.4 | `modDepth` | Delay modulation in milliseconds (max 4), removes metallic ringing |
| `ModRate` | 0.6 | `modRate` | Delay modulation rate in Hz |
| `Width` | 1 | `width` | 0 mono, 1 full stereo |
| `Mix` | 0.3 | `mix` | 0 dry, 1 wet |

`ProcessSample(left, right)` processes one sample and returns the wet signal, for building other
effects around the reverb (the shimmer uses it).

### shimmer

The FDN reverb with a pitch shifter in its feedback loop: every pass through the loop is
shifted (an octave up by default), so the tail blooms upward. A low cut keeps the loop clean and
the feedback backs off automatically when the tail gets loud, so even full shimmer with a long
decay stays at a sane level.

```go
cfg := shimmer.DefaultConfig()
cfg.Reverb.Decay = 8
cfg.Shimmer = 0.6
reverb := shimmer.New(cfg).AddTo(p).In(pad)
```

| Setting | Default | Input / message | Meaning |
|---|---|---|---|
| `Reverb` | large, 6 s decay | messages like `decay`, `damping`, `size` are passed on | The reverb tank (`fdnreverb.Config`, its `Mix` is not used) |
| `Shimmer` | 0.5 | input 1, `shimmer` | 0-1, amount of shifted signal fed back |
| `Semitones` | 12 | `semitones` | Shift in the loop, try 7 (fifth), 19 or -12 |
| `LowCut` | 300 | `lowCut` | Hz, high pass in the loop |
| `Window` | 60 | - | Pitch shifter grain size in ms |
| `Mix` | 0.4 | `mix` | 0 dry, 1 wet |

### convolver

Uniformly partitioned FFT convolution: long impulse responses (like reverbs of several seconds)
only add one partition of latency.

```go
ir := convolver.IRFromSoundFile(sndfile.MustSoundFile[float32]("assets/IR44100/St Nicolaes Church.wav"))
// or a generated one: ir := convolver.SyntheticReverbIR(3.0, 0.6, psalm.SampleRate(), 1)
reverb := convolver.New(ir, convolver.Config{PartitionSize: 512, Mix: 0.35, Gain: 1.0})
```

| Setting | Default | Message | Meaning |
|---|---|---|---|
| `PartitionSize` | 512 | - | Block size (power of 2), the latency in samples (`Latency()`) |
| `Mix` | 1 | `mix` | 0 dry, 1 wet |
| `Gain` | 1 | `gain` | Wet gain |

A mono impulse response is used for both channels, a stereo one maps left to left and right to
right. `IRFromSoundFile` doesn't resample, so use a file at the patch sample rate.
`SyntheticReverbIR(seconds, damping, sampleRate, seed)` generates decaying stereo noise that gets
darker over time, normalized so the reverb roughly keeps the input level.

## Modulation and delay effects

### chorus

`chorus.New(rate, amount, delay, feedback, width, mix, modShaper)` - a 4 tap modulated delay
chorus (pass `nil` as `modShaper` for the default sine).

| Setting | Input | Message | Meaning |
|---|---|---|---|
| rate | 0 | `rate` | Modulation rate in Hz |
| amount | 1 | `amount` | 0-1, modulation depth (up to 20 ms) |
| delay | 2 | `delay` | 0-1, center delay (5-50 ms) |
| feedback | 3 | `feedback` | Feedback |
| width | 4 | `width` | Stereo width |
| mix | 5 | `mix` | 0 dry, 1 wet |

### flanger

`flanger.New(rate, depth, feedback, mix, modShaper)` (`nil` shaper is a sine). The rate is set
at creation.

| Setting | Input | Message | Meaning |
|---|---|---|---|
| depth | 0 | `depth` | 0-1, sweep range (1-20 ms) |
| feedback | 1 | `feedback` | Feedback |
| mix | 2 | `mix` | 0 dry, 1 wet |

### pingpong

`pingpong.New(delayLengthMs, readLocationMs, feedback, mix)` - a stereo delay bouncing between
left and right. `delayLengthMs` is the maximum delay (buffer size), the read location the actual
echo time.

| Setting | Input | Message | Meaning |
|---|---|---|---|
| read location | 0 | `location` | Echo time in ms (at most the delay length) |
| feedback | 1 | `feedback` | Feedback |
| mix | 2 | `mix` | 0 dry, 1 wet |

Use `timing.BPMToMilli(psalm.BPM())` (one beat in ms) to sync echoes to the tempo.

## vocoder

An FFT vocoder: imprints the spectral envelope of the modulator (input 0, usually a voice) on
the carrier (input 1, usually a bright synth). Both inputs are mono (stereo sources are mixed
down), the output is stereo.

```go
voc := vocoder.New(vocoder.DefaultConfig())
voc.AddTo(p)
voc.In(speech, 0, 0, carrier, 0, 1)
```

| Setting | Default | Message | Meaning |
|---|---|---|---|
| `FFTSize` | 1024 | - | Frame size (power of 2), the latency in samples (`Latency()`) |
| `Overlap` | 4 | - | Frames per FFT size |
| `NumBands`, `MinFreq`, `MaxFreq` | 32, 80, 10000 | - | Log spaced analysis bands |
| `Attack`, `Release` | 5, 50 | `attack`, `release` | How fast the bands follow the voice (ms) |
| `Flatten` | 1 | `flatten` | 0-1, how much of the carrier's own tone is removed first |
| `Sibilance`, `SibilanceFreq` | 0.25, 5000 | `sibilance`, `sibilanceFreq` | The voice above this frequency passes straight through, for consonants |
| `FormantShift` | 1 | `formantShift` | Above 1 a smaller voice, below 1 a bigger one |
| `Gain` | 1 | `gain` | Output level |

Carriers need lots of harmonics: detuned saws, a little noise, an open filter. To line up other
parts with the vocoded output, delay them with the [latency](#latency) module by `voc.Latency()`.

## pitchshift

`pitchshift.New(semitones, windowMs, mix)` - a delay line pitch shifter with two crossfading read
heads. Low latency and smooth on dense material. On exposed sustained notes some input
frequencies get a slight warble (sidebands at the head rate, about 20 Hz for an octave up with a
50 ms window).

| Setting | Input | Message | Meaning |
|---|---|---|---|
| semitones | 1 | `semitones` | Shift in semitones (fractions allowed) |
| window | - | - | Grain size in ms (30-100, larger is smoother for low notes) |
| mix | - | `mix` | 0 dry, 1 wet |

## pan

`pan.New(position)` - places a mono input (stereo is mixed down) in the stereo field with an
equal power pan law, scaled so a centered signal keeps its level. Position -1 is left, 1 right.
Changes glide over a buffer, so it can be driven by an LFO on value input 0 (or message `pan`).

## latency

`latency.New(numChannels, samples)` - delays its input by a fixed number of samples, to line up
a signal with one that went through a module with latency (vocoder, convolver).

```go
aligned := latency.New(psalm.Stereo, voc.Latency()).AddTo(p).In(piano)
```
