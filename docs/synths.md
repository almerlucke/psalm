# Synths

Synths: [poly](#poly) (voice allocation), [subtractive](#subtractive), [fm](#fm), [dx](#dx) (6 operator, SysEx), [grains](#grains) (granular sampler), [pulsar](#pulsar), [wavetable](#wavetable), [piano](#piano), [drums](#drums), [sampler](#sampler) (SFZ), [organ](#organ), [waveguide](#waveguide) (plucked and bowed strings), [modal](#modal) (mallets and bells), [drumsynth](#drumsynth), [vocal](#vocal) (voices and choirs), [pd](#pd) (phase distortion), [westcoast](#westcoast), [supersaw](#supersaw).

Presets of all synths are calibrated to the same loudness: a note at amplitude 1 plays at -12 LUFS
(or peaks at -1 dBFS for percussive sounds), see [conventions](conventions.md#levels).

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
| `{"command": "voice", ...}`, or a message without a command | Passes the whole message to all voices (change settings of every voice, `AutomateMessage` of an arrangement works this way) |

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

## wavetable

`synths/wavetable`

A wavetable synth: the oscillator plays one frame of a table (a single cycle) and morphs through the
frames as the **position** moves from 0 to 1. Stacked in unison, with a sine sub and a state variable
filter. Tables are loaded from sound files of consecutive frames, like the 2048 sample tables in
`assets/wavetables2048`.

```go
table := wavetable.MustLoad("assets/wavetables2048/DN Harm.wav", 2048)
pad := wavetable.NewPoly(wavetable.PadPreset(table), 16).AddTo(p)
```

`wavetable.New(cfg)` creates a single voice, `wavetable.NewPoly(cfg, n)` a `poly` synth with `n`
voices.

### Tables

| Function | Meaning |
|---|---|
| `Load(path, frameSize)`, `MustLoad` | A table from a sound file (first channel) of consecutive frames |
| `NewTable(samples, frameSize)` | A table from samples |
| `FromFunc(numFrames, frameSize, f)` | A table from a function of the position and the phase. Every cycle is drawn 16 times oversampled and reduced to the harmonics that fit, so shapes with jumps (pulses, sync) come out practically alias free |
| `FromHarmonics(numFrames, frameSize, f)` | A table from the amplitudes of the harmonics (sines) of every frame, exact |
| `FromSpectrum(numFrames, frameSize, f)` | A table from the sine and cosine amplitudes of the harmonics, for control over the phases |
| `Basic()` | Sine, triangle, sawtooth and square, played when the config has no table |
| `table.Save(path)` | Writes the table as a 32 bit float WAV of consecutive frames, the format most wavetable synths load |

Every frame gets its DC removed, the table is normalized to a peak of 1. Each frame is stored as
band limited levels, each with half the harmonics of the one before. A note reads the levels whose
highest harmonic stays below the Nyquist frequency, crossfading between them as the pitch moves, so
high notes and glides never alias. Load a table once and share it between voices and synths.

### Generated tables

`synths/wavetable/gen` generates tables (2048 sample frames): every generator sweeps one synthesis
parameter over the frames, so moving the position plays the sweep.

```go
pad := wavetable.NewPoly(wavetable.PadPreset(gen.Random(64, 6, 48, 7)), 16).AddTo(p)
_ = gen.Fold(64, 6, 0.2).Save("fold.wav")
```

| Generator | The position sweeps |
|---|---|
| `Brightness(numFrames)` | From a sine to a full sawtooth, the harmonics fading in from low to high like an opening lowpass |
| `PWM(numFrames, minWidth)` | A pulse from a square to `minWidth` (for example 0.02) |
| `Sync(numFrames, maxRatio)` | A hard synced sawtooth from ratio 1 (a plain sawtooth) to `maxRatio` times the cycle |
| `FM(numFrames, ratio, maxIndex)` | Two operator FM, the index from 0 (a sine) to `maxIndex`. The ratio is a whole number: 1 brightens towards a sawtooth, higher ratios give bells and metal |
| `Fold(numFrames, maxGain, bias)` | A sine into a wavefolder, the gain from 1 to `maxGain`. A bias (-1 to 1) adds even harmonics |
| `Vowels(numFrames, pitch)` | Through the vowels a, e, i, o, u: a voice like spectrum shaped by formants placed for a note of `pitch` Hz, most natural near that pitch |
| `Drawbars(numFrames, numHarmonics)` | Organ harmonics fading in one by one, from the fundamental to `numHarmonics` |
| `Random(numFrames, keyframes, numHarmonics, seed)` | Smooth crossfades between keyframes of random spectra, every seed another table |
| `Filter(numFrames, q)` | A resonant lowpass opening over a sawtooth, the cutoff from the fundamental to harmonic 128 (`q` 0.7 smooth, 4 sharp, 10 whistling) |
| `Resonance(numFrames, maxRatio)` | The resonant waveform of phase distortion synths (Casio CZ), the resonance rising to `maxRatio` times the cycle |
| `OddEven(numFrames)` | Odd harmonics only (hollow, square like), through all (sawtooth), to the fundamental with only even harmonics |
| `Comb(numFrames, maxDelay)` | A sawtooth mixed with a copy delayed up to `maxDelay` (0-0.5) of the cycle, notches like a flanger. Half a cycle cancels the odd harmonics, an octave up |
| `Saturate(numFrames, maxDrive)` | A sine into a soft clipper, the drive from 1 to `maxDrive`, towards a square |
| `Quantize(numFrames, minLevels)` | A sine at fewer and fewer levels, from 8 bits (256) to `minLevels` |
| `Steps(numFrames, minSteps)` | A sine held in fewer and fewer steps per cycle, from 256 to `minSteps`: mirror images around multiples of the step count, the stair case sound of early wavetable synths |
| `Crossfade(numFrames, shapes...)` | Through any single cycle shapes in turn: `gen.Sine`, `gen.Triangle`, `gen.Saw`, `gen.Square` or your own functions of the phase |
| `FromSound(samples, numFrames)`, `LoadSound(path, numFrames)` | Slices of a recording taken evenly through the parts that sound (silence is skipped), each looped with a short crossfade |

A slice of a recording is not a single cycle. Its 2048 samples last one cycle at 21.5 Hz, so a note
plays the slice that many times faster: at 220 Hz the recording is about 10 times sped up and
sounds bright, buzzy or vocal. Play sliced tables low and without a lowpass, near 21.5 Hz (MIDI
note 5) a slice plays at its own speed.

The additive generators (vowels, drawbars, random) spread the phases of the harmonics (Schroeder
phases): it sounds the same, but the wave has a lower peak, so the table plays louder.

### Settings

| Setting | Default | Meaning |
|---|---|---|
| `Table` | nil | The table (`Basic()` when nil) |
| `Position` | 0 | 0-1, morphs through the frames |
| `PositionEnv`, `PositionEnvAmount` | | ADSR (ms) and how far it moves the position (-1 to 1) |
| `PositionLFORate`, `PositionLFODepth` | 0, 0 | Hz and how far a sine LFO moves the position either way |
| `Normalize` | true | Brings every frame to the same loudness (quiet frames at most 4 times), so moving the position changes the timbre, not the level |
| `Unison` | 1 | 1-7 oscillators |
| `Detune` | 20 | Cents between the lowest and highest unison oscillator |
| `Spread` | 0.7 | 0-1, stereo width of the unison oscillators |
| `SubLevel` | 0 | Sine an octave below |
| `Filter` | `FilterOff` | `FilterOff`, `LowPass`, `BandPass`, `HighPass` (state variable filter) |
| `Cutoff`, `Resonance` | 8000, 0.2 | Hz, 0-1 |
| `FilterEnv`, `FilterEnvAmount` | | ADSR (ms) and the octaves it adds to the cutoff |
| `KeyTrack` | 0.5 | 0-1, how much the cutoff follows the pitch (relative to middle C) |
| `VibratoRate`, `VibratoDepth` | 0, 0 | Hz and semitones |
| `Glide` | 0 | Milliseconds from the previous note of the voice |
| `Level` | 0.6 | Output level |
| `AmpEnv` | | ADSR of the note (ms) |

Unison keeps the level of a single oscillator. With a bright table the high frames lose the
harmonics above the Nyquist frequency on high notes, so they get a little quieter there.

**Value inputs**: 1 position, 2 cutoff, 3 resonance. The position follows changes within a few
milliseconds and the filter glides over a buffer, so modulating them (for example with an LFO)
never clicks.

**Messages**: `pitch`, `frequency`, `table` (a `*wavetable.Table`), `position`, `positionEnvAmount`,
`positionLFORate`, `positionLFODepth`, `normalize`, `unison`, `detune`, `spread`, `subLevel`,
`filter` (0-3), `cutoff`, `resonance`, `filterEnvAmount`, `keyTrack`, `vibratoRate`, `vibratoDepth`,
`glide`, `level`.

### Presets

Presets take the table, every preset works with any table. Tables differ a lot in what their frames
hold, so try moving the `Position`.

| Name | Function | Character |
|---|---|---|
| `pad` | `PadPreset` | Wide, slow pad: five detuned oscillators drifting slowly through the table |
| `lead` | `LeadPreset` | Singing lead: three oscillators, vibrato, glide, a filter that opens at every note |
| `pluck` | `PluckPreset` | Bright pluck: the position jumps up the table and falls back while the filter closes |
| `bass` | `BassPreset` | Solid bass: one oscillator, a sine sub and a snapping lowpass |
| `growl` | `GrowlPreset` | Moving bass: a fast LFO rocks the position |
| `sweep` | `SweepPreset` | Evolving texture: the position travels through the whole table over four seconds |
| `keys` | `KeysPreset` | Bell like keys: a detuned pair, struck attack, long decay |
| `anthem` | `AnthemPreset` | Huge stack of seven wide oscillators for chords |
| `strings` | `StringsPreset` | String section: five gently detuned oscillators, a slow bow, a little vibrato |
| `vox` | `VoxPreset` | Vocal like lead: a resonant bandpass picks out a formant, vibrato and glide |
| `brass` | `BrassPreset` | Brass section: the filter swells open in the attack and settles |
| `organ` | `OrganPreset` | Steady organ: one oscillator with a sub, full sustain, fast release |
| `reese` | `ReesePreset` | Reese bass: two oscillators detuned far apart, beating and phasing, with a sub |
| `acid` | `AcidPreset` | Squelchy bass line: a resonant lowpass snapping open, glide |
| `stab` | `StabPreset` | Short chord stab: four wide oscillators, a fast closing filter |
| `drone` | `DronePreset` | Breathing drone: seven close oscillators, a very slow sweep, long fades |
| `glass` | `GlassPreset` | Thin, airy bell: a highpass removes the body |
| `riser` | `RiserPreset` | Tension builder: over six seconds the position climbs and the filter opens, hold the note |

Tables that suit them in `assets/wavetables2048`: dark and harmonic `DN Harm` and `VS Morph3`
for pads and leads, `MS 20 Pulse2 PS1` for basses, `OPFoldFMFB`, `MSelfFB02` and `Add Synth3 Icarus`
from soft to harsh for sweeps and growls, `Add Bell` and `Melda CustumWave3 IcarusBell` for keys,
`KaossDrum2` (noise like, little below a few kHz) for hits.

## piano

`synths/piano`

A physically modelled piano, no samples. Every key has one to three strings (one in the low bass,
two up to the tenor, three above), each a digital waveguide: a delay line closed by

- a **loss filter** that sets how long the string rings (about 20 seconds in the low bass, under 2
  at the top) and makes the highs die faster;
- a chain of **allpass filters** for the stiffness of the string: they stretch the partials sharp
  (inharmonicity), fitted per key to the stretch of a real grand (strongest at the top);
- a **tuning allpass** that puts the fundamental on pitch to a fraction of a cent.

What makes it a piano:

- **Two-stage decay.** Every string vibrates in two polarizations: one passes its energy to the
  soundboard fast (the loud prompt sound, the first second), the other keeps it (the long, quiet
  aftersound). The strings of a key are a little out of tune with each other (`Unison`) and share
  the bridge: what they do together drains into the soundboard, what they do against each other
  lasts, and the bridge passes energy between them and to the other polarization (Weinreich's
  coupled strings). So a note falls fast, then lingers and breathes as the strings trade energy.
- **A felt hammer that meets the string.** Every strike is simulated: the hammer (its mass, heavier
  in the bass) flies at the string, its felt stiffens steeply as it is compressed and pushes harder
  going in than coming out (hysteresis), the string gives way under it and the waves leaving the
  strike point come back from both ends. The force that results is the strike: a few milliseconds
  in the bass, a fraction in the treble, shorter and sharper the harder the key is played (a soft
  note is round, a loud one bright), and on some keys the hammer bounces and hits twice. The felt
  of every key is fitted once (when the piano is made) so velocity 0.8 sounds as voiced. The strike
  position leaves out the harmonics with a node where the hammer hits, and the soundboard knock is
  heard with it: a short impact, shorter in the treble (from about 80 ms in the bass to about 12 ms
  at the top), so it never outlasts the short high notes.
- **A string played hard settles.** Its own motion stretches it, so it starts a little sharp (up to
  about 5 cents in the bass at full velocity, a cent at the top) and settles within a few hundred
  milliseconds.
- **Irregular partials.** The partials of a real string don't die away evenly: the bridge takes
  more at some frequencies than at others. Every string has a slightly different pattern of
  partials that decay faster (up to 3 dB per second, with `Variation`).
- **A soundboard** every string sounds through: a board of spruce made of its modes (a few hundred:
  sparse in the bass, one every 20-odd Hz higher up, with the low Q of wood, radiating the bass
  weakly and rolling off the highs), heard from a different place on every side of the stereo image.
  The strings sound through it by convolution (`Body`), or through a measured soundboard response
  (`SoundboardIR`). Every key is voiced for the board, like a technician voicing the hammers: the
  gain that evens out where the key's partials land on the board's modes.
- **The bark of the bass.** A string played hard stretches as it moves, which rings its longitudinal
  mode and adds phantom partials at the sums of its partials: the metallic bark of loud bass notes,
  growing with the velocity (`Bark`).
- **The action** is heard: the dampers landing on the still vibrating strings, the key coming up,
  the sustain pedal lifting all dampers (a soft swish) and letting them land (a thud) (`Mechanics`).
- **Sympathetic resonance.** The strings of all 88 keys ring along with what is played when their
  dampers are up: all of them with the sustain pedal down, the keys held down (also silently), and
  the top keys, which have no dampers (`Resonance`). With the pedal down a halo blooms as the notes
  die away.
- **Dampers** stop the strings when a key is released (slower in the bass). The **sustain pedal**
  lifts them; between up and down it lifts them partly (**half pedaling**): the dampers touch the
  strings lightly, a released note rings on shorter, and the heavy bass strings push the light touch
  away more easily than the treble, so half pedaling clears the treble and lets the bass ring. The
  **sostenuto pedal** keeps the dampers of the keys held when it goes down up until it comes up:
  a bass note caught by it rings under chords played dry above it. The **soft pedal** (una corda)
  shifts the hammers so they strike one string less with a softer part of the felt: quieter and
  veiled.
- **Stretch tuning.** The partials of stiff strings are sharp, so the instrument is tuned like a
  tuner does it: octaves widened until they are clean (the 2nd partial of the lower note on the upper
  note in the treble, 4:2 and 6:3 lower down), which gives the Railsback curve: the low bass some 15
  cents flat, the top some 45 cents sharp, derived from the strings' own inharmonicity (`Stretch`).
- **No two notes alike.** Every key's hammer is a little harder or softer, hits a little elsewhere,
  every key rings a little longer or shorter and is a little louder or softer, as on any real
  instrument; every strike varies a little in force and felt, and the hammer reaches the strings of
  a key a fraction of a millisecond apart (`Variation`). The strikes draw from the random source of
  the patch, so `psalm.Seed` repeats them.
- **A listening position**: on the bench (`Player`), close microphones over the open lid
  (`Studio`), or in the audience (`Hall`): how wide the keys spread, the early reflections of the
  lid, the floor and the stage, and from the audience a little air (`Position`). A reverb adds the
  room after that.
- **Calibrated against a recorded concert grand** (single notes of the University of Iowa Steinway
  at pianissimo, mezzo forte and fortissimo): the decay of every register (a fast first second,
  roughly 6 to 18 dB per second, then a slow aftersound), the brightness of the attack at every
  dynamic (a near pure tone in the treble), the dynamic range (pianissimo to fortissimo some 25 dB
  in the middle, the level follows the speed of the hammer), the stretch tuning and the level over
  the keyboard (even up to C5, a few dB softer from C6, the top octave some 13-16 dB softer).

```go
grand := piano.NewPoly(piano.GrandPreset(), 24).AddTo(p) // a voice per sounding key

// the pedals: messages without a command reach the whole instrument
arrangement.AutomateMessage(arrange.Lane{Sections: map[string]arrange.Curve{"verse": arrange.Const(1)}}, grand, "sustain")
grand.ReceiveMessage(psalm.Message{"sustain": 0.5})   // half pedal
grand.ReceiveMessage(psalm.Message{"sostenuto": 1.0}) // catch the keys held now
grand.ReceiveMessage(psalm.Message{"soft": 1.0})
```

`piano.NewPoly(cfg, n)` creates the instrument (a `*piano.Piano`: `n` voices, the strings that ring
along and the soundboard), it takes the messages of a poly synth (triggers with a duration, or
`noteOn` and `noteOff`). `piano.New(cfg)` is a single voice, without soundboard and resonance. A
voice sounds until its strings have died away, the dampers stopped them, or as long as its key is
held; with the pedal down that can take many seconds. The voices are handed out like piano keys: a
key that still sounds is struck again on its own strings (adding to their vibration, no new voice,
nothing cut off); a new key takes a free voice or, when all sound, the quietest one (released keys
before held ones). The strings and the hammers of all keys are fitted when the piano is created
(some tens of milliseconds), so playing it does no fitting: a silent piano rests from the start, a
note costs a fraction of a millisecond. Sixteen sounding voices with the pedal down take under 10%
of a core (the strings ringing along run at half the sample rate); a piano that is silent rests. The
instrument sounds 128 samples (3 ms) late, the time the soundboard convolution takes; it reports
that latency, so a mixer lines the other strips up with it.

**Note messages**: `pitch` (or `frequency`), `velocity` (0-1, default 0.8: the tone always, the
level by `VelocitySens`), and every setting below. **Instrument messages**, the pedals: `sustain`
(0 up, 1 down, half pedaling in between: from 0.3 to 0.7 of the travel the dampers lift), `sostenuto`
and `soft` (above 0.5 is down).

| Setting | Default | Meaning |
|---|---|---|
| `Hardness` | 0.5 | 0-1, hardness of the hammer felt: soft and round to hard and bright |
| `StrikePosition` | 0.125 | Where the hammer hits, 0.05-0.3 of the string: the harmonics with a node there are weaker (1/8 is classic, nearer the end is brighter, thinner) |
| `Soundboard` | 0.6 | 0-1, the knock of the soundboard in the strike |
| `Body` | 0.6 | 0-1, how much of the sound comes through the soundboard (at 1 a little of the direct strings stays, for clarity) |
| `SoundboardIR` | nil | A measured soundboard response (one slice per side, at the sample rate) instead of the modelled board |
| `Bark` | 0.5 | 0-1, the bark of the bass strings played hard (phantom partials, longitudinal mode) |
| `Mechanics` | 0.5 | 0-1, the noise of the action: dampers, keys, the sustain pedal |
| `Stretch` | 1 | How much of a tuner's stretch tuning, 0 is equal temperament |
| `Variation` | 0.5 | 0-1, how different the keys and every strike are, and how irregularly the partials decay |
| `Treble` | 0 | dB, the top octave louder or softer than voiced |
| `Position` | `Player` | `Player`, `Studio` or `Hall`: where the piano is heard from |
| `Resonance` | 0.5 | 0-1, sympathetic resonance of the strings whose dampers are up |
| `Stiffness` | 1 | Inharmonicity, 1 is a grand, more an upright (or a toy, a struck rod), 0 perfectly harmonic |
| `Decay` | 1 | Scales the ring times |
| `Damping` | 1 | How much faster the highs die |
| `Unison` | 1 | Cents the strings of a key are apart: around 1 is in tune with a slow beat, 10+ honky tonk |
| `Strings` | 3 | Strings per key at most (1-3): fewer are thinner and cheaper |
| `DamperTime` | 0.25 | Seconds a released key rings on (longer in the bass) |
| `Width` | 0.6 | 0-1, the keys spread from left (low) to right (high), as the player hears it |
| `VelocitySens` | 0.7 | 0-1, how much the velocity changes the level (0.7 is a recorded grand: pianissimo to fortissimo some 25 dB) |
| `Level` | 1 | Output level |

**Presets** (`piano.Presets` by name):

| Preset | Character |
|---|---|
| `grand` | A concert grand as it is usually recorded, from close microphones: warm hammers, a lot of soundboard and wood, strings that beat a little |
| `concert` | The grand from the audience: the lid throwing the sound into the hall, brighter hammers to carry |
| `warm` | A grand in a jazz club: close, round hammers, a rich body, the strings ringing along freely |
| `baby` | A small grand: shorter, stiffer strings, less sustain, a boxier soundboard |
| `bright` | Hard hammers for pop and rock, clean and present |
| `upright` | Stiffer, shorter, boxier, a little out of tune |
| `vintage` | An old upright: worn hammers gone hard, strings drifting apart, a noisy action |
| `felt` | A felt piano, soft and close, for quiet music and ambient |
| `cinematic` | Soft, dark hammers from the hall, a lot of sympathetic resonance: the slow film-score piano |
| `honkyTonk` | Strings far out of tune with each other |
| `toy` | One short string per key, struck hard in the middle: a bright, hollow, bell like plink |

A struck string peaks high above its loudness, so the
presets are calibrated to a -1 dB peak at velocity 0.8 (about -14 LUFS); played harder they go
over, leave headroom or use the mixer's limiter.

## drums

`synths/drums`

`drums.NewDrums(soundBank, numVoices, numChannels)` is a `poly` synth of sample `player` voices.
Each trigger plays a sound from the bank, chosen with the `"sound"` key in the note message
(`"speed"` changes its pitch). A note off fades the sound out in 8 ms (a choke): the step
sequencer (`stepseq`, see [Controls](controls.md#stepseq)) uses it for choke groups.

```go
drms := drums.NewDrums(soundBank, 16, 2).AddTo(p)
drms.ValueIn(templ.NewTrigger(0.5, template.Template{"sound": "kick", "speed": 1.0}).
	ValueIn(timer.WithTicks(float64(psalm.PPQ())).AddTo(p)))
```

## sampler

`synths/sampler`

A multi-sample instrument: recordings of an instrument spread over its keys and played at
several strengths. Each recording is a region: the keys and velocities it answers, the key it
was recorded at, how it loops and how it fades. A note plays every region that matches it, so
regions layer; alternate recordings of the same note take turns (round robin) or are picked at
random. Instruments load from SFZ, the plain text format most free sample libraries come in, or
are built in code.

```go
inst, err := sampler.LoadSFZ("Salamander/piano.sfz")       // or sampler.MustSFZ
piano := sampler.NewPoly(sampler.DefaultConfig(inst), 32).AddTo(p)

// in code: a sound from a file or from memory, the keys it plays
snd, _ := sampler.LoadFile("samples/c4.wav")
low := sampler.NewRegion(snd).Keys(0, 64, 60)               // keys 0-64, recorded at 60
inst = sampler.MustInstrument(low, high)
```

`sampler.New(cfg)` is a single voice, `sampler.NewPoly(cfg, n)` a `poly` synth whose voices share
the round robin turns and the choke groups. An instrument doesn't change once built: one can play
in many synths. `LoadSFZWith(path, sampler.MipMaps(4))` keeps band limited copies of every sound,
for regions played far above their key.

### Regions

Start a region with `NewRegion(sound)` (all keys and velocities, root 60, in tune, full velocity
tracking, a short release):

| Field | SFZ | Meaning |
|---|---|---|
| `LoKey`, `HiKey` | `lokey`, `hikey`, `key` | The keys it plays (note names work in SFZ: `c4` is 60) |
| `LoVel`, `HiVel` | `lovel`, `hivel` | The velocities it plays, 1-127 |
| `Root` | `pitch_keycenter` | The key it was recorded at |
| `KeyTrack` | `pitch_keytrack` | Cents per key: 100 in tune, 0 one pitch on every key (drums) |
| `Transpose`, `Tune` | `transpose`, `tune` | Semitones, cents |
| `Volume`, `Amp` | `volume`, `amplitude` | dB, and a linear gain |
| `Pan` | `pan` | -1 to 1 (a stereo sound is balanced) |
| `VelTrack` | `amp_veltrack` | 0-1, how much the velocity sets the loudness (velocity squared) |
| `Offset`, `End` | `offset`, `end` | The frames it plays |
| `Loop` | `loop_mode` | `NoLoop`, `OneShot` (ignores note offs), `LoopContinuous`, `LoopSustain` (loops while held, then plays on) |
| `LoopStart`, `LoopEnd` | `loop_start`, `loop_end` | Frames of the loop |
| `LoopCrossfade` | `loop_crossfade` | Seconds the end of the loop fades into its start |
| `Trigger` | `trigger` | `OnAttack`, or `OnRelease`: plays when the key goes up |
| `RtDecay` | `rt_decay` | dB per second a release region is softer the longer the key was down |
| `Group`, `OffBy`, `OffMode` | `group`, `off_by`, `off_mode` | Choke groups: a region of group `OffBy` stops this one |
| `SeqLength`, `SeqPosition` | `seq_length`, `seq_position` | Round robin |
| `LoRand`, `HiRand` | `lorand`, `hirand` | Random choice per note |
| `Env` | `ampeg_delay`, `_attack`, `_hold`, `_decay`, `_sustain`, `_release` | The amp envelope (seconds, sustain 0-1) |
| `Cutoff`, `Resonance`, `Filter`, `FilVelTrack` | `cutoff`, `resonance`, `fil_type`, `fil_veltrack` | A filter per region |
| `SwLoKey`, `SwHiKey`, `SwLast`, `SwDefault`, `SwLabel` | `sw_lokey`, `sw_hikey`, `sw_last`, `sw_default`, `sw_label` | Keyswitches: the keys `SwLoKey`-`SwHiKey` choose the articulation instead of playing; the region plays while the last one pressed is `SwLast` |
| `Fades` | `xfin_lovel` ... `xfout_hikey`, `xfin_loccN` ... `xfout_hiccN`, `xf_velcurve`, `xf_keycurve`, `xf_cccurve` | Crossfades by velocity, key or a controller (power or `Linear`); the ones by a controller follow it while the note sounds |
| `CCGain` | `gain_ccN`, `volume_onccN` | dB a controller adds at its top |
| `CCEnv` | `ampeg_attackccN` (and `delay`, `hold`, `decay`, `sustain`, `release`) | What a controller adds to the envelope when a note starts |
| `AmpRandom`, `PitchRandom`, `DelayRandom` | `amp_random`, `pitch_random`, `delay_random` | Every note a little different: dB and cents spread around the level and the pitch, seconds later |
| `Delay` | `delay` | Seconds the region starts after the note |
| `VelEnv` | `ampeg_vel2delay` ... `ampeg_vel2release` | Added to the envelope at full velocity |
| `LFOs` | `pitchlfo_*`, `amplfo_*`, `lfoN_freq`, `_wave`, `_delay`, `_fade`, `_pitch`, `_volume` | Vibrato (cents) and tremolo (dB), with a delay and a fade in |
| `EQ` | `eq1_*` ... `eq3_*` (`freq`, `bw`, `gain`) | Peaking filters, bandwidth in octaves |
| `Width` | `width` | A stereo sound's width in percent: 100 as it is, 0 mono |

Controllers reach the sampler as messages, `"cc1"` (the mod wheel) to `"cc127"`, 0-1: give a
keyboard `Controls: map[int]string{1: "cc1"}`. `Instrument.Switches()` lists the keyswitches.

The SFZ loader knows `<control>` (`default_path`, `note_offset`, `octave_offset`, `set_ccN`), `<global>`,
`<master>`, `<group>`, `<region>`, `#define` and `#include`. Opcodes it doesn't know are counted
in `Instrument.Unsupported`. Sounds load with libsndfile: wav, aiff, flac, ogg.

Notes take `pitch` (or `frequency`) and `velocity` (0-1). The levels are those of the recordings.

The sounds an SFZ file names are found from its folder (and its `default_path`). When they live
elsewhere, give a loader that finds them: `LoadSFZWith` hands it every path as the file has it.

```go
inst, err := sampler.LoadSFZWith("scripts/piano.sfz", func(path string) (sampler.Sound, error) {
	rel, _ := filepath.Rel("scripts", path)                // samples/C4.wav
	return sampler.LoadFile(filepath.Join("/data/piano", rel)) // /data/piano/samples/C4.wav
})
```

`cmd/examples/sfz` plays an SFZ instrument from a MIDI keyboard, with `-sounds` and `-rename` for
sounds kept elsewhere.

## organ

`synths/organ`

A tonewheel organ: 91 wheels spinning all the time, which the keys tap at the pitches of the nine
drawbars (16' 5 1/3' 8' 4' 2 2/3' 2' 1 3/5' 1 1/3' 1', 3 dB a step). Like the real one it is a
whole instrument, not a set of voices: keys sharing a wheel share its phase, the wheels fold back
at the ends, the percussion sounds on the first key of a phrase only (single trigger) and the
vibrato scanner moves the whole sound. Put the rotary speaker after it.

```go
org := organ.New(organ.Jazz()).AddTo(p)
leslie := rotary.New(rotary.DefaultConfig()).AddTo(p).In(org)
org.Receive(psalm.Message{"drawbars": "888000000"}, 0)
```

It takes the messages of a poly synth (triggers with a duration, or noteOn and noteOff), and the
sustain pedal.

| Setting | Meaning |
|---|---|
| `Drawbars` | `[9]float64`, 0-8; `organ.Bars("888000000")` reads the way organists write it |
| `Percussion` | `PercOff`, `PercSecond` (4'), `PercThird` (2 2/3'); with it on the 1' is silent |
| `PercFast`, `PercSoft` | A short decay, 6 dB softer |
| `Vibrato` | `VibratoOff`, `V1`-`V3`, `C1`-`C3` (chorus: mixed with the dry sound) |
| `VibratoHz` | The scanner's speed (0 is 6.9 Hz) |
| `Click` | 0-1, the key contacts |
| `Drive` | 0-1, the preamp saturating |

**Value inputs**: 1-9 the drawbars. **Messages**: `drawbars`, `drawbar1`-`drawbar9`,
`percussion`, `percFast`, `percSoft`, `vibrato`, `vibratoHz`, `click`, `drive`, `level`,
`sustain`.

| Preset | Character |
|---|---|
| `jazz` | 888000000, third percussion fast |
| `gospel` | 888800000, deep chorus |
| `full` | Every drawbar out |
| `rock` | 888000000 driven, chorus, more click |
| `ballad` | 806004000, soft, chorus |
| `flute` | 008400000, vibrato |
| `reed` | 800000888, nasal |
| `whistle` | 800000008 |

## waveguide

`synths/waveguide`

Strings that are plucked or bowed: a wave travelling up and down a string, fed by a pluck or kept
going by a bow, heard through the body of the instrument. The pluck is an extended Karplus-Strong
string (in tune within a few cents over the whole range); the bow is the McIntyre, Schumacher and
Woodhouse model after the STK: the bow sticks and slips, which keeps the string going as long as
it moves.

```go
guitar := waveguide.NewPoly(waveguide.Guitar(), 8).AddTo(p)
cello := waveguide.NewPoly(waveguide.Cello(), 4).AddTo(p)
```

| Setting | Model | Meaning |
|---|---|---|
| `Model` | | `Pluck` or `Bow` |
| `Decay` | pluck | Seconds the string rings at 220 Hz; lower strings ring longer |
| `Damping` | pluck | 0-1, how much faster the high partials die |
| `Brightness` | pluck | 0-1, a soft finger to a hard pick; the velocity adds |
| `PickPosition` | pluck | 0-0.5 from the bridge: thin near it, round in the middle (the partials with a node there are missing) |
| `Release` | pluck | Seconds the string dies in once let go, 0 lets it ring (a harp) |
| `Pressure` | bow | 0-1, airy to gritty |
| `BowPosition` | bow | 0.02-0.5 from the bridge |
| `BowAttack`, `BowRelease` | bow | Seconds |
| `VibratoRate`, `VibratoDepth`, `VibratoDelay` | bow | Hz, semitones, seconds |
| `Body`, `BodyMix` | | `GuitarBody`, `ViolinBody`, `CelloBody`, `HarpBody`, or `NoBody`; 0-1 |

The bow's speed is its loudness: the note amplitude sets it (faster than the model allows, a
string jumps to its octave, as a real one does). A bowed note's pitch is within about ten cents:
the friction pulls it, as on a real string. **Value inputs**: 1 pressure, 2 vibrato depth.

| Preset | Character |
|---|---|
| `guitar`, `steel`, `muted` | Nylon, steel with a pick, palm muted |
| `harp` | Plucked in the middle, rings on |
| `bass` | A fingered electric bass |
| `pizzicato` | A violin plucked |
| `violin`, `cello` | Bowed, vibrato coming in |
| `slowStrings` | Swelling in and out |
| `ponticello` | Bowed near the bridge, glassy |
| `flautando` | Bowed lightly over the fingerboard |

## modal

`synths/modal`

Struck things: bars, tubes, bells, tines, each ringing in its modes (partials with their own
pitch ratios and decay times), struck by a mallet. The mallet is a force as long as it touches:
a soft one touches long and wakes few high modes, a hard one is bright. Where it strikes mutes the
modes with a node there.

```go
mallets := modal.NewPoly(modal.Marimba(), 8).AddTo(p)
bell := modal.New(modal.Config{Modes: []modal.Mode{{1, 1, 1}, {2.76, 0.5, 0.6}}, Decay: 3, Level: 1})
```

| Setting | Meaning |
|---|---|
| `Modes` | `[]Mode{Ratio, Amp, Decay}`: pitch over the note, loudness, decay relative to `Decay` |
| `Decay` | Seconds the lowest mode rings |
| `HighDamping` | The higher modes die sooner: decay times (f0/f)^HighDamping |
| `Hardness` | 0-1, the mallet; the velocity adds |
| `Position` | 0-1, where it is struck |
| `Release` | Seconds it dies in once let go (a damper), 0 rings on |
| `Tremolo`, `TremoloDepth` | A vibraphone's motor |
| `Beat` | Each mode splits in two this fraction apart: bells beat |
| `Noise` | 0-1, the knock of the mallet |

**Value inputs**: 1 hardness, 2 tremolo depth.

| Preset | Character |
|---|---|
| `marimba`, `xylophone`, `vibraphone`, `glockenspiel` | The bars of the orchestra |
| `tubularBells` | Chimes |
| `churchBell` | Hum, prime, tierce, quint and nominal, beating |
| `kalimba` | A thumb piano |
| `singingBowl` | Long and beating |
| `woodBlock` | A pitched knock |

## drumsynth

`synths/drumsynth`

Drums from oscillators and noise, the way the classic drum machines make them: a kick is a sine
falling in pitch, a snare two tuned heads and a burst of noise, a hihat six square waves at
clashing pitches through a high filter. A kit is a set of sounds by name, played like the sample
kit of `drums`: a step sequencer row plays a sound by name, `"speed"` changes its pitch, a note
off chokes it; a keyboard plays them on the General MIDI keys.

```go
kit := drumsynth.NewPoly(drumsynth.Config{Kit: drumsynth.Kit909()}, 16).AddTo(p)
seq.Row("kick", stepseq.Sound(kit, "kick"))
```

The kits `Kit808()` and `Kit909()` have `kick`, `snare`, `hat`, `open`, `tomLow`, `tomMid`,
`tomHigh`, `clap`, `rim`, `cowbell` and `cymbal`; the hats choke each other. A `Sound` has a
`Kind` (`Kick`, `Snare`, `Hat`, `Tom`, `Clap`, `Rim`, `Cowbell`, `Cymbal`), `Pitch`, `Decay`,
`Sweep` and `SweepTime` (the pitch falling), `Tone` (the filter of its noise or metal), `Snap`
(the attack: a kick's click, a snare's wires), `Drive`, `Level`, `Pan` and `Choke` (a group).
Change one or add your own:

```go
kit := drumsynth.Kit808()
kit.Sounds["kick"] = drumsynth.Sound{Kind: drumsynth.Kick, Pitch: 45, Decay: 2, Sweep: 14, SweepTime: 0.03, Level: 0.7}
```

The sounds of the kits peak around -3 dB (kick), -5 (snare), -6 (toms, clap) and -10 (hats, rim).

## vocal

`synths/vocal`

Sung vowels: a band limited buzz like the vocal folds, shaped by the five formants of a bass,
tenor, alto or soprano (after the formant table of Csound), with breath, vibrato and the small
drift of a real voice. A choir is a voice sung by several singers at once, each a little out of
tune and time, spread over the stereo field.

```go
choir := vocal.NewPoly(vocal.Choir(), 8).AddTo(p)
choir.Receive(psalm.Message{"vowel": 4.0}, 0) // all sing "u", gliding there
```

| Setting | Meaning |
|---|---|
| `Range` | `Auto` (by the pitch of the note), `Bass`, `Tenor`, `Alto`, `Soprano` |
| `Vowel` | 0-4: a e i o u, fractions morph |
| `VowelGlide` | Seconds a change of vowel takes |
| `Brightness` | 0-1, breathy to pressed; the velocity adds |
| `Breath` | 0-1, the noise of the air |
| `Attack`, `Release` | Seconds |
| `VibratoRate`, `VibratoDepth`, `VibratoDelay` | Hz, semitones, seconds |
| `Singers`, `Detune`, `Spread` | Voices per note, cents they spread over, 0-1 how wide |
| `Steady` | The pitch held perfectly still |

The first formant follows the pitch up when it is higher, as singers do. **Value inputs**: 1 vowel,
2 vibrato depth.

| Preset | Character |
|---|---|
| `solo` | One voice, natural vibrato |
| `choir`, `aah`, `ooh` | Sections on "a" and "u" |
| `breathy` | Close and airy |
| `monks` | Low men's voices on "o", no vibrato |
| `angels` | High voices, wide and airy |
| `robot` | Steady and bright |

## pd

`synths/pd`

Phase distortion after the Casio CZ: a cosine read at a bent speed. Bent a little it stays a sine;
bent further it turns into a saw, a square, a pulse or a resonant sweep, and the DCW envelope bends
it over the note the way a filter envelope opens a subtractive synth. Two lines of an oscillator,
its DCW and its amp (DCA) envelope sound together, detuned or ring modulated. The bends never get
steeper than a few samples: no aliasing high up.

```go
cz := pd.NewPoly(pd.ResoBass(), 4).AddTo(p)
```

A `Line` has `Waveform` (`Saw`, `Square`, `Pulse`, `SawPulse`, `Reso1`-`Reso3`), `Second` and
`Alternate` (another waveform every other cycle), `Octave`, `Detune` (cents), `Level`, `DCW` (0-1,
the distortion at the top of `DCWEnv`), `KeyFollow` (higher notes distorted less), `Velocity` and
`DCAEnv`. The `Config` has `Line1`, `Line2`, `Ring`, `PitchSweep`/`PitchTime`, a vibrato and
`Width` (the lines apart in stereo). **Value input** 1: the distortion of both lines.

Presets: `brass`, `strings`, `ePiano`, `bass`, `resoBass`, `organ`, `bell` (ring modulated),
`lead` (saw and pulse alternating), `sweep`, `clav`.

## westcoast

`synths/westcoast`

A synth in the Buchla way: rich sounds made, not filtered out. A complex oscillator (a principal
sine, frequency and amplitude modulated by a second oscillator) goes into a wave folder, then
through a low pass gate: a filter and an amplifier opened together by a vactrol that lights up at
once and fades slowly, darker as it closes. Struck short, the gate rings like a hand drum.

```go
wc := westcoast.NewPoly(westcoast.Bongo(), 8).AddTo(p)
wc.ValueIn(lfo.New(true, 0.2, 0, 0.6, lfo.Sine).AddTo(p), 0, 1) // the fold breathing
```

| Setting | Meaning |
|---|---|
| `ModWave`, `ModRatio` | The modulator: `ModSine`, `ModTriangle`, `ModSaw`, `ModSquare`; its frequency over the note's |
| `FM`, `AM` | The index of the frequency modulation; 0-1 amplitude modulation |
| `Fold`, `Symmetry` | 0-1 how far the wave folds; -1 to 1 off center (even overtones) |
| `Attack`, `Decay`, `Sustain` | The function generator (seconds); `Sustain` holds while the key is down |
| `EnvToFold`, `EnvToFM` | What the function adds to the fold and the index |
| `Gate` | `Combo` (filter and amplifier), `LowPass`, `VCA` |
| `Resonance`, `Response`, `Color` | The gate: how it rings, how slowly the vactrol lets go, how bright it opens |

**Value inputs**: 1 fold, 2 FM index. Presets: `bongo`, `pluck`, `marimba`, `bell`, `bass`,
`drone`, `metal`, `wobble`.

## supersaw

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

**Value inputs**: 1 cutoff, 2 detune. Presets: `trancePad`, `lead`, `pluck`, `hoover`, `stab`,
`unisonBass`.
