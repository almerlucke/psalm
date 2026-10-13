# Effects

All effects in `modules/effects` have one stereo input (0) and one stereo output (0), unless noted.

**Mix.** `Mix` blends dry and wet with an equal power crossfade (`blocks/mix`): the wet signal of
a reverb, delay, chorus, granulator or pitch shifter doesn't correlate with the dry one, so their
powers add and the loudness stays the same at every mix (0.5 is as loud as dry, where a linear
crossfade dips 3 dB). The flanger keeps a linear crossfade: its sound is the delayed copy cancelling
against the dry signal, deepest at 0.5.

**Described.** Every effect describes its settings: `<effect>.Describe(fx.Settings())` (`params`:
each message key, its range, unit and where it is), for the editor of the panel
(`modules/controls/panel/editor`). Gain, pan, freeverb and pitchshift take the module itself
(`gain.Describe(g)`), the auto-wah is `follower.DescribeWah`. Settings no message changes (a
compressor's auto makeup, the flanger's shape, a granulator's buffer length, the vocoder's bands)
aren't described.

**Smooth changes.** Mixes, feedback, levels and widths glide to a new value in about 30 ms (the
chorus and flanger also glide their delay times), so they can be automated or driven by an LFO
without zipper noise or clicks.

Further down: [phaser](#phaser), [rotary](#rotary), [tremolo](#tremolo), [tape](#tape),
[ringmod](#ringmod); [dynamics](#dynamics) (compressor with sidechain, gate and expander, transient
shaper); [tone and color](#tone-and-color) (channel EQ, saturation, bit crusher, stereo widener);
[following the level](#following-the-level) (envelope follower, auto-wah).

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
| `Mix` | 0.26 | `mix` | 0 dry, 1 wet |

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
| `Mix` | 0.37 | `mix` | 0 dry, 1 wet |

### convolver

Non-uniformly partitioned FFT convolution: the start of the response is convolved in blocks of
`PartitionSize` (its only latency), the rest in blocks that grow four times (each size used three
times) up to 8192 samples, so even a long response (a reverb of several seconds) at a low latency
costs little: a 3 second stereo hall takes about 2% of a core at a partition of 128 samples. With
`MonoInput` (both sides get the same input, like a mono send to a stereo response) the input is
transformed once. The inaudible end of a response (its
energy 90 dB below the whole) is left out. Larger partitions are a little cheaper still, but do
more work at once.

```go
ir := convolver.IRFromSoundFile(sndfile.MustSoundFile[float32]("assets/IR44100/St Nicolaes Church.wav"))
// or a generated one: ir := convolver.SyntheticReverbIR(3.0, 0.6, psalm.SampleRate(), 1)
reverb := convolver.New(ir, convolver.Config{PartitionSize: 512, Mix: 0.3, Gain: 1.0})
```

| Setting | Default | Message | Meaning |
|---|---|---|---|
| `PartitionSize` | 512 | - | Block size (power of 2), the latency in samples (`Latency()`) |
| `Mix` | 1 | `mix` | 0 dry, 1 wet |
| `Gain` | 1 | `gain` | Wet gain |
| `MonoInput` | false | - | Both sides get the same input: it is transformed once (cheaper) |

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

A stereo chorus: a few copies of the input, each read with a slowly moving delay, blended with the
input. Every side processes its own input; the modulation of the right side runs out of phase with
the left for width. Fully wet it keeps about the level of the input.

```go
cfg := chorus.DefaultConfig()
cfg.Mix = 0.35
ens := chorus.New(cfg).AddTo(p).In(strings)
```

| Setting | Default | Input / message | Meaning |
|---|---|---|---|
| `Rate` | 0.6 | input 1, `rate` | Hz, speed of the modulation (every voice a little different) |
| `Depth` | 3 | input 2, `depth` | Milliseconds the delay moves either way (0-10) |
| `Delay` | 15 | `delay` | Milliseconds, the center delay (5-30: short is closer to a flanger, long a doubler) |
| `Voices` | 3 | | Delayed copies per side, 1-4 |
| `Spread` | 1 | `spread` | 0-1, stereo width (0 both sides move together) |
| `Feedback` | 0 | `feedback` | 0-0.9, metallic, resonant chorus |
| `Tone` | 9000 | `tone` | Hz, lowpass on the copies (0 is off) |
| `Mix` | 0.4 | input 3, `mix` | 0 dry, 1 wet |

Settings glide when they change, so moving them doesn't click.

### flanger

A stereo flanger: the input mixed with a copy delayed by a few milliseconds that sweeps up and down,
so the notches it makes sweep through the sound. The sweep follows the tempo when it is set in
beats.

```go
cfg := flanger.DefaultConfig()
cfg.Beats = 8 // a sweep every two bars
jet := flanger.New(cfg).AddTo(p).In(drums)
```

| Setting | Default | Input / message | Meaning |
|---|---|---|---|
| `Rate` | 0.2 | input 1, `rate` | Hz, speed of the sweep |
| `Beats` | 0 | `beats` | Length of a sweep in beats, follows the tempo; used instead of `Rate` when above 0 |
| `Manual` | 0.5 | `manual` | Milliseconds, the shortest delay (the highest notches), 0.1-10 |
| `Depth` | 4 | input 2, `depth` | Milliseconds the sweep reaches above `Manual`, 0-20 |
| `Feedback` | 0.5 | input 3, `feedback` | -0.95 to 0.95: positive rings metallic, negative sounds hollow |
| `Stereo` | 0.5 | `stereo` | 0-1, how far the right sweep is out of phase (1 is half a cycle) |
| `Shape` | `Triangle` | | `flanger.Triangle` or `flanger.Sine` |
| `Mix` | 0.5 | input 4, `mix` | 0 dry, 1 only the delayed copy; 0.5 makes the deepest notches |

Fully wet it keeps the level of the input.

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
| `Time` | 0.75 | input 1, `time` | Echo time in beats (0.5 an 8th, 0.75 a dotted 8th, 1 a quarter) |
| `TimeMs` | 0 | `timeMs` | Echo time in milliseconds instead, used when above 0, doesn't follow the tempo |
| `Feedback` | 0.35 | input 2, `feedback` | Every echo is this much quieter than the one before (at most 0.98) |
| `Mix` | 0.2 | input 3, `mix` | 0 dry, 1 wet |
| `Stereo` | 0 | `stereo` | 0 is a ping pong (everything bounces, starting on the left), 1 a plain stereo delay (each side echoes its own input) |
| `HighCut` | 6000 | `highCut` | Hz, lowpass on every repeat, so the echoes get darker (0 is off) |
| `LowCut` | 120 | `lowCut` | Hz, highpass on every repeat, so the echoes get thinner (0 is off) |
| `MaxSeconds` | 4 | | Longest echo time (the buffer), longer times are clamped |

The defaults are those of `DefaultConfig()`; a `Config` written out leaves out what isn't set (no
tone filters, a ping pong).

### phaser

`modules/effects/phaser`: the sound through a chain of allpass filters whose frequency an LFO sweeps,
mixed with itself: notches that move up and down (an electric piano, strings, a pad).

```go
ph := phaser.New(phaser.DefaultConfig()).AddTo(p).In(keys)
ph = phaser.New(phaser.Config{Stages: 6, Sync: 4, Min: 200, Max: 3000, Feedback: 0.6, Mix: 0.5})
```

| Setting | Default | Input / message | Meaning |
|---|---|---|---|
| `Stages` | 4 | | Allpass stages: 4 (the classic), 6, 8, 12 (more notches) |
| `Rate` | 0.4 | input 1, `rate` | Hz of the sweep |
| `Sync` | 0 | `sync` | Beats of one sweep instead, following the tempo (used when above 0) |
| `Min`, `Max` | 300, 2500 | `min`, `max` | Hz, the range of the sweep (exponential: even over the octaves) |
| `Feedback` | 0.4 | input 2, `feedback` | -0.95 to 0.95, deeper and more resonant notches |
| `Stereo` | 0.25 | `stereo` | How far apart the sweeps of left and right are (part of a cycle) |
| `Mix` | 0.5 | input 3, `mix` | 0.5 the deepest notches; 1 only the shifted sound (a vibrato). A linear crossfade: the notches come from cancelling |

### rotary

`modules/effects/rotary`: a rotary speaker (a Leslie): a horn for the highs and a drum for the lows
turn, the sound coming closer and going away (a Doppler vibrato and a rising and falling level),
picked up by two microphones (stereo). Organ, electric piano, guitar.

```go
leslie := rotary.New(rotary.DefaultConfig()).AddTo(p).In(organ)
ctl.Toggle("Fast", panel.Func(func(v float64) { leslie.SetFast(v > 0.5) })) // chorale / tremolo live
```

| Setting | Default | Input / message | Meaning |
|---|---|---|---|
| `Fast` | false | input 1, `fast` | Tremolo speed (horn 6.7, drum 5.8 turns a second), else chorale (0.83, 0.67). Switched, the horn gets there in about a second, the heavy drum in four to five |
| `Crossover` | 800 | | Hz between the drum and the horn |
| `Drive` | 3 | `drive` | dB into the tube amplifier before it (0 is clean) |
| `Horn` | 0.5 | `horn` | The level of the horn against the drum |
| `Spread` | 1 | `spread` | The stereo width of the microphones |
| `Mix` | 1 | input 2, `mix` | 0 dry, 1 only the speaker |

`Speeds()` returns the turns per second of the horn and the drum now.

### tremolo

`modules/effects/tremolo`: a tremolo (the level moves) or an auto-pan (the place moves). Synced to the
tempo its phase follows the clock of the patch: it pulses on the beats, also when added while
playing or after a restart.

```go
trem := tremolo.New(tremolo.Config{Sync: 0.25, Depth: 0.8, Shape: tremolo.Square}).AddTo(p).In(pad) // a 16ths chop
pan := tremolo.New(tremolo.Config{Mode: tremolo.Pan, Sync: 2, Depth: 0.7}).AddTo(p).In(keys)
```

| Setting | Default | Input / message | Meaning |
|---|---|---|---|
| `Mode` | `Tremolo` | `mode` | `Tremolo` (the level) or `Pan` (left to right, +1 is right) |
| `Shape` | `Sine` | `shape` | `Sine`, `Triangle`, `Square` (on and off, rounded: no clicks) |
| `Rate` | 0 | input 1, `rate` | Hz |
| `Sync` | 0.5 | `sync` | Beats of one cycle instead (0.25 is 16ths), on the beats of the clock |
| `Depth` | 0.5 | input 2, `depth` | 0-1: down to silence at 1, or all the way to the sides |
| `Stereo` | 0 | `stereo` | Tremolo: how far apart left and right are (0.5: they take turns) |

### tape

`modules/effects/tape`: a tape echo: every repeat a little darker and warmer (filtered and saturated on
the way round), wavering with the wow and flutter of the tape. The time is in beats and follows the
tempo. Feedback above 1 builds up into the singing self oscillation of a dub delay; the saturation
keeps it in bounds.

```go
echo := tape.New(tape.DefaultConfig()).AddTo(p).In(lead)
dub := tape.New(tape.Config{Time: 0.75, Feedback: 1.05, Wow: 0.5, HighCut: 2500, LowCut: 300, Mix: 0.4})
```

| Setting | Default | Input / message | Meaning |
|---|---|---|---|
| `Time` | 0.75 | input 1, `time` | Echo time in beats |
| `TimeMs` | 0 | `timeMs` | In milliseconds instead (used when above 0) |
| `Feedback` | 0.45 | input 2, `feedback` | 0-1.2; above 1 the echoes build up |
| `Wow`, `Flutter` | 0.3, 0.2 | `wow`, `flutter` | 0-1, the slow wavering (up to 4 ms) and the fast trembling of the tape |
| `Drive` | 3 dB (0) | `drive` | dB of saturation on the way round |
| `HighCut`, `LowCut` | 4500, 150 | `highCut`, `lowCut` | Hz, the repeats get darker and thinner |
| `Mix` | 0.25 | input 3, `mix` | 0 dry, 1 wet (equal power) |
| `MaxSeconds` | 4 | | The longest echo time |

### ringmod

`modules/effects/ringmod`: a ring modulator (the sound times a sine: every frequency becomes its sum
and its difference with the carrier, metallic and bell like) and a frequency shifter (every frequency
moved by the same number of Hz, the harmonics stop lining up; a Hilbert transform keeps only the
shifted side).

```go
bells := ringmod.New(ringmod.Config{Mode: ringmod.Ring, Freq: 440, Mix: 0.5}).AddTo(p).In(pad)
wide := ringmod.New(ringmod.Config{Mode: ringmod.Shift, Freq: 3, Stereo: -6, Mix: 0.5}) // a slow, wide beating
```

| Setting | Default | Input / message | Meaning |
|---|---|---|---|
| `Mode` | `Ring` | `mode` | `Ring` or `Shift` |
| `Freq` | 300 | input 1, `freq` | Hz of the carrier, or of the shift (negative shifts down) |
| `Stereo` | 0 | `stereo` | Hz the right side differs |
| `Mix` | 0.5 | input 2, `mix` | 0 dry, 1 only the modulated sound |

## Dynamics

### compressor

`modules/effects/compressor`: above a threshold it turns the level down by a ratio, with a soft knee,
attack and release (the detector follows the level, not every wave, so a fast attack doesn't
distort a bass). Input 1 is a sidechain: when something is connected it drives the compression, so
a kick ducks the bass or a pad. The mix blends in the dry sound (parallel compression).

```go
glue := compressor.New(compressor.DefaultConfig()).AddTo(p).In(drums)
duck := compressor.New(compressor.Ducker(10, 180)).AddTo(p) // up to 10 dB, back over 180 ms
duck.In(pad, 0, 0, kick, 0, 1)                                // the pad on 0, the kick on the sidechain
```

| Setting | Default | Input / message | Meaning |
|---|---|---|---|
| `Threshold` | -18 | input 1, `threshold` | dB above which it compresses |
| `Ratio` | 3 | input 2, `ratio` | 4 is 4:1; 20 and up is a limiter |
| `Knee` | 6 | `knee` | dB, the soft knee around the threshold (0 is hard) |
| `Attack`, `Release` | 10, 150 | `attack`, `release` | ms |
| `Makeup` | 0 | `makeup` | dB of gain after it; `AutoMakeup` gives half the reduction at 0 dB instead |
| `Mix` | 1 | input 3, `mix` | 0 dry, 1 compressed (parallel compression in between) |
| `SideCut` | 0 | `sideCut` | Hz, highpass on the detector: the low end doesn't pump it |

`Reduction()` returns the most gain reduction (dB) since it was asked last, for a meter.

### gate

`modules/effects/gate`: below a threshold it turns the level down: all the way to the range (a gate)
or by a ratio (a downward expander), with attack, hold and release. A sidechain on input 1 opens it
(a pad gated by the drums).

```go
g := gate.New(gate.DefaultConfig()).AddTo(p).In(drums)
x := gate.New(gate.Expander(-50)).AddTo(p).In(keys) // gentle 1:2 below -50 dB
```

| Setting | Default | Input / message | Meaning |
|---|---|---|---|
| `Threshold` | -45 | input 1, `threshold` | dB below which it turns down |
| `Ratio` | 0 (a gate) | `ratio` | 2 is a 1:2 expander; 0 (or 100 and more) a gate |
| `Range` | 80 | input 2, `range` | dB, the most it turns down |
| `Attack`, `Hold`, `Release` | 1, 30, 120 | `attack`, `hold`, `release` | ms |
| `SideCut` | 0 | `sideCut` | Hz, highpass on the detector |

### transient

`modules/effects/transient`: finds the start of every hit (a fast follower of the level running ahead
of a slow one) and its decay (falling behind), and turns them up or down: snappier or softer drums,
more or less ring, whatever the level.

```go
snap := transient.New(transient.Config{Attack: 8, Sustain: -6, Mix: 1}).AddTo(p).In(drums)
```

| Setting | Default | Input / message | Meaning |
|---|---|---|---|
| `Attack` | 6 | input 1, `attack` | dB the start of a hit is turned up (negative: down), -24 to 24 |
| `Sustain` | -3 | input 2, `sustain` | dB the decay is turned up (negative: down) |
| `Output` | 0 | `output` | dB after it |
| `Mix` | 1 | `mix` | 0 dry, 1 shaped |

## Tone and color

### eq

`modules/effects/eq`: a channel EQ: a low cut, a low shelf, two peaking bands, a high shelf and a high
cut. Changes glide (a part every buffer), so it can be automated.

```go
cfg := eq.DefaultConfig() // flat: the bands at 100, 500, 2500 and 8000 Hz
cfg.LowCut = 180                                      // the low end out of a pad
cfg.Mid2 = eq.Band{Freq: 3000, Gain: 3, Q: 1.2}         // presence
cfg.High = eq.Band{Freq: 10000, Gain: 2}                // air
pad := eq.New(cfg).AddTo(p).In(padSynth)
```

Messages: `lowCut`, `highCut`, `output`, and per band (`low`, `mid1`, `mid2`, `high`) its `...Freq`,
`...Gain` and `...Q` (`mid1Gain`, `highFreq`, ...). The cuts are 12 dB/octave, 0 is off; a band at 0 dB
does nothing.

### saturate

`modules/effects/saturate`: saturation and distortion: the sound driven into a curve that rounds its
peaks off. It runs four times oversampled (`blocks/oversample`), so the harmonics it makes don't fold
back as aliasing; the dry sound in the mix is delayed to line up (`SignalLatency`, a few samples, the
mixer aligns other strips to it).

```go
warm := saturate.New(saturate.DefaultConfig()).AddTo(p).In(bass)
fuzz := saturate.New(saturate.Config{Shape: saturate.Hard, Drive: 24, Tone: 4000, Output: -12, Mix: 1})
```

| Setting | Default | Input / message | Meaning |
|---|---|---|---|
| `Shape` | `Soft` | `shape` | `Soft` (tape, odd harmonics), `Tube` (lopsided, even harmonics too), `Hard` (clipped, fuzz), `Fold` (folded back, buzzy, changing with the drive) |
| `Drive` | 9 | input 1, `drive` | dB into the curve (0 just touches it, 24 is heavy) |
| `Bias` | 0 | `bias` | -1 to 1, shifts the curve: more even harmonics |
| `Tone` | 9000 | `tone` | Hz, a lowpass after it (0 is off) |
| `Output` | -4 | `output` | dB after it |
| `Mix` | 1 | input 2, `mix` | 0 dry, 1 saturated |

### crusher

`modules/effects/crusher`: a bit crusher: fewer bits (the steps of an old sampler, grainy noise) and a
lower sample rate (held samples, metallic aliasing).

```go
lofi := crusher.New(crusher.DefaultConfig()).AddTo(p).In(drums) // 8 bits at 11 kHz
```

| Setting | Default | Input / message | Meaning |
|---|---|---|---|
| `Bits` | 8 | input 1, `bits` | 1 to 24 (fractions move between), 0 is off |
| `Rate` | 11025 | input 2, `rate` | Hz it holds the sound at (0 is off) |
| `Output` | 0 | `output` | dB after it |
| `Mix` | 1 | input 3, `mix` | 0 dry, 1 crushed |

### widener

`modules/effects/widener`: the sound split into its middle (what both sides share) and its sides (what
differs): more sides is wider, none is mono. The low end can be kept in the middle.

```go
wide := widener.New(widener.DefaultConfig()).AddTo(p).In(pad) // 1.4 wide, mono below 120 Hz
```

| Setting | Default | Input / message | Meaning |
|---|---|---|---|
| `Width` | 1.4 | input 1, `width` | 0 mono, 1 as it is, 2 twice as wide (at most 3) |
| `MonoBelow` | 120 | `monoBelow` | Hz, below it the sound is kept in the middle (0 is off) |

## Following the level

`modules/effects/follower`

**Follower** passes its sound on and sends its level as a value on every control tick, mapped to a
range, for any setting to follow:

```go
f := follower.New(follower.Config{Attack: 5, Release: 200, Gain: 6, Min: 400, Max: 6000}).AddTo(p).In(drums)
filter.ValueIn(f, 0, 1) // the filter opens as the drums get louder
```

**Wah** is an auto-wah: a resonant filter (bandpass, the quack, or lowpass) swept by the level of the
sound, every note opening it as it is played:

```go
wah := follower.NewWah(follower.DefaultWah()).AddTo(p).In(clav)
```

| Wah setting | Default | Meaning |
|---|---|---|
| `Kind` | `BandPass` | `BandPass` or `LowPass` |
| `Min`, `Max` | 350, 2200 | Hz, the sweep from quiet to loud (over the octaves) |
| `Q` | 4 | Resonance |
| `Gain` | 12 | dB of sensitivity (input 1, `gain`) |
| `Attack`, `Release` | 4, 120 | ms |
| `Down` | false | A louder note sweeps it down |
| `Mix` | 1 | 0 dry, 1 wah (input 2, `mix`) |


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

## gain

`gain.New(volume)` - a volume in dB (0 is unity, -120 is silence): a trim at the end of a chain
that has no [mixer](mixer.md), or a level to automate (value input 1, or message `volume`).
Changes glide.

```go
p.Out(gain.New(-2).AddTo(p).In(reverb))
```

## latency

`latency.New(numChannels, samples)` - delays its input by a fixed number of samples, to line up
a signal with one that went through a module with latency (vocoder, convolver).

```go
aligned := latency.New(psalm.Stereo, voc.Latency()).AddTo(p).In(piano)
```
