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
right. `IRFromSoundFile` doesn't resample, so use a file at the patch sample rate. It normalizes
the response to unit energy (`NormalizeIR`), like the synthetic ones, so `Gain` 1 roughly keeps
the input level whatever the recording; recorded responses otherwise differ a lot in level. A fully
wet convolver reports its latency to the mixer (`SignalLatency`), which lines up the other strips;
mixed with its dry signal the latency is a pre delay.
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

A stereo delay whose echoes bounce between left and right. The echo time is set in **beats** and
follows the tempo, also when it changes while playing (the time glides like tape, over about 60 ms).

```go
echo := pingpong.New(pingpong.Config{Time: 0.75, Feedback: 0.4, Mix: 0.3}).AddTo(p).In(lead)
// or start from the defaults: a dotted 8th whose repeats slowly darken
echo = pingpong.New(pingpong.DefaultConfig()).AddTo(p).In(lead)
```

| Setting | Default | Input / message | Meaning |
|---|---|---|---|
| `Time` | 0.75 | input 0, `time` | Echo time in beats (0.5 an 8th, 0.75 a dotted 8th, 1 a quarter) |
| `TimeMs` | 0 | `timeMs` | Echo time in milliseconds instead, used when above 0, doesn't follow the tempo |
| `Feedback` | 0.35 | input 1, `feedback` | Every echo is this much quieter than the one before (at most 0.98) |
| `Mix` | 0.25 | input 2, `mix` | 0 dry, 1 wet |
| `Stereo` | 0 | `stereo` | 0 is a ping pong (everything bounces, starting on the left), 1 a plain stereo delay (each side echoes its own input) |
| `HighCut` | 6000 | `highCut` | Hz, lowpass on every repeat, so the echoes get darker (0 is off) |
| `LowCut` | 120 | `lowCut` | Hz, highpass on every repeat, so the echoes get thinner (0 is off) |
| `MaxSeconds` | 4 | | Longest echo time (the buffer), longer times are clamped |

The defaults are those of `DefaultConfig()`; a `Config` written out leaves out what isn't set (no
tone filters, a ping pong).

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
| `Sibilance`, `SibilanceFreq` | 0.25, 5000 | `sibilance`, `sibilanceFreq` | The voice above this frequency passes straight through, for consonants (only while a carrier plays: without a carrier the vocoder is silent) |
| `FormantShift` | 1 | `formantShift` | Above 1 a smaller voice, below 1 a bigger one |
| `Gain` | 1 | `gain` | Output level |

Carriers need lots of harmonics: detuned saws, a little noise, an open filter. In a
[mixer](mixer.md#latency) the other strips are lined up with the vocoder by themselves; outside a
mixer delay them with the [latency](#latency) module by `voc.Latency()`.

## granulator

Records its stereo input into a buffer and plays grains from it: short windowed slices read a
little behind the input, with pitch, spray and stereo spread. **Freeze** stops the recording, so the
grains keep playing the captured moment while the input moves on. Feedback sends the grains back
into the buffer, so pitched grains stack up (a fifth up stacks fifths).

```go
gran := granulator.New(granulator.DefaultConfig())
gran.AddTo(p)
gran.In(synth)
gran.ReceiveMessage(psalm.Message{"freeze": 1.0}) // or SetFreeze(true), or value input 1
```

| Setting | Default | Input / message | Meaning |
|---|---|---|---|
| `Freeze` | false | input 1 (above 0.5), `freeze` | Stop recording, keep playing the buffer |
| `Delay` | 150 | input 2, `delay` | Milliseconds behind the input the grains read |
| `Density` | 25 | input 3, `density` | Grains per second |
| `GrainSize` | 120 | input 4, `grainSize` | Milliseconds |
| `Pitch` | 0 | input 5, `pitch` | Semitones |
| `Spray` | 80 | `spray` | Milliseconds, random spread of the grain positions |
| `PitchSpread` | 0 | `pitchSpread` | Semitones, random detune per grain |
| `Octave` | 0 | `octave` | 0-1, chance a grain plays an octave up |
| `Reverse` | 0 | `reverse` | 0-1, chance a grain plays backwards |
| `Shape` | 0 | `shape` | 0-1, grain window from smooth to percussive (stutter) |
| `Jitter` | 0.4 | `jitter` | 0-1, randomness of the time between grains |
| `StereoSpread` | 0.7 | `stereoSpread` | 0-1, random pan per grain |
| `Feedback` | 0 | `feedback` | 0-1, grains fed back into the buffer |
| `Mix` | 0.5 | `mix` | 0 dry, 1 wet |
| `BufferSeconds` | 4 | | Length of the buffer, the longest delay |

Every grain is placed so it only reads recorded audio: pitched up grains never catch up with the
input, and frozen grains never cross the seam between the newest and oldest audio, so there are no
clicks from reading stale audio. The level stays steady when the density or grain size changes.

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
