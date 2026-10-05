# Synths

Synths: [poly](#poly) (voice allocation), [subtractive](#subtractive), [fm](#fm), [dx](#dx) (6 operator, SysEx), [grains](#grains) (granular sampler), [pulsar](#pulsar), [drums](#drums).

## poly

`modules/poly`

Polyphonic voice allocation for any voice type. `poly.New(numChannels, voices)` mixes the voices
to one output and assigns notes to free voices. When all voices are busy it steals the oldest
one, with a short fade out so stealing doesn't click.

A voice implements `poly.Voice`:

```go
type Voice interface {
	psalm.Modular
	NoteOn(amplitude float64, message psalm.Message, tick psalm.CtrlTick, config psalm.Config)
	NoteOff()
	Note(duration float64, amplitude float64, message psalm.Message, tick psalm.CtrlTick, config psalm.Config)
	Clear()
	IsActive() bool
}
```

**Messages** (usually built with `templ.NewTrigger` / `templ.NewDurationTrigger`, see [controls](controls.md#templ)):

| Message | Effect |
|---|---|
| `{"command": "trigger", "tick", "amplitude", "message", "duration"}` | Plays a note of `duration` seconds on a free voice, the voice receives `message` (pitch, settings, ...) |
| `{"command": "trigger", "tick", "amplitude", "message", "noteOn": id}` | Starts a held note, remembered under `id` |
| `{"command": "trigger", "tick", "noteOff": id}` | Releases the held note `id` |
| `{"command": "voice", ...}` | Passes the whole message to all voices (change settings of every voice) |

**Value inputs**: plain values on inputs above 0 are passed to all voices on the same input, so
an LFO connected to input 1 of a subtractive poly synth modulates the cutoff of every voice.

`AllNotesOff()` releases all held notes, `CallVoices(func(v))` runs a function on every voice.

## subtractive

`synths/subtractive`

A classic subtractive synth voice: two band limited oscillators, a sub oscillator, noise, ring
modulation, sync and FM, through a choice of filters (state variable, Moog style ladder or
formant), with filter and amp envelopes, two LFOs and glide.

```go
synth := subtractive.NewPoly(subtractive.LeadPreset(), 8).AddTo(p) // config, number of voices

cfg := subtractive.Presets["acidBass"]() // presets by name
cfg.Glide = 60
bass := subtractive.NewPoly(cfg, 1).AddTo(p) // one voice: every note glides from the last
```

`subtractive.New(cfg)` creates a single voice, `subtractive.NewPoly(cfg, n)` a `poly` synth with
`n` voices. `subtractive.DefaultConfig()` is a two saw voice with a 12 dB lowpass.

Signal flow: `osc 1 + osc 2 + sub + ring mod + noise -> drive -> filter -> amp (envelope, tremolo)`

### Oscillators

| Setting | Default | Meaning |
|---|---|---|
| `Osc1Waveform`, `Osc2Waveform` | Sawtooth | Any `blosc.Waveform` (see [blocks](blocks.md#blosc)) |
| `Osc1Level`, `Osc2Level` | 0.5, 0.5 | Mix levels |
| `Osc2Detune` | 0.07 | Osc 2 detune in semitones (fractions for cents, 7 for a fifth, 12 for an octave) |
| `Osc2Octave` | 0 | Osc 2 octave offset |
| `PulseWidth` | 0.5 | Pulse width (0-1) for the pulse width aware waveforms (Rectangle, ...) |
| `SubLevel` | 0 | Square sub oscillator one octave below osc 1 |
| `NoiseLevel` | 0 | Noise level |
| `NoiseType` | `WhiteNoise` | `WhiteNoise` or `PinkNoise` |
| `RingModLevel` | 0 | Level of osc 1 x osc 2 (metallic, bell like tones) |
| `Sync` | false | Hard sync osc 2 to osc 1 |
| `FMAmount` | 0 | Osc 1 frequency modulates osc 2 |
| `Drive` | 1 | Saturation before the filter (1 is clean) |

### Pitch

| Setting | Default | Meaning |
|---|---|---|
| `Glide` | 0 | Portamento in milliseconds from the previous note of the same voice (use 1 voice for mono slides) |
| `FilterEnvToPitch` | 0 | Semitones of pitch sweep from the filter envelope (zaps, kicks, falling sounds) |

### Filter

| Setting | Default | Meaning |
|---|---|---|
| `FilterType` | `FilterSVF` | `FilterSVF` (12 dB state variable), `FilterLadder` (Moog style 24/12 dB ladder) or `FilterFormant` (vowel filter) |
| `FilterMode` | `LowPass` | SVF output: `LowPass`, `BandPass`, `HighPass`, `Notch`, `Peak`, `AllPass` |
| `LadderMode` | `ladder.LowPass24` | Ladder output: `LowPass24`, `LowPass12`, `BandPass24`, `BandPass12`, `HighPass24`, `HighPass12` |
| `FilterDrive` | 1 | Ladder input saturation |
| `Cutoff` | 800 | Cutoff in Hz |
| `Resonance` | 0.3 | 0-1 (the ladder self-oscillates near 1), for the formant filter the sharpness of the formants |
| `FilterEnvAmount` | 3 | Octaves the filter envelope adds to the cutoff (vowel steps for the formant filter, can be negative) |
| `KeyTrack` | 0.5 | 0-1, how much the cutoff follows the pitch (relative to middle C) |
| `Vowel` | 0 | Formant filter vowel 0-4 (a, e, i, o, u, fractions morph) |
| `FormantShift` | 1 | Scales the formant frequencies (above 1 a smaller voice, below 1 a bigger one) |

With the formant filter, all cutoff modulation (filter envelope, LFO cutoff amounts) moves the
vowel instead of a cutoff, in vowel steps.

### Envelopes

`FilterEnv` and `AmpEnv` are ADSR settings (`adsr.NewSetting(attackLevel, attackMs, sustainLevel,
decayMs, sustainMs, releaseMs)` from `github.com/almerlucke/genny/float/envelopes/adsr`). Held
notes stay at the sustain level until note off, timed notes release after their duration.

### LFOs

`LFO1` and `LFO2` are `subtractive.LFOConfig`, per voice:

| Field | Meaning |
|---|---|
| `Waveform` | `LFOSine`, `LFOTriangle`, `LFOSaw`, `LFORamp`, `LFOSquare`, `LFOSampleHold` |
| `Rate` | Hz |
| `Delay` | Fade in time in milliseconds after note on (delayed vibrato) |
| `KeySync` | Restart the LFO on every note |
| `Pitch` | Semitones of vibrato |
| `Cutoff` | Octaves of cutoff modulation (vowel steps for the formant filter) |
| `PulseWidth` | Pulse width modulation |
| `Amp` | Tremolo depth 0-1 |

An LFO without any destination amount, or with rate 0, is off.

### Inputs and messages

**Value inputs** (connect LFO modules or generators to the poly synth): 1 cutoff, 2 resonance,
3 pulse width, 4 vowel.

**Messages** (in the note message of a trigger, or with `"command": "voice"`):

| Group | Keys |
|---|---|
| Pitch | `pitch` (MIDI), `frequency` (Hz), `glide`, `filterEnvToPitch` |
| Oscillators | `osc1Waveform`, `osc2Waveform`, `osc1Level`, `osc2Level`, `osc2Detune`, `osc2Octave`, `pulseWidth`, `subLevel`, `noiseLevel`, `noiseType`, `ringModLevel`, `sync`, `fmAmount`, `drive` |
| Filter | `filterType`, `filterMode`, `ladderMode`, `filterDrive`, `cutoff`, `resonance`, `filterEnvAmount`, `keyTrack`, `vowel`, `formantShift` |
| LFOs | `lfo1Waveform`, `lfo1Rate`, `lfo1Delay`, `lfo1KeySync`, `lfo1Pitch`, `lfo1Cutoff`, `lfo1PulseWidth`, `lfo1Amp`, and the same with `lfo2` |

Settings sent with a note change that voice and stay until changed again, so a sequence can step
through settings note by note (see `cmd/examples/subtractive2`).

### Presets

All presets are functions returning a `Config`, also available by name in `subtractive.Presets`.

| Name | Function | Sound |
|---|---|---|
| `bass` | `BassPreset` | Punchy saw bass with a square an octave below, 24 dB ladder |
| `acidBass` | `AcidBassPreset` | Squelchy resonant saw through a driven ladder |
| `subBass` | `SubBassPreset` | Clean sine sub bass |
| `reeseBass` | `ReeseBassPreset` | Heavily detuned saws with a slow cutoff LFO |
| `fmBass` | `FMBassPreset` | Growly FM bass |
| `squareBass` | `SquareBassPreset` | Rubbery pulse bass with short glide |
| `ladderBass` | `LadderBassPreset` | Fat, driven 24 dB ladder bass |
| `wobble` | `WobblePreset` | Resonant bass for an external cutoff LFO |
| `lfoWobble` | `LFOWobblePreset` | Wobble bass driven by its own LFO |
| `lead` | `LeadPreset` | Bright detuned saw lead |
| `ladderLead` | `LadderLeadPreset` | Warm driven ladder lead with glide and vibrato |
| `syncLead` | `SyncLeadPreset` | Hard sync lead |
| `chipLead` | `ChipLeadPreset` | 8-bit square and pulse lead |
| `chipArp` | `ChipArpPreset` | Thin pulse with a very short envelope for arpeggios |
| `pluck` | `PluckPreset` | Short saw pluck |
| `pad` | `PadPreset` | Wide, slow detuned saw pad |
| `hollowPad` | `HollowPadPreset` | Darker pulse and triangle pad |
| `pwmPad` | `PWMPadPreset` | Pulse width modulated pad with vibrato |
| `choirPad` | `ChoirPadPreset` | "ooh/aah" pad through a morphing formant filter |
| `notchPad` | `NotchPadPreset` | Detuned saws through a swept notch, phaser like |
| `strings` | `StringsPreset` | String ensemble style |
| `brass` | `BrassPreset` | Saw brass with a filter swell, 12 dB ladder |
| `organ` | `OrganPreset` | Square and octave triangle organ |
| `ePiano` | `EPianoPreset` | FM electric piano with tremolo |
| `glassKeys` | `GlassKeysPreset` | Glassy FM bell keys |
| `vibes` | `VibesPreset` | Mallet tone with tremolo |
| `ringBell` | `RingBellPreset` | Ring modulated bell |
| `flute` | `FlutePreset` | Breathy triangle with delayed vibrato |
| `clarinet` | `ClarinetPreset` | Woody square |
| `theremin` | `ThereminPreset` | Sine with long glide and wide vibrato (use 1 voice) |
| `talkingSynth` | `TalkingSynthPreset` | Talk box like formant sweep |
| `synthKick` | `SynthKickPreset` | Analog style kick (play around MIDI 36) |
| `synthSnare` | `SynthSnarePreset` | Tone and noise snare (around MIDI 50) |
| `synthHat` | `SynthHatPreset` | Highpassed noise hat |
| `synthTom` | `SynthTomPreset` | Pitched tom, the pitch sets the size |
| `zap` | `ZapPreset` | Laser zap from a fast pitch drop |
| `riser` | `RiserPreset` | Noise and saw sweep that rises over a few seconds |
| `wind` | `WindPreset` | Noise through a resonant bandpass |

## fm

`synths/fm`

A 4 operator FM synth voice: four sine operators that phase modulate each other in one of 8
classic routings (algorithms). FM is great for bright, glassy and metallic sounds that are hard
to get with subtractive synthesis: electric pianos, bells, mallets, brass, punchy basses.

```go
epiano := fm.NewPoly(fm.EPianoPreset(), 16).AddTo(p)

cfg := fm.Presets["bell"]()
cfg.ModDepth = 0.7 // a bit darker
bells := fm.NewPoly(cfg, 8).AddTo(p)
```

`fm.New(cfg)` creates a single voice, `fm.NewPoly(cfg, n)` a `poly` synth with `n` voices.

### How FM works here

Every operator is a sine oscillator with its own envelope. A **carrier** is heard, a
**modulator** bends the phase of the operator it is connected to, which adds partials
(sidebands) around it at the carrier frequency plus and minus multiples of the modulator
frequency. The modulator level sets how many sidebands appear (the modulation index), so a
modulator envelope that decays makes the sound go from bright to mellow, like a struck string.

- Whole number ratios between operators give harmonic sounds, other ratios inharmonic (bell,
  metal) sounds.
- Velocity makes modulators louder, so harder notes are brighter.

### Algorithms

Operators are numbered 1-4, `a>b` means a modulates b. Operator 4 can also modulate itself
(`Feedback`).

| Algorithm | Routing | Carriers | Typical use |
|---|---|---|---|
| 0 | 4>3>2>1 | 1 | Basses, leads, complex evolving tones |
| 1 | (4+3)>2>1 | 1 | Brass, metallic sounds |
| 2 | (4 + 3>2)>1 | 1 | Plucks |
| 3 | (4>3 + 2)>1 | 1 | Variations of the above |
| 4 | 4>3, 2>1 | 1, 3 | Two layered tones: electric piano, bells, mallets |
| 5 | 4>(3, 2, 1) | 1, 2, 3 | Pads and ensembles |
| 6 | 4>3, 2, 1 | 1, 2, 3 | One modulated tone plus two sines |
| 7 | 4, 3, 2, 1 | all | Additive: organs |

`fm.Algorithms` holds them (`Name`, `Mods` and `Carriers`).

### Settings

Per operator (`cfg.Ops[0]` is operator 1):

| Setting | Meaning |
|---|---|
| `Ratio` | Frequency ratio to the note (1 unison, 2 an octave up, 3.5 inharmonic) |
| `Fixed` | Fixed frequency in Hz instead of a ratio (when > 0), for pitch independent tones |
| `Detune` | Cents |
| `Level` | Carrier: output level. Modulator: modulation index (1 is 2*pi radians) |
| `VelocitySens` | 0-1, how much the note velocity scales the level |
| `Env` | ADSR (`adsr.NewSetting(...)`, durations in ms) |

Voice:

| Setting | Default | Meaning |
|---|---|---|
| `Algorithm` | 0 | Operator routing, see above |
| `Feedback` | 0 | 0-1, operator 4 modulates itself (towards a saw, then noise) |
| `ModDepth` | 1 | Scales all modulator levels: overall brightness |
| `Glide` | 0 | Portamento in ms from the previous note of the voice |
| `VibratoRate`, `VibratoDepth`, `VibratoDelay` | 0 | Vibrato in Hz, semitones, and fade in time in ms |
| `Level` | 1 | Output level |

The note amplitude sets the volume. A separate `velocity` (0-1, default 1) in the note message
drives the velocity sensitivity:

```go
synth.ValueIn(templ.NewDurationTrigger(0.5, 0.2, template.Template{
	"pitch":    sequence.NewLoop(60.0, 64.0, 67.0),
	"velocity": bucket.NewLoop(0.5, 0.8, 1.0).SetMode(bucket.Random),
}).ValueIn(timer.WithTicks(float64(psalm.PPQ())).AddTo(p)))
```

**Value inputs**: 1 mod depth (connect an LFO for timbre sweeps), 2 feedback.

**Messages**: `pitch`, `frequency`, `velocity`, `algorithm`, `feedback`, `modDepth`, `glide`,
`vibratoRate`, `vibratoDepth`, `vibratoDelay`, `level`, and per operator `op1Ratio`,
`op1Fixed`, `op1Detune`, `op1Level`, `op1VelocitySens` (and the same for `op2`-`op4`).

### Presets

| Name | Function | Sound |
|---|---|---|
| `ePiano` | `EPianoPreset` | Tine electric piano, brighter when played harder |
| `bell` | `BellPreset` | Clear bell with inharmonic partials |
| `bass` | `BassPreset` | Punchy bass with a snappy attack |
| `brass` | `BrassPreset` | Brass section with a swelling attack and vibrato |
| `organ` | `OrganPreset` | Additive organ (sub, unison, octave, octave and fifth) |
| `mallet` | `MalletPreset` | Woody marimba like mallet |
| `pad` | `PadPreset` | Warm, slowly evolving pad |
| `pluck` | `PluckPreset` | Bright plucked string |
| `metallic` | `MetallicPreset` | Gong / metal plate |
| `lead` | `LeadPreset` | Bright lead with feedback edge, glide and vibrato |

More presets cover the instrument families of General MIDI. They are original FM designs, use
them as starting points:

| Family | Presets |
|---|---|
| Pianos and keys | `grandPiano`, `honkyTonk` (detuned second string), `harpsichord`, `clavinet` |
| Chromatic percussion | `celesta`, `glockenspiel`, `musicBox`, `xylophone`, `tubularBells` |
| Organs | `percussiveOrgan` (with a decaying third harmonic click), `churchOrgan`, `accordion` |
| Guitars | `nylonGuitar`, `steelGuitar`, `overdriveGuitar` |
| Basses | `uprightBass`, `slapBass`, `fretlessBass` (with glide) |
| Strings | `strings`, `pizzicato`, `harp` |
| Brass | `trumpet`, `frenchHorn` |
| Reeds and pipes | `saxophone`, `clarinet` and `oboe`, `flute` and `panFlute` (breath from a feedback noise operator) |
| Synth leads and pads | `squareLead`, `sawLead`, `warmPad`, `sweepPad` (brightens over a few seconds) |
| Ethnic | `sitar`, `kalimba`, `steelDrum`, `koto` |
| Percussion | `timpani`, `woodblock`, `taiko` |

The function names follow the preset names (`GrandPianoPreset`, `TubularBellsPreset`, ...). They
use the classic FM building blocks, handy when designing your own:

- A modulator at ratio 1 gives all harmonics (strings, brass, saw like), at ratio 2 only odd
  harmonics (`clarinet`, `squareLead`).
- A high ratio modulator with a very fast decay gives the attack of a hammer, pick or mallet
  (`grandPiano`, `xylophone`).
- Inharmonic ratios (1.41, 2.76, 3.5, ...) give bells, bars and drums.
- Feedback gives buzz (`sitar`, `overdriveGuitar`), a saw (`sawLead`), or at its maximum noise
  (breath in `flute`, the skin hit of `taiko`).

## dx

`synths/dx`

A 6 operator FM engine compatible with DX7 style voice data: the 32 classic algorithms and the
full set of voice parameters, so you can load SysEx banks (`.syx` files) and play them.

```go
patches, err := dx.LoadSysEx("my_bank.syx") // a 32 voice bulk dump, or single voice dumps
if err != nil {
	panic(err)
}

synth := dx.NewPoly(patches[0], 12, psalm.Stereo).AddTo(p)
dx.SetPatch(synth, patches[5]) // switch all voices to another patch
```

No factory sounds are included: load banks you own or that are shared for free use. Four
original patches are built in (`dx.Presets`: `glassKeys`, `woodBass`, `softBrass`, `bellPad`),
and `dx.InitVoice()` is the classic single sine starting point.

### Patches

A `dx.Patch` has six `Operator`s (`Ops[0]` is operator 1) and the voice settings, all in the
classic 0-99 style ranges:

| Operator setting | Meaning |
|---|---|
| `EGRates`, `EGLevels` | 4 rates and 4 levels: from level 4 the envelope moves to level 1, 2 and 3, holds level 3 while the note is held and returns to level 4 on release |
| `OutputLevel` | 0-99, about 0.75 dB per step |
| `FixedFreq`, `Coarse`, `Fine` | Ratio mode: coarse ratio (0 is 0.5) times 1 + fine/100. Fixed mode: 1, 10, 100 or 1000 Hz (coarse) times 10^(fine/100) |
| `Detune` | 0-14, 7 is centered |
| `BreakPoint`, `LeftDepth`, `RightDepth`, `LeftCurve`, `RightCurve` | Keyboard level scaling around the break point (39 is C3), curves -LIN, -EXP, +EXP, +LIN |
| `RateScaling` | 0-7, higher notes get faster envelopes |
| `VelocitySens` | 0-7, softer notes get a lower level |
| `AmpModSens` | 0-3, sensitivity to LFO amplitude modulation |

| Voice setting | Meaning |
|---|---|
| `Algorithm` | 0-31 (algorithm 1-32), see `dx.Algorithms` |
| `Feedback` | 0-7 |
| `PitchEGRates`, `PitchEGLevels` | Pitch envelope, level 50 is no shift |
| `OscKeySync` | Restart the operators at every note |
| `LFOSpeed`, `LFODelay`, `LFOWaveform` | LFO (triangle, saw down, saw up, square, sine, sample & hold) |
| `LFOPitchDepth`, `PitchModSens` | Vibrato depth |
| `LFOAmpDepth` | Tremolo depth (with the operators' `AmpModSens`) |
| `LFOKeySync` | Restart the LFO at every note |
| `Transpose` | 0-48, 24 is no transposition |
| `Name` | Up to 10 characters |

### SysEx

| Function | Meaning |
|---|---|
| `dx.LoadSysEx(path)` / `dx.ParseSysEx(data)` | Load a 32 voice bulk dump (4104 bytes) or one or more single voice dumps (163 bytes each), checksums are verified |
| `dx.EncodeBank(patches)` | Write a 32 voice bulk dump (padded with init voices) |
| `dx.EncodeSingle(patch)` | Write a single voice dump |

### Playing

Notes take an optional `velocity` (0-1, default 1) in their message for the velocity
sensitivity, the note amplitude sets the volume. Value input 1 (or message `modScale`) scales all
modulators, an overall brightness control that glides smoothly.

The engine maps the 0-99 parameters to sound with approximations of the original behavior
(level and rate curves, LFO speeds, feedback, pitch envelope range). Patches sound recognizably
like themselves, but not bit identical to the hardware. The approximations are collected in
`synths/dx/eg.go` and easy to tune. Like the original, rising envelope stages are faster than
falling ones at the same rate, and the output has a 10 Hz DC blocker (FM puts a sideband at 0 Hz
when operator frequencies coincide, the hardware removes it in its output stage).

`TestSysExBanks` plays every voice of every bank in `assets/dx7` (if present) and checks for NaN,
clipping, DC offset and voices that never finish.

## grains

`synths/grains`

A granular sampler you play like any synth: every note starts a stream of grains, short windowed
slices of a recording, played at the pitch of the note. Moving where the grains come from, how
dense and long they are and how much they spread turns any recording into pads, drones, keys,
textures or rhythmic stutters.

```go
choir := sndfile.MustMipMapSoundFile[float32]("assets/sounds/various/mixkit-angelical-choir-654.wav", 4)
pad := grains.NewPoly(grains.ChoirPad(choir, 60), 12).AddTo(p) // sample, root note
pad.ValueIn(lfo.New(true, 0.05, 0.2, 0.8, lfo.Sine).AddTo(p), 0, 1) // slowly move the position
```

`grains.New(cfg)` creates a single voice, `grains.NewPoly(cfg, n)` a `poly` synth with `n` voices.
Use mipmapped sound files, so grains that play faster don't alias.

### Settings

| Setting | Default | Meaning |
|---|---|---|
| `Sample` | | The recording (`sndfile.SoundFiler[float32]`) |
| `SoundBank` | | Optional, notes pick a sample with the `sound` key |
| `RootNote` | 60 | MIDI pitch of the recording: playing this note plays it at its own pitch |
| `Position` | 0.3 | 0-1, where in the recording the grains come from |
| `Scan` | 0 | How fast the position moves: 0 freezes a moment, 1 is the original speed, negative goes backwards. The position restarts at every note |
| `Spray` | 0.05 | Seconds, random spread of the grain positions |
| `Density` | 30 | Grains per second |
| `GrainSize` | 120 | Milliseconds |
| `PitchSpread` | 0 | Semitones, random detune per grain (chorus, smear) |
| `Octave` | 0 | 0-1, chance a grain plays an octave up (sparkle, shimmer) |
| `StereoSpread` | 0.5 | 0-1, random pan per grain |
| `Shape` | 0 | 0-1, grain window from smooth to percussive |
| `Reverse` | 0 | 0-1, chance a grain plays backwards |
| `Jitter` | 0.3 | 0-1, randomness of the time between grains |
| `VelocityDensity` | 0 | 0-1, how much the note velocity raises the density |
| `Normalize` | true | Play every recording at the same loudness (measured over the parts that sound) |
| `Level` | 1 | Output level |
| `AmpEnv` | | ADSR of the whole note (ms) |

The level stays about the same when the density, grain size or window shape changes. Notes take
an optional `velocity` (0-1, default 1), the note amplitude sets the volume.

**Value inputs**: 1 position, 2 density, 3 scan, 4 spray, 5 grain size. They only affect new
grains, so modulating them never clicks.

**Messages**: `pitch`, `frequency`, `velocity`, `sound`, `rootNote`, `position`, `scan`, `spray`,
`density`, `grainSize`, `pitchSpread`, `octave`, `stereoSpread`, `shape`, `reverse`, `jitter`,
`level`.

### Presets

Presets take the sample and its root note, every preset works with any recording:

| Name | Function | Character |
|---|---|---|
| `choirPad` | `ChoirPad` | Slow, wide pad of long grains drifting through the sample |
| `frozenDrone` | `FrozenDrone` | One frozen moment, a living drone |
| `glassKeys` | `GlassKeys` | Playable keys: dense short grains, a plucked envelope, denser when played harder |
| `texture` | `Texture` | Restless cloud: wide spray, pitch smear, reversed grains |
| `stutter` | `Stutter` | Rhythmic, percussive grains repeating small pieces |
| `shimmer` | `Shimmer` | Bright pad with grains an octave up |
| `pluck` | `Pluck` | Short plucked tone |

Many recordings only sound in part of the file (a decaying keys sample, a phrase with pauses), set
`Position` to a part that sounds.

## pulsar

`synths/pulsar`

Pulsar synthesis: every note is a train of pulses at the pitch of the note, and every pulse (a
pulsaret) is a short burst of a sine at the formant frequency, shaped by a window. No samples
needed. Three things shape the sound:

- the **formant** puts a spectral peak at a fixed frequency that stays put when the pitch changes,
  like a vowel (two formants make actual vowels);
- the **duty** sets how much of every period the pulse fills, short pulses are bright and thin,
  full pulses round;
- the **pitch**: at very low notes (a few Hz) the pulses become a rhythm, raising the pitch turns
  that rhythm into a tone.

```go
lead := pulsar.NewPoly(pulsar.VowelLeadPreset(), 2).AddTo(p)
lead.ValueIn(templ.NewDurationTrigger(0.5, 0.6, template.Template{
	"pitch":   seq(57.0, 60.0, 64.0),
	"formant": seq(700.0, 450.0, 320.0), // ah, oh, oo
}).ValueIn(timer.WithTicks(float64(psalm.PPQ())).AddTo(p)))
```

`pulsar.New(cfg)` creates a single voice, `pulsar.NewPoly(cfg, n)` a `poly` synth with `n` voices.

| Setting | Default | Meaning |
|---|---|---|
| `Formant` | 800 | Hz, the frequency inside every pulse (the spectral peak) |
| `Formant2`, `Formant2Level` | 0, 0.5 | A second formant in Hz (0 is off) and its level |
| `FormantTracking` | 0 | 0-1, how much the formants follow the pitch (relative to middle C). 0 is a fixed vowel, 1 the same timbre on every note |
| `Duty` | 0.5 | 0-1, the part of every period the pulse sounds |
| `Window` | `Hann` | Pulse envelope: `Hann` (smooth), `Gaussian` (narrow, purer formant), `Percussive` (struck attack, fast decay), `Rectangle` (hard edges, buzzy, some aliasing), `Tapered` (a rectangle with short smooth edges: buzzy without the aliasing) |
| `Masking` | 0 | 0-1, chance a pulse is skipped: crackling, stochastic textures |
| `Jitter` | 0 | 0-1, randomness of the pulse period. Small values (below 0.005) add breath to a tone, larger ones make it rough |
| `StereoSpread` | 0 | 0-1, random pan per pulse. For sparse textures: on a tone the pan changes at the pitch rate, which sounds like noise |
| `NoteSpread` | 0 | 0-1, random pan per note, to widen chords and pads |
| `FormantEnv`, `FormantEnvAmount` | | ADSR (ms) and how many octaves it raises the formants |
| `FormantLFORate`, `FormantLFODepth` | 0, 0 | Hz and octaves of a sine LFO on the formants |
| `VibratoRate`, `VibratoDepth` | 0, 0 | Hz and semitones |
| `Glide` | 0 | Milliseconds from the previous note of the voice |
| `Level` | 1 | Output level |
| `AmpEnv` | | ADSR of the note (ms) |

Every pulse is scaled by its measured energy, so the level stays about the same when the duty,
window or formants change. Pulses shorter than about one formant cycle hold too little of the
formant to be brought up to level, they get quieter instead of turning into spikes. Sparse pulse
trains (low notes, masking) have a high peak for their loudness.

**Value inputs**: 1 formant, 2 duty, 3 masking, 4 second formant. Settings are picked up by the
next pulse, so modulating them never clicks.

**Messages**: `pitch`, `frequency`, `formant`, `formant2`, `formant2Level`, `formantTracking`,
`duty`, `window` (0-4), `masking`, `jitter`, `stereoSpread`, `noteSpread`, `formantEnvAmount`, `formantLFORate`,
`formantLFODepth`, `vibratoRate`, `vibratoDepth`, `glide`, `level`.

### Presets

| Name | Function | Character |
|---|---|---|
| `vowelLead` | `VowelLeadPreset` | Singing "ah" lead with two formants, formant drift, vibrato and glide |
| `buzzBass` | `BuzzBassPreset` | Buzzy bass with tapered pulses, the formant snaps down from two octaves up like a plucked filter |
| `glass` | `GlassPreset` | Bright glassy keys: high formants, a struck attack and a long decay |
| `formantPad` | `FormantPadPreset` | Slow, wide pad sweeping through vowel colours |
| `laser` | `LaserPreset` | Falling chirp, the formant sweeps down four octaves |
| `rhythm` | `RhythmPreset` | For very low notes (send a `frequency` of a few Hz): every pulse is a tick |
| `crackle` | `CracklePreset` | For low notes (30-100 Hz): masked, jittered pulses crackling across the stereo field |
| `choir` | `ChoirPreset` | Soft "ooh" choir pad with breath and slow vibrato |
| `strings` | `StringsPreset` | Bowed string section: short bright pulses, slow bow, vibrato |
| `brass` | `BrassPreset` | Brassy lead, the formant opens up during the attack |
| `mallet` | `MalletPreset` | Struck wooden mallet with a bright strike and quick decay |
| `subBass` | `SubBassPreset` | Deep round bass, almost a sine with a touch of the octave |
| `wobbleBass` | `WobbleBassPreset` | Talking bass, a fast LFO swings the formant like a wah, tapered pulses |
| `insects` | `InsectsPreset` | For low notes (30-60 Hz): chirping insects at night |
| `raindrops` | `RaindropsPreset` | For very low notes (8-20 Hz): irregular drops at random pitches and places |
| `motor` | `MotorPreset` | For low notes (15-40 Hz): a running engine, raise the frequency to rev it up (long glide) |

## drums

`synths/drums`

`drums.NewDrums(soundBank, numVoices, numChannels)` is a `poly` synth of sample `player` voices.
Each trigger plays a sound from the bank, chosen with the `"sound"` key in the note message
(`"speed"` changes its pitch):

```go
drms := drums.NewDrums(soundBank, 16, 2).AddTo(p)
drms.ValueIn(templ.NewTrigger(0.5, template.Template{"sound": "kick", "speed": 1.0}).
	ValueIn(timer.WithTicks(float64(psalm.PPQ())).AddTo(p)))
```
